package secrets

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ssm"
)

var _ SecretStorage = &AWSSystemManagerParameterStore{}

type AWSSystemManagerParameterStore struct {
	KeyID  string // KMS key to use for decryption
	client *ssm.SSM
}

func NewAWSSystemManagerParameterStore(client *ssm.SSM) *AWSSystemManagerParameterStore {
	return &AWSSystemManagerParameterStore{
		client: client,
	}
}

var invalidSecretNameChars = regexp.MustCompile(`[^a-zA-Z0-9_.-/]`)

// SetSecret
// must have the secretsmanager:CreateSecret permission
// if using tags, must have secretsmanager:TagResource
// if using kms customer-managed keys, also need:
// - kms:GenerateDataKey
// - kms:Decrypt
func (s *AWSSystemManagerParameterStore) SetSecret(name string, secret []byte) error {
	name = invalidSecretNameChars.ReplaceAllString(name, "_")
	secretStr := string(secret)

	var keyID *string
	if len(s.KeyID) > 0 {
		keyID = &s.KeyID
	}

	_, err := s.client.PutParameterWithContext(context.TODO(), &ssm.PutParameterInput{
		KeyId:     keyID, // the kms key to use to encrypt. empty = default key
		Name:      &name,
		Overwrite: aws.Bool(true),
		Type:      aws.String("SecureString"),
		Value:     &secretStr,
	})
	if err != nil {
		return fmt.Errorf("ssm: creating secret: %w", err)
	}

	return nil
}

// GetSecret
// must have permission secretsmanager:GetSecretValue
// kms:Decrypt - required only if you use a customer-managed Amazon Web Services KMS key to encrypt the secret
func (s *AWSSystemManagerParameterStore) GetSecret(name string) (secret []byte, err error) {
	name = invalidSecretNameChars.ReplaceAllString(name, "_")

	p, err := s.client.GetParameterWithContext(context.TODO(), &ssm.GetParameterInput{
		Name:           &name,
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		var aerr awserr.Error
		if errors.As(err, &aerr) {
			if aerr.Code() == ssm.ErrCodeParameterNotFound {
				return nil, nil
			}
		}

		return nil, fmt.Errorf("ssm: get secret: %w", err)
	}

	return []byte(*p.Parameter.Value), nil
}
