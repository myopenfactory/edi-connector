package eventlog

import (
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/svc/eventlog"
)

type EventlogHook struct {
	name      string
	formatter logrus.Formatter
}

func New(name string) (*EventlogHook, error) {
	err := eventlog.InstallAsEventCreate(name, eventlog.Error|eventlog.Warning|eventlog.Info)
	defer func() {
		eventlog.Remove(name)
	}()
	if err != nil {
		return nil, err
	}

	l, err := eventlog.Open(name)
	if err != nil {
		return nil, err
	}
	defer l.Close()

	return &EventlogHook{
		name: name,
		formatter: &logrus.TextFormatter{
			DisableColors:    true,
			DisableTimestamp: true,
		},
	}, nil
}

func (h *EventlogHook) Fire(entry *logrus.Entry) error {
	logger, err := eventlog.Open(h.name)
	if err != nil {
		return err
	}
	defer logger.Close()

	const eventID = 1001
	f, err := h.formatter.Format(entry)
	if err != nil {
		return err
	}
	message := string(f)

	switch entry.Level {
	case logrus.PanicLevel:
		return logger.Error(eventID, message)
	case logrus.FatalLevel:
		return logger.Error(eventID, message)
	case logrus.ErrorLevel:
		return logger.Error(eventID, message)
	case logrus.WarnLevel:
		return logger.Warning(eventID, message)
	case logrus.InfoLevel:
		return logger.Info(eventID, message)
	case logrus.DebugLevel:
		return logger.Info(eventID, message)
	default:
		return nil
	}
}

// Levels returns the available logging levels.
func (h *EventlogHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}
