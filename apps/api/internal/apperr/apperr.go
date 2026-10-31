package apperr

import (
	"errors"
	"net/http"
)

type Error struct {
	Status  int
	Code    string
	Message string
	Details any
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Code + ": " + e.Cause.Error()
	}
	return e.Code + ": " + e.Message
}

func (e *Error) Unwrap() error { return e.Cause }

func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func WithDetails(err *Error, details any) *Error {
	err.Details = details
	return err
}

func WithCause(err *Error, cause error) *Error {
	err.Cause = cause
	return err
}

func BadRequest(code, message string) *Error {
	return New(http.StatusBadRequest, code, message)
}

func Unauthorized(code, message string) *Error {
	return New(http.StatusUnauthorized, code, message)
}

func Forbidden(message string) *Error {
	return New(http.StatusForbidden, "forbidden", message)
}

func NotFound(message string) *Error {
	return New(http.StatusNotFound, "not_found", message)
}

func Conflict(code, message string) *Error {
	return New(http.StatusConflict, code, message)
}

func Gone(code, message string) *Error {
	return New(http.StatusGone, code, message)
}

func Unprocessable(code, message string) *Error {
	return New(http.StatusUnprocessableEntity, code, message)
}

func TooMany(message string) *Error {
	return New(http.StatusTooManyRequests, "rate_limited", message)
}

func Internal(cause error) *Error {
	return &Error{
		Status:  http.StatusInternalServerError,
		Code:    "internal_error",
		Message: "Ocurrió un error inesperado.",
		Cause:   cause,
	}
}

func From(err error) *Error {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}
	return Internal(err)
}
