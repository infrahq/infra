package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/cli/browser"
	"github.com/docker/go-units"
	"github.com/goware/urlx"
	"github.com/infrahq/infra/internal/engine"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/registry"
	v1 "github.com/infrahq/infra/internal/v1"
	"github.com/infrahq/infra/internal/version"
	"github.com/mitchellh/go-homedir"
	"github.com/muesli/termenv"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	grpcMetadata "google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientauthenticationv1alpha1 "k8s.io/client-go/pkg/apis/clientauthentication/v1alpha1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type Config struct {
	Host          string `json:"host"`
	Token         string `json:"token"`
	SkipTLSVerify bool   `json:"skip-tls-verify"`
}

func readConfig() (config *Config, err error) {
	config = &Config{}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	contents, err := ioutil.ReadFile(filepath.Join(homeDir, ".infra", "config"))
	if os.IsNotExist(err) {
		return config, nil
	}

	if err != nil {
		return
	}

	if err = json.Unmarshal(contents, &config); err != nil {
		return
	}

	return
}

func writeConfig(config *Config) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if err = os.MkdirAll(filepath.Join(homeDir, ".infra"), os.ModePerm); err != nil {
		return err
	}

	contents, err := json.Marshal(config)
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(filepath.Join(homeDir, ".infra", "config"), []byte(contents), 0644); err != nil {
		return err
	}

	return nil
}

func removeConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	err = os.Remove(filepath.Join(homeDir, ".infra", "config"))
	if err != nil {
		return err
	}
	return nil
}

func printTable(header []string, data [][]string) {
	table := tablewriter.NewWriter(os.Stdout)

	if len(header) > 0 {
		table.SetHeader(header)
		table.SetAutoFormatHeaders(true)
		table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	}

	table.SetAutoWrapText(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)
	table.AppendBulk(data)
	table.Render()
}

func blue(s string) string {
	return termenv.String(s).Bold().Foreground(termenv.ColorProfile().Color("#0057FF")).String()
}

func withClientAuthUnaryInterceptor(token string) grpc.DialOption {
	return grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return invoker(grpcMetadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token), method, req, reply, cc, opts...)
	})
}

func NewClient(host string, token string, skipTlsVerify bool) (v1.V1Client, error) {
	var normalizedHost string
	if host == "" {
		normalizedHost = "localhost:443"
	} else {
		u, err := urlx.Parse(host)
		if err != nil {
			return nil, err
		}

		normalizedHost = u.Host
		if u.Port() == "" {
			normalizedHost += ":443"
		}
	}

	creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: skipTlsVerify})
	conn, err := grpc.Dial(normalizedHost, grpc.WithTransportCredentials(creds), withClientAuthUnaryInterceptor(token))
	if err != nil {
		return nil, err
	}

	return v1.NewV1Client(conn), nil
}

func clientFromConfig() (v1.V1Client, error) {
	config, err := readConfig()
	if err != nil {
		return nil, err
	}

	return NewClient(config.Host, config.Token, config.SkipTLSVerify)
}

func fetchDestinations() ([]*v1.Destination, error) {
	client, err := clientFromConfig()
	if err != nil {
		return nil, err
	}

	res, err := client.ListDestinations(context.Background(), &emptypb.Empty{})
	if err != nil {
		return nil, err
	}

	return res.Destinations, nil
}

