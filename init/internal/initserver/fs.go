package initserver

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"syscall"
)

func (s *Server) mountFilesystems() error {
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

	return nil
}
