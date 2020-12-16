package errors

import (
	"errors"
	"fmt"
	"runtime"
)

// Kind enums
const (
	KindUnknown = iota
	KindBadRequest
	KindUnexpected
	KindAlreadyExists
)

type Error struct {
	Kind int
	Op   Op
	Err  error
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
		case int:
			e.Kind = arg
		}
	}
	if e.Err == nil {
		e.Err = errors.New(KindText(e))
	}
	return e
}

func Kind(err error) int {
	var e Error
	if !errors.As(err, &e) {
		return KindUnexpected
	}

	return e.Kind
}

func KindText(err Error) string {
	switch Kind(err) {
	case KindUnexpected:
		return "Internal Client Error"
	default:
		return "Unknown Error"
	}
}

func Ops(err Error) []Op {
	ops := []Op{err.Op}
	for {
		var embeddedErr Error
		if !errors.As(err.Err, &embeddedErr) {
			break
		}

		ops = append(ops, embeddedErr.Op)
		err = embeddedErr
	}

	return ops
}