func updateKubeconfig() (bool, error) {
	destinations, err := fetchDestinations()
	if err != nil {
		return false, err
	}

	home, err := homedir.Dir()
	if err != nil {
		return false, err
	}

	if destinations == nil {
		destinations = make([]*v1.Destination, 0)
	}

	destinationsJSON, err := json.Marshal(destinations)
	if err != nil {
		return false, err
	}

	// Write destinatinons to json file
	err = os.WriteFile(filepath.Join(home, ".infra", "destinations"), destinationsJSON, 0644)
	if err != nil {
		return false, err
	}

	// Load default config and merge new config in
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	defaultConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
	kubeConfig, err := defaultConfig.RawConfig()
	if err != nil {
		return false, err
	}

	for _, d := range destinations {
		kubeConfig.Clusters[d.Name] = &clientcmdapi.Cluster{
			Server:               "https://localhost:32710/client/" + d.Name,
			CertificateAuthority: filepath.Join(home, ".infra", "client", "cert.pem"),
		}

		executable, err := os.Executable()
		if err != nil {
			return false, err
		}

		kubeConfig.AuthInfos[d.Name] = &clientcmdapi.AuthInfo{
			Exec: &clientcmdapi.ExecConfig{
				Command:    executable,
				Args:       []string{"creds"},
				APIVersion: "client.authentication.k8s.io/v1alpha1",
			},
		}
		kubeConfig.Contexts[d.Name] = &clientcmdapi.Context{
			Cluster:  d.Name,
			AuthInfo: d.Name,
		}
	}

	for name, c := range kubeConfig.Clusters {
		if !strings.HasPrefix(c.Server, "https://localhost:32710/client/") {
			continue
		}

		var exists bool
		for _, d := range destinations {
			if name == d.Name {
				exists = true
			}
		}

		if !exists {
			delete(kubeConfig.Clusters, name)
			delete(kubeConfig.Contexts, name)
			delete(kubeConfig.AuthInfos, name)
		}
	}

	if err = clientcmd.WriteToFile(kubeConfig, defaultConfig.ConfigAccess().GetDefaultFilename()); err != nil {
		return false, err
	}

	return len(destinations) > 0, nil
}

var rootCmd = &cobra.Command{
	Use:   "infra",
	Short: "Infrastructure Identity & Access Management (IAM)",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cmd.SilenceUsage = true
	},
}

func promptShouldSkipTLSVerify(host string) (skipTlsVerify bool, proceed bool, err error) {
	httpClient := &http.Client{}
	httpClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{},
	}
	url, err := urlx.Parse(host)
	if err != nil {
		return false, false, err
	}
	url.Scheme = "https"
	urlString := url.String()

	_, err = httpClient.Get(urlString)
	if err != nil {
		if !errors.As(err, &x509.UnknownAuthorityError{}) && !errors.As(err, &x509.HostnameError{}) {
			return false, false, err
		}

		proceed := false
		fmt.Println()
		fmt.Print("Could not verify certificate for host ")
		fmt.Print(termenv.String(host).Bold())
		fmt.Println()
		prompt := &survey.Confirm{
			Message: "Are you sure you want to continue?",
		}

		err := survey.AskOne(prompt, &proceed, survey.WithIcons(func(icons *survey.IconSet) {
			icons.Question.Text = blue("?")
		}))
		if err != nil {
			fmt.Println(err.Error())
			return false, false, err
		}

		if !proceed {
			return false, false, nil
		}

		return true, true, nil
	}

	return false, true, nil
}

