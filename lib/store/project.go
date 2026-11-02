package store

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// AddProject returns the name as stored, trimmed.
func (s *Store) AddProject(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("Give the project a name.")
	}

	sameName := func(p string) bool { return strings.EqualFold(p, name) }
	if sameName(NoProject) || slices.ContainsFunc(s.Projects, sameName) {
		return "", fmt.Errorf("“%s” is already taken.", name)
	}

	s.Projects = append(s.Projects, name)
	if err := s.save(); err != nil {
		s.Projects = s.Projects[:len(s.Projects)-1]
		return "", err
	}

	return name, nil
}

// RemoveProject keeps the project's keys, outside any project: deleting a
// group must never delete a secret.
func (s *Store) RemoveProject(name string) error {
	projects, entries := s.Projects, s.Entries
	s.Projects = slices.DeleteFunc(slices.Clone(projects), func(p string) bool { return p == name })
	s.Entries = slices.Clone(entries)
	for i := range s.Entries {
		if s.Entries[i].Project == name {
			s.Entries[i].Project = ""
		}
	}

	if err := s.save(); err != nil {
		s.Projects, s.Entries = projects, entries
		return err
	}

	return nil
}

// ProjectSize counts the project's keys.
func (s *Store) ProjectSize(project string) int {
	n := 0

	for _, e := range s.Entries {
		if e.Project == project {
			n++
		}
	}

	return n
}
