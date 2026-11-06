// Package paths says where KeyDepot keeps its files.
package paths

import (
	"path/filepath"

	"github.com/egoist/mygo"
)

// Index is the unencrypted key list.
func Index() (string, error) { return userFile("keys.json") }

// Vault is the encrypted vault.
func Vault() (string, error) { return userFile("vault.keydepot") }

// Shortcuts is the user's shortcut preferences.
func Shortcuts() (string, error) { return userFile("shortcuts.json") }

// AutoLock is when KeyDepot locks itself.
func AutoLock() (string, error) { return userFile("autolock.json") }

func userFile(name string) (string, error) {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, name), nil
}
