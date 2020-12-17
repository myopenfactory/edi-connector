package filesystem

import (
	"io"
	"path/filepath"
	"runtime"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/transform"
)

var defaultFormatter = &logrus.TextFormatter{DisableColors: true}

type crlfTransformer struct{}

func (crlfTransformer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	for nSrc < len(src) {
		if nDst >= len(dst) {
			err = transform.ErrShortDst
			break
		}
		c := src[nSrc]
		if c == '\n' {
			dst[nDst] = '\r'
			dst[nDst+1] = c
			nSrc++
			nDst += 2
		} else {
			dst[nDst] = c
			nSrc++
			nDst++
		}
	}
	return
}

func (crlfTransformer) Reset() {}

type FilesystemHook struct {
	path   string
	writer io.Writer
}

func New(path string) (*FilesystemHook, error) {
	logWriter, err := rotatelogs.New(
		filepath.Join(path, "log.%Y%m%d%H%M.log"),
		rotatelogs.WithMaxAge(24*time.Hour),
		rotatelogs.WithRotationTime(time.Hour),
	)

	if err != nil {
		return nil, err
	}

	var writer io.Writer = logWriter

	if runtime.GOOS == "windows" {
		writer = transform.NewWriter(logWriter, crlfTransformer{})
	}

	return &FilesystemHook{
		path:   path,
		writer: writer,
	}, nil
}

// Fire serializes and writes the log entry to file.
func (h *FilesystemHook) Fire(entry *logrus.Entry) error {
	data, err := defaultFormatter.Format(entry)

	if err != nil {
		return err
	}

	_, err = h.writer.Write(data)
	return err
}

// Levels returns the available logging levels.
func (h *FilesystemHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
