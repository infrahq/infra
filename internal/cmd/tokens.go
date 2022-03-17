package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientauthenticationv1beta1 "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"

	"github.com/infrahq/infra/api"
)

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
	execCredential := &clientauthenticationv1beta1.ExecCredential{}

	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	config, err := currentHostConfig()
	if err != nil {
		return err
	}

	id := config.PolymorphicID
	if id == "" {
		return fmt.Errorf("no active identity")
	}

	if !id.IsUser() && !id.IsMachine() {
		return fmt.Errorf("unsupported identity for operation: %s", id)
	}

	userID, err := id.ID()
	if err != nil {
		return err
	}

	token, err := client.CreateToken(&api.CreateTokenRequest{UserID: userID})
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

	execCredential = &clientauthenticationv1beta1.ExecCredential{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ExecCredential",
			APIVersion: clientauthenticationv1beta1.SchemeGroupVersion.String(),
		},
		Spec: clientauthenticationv1beta1.ExecCredentialSpec{},
		Status: &clientauthenticationv1beta1.ExecCredentialStatus{
			Token:               token.Token,
			ExpirationTimestamp: &metav1.Time{Time: token.Expires},
		},
	}

	bts, err := json.Marshal(execCredential)
	if err != nil {
		return err
	}

	fmt.Println(string(bts))

	return nil
}
