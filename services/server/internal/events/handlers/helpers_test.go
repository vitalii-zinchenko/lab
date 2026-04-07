package handlers_test

import (
	"strings"
	"testing"
)

func mustStringReader(s string) *strings.Reader {
	return strings.NewReader(s)
}

// compile-time check that testing is used (prevents "imported and not used" errors).
var _ *testing.T
