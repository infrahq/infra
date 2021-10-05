package cmd

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/goware/urlx"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/engine"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/registry"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func blue(s string) string {
	return termenv.String(s).Bold().Foreground(termenv.ColorProfile().Color("#0057FF")).String()
}

func NewApiContext(token string) context.Context {
	return context.WithValue(context.Background(), api.ContextAccessToken, token)
}

func NewApiClient(host string, skipTLSVerify bool) (*api.APIClient, error) {
	u, err := urlx.Parse(host)
	if err != nil {
		return nil, err
	}

	config := api.NewConfiguration()
	config.Host = u.Host
	config.Scheme = "https"

	if skipTLSVerify {
		config.HTTPClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					//nolint:gosec // We may purposely set insecureskipverify via a flag
					InsecureSkipVerify: true,
				},
			},
		}
	}

	return api.NewAPIClient(config), nil
}

func apiContextFromConfig() (context.Context, error) {
	config, err := readCurrentConfig()
	if err != nil {
		return nil, err
	}

	if config == nil {
		return nil, &ErrUnauthenticated{}
	}

	return NewApiContext(config.Token), nil
}

func apiClientFromConfig() (*api.APIClient, error) {
	config, err := readCurrentConfig()
	if err != nil {
		return nil, err
	}

	if config == nil {
		return nil, &ErrUnauthenticated{}
	}

	return NewApiClient(config.Host, config.SkipTLSVerify)
}

func clientConfig() clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.WarnIfAllMissing = false

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
}

func updateKubeconfig(destinations []api.Destination) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if len(destinations) > 0 {
		destinationsJSON, err := json.Marshal(destinations)
		if err != nil {
			return err
		}

		// Write destinations to a known json file location for `infra client` to read
		err = os.WriteFile(filepath.Join(homeDir, ".infra", "destinations"), destinationsJSON, 0o600)
		if err != nil {
			return err
		}
	}

	defaultConfig := clientConfig()

	kubeConfig, err := defaultConfig.RawConfig()
	if err != nil {
		return err
	}

	for _, d := range destinations {
		contextName := "infra:" + d.Name

		kubeConfig.Clusters[contextName] = &clientcmdapi.Cluster{
			Server:                   fmt.Sprintf("https://%s/proxy", d.Kubernetes.Endpoint),
			CertificateAuthorityData: []byte(d.Kubernetes.Ca),
		}

		executable, err := os.Executable()
		if err != nil {
			return err
		}

		kubeConfig.AuthInfos[contextName] = &clientcmdapi.AuthInfo{
			Exec: &clientcmdapi.ExecConfig{
				Command:    executable,
				Args:       []string{"token", d.Name},
				APIVersion: "client.authentication.k8s.io/v1beta1",
			},
		}

		kubeConfig.Contexts[contextName] = &clientcmdapi.Context{
			Cluster:  contextName,
			AuthInfo: contextName,
		}
	}

	for name := range kubeConfig.Contexts {
		if !strings.HasPrefix(name, "infra:") {
			continue
		}

		destinationName := strings.ReplaceAll(name, "infra:", "")

		var exists bool

		for _, d := range destinations {
			if destinationName == d.Name {
				exists = true
			}
		}

		if !exists {
			delete(kubeConfig.Clusters, name)
			delete(kubeConfig.Contexts, name)
			delete(kubeConfig.AuthInfos, name)
		}
	}

	if len(destinations) == 0 {
		_, ok := kubeConfig.Contexts[kubeConfig.CurrentContext]
		if !ok {
			kubeConfig.CurrentContext = ""
			for name := range kubeConfig.Contexts {
				kubeConfig.CurrentContext = name
				break
			}
		}
	}

	if err = clientcmd.WriteToFile(kubeConfig, defaultConfig.ConfigAccess().GetDefaultFilename()); err != nil {
		return err
	}

	return nil
}

var rootCmd = &cobra.Command{
	Use:   "infra",
	Short: "Infrastructure Identity & Access Management (IAM)",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cmd.SilenceUsage = true
	},
}

func newLoginCmd() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:     "login REGISTRY",
		Short:   "Login to an Infra Registry",
		Args:    cobra.MaximumNArgs(1),
		Example: "$ infra login infra.example.com",
		RunE: func(cmd *cobra.Command, args []string) error {
			registry := ""
			if len(args) == 1 {
				registry = args[0]
			}

			return login(registry, true)
		},
	}

	return cmd, nil
}

func newLogoutCmd() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Logout of an Infra Registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			return logout()
		},
	}

	return cmd, nil
}

func newListCmd() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List destinations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return list()
		},
	}

	return cmd, nil
}

