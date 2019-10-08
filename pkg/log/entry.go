package log

import (
	"github.com/myopenfactory/client/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Entry interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})

	WithFields(fields map[string]interface{}) Entry

	SystemErr(err error)
}

type entry struct {
	*logrus.Entry
}

func (e *entry) WithFields(fields map[string]interface{}) Entry {
	ent := e.Entry.WithFields(fields)

	return &entry{ent}
}

func (e *entry) SystemErr(err error) {
	clientErr, ok := err.(errors.Error)
	if !ok {
		e.Error(err)
		return
	}

	ent := e.WithFields(errFields(clientErr))
	switch errors.Severity(err) {
	case logrus.WarnLevel:
		ent.Warnf("%v", err)
	case logrus.InfoLevel:
		ent.Infof("%v", err)
	case logrus.DebugLevel:
		ent.Debugf("%v", err)
	default:
		ent.Errorf("%v", err)
	}
}

func errFields(err errors.Error) logrus.Fields {
	f := logrus.Fields{}
	f["operation"] = err.Op
	f["kind"] = errors.KindText(err)
	f["opts"] = errors.Ops(err)

	return f
}
