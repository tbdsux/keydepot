package pubkey

import (
	"crypto/ed25519"
	"encoding/pem"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestDerive(t *testing.T) {
	public, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	sshPublic, err := ssh.NewPublicKey(public)
	if err != nil {
		t.Fatal(err)
	}

	want := ssh.FingerprintSHA256(sshPublic)

	plain, err := ssh.MarshalPrivateKey(private, "")
	if err != nil {
		t.Fatal(err)
	}

	locked, err := ssh.MarshalPrivateKeyWithPassphrase(private, "", []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}

	for name, block := range map[string]*pem.Block{"plain": plain, "passphrase": locked} {
		got, err := Derive(string(pem.EncodeToMemory(block)))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}

		if got.Fingerprint != want || !strings.HasPrefix(got.Authorized, "ssh-ed25519 ") {
			t.Errorf("%s: %+v", name, got)
		}
	}

	if _, err := Derive("not a key"); err == nil {
		t.Error("derived a public key from garbage")
	}
}
