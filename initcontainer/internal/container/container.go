package container

import (
	"fmt"
	"github.com/baepo-cloud/baepo-node/core/types"
	"log/slog"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"
)

type Container struct {
	config  types.InitContainerConfig
	log     *slog.Logger
	rootDir string
	cmd     *exec.Cmd
}

func New(config types.InitContainerConfig) *Container {
	return &Container{
		config:  config,
		log:     slog.With("container", config.ContainerID),
		rootDir: "/mnt/" + config.ContainerID,
	}
}

func (c *Container) Start() error {
	if _, err := os.Stat(c.rootDir); os.IsNotExist(err) {
		if err = c.setupFilesystem(); err != nil {
			return fmt.Errorf("failed to setup filesystem: %w", err)
		}
	}

	if err := c.setupNetworking(); err != nil {
		return fmt.Errorf("failed to setup networking: %w", err)
	}

	workingDir := c.config.WorkingDir
	if workingDir == "" {
		workingDir = "/"
	}

	if err := syscall.Chdir(workingDir); err != nil {
		return fmt.Errorf("failed to change directory to %v: %w", workingDir, err)
	}

	if c.config.User == "" {
		c.config.User = "root"
	}

	targetUser, err := user.Lookup(c.config.User)
	if err != nil {
		if c.config.User != "root" {
			return fmt.Errorf("failed to lookup user %s: %v", c.config.User, err)
		}

		targetUser = &user.User{
			Uid:      "0",
			Gid:      "0",
			Username: "root",
			Name:     "root",
			HomeDir:  "/root",
		}
	}
	c.config.Env["HOME"] = targetUser.HomeDir

	uid, err := strconv.Atoi(targetUser.Uid)
	if err != nil {
		return fmt.Errorf("failed to convert user id to int: %v", err)
	}

	gid, err := strconv.Atoi(targetUser.Gid)
	if err != nil {
		return fmt.Errorf("failed to convert user group id to int: %v", err)
	}

	c.cmd = exec.Command(c.config.Command[0], c.config.Command[1:]...)
	for key, value := range c.config.Env {
		c.cmd.Env = append(c.cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
	c.cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS | // New mount namespace
			syscall.CLONE_NEWPID | // New PID namespace
			syscall.CLONE_NEWUTS, // New UTS namespace
		Credential: &syscall.Credential{
			Uid:         uint32(uid),
			Gid:         uint32(gid),
			Groups:      []uint32{uint32(gid)},
			NoSetGroups: true,
		},
	}
	c.cmd.Stdin = os.Stdin
	c.cmd.Stdout = os.Stdout
	c.cmd.Stderr = os.Stderr
	if err = c.cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute %v: %w", strings.Join(c.config.Command, " "), err)
	}

	return nil
}
