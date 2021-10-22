package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type GlobalOptions struct {
	ConfigFile string
	Host string
	Verbose int
}

func ParseOptions(cmd *cobra.Command, options interface{}) error {
	v := viper.New()

	v.BindPFlags(cmd.Flags())

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

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	if err := v.Unmarshal(options); err != nil {
		return err
	}

	return nil
}
