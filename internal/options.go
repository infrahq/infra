package internal

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type GlobalOptions struct {
	Host       string `mapstructure:"host"`
	ConfigFile string `mapstructure:"config-file"`
	LogLevel   string `mapstructure:"log-level"`
}

func ParseOptions(cmd *cobra.Command, options interface{}) error {
	v := viper.New()

	if err := v.BindPFlags(cmd.Flags()); err != nil {
		return err
	}

	v.SetConfigName("config")

	v.AddConfigPath("/etc/infra")
	v.AddConfigPath("$HOME/.infra")
	v.AddConfigPath(".")

	configfile := cmd.Flags().Lookup("config-file").Value.String()
	if configfile != "" {
		v.SetConfigFile(configfile)
	}

	v.SetEnvPrefix("INFRA")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	var errConfigFileNotFound *viper.ConfigFileNotFoundError
	if err := v.ReadInConfig(); err != nil {
		if errors.As(err, &errConfigFileNotFound) {
			return err
		}
	}

	if err := v.Unmarshal(options); err != nil {
		return err
	}

	return nil
}
