package ports

import "errors"

var (
	ErrNotFound     = errors.New("resource not found")
	ErrAlreadyExist = errors.New("resource already exists")
)
