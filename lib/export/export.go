// Package export writes and reads password-protected files of keys.
package export

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"keydepot/lib/crypt"
	"keydepot/lib/durable"
	"keydepot/lib/store"
	"keydepot/lib/vault"
)

// item names the sealed key list inside an export.
const item = "index"

// maxSize keeps a hostile file from filling memory before it is checked.
const maxSize = 64 << 20

// Write encrypts keys and notes, by entry ID, and the list describing them
// under a data key of their own, so an export reveals nothing about the vault's.
func Write(path, password string, index store.Index, keys, notes map[string]string) error {
	dataKey := crypt.NewKey()
	f := vault.File{Version: vault.Version, Items: make(map[string][]byte, len(keys)+len(notes)+1)}
	list, err := json.Marshal(index)
	if err != nil {
		return err
	}

	if f.Items[item], err = crypt.Seal(dataKey, list, item); err != nil {
		return err
	}

	for id, value := range keys {
		name := store.SecretName(id)
		if f.Items[name], err = crypt.Seal(dataKey, []byte(value), name); err != nil {
			return err
		}
	}

	for id, note := range notes {
		name := store.NoteName(id)
		if f.Items[name], err = crypt.Seal(dataKey, []byte(note), name); err != nil {
			return err
		}
	}

	if f.Password, err = crypt.Wrap(dataKey, password, crypt.RolePassword); err != nil {
		return err
	}

	data, err := json.Marshal(f)
	if err != nil {
		return err
	}

	return durable.WriteFile(path, data)
}

// Read treats path as untrusted: every check runs before Argon2id, and
// nothing is returned unless every key and note decrypts. Exports from before
// notes have none.
func Read(path, password string) (store.Index, map[string]string, map[string]string, error) {
	var index store.Index

	info, err := os.Stat(path)
	if err != nil {
		return index, nil, nil, err
	}

	if info.Size() > maxSize {
		return index, nil, nil, errors.New("This file is too large to be a KeyDepot export.")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return index, nil, nil, err
	}

	f, err := vault.ParseFile(data)
	if err != nil {
		return index, nil, nil, fmt.Errorf("This isn't a KeyDepot export: %w", err)
	}

	if f.Password == nil || f.Items[item] == nil {
		return index, nil, nil, errors.New("This isn't a KeyDepot export.")
	}

	dataKey, err := crypt.Unwrap(f.Password, password, crypt.RolePassword)
	if errors.Is(err, crypt.ErrBadSecret) {
		return index, nil, nil, vault.ErrWrongPassword
	}

	if err != nil {
		return index, nil, nil, err
	}

	list, err := crypt.Unseal(dataKey, f.Items[item], item)
	if err != nil {
		return index, nil, nil, errors.New("This export is damaged.")
	}

	if err := json.Unmarshal(list, &index); err != nil {
		return index, nil, nil, errors.New("This export is damaged.")
	}

	keys := make(map[string]string, len(index.Keys))
	notes := map[string]string{}

	for _, e := range index.Keys {
		name := store.SecretName(e.ID)
		sealed, ok := f.Items[name]
		if !ok {
			return index, nil, nil, fmt.Errorf("This export is damaged: “%s” is missing.", e.Name)
		}

		value, err := crypt.Unseal(dataKey, sealed, name)
		if err != nil {
			return index, nil, nil, fmt.Errorf("This export is damaged: “%s” can't be read.", e.Name)
		}

		keys[e.ID] = string(value)

		name = store.NoteName(e.ID)
		if sealed, ok := f.Items[name]; ok {
			note, err := crypt.Unseal(dataKey, sealed, name)
			if err != nil {
				return index, nil, nil, fmt.Errorf("This export is damaged: the note on “%s” can't be read.", e.Name)
			}

			notes[e.ID] = string(note)
		}
	}

	return index, keys, notes, nil
}
