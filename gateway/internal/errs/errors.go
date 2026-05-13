package errs

import "errors"

var (
	ErrKeyNotFound      = errors.New("key not found")
	ErrPermissionDenied = errors.New("permission denied")
)
