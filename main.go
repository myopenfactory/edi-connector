package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/myopenfactory/edi-connector/config"
	"github.com/myopenfactory/edi-connector/ediconnector"
	serviceCmd "github.com/myopenfactory/edi-connector/internal/cmd/service"
	versionCmd "github.com/myopenfactory/edi-connector/internal/cmd/version"
	"github.com/myopenfactory/edi-connector/log"
	"github.com/myopenfactory/edi-connector/version"
)

var (
	serviceRun *func(context.Context, *slog.Logger, config.Config, string) error
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

	if serviceRun != nil {
		if err := (*serviceRun)(ctx, logger, cfg, cfg.ServiceName); err != nil {
			logger.With("error", err).Error("unable to run edi-connector")
			os.Exit(1)
		}
		return nil
	}

	cl, err := ediconnector.New(logger, cfg)
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

	if len(flag.Args()) > 1 {
		var err error
		switch os.Args[1] {
		case "version":
			err = versionCmd.Run()
		case "service":
			err = serviceCmd.Run(os.Args[2:])
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return
	}

	if err := execute(*configFile, *logLevel); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
