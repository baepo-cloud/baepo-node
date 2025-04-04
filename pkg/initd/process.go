package initd

import (
	"encoding/json"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"
)

type DockerInspectContainerOutput struct {
	Config struct {
		Hostname     string      `json:"Hostname"`
		Domainname   string      `json:"Domainname"`
		User         string      `json:"User"`
		AttachStdin  bool        `json:"AttachStdin"`
		AttachStdout bool        `json:"AttachStdout"`
		AttachStderr bool        `json:"AttachStderr"`
		Tty          bool        `json:"Tty"`
		OpenStdin    bool        `json:"OpenStdin"`
		StdinOnce    bool        `json:"StdinOnce"`
		Env          []string    `json:"Env"`
		Cmd          interface{} `json:"Cmd"`
		Image        string      `json:"Image"`
		Volumes      interface{} `json:"Volumes"`
		WorkingDir   string      `json:"WorkingDir"`
		Entrypoint   []string    `json:"Entrypoint"`
		OnBuild      interface{} `json:"OnBuild"`
		Labels       struct {
		} `json:"Labels"`
	} `json:"Config"`
}

func (d *initd) StartProcess() error {
	dockerInspectFile, err := os.Open("/_config.json")
	if err != nil {
		return fmt.Errorf("could not open docker inspect file: %v", err)
	}

	var containers []*DockerInspectContainerOutput
	if err = json.NewDecoder(dockerInspectFile).Decode(&containers); err != nil {
		return fmt.Errorf("could not decode docker inspect file: %v", err)
	}

	if len(containers) == 0 {
		return fmt.Errorf("no container found")
	}

	container := containers[0]
	workingDir := container.Config.WorkingDir
	if workingDir == "" {
		workingDir = "/"
	}
	if err = syscall.Chdir(workingDir); err != nil {
		return fmt.Errorf("failed to change directory to %v: %w", workingDir, err)
	}

	if err = syscall.Sethostname([]byte(container.Config.Hostname)); err != nil {
		return fmt.Errorf("failed to set hostname: %w", err)
	}

	if err = os.WriteFile("/etc/hostname", []byte(container.Config.Hostname+"\n"), 0x0755); err != nil {
		return fmt.Errorf("failed to write /etc/hostname: %w", err)
	}

	if err = unix.Symlinkat("/proc/self/fd", 0, "/dev/fd"); err != nil {
		return err
	}

	if err = unix.Symlinkat("/proc/self/fd/0", 0, "/dev/stdin"); err != nil {
		return err
	}

	if err = unix.Symlinkat("/proc/self/fd/1", 0, "/dev/stdout"); err != nil {
		return err
	}

	if err = unix.Symlinkat("/proc/self/fd/2", 0, "/dev/stderr"); err != nil {
		return err
	}

	env := container.Config.Env
	username := container.Config.User
	if username == "" {
		username = "root"
	}

	targetUser, err := user.Lookup(username)
	if err != nil {
		return fmt.Errorf("failed to lookup user %s: %v", username, err)
	}

	env = append(env, "HOME="+targetUser.HomeDir)

	uid, err := strconv.Atoi(targetUser.Uid)
	if err != nil {
		return fmt.Errorf("failed to convert user id to int: %v", err)
	}

	gid, err := strconv.Atoi(targetUser.Gid)
	if err != nil {
		return fmt.Errorf("failed to convert user group id to int: %v", err)
	}

	d.cmd = exec.Command(container.Config.Entrypoint[0], container.Config.Entrypoint[1:]...)
	d.cmd.Env = container.Config.Env
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
		return fmt.Errorf("failed to execute %v: %w", container.Config.Entrypoint, err)
	}
	return nil
}
