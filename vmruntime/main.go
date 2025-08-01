package main

import (
	"context"
	"encoding/json"
	"fmt"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/vmruntime/internal/runtime"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		_, _ = fmt.Fprintf(os.Stderr, "Usage: %s <runtime config file>\n", os.Args[0])
		os.Exit(1)
	}

	config, err := getRuntimeConfig()
	if err != nil {
		panic(err)
	}

	r := runtime.New(&runtime.Config{
		RuntimeConfig:         *config,
		InitBinary:            os.Getenv("VMRUNTIME_INIT_BINARY"),
		InitContainerBinary:   os.Getenv("VMRUNTIME_INIT_CONTAINER_BINARY"),
		CloudHypervisorBinary: os.Getenv("VMRUNTIME_CLOUD_HYPERVISOR_BINARY"),
		VMLinux:               os.Getenv("VMRUNTIME_VM_LINUX"),
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)

	errChan := make(chan error, 1)
	go func() {
		errChan <- r.Start(ctx)
	}()

	select {
	case err = <-errChan:
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "runtime error: %v\n", err)
			os.Exit(1)
		}
	case sig := <-sigChan:
		switch sig {
		case syscall.SIGTERM, syscall.SIGINT:
			fmt.Println("initiating graceful shutdown")
			stopCtx, stopCancelCtx := context.WithTimeout(context.Background(), 30*time.Second)
			defer stopCancelCtx()
			if err = r.Stop(stopCtx); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "error during graceful shutdown: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("graceful shutdown completed")
			os.Exit(0)
		case syscall.SIGKILL:
			fmt.Println("force killing")
			cancel()
			stopCtx, stopCancelCtx := context.WithTimeout(context.Background(), 5*time.Second)
			defer stopCancelCtx()
			if err = r.ForceStop(stopCtx); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "error during force stop: %v\n", err)
			}
			os.Exit(1)
		}
	}
}

func getRuntimeConfig() (*coretypes.RuntimeConfig, error) {
	filePath := os.Args[1]
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}

	defer file.Close()
	var config *coretypes.RuntimeConfig
	if err = json.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	return config, nil
}
