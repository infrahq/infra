package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type SecretReader interface {
	Get(secretName string, client *kubernetes.Clientset) (string, error)
}

type KubeSecretReader struct {
	Namespace string
}

func NewSecretReader(namespace string) SecretReader {
	return &KubeSecretReader{Namespace: namespace}
}

// Get returns a K8s secret object with the specified name from a Kubernetes configuration if it exists
func (ksr *KubeSecretReader) Get(secretName string, client *kubernetes.Clientset) (string, error) {
	secretParts := strings.Split(secretName, "/")
	if len(secretParts) != 2 {
		return "", errors.New("invalid Kubernetes secret path seperated, expected exactly 2 parts but was " + fmt.Sprint(len(secretParts)))
	}
	objName := secretParts[0]
	key := secretParts[1]

	retrieved, err := client.CoreV1().Secrets(ksr.Namespace).Get(context.TODO(), objName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	secretVal := retrieved.Data[key]
	if string(secretVal) == "" {
		return "", errors.New("secret could not be found in kubernetes: " + secretName)
	}
	return string(secretVal), nil
}
