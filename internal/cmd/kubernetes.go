package cmd

import (
	"context"
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

func kubernetesSetContext(cli *CLI, cluster, namespace string) error {
	config := clientConfig()

	kubeConfig, err := config.RawConfig()
	if err != nil {
		return err
	}

	name := strings.TrimPrefix(cluster, "infra:")

	// set friendly name based on user input rather than internal format
	friendlyName := strings.ReplaceAll(name, ":", ".")

	contextName := fmt.Sprintf("infra:%s", name)
	kubeContext, ok := kubeConfig.Contexts[contextName]
	if !ok {
		return fmt.Errorf("context not found: %v", friendlyName)
	}

	if namespace != "" {
		kubeContext.Namespace = namespace
	}

	kubeConfig.CurrentContext = contextName
	kubeConfig.Contexts[contextName] = kubeContext

	configFile := config.ConfigAccess().GetDefaultFilename()
	if err := safelyWriteConfigToFile(kubeConfig, configFile); err != nil {
		return err
	}

	fmt.Fprintf(cli.Stderr, "Switched to context %q.\n", friendlyName)
	return nil
}

func updateKubeConfig(client *api.Client, id uid.ID) error {
	ctx := context.TODO()
	destinations, err := listAll(ctx, client.ListDestinations, api.ListDestinationsRequest{})
	if err != nil {
		return err
	}

	user, err := client.GetUser(ctx, id)
	if err != nil {
		return err
	}

	grants, err := listAll(ctx, client.ListGrants, api.ListGrantsRequest{User: id, ShowInherited: true})
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

	type clusterContext struct {
		Namespace string
		URL       string
		CA        []byte
	}

	infraContexts := make(map[string]clusterContext)

	for _, g := range grants {
		parts := strings.Split(g.Resource, ".")
		cluster := parts[0]

		var namespace string
		if len(parts) > 1 {
			namespace = parts[1]
		}

		if namespace == "default" {
			namespace = ""
		}

		contextName := "infra:" + cluster
		if _, ok := infraContexts[contextName]; ok && namespace != "" {
			continue
		}

		var infraContext clusterContext
		for _, d := range destinations {
			if !isResourceForDestination(g.Resource, d.Name) {
				continue
			}

			if isDestinationAvailable(d) {
				infraContext = clusterContext{
					URL: d.Connection.URL,
					CA:  []byte(d.Connection.CA),
				}
				break
			}
		}

		if infraContext.URL == "" {
			continue
		}

		infraContext.Namespace = namespace
		infraContexts[contextName] = infraContext
	}

	for contextName, infraContext := range infraContexts {
		logging.Debugf("creating kubeconfig for %s", contextName)

		u, err := urlx.Parse(infraContext.URL)
		if err != nil {
			return err
		}

		u.Scheme = "https"

		kubeConfig.Clusters[contextName] = &clientcmdapi.Cluster{
			Server:                   u.String(),
			CertificateAuthorityData: infraContext.CA,
		}

		// use existing kubeContext if possible which may contain
		// user-defined overrides. preserve them if possible
		kubeContext, ok := kubeConfig.Contexts[contextName]
		if !ok {
			kubeContext = &clientcmdapi.Context{
				Cluster:   contextName,
				AuthInfo:  user.Name,
				Namespace: infraContext.Namespace,
			}
		}

		kubeConfig.Contexts[contextName] = kubeContext

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

		if _, ok := infraContexts[id]; !ok {
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
