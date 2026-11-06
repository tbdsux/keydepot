package view

import (
	"crypto/ed25519"
	"encoding/pem"
	"testing"

	"github.com/egoist/mygo/ui"
	"golang.org/x/crypto/ssh"
)

func TestViewPublicKey(t *testing.T) {
	_, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	block, err := ssh.MarshalPrivateKey(private, "")
	if err != nil {
		t.Fatal(err)
	}

	signer, err := ssh.NewSignerFromKey(private)
	if err != nil {
		t.Fatal(err)
	}

	fingerprint := ssh.FingerprintSHA256(signer.PublicKey())

	s, _ := newTestStore(t)
	if _, err := s.Add("Deploy", string(pem.EncodeToMemory(block)), ""); err != nil {
		t.Fatal(err)
	}

	asked := 0
	a := newApp(s, func(_ string, done func(error)) {
		asked++

		done(nil)
	})
	tt := ui.NewTester(a.View, 1040, 680)

	if tt.HasText(fingerprint) {
		t.Fatal("fingerprint shows before Show")
	}

	click(t, tt, "Show public key")

	if !tt.HasText(fingerprint) || asked != 1 {
		t.Fatalf("after Show: asked %d, %q", asked, tt.Texts())
	}

	a.Lock()
	tt.Frame()

	if tt.HasText(fingerprint) || a.publicID != "" {
		t.Error("public key still shown after locking")
	}
}
