package main

import (
	"fmt"
	"os"

	"github.com/infrahq/infra/internal/server"
	"github.com/spf13/cobra"
)

func main() {
	var (
		installCmd = &cobra.Command{
			Use:   "install",
			Short: "Install Infra",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("NOT IMPLEMENTED")
			},
		}
	)

	var (
		serverPort int
		serverCmd  = &cobra.Command{
			Use:   "server",
			Short: "Run the infra server",
			Run: func(cmd *cobra.Command, args []string) {
				config := &server.Options{
					Port: serverPort,
				}
				if err := server.Run(config); err != nil {
					panic(err)
				}
			},
		}
	)
	serverCmd.Flags().IntVarP(&serverPort, "port", "p", 3090, "Port to listen on")

	var (
		usersAddCmd = &cobra.Command{
			Use:   "add",
			Short: "Add a user",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("NOT IMPLEMENTED")
			},
		}
	)

	var (
		usersRemoveCmd = &cobra.Command{
			Use:     "remove",
			Short:   "Remove a user",
			Aliases: []string{"rm"},
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("NOT IMPLEMENTED")
			},
		}
	)

	var (
		usersCmd = &cobra.Command{
			Use:     "users",
			Short:   "List & manage users",
			Aliases: []string{"user"},
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("NOT IMPLEMENTED")
			},
		}
	)
	usersCmd.AddCommand(usersAddCmd)
	usersCmd.AddCommand(usersRemoveCmd)

	var (
		rolesAddCmd = &cobra.Command{
			Use:   "add",
			Short: "Add a role",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("NOT IMPLEMENTED")
			},
		}
	)

	var (
		rolesRemoveCmd = &cobra.Command{
			Use:     "remove",
			Short:   "Remove a role",
			Aliases: []string{"rm"},
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("NOT IMPLEMENTED")
			},
		}
	)

	var (
		rolesCmd = &cobra.Command{
			Use:     "roles",
			Short:   "List & manage roles",
			Aliases: []string{"role"},
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("NOT IMPLEMENTED")
			},
		}
	)
	rolesCmd.AddCommand(rolesAddCmd)
	rolesCmd.AddCommand(rolesRemoveCmd)

	var (
		resourcesAddCmd = &cobra.Command{
			Use:   "add",
			Short: "Add a resource",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("NOT IMPLEMENTED")
			},
		}
	)

	var (
		resourcesRemoveCmd = &cobra.Command{
			Use:     "remove",
			Short:   "Remove a resource",
			Aliases: []string{"rm"},
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("NOT IMPLEMENTED")
			},
		}
	)

	var (
		resourcesCmd = &cobra.Command{
			Use:     "resources",
			Short:   "List & manage resources",
			Aliases: []string{"resource"},
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("NOT IMPLEMENTED")
			},
		}
	)
	resourcesCmd.AddCommand(resourcesAddCmd)
	resourcesCmd.AddCommand(resourcesRemoveCmd)

	var rootCmd = &cobra.Command{
		Use:   "infra",
		Short: "Infra: user infrastructure",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(usersCmd)
	rootCmd.AddCommand(rolesCmd)
	rootCmd.AddCommand(resourcesCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
