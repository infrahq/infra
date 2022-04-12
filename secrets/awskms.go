package secrets

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
)

// ensure this interface is implemented properly
var _ SymmetricKeyProvider = &AWSKMSSecretProvider{}

type AWSKMSSecretProvider struct {
	AWSKMSConfig

	kms kmsiface.KMSAPI
}

type AWSKMSConfig struct {
	AWSConfig `mapstructure:",squash"`

	EncryptionAlgorithm string `mapstructure:"encryptionAlgorithm"`
	// aws tags?
}

func NewAWSKMSConfig() AWSKMSConfig {
	return AWSKMSConfig{
		EncryptionAlgorithm: "AES_256",
	}
}

func NewAWSKMSSecretProviderFromConfig(cfg AWSKMSConfig) (*AWSKMSSecretProvider, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, fmt.Errorf("creating aws session: %w", err)
	}

	// for kms service
	awscfg := aws.NewConfig().
		WithEndpoint(cfg.Endpoint).
		WithCredentials(credentials.NewStaticCredentialsFromCreds(
			credentials.Value{
				AccessKeyID:     cfg.AccessKeyID,
				SecretAccessKey: cfg.SecretAccessKey,
			})).
		WithRegion(cfg.Region)

	return &AWSKMSSecretProvider{
		AWSKMSConfig: cfg,
		kms:          kms.New(sess, awscfg),
	}, nil
}

func NewAWSKMSSecretProvider(kmssvc kmsiface.KMSAPI) (*AWSKMSSecretProvider, error) {
	return &AWSKMSSecretProvider{
		AWSKMSConfig: NewAWSKMSConfig(),
		kms:          kmssvc,
	}, nil
}

func (k *AWSKMSSecretProvider) DecryptDataKey(rootKeyID string, keyData []byte) (*SymmetricKey, error) {
	req, out := k.kms.DecryptRequest(&kms.DecryptInput{
		KeyId:               &rootKeyID,
		EncryptionAlgorithm: &k.EncryptionAlgorithm,
		CiphertextBlob:      keyData,
	})
	if err := req.Send(); err != nil {
		return nil, fmt.Errorf("kms: decrypt data key: %w", err)
	}

	return &SymmetricKey{
		unencrypted: out.Plaintext,
		Encrypted:   keyData,
		Algorithm:   *out.EncryptionAlgorithm,
		RootKeyID:   rootKeyID,
	}, nil
}

func (k *AWSKMSSecretProvider) generateRootKey(name string) (*kms.CreateKeyOutput, error) {
	return k.kms.CreateKey(&kms.CreateKeyInput{
		MultiRegion: aws.Bool(true),
		Tags: []*kms.Tag{{
			TagKey:   aws.String("alias"),
			TagValue: &name,
		}},
	})
}

func (k *AWSKMSSecretProvider) GenerateDataKey(rootKeyID string) (*SymmetricKey, error) {
	if rootKeyID == "" {
		ko, err := k.generateRootKey("infra:root")
		if err != nil {
			return nil, fmt.Errorf("kms: generate root key: %w", err)
		}

		rootKeyID = *ko.KeyMetadata.KeyId
	}

	dko, err := k.kms.GenerateDataKey(&kms.GenerateDataKeyInput{
		KeySpec: &k.EncryptionAlgorithm,
		KeyId:   aws.String(rootKeyID),
	})
	if err != nil {
		return nil, fmt.Errorf("kms: generate data key: %w", err)
	}

	return &SymmetricKey{
		unencrypted: dko.Plaintext,
		Encrypted:   dko.CiphertextBlob,
		RootKeyID:   rootKeyID,
		Algorithm:   k.EncryptionAlgorithm,
	}, nil
}
