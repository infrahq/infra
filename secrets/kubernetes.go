package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

type KubernetesSecretProvider struct {
	Namespace string
	client    *kubernetes.Clientset
}

func NewKubernetesSecretProvider(client *kubernetes.Clientset, namespace string) *KubernetesSecretProvider {
	return &KubernetesSecretProvider{
		Namespace: namespace,
		client:    client,
	}
}

var kubernetesInvalidKeyCharacters = regexp.MustCompile(`[^-._a-zA-Z0-9/]`)

// Use secrets when you don't want to store the underlying data, eg secret tokens
func (k *KubernetesSecretProvider) SetSecret(name string, secret []byte) error {
	name = kubernetesInvalidKeyCharacters.ReplaceAllLiteralString(name, "_")

	secretParts := strings.Split(name, "/")
	if len(secretParts) != 2 {
		return errors.New("invalid Kubernetes secret path specified, expected exactly 2 parts but was " + fmt.Sprint(len(secretParts)))
	}

	objName := secretParts[0]
	// key must match [-._a-zA-Z0-9]+
	key := secretParts[1]

	data := map[string][]byte{}
	data[key] = secret
	patch := v1.Secret{
		Data: data,
	}

	d, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	_, err = k.client.CoreV1().Secrets(k.Namespace).Patch(
		context.TODO(),
		objName,
		types.StrategicMergePatchType,
		d,
		metav1.PatchOptions{},
	)
	if err != nil && strings.Contains(err.Error(), "not found") {
		_, err = k.client.CoreV1().Secrets(k.Namespace).Create(context.TODO(), &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: objName,
			},
			Data: data,
		}, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("creating secret: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("patching secret: %w", err)
	}

	return nil
}

func (k *KubernetesSecretProvider) GetSecret(name string) (secret []byte, err error) {
	name = kubernetesInvalidKeyCharacters.ReplaceAllLiteralString(name, "_")

	secretParts := strings.Split(name, "/")
	if len(secretParts) != 2 {
		return nil, errors.New("invalid Kubernetes secret path specified, expected exactly 2 parts but was " + fmt.Sprint(len(secretParts)))
	}

	objName := secretParts[0]
	key := secretParts[1]

	retrieved, err := k.client.CoreV1().Secrets(k.Namespace).Get(context.TODO(), objName, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, nil
		}

		return nil, err
	}

	secretVal, ok := retrieved.Data[key]
	if !ok {
		return nil, errors.New("secret could not be found in kubernetes: " + name)
	}

	return secretVal, nil
}
