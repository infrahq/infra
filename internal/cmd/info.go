package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

func info() error {
	config, err := currentHostConfig()
	if err != nil {
		return err
	}

	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
	defer w.Flush()

	id := config.PolymorphicID
	if id == "" {
		return fmt.Errorf("no active identity")
	}

	switch {
	case id.IsUser():
		userID, err := id.ID()
		if err != nil {
			return err
		}

		provider, err := client.GetProvider(config.ProviderID)
		if err != nil {
			return err
		}

		user, err := client.GetUser(userID)
		if err != nil {
			return err
		}

		userGroups, err := client.ListUserGroups(userID)
		if err != nil {
			return err
		}

		groups := make([]string, 0)
		for _, g := range userGroups {
			groups = append(groups, g.Name)
		}

		fmt.Fprintln(w)
		fmt.Fprintln(w, "Server:\t", config.Host)
		fmt.Fprintf(w, "Identity Provider:\t %s (%s)\n", provider.Name, provider.URL)
		fmt.Fprintln(w, "User:\t", user.Email)
		fmt.Fprintln(w, "Groups:\t", strings.Join(groups, ", "))
		fmt.Fprintln(w)
	case id.IsMachine():
		machineID, err := id.ID()
		if err != nil {
			return err
		}

		machine, err := client.GetMachine(machineID)
		if err != nil {
			fmt.Fprintln(os.Stderr, "6.3")
			return err
		}

		fmt.Fprintln(w)
		fmt.Fprintln(w, "Server:\t", config.Host)
		fmt.Fprintln(w, "Machine User:\t", machine.Name)
		fmt.Fprintln(w)
	default:
		return fmt.Errorf("unsupported identity for operation: %s", id)
	}

	return nil
}
