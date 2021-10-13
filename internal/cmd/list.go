package cmd

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sort"

	"github.com/infrahq/infra/internal/api"
	"github.com/lensesio/tableprinter"
	"k8s.io/client-go/tools/clientcmd"
)

type statusRow struct {
	CurrentlySelected        string `header:"CURRENT"` // * if selected
	Name                     string `header:"NAME"`
	Type                     string `header:"TYPE"`
	Namespace                string `header:"NAMESPACE"`
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

	rows := []statusRow{}

	for _, r := range user.Roles {
		rows = append(rows, newRow(r, kubeConfig.CurrentContext))
	}

	for _, g := range user.Groups {
		for _, r := range g.Roles {
			rows = append(rows, newRow(r, kubeConfig.CurrentContext))
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		// Sort by combined name, descending
		return rows[i].Name+rows[i].Namespace < rows[j].Name+rows[j].Namespace
	})

	ok, err := canReachInternet()
	if !ok {
		for i := range rows {
			rows[i].Status = fmt.Sprintf("💻 → %s → ❌ Can't reach network: (%s)", globe(), err)
		}
	}

	if ok {
		for i, row := range rows {
			// check success case first for speed.
			ok, lastErr := canGetEngineStatus(row)
			if ok {
				rows[i].Status = "✅ OK"
				continue
			}
			// if we had a problem, check all the stops in order to figure out where it's getting stuck
			if ok, err := canConnectToEndpoint(row.Endpoint); !ok {
				rows[i].Status = fmt.Sprintf("💻 → %s → ❌ Can't reach endpoint %q (%s)", globe(), row.Endpoint, err)
				continue
			}

			if ok, err := canConnectToTLSEndpoint(row); !ok {
				rows[i].Status = fmt.Sprintf("💻 → %s → 🌥  → ❌ Can't negotiate TLS (%s)", globe(), err)
				continue
			}
			// if we made it here, we must be talking to something that isn't the engine.
			rows[i].Status = fmt.Sprintf("💻 → %s → 🌥  → 🔒 → ❌ Can't talk to infra engine (%s)", globe(), lastErr)
		}
	}

	printTable(rows)

	err = updateKubeconfig(user)
	if err != nil {
		return err
	}

	return nil
}

func newRow(role api.Role, currentContext string) statusRow {
	row := statusRow{
		Name:      role.Destination.Name,
		Status:    "💻 → ❌ Can't reach internet",
		Namespace: role.Namespace,
	}

	if k8s, ok := role.Destination.GetKubernetesOk(); ok {
		row.Endpoint = k8s.Endpoint
		row.CertificateAuthorityData = []byte(k8s.Ca)
		row.Type = "Kubernetes"
	}

	var contextName string
	if role.Namespace != "" {
		contextName = fmt.Sprintf("infra:%s:%s", role.Destination.Name, role.Namespace)
	} else {
		contextName = fmt.Sprintf("infra:%s", role.Destination.Name)
	}

	if currentContext == contextName {
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
