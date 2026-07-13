// Package apperr defines shared application error codes for storage adapters.
package apperr

import (
	"errors"
	"fmt"
)

// Code classifies adapter/application failures.
type Code string

const (
	NotFound     Code = "not_found"
	Conflict     Code = "conflict"
	Validation   Code = "validation"
	Permission   Code = "permission"
	Unavailable  Code = "unavailable"
	Internal     Code = "internal"
)

// Error is a typed application error.
type Error struct {
	Code    Code
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Is reports whether err has the given code.
func Is(err error, code Code) bool {
	var e *Error
	if !errors.As(err, &e) {
		return false
	}
	return e.Code == code
}

func New(code Code, message string) error {
	return &Error{Code: code, Message: message}
}

func Wrap(code Code, message string, err error) error {
	return &Error{Code: code, Message: message, Err: err}
}
