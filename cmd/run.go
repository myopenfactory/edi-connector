package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"
	
	"github.com/spf13/cobra"
	"github.com/myopenfactory/client/pkg/client"
	"github.com/myopenfactory/client/pkg/log"
)

var (
	username string
	password string
	url string
	clientcert string
	cafile string
	proxy string

	logLevel string
	logFolder string
	logSyslog string

	logMailHost string
	logMailPort int
	logMailFrom string
	logMailTo string
	logMailUsername string
	logMailPassword string
)

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVarP(&username, "username", "u", "", "username")
	runCmd.Flags().StringVarP(&password, "password", "p", "", "password")
	runCmd.Flags().StringVar(&url, "url", "", "base url for the plattform")
	runCmd.Flags().StringVarP(&clientcert, "clientcert", "k", "", "client certificate (PEM)")
	runCmd.Flags().StringVarP(&cafile, "cafile", "c", "", "ca file (PEM)")
	runCmd.Flags().StringVar(&proxy, "proxy", "", "proxy url")

	runCmd.Flags().StringVar(&logLevel, "log.level", "INFO", "log level")
	runCmd.Flags().StringVar(&logFolder, "log.folder", "", "folder for log files")
	runCmd.Flags().StringVar(&logSyslog, "log.syslog", "", "syslog server address")

	runCmd.Flags().StringVar(&logMailHost, "log.mail.host", "", "mail server address")
	runCmd.Flags().IntVar(&logMailPort, "log.mail.port", 25, "mail server port")
	runCmd.Flags().StringVar(&logMailFrom, "log.mail.from", "", "sender email address")
	runCmd.Flags().StringVar(&logMailTo, "log.mail.to", "", "receiver email address")
	runCmd.Flags().StringVar(&logMailUsername, "log.mail.username", "", "mail server username")
	runCmd.Flags().StringVar(&logMailPassword, "log.mail.password", "", "mail server password")
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "runs the client",
	Run: runClient,
}

func runClient(cmd *cobra.Command, args []string) {
	if logSyslog != "" {
		log.WithSyslog(logSyslog)
	}

	addr := fmt.Sprintf("%s:%d", logMailHost, logMailPort)
	if addr != ":" {
		log.WithMail("myOpenFactory Client", addr, logMailFrom, logMailTo, logMailUsername, logMailUsername)
	}

	if logFolder != "" {
		log.WithFolder(logFolder)
	}

	log.Infof("Stating myOpenFactory client %v", version)

	opts := []client.Option{
		client.WithUsername(username),
		client.WithPassword(password),
		client.WithURL(url),
		client.WithCert(clientcert),
		client.WithCA(cafile),
		client.WithProxy(proxy),
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


