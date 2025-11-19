package credentials

import (
	"fmt"

	"github.com/danieljoos/wincred"
)

type windowsCredManager struct {
	serviceName string
}

func NewCredManager(serviceName string) *windowsCredManager {
	return &windowsCredManager{serviceName: serviceName}
}

func (m *windowsCredManager) CreateCredential(name, username, password string) error {
	cred := wincred.NewGenericCredential(m.generateWindowsCredName(name))
	cred.UserName = username
	cred.CredentialBlob = []byte(password)
	err := cred.Write()
	if err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}
	return nil
}

func (m *windowsCredManager) GetCredential(name string) (*PasswordAuth, error) {
	credential, err := wincred.GetGenericCredential(m.generateWindowsCredName(name))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve credential: %w", err)
	}
	return &PasswordAuth{
		Username: credential.UserName,
		Password: string(credential.CredentialBlob),
	}, nil
}

func (m *windowsCredManager) generateWindowsCredName(name string) string {
	credName := m.serviceName
	if name != "" {
		credName = m.serviceName + "/" + name
	}
	return credName
}
