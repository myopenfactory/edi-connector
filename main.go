package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	cmdpkg "github.com/myopenfactory/client/pkg/cmd"
	"github.com/myopenfactory/client/pkg/cmd/bootstrap"
	"github.com/myopenfactory/client/pkg/cmd/config"
	"github.com/myopenfactory/client/pkg/cmd/service"
	"github.com/myopenfactory/client/pkg/cmd/update"
	"github.com/myopenfactory/client/pkg/cmd/version"
	"github.com/myopenfactory/client/pkg/log"
	versionpkg "github.com/myopenfactory/client/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	var configFile string
	var logLevel string
	var log *log.Logger

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
		log = cmdpkg.InitializeLogger()
	})

	cmds := &cobra.Command{
		Use:   "myof-client",
		Short: "myof-client controls the client",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			log.Debugf("Using config: %v", viper.ConfigFileUsed())
		},
		Run: func(cmd *cobra.Command, args []string) {
			log.Infof("Starting myOpenFactory client %v", versionpkg.Version)

			cl, err := cmdpkg.InitializeClient()
			if err != nil {
				log.Errorf("error while creating client: %v", err)
				os.Exit(1)
			}

			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Errorf("recover client: %v", r)
						log.Errorf("%s", debug.Stack())
					}
				}()
				if err := cl.Run(); err != nil {
					log.Errorf("error while running client: %v", err)
					os.Exit(1)
				}
			}()

			stop := make(chan os.Signal, 1)
			signal.Notify(stop, os.Interrupt)
			signal.Notify(stop, os.Kill)

			<-stop

			log.Infof("closing client")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cl.Shutdown(ctx)
			log.Debug("client gracefully stopped")
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
		log.Error(err)
		os.Exit(1)
	}
}