var loginCmd = &cobra.Command{
	Use:     "login REGISTRY",
	Short:   "Login to an Infra Registry",
	Args:    cobra.ExactArgs(1),
	Example: "$ infra login infra.example.com",
	RunE: func(cmd *cobra.Command, args []string) error {
		skipTLSVerify, proceed, err := promptShouldSkipTLSVerify(args[0])
		if err != nil {
			return err
		}

		if !proceed {
			return nil
		}

		client, err := NewClient(args[0], "", skipTLSVerify)
		if err != nil {
			return err
		}

		res, err := client.Status(context.Background(), &emptypb.Empty{})
		if err != nil {
			return err
		}

		var loginRes *v1.LoginResponse
		if !res.Admin {
			fmt.Println()
			fmt.Println(blue("Welcome to Infra. Get started by creating your admin user:"))
			email := ""
			emailPrompt := &survey.Input{
				Message: "Email",
			}
			err = survey.AskOne(emailPrompt, &email, survey.WithShowCursor(true), survey.WithValidator(survey.Required), survey.WithIcons(func(icons *survey.IconSet) {
				icons.Question.Text = blue("?")
			}))
			if err == terminal.InterruptErr {
				return nil
			}

			password := ""
			passwordPrompt := &survey.Password{
				Message: "Password",
			}
			err = survey.AskOne(passwordPrompt, &password, survey.WithShowCursor(true), survey.WithValidator(survey.Required), survey.WithIcons(func(icons *survey.IconSet) {
				icons.Question.Text = blue("?")
			}))
			if err == terminal.InterruptErr {
				return nil
			}

			fmt.Println(blue("✓") + " Creating admin user...")
			loginRes, err = client.Signup(context.Background(), &v1.SignupRequest{Email: email, Password: password})
			if err != nil {
				return err
			}
		} else {
			sourcesRes, err := client.ListSources(context.Background(), &emptypb.Empty{})
			if err != nil {
				return err
			}

			sort.Slice(sourcesRes.Sources, func(i, j int) bool {
				return sourcesRes.Sources[i].Created > sourcesRes.Sources[j].Created
			})

			options := []string{}
			for _, srs := range sourcesRes.Sources {
				switch srs.Type {
				case v1.SourceType_OKTA:
					options = append(options, fmt.Sprintf("Okta [%s]", srs.Okta.Domain))
				case v1.SourceType_INFRA:
					options = append(options, "Username & password")
				}
			}

			var option int
			if len(options) > 1 {
				prompt := &survey.Select{
					Message: "Choose a login method:",
					Options: options,
				}
				err = survey.AskOne(prompt, &option, survey.WithIcons(func(icons *survey.IconSet) {
					icons.Question.Text = blue("?")
				}))
				if err == terminal.InterruptErr {
					return nil
				}
			}

			source := sourcesRes.Sources[option]
			var loginReq v1.LoginRequest

			switch source.Type {
			case v1.SourceType_OKTA:
				// Start OIDC flow
				// Get auth code from Okta
				// Send auth code to Infra to login as a user
				state := generate.RandString(12)
				authorizeUrl := "https://" + source.Okta.Domain + "/oauth2/v1/authorize?redirect_uri=" + "http://localhost:8301&client_id=" + source.Okta.ClientId + "&response_type=code&scope=openid+email&nonce=" + generate.RandString(10) + "&state=" + state

				fmt.Println(blue("✓") + " Logging in with Okta...")
				ls, err := newLocalServer()
				if err != nil {
					return err
				}

				err = browser.OpenURL(authorizeUrl)
				if err != nil {
					return err
				}

				code, recvstate, err := ls.wait()
				if err != nil {
					return err
				}

				if state != recvstate {
					return errors.New("received state is not the same as sent state")
				}

				loginReq.Type = v1.SourceType_OKTA
				loginReq.Okta = &v1.LoginRequest_Okta{
					Domain: source.Okta.Domain,
					Code:   code,
				}

			case v1.SourceType_INFRA:
				email := ""
				emailPrompt := &survey.Input{
					Message: "Email",
				}
				err = survey.AskOne(emailPrompt, &email, survey.WithShowCursor(true), survey.WithValidator(survey.Required), survey.WithIcons(func(icons *survey.IconSet) {
					icons.Question.Text = blue("?")
				}))
				if err == terminal.InterruptErr {
					return nil
				}

				password := ""
				passwordPrompt := &survey.Password{
					Message: "Password",
				}
				err = survey.AskOne(passwordPrompt, &password, survey.WithShowCursor(true), survey.WithValidator(survey.Required), survey.WithIcons(func(icons *survey.IconSet) {
					icons.Question.Text = blue("?")
				}))
				if err == terminal.InterruptErr {
					return nil
				}

				fmt.Println(blue("✓") + " Logging in with username & password...")

				loginReq.Type = v1.SourceType_INFRA
				loginReq.Infra = &v1.LoginRequest_Infra{
					Email:    email,
					Password: password,
				}
			}

			loginRes, err = client.Login(context.Background(), &loginReq)
			if err != nil {
				return err
			}
		}

		config := &Config{
			Host:          args[0],
			Token:         loginRes.Token,
			SkipTLSVerify: skipTLSVerify,
		}

		err = writeConfig(config)
		if err != nil {
			return err
		}

		fmt.Println(blue("✓") + " Logged in")

		// Generate client certs
		home, err := homedir.Dir()
		if err != nil {
			return err
		}

		if err = os.MkdirAll(filepath.Join(home, ".infra", "client"), os.ModePerm); err != nil {
			return err
		}

		if _, err := os.Stat(filepath.Join(home, ".infra", "client", "cert.pem")); os.IsNotExist(err) {
			certBytes, keyBytes, err := generate.SelfSignedCert([]string{"localhost", "localhost:32710"})
			if err != nil {
				return err
			}

			if err = ioutil.WriteFile(filepath.Join(home, ".infra", "client", "cert.pem"), certBytes, 0644); err != nil {
				return err
			}

			if err = ioutil.WriteFile(filepath.Join(home, ".infra", "client", "key.pem"), keyBytes, 0644); err != nil {
				return err
			}

			// Kill client
			contents, err := ioutil.ReadFile(filepath.Join(home, ".infra", "client", "pid"))
			if err != nil && !os.IsNotExist(err) {
				return err
			}

			var pid int
			if !os.IsNotExist(err) {
				pid, err = strconv.Atoi(string(contents))
				if err != nil {
					return err
				}

				process, _ := os.FindProcess(int(pid))
				process.Kill()
			}

			os.Remove(filepath.Join(home, ".infra", "client", "pid"))
		}

		updated, err := updateKubeconfig()
		if err != nil {
			return err
		}

		if updated {
			fmt.Println(blue("✓") + " Kubeconfig updated")
		}

		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout of an Infra Registry",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := readConfig()
		if err != nil {
			return err
		}

		if config.Token == "" {
			return nil
		}

		client, err := NewClient(config.Host, config.Token, config.SkipTLSVerify)
		if err != nil {
			return err
		}

		client.Logout(context.Background(), &emptypb.Empty{})

		err = removeConfig()
		if err != nil {
			return err
		}

		home, err := homedir.Dir()
		if err != nil {
			return err
		}

		contents, err := ioutil.ReadFile(filepath.Join(home, ".infra", "client", "pid"))
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		var pid int
		if !os.IsNotExist(err) {
			pid, err = strconv.Atoi(string(contents))
			if err != nil {
				return err
			}

			process, _ := os.FindProcess(int(pid))
			process.Kill()
		}

		os.Remove(filepath.Join(home, ".infra", "client", "pid"))
		os.Remove(filepath.Join(home, ".infra", "destinations"))

		// Load default config and merge new config in
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		defaultConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
		kubeConfig, err := defaultConfig.RawConfig()
		if err != nil {
			return err
		}

		for name, c := range kubeConfig.Clusters {
			if strings.HasPrefix(c.Server, "https://localhost:32710/client/") {
				delete(kubeConfig.Clusters, name)
				delete(kubeConfig.Contexts, name)
				delete(kubeConfig.AuthInfos, name)
			}
		}

		if len(kubeConfig.Contexts) == 0 {
			os.Remove(defaultConfig.ConfigAccess().GetDefaultFilename())
		} else {
			_, ok := kubeConfig.Contexts[kubeConfig.CurrentContext]
			if !ok {
				var firstName string
				for name := range kubeConfig.Contexts {
					firstName = name
					break
				}
				kubeConfig.CurrentContext = firstName
			}
		}

		if err = clientcmd.WriteToFile(kubeConfig, defaultConfig.ConfigAccess().GetDefaultFilename()); err != nil {
			return err
		}
		return nil
	},
}

