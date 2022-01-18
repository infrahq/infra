package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/goware/urlx"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/logging"
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

func updateKubeconfig() error {
	config, err := currentHostConfig()
	if err != nil {
		return err
	}

	if config.ID == 0 {
		return fmt.Errorf("no active user")
	}

	defaultConfig := clientConfig()
	kubeConfig, err := defaultConfig.RawConfig()
	if err != nil {
		return err
	}

	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	grants, err := client.ListUserGrants(config.ID)
	if err != nil {
		return err
	}

	contexts := make(map[string]bool)

	for _, grant := range grants {
		parts := strings.Split(grant.Resource, ".")

		kind := parts[0]

		if kind != "kubernetes" {
			continue
		}

		cluster := parts[1]

		var namespace string
		if len(parts) > 2 {
			namespace = parts[2]
		}

		destinations, err := client.ListDestinations(api.ListDestinationsRequest{Name: kind + "." + cluster})
		if err != nil {
			return err
		}

		if len(destinations) == 0 {
			continue
		}

		context := "infra:" + cluster

		if namespace != "" {
			context += ":" + namespace
		}

		url := destinations[0].Connection.URL
		ca := destinations[0].Connection.CA

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

		contexts[context] = true
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

		if _, ok := contexts[c]; !ok {
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
