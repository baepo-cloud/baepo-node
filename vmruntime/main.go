package main

import (
	"context"
	"encoding/json"
	"fmt"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/vmruntime/internal/runtime"
	"os"
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
		RuntimeConfig:             *config,
		InitBinaryPath:            os.Getenv("VMRUNTIME_INIT_BINARY_PATH"),
		InitContainerBinaryPath:   os.Getenv("VMRUNTIME_INIT_CONTAINER_BINARY_PATH"),
		CloudHypervisorBinaryPath: os.Getenv("VMRUNTIME_CLOUD_HYPERVISOR_BINARY_PATH"),
		VMLinuxPath:               os.Getenv("VMRUNTIME_VM_LINUX_PATH"),
	})
	if err = r.Start(context.Background()); err != nil {
		panic(err)
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
