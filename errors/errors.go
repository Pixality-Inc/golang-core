package errors

import (
	goErrors "errors"
)

var (
	Join   = goErrors.Join
	Unwrap = goErrors.Unwrap
	Is     = goErrors.Is
	As     = goErrors.As
)

type Code string

type Params = map[string]any

type Error interface {
	Code() Code
	Error() string
	Params() Params
	Unwrap() error
	Throw() error
}

type ImplError struct {
	code    Code
	message string
	params  Params
	cause   error
}

func New(code Code, message string, options ...Option) *ImplError {
	errorImpl := &ImplError{
		code:    code,
		message: message,
		params:  make(Params, 0),
		cause:   nil,
	}

	for _, option := range options {
		option.Apply(errorImpl)
	}

	return errorImpl
}

func (e *ImplError) Code() Code {
	return e.code
}

func (e *ImplError) Error() string {
	return e.message
}

func (e *ImplError) Params() Params {
	return e.params
}

func (e *ImplError) Unwrap() error {
	return e.cause
}

func (e *ImplError) Throw() error {
	return e
}
