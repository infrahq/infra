package cmd

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sort"

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
	client, err := apiClientFromConfig()
	if err != nil {
		return err
	}

	ctx, err := apiContextFromConfig()
	if err != nil {
		return err
	}

	destinations, res, err := client.DestinationsApi.ListDestinations(ctx).Execute()
	if err != nil {
		switch res.StatusCode {
		case http.StatusForbidden:
			fmt.Fprintln(os.Stderr, "Session has expired.")

			if err = login("", false); err != nil {
				return err
			}

			return list()

		default:
			return err
		}
	}

	sort.Slice(destinations, func(i, j int) bool {
		return destinations[i].Created > destinations[j].Created
	})

	kubeConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), nil).RawConfig()
	if err != nil {
		println(err.Error())
	}

	rows := []statusRow{}

	for _, d := range destinations {
		row := statusRow{
			Name:   d.Name,
			Status: "💻 → ❌ Can't reach internet",
		}

		if kube, ok := d.GetKubernetesOk(); ok {
			row.Endpoint = kube.Endpoint
			row.CertificateAuthorityData = []byte(kube.Ca)
			row.Type = "kubernetes"

			if kubeConfig.CurrentContext == row.Name {
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

	err = updateKubeconfig(destinations)
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
