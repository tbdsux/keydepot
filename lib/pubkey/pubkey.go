// Package pubkey works out the public half of an SSH private key.
package pubkey

import (
	"errors"
	"strings"

	"golang.org/x/crypto/ssh"
)

// Key is a private key's public half.
type Key struct {
	// Authorized is the line servers list in authorized_keys.
	Authorized string
	// Fingerprint is the SHA256 fingerprint ssh-keygen -l shows.
	Fingerprint string
}

// Derive reads OpenSSH, PEM and PKCS#8 keys. An OpenSSH key with a passphrase
// still works: its public half isn't encrypted.
func Derive(private string) (Key, error) {
	var public ssh.PublicKey

	signer, err := ssh.ParsePrivateKey([]byte(private))

	var missing *ssh.PassphraseMissingError

	switch {
	case err == nil:
		public = signer.PublicKey()

	case errors.As(err, &missing) && missing.PublicKey != nil:
		public = missing.PublicKey

	case errors.As(err, &missing):
		return Key{}, errors.New("This key has a passphrase, so its public key can't be read.")

	default:
		return Key{}, errors.New("This isn't an SSH key KeyDepot can read.")
	}

	return Key{
		Authorized:  strings.TrimSpace(string(ssh.MarshalAuthorizedKey(public))),
		Fingerprint: ssh.FingerprintSHA256(public),
	}, nil
}
