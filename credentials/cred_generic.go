//go:build !windows

package credentials

func NewDefaultCredManager() CredManager {
	return NewEnvCredManager()
}
