package credentials

type PasswordAuth struct {
	Username string
	Password string
}

type CredManager interface {
	GetCredential(name string) (*PasswordAuth, error)
}
