package log

import (
	"github.com/sirupsen/logrus"
)

type Entry interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})

	WithFields(fields map[string]interface{}) Entry
}

type entry struct {
	*logrus.Entry
}

func (e *entry) WithFields(fields map[string]interface{}) Entry {
	ent := e.Entry.WithFields(fields)

	return &entry{ent}
}