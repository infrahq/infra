package cmd

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/logging"
	"github.com/lensesio/tableprinter"
	"k8s.io/client-go/tools/clientcmd"
)

type ListOptions struct {
	internal.Options `mapstructure:",squash"`
}

type statusRow struct {
	CurrentlySelected        string `header:"CURRENT"` // * if selected
	ID                       string `header:"ID"`
	Name                     string `header:"NAME"`
	Kind                     string `header:"KIND"`
	Status                   string `header:"STATUS"`
	Labels                   string `header:"LABELS"`
	Endpoint                 string // don't display in table
	CertificateAuthorityData []byte // don't display in table
}

func list(options *ListOptions) error {
	config, err := currentHostConfig()
	if err != nil {
		return err
	}

	client, err := apiClientFromConfig(options.Host)
	if err != nil {
		return err
	}

	ctx, err := apiContextFromConfig(options.Host)
	if err != nil {
		return err
	}

	users, res, err := client.UsersAPI.ListUsers(ctx).Email(config.Name).Execute()
	if err != nil {
		switch res.StatusCode {
		case http.StatusForbidden:
			fmt.Fprintln(os.Stderr, "Session has expired.")

			if err = login(&LoginOptions{Current: true}); err != nil {
				return err
			}

			return list(options)

		default:
			return errWithResponseContext(err, res)
		}
	}

	// This shouldn't be possible but check nonetheless
	switch {
	case len(users) < 1:
		return fmt.Errorf("user \"%s\" not found", config.Name)
	case len(users) > 1:
		return fmt.Errorf("found multiple users \"%s\"", config.Name)
	}

	user := users[0]

	kubeConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), nil).RawConfig()
	if err != nil {
		logging.S.Errorf("k8s error: %w", err)
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

	sort.SliceStable(rowsList, func(i, j int) bool {
		// Sort by combined name, descending
		return rowsList[i].Name+rowsList[i].ID < rowsList[j].Name+rowsList[j].ID
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

	if err := updateKubeconfig(user); err != nil {
		return err
	}

	return nil
}

func newRow(role api.Role, currentContext string) statusRow {
	row := statusRow{
		ID:     role.Destination.Name[:12],
		Name:   role.Destination.Alias,
		Status: "💻 → ❌ Can't reach internet",
		Labels: strings.Join(role.Destination.Labels, ", "),
	}

	if k8s, ok := role.Destination.GetKubernetesOk(); ok {
		row.Endpoint = k8s.Endpoint
		row.CertificateAuthorityData = []byte(k8s.Ca)
		row.Kind = "Kubernetes"
	}

	parts := strings.Split(currentContext, ":")
	// TODO (#546): check against user specified prefix
	if len(parts) >= 2 && parts[0] == "infra" {
		// check "infra:<ALIAS>[@<NAME>][:<NAMESPACE>]"
		parts := strings.Split(parts[1], "@")
		if parts[0] == role.Destination.Alias {
			if len(parts) > 1 && parts[1] == role.Destination.Name[:12] {
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
