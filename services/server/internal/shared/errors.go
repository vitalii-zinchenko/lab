package shared

import "errors"

// ErrNotFound is returned when a queried record does not exist.
var ErrNotFound = errors.New("record not found")
