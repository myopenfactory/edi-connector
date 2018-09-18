// +build windows

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

var (
	serviceName     string
	serviceUsername string
	servicePassword string
)

func init() {
	rootCmd.AddCommand(serviceCmd)

	serviceCmd.PersistentFlags().StringVar(&serviceName, "name", "myof-client", "name of the service")

	serviceCmd.AddCommand(serviceUninstallCmd)
	serviceCmd.AddCommand(serviceInstallCmd)

	serviceInstallCmd.Flags().StringVar(&serviceUsername, "logon", "", "windows logon name for the service")
	serviceInstallCmd.Flags().StringVar(&servicePassword, "password", "", "windows logon password for the service")
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
			DisplayName:  "myOpenFactory Client",
			Description:  "myOpenFactory Client to connect to the plattform",
			StartType:    mgr.StartAutomatic,
			ErrorControl: mgr.ServiceRestart,
		}
		if serviceUsername != "" {
			config.ServiceStartName = serviceUsername
			config.Password = servicePassword
		}
		s, err = m.CreateService(serviceName, exepath, config, "--config", cfgFile)
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
