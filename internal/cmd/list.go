package cmd

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/lensesio/tableprinter"

	"github.com/infrahq/infra/internal/api"
)

type listRow struct {
	CurrentlySelected        string `header:" "` // * if selected
	Name                     string `header:"NAME"`
	Kind                     string `header:"KIND"`
	ID                       string `header:"ID"`
	Labels                   string `header:"LABELS"`
	Endpoint                 string // don't display in table
	CertificateAuthorityData []byte // don't display in table
}

func list() error {
	config, err := currentHostConfig()
	if err != nil {
		return err
	}

	client, err := defaultAPIClient()
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

			return list()
		}

		return err
	}

	switch {
	case len(users) < 1:
		//lint:ignore ST1005, user facing error
		return fmt.Errorf("User %q not found", config.Name)
	case len(users) > 1:
		//lint:ignore ST1005, user facing error
		return fmt.Errorf("Found multiple users %q in Infra, the server configuration is invalid", config.Name)
	}

	rows := make(map[string]listRow)
	// todo: iterate grants and groups and call newRow() to display them?

	rowsList := make([]listRow, 0)
	for _, r := range rows {
		rowsList = append(rowsList, r)
	}

	sort.SliceStable(rowsList, func(i, j int) bool {
		// Sort by combined name, descending
		return rowsList[i].Name+rowsList[i].ID < rowsList[j].Name+rowsList[j].ID
	})

	printTable(rowsList)

	// if err := updateKubeconfig(user); err != nil {
	// 	return err
	// }

	return nil
}

func newRow(grant api.Grant, currentContext string) listRow {
	row := listRow{}

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
