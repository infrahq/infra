package cmd

import (
	"github.com/infrahq/infra/internal/server"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Infra",
	Run: func(cmd *cobra.Command, args []string) {
		server.Run()
	},
}
