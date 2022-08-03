package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/logging"
	"github.com/spf13/cobra"
)

func newSettingsCmd(cli *CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "settings",
		Short:   "Manage settings",
		Aliases: []string{"setting"},
		Group:   "Management commands:",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := rootPreRun(cmd.Flags()); err != nil {
				return err
			}
			return mustBeLoggedIn()
		},
	}

	cmd.AddCommand(newSettingsViewCmd(cli))
	cmd.AddCommand(newSettingsEditCmd(cli))
	cmd.AddCommand(newSettingsResetCmd(cli))
	return cmd
}

func newSettingsViewCmd(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:   "view",
		Short: "View settings",
		Args:  NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			logging.Debugf("call server: get settings")
			settings, err := client.GetSettings(&api.GetSettingsRequest{PasswordRequirements: true})
			if err != nil {
				return err
			}

			printSettings(cli, settings)
			return nil
		},
	}
}

type settingsEditOptions struct {
	Length    int
	Lowercase int
	Uppercase int
	Symbols   int
	Numbers   int
}

func newSettingsEditCmd(cli *CLI) *cobra.Command {
	var opts settingsEditOptions

	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Update settings",
		Args:  NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			logging.Debugf("call server: get settings")
			settings, err := client.GetSettings(&api.GetSettingsRequest{PasswordRequirements: true})
			if err != nil {
				return err
			}

			if opts.Length > -1 {
				settings.PasswordRequirements.LengthMin = opts.Length
			}
			if opts.Lowercase > -1 {
				settings.PasswordRequirements.LowercaseMin = opts.Lowercase
			}
			if opts.Uppercase > -1 {
				settings.PasswordRequirements.UppercaseMin = opts.Uppercase
			}
			if opts.Symbols > -1 {
				settings.PasswordRequirements.SymbolMin = opts.Symbols
			}
			if opts.Numbers > -1 {
				settings.PasswordRequirements.NumberMin = opts.Numbers
			}

			logging.Debugf("call server: update settings")
			return client.UpdateSettings(settings)
		},
	}

	cmd.Flags().IntVar(&opts.Length, "length", -1, "Set minimum password length")
	cmd.Flags().IntVar(&opts.Lowercase, "lowercase", -1, "Set minimum lowercase letters")
	cmd.Flags().IntVar(&opts.Uppercase, "uppercase", -1, "Set minimum uppercase letters")
	cmd.Flags().IntVar(&opts.Symbols, "symbols", -1, "Set minimum symbols")
	cmd.Flags().IntVar(&opts.Numbers, "numbers", -1, "Set minimum numbers")
	return cmd
}

func newSettingsResetCmd(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:     "reset",
		Short:   "Reset settings to default",
		Aliases: []string{"rm"},
		Args:    NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			logging.Debugf("call server: get settings")
			settings, err := client.GetSettings(&api.GetSettingsRequest{PasswordRequirements: true})
			if err != nil {
				return err
			}

			settings.PasswordRequirements.LengthMin = 8
			settings.PasswordRequirements.LowercaseMin = 0
			settings.PasswordRequirements.UppercaseMin = 0
			settings.PasswordRequirements.NumberMin = 0
			settings.PasswordRequirements.SymbolMin = 0

			logging.Debugf("call server: update settings")
			return client.UpdateSettings(settings)
		},
	}
}

func printSettings(cli *CLI, settings *api.Settings) {
	w := tabwriter.NewWriter(cli.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
	defer w.Flush()
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Password requirements:")
	fmt.Fprintln(w, "\t\tMinimum length:", settings.PasswordRequirements.LengthMin)
	fmt.Fprintln(w, "\t\tMinimum lowercase letters:", settings.PasswordRequirements.LowercaseMin)
	fmt.Fprintln(w, "\t\tMinimum uppercase letters:", settings.PasswordRequirements.UppercaseMin)
	fmt.Fprintln(w, "\t\tMinimum numbers:", settings.PasswordRequirements.NumberMin)
	fmt.Fprintln(w, "\t\tMinimum symbols:", settings.PasswordRequirements.SymbolMin)
	fmt.Fprintln(w)
}
