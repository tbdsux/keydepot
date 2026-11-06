package vault

import (
	"encoding/json"
	"errors"
	"fmt"

	"keydepot/lib/crypt"
)

// How keys are protected. A password-wrapped data key in the vault means
// ModePassword whatever else says otherwise; in the other modes the Keychain
// item decides, so editing a file can't lower the protection.
const (
	ModeSystem   = "system"
	ModePassword = "password"
	ModeNone     = "none"
)

// wrapped returns the file's wrapped key, which is slow, off the lock.
func (v *Vault) wrapped(pick func(*File) *crypt.WrappedKey) *crypt.WrappedKey {
	v.mu.Lock()
	defer v.mu.Unlock()

	return pick(&v.file)
}

// UnlockPassword loads the data key with the password.
func (v *Vault) UnlockPassword(password string) error {
	w := v.wrapped(func(f *File) *crypt.WrappedKey { return f.Password })
	if w == nil {
		return errors.New("This vault has no password.")
	}

	key, err := crypt.Unwrap(w, password, crypt.RolePassword)
	if errors.Is(err, crypt.ErrBadSecret) {
		return ErrWrongPassword
	}

	if err != nil {
		return err
	}

	v.unlocked(key)

	return nil
}

// UnlockRecovery loads the data key with the recovery code.
func (v *Vault) UnlockRecovery(code string) error {
	w := v.wrapped(func(f *File) *crypt.WrappedKey { return f.Recovery })
	if w == nil {
		return errors.New("This vault has no recovery code.")
	}

	key, err := crypt.Unwrap(w, NormalizeCode(code), crypt.RoleRecovery)
	if errors.Is(err, crypt.ErrBadSecret) {
		return errWrongCode
	}

	if err != nil {
		return err
	}

	v.unlocked(key)

	return nil
}

func (v *Vault) unlocked(key []byte) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.dataKey = key
	// A switch into password mode that was interrupted left this behind, and
	// it opens nothing now, but it has no reason to stay.
	v.keychain.Delete(KeychainItem)
}

// Protect sets the mode, on a new vault or an unlocked one. It returns the
// recovery code of ModePassword. Each step leaves a vault that opens, however
// far it got: see the rule on the mode constants.
func (v *Vault) Protect(mode, password string) (string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.exists && v.dataKey == nil {
		if _, err := v.loadLocked(); err != nil {
			return "", err
		}
	}

	if mode == ModePassword {
		return v.intoPassword(password)
	}

	if v.dataKey == nil {
		v.dataKey = crypt.NewKey()
	}

	data, err := json.Marshal(keychainItem{Mode: mode, Key: v.dataKey})
	if err != nil {
		return "", err
	}

	if err := v.keychain.Set(KeychainItem, data); err != nil {
		return "", err
	}

	f := v.file
	f.Mode, f.Password, f.Recovery = mode, nil, nil

	return "", v.save(f)
}

// intoPassword replaces the data key, which the Keychain could give up until
// now, in one write of the vault.
func (v *Vault) intoPassword(password string) (string, error) {
	dataKey := crypt.NewKey()
	items := make(map[string][]byte, len(v.file.Items))

	for name, sealed := range v.file.Items {
		value, err := crypt.Unseal(v.dataKey, sealed, name)
		if err != nil {
			return "", fmt.Errorf("A key is damaged, so nothing was changed: %w", err)
		}

		if items[name], err = crypt.Seal(dataKey, value, name); err != nil {
			return "", err
		}

		clear(value)
	}

	code := newRecoveryCode()
	byPassword, err := crypt.Wrap(dataKey, password, crypt.RolePassword)
	if err != nil {
		return "", err
	}

	byCode, err := crypt.Wrap(dataKey, code, crypt.RoleRecovery)
	if err != nil {
		return "", err
	}

	f := v.file
	f.Mode, f.Password, f.Recovery, f.Items = ModePassword, byPassword, byCode, items
	if err := v.save(f); err != nil {
		return "", err
	}

	clear(v.dataKey)

	v.dataKey = dataKey
	// Leftover, it opens nothing: the next unlock retries.
	v.keychain.Delete(KeychainItem)

	return code, nil
}

// ChangePassword keeps the recovery code valid.
func (v *Vault) ChangePassword(password string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.dataKey == nil {
		return ErrLocked
	}

	w, err := crypt.Wrap(v.dataKey, password, crypt.RolePassword)
	if err != nil {
		return err
	}

	f := v.file
	f.Password = w

	return v.save(f)
}

// RenewRecoveryCode invalidates the old code.
func (v *Vault) RenewRecoveryCode() (string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.dataKey == nil {
		return "", ErrLocked
	}

	code := newRecoveryCode()
	w, err := crypt.Wrap(v.dataKey, code, crypt.RoleRecovery)
	if err != nil {
		return "", err
	}

	f := v.file
	f.Recovery = w

	return code, v.save(f)
}
