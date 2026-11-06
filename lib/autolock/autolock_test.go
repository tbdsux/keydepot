package autolock

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenAndSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "autolock.json")

	l, err := Open(path)
	if err != nil || l.After() != Default {
		t.Fatalf("missing file: %v, %v", l.After(), err)
	}

	for _, d := range []time.Duration{0, 5 * time.Minute} {
		if err := l.SetAfter(d); err != nil {
			t.Fatal(err)
		}

		if again, err := Open(path); err != nil || again.After() != d {
			t.Errorf("saved %v, read %v, %v", d, again.After(), err)
		}
	}

	if err := os.WriteFile(path, []byte(`{"lockAfterMinutes":9223372036854775807}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if l, err := Open(path); err != nil || l.After() != maxMinutes*time.Minute {
		t.Errorf("huge value: %v, %v", l.After(), err)
	}

	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}

	if l, err := Open(path); err == nil || l.After() != Default {
		t.Errorf("damaged file: %v, %v", l.After(), err)
	}
}
