package service

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

func Run(args []string) error {
	flagSet := flag.NewFlagSet("service", flag.ExitOnError)
	serviceName := flagSet.String("name", "EDI-Connector", "Name of the service.")
	flagSet.Parse(args)

	remainderArgs := flagSet.Args()
	if len(args) != 1 {
		return fmt.Errorf("missing service command")
	}

	switch remainderArgs[0] {
	case "install":
		return install(*serviceName)
	case "uninstall":
		return uninstall(*serviceName)
	case "start":
		return start(*serviceName)
	case "stop":
		return stop(*serviceName)
	case "restart":
		if err := stop(*serviceName); err != nil {
			return fmt.Errorf("failed to stop service: %w", err)
		}
		if err := start(*serviceName); err != nil {
			return fmt.Errorf("failed to start service: %w", err)
		}
		return nil
	default:
		return nil
	}
}

func install(serviceName string) error {
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
		ErrorControl: mgr.NoAction,
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

	return nil
}

func uninstall(serviceName string) error {
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

	return nil
}

func start(serviceName string) error {
	manager, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("could not connect to mgr: %v", err)
	}
	defer manager.Disconnect()

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
		return fmt.Errorf("service already running")
	}

	err = service.Start()
	if err != nil {
		return fmt.Errorf("could not start service: %v", err)
	}

	return nil
}

func stop(serviceName string) error {
	manager, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("could not connect to mgr: %v", err)
	}
	defer manager.Disconnect()

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
