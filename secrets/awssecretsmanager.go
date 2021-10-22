package secrets

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

var _ SecretStorage = &AWSSecretsManager{}

type AWSSecretsManager struct {
	UseSecretMaps bool // TODO: support storing to json maps if this is enabled.

	client *secretsmanager.SecretsManager
}

func NewAWSSecretsManager(client *secretsmanager.SecretsManager) *AWSSecretsManager {
	return &AWSSecretsManager{
		client: client,
	}
}

// SetSecret
// must have the secretsmanager:CreateSecret permission
// if using tags, must have secretsmanager:TagResource
// if using kms customer-managed keys, also need:
// - kms:GenerateDataKey
// - kms:Decrypt
func (s *AWSSecretsManager) SetSecret(name string, secret []byte) error {
	name = strings.ReplaceAll(name, ":", "_")

	_, err := s.client.CreateSecretWithContext(context.TODO(), &secretsmanager.CreateSecretInput{
		Name:         &name,
		SecretBinary: secret,
	})
	if err != nil {
		var aerr awserr.Error
		if errors.As(err, &aerr) {
			if aerr.Code() == secretsmanager.ErrCodeResourceExistsException {
				// try replacing instead
				_, err = s.client.UpdateSecretWithContext(context.TODO(), &secretsmanager.UpdateSecretInput{
					SecretBinary: secret,
					SecretId:     &name,
				})
				if err != nil {
					return fmt.Errorf("aws sm: update secret: %w", err)
				}

				return nil
			}
		}

		return fmt.Errorf("aws sm: creating secret: %w", err)
	}

	return nil
}

// GetSecret
// must have permission secretsmanager:GetSecretValue
// kms:Decrypt - required only if you use a customer-managed Amazon Web Services KMS key to encrypt the secret
func (s *AWSSecretsManager) GetSecret(name string) (secret []byte, err error) {
	name = strings.ReplaceAll(name, ":", "_")

	sec, err := s.client.GetSecretValueWithContext(context.TODO(), &secretsmanager.GetSecretValueInput{
		SecretId: &name,
	})
	if err != nil {
		var aerr awserr.Error
		if errors.As(err, &aerr) {
			if aerr.Code() == secretsmanager.ErrCodeResourceNotFoundException {
				return nil, nil
			}
		}

		return nil, fmt.Errorf("aws sm: get secret: %w", err)
	}

	return sec.SecretBinary, nil
}
