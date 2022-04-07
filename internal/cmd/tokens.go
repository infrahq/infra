package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientauthenticationv1beta1 "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"

	"github.com/infrahq/infra/api"
)

func newTokensCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "tokens",
		Short:  "Create & manage tokens",
		Hidden: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return mustBeLoggedIn()
		},
	}

	cmd.AddCommand(newTokensAddCmd())

	return cmd
}

func newTokensAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "Create a token",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := tokensCreate(); err != nil {
				return err
			}

			return nil
		},
	}
}

func tokensCreate() error {
	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	token, err := client.CreateToken()
	if err != nil {
		if errors.Is(err, api.ErrForbidden) {
			fmt.Fprintln(os.Stderr, "Session has expired.")

			if err = relogin(); err != nil {
				return err
			}

			return tokensCreate()
		}

		return err
	}

	execCredential := &clientauthenticationv1beta1.ExecCredential{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ExecCredential",
			APIVersion: clientauthenticationv1beta1.SchemeGroupVersion.String(),
		},
		Spec: clientauthenticationv1beta1.ExecCredentialSpec{},
		Status: &clientauthenticationv1beta1.ExecCredentialStatus{
			Token:               token.Token,
			ExpirationTimestamp: &metav1.Time{Time: time.Time(token.Expires)},
		},
	}

	bts, err := json.Marshal(execCredential)
	if err != nil {
		return err
	}

	fmt.Println(string(bts))

	return nil
}
