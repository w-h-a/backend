package store

import "errors"

var (
	ErrNotFound = errors.New("not found")
	ErrAuthn    = errors.New("unauthenticated")
	ErrAuthz    = errors.New("unauthorized")
)
