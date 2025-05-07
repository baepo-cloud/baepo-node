package container

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func (c *Container) setupFilesystem() error {
	if err := os.MkdirAll(c.rootDir, 0644); err != nil {
		return fmt.Errorf("failed to create root directory: %w", err)
	}

	mounts := []struct {
		source string
		target string
		fstype string
		flags  uintptr
		data   string
	}{
		{c.config.Volume, c.rootDir, "ext4", unix.MS_RELATIME, ""},
		{"devtmpfs", c.rootDir + "/dev", "devtmpfs", unix.MS_NOSUID, "mode=0755"},
		{"proc", c.rootDir + "/proc", "proc", 0, ""},
		{"sysfs", c.rootDir + "/sys", "sysfs", 0, ""},
		{"tmpfs", c.rootDir + "/tmp", "tmpfs", 0, "mode=1777"},
		{"tmpfs", c.rootDir + "/run", "tmpfs", 0, ""},
	}
	for _, m := range mounts {
		_ = os.MkdirAll(m.target, 0644)
		if err := syscall.Mount(m.source, m.target, m.fstype, m.flags, m.data); err != nil {
			return fmt.Errorf("failed to mount %s on %s: %v", m.source, m.target, err)
		}
	}

	if err := unix.Chroot(c.rootDir); err != nil {
		return fmt.Errorf("failed to change the root directory: %w", err)
	}

	return nil
}
