package main

import (
	"encoding/json"
	"github.com/baepo-cloud/baepo-node/internal/initd"
	"os"
)

func main() {
	configFile, err := os.Open("/config.json")
	if err != nil {
		panic(err)
	}
	defer configFile.Close()

	var config initd.Config
	if err = json.NewDecoder(configFile).Decode(&config); err != nil {
		panic(err)
	}

	err = initd.Run(config)
	if err != nil {
		panic(err)
	}
}
