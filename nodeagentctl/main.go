package main

import (
	"github.com/baepo-cloud/baepo-node/nodeagentctl/internal/cmd"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
