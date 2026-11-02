package crypt

import (
	"errors"
	"testing"
)

func init() { kdfSettings = Params{Time: 1, Memory: 8 * 1024, Threads: 1} }

// Hostile costs must fail before Argon2id runs: these would otherwise panic
// or take the machine's memory.
func TestKDFCostsAreCapped(t *testing.T) {
	for _, p := range []Params{
		{},
		{Time: 1, Memory: 8 * 1024},
		{Time: 1, Memory: 1 << 31, Threads: 1},
		{Time: 1 << 30, Memory: 8 * 1024, Threads: 1},
		{Time: 1, Memory: 1, Threads: 1},
	} {
		w := &WrappedKey{Salt: make([]byte, saltSize), KDF: p}
		if _, err := Unwrap(w, "x", RolePassword); err == nil || errors.Is(err, ErrBadSecret) {
			t.Errorf("%+v: %v", p, err)
		}
	}

	if err := kdfSettings.check(); err != nil {
		t.Errorf("test settings: %v", err)
	}

	if err := (Params{Time: 3, Memory: 64 * 1024, Threads: 4}).check(); err != nil {
		t.Errorf("app settings: %v", err)
	}
}
