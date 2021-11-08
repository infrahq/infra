package secrets

type AWSConfig struct {
	Endpoint        string `yaml:"endpoint" validate:"required"`
	Region          string `yaml:"region" validate:"required"`
	AccessKeyID     string `yaml:"accessKeyID" validate:"required"`
	SecretAccessKey string `yaml:"secretAccessKey" validate:"required"`
}
