package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/myopenfactory/edi-connector/v2/config"
	"github.com/myopenfactory/edi-connector/v2/connector"
	"github.com/myopenfactory/edi-connector/v2/log"
	"github.com/myopenfactory/edi-connector/v2/version"
)

func execute(configFile string, logLevel string) error {
	cfg, configFile, err := config.ReadConfig(configFile)
	if err != nil {
		return err
	}

	if logLevel != "" {
		cfg.Log.Level = logLevel
	}

	logger, err := log.NewFromConfig(cfg.Log)
	if err != nil {
		return err
	}

	logger.Info("client", "version", version.Version)
	logger.Info("Loaded config from", "path", configFile)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	signal.Notify(stop, os.Kill)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-stop

		logger.Info("closing client, please notice this could take up to one minute")
		cancel()
	}()

	defer func() {
		if err := recover(); err != nil {
			logger.With("error", err).Error("recovered a panic")
		}
	}()

	cl, err := connector.New(logger, cfg)
	if err != nil {
		return fmt.Errorf("failed to create edi-connector: %w", err)
	}

	if err := cl.Run(ctx); err != nil {
		return fmt.Errorf("failed to run edi-connector: %w", err)
	}

	return nil
}

func main() {
	configFile := flag.String("config", "", "Config file.")
	logLevel := flag.String("log_level", "", "Log level.")

	flag.Parse()

	if len(flag.Args()) > 0 {
		switch flag.Arg(0) {
		case "version":
			fmt.Printf("Version: %s\n", version.Version)
			fmt.Printf("Date: %s\n", version.Date)
			fmt.Printf("Commit: %s\n", version.Commit)
		default:
			fmt.Printf("Unknown parameter: %s\n", flag.Arg(0))
		}
		return
	}
	if isWindowsService() {
		err := serviceRun(*configFile, *logLevel)
		if err != nil {
			fmt.Println()
			os.Exit(1)
		}
	}
	if err := execute(*configFile, *logLevel); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
