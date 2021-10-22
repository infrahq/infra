package cmd

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type GlobalOptions struct {
	ConfigFile string
	Host       string
	Verbose    int
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

	configfile := cmd.Flags().Lookup("configfile").Value.String()
	if configfile != "" {
		v.SetConfigFile(configfile)
	}

	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.SetEnvPrefix("INFRA")
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
