package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/myopenfactory/edi-connector/log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

func init() {
	Service.PersistentFlags().String("name", "EDI-Connector", "name of the service")
	viper.BindPFlag("service.name", Service.PersistentFlags().Lookup("name"))

	Service.AddCommand(serviceInstallCmd)
	Service.AddCommand(serviceUninstallCmd)
	Service.AddCommand(serviceStartCmd)
	Service.AddCommand(serviceStopCmd)
	Service.AddCommand(serviceRestartCmd)
}

// serviceInstallCmd represents the install service command
var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "install as windows service",
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := viper.GetString("service.name")
		fmt.Println("Install as service:", serviceName)
		exepath, err := exePath()
		if err != nil {
			fmt.Println("could not get the exe path:", err)
			os.Exit(1)
		}

		m, err := mgr.Connect()
		if err != nil {
			fmt.Println("could not connect to mgr:", err)
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
			Description:  "myOpenFactory EDI-Connector to connect to the EDI platform.",
			StartType:    mgr.StartAutomatic,
			ErrorControl: mgr.ServiceRestart,
		}
		s, err = m.CreateService(serviceName, exepath, config)
		if err != nil {
			fmt.Println("could not create service:", err)
			os.Exit(1)
		}
		defer s.Close()

		err = eventlog.InstallAsEventCreate(serviceName, eventlog.Error|eventlog.Warning|eventlog.Info)
		if err != nil {
			s.Delete()
			fmt.Println("SetupEventLogSource() failed:", err)
			os.Exit(1)
		}
	},
}

// serviceUninstallCmd represents the uninstall service command
var serviceUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "uninstall the windows service",
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := viper.GetString("service.name")

		m, err := mgr.Connect()
		if err != nil {
			fmt.Println("could not connect to mgr:", err)
			os.Exit(1)
		}
		defer m.Disconnect()

		s, err := m.OpenService(serviceName)
		if err != nil {
			fmt.Println("service not installed:", err)
		} else {
			defer s.Close()
			fmt.Println("Uninstall service:", serviceName)
			if err = s.Delete(); err != nil {
				fmt.Println("could not delete server:", err)
				os.Exit(1)
			}
		}

		el, err := eventlog.Open(serviceName)
		if err != nil {
			fmt.Println("eventlog not installed:", err)
		} else {
			el.Close()
			if err = eventlog.Remove(serviceName); err != nil {
				fmt.Println("RemoveEventLogSource() failed:", err)
			}
		}
	},
}

// var serviceRunCmd = &cobra.Command{
// 	Use:   "run",
// 	Short: "run the windows service",
// 	Run: func(cmd *cobra.Command, args []string) {
// 		logger := log.New(config.ParseLogOptions()...)
// 		logger.Infof("Using config: %s", viper.ConfigFileUsed())

// 		clientOpts, err := config.ParseClientOptions()
// 		if err != nil {
// 			logger.Errorf("error while creating client: %v", err)
// 			os.Exit(1)
// 		}

// 		clientOpts = append(clientOpts, ediconnector.WithLogger(logger))

// 		cl, err := ediconnector.New(clientOpts...)
// 		if err != nil {
// 			logger.Errorf("error while creating client: %v", err)
// 			os.Exit(1)
// 		}

// 		run := svc.Run
// 		if viper.GetBool("service.debug") {
// 			run = debug.Run
// 		}

// 		ctx, cancel := context.WithCancel(context.Background())

// 		go func() {
// 			defer func() {
// 				if r := recover(); r != nil {
// 					logger.Errorf("recover client: %v", r)
// 					logger.Errorf("%s", rdbg.Stack())
// 				}
// 			}()
// 			if err := cl.Run(ctx); err != nil {
// 				logger.Errorf("error while running client: %v", err)
// 				os.Exit(1)
// 			}
// 		}()

// 		serviceName := viper.GetString("service.name")
// 		if err := run(serviceName, &service{client: cl, cancel: cancel}); err != nil {
// 			logger.Errorf("service failed: %q: %v", serviceName, err)
// 			return
// 		}
// 		logger.Infof("service stopped: %q", serviceName)
// 	},
// }

var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "start the windows service",
	Run: func(cmd *cobra.Command, args []string) {
		logger := log.New()

		if err := start(); err != nil {
			logger.Error("failed to start service", "error", err)
			return
		}
		logger.Info("service started")
	},
}

var serviceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "stop the windows service",
	Run: func(cmd *cobra.Command, args []string) {
		logger := log.New()

		if err := stop(); err != nil {
			logger.Error("failed to stop service", "error", err)
			return
		}
		logger.Info("service stopped")
	},
}

var serviceRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "restart the windows service",
	Run: func(cmd *cobra.Command, args []string) {
		logger := log.New()

		if err := stop(); err != nil {
			logger.Error("failed to stop service", "error", err)
			return
		}
		logger.Info("service stopped")

		if err := start(); err != nil {
			logger.Error("failed to start service", "error", err)
			return
		}
		logger.Info("service started")
	},
}

func start() error {
	manager, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("could not connect to mgr: %v", err)
	}
	defer manager.Disconnect()

	serviceName := viper.GetString("service.name")
	service, err := manager.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer service.Close()

	status, err := service.Query()
	if err != nil {
		return fmt.Errorf("could not retrieve service status: %v", err)
	}

	if status.State == svc.Running {
		return nil
	}

	err = service.Start()
	if err != nil {
		return fmt.Errorf("could not start service: %v", err)
	}

	return nil
}

func stop() error {
	manager, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("could not connect to mgr: %v", err)
	}
	defer manager.Disconnect()

	serviceName := viper.GetString("service.name")
	service, err := manager.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer service.Close()

	status, err := service.Query()
	if err != nil {
		return fmt.Errorf("could not retrieve service status: %v", err)
	}

	if status.State == svc.Stopped {
		return nil // already stopped
	}

	status, err = service.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("could not send stop: %v", err)
	}

	timeout := time.Now().Add(10 * time.Second)
	for status.State != svc.Stopped {
		if timeout.Before(time.Now()) {
			return fmt.Errorf("timeout waiting for service to stop")
		}
		time.Sleep(300 * time.Millisecond)
		status, err = service.Query()
		if err != nil {
			return fmt.Errorf("could not retrieve service status: %v", err)
		}
	}
	return nil
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
