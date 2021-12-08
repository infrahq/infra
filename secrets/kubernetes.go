package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var _ SecretStorage = &KubernetesSecretProvider{}

type KubernetesSecretProvider struct {
	KubernetesConfig
	client *kubernetes.Clientset
}

type KubernetesConfig struct {
	Namespace string `yaml:"namespace"`
}

func NewKubernetesConfig() KubernetesConfig {
	return KubernetesConfig{
		Namespace: getDefaultNamespace(),
	}
}

func NewKubernetesSecretProviderFromConfig(cfg KubernetesConfig) (*KubernetesSecretProvider, error) {
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("getting in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("creating k8s config: %w", err)
	}

	return &KubernetesSecretProvider{
		KubernetesConfig: cfg,
		client:           clientset,
	}, nil
}

func NewKubernetesSecretProvider(client *kubernetes.Clientset, namespace string) *KubernetesSecretProvider {
	return &KubernetesSecretProvider{
		KubernetesConfig: KubernetesConfig{
			Namespace: namespace,
		},
		client: client,
	}
}

var kubernetesInvalidKeyCharacters = regexp.MustCompile(`[^-._a-zA-Z0-9/]`)

// Use secrets when you don't want to store the underlying data, eg secret tokens
func (k *KubernetesSecretProvider) SetSecret(name string, secret []byte) error {
	name = kubernetesInvalidKeyCharacters.ReplaceAllLiteralString(name, "_")

	secretParts := strings.Split(name, "/")
	if len(secretParts) != 2 {
		return fmt.Errorf("invalid Kubernetes secret path specified, expected exactly 2 parts but was %d", len(secretParts))
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
			return fmt.Errorf("k8s: creating secret: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("k8s: patching secret: %w", err)
	}

	return nil
}

func (k *KubernetesSecretProvider) GetSecret(name string) (secret []byte, err error) {
	name = kubernetesInvalidKeyCharacters.ReplaceAllLiteralString(name, "_")

	secretParts := strings.Split(name, "/")
	if len(secretParts) != 2 {
		return nil, fmt.Errorf("invalid Kubernetes secret path specified, expected exactly 2 parts but was %d", len(secretParts))
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
		return nil, fmt.Errorf("secret could not be found in kubernetes: %s", name)
	}

	return secretVal, nil
}

var defaultInstallNamespace = "default"

func getDefaultNamespace() string {
	contents, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return defaultInstallNamespace
	}

	if len(contents) > 0 {
		return string(contents)
	}

	return defaultInstallNamespace
}
