package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/myopenfactory/edi-connector/v2/credentials"
	"gopkg.in/yaml.v3"
)

type ProcessConfig struct {
	Id       string
	Type     string
	AuthName string
	Settings map[string]any
}

type LogOptions struct {
	Level  string
	Folder string
}

type Config struct {
	Proxy       string
	RunWaitTime time.Duration
	Inbounds    []ProcessConfig
	Outbounds   []ProcessConfig
	Log         LogOptions
	Url         string
	Username    string
	Password    string
	CAFile      string `mapstructure:"cafile"`
	ServiceName string
}

func ReadConfig(configFile string) (Config, string, error) {
	if configFile == "" {
		workdir, err := os.Getwd()
		if err != nil {
			return Config{}, "", fmt.Errorf("failed to get working directory: %w", err)
		}
		searchLocations := []string{workdir}
		switch {
		case runtime.GOOS == "windows":
			searchLocations = append(searchLocations, filepath.Join(os.Getenv("ProgramData"), "myOpenFactory", "EDI-Connector"))
		case runtime.GOOS == "linux":
			searchLocations = append(searchLocations, filepath.Join("etc", "myopenfactory", "edi-connector"))
		}

		for _, searchLocation := range searchLocations {
			path := filepath.Join(searchLocation, "config.yaml")
			if _, err := os.Stat(path); os.IsNotExist(err) {
				continue
			}
			configFile = path
		}
	}
	file, err := os.Open(configFile)
	if err != nil {
		return Config{}, "", fmt.Errorf("failed to read config file: %w", err)
	}
	defer file.Close()

	var cfg Config
	cfg.RunWaitTime = time.Minute
	cfg.Url = "https://rest.ediplatform.services"
	cfg.ServiceName = "EDI-Connector"
	if proxy := os.Getenv("HTTP_PROXY"); proxy != "" {
		cfg.Proxy = proxy
	}
	if proxy := os.Getenv("HTTPS_PROXY"); proxy != "" {
		cfg.Proxy = proxy
	}

	if err := yaml.NewDecoder(file).Decode(&cfg); err != nil {
		return Config{}, "", fmt.Errorf("failed to decode configuration file: %w", err)
	}

	credManager := credentials.NewCredManager(cfg.ServiceName)
	if cfg.Password != "" {
		err := credManager.CreateCredential("", cfg.Username, cfg.Password)
		if err != nil {
			return Config{}, "", fmt.Errorf("failed to save credentials: %w", err)
		}
		return Config{}, "", fmt.Errorf("password found in config file. It has been stored in the credential manager. Please remove the password from the config file and run again")
	}
	auth, err := credManager.GetCredential("")
	if err != nil {
		return Config{}, "", fmt.Errorf("failed to retrieve username and password from credential manager")
	}
	cfg.Username = auth.Username
	cfg.Password = auth.Password

	return cfg, configFile, nil
}
