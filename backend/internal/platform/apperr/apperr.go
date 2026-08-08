package apperr

import "net/http"

type Error struct {
	Code       string
	HTTPStatus int
	Message    string
	err        error
}

func (e *Error) Error() string {
	if e.err != nil {
		return e.Message + ": " + e.err.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.err }

func New(code string, status int, message string) *Error {
	return &Error{Code: code, HTTPStatus: status, Message: message}
}

func Invalid(message string) *Error      { return New("INVALID_ARGUMENT", http.StatusBadRequest, message) }
func Unauthorized(message string) *Error { return New("UNAUTHORIZED", http.StatusUnauthorized, message) }
func NotFound(message string) *Error     { return New("NOT_FOUND", http.StatusNotFound, message) }
func Conflict(code, msg string) *Error   { return New(code, http.StatusConflict, msg) }
func Internal() *Error                   { return New("INTERNAL", http.StatusInternalServerError, "internal error") }
