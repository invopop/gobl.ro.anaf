package converter

import (
	"errors"
	"fmt"
)

// Error is the standard error type of the converter.
type Error struct {
	key     string
	message string
	cause   error
}

var (
	// ErrInvalidDocument means the envelope does not carry a GOBL invoice.
	ErrInvalidDocument = NewError("invalid_document")

	// ErrNotCompliant means the invoice breaks a Romanian rule the addon
	// checks before anything is rendered.
	ErrNotCompliant = NewError("not_compliant")

	// ErrUnsupported means Romania does not accept a document of this kind.
	ErrUnsupported = NewError("unsupported")

	// ErrSkipped means the document is never reported to ANAF.
	ErrSkipped = NewError("skipped")

	// ErrConversion means the document could not be rendered.
	ErrConversion = NewError("conversion")

	// ErrUnreadable means a received document is not a UBL or CII invoice.
	ErrUnreadable = NewError("unreadable")
)

// NewError instantiates a new error with the given key.
func NewError(key string) *Error {
	return &Error{key: key}
}

func (e *Error) copy() *Error {
	ne := new(Error)
	*ne = *e
	return ne
}

// WithMsg returns a copy of the error carrying a message.
func (e *Error) WithMsg(message string, args ...any) *Error {
	ne := e.copy()
	ne.message = fmt.Sprintf(message, args...)
	return ne
}

// WithCause returns a copy of the error wrapping the given cause.
func (e *Error) WithCause(cause error) *Error {
	ne := e.copy()
	ne.cause = cause
	return ne
}

// Error provides the string representation of the error.
func (e *Error) Error() string {
	out := e.key
	if e.message != "" {
		out = e.message
	}
	if e.cause == nil {
		return out
	}
	return fmt.Sprintf("%s (%s)", out, e.cause.Error())
}

// Is reports whether the target error shares this error's key.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return errors.Is(e.cause, target)
	}
	return e.key == t.key
}

// Unwrap returns the error that caused this error.
func (e *Error) Unwrap() error {
	return e.cause
}
