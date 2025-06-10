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
	"path/filepath"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

func (p *Provider) BuildInitRamFS(ctx context.Context, opts types.RuntimeCreateOptions) error {
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
		initConfig.Containers[index] = coretypes.InitContainerConfig{
			ContainerID:   container.ID,
			ContainerSpec: *container.Spec.ToCore(),
			Volume:        fmt.Sprintf("/dev/vd%v", string(alphabet[index%len(alphabet)])),
		}
	}

	configFile, err := os.Create(filepath.Join(tmpDir, "config.json"))
	if err != nil {
		return err
	}
	defer configFile.Close()
	if err = json.NewEncoder(configFile).Encode(initConfig); err != nil {
		return err
	}

	initPath := filepath.Join(tmpDir, "init")
	if err = copyFile(p.config.InitBinary, initPath); err != nil {
		return fmt.Errorf("failed to copy init binary: %w", err)
	} else if err = os.Chmod(initPath, 0755); err != nil {
		return fmt.Errorf("failed to make init binary executable: %w", err)
	}

	initContainerPath := filepath.Join(tmpDir, "initcontainer")
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
