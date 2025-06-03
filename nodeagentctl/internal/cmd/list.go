package cmd

import (
	"connectrpc.com/connect"
	"github.com/baepo-cloud/baepo-cli/pkg/iostream"
	nodev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/node/v1"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List machines",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}

			res, err := client.ListMachines(cmd.Context(), connect.NewRequest(&nodev1pb.NodeListMachinesRequest{}))
			if err != nil {
				return err
			}

			ioStream.Array(res.Msg.Machines, []any{
				iostream.FieldConfig{
					DisplayName: "ID",
					FormatFunc: func(obj *nodev1pb.Machine) string {
						return obj.MachineId
					},
				},
				iostream.FieldConfig{
					DisplayName: "State",
					FormatFunc: func(obj *nodev1pb.Machine) string {
						return obj.State.String()
					},
				},
				iostream.FieldConfig{
					DisplayName: "Desired State",
					FormatFunc: func(obj *nodev1pb.Machine) string {
						return obj.DesiredState.String()
					},
				},
			}, iostream.ObjectOptions{Full: true})
			return nil
		},
	})
}
