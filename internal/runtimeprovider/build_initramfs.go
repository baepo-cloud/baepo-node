package runtimeprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/initd"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

func (p *Provider) BuildInitRamFS(ctx context.Context, opts types.RuntimeCreateOptions) error {
	tmpDir, err := os.MkdirTemp("", "baepo-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	maskSize, _ := opts.NetworkInterface.NetworkCIDR.Mask.Size()
	initConfig := initd.Config{
		IPAddress:      fmt.Sprintf("%s/%d", opts.NetworkInterface.IPAddress.String(), maskSize),
		MacAddress:     opts.NetworkInterface.MacAddress.String(),
		GatewayAddress: opts.NetworkInterface.GatewayAddress.String(),
		Env:            opts.Spec.Env,
		User:           opts.Spec.User,
		WorkingDir:     opts.Spec.WorkingDir,
		Command:        opts.Spec.Command,
		Hostname:       opts.MachineID,
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
	if err = copyFile(p.initBinary, initPath); err != nil {
		return fmt.Errorf("failed to copy init binary: %w", err)
	} else if err = os.Chmod(initPath, 0755); err != nil {
		return fmt.Errorf("failed to make init binary executable: %w", err)
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
