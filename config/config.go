package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

type ProcessConfig struct {
	Id       string         `json:"id" yaml:"id"`
	Type     string         `json:"type" yaml:"type"`
	AuthName string         `json:"authName" yaml:"authName"`
	Settings map[string]any `json:"settings" yaml:"settings"`
}

type LogOptions struct {
	Level  string `json:"level" yaml:"level"`
	Folder string `json:"folder" yaml:"folder"`
	Type   string `json:"type" yaml:"type"`
}

type Config struct {
	InstancePort int             `json:"instancePort" yaml:"instancePort"`
	Proxy        string          `json:"proxy" yaml:"proxy"`
	RunWaitTime  string          `json:"runWaitTime" yaml:"runWaitTime"`
	Inbounds     []ProcessConfig `json:"inbounds" yaml:"inbounds"`
	Outbounds    []ProcessConfig `json:"outbounds" yaml:"outbounds"`
	Log          LogOptions      `json:"log" yaml:"log"`
	Url          string          `json:"url" yaml:"url"`
	CAFile       string          `json:"caFile" yaml:"caFile"`
}

type Format int

type decoder interface {
	Decode(v any) error
}

const (
	Error Format = iota
	Json
	Yaml
)

func ReadConfigFromFile(configFile string) (Config, string, error) {
	format := formatFromFileName(configFile)
	if format == Error {
		workdir, err := os.Getwd()
		if err != nil {
			return Config{}, "", fmt.Errorf("failed to get working directory: %w", err)
		}
		searchLocations := []string{workdir}
		switch {
		case runtime.GOOS == "windows":
			searchLocations = append(searchLocations, filepath.Join(os.Getenv("ProgramData"), "myOpenFactory Software GmbH", "EDI-Connector"))
		case runtime.GOOS == "linux":
			searchLocations = append(searchLocations, filepath.Join("etc", "myopenfactory", "edi-connector"))
		}

		for _, searchLocation := range searchLocations {
			entires, err := os.ReadDir(searchLocation)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return Config{}, "", fmt.Errorf("failed to list config directory: %w", err)
			}
			for _, entry := range entires {
				if entry.IsDir() {
					continue
				}
				format = formatFromFileName(entry.Name())
				if format != Error {
					configFile = filepath.Join(searchLocation, entry.Name())
					break
				}
			}
		}
		if format == Error {
			return Config{}, "", fmt.Errorf("no config file found")
		}
	}
	file, err := os.Open(configFile)
	if err != nil {
		return Config{}, "", fmt.Errorf("failed to read config file: %w", err)
	}
	defer file.Close()
	config, err := ReadConfig(file, format)
	return config, configFile, err
}

func formatFromFileName(fileName string) Format {
	if strings.HasSuffix(fileName, ".json") {
		return Json
	}
	if strings.HasSuffix(fileName, ".yaml") {
		return Yaml
	}
	return Error
}

func ReadConfig(configReader io.Reader, format Format) (Config, error) {
	var cfg Config
	cfg.RunWaitTime = "1m"
	cfg.Url = "https://rest.ediplatform.services"
	if proxy := os.Getenv("HTTP_PROXY"); proxy != "" {
		cfg.Proxy = proxy
	}
	if proxy := os.Getenv("HTTPS_PROXY"); proxy != "" {
		cfg.Proxy = proxy
	}
	cfg.Log.Type = "STDOUT_TEXT"
	if runtime.GOOS == "windows" {
		cfg.Log.Type = "EVENT"
	}
	if configReader != nil && format != Error {
		var decoder decoder
		switch format {
		case Json:
			decoder = json.NewDecoder(configReader)
		case Yaml:
			decoder = yaml.NewDecoder(configReader)
		}
		if err := decoder.Decode(&cfg); err != nil {
			return Config{}, fmt.Errorf("failed to decode configuration file: %w", err)
		}
	}

	return cfg, nil
}

func Decode(source map[string]any, target any) error {
	bytes, err := json.Marshal(source)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, target)
}
