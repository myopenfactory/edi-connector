//go:build windows

package credentials

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/danieljoos/wincred"
)

func newTestManager(t *testing.T) *dpapiCredManager {
	t.Helper()
	return &dpapiCredManager{
		serviceName: serviceName,
		storeDir:    t.TempDir(),
	}
}

func uniqueCredName(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("test-%d", time.Now().UnixNano())
}

func writeLegacyCredential(t *testing.T, name, username, password string) {
	t.Helper()
	target := NewLegacyCredManager().legacyName(name)
	if cred, err := wincred.GetGenericCredential(target); err == nil {
		_ = cred.Delete()
	}
	cred := wincred.NewGenericCredential(target)
	cred.UserName = username
	cred.CredentialBlob = []byte(password)
	if err := cred.Write(); err != nil {
		t.Fatalf("failed to write legacy credential %q: %v", target, err)
	}
	t.Cleanup(func() {
		if cred, err := wincred.GetGenericCredential(target); err == nil {
			_ = cred.Delete()
		}
	})
}

func writeStoredFile(t *testing.T, m *dpapiCredManager, fileName string, stored storedCredential) {
	t.Helper()
	data, err := json.Marshal(stored)
	if err != nil {
		t.Fatalf("failed to marshal stored credential: %v", err)
	}
	protected, err := dpapiProtect(data)
	if err != nil {
		t.Fatalf("failed to protect data: %v", err)
	}
	if err := os.WriteFile(filepath.Join(m.storeDir, fileName), protected, 0o600); err != nil {
		t.Fatalf("failed to write stored credential file: %v", err)
	}
}

func TestDpapiRoundtrip(t *testing.T) {
	data := []byte(`{"username":"user","password":"secret"}`)
	protected, err := dpapiProtect(data)
	if err != nil {
		t.Fatalf("failed to protect data: %v", err)
	}
	if string(protected) == string(data) {
		t.Fatal("protected data is identical to the plain data")
	}
	unprotected, err := dpapiUnprotect(protected)
	if err != nil {
		t.Fatalf("failed to unprotect data: %v", err)
	}
	if string(unprotected) != string(data) {
		t.Fatalf("roundtrip mismatch: got %q want %q", unprotected, data)
	}
}

func TestDpapiProtectEmpty(t *testing.T) {
	if _, err := dpapiProtect(nil); err == nil {
		t.Fatal("expected an error for empty data")
	}
}

func TestDpapiUnprotectEmpty(t *testing.T) {
	if _, err := dpapiUnprotect(nil); err == nil {
		t.Fatal("expected an error for empty data")
	}
}

func TestStoredCredentialRoundtrip(t *testing.T) {
	m := newTestManager(t)
	base := uniqueCredName(t)
	for _, name := range []string{base + "/default", base + "/auth", base + "/a/b:c*"} {
		auth := &PasswordAuth{Username: "user", Password: "password"}
		if err := m.writeStored(name, auth); err != nil {
			t.Fatalf("failed to write credential %q: %v", name, err)
		}
		got, err := m.readStored(name)
		if err != nil {
			t.Fatalf("failed to read credential %q: %v", name, err)
		}
		if got.Username != auth.Username || got.Password != auth.Password {
			t.Fatalf("credential %q mismatch: got %+v want %+v", name, got, auth)
		}
	}
}

func TestGetCredentialMissing(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.GetCredential(uniqueCredName(t)); err == nil {
		t.Fatal("expected an error for a missing credential")
	}
}

func TestGetCredentialMigratesFromLegacy(t *testing.T) {
	m := newTestManager(t)
	name := uniqueCredName(t)
	writeLegacyCredential(t, name, "user", "secret")

	auth, err := m.GetCredential(name)
	if err != nil {
		t.Fatalf("failed to get credential: %v", err)
	}
	if auth.Username != "user" || auth.Password != "secret" {
		t.Fatalf("unexpected credential: %+v", auth)
	}
	if _, err := os.Stat(m.credFile(name)); err != nil {
		t.Fatalf("expected the credential file to be created: %v", err)
	}
	if _, err := wincred.GetGenericCredential(NewLegacyCredManager().legacyName(name)); err == nil {
		t.Fatal("expected the legacy credential to be deleted")
	}
}

