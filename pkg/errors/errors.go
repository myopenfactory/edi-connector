package errors

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"

	"github.com/sirupsen/logrus"
)

// Kind enums
const (
	KindBadRequest    = http.StatusBadRequest
	KindUnexpected    = http.StatusInternalServerError
	KindAlreadyExists = http.StatusConflict
)

type Error struct {
	Kind     int
	Op       Op
	Err      error
	Severity logrus.Level
}

func (e Error) Error() string {
	return e.Err.Error()
}

func Is(err error, kind int) bool {
	if err == nil {
		return false
	}
	return Kind(err) == kind
}

type Op string

func (o Op) String() string {
	return string(o)
}

func E(op Op, args ...interface{}) error {
	e := Error{Op: op}
	if len(args) == 0 {
		msg := "errors.E called with 0 args"
		_, file, line, ok := runtime.Caller(1)
		if ok {
			msg = fmt.Sprintf("%v - %v:%v", msg, file, line)
		}
		e.Err = errors.New(msg)
	}
	for _, arg := range args {
		switch arg := arg.(type) {
		case error:
			e.Err = arg
		case string:
			e.Err = errors.New(arg)
		case logrus.Level:
			e.Severity = arg
		case int:
			e.Kind = arg
		}
	}
	if e.Err == nil {
		e.Err = errors.New(KindText(e))
	}
	return e
}

func Severity(err error) logrus.Level {
	e, ok := err.(Error)
	if !ok {
		return logrus.ErrorLevel
	}
	if e.Severity < logrus.ErrorLevel {
		return Severity(e.Err)
	}

	return e.Severity
}

func Expect(err error, kinds ...int) logrus.Level {
	for _, kind := range kinds {
		if Kind(err) == kind {
			return logrus.InfoLevel
		}
	}
	return logrus.ErrorLevel
}

func Kind(err error) int {
	e, ok := err.(Error)
	if !ok {
		return KindUnexpected
	}

	if e.Kind != 0 {
		return e.Kind
	}

	return Kind(e.Err)
}

func KindText(err Error) string {
	return http.StatusText(Kind(err))
}

func Ops(err Error) []Op {
	ops := []Op{err.Op}
	for {
		embeddedErr, ok := err.Err.(Error)
		if !ok {
			break
		}

		ops = append(ops, embeddedErr.Op)
		err = embeddedErr
	}

	return ops
}
