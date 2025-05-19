package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"gopkg.in/yaml.v3"
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
	cfg.Url = "https://edi.myopenfactory.net"
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

	return cfg, configFile, nil
}
