package runtimeprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/types"
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

	maskSize, _ := opts.NetworkInterface.NetworkCIDR.Mask.Size()
	initConfig := types.InitConfig{
		IPAddress:      fmt.Sprintf("%s/%d", opts.NetworkInterface.IPAddress.String(), maskSize),
		MacAddress:     opts.NetworkInterface.MacAddress.String(),
		GatewayAddress: opts.NetworkInterface.GatewayAddress.String(),
		Hostname:       opts.MachineID,
		Containers:     make([]types.InitContainerConfig, len(opts.Spec.Containers)),
	}
	machineVolumes := map[string]*types.MachineVolume{}
	for _, machineVolume := range opts.Volumes {
		machineVolumes[machineVolume.Container] = machineVolume
	}

	for index, ctr := range opts.Spec.Containers {
		machineVolume, ok := machineVolumes[ctr.Name]
		if !ok {
			return fmt.Errorf("failed to find machine volume")
		}

		imageSpec := machineVolume.Image.Spec
		cfg := types.InitContainerConfig{
			Name:       ctr.Name,
			Env:        imageSpec.Env,
			Command:    imageSpec.Command,
			User:       imageSpec.User,
			WorkingDir: imageSpec.WorkingDir,
			Volume:     fmt.Sprintf("/dev/vd%v", string(alphabet[index%len(alphabet)])),
		}
		if cfg.Name == "" {
			cfg.Name = opts.MachineID
		}
		for key, value := range ctr.Env {
			cfg.Env[key] = value
		}
		if ctr.WorkingDir != "" {
			cfg.WorkingDir = ctr.WorkingDir
		}
		if ctr.Command != nil {
			cfg.Command = ctr.Command
		}

		initConfig.Containers[index] = cfg
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
		p.getInitRamFSPath(opts.MachineID),
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
