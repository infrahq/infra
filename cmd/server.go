package cmd

import (
	"github.com/infrahq/infra/server"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the infra server",
	Run: func(cmd *cobra.Command, args []string) {
		server.Run()
	},
}
