package main

import (
	"encoding/json"
	"github.com/baepo-cloud/baepo-node/init/initserver"
	"github.com/baepo-cloud/baepo-node/internal/types"
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

	err = initserver.New(config).Run()
	if err != nil {
		panic(err)
	}
}
