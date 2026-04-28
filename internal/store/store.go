package store

import (
	"errors"
)

var (
	// ErrNotFound is returned by a Find function when the requested
	// item is not found.
	ErrNotFound = errors.New("not found")

	// ErrAlreadyExists is returned by a create function when a
	// constraint is violated.
	ErrAlreadyExists = errors.New("item already exists")
)
