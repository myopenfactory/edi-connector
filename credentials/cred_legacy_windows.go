package credentials

import (
	"fmt"

	"github.com/danieljoos/wincred"
)

// legacyCredManager reads credentials from the Windows credential manager.
// It is only used to migrate credentials that were stored before the switch
// to the DPAPI based credential store.
type legacyCredManager struct {
	serviceName string
}

func NewLegacyCredManager() *legacyCredManager {
	return &legacyCredManager{serviceName: serviceName}
}

// NewWindosCredManager is deprecated and only kept for backwards
// compatibility. NewDefaultCredManager stores credentials encrypted with
// DPAPI instead.
func NewWindosCredManager() *legacyCredManager {
	return NewLegacyCredManager()
}

func (m *legacyCredManager) GetCredential(name string) (*PasswordAuth, error) {
	credential, err := wincred.GetGenericCredential(m.legacyName(name))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve credential: %w", err)
	}
	return &PasswordAuth{
		Username: credential.UserName,
		Password: string(credential.CredentialBlob),
	}, nil
}

func (m *legacyCredManager) deleteCredential(name string) error {
	credential, err := wincred.GetGenericCredential(m.legacyName(name))
	if err != nil {
		return fmt.Errorf("failed to retrieve credential: %w", err)
	}
	if err := credential.Delete(); err != nil {
		return fmt.Errorf("failed to delete credential: %w", err)
	}
	return nil
}

func (m *legacyCredManager) legacyName(name string) string {
	credName := m.serviceName
	if name != "" {
		credName = m.serviceName + "/" + name
	}
	return credName
}
