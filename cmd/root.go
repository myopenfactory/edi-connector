package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/myopenfactory/client/pkg/client"
	"github.com/myopenfactory/client/pkg/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
}

var rootCmd = &cobra.Command{
	Use:   "myof-client",
	Short: "myof-client is a very simple message acces for the myopenfactory platform",
	Run: func(cmd *cobra.Command, args []string) {
		log.Infof("Starting myOpenFactory client %v", Version)

		opts := []client.Option{
			client.WithUsername(viper.GetString("username")),
			client.WithPassword(viper.GetString("password")),
			client.WithURL(viper.GetString("url")),
			client.WithCA(viper.GetString("cafile")),
			client.WithCert(viper.GetString("clientcert")),
		}
		os.Setenv("HTTP_PROXY", viper.GetString("proxy"))

		cl, err := client.New(fmt.Sprintf("Core_"+Version), opts...)
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
		log.Infof("client gracefully stopped")
	},
}

// Execute execute the application with cobra
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	viper.SetEnvPrefix("client")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		cfgPath := configPath()
		cfgFile = filepath.Join(cfgPath, "config.properties")
		if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
			file, err := ioutil.TempFile("", "config.*.properties")
			if err != nil {
				fmt.Println("failed to create temporary config", err)
				os.Exit(1)
			}
			defer os.Remove(file.Name())
			cfgFile = file.Name()
		}
		viper.SetConfigFile(cfgFile)
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("failed to read config:", err)
		os.Exit(1)
	}

	logLevel := viper.GetString("log.level")
	if logLevel != "" {
		log.WithLevel(logLevel)
	}

	logSyslog := viper.GetString("log.syslog")
	if logSyslog != "" {
		log.WithSyslog(logSyslog)
	}

	eventLog := viper.GetString("log.eventlog")
	if eventLog != "" {
		log.WithEventlog(eventLog)
	}

	logMailHost := viper.GetString("log.mail.host")
	if logMailHost != "" {
		addr := fmt.Sprintf("%s:%d", logMailHost, viper.GetInt("log.mail.port"))
		logMailFrom := viper.GetString("log.mail.from")
		logMailTo := viper.GetString("log.mail.to")
		logMailUsername := viper.GetString("log.mail.username")
		logMailPassword := viper.GetString("log.mail.password")
		log.WithMail("myOpenFactory Client", addr, logMailFrom, logMailTo, logMailUsername, logMailPassword)
	}

	logFolder := viper.GetString("log.folder")
	if logFolder != "" {
		log.WithFilesystem(logFolder)
	}

	log.Infof("Using config: %s", viper.ConfigFileUsed())
}

func installPath() string {
	switch {
	case runtime.GOOS == "windows":
		return filepath.Join(os.Getenv("ProgramFiles"), "myOpenFactory", "Client")
	case runtime.GOOS == "linux":
		return filepath.Join("opt", "myopenfactory", "client")
	default:
		return ""
	}
}

func configPath() string {
	switch {
	case runtime.GOOS == "windows":
		return filepath.Join(os.Getenv("ProgramData"), "myOpenFactory", "Client")
	case runtime.GOOS == "linux":
		return filepath.Join("etc", "myopenfactory", "client")
	default:
		return ""
	}
}
