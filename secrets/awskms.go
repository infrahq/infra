package secrets

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
)

// ensure this interface is implemented properly
var _ SecretSymmetricKeyProvider = &AWSKMSSecretProvider{}

type AWSKMSSecretProvider struct {
	kms kmsiface.KMSAPI
}

func NewAWSKMSSecretProvider(kmssvc kmsiface.KMSAPI) (*AWSKMSSecretProvider, error) {
	return &AWSKMSSecretProvider{
		kms: kmssvc,
	}, nil
}

func (k *AWSKMSSecretProvider) DecryptDataKey(rootKeyID string, keyData []byte) (*SymmetricKey, error) {
	req, out := k.kms.DecryptRequest(&kms.DecryptInput{
		KeyId:               &rootKeyID,
		EncryptionAlgorithm: aws.String("AES_256"),
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

func (k *AWSKMSSecretProvider) GenerateDataKey(name, rootKeyID string) (*SymmetricKey, error) {
	if rootKeyID == "" {
		ko, err := k.generateRootKey(name + ":root")
		if err != nil {
			return nil, fmt.Errorf("kms: generate root key: %w", err)
		}

		rootKeyID = *ko.KeyMetadata.KeyId
	}

	dko, err := k.kms.GenerateDataKey(&kms.GenerateDataKeyInput{
		KeySpec: aws.String("AES_256"),
		KeyId:   aws.String(rootKeyID),
	})
	if err != nil {
		return nil, fmt.Errorf("kms: generate data key: %w", err)
	}

	return &SymmetricKey{
		unencrypted: dko.Plaintext,
		Encrypted:   dko.CiphertextBlob,
		RootKeyID:   rootKeyID,
		Algorithm:   "AES_256",
	}, nil
}
