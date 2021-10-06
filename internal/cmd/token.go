package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/infrahq/infra/internal/api"
	"golang.org/x/term"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientauthenticationv1beta1 "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"
)

func token(destination string) error {
	execCredential := &clientauthenticationv1beta1.ExecCredential{}

	err := getCache("tokens", destination, execCredential)
	if err != nil {
		return err
	}

	if isExpired(execCredential) {
		client, err := apiClientFromConfig()
		if err != nil {
			return err
		}

		ctx, err := apiContextFromConfig()
		if err != nil {
			return err
		}

		credReq := client.TokensApi.CreateToken(ctx).Body(api.TokenRequest{Destination: &destination})

		cred, res, err := credReq.Execute()
		if err != nil {
			switch res.StatusCode {
			case http.StatusForbidden:
				if !term.IsTerminal(int(os.Stdin.Fd())) {
					return &ErrUnauthenticated{}
				}

				fmt.Fprintln(os.Stderr, "Session has expired.")

				if err = login("", false); err != nil {
					return err
				}

				return token(destination)

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
		if err := setCache("tokens", destination, execCredential); err != nil {
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

// getCache populates obj with whatever is in the cache
func getCache(path, name string, obj interface{}) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	path = filepath.Join(homeDir, ".infra", "cache", path)
	if err = os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	fullPath := filepath.Join(path, name)

	f, err := os.Open(fullPath)
	if os.IsNotExist(err) {
		return nil
	}

	if err != nil {
		return err
	}

	defer f.Close()

	d := json.NewDecoder(f)
	if err := d.Decode(obj); err != nil {
		return err
	}

	return nil
}

func setCache(path, name string, obj interface{}) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	path = filepath.Join(homeDir, ".infra", "cache", path)
	fullPath := filepath.Join(path, name)

	if err = os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer f.Close()

	e := json.NewEncoder(f)
	if err := e.Encode(obj); err != nil {
		return err
	}

	return nil
}

// isExpired returns true if the credential is expired or incomplete.
// it only returns false if the credential is good to use.
func isExpired(cred *clientauthenticationv1beta1.ExecCredential) bool {
	if cred == nil {
		return true
	}

	if cred.Status == nil {
		return true
	}

	if cred.Status.ExpirationTimestamp == nil {
		return true
	}

	// make sure it expires in more than 1 second from now.
	now := time.Now().Add(1 * time.Second)
	// only valid if it hasn't expired yet
	return cred.Status.ExpirationTimestamp.Time.Before(now)
}
