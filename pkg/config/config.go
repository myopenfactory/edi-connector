package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/myopenfactory/client/pkg/client"
	"github.com/myopenfactory/client/pkg/errors"
	"github.com/myopenfactory/client/pkg/log"
	"github.com/spf13/viper"
)

func ParseClientOptions() ([]client.Option, error) {
	const op errors.Op = "config.ParseClientOptions"
	opts := []client.Option{}

	if runWaitTime := viper.GetString("runwaittime"); runWaitTime != "" {
		d, err := time.ParseDuration(runWaitTime)
		if err != nil {
			return nil, err
		}
		opts = append(opts, client.WithRunWaitTime(d))
	}

	if healthWaitTime := viper.GetString("healthwaitttime"); healthWaitTime != "" {
		d, err := time.ParseDuration(healthWaitTime)
		if err != nil {
			return nil, err
		}
		opts = append(opts, client.WithHealthWaitTime(d))
	}

	if url := viper.GetString("url"); url != "" {
		opts = append(opts, client.WithURL(url))
	}

	cafile := viper.GetString("cafile")
	clientcert := viper.GetString("clientcert")
	if clientcert == "" {
		clientcert = filepath.Join(filepath.Dir(viper.ConfigFileUsed()), "client.crt")
	}
	if strings.HasPrefix(clientcert, "./") {
		clientcert = filepath.Join(filepath.Dir(viper.ConfigFileUsed()), clientcert)
	}
	if _, err := os.Stat(clientcert); os.IsNotExist(err) {
		return nil, errors.E(op, "client certificate does not exist", errors.KindUnexpected)
	}

	if proxy := viper.GetString("proxy"); proxy != "" {
		opts = append(opts, client.WithProxy(proxy))
	}
	if proxy := os.Getenv("HTTP_PROXY"); proxy != "" {
		opts = append(opts, client.WithProxy(proxy))
	}

	crt, err := tls.LoadX509KeyPair(clientcert, clientcert)
	if err != nil {
		return nil, errors.E(op, fmt.Errorf("error loading client certificate: %v", err))
	}
	opts = append(opts, client.WithMTLS(crt))

	if cafile != "" {
		pem, err := ioutil.ReadFile(cafile)
		if err != nil {
			return nil, errors.E(op, fmt.Errorf("error while loading ca certificates: %w", err))
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(pem)
		opts = append(opts, client.WithCertPool(pool))
	}

	opts = append(opts, client.WithLogger(log.New(
		ParseLogOptions()...,
	)))

	if username := viper.GetString("username"); username != "" {
		username = strings.TrimSpace(username)
		opts = append(opts, client.WithUsername(username))
	}

	if password := viper.GetString("password"); password != "" {
		password = strings.TrimSpace(password)
		opts = append(opts, client.WithPassword(password))
	}

	return opts, nil
}

func ParseLogOptions() []log.Option {
	opts := []log.Option{}

	logLevel := viper.GetString("log.level")
	if logLevel == "" {
		logLevel = "INFO"
	}
	opts = append(opts, log.WithLevel(logLevel))

	logSyslog := viper.GetString("log.syslog")
	if logSyslog != "" {
		opts = append(opts, log.WithSyslog(logSyslog))
	}

	eventLog := viper.GetString("log.eventlog")
	if eventLog != "" {
		opts = append(opts, log.WithEventlog(eventLog))
	}

	logMailHost := viper.GetString("log.mail.host")
	if logMailHost != "" {
		addr := fmt.Sprintf("%s:%d", logMailHost, viper.GetInt("log.mail.port"))
		logMailFrom := viper.GetString("log.mail.from")
		logMailTo := viper.GetString("log.mail.to")
		logMailUsername := viper.GetString("log.mail.username")
		logMailPassword := viper.GetString("log.mail.password")
		opts = append(opts, log.WithMail("myOpenFactory Client", addr, logMailFrom, logMailTo, logMailUsername, logMailPassword))
	}

	logFolder := viper.GetString("log.folder")
	if logFolder != "" {
		opts = append(opts, log.WithFilesystem(logFolder))
	}

	return opts
}
