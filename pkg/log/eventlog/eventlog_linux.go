package eventlog

import (
	log "github.com/sirupsen/logrus"
)

func New(name string) (log.Hook, error) {
	log.Warnf("linux does not support eventlog")
	return nil, nil
}
