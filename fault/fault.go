// Package fault defines transport-neutral, client-safe application errors.
package fault

import (
	"context"
	"errors"
	"fmt"
)

// Code classifies an error independently of HTTP, Thrift, or any other
// transport protocol.
type Code string

const (
	CodeCancelled          Code = "cancelled"
	CodeInvalidArgument    Code = "invalid_argument"
	CodeUnauthenticated    Code = "unauthenticated"
	CodePermissionDenied   Code = "permission_denied"
	CodeNotFound           Code = "not_found"
	CodeConflict           Code = "conflict"
	CodeFailedPrecondition Code = "failed_precondition"
	CodeResourceExhausted  Code = "resource_exhausted"
	CodeDeadlineExceeded   Code = "deadline_exceeded"
	CodeUnavailable        Code = "unavailable"
	CodeInternal           Code = "internal"
)

var defaultMessages = map[Code]string{
	CodeCancelled:          "request cancelled",
	CodeInvalidArgument:    "invalid request",
	CodeUnauthenticated:    "authentication required",
	CodePermissionDenied:   "permission denied",
	CodeNotFound:           "resource not found",
	CodeConflict:           "resource conflict",
	CodeFailedPrecondition: "failed precondition",
	CodeResourceExhausted:  "resource exhausted",
	CodeDeadlineExceeded:   "deadline exceeded",
	CodeUnavailable:        "service unavailable",
	CodeInternal:           "internal error",
}

// Error is an application error with a message that is safe to return to a
// client. The wrapped cause is retained for logs and errors.Is/errors.As, but
// must not be serialized to an external response.
type Error struct {
	code        Code
	safeMessage string
	cause       error
}

// New creates a client-safe error.
func New(code Code, safeMessage string) error {
	return newError(code, safeMessage, nil)
}

// Wrap creates a client-safe error while retaining the internal cause.
func Wrap(code Code, safeMessage string, cause error) error {
	return newError(code, safeMessage, cause)
}

func newError(code Code, safeMessage string, cause error) *Error {
	if code == "" {
		code = CodeInternal
	}
	if safeMessage == "" {
		safeMessage = DefaultMessage(code)
	}
	return &Error{code: code, safeMessage: safeMessage, cause: cause}
}

// Error implements error. It includes the cause for internal diagnostics;
// external transports must use SafeMessage instead.
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.cause == nil {
		return fmt.Sprintf("%s: %s", e.code, e.safeMessage)
	}
	return fmt.Sprintf("%s: %s: %v", e.code, e.safeMessage, e.cause)
}

// Unwrap exposes the internal cause to errors.Is and errors.As.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// Code returns the transport-neutral error code.
func (e *Error) Code() Code {
	if e == nil || e.code == "" {
		return CodeInternal
	}
	return e.code
}

// SafeMessage returns the client-safe message stored on this error.
func (e *Error) SafeMessage() string {
	if e == nil || e.safeMessage == "" {
		return DefaultMessage(CodeInternal)
	}
	return e.safeMessage
}

// CodeOf extracts an application code. Unclassified errors are internal.
func CodeOf(err error) Code {
	if err == nil {
		return ""
	}

	var target *Error
	if errors.As(err, &target) {
		return target.Code()
	}
	if errors.Is(err, context.Canceled) {
		return CodeCancelled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return CodeDeadlineExceeded
	}
	return CodeInternal
}

// SafeMessage returns a message suitable for an external response. It never
// exposes the text of an unclassified error.
func SafeMessage(err error) string {
	if err == nil {
		return ""
	}

	var target *Error
	if errors.As(err, &target) {
		return target.SafeMessage()
	}
	return DefaultMessage(CodeOf(err))
}

// IsCode reports whether err has code.
func IsCode(err error, code Code) bool {
	return err != nil && CodeOf(err) == code
}

// DefaultMessage returns the generic client-safe message for a code.
func DefaultMessage(code Code) string {
	if message, ok := defaultMessages[code]; ok {
		return message
	}
	return defaultMessages[CodeInternal]
}
