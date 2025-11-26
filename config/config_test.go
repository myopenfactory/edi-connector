package config

import (
	"runtime"
	"strings"
	"testing"
)

var (
	yamlConfig = `
proxy: localhost
runWaitTime: "5m"
log:
  level: DEBUG
  folder: path
  type: type
url: testurl
caFile: testca
inbounds:
- id: 4711
  type: test
  authName: auth
  settings:
    test1: test2
    test3:
      test4: value
outbounds:
- id: 4711
  type: test
  authName: auth
  settings:
    test1: test2
    test3:
      test4: value
- id: 4711
  type: test
  authName: auth
  settings:
    test1: test2
    test3:
      test4: value
`
	jsonConfig = `
{
  "proxy": "localhost",
  "runWaitTime": "5m",
  "log": {
    "level": "DEBUG",
    "folder": "path",
    "type": "type"
  },
  "url": "testurl",
  "caFile": "testca",
  "inbounds": [
    {
      "id": "4711",
      "type": "test",
      "authName": "auth",
      "settings": {
        "test1": "test2",
        "test3": {
          "test4": "value"
        }
      }
    }
  ],
  "outbounds": [
    {
      "id": "4711",
      "type": "test",
      "authName": "auth",
      "settings": {
        "test1": "test2",
        "test3": {
          "test4": "value"
        }
      }
    },
    {
      "id": "4711",
      "type": "test",
      "authName": "auth",
      "settings": {
        "test1": "test2",
        "test3": {
          "test4": "value"
        }
      }
    }
  ]
}`
)

func TestFormatFromFileName(t *testing.T) {
	f := formatFromFileName("test.json")
	if f != Json {
		t.Errorf("wrong type parsed, wanted Json got: %v", f)
	}
	f = formatFromFileName("test.yaml")
	if f != Yaml {
		t.Errorf("wrong type parsed, wanted Yaml got: %v", f)
	}
	f = formatFromFileName("test.txt")
	if f != Error {
		t.Errorf("wrong type parsed, wanted Error got: %v", f)
	}
}

func TestNewConfigFromEmpty(t *testing.T) {
	cfg, err := ReadConfig(nil, Yaml)
	if err != nil {
		t.Errorf("failed to read configfile: %v", err)
		return
	}
	if cfg.RunWaitTime != "1m" {
		t.Errorf("wrong runWaitTime wanted 1m got: %v", cfg.RunWaitTime)
	}
	if cfg.Url != "https://rest.ediplatform.services" {
		t.Errorf("wrong url wanted 'https://rest.ediplatform.services' got: %v", cfg.Url)
	}
	if runtime.GOOS == "windows" {
		if cfg.Log.Type != "EVENT" {
			t.Errorf("wrong log type wanted 'EVENT' got: %v", cfg.Log.Type)
		}
	} else {
		if cfg.Log.Type != "STDOUT_TEXT" {
			t.Errorf("wrong log type wanted 'STDOUT_TEXT' got: %v", cfg.Log.Type)
		}
	}
}

func TestNewConfigFromYamlMaximal(t *testing.T) {
	checkConfig(t, yamlConfig, Yaml)
}

func TestNewConfigFromJsonMaximal(t *testing.T) {
	checkConfig(t, jsonConfig, Json)
}

func checkConfig(t *testing.T, configString string, format Format) {
	cfg, err := ReadConfig(strings.NewReader(configString), format)
	if err != nil {
		t.Errorf("failed to read configfile: %v", err)
		return
	}
	if cfg.Proxy != "localhost" {
		t.Errorf("wrong proxy wanted 'localhost' got: %v", cfg.Proxy)
	}
	if cfg.RunWaitTime != "5m" {
		t.Errorf("wrong runWaitTime wanted 5m got: %v", cfg.RunWaitTime)
	}
	if cfg.Log.Level != "DEBUG" {
		t.Errorf("wrong log.level wanted 'DEBUG' got: %v", cfg.Log.Level)
	}
	if cfg.Log.Folder != "path" {
		t.Errorf("wrong log.folder wanted 'path' got: %v", cfg.Log.Folder)
	}
	if cfg.Log.Type != "type" {
		t.Errorf("wrong log.type wanted 'type' got: %v", cfg.Log.Type)
	}
	if cfg.Url != "testurl" {
		t.Errorf("wrong url wanted 'testurl' got: %v", cfg.Url)
	}
	if cfg.CAFile != "testca" {
		t.Errorf("wrong url wanted 'testca' got: %v", cfg.CAFile)
	}

	if len(cfg.Inbounds) != 1 {
		t.Errorf("wrong number of inbound processes wanted 1 got: %v", len(cfg.Inbounds))
	}
	for _, process := range cfg.Inbounds {
		checkProcess(t, process)
	}
	if len(cfg.Outbounds) != 2 {
		t.Errorf("wrong number of outbound processes wanted 2 got: %v", len(cfg.Outbounds))
	}
	for _, process := range cfg.Outbounds {
		checkProcess(t, process)
	}
}

func checkProcess(t *testing.T, cfg ProcessConfig) {
	if cfg.Id != "4711" {
		t.Errorf("wrong id wanted '4711' got: %v", cfg.Id)
	}
	if cfg.Type != "test" {
		t.Errorf("wrong type wanted 'test' got: %v", cfg.Type)
	}
	if cfg.AuthName != "auth" {
		t.Errorf("wrong authname wanted 'auth' got: %v", cfg.AuthName)
	}
	settings := settings{}
	err := Decode(cfg.Settings, &settings)
	if err != nil {
		t.Errorf("failed to decode settings: %v", err)
	}
	if settings.Test1 != "test2" {
		t.Errorf("wrong test1 wanted 'test2' got: %v", settings.Test1)
	}
	if settings.Test3.Test4 != "value" {
		t.Errorf("wrong test4 wanted 'value' got: %v", settings.Test3.Test4)
	}

}

type settings struct {
	Test1 string
	Test3 testStruct
}

type testStruct struct {
	Test4 string
}