var destinationCmd = &cobra.Command{
	Use:   "destination",
	Short: "Manage infrastructure destinations",
}

var destinationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List destinations",
	RunE: func(cmd *cobra.Command, args []string) error {
		destinations, err := fetchDestinations()
		if err != nil {
			return err
		}

		sort.Slice(destinations, func(i, j int) bool {
			return destinations[i].Created > destinations[j].Created
		})

		kubeConfig, _ := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), nil).RawConfig()

		rows := [][]string{}
		for _, dest := range destinations {
			switch dest.Type {
			case v1.DestinationType_KUBERNETES:
				star := ""
				if dest.Name == kubeConfig.CurrentContext {
					star = "*"
				}
				rows = append(rows, []string{dest.Name + star, dest.Kubernetes.Endpoint})
			}
		}

		printTable([]string{"NAME", "ENDPOINT"}, rows)
		fmt.Println()
		fmt.Println("To connect, run \"kubectl config use-context <name>\"")
		fmt.Println()

		_, err = updateKubeconfig()
		if err != nil {
			return err
		}

		return nil
	},
}

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users",
}

var userCreateCmd = &cobra.Command{
	Use:     "create EMAIL PASSWORD",
	Short:   "create a user",
	Args:    cobra.ExactArgs(2),
	Example: "$ infra user create admin@example.com password",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := clientFromConfig()
		if err != nil {
			return err
		}

		_, err = client.CreateUser(context.Background(), &v1.CreateUserRequest{
			Email:    args[0],
			Password: args[1],
		})

		return err
	},
}

