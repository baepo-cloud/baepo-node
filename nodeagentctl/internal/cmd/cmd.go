package cmd

import (
	"context"
	"github.com/baepo-cloud/baepo-cli/pkg/iostream"
	"github.com/baepo-cloud/baepo-proto/go/baepo/node/v1/nodev1pbconnect"
	"github.com/spf13/cobra"
	"net"
	"net/http"
	"os"
	"path/filepath"
)

var (
	rootCmd = &cobra.Command{
		Use:   "nodeagentctl",
		Short: "Baepo node agent cli",
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				panic(err)
			}
		},
	}

	ioStream = iostream.New(false)
)

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		return err
	}

	return nil
}

func newClient() (nodev1pbconnect.NodeServiceClient, error) {
	storageDir := os.Getenv("NODE_STORAGE_DIRECTORY")
	if storageDir == "" {
		storageDir = "/var/lib/baepo"
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", filepath.Join(storageDir, "agent.sock"))
			},
		},
	}
	return nodev1pbconnect.NewNodeServiceClient(httpClient, "http://agent"), nil
}
