package cmd

import (
	"connectrpc.com/connect"
	"fmt"
	nodev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/node/v1"
	"github.com/spf13/cobra"
	"os"
)

func init() {
	cmd := &cobra.Command{
		Use:   "logs <machine-id>",
		Short: "Get logs of a machine",
	}
	follow := cmd.Flags().BoolP("follow", "f", false, "Follow log output")
	container := cmd.Flags().StringP("container", "c", "", "Filter by container name")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		if len(args) < 1 {
			ioStream.Error("You must provide a machine ID.")
			return nil
		}

		req := &nodev1pb.NodeGetMachineLogsRequest{
			MachineId: args[0],
			Follow:    *follow,
		}
		if *container != "" {
			req.ContainerName = container
		}
		res, err := client.GetMachineLogs(cmd.Context(), connect.NewRequest(req))
		if err != nil {
			return err
		}

		for res.Receive() {
			msg := res.Msg()
			writer := os.Stdout
			if msg.Error {
				writer = os.Stderr
			}

			_, _ = fmt.Fprint(writer, string(msg.Content))
		}
		return nil
	}
	rootCmd.AddCommand(cmd)
}
