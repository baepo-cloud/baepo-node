package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "run <config file>",
		Args:  cobra.MinimumNArgs(1),
		Short: "Run a machine",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	})
}
