package cmd

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"

	"github.com/goware/urlx"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/uid"
)

func clientConfig() clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.WarnIfAllMissing = false

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
}

func kubernetesSetContext(cluster, namespace string) error {
	config := clientConfig()

	kubeconfig, err := config.RawConfig()
	if err != nil {
		return err
	}

	name := cluster

	if namespace != "" {
		name = fmt.Sprintf("%s:%s", cluster, namespace)
	}

	// set friendly name based on user input rather than internal format
	friendlyName := strings.ReplaceAll(name, ":", ".")

	context := fmt.Sprintf("infra:%s", name)
	if _, ok := kubeconfig.Contexts[context]; !ok {
		return fmt.Errorf("context not found: %v", friendlyName)
	}

	kubeconfig.CurrentContext = context

	if err := clientcmd.WriteToFile(kubeconfig, config.ConfigAccess().GetDefaultFilename()); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Switched to context %q.\n", friendlyName)

	return nil
}

func updateKubeconfig(client *api.Client, identityPolymorphicID uid.PolymorphicID) error {
	destinations, err := client.ListDestinations(api.ListDestinationsRequest{})
	if err != nil {
		return nil
	}

	id, err := identityPolymorphicID.ID()
	if err != nil {
		return err
	}

	identity, err := client.GetIdentity(id)
	if err != nil {
		return err
	}

	grants, err := client.ListIdentityGrants(id)
	if err != nil {
		return err
	}

	groups, err := client.ListIdentityGroups(id)
	if err != nil {
		return err
	}

	for _, g := range groups {
		groupGrants, err := client.ListGroupGrants(g.ID)
		if err != nil {
			return err
		}

		grants = append(grants, groupGrants...)
	}

	return writeKubeconfig(identity, destinations, grants)
}

func writeKubeconfig(identity *api.Identity, destinations []api.Destination, grants []api.Grant) error {
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

		if len(parts) < 2 {
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

		var (
			url, ca string
			exists  bool
		)

		for _, d := range destinations {
			// eg resource:  "kubernetes.foo.bar"
			// eg dest name: "kubernetes.foo"
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

		// get TLS server name from the certificate
		block, _ := pem.Decode([]byte(ca))
		if block == nil {
			return fmt.Errorf("unknown certificate format")
		}

		certs, err := x509.ParseCertificates(block.Bytes)
		if err != nil {
			return err
		}

		if len(certs) == 0 {
			return fmt.Errorf("no certficates found")
		}

		kubeConfig.Clusters[context] = &clientcmdapi.Cluster{
			Server:                   u.String(),
			CertificateAuthorityData: []byte(ca),
		}

		kubeConfig.Contexts[context] = &clientcmdapi.Context{
			Cluster:   context,
			AuthInfo:  identity.Name,
			Namespace: namespace,
		}

		executable, err := os.Executable()
		if err != nil {
			return err
		}

		kubeConfig.AuthInfos[identity.Name] = &clientcmdapi.AuthInfo{
			Exec: &clientcmdapi.ExecConfig{
				Command:         executable,
				Args:            []string{"tokens", "add"},
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

	kubeConfigFilename := defaultConfig.ConfigAccess().GetDefaultFilename()
	if err := clientcmd.WriteToFile(kubeConfig, kubeConfigFilename); err != nil {
		return err
	}
	return nil
}
