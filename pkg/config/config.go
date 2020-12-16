package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
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
	var err error

	runWaitTimeDuration := time.Minute
	runWaitTime := viper.GetString("runwaittime")
	if runWaitTime != "" {
		runWaitTimeDuration, err = time.ParseDuration(runWaitTime)
		if err != nil {
			return nil, err
		}
	}

	healthWaitTimeDuration := 15 * time.Minute
	healthWaitTime := viper.GetString("healthwaitttime")
	if healthWaitTime != "" {
		healthWaitTimeDuration, err = time.ParseDuration(healthWaitTime)
		if err != nil {
			return nil, err
		}
	}

	url := viper.GetString("url")
	if url == "" {
		url = "https://myopenfactory.net"
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

	proxy := viper.GetString("proxy")
	if proxy == "" {
		proxy = os.Getenv("HTTP_PROXY")
	}

	cl, err := createHTTPClient(clientcert, cafile)
	if err != nil {
		return nil, errors.E(op, fmt.Errorf("could not create http client: %w", err), errors.KindUnexpected)
	}

	return []client.Option{client.WithUsername(viper.GetString("username")), client.WithPassword(viper.GetString("password")), client.WithURL(url), client.WithProxy(proxy), client.WithHealthWaitTime(healthWaitTimeDuration), client.WithRunWaitTime(runWaitTimeDuration), client.WithClient(cl)}, nil
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

func createHTTPClient(cert, ca string) (*http.Client, error) {
	const op errors.Op = "config.createHTTPClient"

	if cert == "" {
		return nil, errors.E(op, fmt.Errorf("error while loading client certificate: no client certificate specified"))
	}

	var config tls.Config
	crt, err := tls.LoadX509KeyPair(cert, cert)
	if err != nil {
		return nil, errors.E(op, fmt.Errorf("error loading client certificate: %v", err))
	}
	config.Certificates = []tls.Certificate{crt}

	if ca != "" {
		pemData, err := ioutil.ReadFile(ca)
		if err != nil {
			return nil, errors.E(op, fmt.Errorf("error while loading ca certificates: %w", err))
		}
		certs := x509.NewCertPool()
		certs.AppendCertsFromPEM(pemData)
		config.RootCAs = certs
		config.BuildNameToCertificate()
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &config,
			Proxy:           http.ProxyFromEnvironment,
		},
	}, nil
}
