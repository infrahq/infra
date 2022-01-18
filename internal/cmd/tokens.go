package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientauthenticationv1beta1 "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"

	"github.com/infrahq/infra/internal/api"
)

func tokensCreate(destination string) error {
	execCredential := &clientauthenticationv1beta1.ExecCredential{}

	err := getCache("tokens", destination, execCredential)
	if !os.IsNotExist(err) && err != nil {
		return err
	}

	if os.IsNotExist(err) || isExpired(execCredential) {
		client, err := defaultAPIClient()
		if err != nil {
			return err
		}

		token, err := client.CreateToken(&api.TokenRequest{Destination: destination})
		if err != nil {
			if errors.Is(err, api.ErrForbidden) {
				fmt.Fprintln(os.Stderr, "Session has expired.")

				if err = login(&LoginOptions{Current: true}); err != nil {
					return err
				}

				return tokensCreate(destination)
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
				ExpirationTimestamp: &metav1.Time{Time: time.Unix(token.Expires, 0)},
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
	infraDir, err := infraHomeDir()
	if err != nil {
		return err
	}

	path = filepath.Join(infraDir, "cache", path)
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	fullPath := filepath.Join(path, name)

	if _, err := os.Stat(fullPath); err != nil {
		return err
	}

	f, err := os.Open(fullPath)
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
	infraDir, err := infraHomeDir()
	if err != nil {
		return err
	}

	path = filepath.Join(infraDir, "cache", path)
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	fullPath := filepath.Join(path, name)

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
