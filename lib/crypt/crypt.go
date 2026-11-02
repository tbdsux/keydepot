// Package crypt seals data with AES-GCM and wraps data keys with Argon2id.
package crypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"

	"golang.org/x/crypto/argon2"
)

const (
	// KeySize is the size of a data key, in bytes.
	KeySize  = 32
	saltSize = 16

	// RolePassword and RoleRecovery are additional data that tie each wrapped key to its role.
	RolePassword = "wrap.password"
	RoleRecovery = "wrap.recovery"
)

// ErrBadSecret is Unwrap given the wrong password or code.
var ErrBadSecret = errors.New("wrong secret")

// kdfSettings are the costs new wrapped keys use.
var kdfSettings = Params{Time: 3, Memory: 64 * 1024, Threads: 4}

// Params are the Argon2id costs of a wrapped key.
type Params struct {
	Time    uint32 `json:"time"`
	Memory  uint32 `json:"memory"` // KiB
	Threads uint8  `json:"threads"`
}

// check guards Argon2id, which panics on zero costs, and keeps a hostile file
// from asking for more memory or time than any file KeyDepot writes.
func (p Params) check() error {
	if p.Time < 1 || p.Time > 16 || p.Threads < 1 || p.Threads > 16 || p.Memory < 8*1024 || p.Memory > 1024*1024 {
		return errors.New("This file asks for unsupported password settings.")
	}

	return nil
}

// WrappedKey is the data key, encrypted with a key derived from a secret.
type WrappedKey struct {
	Salt   []byte `json:"salt"`
	KDF    Params `json:"kdf"`
	Sealed []byte `json:"sealed"`
}

// Seal encrypts plain with key, tying it to role.
func Seal(key, plain []byte, role string) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	rand.Read(nonce)

	return gcm.Seal(nonce, nonce, plain, []byte(role)), nil
}

// Unseal decrypts what Seal made for the same key and role.
func Unseal(key, sealed []byte, role string) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(sealed) < gcm.NonceSize() {
		return nil, errors.New("sealed data is too short")
	}

	return gcm.Open(nil, sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():], []byte(role))
}

// Wrap encrypts dataKey with a key derived from secret.
func Wrap(dataKey []byte, secret, role string) (*WrappedKey, error) {
	w := &WrappedKey{Salt: make([]byte, saltSize), KDF: kdfSettings}
	rand.Read(w.Salt)

	var err error

	w.Sealed, err = Seal(derive(secret, w.Salt, w.KDF), dataKey, role)

	return w, err
}

// Unwrap reverses Wrap, returning ErrBadSecret for the wrong secret.
func Unwrap(w *WrappedKey, secret, role string) ([]byte, error) {
	if err := w.KDF.check(); err != nil {
		return nil, err
	}

	if len(w.Salt) != saltSize {
		return nil, errors.New("This file is damaged.")
	}

	key, err := Unseal(derive(secret, w.Salt, w.KDF), w.Sealed, role)
	if err != nil {
		return nil, ErrBadSecret
	}

	if len(key) != KeySize {
		return nil, errors.New("This file is damaged.")
	}

	return key, nil
}

func derive(secret string, salt []byte, p Params) []byte {
	return argon2.IDKey([]byte(secret), salt, p.Time, p.Memory, p.Threads, KeySize)
}

// NewKey returns a random data key.
func NewKey() []byte {
	key := make([]byte, KeySize)
	rand.Read(key)

	return key
}
