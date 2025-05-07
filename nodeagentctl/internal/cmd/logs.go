package cmd

import (
	"connectrpc.com/connect"
	"fmt"
	nodev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/node/v1"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "logs <machine-id>",
		Short: "Get logs of a machine",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}

			if len(args) < 1 {
				ioStream.Error("You must provide a machine ID.")
				return nil
			}

			res, err := client.GetMachineLogs(cmd.Context(), connect.NewRequest(&nodev1pb.NodeGetMachineLogsRequest{
				MachineId: args[0],
			}))
			if err != nil {
				return err
			}

			for res.Receive() {
				msg := res.Msg()
				fmt.Println(msg.String())
			}
			return nil
		},
	})
}
