package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/pkg/errors"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func preUpdate() error {
	// Skip preUpdate if no service is registered
	if viper.GetString("service.name") == "" {
		return nil
	}

	serviceManager, err := mgr.Connect()
	if err != nil {
		return errors.Wrap(err, "service manager: connection failed")
	}
	defer serviceManager.Disconnect()

	services, err := serviceManager.ListServices()
	if err != nil {
		return errors.Wrap(err, "service manager: failed to list services")
	}

	for _, service := range services {
		if !strings.HasPrefix("myof-", service) {
			continue
		}
		if err := controlService(serviceManager, service, svc.Stop, svc.Stopped); err != nil {
			return err
		}
	}

	return nil
}

func postUpdate() error {
	// Skip postUpdate if no service is registered
	if viper.GetString("service.name") == "" {
		return nil
	}

	serviceManager, err := mgr.Connect()
	if err != nil {
		return errors.Wrap(err, "service manager: connection failed")
	}
	defer serviceManager.Disconnect()

	services, err := serviceManager.ListServices()
	if err != nil {
		return errors.Wrap(err, "service manager: failed to list services")
	}

	for _, service := range services {
		if !strings.HasPrefix("myof-", service) {
			continue
		}
		startService(serviceManager, service)
	}

	return nil
}

func startService(m *mgr.Mgr, name string) error {
	s, err := m.OpenService(name)
	if err != nil {
		return errors.Wrap(err, "could not access service")
	}

	defer s.Close()
	err = s.Start("is", "manual-started")
	if err != nil {
		return errors.Wrap(err, "could not start service")
	}

	return nil
}

func controlService(m *mgr.Mgr, name string, c svc.Cmd, to svc.State) error {
	s, err := m.OpenService(name)
	if err != nil {
		return errors.Wrap(err, "could not access service")
	}
	defer s.Close()

	status, err := s.Control(c)
	if err != nil {
		return errors.Wrapf(err, "could not send control=%d", c)
	}

	timeout := time.Now().Add(10 * time.Second)
	for status.State != to {
		if timeout.Before(time.Now()) {
			return fmt.Errorf("timeout waiting for service to go to state=%d", to)
		}
		time.Sleep(300 * time.Millisecond)
		status, err = s.Query()
		if err != nil {
			return errors.Wrap(err, "could not retrieve service status")
		}
	}

	return nil
}
