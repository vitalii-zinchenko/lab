package shared

import "errors"

// IsForeignKeyViolation returns true for PostgreSQL error code 23503.
func IsForeignKeyViolation(err error) bool {
	type pgErrorCode interface {
		SQLState() string
	}
	var pgErr pgErrorCode
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23503"
	}
	return false
}
