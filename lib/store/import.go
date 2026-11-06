package store

import (
	"fmt"
	"slices"
	"strings"
)

// ImportKeys only adds: imported keys get new IDs and overwrite nothing. It
// checks every key and note before adding any, as the export's contents are
// untrusted.
func (s *Store) ImportKeys(index Index, keys, notes map[string]string) (int, error) {
	for _, e := range index.Keys {
		if err := CheckKey(e.Name, keys[e.ID]); err != nil {
			return 0, fmt.Errorf("“%s”: %w", e.Name, err)
		}

		if strings.TrimSpace(keys[e.ID]) == "" {
			return 0, fmt.Errorf("“%s” has no key.", e.Name)
		}

		if err := CheckNote(notes[e.ID]); err != nil {
			return 0, fmt.Errorf("“%s”: %w", e.Name, err)
		}
	}

	for _, p := range index.Projects {
		s.importProject(p)
	}

	for i, e := range index.Keys {
		added, err := s.Add(e.Name, keys[e.ID], s.importProject(e.Project))
		if err != nil {
			return i, err
		}

		if err := s.SetNote(added.ID, notes[e.ID]); err != nil {
			return i + 1, fmt.Errorf("“%s” was imported without its note: %w", e.Name, err)
		}
	}

	return len(index.Keys), nil
}

// importProject finds or makes the project; a name it can't use files the key
// outside any project, as keys never depend on one.
func (s *Store) importProject(name string) string {
	name = strings.TrimSpace(name)
	if i := slices.IndexFunc(s.Projects, func(p string) bool { return strings.EqualFold(p, name) }); i >= 0 {
		return s.Projects[i]
	}

	made, err := s.AddProject(name)
	if err != nil {
		return ""
	}

	return made
}
