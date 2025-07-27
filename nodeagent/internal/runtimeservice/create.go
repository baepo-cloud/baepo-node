package runtimeservice

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
)

func (s *Service) Start(ctx context.Context, opts types.RuntimeStartOptions) error {
	if err := s.createRuntimeConfigFile(opts); err != nil {
		return fmt.Errorf("failed to create runtime config file: %w", err)
	}

	cmd := exec.Command(s.config.RuntimeBinary, s.getRuntimeConfigPath(opts.Machine.ID))
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	cmd.Env = os.Environ()

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err = cmd.Start(); err != nil {
		return fmt.Errorf("failed to start runtime: %w", err)
	}

	go pipeToLogger(stdoutPipe, slog.Default(), slog.LevelDebug, "stdout")
	go pipeToLogger(stderrPipe, slog.Default(), slog.LevelDebug, "stderr")

	return nil
}

func (s *Service) createRuntimeConfigFile(opts types.RuntimeStartOptions) error {
	runtimeDir := s.getRuntimeDir(opts.Machine.ID)
	_ = os.MkdirAll(runtimeDir, 0644)

	initConfig := coretypes.RuntimeConfig{
		WorkingDir: runtimeDir,
		MachineID:  opts.Machine.ID,
		Cpus:       opts.Machine.Spec.Cpus,
		MemoryMB:   opts.Machine.Spec.MemoryMB,
		Network: coretypes.RuntimeNetworkConfig{
			InterfaceName:  opts.Machine.NetworkInterface.Name,
			IPAddress:      opts.Machine.NetworkInterface.IPAddress,
			MacAddress:     opts.Machine.NetworkInterface.MacAddress,
			GatewayAddress: opts.Machine.NetworkInterface.GatewayAddress,
			Hostname:       opts.Machine.ID,
		},
		Containers: make([]coretypes.RuntimeContainerConfig, len(opts.Machine.Containers)),
	}
	containerVolumes := map[string]*types.MachineVolume{}
	for _, machineVolume := range opts.Machine.Volumes {
		containerVolumes[machineVolume.ContainerID] = machineVolume
	}

	for index, container := range opts.Machine.Containers {
		volume, ok := containerVolumes[container.ID]
		if !ok {
			return fmt.Errorf("failed to find machine volume")
		}

		imageSpec := volume.Image.Spec
		containerSpec := *container.Spec.ToCore()
		if containerSpec.Env == nil {
			containerSpec.Env = map[string]string{}
		}
		for key, value := range imageSpec.Env {
			if _, ok = containerSpec.Env[key]; !ok {
				containerSpec.Env[key] = value
			}
		}
		if containerSpec.WorkingDir == nil || *containerSpec.WorkingDir == "" {
			workingDir := imageSpec.WorkingDir
			if workingDir == "" {
				workingDir = "/"
			}
			containerSpec.WorkingDir = &workingDir
		}
		if containerSpec.User == nil || *containerSpec.User == "" {
			user := imageSpec.User
			if user == "" {
				user = "root"
			}
			containerSpec.User = &user
		}
		if containerSpec.Command == nil {
			containerSpec.Command = imageSpec.Command
		}

		initConfig.Containers[index] = coretypes.RuntimeContainerConfig{
			ContainerID:   container.ID,
			ContainerSpec: containerSpec,
			VolumePath:    *volume.Volume.Path,
		}
	}

	configPath := s.getRuntimeConfigPath(opts.Machine.ID)
	configFile, err := os.Create(configPath)
	if err != nil {
		return err
	}

	defer configFile.Close()
	if err = json.NewEncoder(configFile).Encode(initConfig); err != nil {
		return err
	}

	return nil
}

func pipeToLogger(r io.Reader, logger *slog.Logger, level slog.Level, stream string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		logger.Log(context.Background(), level, scanner.Text(), slog.String("stream", stream))
	}
	if err := scanner.Err(); err != nil {
		logger.Error("log stream error", slog.String("stream", stream), slog.Any("error", err))
	}
}
