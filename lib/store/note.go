package store

import (
	"errors"
	"fmt"

	"github.com/egoist/mygo"
)

// maxNoteSize keeps notes short beside the keys they describe.
const maxNoteSize = 64 << 10

// NoteName is the key's note's name among the secrets. Notes are sealed like
// keys, as they often say what a key opens.
func NoteName(id string) string { return "note." + id }

// CheckNote says why a note can't be stored, if so.
func CheckNote(note string) error {
	if len(note) > maxNoteSize {
		return fmt.Errorf("This note is %d bytes. The limit is %d.", len(note), maxNoteSize)
	}

	return nil
}

// Note is empty for a key without one.
func (s *Store) Note(id string) (string, error) {
	value, err := s.secrets.Get(NoteName(id))
	if errors.Is(err, mygo.ErrSecretNotFound) {
		return "", nil
	}

	return string(value), err
}

// SetNote removes the note when it's empty.
func (s *Store) SetNote(id, note string) error {
	if err := CheckNote(note); err != nil {
		return err
	}

	if note == "" {
		return s.secrets.Delete(NoteName(id))
	}

	return s.secrets.Set(NoteName(id), []byte(note))
}
