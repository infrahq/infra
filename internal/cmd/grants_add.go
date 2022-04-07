package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type grantsCmdOptionsNew struct {
	Identity string `mapstructure:"identity"`
	IsGroup  bool   `mapstructure:"group"`
	Role     string `mapstructure:"role"`
	Provider string `mapstructure:"provider"`
	Resource string `mapstructure:"resource"`
}

func newGrantAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [IDENTITY]",
		Short: "Grant access to a resource",
		Long: `Grant one or more identities an access level of role to a resource. 

[--resource] is required:
  # Grant identity access to a cluster, namespace, or infra
  $ infra grants add johndoe@acme.com -d kubernetes.prod
  $ infra grants add johndoe@acme.com -d kubernetes.production.default
  $ infra grants add johndoe@acme.com -d infra

[IDENTITY] or [--identity] is required; [IDENTITY] will take precedence
  # Grant user access 
  $ infra grants add johndoe@acme.com ...
  $ infra grants add -i johndoe@acme.com ...

  # Grant machine access
  $ infra grants add janeDoeMachine ...

[--group] is required when granting access to a group of identities
  $ infra grants add devAdmins@acme.com -g ...
  $ infra grants add devMachines -g ...

[--provider] is required if identity is of type 'user' or 'group', and more than two identity providers are connected
  $ infra grants add johndoe@acme.com -p oktaDev ...
  $ infra grants add devGroup -g -p oktaProd ...

[--role] is optional; use if further fine grained permissions are needed
  $ infra grants add janedoe -r admin ...

For full documentation on grants, see  https://github.com/infrahq/infra/blob/main/docs/using-infra/grants.md 
`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var options grantsCmdOptionsNew
			if err := parseOptions(cmd, &options, "INFRA_ACCESS"); err != nil {
				return err
			}

			if len(args) == 1 {
				if options.Identity != "" {
					fmt.Fprintf(os.Stderr, CmdOptionOverlapMsg, "Identity", "--identity", args[0])
				}
				options.Identity = args[0]
			} else if len(args) == 0 && options.Identity == "" {
				return errors.New("IDENTITY is a required field")
			}
			return addGrant(options)
		},
	}
	cmd.Flags().StringP("resource", "r", "", "[required] Name of resource that identity be given access to")
	cmd.Flags().BoolP("group", "g", false, "Marks identity as type 'group'")
	cmd.Flags().StringP("identity", "i", "", "Name of identity")
	cmd.Flags().StringP("role", "r", models.BasePermissionConnect, "Type of access that identity will be given")
	cmd.Flags().StringP("provider", "p", "", "Name of identity provider")

	cmd.Flags().SortFlags = false
	if err := cmd.MarkFlagRequired("resource"); err != nil {
		panic("cannot mark flag --resource as required")
	}
	return cmd
}

func multipleProvidersConnected(client *api.Client) (bool, error) {
	providers, err := client.ListProviders("")
	if err != nil {
		return false, err
	}

	return len(providers) >= 2, nil
}

func addGrant(cmdOptions grantsCmdOptionsNew) error {
	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	var identityKind models.IdentityKind
	if cmdOptions.IsGroup {
		identityKind = models.GroupKind
	} else {
		identityKind, err = checkUserOrMachine(cmdOptions.Identity)
		if err != nil {
			return err
		}
	}

	var provider api.Provider
	switch identityKind {
	case models.GroupKind, models.UserKind:
		if cmdOptions.Provider == "" {
			multipleProvidersConnected, err := multipleProvidersConnected(client)
			if err != nil {
				return err
			}
			if multipleProvidersConnected {
				return fmt.Errorf("More than one provider is connected to this server. For %s identity type, please specify provider with -p or --provider.", identityKind.String())
			}
		} else {
			providers, err := client.ListProviders(cmdOptions.Provider)
			if err != nil {
				return err
			}

			if len(providers) == 0 {
				return fmt.Errorf("No provider found with name %s", cmdOptions.Provider)
			} else if len(providers) > 2 {
				panic(fmt.Sprintf(DuplicateEntryPanic, "provider", cmdOptions.Provider))
			}

			provider = providers[0]
		}
	case models.MachineKind:
		if cmdOptions.Provider != "" {
			logging.S.Debugf("machine must be a local identity; overwriting --provider with %s", models.InternalInfraProviderName)
		}

		providers, err := client.ListProviders(models.InternalInfraProviderName)
		if err != nil {
			return err
		}

		if len(providers) == 0 {
			return fmt.Errorf("No local provider found. To enable local users, create a local provider named 'infra'")
		} else if len(providers) > 2 {
			panic(fmt.Sprintf(DuplicateEntryPanic, "provider", models.InternalInfraProviderName))
		}
		provider = providers[0]
	}

	var id uid.PolymorphicID
	switch identityKind {
	case models.GroupKind:
		groups, err := client.ListGroups(api.ListGroupsRequest{Name: cmdOptions.Identity, ProviderID: provider.ID})
		if err != nil {
			return err
		}

		switch len(groups) {
		case 0:
			newGroup, err := client.CreateGroup(&api.CreateGroupRequest{
				Name:       cmdOptions.Identity,
				ProviderID: provider.ID,
			})
			if err != nil {
				return err
			}

			id = uid.NewGroupPolymorphicID(newGroup.ID)
		case 1:
			id = uid.NewGroupPolymorphicID(groups[0].ID)
		case 2:
			panic(fmt.Sprintf(DuplicateEntryPanic, "group", cmdOptions.Identity))
		}
	case models.UserKind, models.MachineKind:
		identities, err := client.ListIdentities(api.ListIdentitiesRequest{Name: cmdOptions.Identity, ProviderID: provider.ID})
		if err != nil {
			return err
		}

		switch len(identities) {
		case 0:
			response, err := client.CreateIdentity(&api.CreateIdentityRequest{
				Name:       cmdOptions.Identity,
				Kind:       identityKind.String(),
				ProviderID: provider.ID,
			})
			if err != nil {
				return err
			}
			id = uid.NewIdentityPolymorphicID(response.ID)
		case 1:
			id = uid.NewIdentityPolymorphicID(identities[0].ID)
		case 2:
			panic(fmt.Sprintf(DuplicateEntryPanic, "identity", cmdOptions.Identity))
		}
	default:
		panic("kind must be either user, machine, or group")
	}

	_, err = client.CreateGrant(&api.CreateGrantRequest{
		Subject:   id,
		Privilege: cmdOptions.Role,
		Resource:  cmdOptions.Resource,
	})
	if err != nil {
		return err
	}

	fmt.Println("Access granted!")

	return nil
}
