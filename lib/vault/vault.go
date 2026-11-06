// Package vault keeps every private key, encrypted, in one file.
package vault

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"runtime"
	"sync"

	"github.com/egoist/mygo"
	"keydepot/lib/crypt"
	"keydepot/lib/durable"
)

const (
	// Version is the newest .keydepot format this build reads.
	Version = 1

	// KeychainItem names the Keychain's copy of the data key.
	KeychainItem = "vault"
)

var (
	// ErrLocked means the vault needs unlocking first.
	ErrLocked = errors.New("Unlock KeyDepot first.")
	// ErrWrongPassword is a password that doesn't open the file.
	ErrWrongPassword = errors.New("That password isn't right.")

	errWrongCode = errors.New("That recovery code isn't right.")
)

// Keychain is the part of mygo.Secrets the vault uses.
type Keychain interface {
	Set(key string, value []byte) error
	Get(key string) ([]byte, error)
	Delete(key string) error
}

// StorageName is how the interface names where the system keeps secrets.
func StorageName() string {
	switch runtime.GOOS {
	case "darwin":
		return "macOS Keychain"

	case "windows":
		return "Windows Credential Manager"
	}

	return "Secret Service"
}

// File is the .keydepot format, of both the vault and exports.
type File struct {
	Version int `json:"version"`
	// Mode is only what Settings shows before unlocking. See the mode constants.
	Mode     string            `json:"mode,omitempty"`
	Password *crypt.WrappedKey `json:"password,omitempty"`
	Recovery *crypt.WrappedKey `json:"recovery,omitempty"`
	Items    map[string][]byte `json:"items"`
}

// keychainItem is the Keychain's copy of the data key, in the modes without a
// KeyDepot password.
type keychainItem struct {
	Mode string `json:"mode"`
	Key  []byte `json:"key"`
}

// Vault holds every private key, encrypted, in one file. It is the store's
// Secrets in the app. Its data key lives in memory from unlocking until
// lock.
type Vault struct {
	mu       sync.Mutex
	keychain Keychain
	path     string
	file     File
	exists   bool
	dataKey  []byte
}

// Open fails on an unreadable file rather than starting empty, which the
// next save would write over.
func Open(keychain Keychain, path string) (*Vault, error) {
	v := &Vault{keychain: keychain, path: path, file: File{Version: Version, Items: map[string][]byte{}}}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return v, nil
	}

	if err != nil {
		return nil, err
	}

	if v.file, err = ParseFile(data); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	v.exists = true

	return v, nil
}

// ParseFile reads a .keydepot file, refusing versions it does not know.
func ParseFile(data []byte) (File, error) {
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return f, err
	}

	if f.Version < 1 || f.Version > Version {
		return f, fmt.Errorf("This file has version %d, which this KeyDepot can't read.", f.Version)
	}

	if f.Items == nil {
		f.Items = map[string][]byte{}
	}

	return f, nil
}

func (v *Vault) save(f File) error {
	data, err := json.Marshal(f)
	if err != nil {
		return err
	}

	if err := durable.WriteFile(v.path, data); err != nil {
		return err
	}

	v.file, v.exists = f, true

	return nil
}

// NeedsSetup reports that the user has yet to choose a protection.
func (v *Vault) NeedsSetup() bool {
	v.mu.Lock()
	defer v.mu.Unlock()

	return !v.exists
}

// PasswordProtected reports that a KeyDepot password opens the vault.
func (v *Vault) PasswordProtected() bool {
	v.mu.Lock()
	defer v.mu.Unlock()

	return v.file.Password != nil
}

// ShownMode is the mode for display only: its source is a plain file.
func (v *Vault) ShownMode() string {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.file.Password != nil {
		return ModePassword
	}

	return v.file.Mode
}

// Load returns the mode the Keychain item names, and readies the data key. An
// item missing from a vault with keys is an error: it must never open them.
func (v *Vault) Load() (string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	return v.loadLocked()
}

func (v *Vault) loadLocked() (string, error) {
	if v.file.Password != nil {
		if v.dataKey == nil {
			return ModePassword, ErrLocked
		}

		return ModePassword, nil
	}

	data, err := v.keychain.Get(KeychainItem)
	if errors.Is(err, mygo.ErrSecretNotFound) {
		if len(v.file.Items) == 0 {
			// Nothing to protect: choosing again is safe, and setting
			// up afresh is better than a vault that can never open.
			v.exists = false
			return "", ErrLocked
		}

		return "", fmt.Errorf("The key that opens your vault is missing from %s, so KeyDepot can't open your keys.", StorageName())
	}

	if err != nil {
		return "", err
	}

	var item keychainItem
	if err := json.Unmarshal(data, &item); err != nil {
		return "", fmt.Errorf("%s holds a damaged vault key: %w", StorageName(), err)
	}

	if len(item.Key) != crypt.KeySize || (item.Mode != ModeSystem && item.Mode != ModeNone) {
		return "", fmt.Errorf("%s holds a damaged vault key.", StorageName())
	}

	v.dataKey = item.Key

	return item.Mode, nil
}

// readyLocked readies the data key for one call, which calls done after. A key
// it had to load, which writes in the modes without a password do without a
// prompt, is forgotten then: a locked vault keeps no key in memory.
func (v *Vault) readyLocked() (done func(), err error) {
	if v.dataKey != nil {
		return func() {}, nil
	}

	if _, err := v.loadLocked(); err != nil {
		return nil, err
	}

	return v.forgetLocked, nil
}

func (v *Vault) forgetLocked() {
	clear(v.dataKey)

	v.dataKey = nil
}

// Lock forgets the data key.
// shortcut: Go may keep copies the clear can't reach, upgrade to locked memory if memory dumps enter the threat model.
func (v *Vault) Lock() {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.forgetLocked()
}

// Set encrypts value under key and saves the vault.
func (v *Vault) Set(key string, value []byte) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	done, err := v.readyLocked()
	if err != nil {
		return err
	}

	defer done()

	sealed, err := crypt.Seal(v.dataKey, value, key)
	if err != nil {
		return err
	}

	f := v.file
	f.Items = maps.Clone(v.file.Items)
	f.Items[key] = sealed

	return v.save(f)
}

// Get decrypts the value under key.
func (v *Vault) Get(key string) ([]byte, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	done, err := v.readyLocked()
	if err != nil {
		return nil, err
	}

	defer done()

	sealed, ok := v.file.Items[key]
	if !ok {
		return nil, mygo.ErrSecretNotFound
	}

	value, err := crypt.Unseal(v.dataKey, sealed, key)
	if err != nil {
		return nil, errors.New("This key is damaged or was changed outside KeyDepot.")
	}

	return value, nil
}

// Delete needs no data key, as the item is removed unread.
func (v *Vault) Delete(key string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if _, ok := v.file.Items[key]; !ok {
		return nil
	}

	f := v.file
	f.Items = maps.Clone(v.file.Items)
	delete(f.Items, key)

	return v.save(f)
}

// Locked reports that no data key is in memory.
func (v *Vault) Locked() bool {
	v.mu.Lock()
	defer v.mu.Unlock()

	return v.dataKey == nil
}
