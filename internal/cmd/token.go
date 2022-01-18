package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientauthenticationv1beta1 "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"

	"github.com/infrahq/infra/internal/api"
)

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

	if config.ID == 0 {
		return fmt.Errorf("no active user")
	}

	token, err := client.CreateToken(&api.CreateTokenRequest{
		UserID: config.ID,
	})
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
