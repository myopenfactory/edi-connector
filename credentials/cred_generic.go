//go:build !windows

package credentials

import (
	"fmt"
	"os"
	"strings"
)

type envCredManager struct {
	serviceName string
}

func NewCredManager(serviceName string) *envCredManager {
	return &envCredManager{
		serviceName: strings.ToUpper(serviceName),
	}
}

func (m *envCredManager) CreateCredential(name, username, password string) error {
	return fmt.Errorf("not supported to store credentials")
}

func (m *envCredManager) GetCredential(name string) (*PasswordAuth, error) {
	envName := m.serviceName
	if name != "" {
		envName = m.serviceName + "_" + strings.ToUpper(name)
	}
	auth, ok := os.LookupEnv(envName)
	if !ok {
		return nil, fmt.Errorf("failed to load authentication environment variable")
	}

	authElements := strings.SplitN(auth, ":", 2)
	if len(authElements) != 2 {
		return nil, fmt.Errorf("invalid auth format: expected 'username:password'")
	}

	return &PasswordAuth{
		Username: authElements[0],
		Password: authElements[1],
	}, nil
}
