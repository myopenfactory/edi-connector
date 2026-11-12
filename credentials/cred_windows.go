package credentials

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/danieljoos/wincred"
)

const serviceName = "EDI-Connector"

const invalidFileNameChars = `<>:"/\|?*`

// CredentialInfo describes a stored credential for display purposes.
type CredentialInfo struct {
	Name     string
	Username string
}

// storedCredential is the format of the data encrypted with DPAPI.
type storedCredential struct {
	Name     string `json:"name,omitempty"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// dpapiCredManager stores credentials in the machine wide program data
// directory, encrypted with the Windows Data Protection API (DPAPI). The
// directory is shared between the installer (administrator) and the service
// (LocalSystem), access is restricted to those.
type dpapiCredManager struct {
	serviceName string
	storeDir    string
}

// NewDefaultCredManager returns the credential manager used on Windows.
// Credentials stored in the Windows credential manager are migrated to the
// DPAPI based store on startup.
func NewDefaultCredManager() CredManager {
	storeDir, err := dpapiStoreDir()
	if err != nil {
		return NewLegacyCredManager()
	}
	m := &dpapiCredManager{
		serviceName: serviceName,
		storeDir:    storeDir,
	}
	m.migrateFromLegacy()
	return m
}

// ListCredentials returns all stored credentials, from the DPAPI store and
// from the legacy Windows credential manager.
func ListCredentials() ([]CredentialInfo, error) {
	creds := make([]CredentialInfo, 0)
	if storeDir, err := dpapiStoreDir(); err == nil {
		m := &dpapiCredManager{serviceName: serviceName, storeDir: storeDir}
		m.migrateFromLegacy()
		stored, err := m.listStored()
		if err != nil {
			return nil, err
		}
		creds = append(creds, stored...)
	}
	legacy, err := listLegacyCredentials(serviceName)
	if err != nil {
		return nil, err
	}
	creds = append(creds, legacy...)
	sort.Slice(creds, func(i, j int) bool { return creds[i].Name < creds[j].Name })
	return creds, nil
}

// SetCredential creates or updates a credential in the DPAPI store.
func SetCredential(name, username, password string) error {
	storeDir, err := dpapiStoreDir()
	if err != nil {
		return err
	}
	m := &dpapiCredManager{serviceName: serviceName, storeDir: storeDir}
	return m.writeStored(name, &PasswordAuth{
		Username: username,
		Password: password,
	})
}

// DeleteCredential removes a credential from the DPAPI store. A credential
// that does not exist is not an error.
func DeleteCredential(name string) error {
	storeDir, err := dpapiStoreDir()
	if err != nil {
		return err
	}
	m := &dpapiCredManager{serviceName: serviceName, storeDir: storeDir}
	if err := os.Remove(m.credFile(name)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete credential %q: %w", name, err)
	}
	return nil
}

func dpapiStoreDir() (string, error) {
	programData := os.Getenv("ProgramData")
	if programData == "" {
		return "", fmt.Errorf("environment variable ProgramData is not set")
	}
	return filepath.Join(programData, "myOpenFactory Software GmbH", "EDI-Connector", "credentials"), nil
}

func (m *dpapiCredManager) GetCredential(name string) (*PasswordAuth, error) {
	auth, err := m.readStored(name)
	if err == nil {
		return auth, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	// The credential is not (yet) stored with DPAPI. Fall back to the legacy
	// Windows credential manager and migrate it on the fly.
	legacy := NewLegacyCredManager()
	auth, err = legacy.GetCredential(name)
	if err != nil {
		return nil, err
	}
	if err := m.createStored(name, auth); err != nil && !errors.Is(err, fs.ErrExist) {
		return nil, err
	}
	return auth, nil
}

func (m *dpapiCredManager) migrateFromLegacy() {
	creds, err := wincred.FilteredList(m.serviceName + "*")
	if err != nil {
		return
	}
	for _, c := range creds {
		name, ok := legacyCredName(m.serviceName, c.TargetName)
		if !ok {
			continue
		}
		auth := &PasswordAuth{
			Username: c.UserName,
			Password: string(c.CredentialBlob),
		}
		// An existing DPAPI credential is authoritative and never overwritten.
		// The legacy credential is kept if it cannot be written to the DPAPI
		// store, it will be retried on the next startup.
		_ = m.createStored(name, auth)
	}
}

func legacyCredName(serviceName, targetName string) (string, bool) {
	if targetName == serviceName {
		return "", true
	}
	name, found := strings.CutPrefix(targetName, serviceName+"/")
	if !found {
		return "", false
	}
	return name, true
}

func listLegacyCredentials(serviceName string) ([]CredentialInfo, error) {
	creds, err := wincred.FilteredList(serviceName + "*")
	if err != nil {
		return nil, fmt.Errorf("failed to list credentials from the credential manager: %w", err)
	}
	infos := make([]CredentialInfo, 0, len(creds))
	for _, c := range creds {
		name, ok := legacyCredName(serviceName, c.TargetName)
		if !ok {
			continue
		}
		if name == "" {
			name = "(default)"
		}
		infos = append(infos, CredentialInfo{
			Name:     name,
			Username: c.UserName,
		})
	}
	return infos, nil
}

func (m *dpapiCredManager) listStored() ([]CredentialInfo, error) {
	entries, err := os.ReadDir(m.storeDir)
	if err != nil {
		if os.IsNotExist(err) {
			entries = nil
		} else {
			return nil, fmt.Errorf("failed to read credential store directory: %w", err)
		}
	}
	creds := make([]CredentialInfo, 0, len(entries))
	for _, entry := range entries {
		fileName := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(fileName, ".cred") {
			continue
		}
		name := strings.TrimSuffix(fileName, ".cred")
		if name == "default" {
			name = "(default)"
		}
		data, err := os.ReadFile(filepath.Join(m.storeDir, fileName))
		if err != nil {
			return nil, fmt.Errorf("failed to read credential file %q: %w", fileName, err)
		}
		plain, err := dpapiUnprotect(data)
		if err != nil {
			creds = append(creds, CredentialInfo{Name: name})
			continue
		}
		var stored storedCredential
		if err := json.Unmarshal(plain, &stored); err != nil {
			return nil, fmt.Errorf("failed to parse credential file %q: %w", fileName, err)
		}
		if stored.Name != "" {
			name = stored.Name
		}
		creds = append(creds, CredentialInfo{
			Name:     name,
			Username: stored.Username,
		})
	}
	sort.Slice(creds, func(i, j int) bool { return creds[i].Name < creds[j].Name })
	return creds, nil
}

func (m *dpapiCredManager) readStored(name string) (*PasswordAuth, error) {
	data, err := os.ReadFile(m.credFile(name))
	if err != nil {
		return nil, fmt.Errorf("failed to read credential %q: %w", name, err)
	}
	plain, err := dpapiUnprotect(data)
	if err != nil {
		return nil, err
	}
	var stored storedCredential
	if err := json.Unmarshal(plain, &stored); err != nil {
		return nil, fmt.Errorf("failed to parse stored credential %q: %w", name, err)
	}
	return &PasswordAuth{
		Username: stored.Username,
		Password: stored.Password,
	}, nil
}

// writeStored creates or replaces the credential.
func (m *dpapiCredManager) writeStored(name string, auth *PasswordAuth) error {
	return m.store(name, auth, true)
}

// createStored creates the credential and fails with fs.ErrExist if it
// already exists.
func (m *dpapiCredManager) createStored(name string, auth *PasswordAuth) error {
	return m.store(name, auth, false)
}

func (m *dpapiCredManager) store(name string, auth *PasswordAuth, overwrite bool) error {
	data, err := json.Marshal(storedCredential{
		Name:     name,
		Username: auth.Username,
		Password: auth.Password,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal credential %q: %w", name, err)
	}
	protected, err := dpapiProtect(data)
	if err != nil {
		return err
	}
	path := m.credFile(name)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create credential store directory: %w", err)
	}
	if err := restrictAccess(dir); err != nil {
		return err
	}

	// The credential is written to a temporary file first, so that readers
	// never see a partial file and a failed write keeps the old credential.
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to write credential %q: %w", name, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(protected); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to write credential %q: %w", name, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to write credential %q: %w", name, err)
	}
	if overwrite {
		err = os.Rename(tmpName, path)
	} else {
		// A hard link fails if the target exists, unlike a rename.
		err = os.Link(tmpName, path)
	}
	if err != nil {
		return fmt.Errorf("failed to write credential %q: %w", name, err)
	}
	// The credential is now stored with DPAPI, remove a possibly existing
	// legacy credential so it is not left behind in the credential manager.
	_ = NewLegacyCredManager().deleteCredential(name)
	return nil
}

// restrictAccess applies a protected ACL that only grants SYSTEM and
// Administrators access and makes Administrators the owner. Existing explicit
// entries on the directory and its files are reset first.
func restrictAccess(dir string) error {
	steps := [][]string{
		{dir, "/reset", "/T", "/C", "/Q"},
		{dir, "/setowner", "*S-1-5-32-544", "/T", "/C", "/Q"},
		{dir, "/inheritance:r", "/grant:r", "*S-1-5-18:(OI)(CI)F", "*S-1-5-32-544:(OI)(CI)F"},
	}
	for _, args := range steps {
		out, err := exec.Command("icacls", args...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to restrict access to credential store: %w: %s", err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

func (m *dpapiCredManager) credFile(name string) string {
	fileName := "default"
	if name != "" {
		fileName = sanitizeCredentialFileName(name)
	}
	return filepath.Join(m.storeDir, fileName+".cred")
}

// sanitizeCredentialFileName percent-encodes characters that are not allowed
// in file names, and the percent sign itself, so that different names never
// share a file. A trailing dot or space is encoded as well.
func sanitizeCredentialFileName(name string) string {
	var b strings.Builder
	for i := 0; i < len(name); i++ {
		c := name[i]
		last := i == len(name)-1
		if c < 0x20 || c == '%' || strings.IndexByte(invalidFileNameChars, c) >= 0 || (last && (c == '.' || c == ' ')) {
			fmt.Fprintf(&b, "%%%02X", c)
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}