func newRegistryCmd() (*cobra.Command, error) {
	var options registry.Options

	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Start Infra Registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			return registry.Run(options)
		},
	}

	defaultInfraHome := filepath.Join("~", ".infra")

	cmd.Flags().StringVarP(&options.ConfigPath, "config", "c", "", "config file")
	cmd.Flags().StringVar(&options.DefaultApiKey, "initial-apikey", os.Getenv("INFRA_REGISTRY_DEFAULT_API_KEY"), "initial api key for adding destinations")
	cmd.Flags().StringVar(&options.DBPath, "db", filepath.Join(defaultInfraHome, "infra.db"), "path to database file")
	cmd.Flags().StringVar(&options.TLSCache, "tls-cache", filepath.Join(defaultInfraHome, "cache"), "path to directory to cache tls self-signed and Let's Encrypt certificates")
	cmd.Flags().BoolVar(&options.UI, "ui", false, "enable ui")
	cmd.Flags().StringVar(&options.UIProxy, "ui-proxy", "", "proxy ui requests to this host")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	if filepath.Dir(options.DBPath) == defaultInfraHome {
		options.DBPath = filepath.Join(homeDir, ".infra", "infra.db")
	}

	if filepath.Dir(options.TLSCache) == defaultInfraHome {
		options.TLSCache = filepath.Join(homeDir, ".infra", "cache")
	}

	defaultSync := 30

	osSync := os.Getenv("INFRA_SYNC_INTERVAL_SECONDS")
	if osSync != "" {
		defaultSync, err = strconv.Atoi(osSync)
		if err != nil {
			logging.L.Error("could not convert INFRA_SYNC_INTERVAL_SECONDS to an integer: " + err.Error())
		}
	}

	cmd.Flags().IntVar(&options.SyncInterval, "sync-interval", defaultSync, "the interval (in seconds) at which Infra will poll sources for users and groups")

	return cmd, nil
}

func newEngineCmd() (*cobra.Command, error) {
	var options engine.Options

	cmd := &cobra.Command{
		Use:   "engine",
		Short: "Start Infra Engine",
		RunE: func(cmd *cobra.Command, args []string) error {
			if options.Registry == "" {
				return errors.New("registry not specified (--registry or INFRA_ENGINE_REGISTRY)")
			}
			if options.Registry != "infra" && options.APIKey == "" {
				return errors.New("api-key not specified (--api-key or INFRA_ENGINE_API_KEY)")
			}
			return engine.Run(options)
		},
	}

	defaultInfraHome := filepath.Join("~", ".infra")

	cmd.PersistentFlags().BoolVar(&options.ForceTLSVerify, "force-tls-verify", false, "force TLS verification")
	cmd.Flags().StringVarP(&options.Registry, "registry", "r", os.Getenv("INFRA_ENGINE_REGISTRY"), "registry hostname")
	cmd.Flags().StringVarP(&options.Name, "name", "n", os.Getenv("INFRA_ENGINE_NAME"), "cluster name")
	cmd.Flags().StringVar(&options.TLSCache, "tls-cache", filepath.Join(defaultInfraHome, "cache"), "path to directory to cache tls self-signed and Let's Encrypt certificates")
	cmd.Flags().StringVar(&options.APIKey, "api-key", os.Getenv("INFRA_ENGINE_API_KEY"), "api key")

	if filepath.Dir(options.TLSCache) == defaultInfraHome {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}

		options.TLSCache = filepath.Join(homeDir, ".infra", "cache")
	}

	return cmd, nil
}

func newVersionCmd() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Display the Infra build version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return version()
		},
	}

	return cmd, nil
}

func newTokenCmd() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   "token DESTINATION",
		Short: "Generate a JWT token for connecting to a destination, e.g. Kubernetes",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("expecting destination as an argument")
			}

			return token(args[0])
		},
	}

	return cmd, nil
}

func NewRootCmd() (*cobra.Command, error) {
	cobra.EnableCommandSorting = false

	loginCmd, err := newLoginCmd()
	if err != nil {
		return nil, err
	}

	logoutCmd, err := newLogoutCmd()
	if err != nil {
		return nil, err
	}

	listCmd, err := newListCmd()
	if err != nil {
		return nil, err
	}

	tokenCmd, err := newTokenCmd()
	if err != nil {
		return nil, err
	}

	versionCmd, err := newVersionCmd()
	if err != nil {
		return nil, err
	}

	registryCmd, err := newRegistryCmd()
	if err != nil {
		return nil, err
	}

	engineCmd, err := newEngineCmd()
	if err != nil {
		return nil, err
	}

	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(tokenCmd)
	rootCmd.AddCommand(versionCmd)

	rootCmd.AddCommand(registryCmd)
	rootCmd.AddCommand(engineCmd)

	return rootCmd, nil
}

func Run() error {
	cmd, err := NewRootCmd()
	if err != nil {
		return err
	}

	return cmd.Execute()
}
