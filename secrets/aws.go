package secrets

type AWSConfig struct {
	Endpoint        string `mapstructure:"endpoint" validate:"required"`
	Region          string `mapstructure:"region" validate:"required"`
	AccessKeyID     string `mapstructure:"accessKeyID" validate:"required"`
	SecretAccessKey string `mapstructure:"secretAccessKey" validate:"required"`
}
