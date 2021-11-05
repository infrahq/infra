package cmd

import (
	"os"
	"fmt"
	"strings"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/logging"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type KubernetesOptions struct {
	Name             string
	LabelSelector    []string `mapstructure:"labels"`
	KindSelector     string   `mapstructure:"kind"`
	IDSelector       string   `mapstructure:"id"`
	internal.Options `mapstructure:",squash"`
}

func newKubernetesUseCmd() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   "use [NAME]",
		Short: "Set the Kubernetes current context",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}

			options := KubernetesOptions{
				Name: name,
			}

			if err := internal.ParseOptions(cmd, &options); err != nil {
				return err
			}

			return kubernetesUseContext(&options)
		},
	}

	cmd.Flags().StringP("id", "i", "", "ID")
	cmd.Flags().StringP("kind", "k", "", "kind")
	cmd.Flags().StringSliceP("labels", "L", []string{}, "labels")

	return cmd, nil
}

func kubernetesUseContext(options *KubernetesOptions) error {
	logging.S.Infof("%s", options)
	return nil
}

func clientConfig() clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.WarnIfAllMissing = false

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
}

func updateKubeconfig(user api.User) error {
	defaultConfig := clientConfig()

	kubeConfig, err := defaultConfig.RawConfig()
	if err != nil {
		return err
	}

	// deduplicate roles
	aliases := make(map[string]map[string]bool)
	roles := make(map[string]api.Role)

	for _, r := range user.Roles {
		if _, ok := aliases[r.Destination.Alias]; !ok {
			aliases[r.Destination.Alias] = make(map[string]bool)
		}

		aliases[r.Destination.Alias][r.Destination.Name] = true
		roles[r.Id] = r
	}

	for _, g := range user.Groups {
		for _, r := range g.Roles {
			if _, ok := aliases[r.Destination.Alias]; !ok {
				aliases[r.Destination.Alias] = make(map[string]bool)
			}

			aliases[r.Destination.Alias][r.Destination.Name] = true
			roles[r.Id] = r
		}
	}

	for _, role := range roles {
		name := role.Destination.Name[:12]
		alias := role.Destination.Alias

		// TODO (#546): allow user to specify prefix, default ""
		// format: "infra:<ALIAS>"
		contextName := fmt.Sprintf("infra:%s", alias)

		if len(aliases[alias]) > 1 {
			// disambiguate destination by appending the ID
			// format: "infra:<ALIAS>@<NAME>"
			contextName = fmt.Sprintf("%s@%s", contextName, name)
		}

		if role.Namespace != "" {
			// destination is scoped to a namespace
			// format: "infra:<ALIAS>[@<NAME>]:<NAMESPACE>"
			contextName = fmt.Sprintf("%s:%s", contextName, role.Namespace)
		}

		logging.S.Debugf("creating kubeconfig for %s", contextName)

		kubeConfig.Clusters[contextName] = &clientcmdapi.Cluster{
			Server:                   fmt.Sprintf("https://%s/proxy", role.Destination.Kubernetes.Endpoint),
			CertificateAuthorityData: []byte(role.Destination.Kubernetes.Ca),
		}

		kubeConfig.Contexts[contextName] = &clientcmdapi.Context{
			Cluster:   contextName,
			AuthInfo:  contextName,
			Namespace: role.Namespace,
		}

		executable, err := os.Executable()
		if err != nil {
			return err
		}

		kubeConfig.AuthInfos[contextName] = &clientcmdapi.AuthInfo{
			Exec: &clientcmdapi.ExecConfig{
				Command:         executable,
				Args:            []string{"tokens", "create", role.Destination.Name},
				APIVersion:      "client.authentication.k8s.io/v1beta1",
				InteractiveMode: clientcmdapi.IfAvailableExecInteractiveMode,
			},
		}
	}

	for contextName := range kubeConfig.Contexts {
		parts := strings.Split(contextName, ":")

		// shouldn't be possible but we don't actually care
		if len(parts) < 1 {
			continue
		}

		if parts[0] != "infra" {
			continue
		}

		found := false

		for _, r := range roles {
			parts := strings.Split(parts[1], "@")

			switch {
			case len(parts) == 1:
				found = parts[0] == r.Destination.Alias
			case len(parts) > 1:
				found = parts[0] == r.Destination.Alias && parts[1] == r.Destination.Name[:12]
			}

			if found {
				break
			}
		}

		if !found {
			delete(kubeConfig.Clusters, contextName)
			delete(kubeConfig.Contexts, contextName)
			delete(kubeConfig.AuthInfos, contextName)
		}
	}

	kubeConfigFilename := defaultConfig.ConfigAccess().GetDefaultFilename()

	if err := clientcmd.WriteToFile(kubeConfig, kubeConfigFilename); err != nil {
		return err
	}

	return nil
}

func switchToFirstInfraContext() (string, error) {
	defaultConfig := clientConfig()

	kubeConfig, err := defaultConfig.RawConfig()
	if err != nil {
		return "", err
	}

	resultContext := ""

	if kubeConfig.Contexts[kubeConfig.CurrentContext] != nil && strings.HasPrefix(kubeConfig.CurrentContext, "infra:") {
		// if the current context is an infra-controlled context, stay there
		resultContext = kubeConfig.CurrentContext
	} else {
		for _, c := range kubeConfig.Contexts {
			if !strings.HasPrefix(c.Cluster, "infra:") {
				continue
			}

			// prefer a context with "default" or no namespace
			if c.Namespace == "" || c.Namespace == "default" {
				resultContext = c.Cluster
				break
			}

			resultContext = c.Cluster
		}
	}

	if resultContext != "" {
		kubeConfig.CurrentContext = resultContext
		if err = clientcmd.WriteToFile(kubeConfig, defaultConfig.ConfigAccess().GetDefaultFilename()); err != nil {
			return "", err
		}
	}

	return resultContext, nil
}
