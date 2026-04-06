package storage

import "errors"

var(
	ErrFileNotFound = errors.New("file not found")
	ErrFileAlreadyExists = errors.New("file already exists")
)