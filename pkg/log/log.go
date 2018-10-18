package log

import (
	"github.com/sirupsen/logrus"
	"os"
	"fmt"

	"github.com/myopenfactory/client/pkg/log/filesystem"
	"github.com/myopenfactory/client/pkg/log/mail"
	"github.com/myopenfactory/client/pkg/log/syslog"
	"github.com/myopenfactory/client/pkg/log/eventlog"
)

type Logger struct {
	*logrus.Logger
}

var defaultLogger = New("INFO")

func New(level string) *Logger {
	l := logrus.New()
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		panic(err)
	}
	l.Level = logLevel

	return &Logger{Logger: l}
}

func (l *Logger) WithFields(fields map[string]interface{}) Entry {
	e := l.Logger.WithFields(fields)

	return &entry{e}
}

func Debugf(format string, args ...interface{}) {
	defaultLogger.Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	defaultLogger.Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	defaultLogger.Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	defaultLogger.Errorf(format, args...)
}

func WithFields(fields map[string]interface{}) Entry {
	return defaultLogger.WithFields(fields)
}

func WithLevel(level string) {
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		Errorf("failed parsing log level: %s", level)
		os.Exit(1)
	}
	defaultLogger.SetLevel(lvl)
}

func WithSyslog(address string) {
	hook, err := syslog.New(address)
	if err != nil {
		Errorf("failed to initialize syslog: %v", address)
		os.Exit(1)
	}
	defaultLogger.Logger.AddHook(hook)
}

func WithFilesystem(path string) {
	hook, err := filesystem.New(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defaultLogger.Logger.AddHook(hook)
}

func WithMail(appname, address, sender, receiver, username, password string) {
	hook := mail.New(appname, address, sender, receiver, username, password)
	defaultLogger.Logger.AddHook(hook)
}

func WithEventlog(name string) {
	hook, err := eventlog.New(name)
	if err != nil {
		Errorf("failed to initialize eventlog: %q: %v", name, err)
		os.Exit(1)
	}
	defaultLogger.Logger.AddHook(hook)
}
