package http

import "errors"

var (
	ErrBadRequest          = errors.New("bad request")
	ErrNotFound            = errors.New("not found")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrForbidden           = errors.New("forbidden")
	ErrInternalServerError = errors.New("internal server error")
)
