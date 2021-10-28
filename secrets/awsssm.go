package secrets

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

var _ SecretStorage = &AWSSSM{}

// AWSSSM is the AWS System Manager Parameter Store (aka SSM PS)
type AWSSSM struct {
	AWSSSMConfig
	client *ssm.SSM
}

type AWSSSMConfig struct {
	AWSConfig
	KeyID string `yaml:"keyId"` // KMS key to use for decryption
}

func NewAWSSSMSecretProviderFromConfig(cfg AWSSSMConfig) (*AWSSSM, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, fmt.Errorf("creating aws session: %w", err)
	}

	awscfg := aws.NewConfig().
		WithCredentials(credentials.NewCredentials(&credentials.StaticProvider{
			Value: credentials.Value{
				AccessKeyID:     cfg.AccessKeyID,
				SecretAccessKey: cfg.SecretAccessKey,
			},
		})).
		WithEndpoint(cfg.Endpoint).
		WithRegion(cfg.Region)

	return &AWSSSM{
		AWSSSMConfig: cfg,
		client:       ssm.New(sess, awscfg),
	}, nil
}

func NewAWSSSM(client *ssm.SSM) *AWSSSM {
	return &AWSSSM{
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
func (s *AWSSSM) SetSecret(name string, secret []byte) error {
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
func (s *AWSSSM) GetSecret(name string) (secret []byte, err error) {
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
