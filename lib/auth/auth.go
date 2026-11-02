// Package auth asks the operating system to prove who the user is.
package auth

import (
	"errors"
)

// ErrCanceled is the user dismissing the prompt, which needs no message.
var ErrCanceled = errors.New("authentication canceled")
