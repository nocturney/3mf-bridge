package prusa3mf

import (
	"errors"
	"fmt"
)

// Error is returned for every refused or failed conversion. Code is stable and
// machine-readable; the message names the offending source detail.
type Error struct {
	Code string
	msg  string
}

func (e *Error) Error() string { return e.msg }

func fail(code, format string, args ...any) error {
	return &Error{Code: code, msg: fmt.Sprintf(format, args...)}
}

// ErrorCode returns the stable code carried by err, or CONVERSION_FAILED for
// errors that did not originate in this package.
func ErrorCode(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return "CONVERSION_FAILED"
}
