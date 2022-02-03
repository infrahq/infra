package cmd

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/goware/urlx"
	"github.com/lensesio/tableprinter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/config"
	"github.com/infrahq/infra/internal/engine"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/registry"
	"github.com/infrahq/infra/uid"
)

func infraHomeDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	infraDir := filepath.Join(homeDir, ".infra")

	err = os.MkdirAll(infraDir, os.ModePerm)
	if err != nil {
		return "", err
	}

	return infraDir, nil
}

func printTable(data interface{}) {
	table := tableprinter.New(os.Stdout)

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

func defaultAPIClient() (*api.Client, error) {
	config, err := readHostConfig("")
	if err != nil {
		return nil, err
	}

	return apiClient(config.Host, config.Token, config.SkipTLSVerify)
}

func apiClient(host string, token string, skipTLSVerify bool) (*api.Client, error) {
	u, err := urlx.Parse(host)
	if err != nil {
		return nil, err
	}

	u.Scheme = "https"

	return &api.Client{
		Url:   u.String(),
		Token: token,
		Http: http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					//nolint:gosec // We may purposely set insecureskipverify via a flag
					InsecureSkipVerify: skipTLSVerify,
				},
			},
		},
	}, nil
}

var loginCmd = &cobra.Command{
	Use:     "login [HOST]",
	Short:   "Login to Infra",
	Example: "$ infra login",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var host string
		if len(args) == 1 {
			host = args[0]
		}

		if err := login(host); err != nil {
			return err
		}

		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:     "logout",
	Short:   "Logout of Infra",
	Example: "$ infra logout",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := logout(); err != nil {
			return err
		}

		return nil
	},
}

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List destinations and your access",
	RunE: func(cmd *cobra.Command, args []string) error {
		return list()
	},
}

func newUseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use [DESTINATION]",
		Short: "Connect to a destination",
		Example: `
# Connect to a Kubernetes cluster
infra use kubernetes.development

# Connect to a Kubernetes namespace
infra use kubernetes.development.kube-system
		`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}

			err := updateKubeconfig()
			if err != nil {
				return err
			}

			parts := strings.Split(name, ".")

			if len(parts) < 2 {
				return errors.New("invalid argument")
			}

			if len(parts) <= 2 || parts[2] == "default" {
				return kubernetesSetContext("infra:" + parts[1])
			}

			return kubernetesSetContext("infra:" + parts[1] + ":" + parts[2])
		},
	}

	return cmd
}

var accessListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List access",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := defaultAPIClient()
		if err != nil {
			return err
		}

		grants, err := client.ListGrants(api.ListGrantsRequest{})
		if err != nil {
			return err
		}

		type row struct {
			Provider string `header:"PROVIDER"`
			Identity string `header:"IDENTITY"`
			Access   string `header:"ACCESS"`
			Name     string `header:"DESTINATION"`
		}

		var rows []row
		for _, c := range grants {
			provider, identity, err := info(client, c)
			if err != nil {
				return err
			}

			rows = append(rows, row{
				Provider: provider,
				Identity: identity,
				Name:     c.Resource,
				Access:   c.Privilege,
			})
		}

		printTable(rows)

		return nil
	},
}

