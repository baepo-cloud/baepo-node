package main

import (
	"encoding/json"
	"github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/initcontainer/internal/container"
	"os"
)

func main() {
	var config types.InitContainerConfig
	if err := json.Unmarshal([]byte(os.Args[1]), &config); err != nil {
		panic(err)
	}

	if err := container.New(config).Start(); err != nil {
		panic(err)
	}
}
