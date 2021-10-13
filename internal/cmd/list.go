package cmd

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/infrahq/infra/internal/api"
	"github.com/lensesio/tableprinter"
	"k8s.io/client-go/tools/clientcmd"
)

type statusRow struct {
	CurrentlySelected        string `header:"CURRENT"` // * if selected
	Name                     string `header:"NAME"`
	Type                     string `header:"TYPE"`
	Status                   string `header:"STATUS"`
	Endpoint                 string // don't display in table
	CertificateAuthorityData []byte // don't display in table
}

func list() error {
	config, err := currentRegistryConfig()
	if err != nil {
		return err
	}

	client, err := apiClientFromConfig()
	if err != nil {
		return err
	}

	ctx, err := apiContextFromConfig()
	if err != nil {
		return err
	}

	users, res, err := client.UsersApi.ListUsers(ctx).Email(config.Name).Execute()
	if err != nil {
		switch res.StatusCode {
		case http.StatusForbidden:
			fmt.Fprintln(os.Stderr, "Session has expired.")

			if err = login("", false, LoginOptions{}); err != nil {
				return err
			}

			return list()

		default:
			return err
		}
	}

	// This shouldn't be possible but check nonetheless
	switch {
	case len(users) < 1:
		return fmt.Errorf("User \"%s\" not found", config.Name)
	case len(users) > 1:
		return fmt.Errorf("Found multiple users \"%s\"", config.Name)
	}

	user := users[0]

	kubeConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), nil).RawConfig()
	if err != nil {
		println(err.Error())
	}

	// deduplicate rows
	rows := make(map[string]statusRow)

	for _, r := range user.Roles {
		rows[r.Destination.Id] = newRow(r, kubeConfig.CurrentContext)
	}

	for _, g := range user.Groups {
		for _, r := range g.Roles {
			rows[r.Destination.Id] = newRow(r, kubeConfig.CurrentContext)
		}
	}

	rowsList := make([]statusRow, 0)

	for _, r := range rows {
		rowsList = append(rowsList, r)
	}

	sort.Slice(rowsList, func(i, j int) bool {
		// Sort by combined name, descending
		return rowsList[i].Name < rowsList[j].Name
	})

	ok, err := canReachInternet()
	if !ok {
		for i := range rowsList {
			rowsList[i].Status = fmt.Sprintf("💻 → %s → ❌ Can't reach network: (%s)", globe(), err)
		}
	}

	if ok {
		for i, row := range rowsList {
			// check success case first for speed.
			ok, lastErr := canGetEngineStatus(row)
			if ok {
				rowsList[i].Status = "✅ OK"
				continue
			}
			// if we had a problem, check all the stops in order to figure out where it's getting stuck
			if ok, err := canConnectToEndpoint(row.Endpoint); !ok {
				rowsList[i].Status = fmt.Sprintf("💻 → %s → ❌ Can't reach endpoint %q (%s)", globe(), row.Endpoint, err)
				continue
			}

			if ok, err := canConnectToTLSEndpoint(row); !ok {
				rowsList[i].Status = fmt.Sprintf("💻 → %s → 🌥  → ❌ Can't negotiate TLS (%s)", globe(), err)
				continue
			}
			// if we made it here, we must be talking to something that isn't the engine.
			rowsList[i].Status = fmt.Sprintf("💻 → %s → 🌥  → 🔒 → ❌ Can't talk to infra engine (%s)", globe(), lastErr)
		}
	}

	printTable(rowsList)

	err = updateKubeconfig(user)
	if err != nil {
		return err
	}

	return nil
}

func newRow(role api.Role, currentContext string) statusRow {
	row := statusRow{
		Name:   role.Destination.Name,
		Status: "💻 → ❌ Can't reach internet",
	}

	if k8s, ok := role.Destination.GetKubernetesOk(); ok {
		row.Endpoint = k8s.Endpoint
		row.CertificateAuthorityData = []byte(k8s.Ca)
		row.Type = "Kubernetes"
	}

	parts := strings.Split(currentContext, ":")
	if len(parts) >= 2 && parts[0] == "infra" && parts[1] == role.Destination.Name {
		row.CurrentlySelected = "*"
	}

	return row
}

func globe() string {
	//nolint:gosec // No need for crypto random
	switch rand.Intn(3) {
	case 1:
		return "🌍"
	case 2:
		return "🌏"
	default:
		return "🌎"
	}
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
