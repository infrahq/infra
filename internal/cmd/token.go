package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/infrahq/infra/internal/api"
	"golang.org/x/term"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientauthenticationv1beta1 "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"
)

func token(destination string) error {
	client, err := apiClientFromConfig()
	if err != nil {
		return err
	}

	ctx, err := apiContextFromConfig()
	if err != nil {
		return err
	}

	execCredential := &clientauthenticationv1beta1.ExecCredential{}

	err = getCache("dest_tokens", destination, execCredential)
	if err != nil {
		return err
	}

	if isExpired(execCredential) {
		credReq := client.CredsApi.CreateCred(ctx).Body(api.CredRequest{Destination: &destination})

		cred, res, err := credReq.Execute()
		if err != nil {
			switch res.StatusCode {
			case http.StatusForbidden:
				if !term.IsTerminal(int(os.Stdin.Fd())) {
					return err
				}

				config, err := readCurrentConfig()
				if err != nil {
					return &ErrUnauthenticated{}
				}

				err = login(config.Host)
				if err != nil {
					return &ErrUnauthenticated{}
				}

				ctx, err := apiContextFromConfig()
				if err != nil {
					return &ErrUnauthenticated{}
				}

				cred, _, err = client.CredsApi.CreateCred(ctx).Body(api.CredRequest{Destination: &destination}).Execute()
				if err != nil {
					return &ErrUnauthenticated{}
				}

			default:
				return err
			}
		}

		execCredential = &clientauthenticationv1beta1.ExecCredential{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ExecCredential",
				APIVersion: clientauthenticationv1beta1.SchemeGroupVersion.String(),
			},
			Spec: clientauthenticationv1beta1.ExecCredentialSpec{},
			Status: &clientauthenticationv1beta1.ExecCredentialStatus{
				Token:               cred.Token,
				ExpirationTimestamp: &metav1.Time{Time: time.Unix(cred.Expires, 0)},
			},
		}
		if err := setCache("dest_tokens", destination, execCredential); err != nil {
			return err
		}
	}

	bts, err := json.Marshal(execCredential)
	if err != nil {
		return err
	}

	fmt.Println(string(bts))

	return nil
}
