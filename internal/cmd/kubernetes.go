package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/goware/urlx"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/uid"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func clientConfig() clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.WarnIfAllMissing = false

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
}

func kubernetesSetContext(name string) error {
	config := clientConfig()

	kubeconfig, err := config.RawConfig()
	if err != nil {
		return err
	}

	if _, ok := kubeconfig.Contexts[name]; !ok {
		return fmt.Errorf("kubecontext %s not found", name)
	}

	kubeconfig.CurrentContext = name

	if err := clientcmd.WriteToFile(kubeconfig, config.ConfigAccess().GetDefaultFilename()); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Switched to context %q.\n", kubeconfig.CurrentContext)

	return nil
}

func updateKubeconfig(client *api.Client, userID uid.ID) error {
	destinations, err := client.ListDestinations(api.ListDestinationsRequest{})
	if err != nil {
		return nil
	}

	grants, err := client.ListUserGrants(userID)
	if err != nil {
		return nil
	}

	groups, err := client.ListUserGroups(userID)
	if err != nil {
		return nil
	}

	for _, g := range groups {
		groupGrants, err := client.ListGroupGrants(g.ID)
		if err != nil {
			return nil
		}

		grants = append(grants, groupGrants...)
	}

	return writeKubeconfig(destinations, grants)
}

func writeKubeconfig(destinations []api.Destination, grants []api.Grant) error {
	defaultConfig := clientConfig()
	kubeConfig, err := defaultConfig.RawConfig()
	if err != nil {
		return err
	}

	keep := make(map[string]bool)
	for _, g := range grants {

		parts := strings.Split(g.Resource, ".")

		kind := parts[0]

		if kind != "kubernetes" {
			continue
		}

		cluster := parts[1]

		var namespace string
		if len(parts) > 2 {
			namespace = parts[2]
		}

		context := "infra:" + cluster

		if namespace != "" {
			context += ":" + namespace
		}

		var url, ca string
		var exists bool
		for _, d := range destinations {
			if strings.HasPrefix(g.Resource, d.Name) {
				url = d.Connection.URL
				ca = d.Connection.CA
				exists = true
				break
			}
		}

		if !exists {
			continue
		}

		u, err := urlx.Parse(url)
		if err != nil {
			return err
		}

		u.Scheme = "https"

		logging.S.Debugf("creating kubeconfig for %s", context)

		kubeConfig.Clusters[context] = &clientcmdapi.Cluster{
			Server:                   fmt.Sprintf("%s/proxy", u.String()),
			CertificateAuthorityData: []byte(ca),
		}

		kubeConfig.Contexts[context] = &clientcmdapi.Context{
			Cluster:   context,
			AuthInfo:  context,
			Namespace: namespace,
		}

		executable, err := os.Executable()
		if err != nil {
			return err
		}

		kubeConfig.AuthInfos[context] = &clientcmdapi.AuthInfo{
			Exec: &clientcmdapi.ExecConfig{
				Command:         executable,
				Args:            []string{"tokens", "create"},
				APIVersion:      "client.authentication.k8s.io/v1beta1",
				InteractiveMode: clientcmdapi.IfAvailableExecInteractiveMode,
			},
		}

		keep[context] = true
	}

	// cleanup others
	for c := range kubeConfig.Contexts {
		parts := strings.Split(c, ":")

		if len(parts) < 1 {
			continue
		}

		if parts[0] != "infra" {
			continue
		}

		if _, ok := keep[c]; !ok {
			delete(kubeConfig.Clusters, c)
			delete(kubeConfig.Contexts, c)
			delete(kubeConfig.AuthInfos, c)
		}
	}

	kubeConfigFilename := defaultConfig.ConfigAccess().GetDefaultFilename()

	if err := clientcmd.WriteToFile(kubeConfig, kubeConfigFilename); err != nil {
		return err
	}

	return nil
}

func clearKubeconfig() error {
	defaultConfig := clientConfig()
	kubeConfig, err := defaultConfig.RawConfig()
	if err != nil {
		return err
	}

	for c := range kubeConfig.Contexts {
		parts := strings.Split(c, ":")

		if len(parts) < 1 {
			continue
		}

		if parts[0] != "infra" {
			continue
		}

		delete(kubeConfig.Clusters, c)
		delete(kubeConfig.Contexts, c)
		delete(kubeConfig.AuthInfos, c)
	}

	return nil
}
