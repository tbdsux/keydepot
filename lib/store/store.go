// Package store keeps keys in the vault, and what the app lists about them beside it.
package store

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"keydepot/lib/durable"
)

// Entry is what the app lists about a key. The key itself is only ever in
// the encrypted vault.
type Entry struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Type    string    `json:"type"`
	AddedAt time.Time `json:"addedAt"`
	Project string    `json:"project,omitempty"`
	// UpdatedAt is when the key itself last changed; renaming doesn't count.
	UpdatedAt time.Time `json:"updatedAt,omitzero"`
}

// maxKeySize keeps pasted or imported text from bloating the vault, which is
// written whole; it is well above any real private key.
const maxKeySize = 1 << 20

// NoProject is how the interface shows a key outside any project, so no
// project may take the name.
const NoProject = "No project"

// Index is the JSON file's shape.
type Index struct {
	Projects []string `json:"projects"`
	Keys     []Entry  `json:"keys"`
}

// Secrets is what the store needs of the vault.
type Secrets interface {
	Set(key string, value []byte) error
	Get(key string) ([]byte, error)
	Delete(key string) error
}

// Store keeps keys in the vault, and their names and types in a JSON file
// beside it, which the app can show while the vault is locked.
type Store struct {
	secrets Secrets
	path    string
	Entries []Entry
	// projects holds names, which are unique and which keys refer to.
	Projects []string
}

// Open fails on an unreadable index rather than starting empty, which
// the next save would write over.
func Open(secrets Secrets, path string) (*Store, error) {
	s := &Store{secrets: secrets, path: path}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}

	if err != nil {
		return nil, err
	}

	var index Index

	// Files from before projects are a bare array of keys.
	if bytes.HasPrefix(bytes.TrimSpace(data), []byte("[")) {
		err = json.Unmarshal(data, &index.Keys)
	} else {
		err = json.Unmarshal(data, &index)
	}

	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	s.Entries, s.Projects = index.Keys, index.Projects

	return s, nil
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(Index{Projects: s.Projects, Keys: s.Entries}, "", "  ")
	if err != nil {
		return err
	}

	return durable.WriteFile(s.path, data)
}

// SecretName is the key's name among the secrets.
func SecretName(id string) string { return "key." + id }

// Add stores value exactly as given: OpenSSH refuses keys that lost their
// final newline.
func (s *Store) Add(name, value, project string) (Entry, error) {
	if err := CheckKey(name, value); err != nil {
		return Entry{}, err
	}

	if strings.TrimSpace(value) == "" {
		return Entry{}, errors.New("Paste the private key.")
	}

	name = strings.TrimSpace(name)
	e := Entry{ID: rand.Text(), Name: name, Type: DetectKeyType(value), AddedAt: time.Now(), Project: project}
	if err := s.secrets.Set(SecretName(e.ID), []byte(value)); err != nil {
		return Entry{}, err
	}

	s.Entries = append(s.Entries, e)
	if err := s.save(); err != nil {
		s.Entries = s.Entries[:len(s.Entries)-1]
		s.secrets.Delete(SecretName(e.ID))

		return Entry{}, err
	}

	return e, nil
}

// CheckKey says why a name and key can't be stored, if so.
func CheckKey(name, value string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("Give the key a name.")
	}

	if len(value) > maxKeySize {
		return fmt.Errorf("This key is %d bytes. The limit is %d.", len(value), maxKeySize)
	}

	return nil
}

// Update renames a key and, unless value is empty, replaces it. The old key
// comes back if the index can't be saved, so a failure loses nothing.
func (s *Store) Update(id, name, value string) error {
	if err := CheckKey(name, value); err != nil {
		return err
	}

	i := slices.IndexFunc(s.Entries, func(e Entry) bool { return e.ID == id })
	if i < 0 {
		return errors.New("This key no longer exists.")
	}

	before := s.Entries
	s.Entries = slices.Clone(before)
	s.Entries[i].Name = strings.TrimSpace(name)

	var old []byte

	if value != "" {
		var err error

		old, err = s.secrets.Get(SecretName(id))
		if err != nil && !errors.Is(err, mygo.ErrSecretNotFound) {
			s.Entries = before
			return err
		}

		if err := s.secrets.Set(SecretName(id), []byte(value)); err != nil {
			s.Entries = before
			return err
		}

		s.Entries[i].Type, s.Entries[i].UpdatedAt = DetectKeyType(value), time.Now()
	}

	err := s.save()
	if err == nil {
		return nil
	}

	s.Entries = before

	switch {
	case value == "":

	case old != nil:
		err = errors.Join(err, s.secrets.Set(SecretName(id), old))

	default:
		err = errors.Join(err, s.secrets.Delete(SecretName(id)))
	}

	return err
}

// Get returns the key itself.
func (s *Store) Get(id string) (string, error) {
	value, err := s.secrets.Get(SecretName(id))
	return string(value), err
}

// Remove deletes the secret first: a failure then leaves a key the user can
// still see and retry, never one hidden in the vault.
func (s *Store) Remove(id string) error {
	if err := s.secrets.Delete(SecretName(id)); err != nil {
		return err
	}

	before := s.Entries
	s.Entries = slices.DeleteFunc(slices.Clone(before), func(e Entry) bool { return e.ID == id })
	if err := s.save(); err != nil {
		s.Entries = before
		return err
	}

	return nil
}

// MoveKey puts a key in a project, or in none for "".
func (s *Store) MoveKey(id, project string) error {
	before := s.Entries
	s.Entries = slices.Clone(before)
	for i := range s.Entries {
		if s.Entries[i].ID == id {
			s.Entries[i].Project = project
		}
	}

	if err := s.save(); err != nil {
		s.Entries = before
		return err
	}

	return nil
}
