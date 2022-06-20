//go:build !windows

package eventlog

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

func New(name string) (log.Hook, error) {
	return nil, fmt.Errorf("linux does not support eventlog")
}
