package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"io"
	"os"
	"os/exec"
	"path"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

func (r *Runtime) buildInitRamFS(ctx context.Context) error {
	_ = os.MkdirAll(r.config.WorkingDir, 0644)

	tmpDir, err := os.MkdirTemp("", "baepo-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	maskSize, _ := r.config.Network.NetworkCIDR.Mask.Size()
	initConfig := coretypes.InitConfig{
		IPAddress:      fmt.Sprintf("%s/%d", r.config.Network.IPAddress.String(), maskSize),
		MacAddress:     r.config.Network.MacAddress.String(),
		GatewayAddress: r.config.Network.GatewayAddress.String(),
		Hostname:       r.config.MachineID,
		Containers:     make([]coretypes.InitContainerConfig, len(r.config.Containers)),
	}
	for index, container := range r.config.Containers {
		initConfig.Containers[index] = coretypes.InitContainerConfig{
			ContainerID:   container.ContainerID,
			ContainerSpec: container.ContainerSpec,
			Volume:        fmt.Sprintf("/dev/vd%v", string(alphabet[index%len(alphabet)])),
		}
	}

	configFile, err := os.Create(r.getInitConfigPath())
	if err != nil {
		return err
	}
	defer configFile.Close()
	if err = json.NewEncoder(configFile).Encode(initConfig); err != nil {
		return err
	} else if err = copyFile(r.getInitConfigPath(), path.Join(tmpDir, "config.json")); err != nil {
		return fmt.Errorf("failed to copy init config: %w", err)
	}

	initPath := path.Join(tmpDir, "init")
	if err = copyFile(r.config.InitBinaryPath, initPath); err != nil {
		return fmt.Errorf("failed to copy init binary: %w", err)
	} else if err = os.Chmod(initPath, 0755); err != nil {
		return fmt.Errorf("failed to make init binary executable: %w", err)
	}

	initContainerPath := path.Join(tmpDir, "initcontainer")
	if err = copyFile(r.config.InitContainerBinaryPath, initContainerPath); err != nil {
		return fmt.Errorf("failed to copy initcontainer binary: %w", err)
	} else if err = os.Chmod(initContainerPath, 0755); err != nil {
		return fmt.Errorf("failed to make initcontainer binary executable: %w", err)
	}

	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", fmt.Sprintf(
		"find . -print0 | cpio --null -ov --format=newc | gzip -9 > %v",
		r.getInitRamFSPath(),
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
