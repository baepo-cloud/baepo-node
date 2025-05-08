package main

import (
	"encoding/json"
	"fmt"
	"github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/init/internal/bootstrap"
	"github.com/baepo-cloud/baepo-node/init/internal/containerservice"
	"github.com/baepo-cloud/baepo-node/init/internal/initserver"
	"github.com/baepo-cloud/baepo-node/init/internal/logservice"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	configFile, err := os.Open("/config.json")
	if err != nil {
		panic(err)
	}
	defer configFile.Close()

	var config types.InitConfig
	if err = json.NewDecoder(configFile).Decode(&config); err != nil {
		panic(err)
	}

	slog.Info("starting init")
	if err = bootstrap.MountFilesystems(); err != nil {
		panic(fmt.Errorf("failed to mount filesystem: %v", err))
	} else if err = bootstrap.SetupNetwork(config); err != nil {
		panic(fmt.Errorf("failed to setup network: %v", err))
	}

	logService, err := logservice.New("/logs")
	if err != nil {
		panic(err)
	}

	containerService := containerservice.New(logService)
	containerService.Start()

	errChan := make(chan error, 1)
	for _, containerConfig := range config.Containers {
		go func() {
			errChan <- containerService.StartContainer(containerConfig)
		}()
	}

	initServer := initserver.New(containerService, logService)
	go func() {
		errChan <- initServer.Start()
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case err = <-errChan:
			panic(err)
			//case sig := <-sigChan:
			//for _, ctr := range s.containers {
			//	_ = ctr.cmd.Process.Signal(sig)
			//}
		}
	}

	//containerService.Stop()
	//slog.Error("shutting down", slog.Any("error", err))
	//syscall.Sync()
	//_ = syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
}
