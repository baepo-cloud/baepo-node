package main

import "github.com/baepo-cloud/baepo-node/vmruntime/internal/cmd"

func main() {
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
