package cmd

import (
	"context"
	"encoding/json"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientauthenticationv1beta1 "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"
)

func newTokensCmd(cli *CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "tokens",
		Short:  "Create & manage tokens",
		Hidden: true,
	}

	cmd.AddCommand(newTokensAddCmd(cli))

	return cmd
}

func newTokensAddCmd(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "Create a token",
		Args:  NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return tokensCreate(cli)
		},
	}
}

func tokensCreate(cli *CLI) error {
	ctx := context.Background()
	client, err := cli.apiClient()
	if err != nil {
		return err
	}

	token, err := client.CreateToken(ctx)
	if err != nil {
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

	cli.Output(string(bts))

	return nil
}
