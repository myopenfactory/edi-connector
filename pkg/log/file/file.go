package file

import (
	"os"
	
	"github.com/sirupsen/logrus"
)

var defaultFormatter = &logrus.TextFormatter{DisableColors: true}

type FileHook struct {
	path string
	formatter logrus.Formatter
}

func New(path string) *FileHook {
	return &FileHook{
		path:  path,
		formatter: defaultFormatter,
	}
}

func (h *FileHook) Fire(entry *logrus.Entry) error {
	if entry == nil {
		return nil
	}

	fd, err := os.OpenFile(h.path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer fd.Close()

	msg, err := h.formatter.Format(entry)
	if err != nil {
		return err
	}

	fd.Write(msg)
	return nil
}

// Levels returns the available logging levels.
func (h *FileHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