var userDeleteCmd = &cobra.Command{
	Use:   "delete USER",
	Short: "delete a user",
	Args:  cobra.ExactArgs(1),
	Example: heredoc.Doc(`
			$ infra user delete user@example.com`),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := clientFromConfig()
		if err != nil {
			return err
		}

		res, err := client.ListUsers(context.Background(), &v1.ListUsersRequest{Email: args[0]})
		if err != nil {
			return err
		}

		for _, u := range res.Users {
			_, err := client.DeleteUser(context.Background(), &v1.DeleteUserRequest{Id: u.Id})
			if err != nil {
				return err
			}
		}

		return nil
	},
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List users",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := clientFromConfig()
		if err != nil {
			return err
		}

		res, err := client.ListUsers(context.TODO(), &v1.ListUsersRequest{})
		if err != nil {
			return err
		}

		sort.Slice(res.Users, func(i, j int) bool {
			return res.Users[i].Created > res.Users[j].Created
		})

		rows := [][]string{}
		for _, user := range res.Users {
			admin := ""
			if user.Admin {
				admin = "x"
			}

			rows = append(rows, []string{user.Email, units.HumanDuration(time.Now().UTC().Sub(time.Unix(user.Created, 0))) + " ago", admin})
		}

		printTable([]string{"EMAIL", "CREATED", "ADMIN"}, rows)

		return nil
	},
}

var sourceCmd = &cobra.Command{
	Use:   "source",
	Short: "Manage identity sources",
}

var sourceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List sources",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := clientFromConfig()
		if err != nil {
			return err
		}

		res, err := client.ListSources(context.Background(), &emptypb.Empty{})
		if err != nil {
			return err
		}

		sort.Slice(res.Sources, func(i, j int) bool {
			return res.Sources[i].Created > res.Sources[j].Created
		})

		rows := [][]string{}
		for _, source := range res.Sources {
			info := ""
			typeString := ""
			switch source.Type {
			case v1.SourceType_OKTA:
				info = source.Okta.Domain
				typeString = "okta"
			case v1.SourceType_INFRA:
				info = "Built-in source"
			}
			rows = append(rows, []string{source.Id, typeString, units.HumanDuration(time.Now().UTC().Sub(time.Unix(source.Created, 0))) + " ago", info})
		}

		printTable([]string{"SOURCE ID", "KIND", "CREATED", "DESCRIPTION"}, rows)

		return nil
	},
}

