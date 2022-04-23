package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// ExactArgs validates that a cobra command is executed with the exactly
// number of command line arguments, otherwise it returns an error that includes
// the usage string.
func ExactArgs(number int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == number {
			return nil
		}
		return fmt.Errorf(
			"%q requires exactly %d %s.\nSee \"%s --help\".\n\nUsage:  %s\n",
			cmd.CommandPath(),
			number,
			pluralize("argument", number),
			cmd.CommandPath(),
			cmd.UseLine())
	}
}

func pluralize(word string, number int) string {
	if number == 1 {
		return word
	}
	return word + "s"
}

// MaxArgs validates that a cobra command is executed with at most max command
// line arguments, otherwise it returns an error that includes the usage string.
func MaxArgs(max int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) <= max {
			return nil
		}
		return fmt.Errorf(
			"%q accepts at most %d %s.\nSee \"%s --help\".\n\nUsage:  %s\n",
			cmd.CommandPath(),
			max,
			pluralize("argument", max),
			cmd.CommandPath(),
			cmd.UseLine())
	}
}

// NoArgs validates that a cobra command is executed with no arguments, otherwise
// it returns an error that includes the usage string.
func NoArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	return fmt.Errorf(
		"%q accepts no arguments.\nSee \"%s --help\".\n\nUsage:  %s\n",
		cmd.CommandPath(),
		cmd.CommandPath(),
		cmd.UseLine())
}
