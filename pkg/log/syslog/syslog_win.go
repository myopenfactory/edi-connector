// +build windows nacl plan9

package syslog

import (
	log "github.com/sirupsen/logrus"
)

func New(url string) (log.Hook, error) {
	log.Warnf("windows does not support syslog")
	return nil, nil
}