func newSourceCreateCmd() *cobra.Command {
	var apiToken, domain, clientID, clientSecret string

	cmd := &cobra.Command{
		Use:   "create KIND",
		Short: "Connect an identity source",
		Args:  cobra.ExactArgs(1),
		Example: heredoc.Doc(`
			$ infra source create okta \
				--domain example.okta.com \
				--api-token 001XJv9xhv899sdfns938haos3h8oahsdaohd2o8hdao82hd \
				--client-id 0oapn0qwiQPiMIyR35d6 \
				--client-secret jfpn0qwiQPiMIfs408fjs048fjpn0qwiQPiMajsdf08j10j2`),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromConfig()
			if err != nil {
				return err
			}

			switch args[0] {
			case "okta":
				_, err := client.CreateSource(context.Background(), &v1.CreateSourceRequest{
					Type: v1.SourceType_OKTA,
					Okta: &v1.CreateSourceRequest_Okta{
						Domain:       domain,
						ApiToken:     apiToken,
						ClientId:     clientID,
						ClientSecret: clientSecret,
					},
				})
				if err != nil {
					fmt.Println(blue("✕") + " Source creation aborted")
					return err
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&apiToken, "api-token", "", "Api Token")
	cmd.Flags().StringVar(&domain, "domain", "", "Domain (e.g. example.okta.com)")
	cmd.Flags().StringVar(&clientID, "client-id", "", "Client ID for single sign-on")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "Client Secret for single sign-on")

	return cmd
}

var sourceDeleteCmd = &cobra.Command{
	Use:     "delete ID",
	Aliases: []string{"rm"},
	Short:   "Delete an identity source",
	Args:    cobra.ExactArgs(1),
	Example: heredoc.Doc(`
			$ infra source delete n7bha2pxjpa01a`),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := clientFromConfig()
		if err != nil {
			return err
		}
		_, err = client.DeleteSource(context.Background(), &v1.DeleteSourceRequest{
			Id: args[0],
		})

		return err
	},
}

var apikeyCmd = &cobra.Command{
	Use:   "apikey",
	Short: "Manage API Keys",
}

var apikeyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List API Keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := clientFromConfig()
		if err != nil {
			return err
		}

		res, err := client.ListApiKeys(context.Background(), &emptypb.Empty{})
		if err != nil {
			return err
		}

		sort.Slice(res.ApiKeys, func(i, j int) bool {
			return res.ApiKeys[i].Created > res.ApiKeys[j].Created
		})

		rows := [][]string{}
		for _, apikey := range res.ApiKeys {
			rows = append(rows, []string{apikey.Name, apikey.Key})
		}

		printTable([]string{"NAME", "APIKEY"}, rows)

		return nil
	},
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

	home, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	cmd.Flags().StringVarP(&options.ConfigPath, "config", "c", "", "config file")
	cmd.Flags().StringVar(&options.DefaultApiKey, "initial-apikey", os.Getenv("INFRA_REGISTRY_DEFAULT_API_KEY"), "initial api key for adding destinations")
	cmd.Flags().StringVar(&options.DBPath, "db", filepath.Join(home, ".infra", "infra.db"), "path to database file")
	cmd.Flags().StringVar(&options.TLSCache, "tls-cache", filepath.Join(home, ".infra", "cache"), "path to directory to cache tls self-signed and Let's Encrypt certificates")

	return cmd, nil
}

func newEngineCmd() *cobra.Command {
	var options engine.Options

	cmd := &cobra.Command{
		Use:   "engine",
		Short: "Start Infra Engine",
		RunE: func(cmd *cobra.Command, args []string) error {
			if options.Registry == "" {
				return errors.New("registry not specified (--registry or INFRA_ENGINE_REGISTRY)")
			}
			if options.Name == "" {
				return errors.New("name not specified (--name or INFRA_ENGINE_NAME)")
			}
			if options.Registry != "infra" && options.APIKey == "" {
				return errors.New("api-key not specified (--api-key or INFRA_ENGINE_API_KEY)")
			}
			return engine.Run(options)
		},
	}

	skipTLSVerify := true
	// TODO (https://github.com/infrahq/infra/issues/58): warn users instead of skipping TLS verification
	// OR find a way to include the server certificate in the api key
	// skipTLSVerify := len(os.Getenv("INFRA_ENGINE_SKIP_TLS_VERIFY")) > 0
	cmd.PersistentFlags().BoolVarP(&options.SkipTLSVerify, "skip-tls-verify", "k", skipTLSVerify, "skip TLS verification")
	cmd.Flags().StringVarP(&options.Registry, "registry", "r", os.Getenv("INFRA_ENGINE_REGISTRY"), "registry hostname")
	cmd.Flags().StringVarP(&options.Name, "name", "n", os.Getenv("INFRA_ENGINE_NAME"), "cluster name")
	cmd.Flags().StringVar(&options.APIKey, "api-key", os.Getenv("INFRA_ENGINE_API_KEY"), "api key")

	return cmd
}

