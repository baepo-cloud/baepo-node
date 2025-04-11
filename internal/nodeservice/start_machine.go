package nodeservice

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"log/slog"
	"strings"
)

func (s *Service) StartMachine(ctx context.Context, opts types.NodeStartMachineOptions) (*types.Machine, error) {
	slog.Info("starting machine", slog.String("machine-id", opts.MachineID))
	machine := &types.Machine{
		ID: opts.MachineID,
		Spec: &types.MachineSpec{
			Image:    opts.Spec.Image,
			Vcpus:    opts.Spec.Vcpus,
			MemoryMB: opts.Spec.MemoryMB,
			Env:      map[string]string{},
		},
	}

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

	volume, err := s.volumeProvider.CreateVolume(ctx, image)
	if err != nil {
		return nil, fmt.Errorf("failed to create machine volume: %w", err)
	}
	machine.Volume = volume

	machineNetwork, err := s.networkProvider.AllocateInterface()
	if err != nil {
		return nil, fmt.Errorf("failed to allocate machine network: %w", err)
	}
	machine.NetworkInterface = machineNetwork

	machine.RuntimePID, err = s.runtimeProvider.Create(ctx, machine)
	if err != nil {
		return nil, fmt.Errorf("failed to create machine: %w", err)
	}

	err = s.runtimeProvider.Boot(ctx, machine)
	if err != nil {
		return nil, fmt.Errorf("failed to boot machine: %w", err)
	}

	s.lock.Lock()
	s.machines[machine.ID] = machine
	s.lock.Unlock()

	return machine, nil
}