func newAccessGrantCmd() *cobra.Command {
	var (
		user     string
		group    string
		provider string
		role     string
	)

	cmd := &cobra.Command{
		Use:   "grant DESTINATION",
		Short: "Grant access",
		Example: `
# Grant user admin access to a cluster
infra grant -u suzie@acme.com -r admin kubernetes.production 

# Grant group admin access to a namespace
infra grant -g Engineering -r admin kubernetes.production.default

# Grant user admin access to infra itself
infra grant -u admin@acme.com -r admin infra
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			providers, err := client.ListProviders(provider)
			if err != nil {
				return err
			}

			if len(providers) == 0 {
				return errors.New("No identity providers connected")
			}

			if len(providers) > 1 {
				return errors.New("Specify provider with -p or --provider")
			}

			if group != "" {
				if user != "" {
					return errors.New("only one of -g and -u are allowed")
				}

				// create user if they don't exist
				groups, err := client.ListGroups(api.ListGroupsRequest{Name: group})
				if err != nil {
					return err
				}

				var id uid.ID

				if len(groups) == 0 {
					newGroup, err := client.CreateGroup(&api.CreateGroupRequest{
						Name:       group,
						ProviderID: providers[0].ID,
					})
					if err != nil {
						return err
					}

					id = newGroup.ID
				} else {
					id = groups[0].ID
				}

				_, err = client.CreateGrant(&api.CreateGrantRequest{
					Identity:  fmt.Sprintf("g:%s", id),
					Resource:  args[0],
					Privilege: role,
				})
				if err != nil {
					return err
				}
			}

			if user != "" {
				if group != "" {
					return errors.New("only one of -g and -u are allowed")
				}

				// create user if they don't exist
				users, err := client.ListUsers(api.ListUsersRequest{Email: user})
				if err != nil {
					return err
				}

				var id uid.ID

				if len(users) == 0 {
					newUser, err := client.CreateUser(&api.CreateUserRequest{
						Email:      user,
						ProviderID: providers[0].ID,
					})
					if err != nil {
						return err
					}

					id = newUser.ID
				} else {
					id = users[0].ID
				}

				_, err = client.CreateGrant(&api.CreateGrantRequest{
					Identity:  fmt.Sprintf("u:%s", id),
					Resource:  args[0],
					Privilege: role,
				})

				if err != nil {
					return err
				}
			}

			fmt.Println("Access granted")

			return nil
		},
	}

	cmd.Flags().StringVarP(&user, "user", "u", "", "User to grant access to")
	cmd.Flags().StringVarP(&group, "group", "g", "", "Group to grant access to")
	cmd.Flags().StringVarP(&provider, "provider", "p", "", "Provider from which to grant user access to")
	cmd.Flags().StringVarP(&role, "role", "r", "", "Role to grant")

	return cmd
}

func newAccessRevokeCmd() *cobra.Command {
	var (
		user     string
		group    string
		provider string
		role     string
	)

	cmd := &cobra.Command{
		Use:   "revoke DESTINATION",
		Short: "Revoke access",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			providers, err := client.ListProviders(provider)
			if err != nil {
				return err
			}

			if len(providers) == 0 {
				return errors.New("No identity providers connected")
			}

			if len(providers) > 1 {
				return errors.New("Specify provider with -p or --provider")
			}

			var identity string

			if group != "" {
				if user != "" {
					return errors.New("only one of -g and -u are allowed")
				}

				groups, err := client.ListGroups(api.ListGroupsRequest{Name: group})
				if err != nil {
					return err
				}

				if len(groups) == 0 {
					return errors.New("no such group")
				}

				identity = "g:" + groups[0].ID.String()
			}

			if user != "" {
				if group != "" {
					return errors.New("only one of -g and -u are allowed")
				}

				users, err := client.ListUsers(api.ListUsersRequest{Email: user})
				if err != nil {
					return err
				}

				if len(users) == 0 {
					return errors.New("no such user")
				}

				identity = "u:" + users[0].ID.String()
			}

			grants, err := client.ListGrants(api.ListGrantsRequest{Resource: args[0], Identity: identity, Privilege: role})
			if err != nil {
				return err
			}

			for _, g := range grants {
				err := client.DeleteGrant(g.ID)
				if err != nil {
					return err
				}
			}

			fmt.Println("Access revoked")

			return nil
		},
	}

	cmd.Flags().StringVarP(&user, "user", "u", "", "User to revoke access from")
	cmd.Flags().StringVarP(&group, "group", "g", "", "Group to revoke access from")
	cmd.Flags().StringVarP(&provider, "provider", "p", "", "Provider from which to revoke access from")
	cmd.Flags().StringVarP(&role, "role", "r", "", "Role to revoke")

	return cmd
}

func newAccessCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "access",
		Short: "Manage access",
	}

	cmd.AddCommand(accessListCmd)
	cmd.AddCommand(newAccessGrantCmd())
	cmd.AddCommand(newAccessRevokeCmd())

	return cmd
}

var destinationsListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List connected destinations",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := defaultAPIClient()
		if err != nil {
			return err
		}

		destinations, err := client.ListDestinations(api.ListDestinationsRequest{})
		if err != nil {
			return err
		}

		type row struct {
			Name string `header:"NAME"`
			URL  string `header:"URL"`
		}

		var rows []row
		for _, d := range destinations {
			rows = append(rows, row{
				Name: d.Name,
				URL:  d.Connection.URL,
			})
		}

		printTable(rows)

		return nil
	},
}

var destinationsAddCmd = &cobra.Command{
	Use:   "add TYPE NAME",
	Short: "Connect a destination",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := defaultAPIClient()
		if err != nil {
			return err
		}

		token, err := client.CreateAccessKey(&api.CreateAccessKeyRequest{
			Name: args[0],
			Ttl:  time.Hour * 8760,

			// TODO: extract permissions out of access into api package
			Permissions: []string{
				string(access.PermissionUserRead),
				string(access.PermissionGroupRead),
				string(access.PermissionGrantRead),
				string(access.PermissionDestinationRead),
				string(access.PermissionDestinationCreate),
				string(access.PermissionDestinationUpdate),
			},
		})
		if err != nil {
			return err
		}

		if args[0] != "kubernetes" {
			return fmt.Errorf("Supported types: `kubernetes`")
		}

		config, err := currentHostConfig()
		if err != nil {
			return err
		}

		command := "    helm install infra-engine infrahq/engine"
		if len(args) > 1 {
			command += fmt.Sprintf(" --set config.name=%s", args[1])
		}
		command += fmt.Sprintf(" --set config.accessKey=%s ", token.AccessKey)
		command += fmt.Sprintf(" --set config.server=%s ", config.Host)

		// TODO: replace me with a certificate fingerprint
		// so even when users have self-signed certificates
		// infra can establish a secure TLS connection
		if config.SkipTLSVerify {
			command += "  --set config.skipTLSVerify=true"
		}

		fmt.Println()
		fmt.Println("Run the following command to connect a kubernetes cluster:")
		fmt.Println()
		fmt.Println(command)
		fmt.Println()
		fmt.Println()
		return nil
	},
}

var destinationsRemoveCmd = &cobra.Command{
	Use:     "remove DESTINATION",
	Aliases: []string{"rm"},
	Short:   "Disconnect a destination",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := defaultAPIClient()
		if err != nil {
			return err
		}

		destinations, err := client.ListDestinations(api.ListDestinationsRequest{Name: args[0]})
		if err != nil {
			return err
		}

		if len(destinations) == 0 {
			return fmt.Errorf("no destinations named %s", args[0])
		}

		for _, d := range destinations {
			err := client.DeleteDestination(d.ID)
			if err != nil {
				return err
			}
		}

		return nil
	},
}

func newDestinationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "destinations",
		Short: "Connect & manage destinations",
	}

	cmd.AddCommand(destinationsListCmd)
	cmd.AddCommand(destinationsAddCmd)
	cmd.AddCommand(destinationsRemoveCmd)

	return cmd
}

func newServerCmd() (*cobra.Command, error) {
	var (
		options    registry.Options
		configFile string
	)

	var err error

	parseConfig := func() {
		if configFile == "" {
			return
		}

		var contents []byte

		contents, err = ioutil.ReadFile(configFile)
		if err != nil {
			return
		}

		err = yaml.Unmarshal(contents, &options)
	}

	cobra.OnInitialize(parseConfig)

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start Infra Server",

		RunE: func(cmd *cobra.Command, args []string) error {
			if err != nil {
				return err
			}

			return registry.Run(options)
		},
	}

	infraDir, err := infraHomeDir()
	if err != nil {
		return nil, err
	}

	cmd.Flags().StringVarP(&configFile, "config-file", "f", "", "Server configuration file")
	cmd.Flags().StringVar(&options.AdminAccessKey, "admin-access-key", "file:"+filepath.Join(infraDir, "admin-access-key"), "Admin access key (secret)")
	cmd.Flags().StringVar(&options.AccessKey, "access-key", "file:"+filepath.Join(infraDir, "access-key"), "Access key (secret)")
	cmd.Flags().StringVar(&options.TLSCache, "tls-cache", filepath.Join(infraDir, "tls"), "Directory to cache TLS certificates")
	cmd.Flags().StringVar(&options.DBFile, "db-file", filepath.Join(infraDir, "db"), "Path to database file")
	cmd.Flags().StringVar(&options.DBEncryptionKey, "db-encryption-key", filepath.Join(infraDir, "key"), "Database encryption key")
	cmd.Flags().StringVar(&options.DBEncryptionKeyProvider, "db-encryption-key-provider", "native", "Database encryption key provider")
	cmd.Flags().StringVar(&options.DBHost, "db-host", "", "Database host")
	cmd.Flags().IntVar(&options.DBPort, "db-port", 5432, "Database port")
	cmd.Flags().StringVar(&options.DBName, "db-name", "", "Database name")
	cmd.Flags().StringVar(&options.DBUser, "db-user", "", "Database user")
	cmd.Flags().StringVar(&options.DBPassword, "db-password", "", "Database password (secret)")
	cmd.Flags().StringVar(&options.DBParameters, "db-parameters", "", "Database additional connection parameters")
	cmd.Flags().BoolVar(&options.EnableTelemetry, "enable-telemetry", true, "Enable telemetry")
	cmd.Flags().BoolVar(&options.EnableCrashReporting, "enable-crash-reporting", true, "Enable crash reporting")
	cmd.Flags().DurationVarP(&options.SessionDuration, "session-duration", "d", time.Hour*12, "Session duration")

	return cmd, nil
}

func newEngineCmd() *cobra.Command {
	var (
		options          engine.Options
		engineConfigFile string
		err              error
	)

	parseConfig := func() {
		if engineConfigFile == "" {
			return
		}

		var contents []byte

		contents, err = ioutil.ReadFile(engineConfigFile)
		if err != nil {
			return
		}

		err = yaml.Unmarshal(contents, &options)
	}

	cobra.OnInitialize(parseConfig)

	cmd := &cobra.Command{
		Use:   "engine",
		Short: "Start Infra Engine",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err != nil {
				return err
			}

			if err := engine.Run(options); err != nil {
				return err
			}

			return nil
		},
	}

	infraDir, err := infraHomeDir()
	if err != nil {
		return nil
	}

	cmd.Flags().StringVarP(&engineConfigFile, "config-file", "f", "", "Engine config file")
	cmd.Flags().StringVarP(&options.Name, "name", "n", "", "Destination name")
	cmd.Flags().StringVar(&options.AccessKey, "access-key", "", "Infra access key (use file:// to load from a file)")
	cmd.Flags().StringVar(&options.TLSCache, "tls-cache", filepath.Join(infraDir, "tls"), "Directory to cache TLS certificates")
	cmd.Flags().StringVar(&options.Server, "server", "", "Infra Server hostname")
	cmd.Flags().BoolVar(&options.SkipTLSVerify, "skip-tls-verify", true, "Skip TLS verification")

	return cmd
}

var tokensCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a token",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := tokensCreate(); err != nil {
			return err
		}

		return nil
	},
}

func newTokensCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tokens",
		Short: "Create & manage tokens",
	}

	cmd.AddCommand(tokensCreateCmd)

	return cmd
}

var providersListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List identity providers",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := defaultAPIClient()
		if err != nil {
			return err
		}

		providers, err := client.ListProviders("")
		if err != nil {
			return err
		}

		type row struct {
			Name string `header:"NAME"`
			URL  string `header:"URL"`
		}

		var rows []row
		for _, p := range providers {
			rows = append(rows, row{Name: p.Name, URL: p.URL})
		}

		printTable(rows)

		return nil
	},
}

func newProvidersAddCmd() *cobra.Command {
	var (
		url          string
		clientID     string
		clientSecret string
	)

	cmd := &cobra.Command{
		Use:   "add NAME",
		Short: "Connect an identity provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			_, err = client.CreateProvider(&api.CreateProviderRequest{
				Name:         args[0],
				URL:          url,
				ClientID:     clientID,
				ClientSecret: clientSecret,
			})
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "url or domain (e.g. acme.okta.com)")
	cmd.Flags().StringVar(&clientID, "client-id", "", "OpenID Client ID")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "OpenID Client Secret")

	return cmd
}

var providersRemoveCmd = &cobra.Command{
	Use:     "remove PROVIDER",
	Aliases: []string{"rm"},
	Short:   "Disconnect an identity provider",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := defaultAPIClient()
		if err != nil {
			return err
		}

		providers, err := client.ListProviders(args[0])
		if err != nil {
			return err
		}

		for _, p := range providers {
			err := client.DeleteProvider(p.ID)
			if err != nil {
				return err
			}
		}

		return nil
	},
}

func newProvidersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "Connect & manage identity providers",
	}

	cmd.AddCommand(providersListCmd)
	cmd.AddCommand(newProvidersAddCmd())
	cmd.AddCommand(providersRemoveCmd)

	return cmd
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display the info about the current session",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := currentHostConfig()
		if err != nil {
			return err
		}

		client, err := defaultAPIClient()
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
		defer w.Flush()

		fmt.Fprintln(w)
		fmt.Fprintln(w, "Server:\t", config.Host)

		if config.ID == 0 {
			fmt.Fprintln(w, "User:\t", "system")
			fmt.Fprintln(w)
			return nil
		}

		provider, err := client.GetProvider(config.ProviderID)
		if err != nil {
			return err
		}

		fmt.Fprintln(w, "Identity Provider:\t", provider.Name, fmt.Sprintf("(%s)", provider.URL))

		user, err := client.GetUser(config.ID)
		if err != nil {
			return err
		}

		fmt.Fprintln(w, "User:\t", user.Email)

		groups, err := client.ListUserGroups(config.ID)
		if err != nil {
			return err
		}

		var names string
		for i, g := range groups {
			if i != 0 {
				names += ", "
			}

			names += g.Name
		}

		fmt.Fprintln(w, "Groups:\t", names)

		admin := false
		for _, p := range user.Permissions {
			if p == "infra.*" {
				admin = true
			}
		}

		fmt.Fprintln(w, "Admin:\t", admin)
		fmt.Fprintln(w)

		return nil
	},
}

func newImportCmd() *cobra.Command {
	var replace bool

	cmd := &cobra.Command{
		Use:   "import [FILE]",
		Short: "Import an infra server configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			contents, err := ioutil.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("reading configuration file: %w", err)
			}

			var c config.Config
			err = yaml.Unmarshal(contents, &c)
			if err != nil {
				return err
			}

			return config.Import(client, c, replace)
		},
	}

	cmd.Flags().BoolVar(&replace, "replace", false, "replace any existing configuration")

	return cmd
}

func newMachinesCreateCmd() *cobra.Command {
	var options MachinesCreateOptions

	cmd := &cobra.Command{
		Use:   "create [NAME] [DESCRIPTION] [PERMISSIONS]",
		Short: "Create a machine identity, e.x. a service that needs to access infrastructure",
		Args:  cobra.MaximumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				options.Name = args[0]
			}

			if len(args) > 1 {
				options.Description = args[1]
			}

			if len(args) == 3 {
				options.Permissions = args[2]
			}

			return createMachine(&options)
		},
	}

	return cmd
}

var machinesListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List machines",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := defaultAPIClient()
		if err != nil {
			return err
		}

		machines, err := client.ListMachines(api.ListMachinesRequest{})
		if err != nil {
			return err
		}

		type row struct {
			Name        string   `header:"Name"`
			Permissions []string `header:"Permissions"`
			Description string   `header:"Description"`
		}

		var rows []row
		for _, m := range machines {
			rows = append(rows, row{
				Name:        m.Name,
				Permissions: m.Permissions,
				Description: m.Description,
			})
		}

		printTable(rows)

		return nil
	},
}

var machinesDeleteCmd = &cobra.Command{
	Use:   "remove MACHINE",
	Short: "Remove a machine identity",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := defaultAPIClient()
		if err != nil {
			return err
		}

		machines, err := client.ListMachines(api.ListMachinesRequest{Name: args[0]})
		if err != nil {
			return err
		}

		for _, m := range machines {
			err := client.DeleteMachine(m.ID)
			if err != nil {
				return err
			}
		}

		return nil
	},
}

func newMachinesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "machines",
		Short: "Create & manage machine identities",
	}

	cmd.AddCommand(newMachinesCreateCmd())
	cmd.AddCommand(machinesListCmd)
	cmd.AddCommand(machinesDeleteCmd)

	return cmd
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the Infra version",
	RunE: func(cmd *cobra.Command, args []string) error {
		return version()
	},
}

func NewRootCmd() (*cobra.Command, error) {
	cobra.EnableCommandSorting = false

	var level string

	rootCmd := &cobra.Command{
		Use:               "infra",
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		SilenceUsage:      true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return logging.SetLevel(level)
		},
	}

	serverCmd, err := newServerCmd()
	if err != nil {
		return nil, err
	}

	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(newUseCmd())
	rootCmd.AddCommand(newAccessCmd())
	rootCmd.AddCommand(newDestinationsCmd())
	rootCmd.AddCommand(newProvidersCmd())
	rootCmd.AddCommand(newMachinesCmd())
	rootCmd.AddCommand(newTokensCmd())
	rootCmd.AddCommand(newImportCmd())
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(newEngineCmd())
	rootCmd.AddCommand(versionCmd)

	rootCmd.PersistentFlags().StringVar(&level, "log-level", "info", "Log level (error, warn, info, debug)")

	return rootCmd, nil
}

func Run() error {
	cmd, err := NewRootCmd()
	if err != nil {
		return err
	}

	return cmd.Execute()
}
