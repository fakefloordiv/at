package http1

import "errors"

var (
	ErrBadRequest = errors.New("bad syntax")
	ErrTooLong    = errors.New("host value is too long")
	ErrNoHost     = errors.New("no host value is presented")
)
