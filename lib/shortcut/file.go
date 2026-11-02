package shortcut

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/egoist/mygo/ui"
	"keydepot/lib/durable"
)

// Load skips what it can't read: the file holds only preferences.
func Load(path string) (map[string]Shortcut, error) {
	shortcuts := map[string]Shortcut{}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return shortcuts, nil
	}

	if err != nil {
		return shortcuts, err
	}

	var saved map[string]string
	if err := json.Unmarshal(data, &saved); err != nil {
		return shortcuts, fmt.Errorf("%s: %w", path, err)
	}

	for id, text := range saved {
		if text == "" {
			shortcuts[id] = Shortcut{}
		} else if s, ok := Parse(text); ok {
			shortcuts[id] = s
		}
	}

	return shortcuts, nil
}

// Save writes shortcuts to path; an empty value means removed.
func Save(path string, shortcuts map[string]Shortcut) error {
	saved := make(map[string]string, len(shortcuts))

	for id, s := range shortcuts {
		saved[id] = ""

		if s.Key != ui.KeyUnknown {
			saved[id] = s.String()
		}
	}

	data, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		return err
	}

	return durable.WriteFile(path, data)
}
