package initd

import (
	"fmt"
	"golang.org/x/sys/unix"
	"log/slog"
	"os"
	"syscall"
)

func (d *initd) MountFilesystems() error {
	for _, directory := range []string{"/dev", "/mnt"} {
		if err := os.MkdirAll(directory, 0644); err != nil {
			return err
		}
	}

	mounts := []struct {
		source string
		target string
		fstype string
		flags  uintptr
		data   string
	}{
		{"devtmpfs", "/dev", "devtmpfs", unix.MS_NOSUID, "mode=0755"},
		{"/dev/vda", "/mnt", "ext4", unix.MS_RELATIME, ""},
		{"devtmpfs", "/mnt/dev", "devtmpfs", unix.MS_NOSUID, "mode=0755"},
		{"proc", "/mnt/proc", "proc", 0, ""},
		{"sysfs", "/mnt/sys", "sysfs", 0, ""},
		{"tmpfs", "/mnt/tmp", "tmpfs", 0, "mode=1777"},
		{"tmpfs", "/mnt/run", "tmpfs", 0, ""},
	}

	for _, m := range mounts {
		slog.Info("mounting fs", slog.String("source", m.source), slog.String("target", m.target))
		if err := syscall.Mount(m.source, m.target, m.fstype, m.flags, m.data); err != nil {
			return fmt.Errorf("failed to mount %s on %s: %v", m.source, m.target, err)
		}
	}
	return nil
}
