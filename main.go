package main

import (
	"context"
	"fmt"
	stdlog "log"
	"log/slog"
	"os"
	"os/signal"

	"github.com/myopenfactory/edi-connector/cmd"
	"github.com/myopenfactory/edi-connector/config"
	"github.com/myopenfactory/edi-connector/ediconnector"
	"github.com/myopenfactory/edi-connector/log"
	"github.com/myopenfactory/edi-connector/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	serviceRun *func(context.Context, *slog.Logger, config.Config) error
)

func main() {
	cmds := &cobra.Command{
		Use:   "edi-connector",
		Short: "runs the edi-connector",
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		Run: func(cmd *cobra.Command, args []string) {
			var cfg config.Config
			cfg, err := config.ReadConfig(cmd.Flag("config").Value.String())
			if err != nil {
				stdlog.Fatal(err)
			}

			logger, err := log.NewFromConfig(cfg.Log)
			if err != nil {
				stdlog.Fatal(err)
			}
			logger.Info("client", "version", version.Version)
			logger.Debug("Using config", "config", viper.ConfigFileUsed())

			ctx := context.WithValue(context.Background(), config.ConfigContextKey, cfg)
			ctx = context.WithValue(ctx, config.LoggerContextKey, logger)
			cmd.SetContext(ctx)

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
				if err := (*serviceRun)(ctx, logger, cfg); err != nil {
					logger.With("error", err).Error("unable to run edi-connector")
					os.Exit(1)
				}
				return
			}

			run := func() error {
				cl, err := ediconnector.New(logger, cfg)
				if err != nil {
					return fmt.Errorf("failed to create edi-connector: %w", err)
				}

				if err := cl.Run(ctx); err != nil {
					return fmt.Errorf("failed to run edi-connector: %w", err)
				}
				return nil
			}

			if err := run(); err != nil {
				logger.With("error", err).Error("unable to run edi-connector")
				os.Exit(1)
			}

		},
	}

	cmds.PersistentFlags().String("config", "", "config file")
	cmds.PersistentFlags().String("log_level", "INFO", "log level")
	viper.BindPFlag("log.level", cmds.PersistentFlags().Lookup("log_level"))

	cmds.AddCommand(cmd.Version)
	cmds.AddCommand(cmd.Service)

	if err := cmds.Execute(); err != nil {
		stdlog.Fatal(err)
		os.Exit(1)
	}
}
