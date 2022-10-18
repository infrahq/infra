package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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

	name := strings.TrimPrefix(cluster, "infra:")

	if namespace != "" {
		name = fmt.Sprintf("%s:%s", name, namespace)
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

func updateKubeConfig(client *api.Client, id uid.ID) error {
	destinations, err := listAll(client.ListDestinations, api.ListDestinationsRequest{})
	if err != nil {
		return err
	}

	user, err := client.GetUser(id)
	if err != nil {
		return err
	}

	grants, err := listAll(client.ListGrants, api.ListGrantsRequest{User: id, ShowInherited: true})
	if err != nil {
		return err
	}

	return writeKubeconfig(user, destinations, grants)
}

func writeKubeconfig(user *api.User, destinations []api.Destination, grants []api.Grant) error {
	defaultConfig := clientConfig()

	kubeConfig, err := defaultConfig.RawConfig()
	if err != nil {
		return err
	}

	keep := make(map[string]bool)

	for _, g := range grants {
		parts := strings.Split(g.Resource, ".")

		cluster := parts[0]

		var namespace string
		if len(parts) > 1 {
			namespace = parts[1]
		}

		context := "infra:" + cluster

		if namespace != "" {
			context += ":" + namespace
		}

		var (
			url    string
			ca     []byte
			exists bool
		)

		for _, d := range destinations {
			if !isResourceForDestination(g.Resource, d.Name) {
				continue
			}

			if isDestinationAvailable(d) {
				url = d.Connection.URL
				ca = []byte(d.Connection.CA)
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

		logging.Debugf("creating kubeconfig for %s", context)

		kubeConfig.Clusters[context] = &clientcmdapi.Cluster{
			Server:                   u.String(),
			CertificateAuthorityData: ca,
		}

		// use existing kubeContext if possible which may contain
		// user-defined overrides. preserve them if possible
		kubeContext, ok := kubeConfig.Contexts[context]
		if !ok {
			kubeContext = &clientcmdapi.Context{
				Cluster:   context,
				AuthInfo:  user.Name,
				Namespace: namespace,
			}
		}

		if namespace != "" {
			// force the namespace if defined by Infra
			if kubeContext.Namespace != namespace {
				kubeContext.Namespace = namespace
			}
		}

		kubeConfig.Contexts[context] = kubeContext

		executable, err := os.Executable()
		if err != nil {
			return err
		}

		kubeConfig.AuthInfos[user.Name] = &clientcmdapi.AuthInfo{
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
	for id, ctx := range kubeConfig.Contexts {
		parts := strings.Split(id, ":")

		if len(parts) < 1 {
			continue
		}

		if parts[0] != "infra" {
			continue
		}

		if _, ok := keep[id]; !ok {
			delete(kubeConfig.AuthInfos, ctx.AuthInfo)
			delete(kubeConfig.Clusters, ctx.Cluster)
			delete(kubeConfig.Contexts, id)
		}
	}

	configFile := defaultConfig.ConfigAccess().GetDefaultFilename()

	return safelyWriteConfigToFile(kubeConfig, configFile)
}

// safelyWriteConfigToFile creates a temp file, then overwrites the target
func safelyWriteConfigToFile(kubeConfig clientcmdapi.Config, fileToWrite string) error {
	// get the directory of the file we're writing to avoid cross-filesystem moves
	configDir := filepath.Dir(fileToWrite)
	if err := os.MkdirAll(configDir, os.ModePerm); err != nil {
		return err
	}

	temp, err := ioutil.TempFile(configDir, "infra-kube-config-")
	if err != nil {
		return fmt.Errorf("failed to create temp kube config file: %w", err)
	}

	// write the new config to a temporary file then move it in an atomic operation
	// this ensures we don't wipe the kube config in the case of an interrupt
	if err := clientcmd.WriteToFile(kubeConfig, temp.Name()); err != nil {
		if nestedErr := temp.Close(); err != nil {
			logging.L.Debug().Err(nestedErr).Msg("failed to close temp config file on write error")
		}
		if nestedErr := os.Remove(temp.Name()); err != nil {
			logging.L.Debug().Err(nestedErr).Msg("failed to delete temp config file on write error")
		}
		return fmt.Errorf("could not write kube config to temp file: %w", err)
	}

	if err := temp.Close(); err != nil {
		return fmt.Errorf("failed to close temp kube config file: %w", err)
	}

	// move the temp file to overwrite the kube config
	err = os.Rename(temp.Name(), fileToWrite)
	if err != nil {
		return fmt.Errorf("could not overwrite kube config: %w", err)
	}

	return nil
}

func clearKubeconfig() error {
	defaultConfig := clientConfig()

	kubeConfig, err := defaultConfig.RawConfig()
	if err != nil {
		return err
	}

	for id, ctx := range kubeConfig.Contexts {
		parts := strings.Split(id, ":")

		if len(parts) < 1 {
			continue
		}

		if parts[0] != "infra" {
			continue
		}

		delete(kubeConfig.AuthInfos, ctx.AuthInfo)
		delete(kubeConfig.Clusters, ctx.Cluster)
		delete(kubeConfig.Contexts, id)
	}

	if strings.HasPrefix(kubeConfig.CurrentContext, "infra:") {
		kubeConfig.CurrentContext = ""
	}

	kubeConfigFilename := defaultConfig.ConfigAccess().GetDefaultFilename()
	if err := clientcmd.WriteToFile(kubeConfig, kubeConfigFilename); err != nil {
		return err
	}
	return nil
}
