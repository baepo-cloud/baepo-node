package main

import (
	"encoding/json"
	"github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/init/internal/initserver"
	"github.com/baepo-cloud/baepo-node/init/internal/logservice"
	"os"
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

	logService, err := logservice.New("/logs")
	if err != nil {
		panic(err)
	}

	if err = initserver.New(logService, config).Run(); err != nil {
		panic(err)
	}
}
