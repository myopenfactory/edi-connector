package cmd

import (
	"github.com/myopenfactory/client/pkg/client"
	"fmt"
	"os"
	"path/filepath"
	"context"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"github.com/myopenfactory/client/pkg/log"
	"github.com/spf13/viper"
)

var (
	serviceName     string
	serviceUsername string
	servicePassword string
	serviceDebug    bool
)

func init() {
	rootCmd.AddCommand(serviceCmd)

	serviceCmd.PersistentFlags().StringVar(&serviceName, "name", "myOpenFactory Client", "name of the service")

	serviceCmd.AddCommand(serviceUninstallCmd)

	serviceInstallCmd.Flags().StringVar(&serviceUsername, "logon", "", "windows logon name for the service")
	serviceInstallCmd.Flags().StringVar(&servicePassword, "password", "", "windows logon password for the service")
	serviceCmd.AddCommand(serviceInstallCmd)

	serviceRunCmd.Flags().BoolVar(&serviceDebug, "debug", false, "debug windows service")
	serviceCmd.AddCommand(serviceRunCmd)
}

// serviceCmd represents the service command
var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "administrate windows service",
}

// serviceInstallCmd represents the install service command
var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "install as windows service",
	Run: func(cmd *cobra.Command, args []string) {
		if cfgFile == "" {
			fmt.Printf("config file is required")
			os.Exit(1)
		}
		fmt.Printf("Install as service: %s\n", serviceName)
		exepath, err := exePath()
		if err != nil {
			fmt.Printf("could not get the exe path: %v", err)
			os.Exit(1)
		}

		m, err := mgr.Connect()
		if err != nil {
			fmt.Printf("could not connect to mgr: %v", err)
			os.Exit(1)
		}
		defer m.Disconnect()

		s, err := m.OpenService(serviceName)
		if err == nil {
			s.Close()
			fmt.Printf("service %s already exists", serviceName)
			os.Exit(1)
		}
		config := mgr.Config{
			DisplayName:  serviceName,
			Description:  "myOpenFactory Client to connect to the plattform",
			StartType:    mgr.StartAutomatic,
			ErrorControl: mgr.ServiceRestart,
		}
		if serviceUsername != "" {
			config.ServiceStartName = serviceUsername
			config.Password = servicePassword
		}
		s, err = m.CreateService(serviceName, exepath, config, "service", "run", "--config", cfgFile, "--name", serviceName)
		if err != nil {
			fmt.Printf("could not create service: %v", err)
			os.Exit(1)
		}
		defer s.Close()

		err = eventlog.InstallAsEventCreate(serviceName, eventlog.Error|eventlog.Warning|eventlog.Info)
		if err != nil {
			s.Delete()
			fmt.Printf("SetupEventLogSource() failed: %s", err)
			os.Exit(1)
		}
	},
}

// serviceUninstallCmd represents the uninstall service command
var serviceUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "uninstall the windows service",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Uninstall service: %s", serviceName)
		m, err := mgr.Connect()
		if err != nil {
			fmt.Printf("could not connect to mgr: %v", err)
			os.Exit(1)
		}
		defer m.Disconnect()

		s, err := m.OpenService(serviceName)
		if err != nil {
			fmt.Printf("service %s is not installed", serviceName)
			os.Exit(1)
		}
		defer s.Close()

		err = s.Delete()
		if err != nil {
			fmt.Printf("could not delete server: %v", err)
			os.Exit(1)
		}

		err = eventlog.Remove(serviceName)
		if err != nil {
			fmt.Printf("RemoveEventLogSource() failed: %s", err)
			os.Exit(1)
		}
	},
}

var serviceRunCmd = &cobra.Command{
	Use: "run",
	Short: "run the windows service",
	Run: func(cmd *cobra.Command, args []string) {
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

		log.Infof("Starting myOpenFactory client %v", version)

		cafile := viper.GetString("cafile")
		clientcert := viper.GetString("clientcert")

		if _, err := os.Stat(cafile); os.IsNotExist(err) {
			cafile = filepath.Join(filepath.Dir(cfgFile), cafile)
		}

		if _, err := os.Stat(clientcert); os.IsNotExist(err) {
			clientcert = filepath.Join(filepath.Dir(cfgFile), clientcert)
		}

		opts := []client.Option{
			client.WithUsername(viper.GetString("username")),
			client.WithPassword(viper.GetString("password")),
			client.WithURL(viper.GetString("url")),
			client.WithCA(cafile),
			client.WithCert(clientcert),
		}
		os.Setenv("HTTP_PROXY", viper.GetString("proxy"))

		cl, err := client.New(fmt.Sprintf("Core_"+version), opts...)
		if err != nil {
			log.Errorf("error while creating client: %v", err)
			os.Exit(1)
		}

		if serviceDebug {
			elog = debug.New(serviceName)
		} else {
			elog, err = eventlog.Open(serviceName)
			if err != nil {
				log.Errorf("failed to open eventlog: %v", err)
				os.Exit(1)
			}
		}
		defer elog.Close()

		elog.Info(1, fmt.Sprintf("starting service: %q", serviceName))
		run := svc.Run
		if serviceDebug {
			run = debug.Run
		}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("recover client: %v", r)
				}
			}()
			if err := cl.Run(); err != nil {
				log.Errorf("error while running client: %v", err)
				os.Exit(1)
			}
		}()

		err = run(serviceName, &service{client: cl})
		if err != nil {
			elog.Error(1, fmt.Sprintf("service failed: %q: %v", serviceName, err))
			return
		}
		elog.Info(1, fmt.Sprintf("service stopped: %q", serviceName))
	},
}

func exePath() (string, error) {
	prog := os.Args[0]
	p, err := filepath.Abs(prog)
	if err != nil {
		return "", err
	}

	fi, err := os.Stat(p)
	if err == nil {
		if !fi.Mode().IsDir() {
			return p, nil
		}
		err = fmt.Errorf("%s is directory", p)
	}

	if filepath.Ext(p) == "" {
		p += ".exe"
		fi, err := os.Stat(p)
		if err == nil {
			if !fi.Mode().IsDir() {
				return p, nil
			}
			err = fmt.Errorf("%s is directory", p)
		}
	}

	return "", err

}

var elog debug.Log

type service struct{
	client *client.Client
}

func (m *service) Execute(args []string, r <- chan svc.ChangeRequest, changes chan <- svc.Status) (bool, uint32) {
	deadline := 5*time.Second
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted, WaitHint: uint32(deadline.Seconds()) * 1000}
	for {
		select {
		case c := <- r:
			switch c.Cmd {
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				ctx, cancel := context.WithTimeout(context.Background(), deadline)
				defer cancel()
				m.client.Shutdown(ctx)
				return false, 0
			}
		}
	}
}