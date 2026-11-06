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

	key, items, rotated := v.dataKey, v.file.Items, false

	switch {
	case key == nil:
		key = crypt.NewKey()

	case v.file.Password != nil:
		// A copy of the vault from before, with its password or code, opens
		// the old key: it must open nothing saved from now on.
		var err error
		if key, items, err = v.rekeyed(); err != nil {
			return "", err
		}

		rotated = true
	}

	data, err := json.Marshal(keychainItem{Mode: mode, Key: key})
	if err != nil {
		return "", err
	}

	if err := v.keychain.Set(KeychainItem, data); err != nil {
		return "", err
	}

	f := v.file
	f.Mode, f.Password, f.Recovery, f.Items = mode, nil, nil, items
	if err := v.save(f); err != nil {
		return "", err
	}

	if rotated {
		clear(v.dataKey)
	}

	v.dataKey = key

	return "", nil
}

// rekeyed seals every item again under a new data key, so that no old copy
// of the vault, with the secret that opened it, opens what is saved next.
func (v *Vault) rekeyed() ([]byte, map[string][]byte, error) {
	dataKey := crypt.NewKey()
	items := make(map[string][]byte, len(v.file.Items))

	for name, sealed := range v.file.Items {
		value, err := crypt.Unseal(v.dataKey, sealed, name)
		if err != nil {
			return nil, nil, fmt.Errorf("A key is damaged, so nothing was changed: %w", err)
		}

		if items[name], err = crypt.Seal(dataKey, value, name); err != nil {
			return nil, nil, err
		}

		clear(value)
	}

	return dataKey, items, nil
}

// intoPassword replaces the data key, which the Keychain, an old password or
// an old code could give up until now, in one write of the vault. Changing
// the password and renewing the code both come here.
func (v *Vault) intoPassword(password string) (string, error) {
	dataKey, items, err := v.rekeyed()
	if err != nil {
		return "", err
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

// RenewRecoveryCode replaces the data key with the code, as only that stops
// the old code opening a copy of the vault from before. It needs the
// password to wrap the new key, and checks it first.
func (v *Vault) RenewRecoveryCode(password string) (string, error) {
	if err := v.UnlockPassword(password); err != nil {
		return "", err
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	return v.intoPassword(password)
}
