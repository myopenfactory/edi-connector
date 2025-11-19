package credentials

type PasswordAuth struct {
	Username string
	Password string
}

type CredManager interface {
	CreateCredential(name, username, password string) error
	GetCredential(name string) (*PasswordAuth, error)
}
