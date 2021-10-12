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

	// deduplicate destinations from combination of user roles and group roles
	destinations := make(map[string]api.Destination)
	for _, r := range user.Roles {
		destinations[r.Destination.Id] = r.Destination
	}

	for _, g := range user.Groups {
		for _, r := range g.Roles {
			destinations[r.Destination.Id] = r.Destination
		}
	}

	destinationList := make([]api.Destination, 0)
	for _, d := range destinations {
		destinationList = append(destinationList, d)
	}

	sort.Slice(destinationList, func(i, j int) bool {
		return destinationList[i].Name > destinationList[j].Name
	})

	kubeConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), nil).RawConfig()
	if err != nil {
		println(err.Error())
	}

	rows := []statusRow{}

	for _, d := range destinationList {
		row := statusRow{
			Name:   d.Name,
			Status: "💻 → ❌ Can't reach internet",
		}

		if kube, ok := d.GetKubernetesOk(); ok {
			row.Endpoint = kube.Endpoint
			row.CertificateAuthorityData = []byte(kube.Ca)
			row.Type = "kubernetes"

			if kubeConfig.CurrentContext == fmt.Sprintf("infra:%s", row.Name) {
				row.CurrentlySelected = "*"
			}
		}

		rows = append(rows, row)
	}

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
