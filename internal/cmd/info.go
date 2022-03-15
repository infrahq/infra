package cmd

import (
	"fmt"
	"os"
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

	if config.PolymorphicID.IsUser() {
		provider, err := client.GetProvider(config.ProviderID)
		if err != nil {
			return err
		}

		user, err := client.GetUser(config.ID)
		if err != nil {
			return err
		}

		groups, err := client.ListUserGroups(config.ID)
		if err != nil {
			return err
		}

		var groupsStr string
		for i, g := range groups {
			if i != 0 {
				groupsStr += ", "
			}

			groupsStr += g.Name
		}

		fmt.Fprintln(w)
		fmt.Fprintln(w, "Server:\t", config.Host)
		fmt.Fprintf(w, "Identity Provider:\t %s (%s)\n", provider.Name, provider.URL)
		fmt.Fprintln(w, "User:\t", user.Email)
		fmt.Fprintln(w)
	} else if config.PolymorphicID.IsMachine() {
		machine, err := client.GetMachine(config.ID)
		if err != nil {
			return err
		}

		fmt.Fprintln(w)
		fmt.Fprintln(w, "Server:\t", config.Host)
		fmt.Fprintln(w, "Machine User:\t", machine.Name)
		fmt.Fprintln(w)
	}

	return nil
}
