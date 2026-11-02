package view

import (
	"path/filepath"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"keydepot/lib/store"
	"keydepot/lib/vault"
)

const testKey = "-----BEGIN OPENSSH PRIVATE KEY-----\nabc\n-----END OPENSSH PRIVATE KEY-----\n"

// fakeSecrets keeps tests out of the real keychain.
type fakeSecrets map[string][]byte

func (f fakeSecrets) Set(k string, v []byte) error { f[k] = v; return nil }

func (f fakeSecrets) Get(k string) ([]byte, error) {
	v, ok := f[k]
	if !ok {
		return nil, mygo.ErrSecretNotFound
	}

	return v, nil
}

func (f fakeSecrets) Delete(k string) error { delete(f, k); return nil }

func newTestStore(t *testing.T) (*store.Store, fakeSecrets) {
	t.Helper()

	f := fakeSecrets{}
	s, err := store.Open(f, filepath.Join(t.TempDir(), "keys.json"))
	if err != nil {
		t.Fatal(err)
	}

	return s, f
}

const (
	testPassword = "correct horse"
	exportPass   = "a long export password"
)

// newVaultApp is the app as main wires it, but for the system prompt.
func newVaultApp(t *testing.T, systemAuth bool) (*App, *vault.Vault, fakeSecrets, *[]string, *ui.Tester) {
	t.Helper()

	f := fakeSecrets{}
	dir := t.TempDir()
	v, err := vault.Open(f, filepath.Join(dir, "vault.keydepot"))
	if err != nil {
		t.Fatal(err)
	}

	s, err := store.Open(v, filepath.Join(dir, "keys.json"))
	if err != nil {
		t.Fatal(err)
	}

	a := newApp(s, nil)
	a.vault, a.systemAuth, a.touchID = v, systemAuth, systemAuth
	a.auth = a.prompt

	var prompts []string

	a.authenticate = func(reason string) error {
		prompts = append(prompts, reason)
		return nil
	}

	return a, v, f, &prompts, ui.NewTester(a.View, 1040, 680)
}

func click(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()

	if err := tt.Click(label); err != nil {
		t.Fatalf("%v in %q", err, tt.Texts())
	}

	tt.Frame()
}
