package vault

import (
	"crypto/rand"
	"strings"
)

// newRecoveryCode returns about 130 random bits, far beyond guessing.
func newRecoveryCode() string { return rand.Text() }

// NormalizeCode forgives how a code was retyped.
func NormalizeCode(code string) string {
	return strings.ToUpper(strings.NewReplacer(" ", "", "-", "", "\n", "").Replace(code))
}

// FormatCode splits a code into groups for reading.
func FormatCode(code string) string {
	var groups []string

	for len(code) > 4 {
		groups = append(groups, code[:4])
		code = code[4:]
	}

	return strings.Join(append(groups, code), "-")
}
