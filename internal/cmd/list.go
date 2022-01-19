package cmd

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/lensesio/tableprinter"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/logging"
)

type ListOptions struct {
	internal.Options `mapstructure:",squash"`
}

type listRow struct {
	CurrentlySelected        string `header:" "` // * if selected
	Name                     string `header:"NAME"`
	Kind                     string `header:"KIND"`
	ID                       string `header:"ID"`
	Labels                   string `header:"LABELS"`
	Endpoint                 string // don't display in table
	CertificateAuthorityData []byte // don't display in table
}

func list(options *ListOptions) error {
	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	config, err := currentHostConfig()
	if err != nil {
		return err
	}

	users, err := client.ListUsers(config.Name)
	if err != nil {
		if errors.Is(err, api.ErrForbidden) {
			fmt.Fprintln(os.Stderr, "Session has expired.")

			if err = login(&LoginOptions{Current: true}); err != nil {
				return err
			}

			return list(options)
		}

		return err
	}

	switch {
	case len(users) < 1:
		//lint:ignore ST1005, user facing error
		return fmt.Errorf("User %q not found, is this account still valid?", config.Name)
	case len(users) > 1:
		//lint:ignore ST1005, user facing error
		return fmt.Errorf("Found multiple users %q in Infra, the server configuration is invalid", config.Name)
	}

	user := users[0]

	kubeConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), nil).RawConfig()
	if err != nil {
		logging.S.Errorf("k8s error: %w", err)
	}

	// deduplicate rows
	rows := make(map[string]listRow)
	for _, r := range user.Grants {
		rows[r.Destination.ID] = newRow(r, kubeConfig.CurrentContext)
	}

	for _, g := range user.Groups {
		for _, r := range g.Grants {
			rows[r.Destination.ID] = newRow(r, kubeConfig.CurrentContext)
		}
	}

	rowsList := make([]listRow, 0)
	for _, r := range rows {
		rowsList = append(rowsList, r)
	}

	sort.SliceStable(rowsList, func(i, j int) bool {
		// Sort by combined name, descending
		return rowsList[i].Name+rowsList[i].ID < rowsList[j].Name+rowsList[j].ID
	})

	printTable(rowsList)

	if err := updateKubeconfig(user); err != nil {
		return err
	}

	return nil
}

func newRow(grant api.Grant, currentContext string) listRow {
	row := listRow{
		ID:     grant.Destination.NodeID[:12],
		Name:   grant.Destination.Name,
		Labels: strings.Join(grant.Destination.Labels, ", "),
	}

	if k8s := grant.Destination.Kubernetes; k8s != nil {
		row.Endpoint = k8s.Endpoint
		row.CertificateAuthorityData = []byte(k8s.CA)
		row.Kind = "kubernetes"
	}

	parts := strings.Split(currentContext, ":")
	// TODO (#546): check against user specified prefix
	if len(parts) >= 2 && parts[0] == "infra" {
		// check "infra:<ALIAS>[@<NAME>][:<NAMESPACE>]"
		parts := strings.Split(parts[1], "@")
		if parts[0] == grant.Destination.Name {
			if len(parts) > 1 && parts[1] == grant.Destination.NodeID[:12] {
				// check "<ALIAS>@<NAME>"
				row.CurrentlySelected = "*"
			} else if len(parts) == 1 {
				// check "<ALIAS>"
				row.CurrentlySelected = "*"
			}
		}
	}

	return row
}

func printTable(data interface{}) {
	table := tableprinter.New(os.Stdout)

	table.AutoFormatHeaders = true
	table.HeaderAlignment = tableprinter.AlignLeft
	table.AutoWrapText = false
	table.DefaultAlignment = tableprinter.AlignLeft
	table.CenterSeparator = ""
	table.ColumnSeparator = ""
	table.RowSeparator = ""
	table.HeaderLine = false
	table.BorderBottom = false
	table.BorderLeft = false
	table.BorderRight = false
	table.BorderTop = false
	table.Print(data)
}
