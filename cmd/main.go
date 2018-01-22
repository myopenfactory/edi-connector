package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"
	"flag"

	"myopenfactory.io/x/app/tatooine/pkg/client"
	"myopenfactory.io/x/app/tatooine/pkg/log"
	"myopenfactory.io/x/app/tatooine/pkg/config"
)

var (
	version string
	cfg string
)

func main() {
	flag.StringVar(&cfg, "config", "/myof/config/client.yml", "config file")
	flag.StringVar(&cfg, "c", "/myof/config/client.yml", "config file (shorthand)")
	flag.Parse()

	if err := config.Load(cfg); err != nil {
		log.Errorf("%v", err)
		os.Exit(1)
	}

	addr := config.LogSyslogAddress()
	if addr != "" {
		log.WithSyslog(addr)
	}

	addr = config.LogMailAddress()
	if addr != "" {
		log.WithMail("myOpenFactory Client", addr, config.LogMailSender(), config.LogMailReceiver(), config.LogMailUsername(), config.LogMailPassword())
	}

	file := config.LogFile()
	if file != "" {
		log.WithFile(file)
	}

	log.Infof("Stating myOpenFactory client %v", version)

	opts := []client.Option{
		client.WithUsername(config.Username()),
		client.WithPassword(config.Password()),
		client.WithURL(config.URL()),
		client.WithCert(config.ClientCertificate()),
		client.WithCA(config.CertificateAuthority()),
		client.WithProxy(config.Proxy()),
	}

	cl, err := client.New(fmt.Sprintf("Core_" + version), opts...)
	if err != nil {
		log.Errorf("error while creating client: %v", err)
		os.Exit(1)
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("recover client: %v", r)
			}
		}()
		log.Infof("started client")
		if err := cl.Run(); err != nil {
			log.Errorf("error while running client: %v", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop

	log.Infof("closing client")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cl.Shutdown(ctx)
	log.Infof("client gracefully stopped")
}
