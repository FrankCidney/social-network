package apperror

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")       // e.g. email already registered
	ErrUnauthorized = errors.New("unauthorized")   // bad credentials or missing session
	ErrForbidden    = errors.New("forbidden")      // authenticated but not allowed
	ErrBadInput     = errors.New("bad input")      // validation failure
	ErrInternal     = errors.New("internal error") // unexpected failures
	ErrRateLimited  = errors.New("rate limited")
)

// AppError wraps a sentinel with a human-readable message to return to the frontend
type AppError struct {
	Err     error // sentinel errors to use in errors.Is checks to direct app flow
	Code    string
	Message string // human readable message to send back to client
}

func (e *AppError) Error() string {
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// Constructors - these create wrapped errors (sentinel wrapped inside AppError)

func NotFound(msg string) *AppError {
	return &AppError{Err: ErrNotFound, Code: "not_found", Message: msg}
}

func Conflict(msg string) *AppError {
	return &AppError{Err: ErrConflict, Code: "conflict", Message: msg}
}

func Unauthorized(msg string) *AppError {
	return &AppError{Err: ErrUnauthorized, Code: "unauthorized", Message: msg}
}

func Forbidden(msg string) *AppError {
	return &AppError{Err: ErrForbidden, Code: "forbidden", Message: msg}
}

func BadInput(msg string) *AppError {
	return &AppError{Err: ErrBadInput, Code: "bad_input", Message: msg}
}

func Internal(msg string) *AppError {
	return &AppError{Err: ErrInternal, Code: "internal", Message: msg}
}

func RateLimited(msg string) *AppError {
	return &AppError{Err: ErrRateLimited, Code: "rate_limited", Message: msg}
}
