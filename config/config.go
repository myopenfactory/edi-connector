package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type ContextKey string

const (
	ConfigContextKey = ContextKey("config")
	LoggerContextKey = ContextKey("logger")
)

type ProcessConfig struct {
	Id       string
	Type     string
	Settings map[string]any
}

type LogOptions struct {
	Level  string
	Folder string
}

type Config struct {
	Proxy             string
	RunWaitTime       time.Duration
	Inbounds          []ProcessConfig
	Outbounds         []ProcessConfig
	Log               LogOptions
	Url               string
	Username          string
	Password          string
	CAFile            string `mapstructure:"cafile"`
	ClientCertificate string `mapstructure:"clientcert"`
}

func ReadConfig(configfile string) (Config, error) {
	if configfile != "" {
		viper.SetConfigFile(configfile)
	}

	viper.SetEnvPrefix("client")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	viper.SetConfigName("config")

	switch {
	case runtime.GOOS == "windows":
		viper.AddConfigPath(filepath.Join(os.Getenv("ProgramData"), "myOpenFactory", "EDI-Connector"))
	case runtime.GOOS == "linux":
		viper.AddConfigPath(filepath.Join("etc", "myopenfactory", "edi-connector"))
	}
	viper.AddConfigPath(".")

	viper.SetDefault("runwaittime", time.Minute)
	viper.SetDefault("url", "https://edi.myopenfactory.net")

	if proxy := os.Getenv("HTTP_PROXY"); proxy != "" {
		viper.SetDefault("proxy", proxy)
	}
	if proxy := os.Getenv("HTTPS_PROXY"); proxy != "" {
		viper.SetDefault("proxy", proxy)
	}

	if err := viper.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("failed to read config: %w", err)
	}

	if clientcert := viper.GetString("clientcert"); strings.HasPrefix(clientcert, "./") {
		viper.Set("clientcert", filepath.Join(filepath.Dir(viper.ConfigFileUsed()), clientcert))
	}

	if clientcert := viper.GetString("clientcert"); clientcert == "" {
		clientcert = filepath.Join(filepath.Dir(viper.ConfigFileUsed()), "client.crt")
		if _, err := os.Stat(clientcert); err == nil {
			viper.Set("clientcert", clientcert)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}
