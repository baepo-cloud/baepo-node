package runtimeprovider

import (
	"context"
	"encoding/json"
	"fmt"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"io"
	"os"
	"os/exec"
	"path"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

func (p *Provider) BuildInitRamFS(ctx context.Context, opts types.RuntimeCreateOptions) error {
	machineDir := p.getMachineDir(opts.Machine.ID)
	_ = os.MkdirAll(machineDir, 0644)

	tmpDir, err := os.MkdirTemp("", "baepo-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	maskSize, _ := opts.Machine.NetworkInterface.NetworkCIDR.Mask.Size()
	initConfig := coretypes.InitConfig{
		IPAddress:      fmt.Sprintf("%s/%d", opts.Machine.NetworkInterface.IPAddress.String(), maskSize),
		MacAddress:     opts.Machine.NetworkInterface.MacAddress.String(),
		GatewayAddress: opts.Machine.NetworkInterface.GatewayAddress.String(),
		Hostname:       opts.Machine.ID,
		Containers:     make([]coretypes.InitContainerConfig, len(opts.Machine.Containers)),
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
		if containerSpec.Restart == nil {
			containerSpec.Restart = &coretypes.ContainerRestartSpec{
				Policy:     coretypes.ContainerRestartPolicyOnFailure,
				MaxRetries: 3,
			}
		}

		initConfig.Containers[index] = coretypes.InitContainerConfig{
			ContainerID:   container.ID,
			ContainerSpec: containerSpec,
			Volume:        fmt.Sprintf("/dev/vd%v", string(alphabet[index%len(alphabet)])),
		}
	}

	initConfigFilePath := p.getInitConfigPath(opts.Machine.ID)
	configFile, err := os.Create(initConfigFilePath)
	if err != nil {
		return err
	}
	defer configFile.Close()
	if err = json.NewEncoder(configFile).Encode(initConfig); err != nil {
		return err
	} else if err = copyFile(initConfigFilePath, path.Join(tmpDir, "config.json")); err != nil {
		return fmt.Errorf("failed to copy init config: %w", err)
	}

	initPath := path.Join(tmpDir, "init")
	if err = copyFile(p.config.InitBinary, initPath); err != nil {
		return fmt.Errorf("failed to copy init binary: %w", err)
	} else if err = os.Chmod(initPath, 0755); err != nil {
		return fmt.Errorf("failed to make init binary executable: %w", err)
	}

	initContainerPath := path.Join(tmpDir, "initcontainer")
	if err = copyFile(p.config.InitContainerBinary, initContainerPath); err != nil {
		return fmt.Errorf("failed to copy initcontainer binary: %w", err)
	} else if err = os.Chmod(initContainerPath, 0755); err != nil {
		return fmt.Errorf("failed to make initcontainer binary executable: %w", err)
	}

	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", fmt.Sprintf(
		"find . -print0 | cpio --null -ov --format=newc | gzip -9 > %v",
		p.getInitRamFSPath(opts.Machine.ID),
	))
	cmd.Dir = tmpDir
	if err = cmd.Run(); err != nil {
		return err
	}

	return nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