var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "Display the Infra build version",
	RunE: func(cmd *cobra.Command, args []string) error {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Client:\t", version.Version)

		client, err := clientFromConfig()
		if err != nil {
			return err
		}

		// Note that we use the client to get this version, but it is in fact the server version
		res, err := client.Version(context.Background(), &emptypb.Empty{})
		if err != nil {
			fmt.Fprintln(w, blue("✕")+" Could not retrieve registry version")
			w.Flush()
			return err
		}

		fmt.Fprintln(w, "Registry:\t", res.Version)
		fmt.Fprintln(w)
		w.Flush()

		return nil
	},
}

var credsCmd = &cobra.Command{
	Use:    "creds",
	Short:  "Generate credentials",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO (https://github.com/infrahq/infra/issues/59): First try to read cached token
		client, err := clientFromConfig()
		if err != nil {
			return err
		}

		res, err := client.CreateCred(context.Background(), &emptypb.Empty{})
		if err != nil {
			return err
		}

		expiry := time.Unix(res.Expires, 0)

		execCredential := &clientauthenticationv1alpha1.ExecCredential{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ExecCredential",
				APIVersion: clientauthenticationv1alpha1.SchemeGroupVersion.String(),
			},
			Spec: clientauthenticationv1alpha1.ExecCredentialSpec{},
			Status: &clientauthenticationv1alpha1.ExecCredentialStatus{
				Token:               res.Token,
				ExpirationTimestamp: &metav1.Time{Time: expiry},
			},
		}

		bts, err := json.Marshal(execCredential)
		if err != nil {
			return err
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		if err = os.MkdirAll(filepath.Join(home, ".infra", "cache"), os.ModePerm); err != nil {
			return err
		}

		contents, err := ioutil.ReadFile(filepath.Join(home, ".infra", "client", "pid"))
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		var pid int
		// see if proxy process is already running
		if !os.IsNotExist(err) {
			pid, err = strconv.Atoi(string(contents))
			if err != nil {
				return err
			}

			// verify process is still running
			process, err := os.FindProcess(int(pid))
			if process == nil || err != nil {
				pid = 0
			}

			err = process.Signal(syscall.Signal(0))
			if err != nil {
				pid = 0
			}
		}

		if pid == 0 {
			os.Remove(filepath.Join(home, ".infra", "client", "pid"))

			cmd := exec.Command(os.Args[0], "client")
			err = cmd.Start()
			if err != nil {
				return err
			}

			for {
				if _, err := os.Stat(filepath.Join(home, ".infra", "client", "pid")); os.IsNotExist(err) {
					time.Sleep(25 * time.Millisecond)
				} else {
					break
				}
			}
		}

		fmt.Println(string(bts))

		return nil
	},
}

var clientCmd = &cobra.Command{
	Use:    "client",
	Short:  "Run local client to relay requests",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Starting client")
		return RunLocalClient()
	},
}

func NewRootCmd() (*cobra.Command, error) {
	cobra.EnableCommandSorting = false

	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)

	userCmd.AddCommand(userCreateCmd)
	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userDeleteCmd)
	rootCmd.AddCommand(userCmd)

	destinationCmd.AddCommand(destinationListCmd)
	rootCmd.AddCommand(destinationCmd)

	sourceCmd.AddCommand(sourceListCmd)
	sourceCmd.AddCommand(newSourceCreateCmd())
	sourceCmd.AddCommand(sourceDeleteCmd)
	rootCmd.AddCommand(sourceCmd)

	apikeyCmd.AddCommand(apikeyListCmd)
	rootCmd.AddCommand(apikeyCmd)

	registryCmd, err := newRegistryCmd()
	if err != nil {
		return nil, err
	}
	rootCmd.AddCommand(registryCmd)
	rootCmd.AddCommand(newEngineCmd())

	rootCmd.AddCommand(versionCmd)

	// Hidden commands
	rootCmd.AddCommand(credsCmd)
	rootCmd.AddCommand(clientCmd)

	return rootCmd, nil
}

func Run() error {
	cmd, err := NewRootCmd()
	if err != nil {
		return err
	}

	return cmd.Execute()
}
