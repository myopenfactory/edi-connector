package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/myopenfactory/client/pkg/client"
	"github.com/myopenfactory/client/pkg/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringP("username", "u", "", "username")
	viper.BindPFlag("username", runCmd.Flags().Lookup("username"))

	runCmd.Flags().StringP("password", "p", "", "password")
	viper.BindPFlag("password", runCmd.Flags().Lookup("password"))

	runCmd.Flags().String("url", "", "base url for the plattform")
	viper.BindPFlag("url", runCmd.Flags().Lookup("url"))

	runCmd.Flags().StringP("clientcert", "k", "", "client certificate (PEM)")
	viper.BindPFlag("clientcert", runCmd.Flags().Lookup("clientcert"))

	runCmd.Flags().StringP("cafile", "c", "", "ca file (PEM)")
	viper.BindPFlag("cafile", runCmd.Flags().Lookup("cafile"))

	runCmd.Flags().String("proxy", "", "proxy url")
	viper.BindPFlag("proxy", runCmd.Flags().Lookup("proxy"))

	runCmd.Flags().String("log.level", "INFO", "log level")
	viper.BindPFlag("log.level", runCmd.Flags().Lookup("log.level"))

	runCmd.Flags().String("log.folder", "", "folder for log files")
	viper.BindPFlag("log.folder", runCmd.Flags().Lookup("log.folder"))

	runCmd.Flags().String("log.syslog", "", "syslog server address")
	viper.BindPFlag("log.syslog", runCmd.Flags().Lookup("log.syslog"))

	runCmd.Flags().String("log.mail.host", "", "mail server address")
	viper.BindPFlag("log.mail.host", runCmd.Flags().Lookup("log.mail.host"))

	runCmd.Flags().Int("log.mail.port", 25, "mail server port")
	viper.BindPFlag("log.mail.port", runCmd.Flags().Lookup("log.mail.port"))

	runCmd.Flags().String("log.mail.from", "", "sender email address")
	viper.BindPFlag("log.mail.from", runCmd.Flags().Lookup("log.mail.from"))

	runCmd.Flags().String("log.mail.to", "", "receiver email address")
	viper.BindPFlag("log.mail.to", runCmd.Flags().Lookup("log.mail.to"))

	runCmd.Flags().String("log.mail.username", "", "mail server username")
	viper.BindPFlag("log.mail.username", runCmd.Flags().Lookup("log.mail.username"))

	runCmd.Flags().String("log.mail.password", "", "mail server password")
	viper.BindPFlag("log.mail.password", runCmd.Flags().Lookup("log.mail.password"))
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "runs the client",
	Run:   runClient,
}

func runClient(cmd *cobra.Command, args []string) {
	logLevel := viper.GetString("log.level")
	if logLevel != "" {
		log.WithLevel(logLevel)
	}

	logSyslog := viper.GetString("log.syslog")
	if logSyslog != "" {
		log.WithSyslog(logSyslog)
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

	log.Infof("Stating myOpenFactory client %v", version)

	opts := []client.Option{
		client.WithUsername(viper.GetString("username")),
		client.WithPassword(viper.GetString("password")),
		client.WithURL(viper.GetString("url")),
		client.WithCert(viper.GetString("clientcert")),
		client.WithCA(viper.GetString("cafile")),
		client.WithProxy(viper.GetString("proxy")),
	}

	cl, err := client.New(fmt.Sprintf("Core_"+version), opts...)
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
