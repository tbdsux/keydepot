//go:build !darwin

package auth

import (
	"errors"
)

// The system prompt is macOS only: elsewhere Settings doesn't offer it.
func HasTouchID() bool { return false }

// HasSystemAuth reports whether Authenticate can prompt at all.
func HasSystemAuth() bool { return false }

// Authenticate always fails here.
func Authenticate(string) error { return errors.New("The system prompt isn't available here.") }
