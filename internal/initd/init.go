package initd

import (
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/initd/types"
	"github.com/vishvananda/netlink"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

type (
	Config struct {
		IPAddress      *netlink.Addr
		MacAddress     net.HardwareAddr
		GatewayAddress net.IP
	}

	initd struct {
		config Config
		cmd    *exec.Cmd
	}
)

var _ types.InitD = (*initd)(nil)

const ServerPort = 9000

func Run(config Config) error {
	slog.Info("starting init process")
	init := &initd{
		config: config,
	}

	if err := init.MountFilesystems(); err != nil {
		panic(err)
	}

	if err := syscall.Chroot("/mnt"); err != nil {
		return fmt.Errorf("failed to chroot to /mnt: %v", err)
	}

	if err := init.SetupNetwork(); err != nil {
		return fmt.Errorf("failed to setup network: %v", err)
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- init.StartProcess()
	}()
	go func() {
		errChan <- init.StartServer()
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var err error
	running := true
	for running {
		select {
		case err = <-errChan:
			running = false
		case sig := <-sigChan:
			_ = init.cmd.Process.Signal(sig)
		}
		time.Sleep(10 * time.Second)
	}

	slog.Error("shutting down", slog.Any("error", err))
	syscall.Sync()
	_ = syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
	return nil
}

func (d *initd) MainCmd() *exec.Cmd {
	return d.cmd
}
