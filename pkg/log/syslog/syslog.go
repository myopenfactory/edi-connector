//go:build !linux && !windows

package syslog

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

func New(url string) (log.Hook, error) {
	return nil, fmt.Errorf("not implemented")
}
