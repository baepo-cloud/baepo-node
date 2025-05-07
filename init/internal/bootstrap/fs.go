package bootstrap

import (
	"fmt"
	"golang.org/x/sys/unix"
	"log/slog"
	"os"
	"syscall"
)

func MountFilesystems() error {
	if err := os.MkdirAll("/dev", 0644); err != nil {
		return err
	}

	mounts := []struct {
		source string
		target string
		fstype string
		flags  uintptr
		data   string
	}{
		{"devtmpfs", "/dev", "devtmpfs", unix.MS_NOSUID, "mode=0755"},
	}
	for _, m := range mounts {
		if err := syscall.Mount(m.source, m.target, m.fstype, m.flags, m.data); err != nil {
			return fmt.Errorf("failed to mount %s on %s: %v", m.source, m.target, err)
		}
	}

	symlinks := []struct {
		source      string
		destination string
	}{
		{source: "/proc/self/fd", destination: "/dev/fd"},
		{source: "/proc/self/fd/0", destination: "/dev/stdin"},
		{source: "/proc/self/fd/1", destination: "/dev/stdout"},
		{source: "/proc/self/fd/2", destination: "/dev/stderr"},
	}
	for _, symlink := range symlinks {
		if err := unix.Symlinkat(symlink.source, 0, symlink.destination); err != nil {
			slog.Warn("failed to create symlink",
				slog.String("source", symlink.source),
				slog.String("destination", symlink.destination),
				slog.Any("error", err))
		}
	}

	return nil
}
