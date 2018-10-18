package syslog

import (
	"log/syslog"

	log "github.com/sirupsen/logrus"
	logrus_syslog "github.com/sirupsen/logrus/hooks/syslog"
)

func New(url string) (log.Hook, error) {
	return logrus_syslog.NewSyslogHook("udp", url, syslog.LOG_INFO, "")
}
