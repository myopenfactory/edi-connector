package cmd

import (
	"github.com/myopenfactory/client/pkg/log"
	"fmt"
	"strings"
	"time"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
	"github.com/spf13/cobra"
)

// Do the interface allocations only once for common
// Errno values.
const (
	errnoERROR_IO_PENDING = 997
)

var (
	errERROR_IO_PENDING error = syscall.Errno(errnoERROR_IO_PENDING)
)

// errnoErr returns common boxed Errno values, to prevent
// allocations at runtime.
func errnoErr(e syscall.Errno) error {
	switch e {
	case 0:
		return nil
	case errnoERROR_IO_PENDING:
		return errERROR_IO_PENDING
	}

	// TODO: add more here, after collecting data on the common
	// error values see on Windows. (perhaps when running
	// all.bat?)
	return e
}

var (
	modadvapi32              = windows.NewLazySystemDLL("advapi32.dll")
	procCheckTokenMembership = modadvapi32.NewProc("CheckTokenMembership")
)

func isAdmin() (bool, error) {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(&windows.SECURITY_NT_AUTHORITY, 2, windows.SECURITY_BUILTIN_DOMAIN_RID, windows.DOMAIN_ALIAS_RID_ADMINS, 0, 0, 0, 0, 0, 0, &sid)
	if err != nil {
		return false, err
	}

	var b int32
	r1, _, e1 := syscall.Syscall(procCheckTokenMembership.Addr(), 3, uintptr(0), uintptr(unsafe.Pointer(sid)), uintptr(unsafe.Pointer(&b)))
	if r1 == 0 {
		if e1 != 0 {
			return false, errnoErr(e1)
		} else {
			return false, syscall.EINVAL
		}
	}
	return b != 0, nil
}

func preUpdate(cmd *cobra.Command, args[]string) error {
	admin, err := isAdmin()
	if err != nil {
		return err
	}
	if !admin {
		log.Infof("no admin rights provided, no service handling before update")
		return nil
	}

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

func postUpdate(cmd *cobra.Command, args[]string) error {
	admin, err := isAdmin()
	if err != nil {
		return err
	}
	if !admin {
		log.Infof("no admin rights provided, no service handling after update")
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
