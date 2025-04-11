package initd

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"
)

func (d *initd) StartProcess() error {
	workingDir := d.config.WorkingDir
	if workingDir == "" {
		workingDir = "/"
	}
	if err := syscall.Chdir(workingDir); err != nil {
		return fmt.Errorf("failed to change directory to %v: %w", workingDir, err)
	}

	if err := syscall.Sethostname([]byte(d.config.Hostname)); err != nil {
		return fmt.Errorf("failed to set hostname: %w", err)
	}

	if err := os.WriteFile("/etc/hostname", []byte(d.config.Hostname+"\n"), 0x0755); err != nil {
		return fmt.Errorf("failed to write /etc/hostname: %w", err)
	}

	if err := unix.Symlinkat("/proc/self/fd", 0, "/dev/fd"); err != nil {
		return err
	}

	if err := unix.Symlinkat("/proc/self/fd/0", 0, "/dev/stdin"); err != nil {
		return err
	}

	if err := unix.Symlinkat("/proc/self/fd/1", 0, "/dev/stdout"); err != nil {
		return err
	}

	if err := unix.Symlinkat("/proc/self/fd/2", 0, "/dev/stderr"); err != nil {
		return err
	}

	if d.config.User == "" {
		d.config.User = "root"
	}

	targetUser, err := user.Lookup(d.config.User)
	if err != nil {
		return fmt.Errorf("failed to lookup user %s: %v", d.config.User, err)
	}
	d.config.Env["HOME"] = targetUser.HomeDir

	uid, err := strconv.Atoi(targetUser.Uid)
	if err != nil {
		return fmt.Errorf("failed to convert user id to int: %v", err)
	}

	gid, err := strconv.Atoi(targetUser.Gid)
	if err != nil {
		return fmt.Errorf("failed to convert user group id to int: %v", err)
	}

	d.cmd = exec.Command(d.config.Command[0], d.config.Command[1:]...)
	for key, value := range d.config.Env {
		d.cmd.Env = append(d.cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
	d.cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid:         uint32(uid),
			Gid:         uint32(gid),
			Groups:      []uint32{uint32(gid)},
			NoSetGroups: true,
		},
	}
	d.cmd.Stdin = os.Stdin
	d.cmd.Stdout = os.Stdout
	d.cmd.Stderr = os.Stderr
	if err = d.cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute %v: %w", strings.Join(d.config.Command, " "), err)
	}

	return nil
}
