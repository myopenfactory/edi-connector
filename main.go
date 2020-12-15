package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/myopenfactory/client/pkg/client"
	"github.com/myopenfactory/client/pkg/cmd/bootstrap"
	"github.com/myopenfactory/client/pkg/cmd/config"
	"github.com/myopenfactory/client/pkg/cmd/service"
	"github.com/myopenfactory/client/pkg/cmd/update"
	"github.com/myopenfactory/client/pkg/cmd/version"
	configpkg "github.com/myopenfactory/client/pkg/config"
	"github.com/myopenfactory/client/pkg/errors"
	"github.com/myopenfactory/client/pkg/log"
	versionpkg "github.com/myopenfactory/client/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
)

func main() {
	var configFile string
	var logLevel string
	var logger *log.Logger

	cobra.OnInitialize(func() {
		viper.SetEnvPrefix("client")
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		viper.AutomaticEnv()

		switch {
		case runtime.GOOS == "windows":
			viper.AddConfigPath(filepath.Join(os.Getenv("ProgramData"), "myOpenFactory", "Client"))
		case runtime.GOOS == "linux":
			viper.AddConfigPath(filepath.Join("etc", "myopenfactory", "client"))
		}
		viper.AddConfigPath(".")

		if configFile != "" {
			viper.SetConfigFile(configFile)
		}

		if err := viper.ReadInConfig(); err != nil {
			err, ok := err.(viper.ConfigFileNotFoundError)
			if !ok {
				fmt.Printf("failed to read config: %s: %v\n", viper.ConfigFileUsed(), err)
				os.Exit(1)
			}
		}

		viper.Set("log.level", logLevel)
		if proxy := viper.GetString("proxy"); proxy != "" {
			os.Setenv("HTTP_PROXY", proxy)
		}
		logger = log.New(configpkg.ParseLogOptions())
	})

	cmds := &cobra.Command{
		Use:   "myof-client",
		Short: "myof-client controls the client",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			logger.Infof("myOpenFactory Client: %v", versionpkg.Version)
			if viper.ConfigFileUsed() == "" {
				logger.Debugf("Using config: no config found")
			} else {
				logger.Debugf("Using config: %v", viper.ConfigFileUsed())
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			const op errors.Op = "main.Run"

			clientOpts, err := configpkg.ParseClientOptions()
			if err != nil {
				logger.SystemErr(errors.E(op, err))
				os.Exit(1)
			}

			cl, err := client.New(logger, fmt.Sprintf("Core_%s", versionpkg.Version), clientOpts...)
			if err != nil {
				logger.SystemErr(errors.E(op, err))
				os.Exit(1)
			}

			stop := make(chan os.Signal, 1)
			signal.Notify(stop, os.Interrupt)
			signal.Notify(stop, os.Kill)

			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				<-stop

				logger.Infof("closing client, please notice this could take up to one minute")
				cancel()
			}()

			go func() {
				defer func() {
					if r := recover(); r != nil {
						logger.SystemErr(errors.E(op, err))
					}
				}()
				if err := cl.Health(ctx); err != nil {
					logger.SystemErr(errors.E(op, err))
					os.Exit(1)
				}
			}()

			defer func() {
				if r := recover(); r != nil {
					logger.SystemErr(errors.E(op, err))
				}
			}()
			if err := cl.Run(ctx); err != nil {
				logger.SystemErr(errors.E(op, err))
				os.Exit(1)
			}
			logger.Debug("client gracefully stopped")
		},
	}

	cmds.PersistentFlags().StringVar(&configFile, "config", "", "config file")
	cmds.PersistentFlags().StringVar(&logLevel, "log_level", "INFO", "log level")

	cmds.AddCommand(version.Command)
	cmds.AddCommand(config.Command)
	cmds.AddCommand(bootstrap.Command)
	cmds.AddCommand(update.Command)
	cmds.AddCommand(service.Command)

	if err := cmds.Execute(); err != nil {
		logger.Error(err)
		os.Exit(1)
	}
}
