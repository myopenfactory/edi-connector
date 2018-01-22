package config

import (
	"os"
	"strings"
	"fmt"

	"github.com/spf13/viper"
	"myopenfactory.io/x/app/tatooine/pkg/log"
)

func Load(file string) error {
	viper.SetEnvPrefix("client")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	vp := viper.GetString("config")
	if vp != "" {
		file = vp
	}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return fmt.Errorf("config file does not exists: %v", file)
	}

	viper.SetConfigFile(file)
	log.Infof("Using config file: %v", file)
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to load config: %s: %s", file, err)
	}

	return nil
}