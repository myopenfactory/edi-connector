package update

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func isAdmin() (bool, error) {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(&windows.SECURITY_NT_AUTHORITY, 2, windows.SECURITY_BUILTIN_DOMAIN_RID, windows.DOMAIN_ALIAS_RID_ADMINS, 0, 0, 0, 0, 0, 0, &sid)
	if err != nil {
		return false, err
	}
	token := windows.Token(0)
	return token.IsMember(sid)
}

func preUpdate(cmd *cobra.Command, args []string) error {
	admin, err := isAdmin()
	if err != nil {
		return err
	}
	if !admin {
		return errors.New("no admin right provided, can't stop service")
	}

	// Skip preUpdate if no service is registered
	if viper.GetString("service.name") == "" {
		return nil
	}

	serviceManager, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("service manager: connection failed: %w", err)
	}
	defer serviceManager.Disconnect()

	services, err := serviceManager.ListServices()
	if err != nil {
		return fmt.Errorf("service manager: failed to list services: %w", err)
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

func postUpdate(cmd *cobra.Command, args []string) error {
	serviceManager, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("service manager: connection failed: %w", err)
	}
	defer serviceManager.Disconnect()

	services, err := serviceManager.ListServices()
	if err != nil {
		return fmt.Errorf("service manager: failed to list services: %w", err)
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
		return fmt.Errorf("could not access service: %w", err)
	}

	defer s.Close()
	err = s.Start("is", "manual-started")
	if err != nil {
		return fmt.Errorf("could not start service: %w", err)
	}

	return nil
}

func controlService(m *mgr.Mgr, name string, c svc.Cmd, to svc.State) error {
	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("could not access service: %w", err)
	}
	defer s.Close()

	status, err := s.Control(c)
	if err != nil {
		return fmt.Errorf("could not send control=%d: %w", c, err)
	}

	timeout := time.Now().Add(10 * time.Second)
	for status.State != to {
		if timeout.Before(time.Now()) {
			return fmt.Errorf("timeout waiting for service to go to state=%d", to)
		}
		time.Sleep(300 * time.Millisecond)
		status, err = s.Query()
		if err != nil {
			return fmt.Errorf("could not retrieve service status: %w", err)
		}
	}

	return nil
}
