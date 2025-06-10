package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vmruntime",
	Short: "Baepo vm runtime",
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.Help(); err != nil {
			panic(err)
		}
	},
}

func Execute() error {
	return rootCmd.Execute()
}
