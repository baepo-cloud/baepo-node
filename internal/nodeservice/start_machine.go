package nodeservice

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"log/slog"
	"strings"
	"time"
)

func (s *Service) StartMachine(ctx context.Context, opts types.NodeStartMachineOptions) (machine *types.Machine, err error) {
	startedAt := time.Now()
	log := slog.With(slog.String("machine-id", opts.MachineID))
	log.Info("starting machine")

	defer func() {
		log = log.With(slog.Duration("duration", time.Now().Sub(startedAt)))
		if err != nil {
			log.Error("failed to start machine", slog.Any("error", err))
		} else {
			log.Info("machine started")
		}
	}()

	machine = &types.Machine{
		ID:     opts.MachineID,
		Status: types.MachineStatusStarting,
		Spec: &types.MachineSpec{
			Image:    opts.Spec.Image,
			Vcpus:    opts.Spec.Vcpus,
			MemoryMB: opts.Spec.MemoryMB,
			Env:      map[string]string{},
		},
	}
	if err = s.db.WithContext(ctx).Save(&machine).Error; err != nil {
		return nil, fmt.Errorf("failed to save machine: %w", err)
	}

	log.Info("fetching machine image")
	imageRef, err := name.ParseReference(machine.Spec.Image)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image reference: %w", err)
	}

	image, err := remote.Image(imageRef)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch remote image: %w", err)
	}

	imageConfigFile, err := image.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image config file: %w", err)
	}

	machine.Spec.User = imageConfigFile.Config.User
	machine.Spec.WorkingDir = imageConfigFile.Config.WorkingDir
	for _, env := range imageConfigFile.Config.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 1 {
			parts = append(parts, "")
		}
		machine.Spec.Env[parts[0]] = parts[1]
	}
	for key, value := range machine.Spec.Env {
		machine.Spec.Env[key] = value
	}
	machine.Spec.Command = append(imageConfigFile.Config.Entrypoint, imageConfigFile.Config.Cmd...)

	log.Info("creating machine volume")
	volume, err := s.volumeProvider.CreateVolume(ctx, image)
	if err != nil {
		return nil, fmt.Errorf("failed to create machine volume: %w", err)
	}
	machine.Volume = volume

	machine.Volume.MachineID = machine.ID
	if err = s.db.WithContext(ctx).Save(&machine.Volume).Error; err != nil {
		return nil, fmt.Errorf("failed to save volume: %w", err)
	}

	log.Info("creating machine network")
	machineNetwork, err := s.networkProvider.AllocateInterface(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate machine network: %w", err)
	}
	machine.NetworkInterface = machineNetwork

	machine.NetworkInterface.MachineID = machine.ID
	if err = s.db.WithContext(ctx).Save(&machine.NetworkInterface).Error; err != nil {
		return nil, fmt.Errorf("failed to save network: %w", err)
	}

	log.Info("creating machine runtime")
	pid, err := s.runtimeProvider.Create(ctx, machine)
	if err != nil {
		return nil, fmt.Errorf("failed to create machine: %w", err)
	}
	machine.RuntimePID = &pid
	if err = s.db.WithContext(ctx).Save(&machine).Error; err != nil {
		return nil, fmt.Errorf("failed to save machine: %w", err)
	}

	log.Info("booting machine")
	err = s.runtimeProvider.Boot(ctx, machine)
	if err != nil {
		return nil, fmt.Errorf("failed to boot machine: %w", err)
	}
	return machine, nil
}