func TestSetCredential(t *testing.T) {
	name := uniqueCredName(t)
	if err := SetCredential(name, "user", "password"); err != nil {
		t.Fatalf("failed to set credential: %v", err)
	}
	storeDir, err := dpapiStoreDir()
	if err != nil {
		t.Skipf("dpapi store directory not available: %v", err)
	}
	m := &dpapiCredManager{serviceName: serviceName, storeDir: storeDir}
	t.Cleanup(func() { _ = os.Remove(m.credFile(name)) })
	auth, err := m.readStored(name)
	if err != nil {
		t.Fatalf("failed to read credential: %v", err)
	}
	if auth.Username != "user" || auth.Password != "password" {
		t.Fatalf("unexpected credential: %+v", auth)
	}
}

func TestListStored(t *testing.T) {
	m := newTestManager(t)
	writeStoredFile(t, m, "default.cred", storedCredential{Username: "default-user", Password: "p"})
	writeStoredFile(t, m, "auth.cred", storedCredential{Name: "auth", Username: "auth-user", Password: "p"})
	writeStoredFile(t, m, "other_auth_1.cred", storedCredential{Name: "other/auth:1", Username: "other-user", Password: "p"})
	writeStoredFile(t, m, "ignored.txt", storedCredential{Username: "ignored", Password: "p"})
	if err := os.MkdirAll(filepath.Join(m.storeDir, "sub.cred"), 0o700); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	creds, err := m.listStored()
	if err != nil {
		t.Fatalf("failed to list credentials: %v", err)
	}
	if len(creds) != 3 {
		t.Fatalf("expected 3 credentials, got %d: %+v", len(creds), creds)
	}
	byName := make(map[string]CredentialInfo, len(creds))
	for _, c := range creds {
		byName[c.Name] = c
	}
	cases := []struct {
		name     string
		username string
	}{
		{"(default)", "default-user"},
		{"auth", "auth-user"},
		{"other/auth:1", "other-user"},
	}
	for _, c := range cases {
		cred, ok := byName[c.name]
		if !ok {
			t.Fatalf("credential %q not listed", c.name)
		}
		if cred.Username != c.username {
			t.Fatalf("unexpected credential for %q: %+v", c.name, cred)
		}
	}
}

func TestListStoredMissingDir(t *testing.T) {
	m := &dpapiCredManager{serviceName: serviceName, storeDir: filepath.Join(t.TempDir(), "missing")}
	creds, err := m.listStored()
	if err != nil {
		t.Fatalf("failed to list credentials: %v", err)
	}
	if len(creds) != 0 {
		t.Fatalf("expected no credentials, got %+v", creds)
	}
}

func TestCredFileNaming(t *testing.T) {
	m := &dpapiCredManager{serviceName: serviceName, storeDir: "store"}
	cases := []struct {
		name string
		want string
	}{
		{"", filepath.Join("store", "default.cred")},
		{"auth", filepath.Join("store", "auth.cred")},
		{"a/b:c*", filepath.Join("store", "a_b_c_.cred")},
		{"***", filepath.Join("store", "unnamed.cred")},
	}
	for _, c := range cases {
		if got := m.credFile(c.name); got != c.want {
			t.Fatalf("credFile(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestLegacyCredName(t *testing.T) {
	cases := []struct {
		target string
		name   string
		ok     bool
	}{
		{"EDI-Connector", "", true},
		{"EDI-Connector/auth", "auth", true},
		{"EDI-Connector/a/b", "a/b", true},
		{"EDI-ConnectorX", "", false},
		{"other", "", false},
	}
	for _, c := range cases {
		name, ok := legacyCredName(serviceName, c.target)
		if name != c.name || ok != c.ok {
			t.Fatalf("legacyCredName(%q) = (%q, %v), want (%q, %v)", c.target, name, ok, c.name, c.ok)
		}
	}
}
