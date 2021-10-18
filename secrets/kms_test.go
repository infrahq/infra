package secrets

// testing with https://github.com/nsmithuk/local-kms
// though not required, you can run a local kms with:
// docker run -p 8380:8080 nsmithuk/local-kms

// ensure this interface is implemented properly
var _ SecretProvider = &AWSKMSSecretProvider{}

// see secrets_test.go for all shared tests
