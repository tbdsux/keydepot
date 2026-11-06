package export

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo"
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

func editFile(t *testing.T, path string, edit func(*vault.File)) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var f vault.File
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatal(err)
	}

	edit(&f)

	if data, err = json.Marshal(f); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestExportRoundTrip(t *testing.T) {
	src, _ := newTestStore(t)
	if _, err := src.AddProject("Work"); err != nil {
		t.Fatal(err)
	}

	a, err := src.Add("Deploy", testKey, "Work")
	if err != nil {
		t.Fatal(err)
	}

	b, err := src.Add("Loose", "other key", "")
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "export.keydepot")
	keys := map[string]string{a.ID: testKey, b.ID: "other key"}
	notes := map[string]string{a.ID: "Deploys the website"}
	if err := Write(path, exportPass, store.Index{Projects: src.Projects, Keys: src.Entries}, keys, notes); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	for _, leak := range []string{"BEGIN", "Deploy", "Work", "Loose", "website"} {
		if strings.Contains(string(data), leak) {
			t.Errorf("export shows %q", leak)
		}
	}

	if _, _, _, err := Read(path, "wrong password"); !errors.Is(err, vault.ErrWrongPassword) {
		t.Errorf("wrong password: %v", err)
	}

	dst, _ := newTestStore(t)
	existing, err := dst.Add("Deploy", "kept", "")
	if err != nil {
		t.Fatal(err)
	}

	index, read, readNotes, err := Read(path, exportPass)
	if err != nil {
		t.Fatal(err)
	}

	n, err := dst.ImportKeys(index, read, readNotes)
	if err != nil || n != 2 {
		t.Fatalf("import: %d, %v", n, err)
	}

	if len(dst.Entries) != 3 || dst.ProjectSize("Work") != 1 {
		t.Fatalf("entries %+v", dst.Entries)
	}

	if got, _ := dst.Get(existing.ID); got != "kept" {
		t.Errorf("import overwrote a key: %q", got)
	}

	for _, e := range dst.Entries[1:] {
		if e.ID == a.ID || e.ID == b.ID {
			t.Errorf("import kept the export's ID: %+v", e)
		}
	}

	if got, _ := dst.Get(dst.Entries[1].ID); got != testKey {
		t.Errorf("imported key %q", got)
	}

	if got, _ := dst.Note(dst.Entries[1].ID); got != notes[a.ID] {
		t.Errorf("imported note %q", got)
	}

	if got, _ := dst.Note(dst.Entries[2].ID); got != "" {
		t.Errorf("a key without a note got %q", got)
	}

	// Importing again only adds, too.
	if n, err := dst.ImportKeys(index, read, readNotes); err != nil || n != 2 || len(dst.Entries) != 5 {
		t.Errorf("second import: %d, %v, %d entries", n, err, len(dst.Entries))
	}
}

func TestReadExportRefusesUntrustedFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "export.keydepot")

	good := func() {
		t.Helper()

		e := store.Entry{ID: "A", Name: "Deploy"}
		if err := Write(path, exportPass, store.Index{Keys: []store.Entry{e}}, map[string]string{"A": testKey}, nil); err != nil {
			t.Fatal(err)
		}
	}

	good()

	if _, _, _, err := Read(path, exportPass); err != nil {
		t.Fatal(err)
	}

	for name, edit := range map[string]func(*vault.File){
		"future version":  func(f *vault.File) { f.Version = 99 },
		"no password":     func(f *vault.File) { f.Password = nil },
		"hostile memory":  func(f *vault.File) { f.Password.KDF.Memory = 1 << 31 },
		"hostile threads": func(f *vault.File) { f.Password.KDF.Threads = 0 },
		"no list":         func(f *vault.File) { delete(f.Items, item) },
		"changed list":    func(f *vault.File) { f.Items[item][20] ^= 1 },
		"missing key":     func(f *vault.File) { delete(f.Items, store.SecretName("A")) },
		"changed key":     func(f *vault.File) { f.Items[store.SecretName("A")][20] ^= 1 },
	} {
		good()
		editFile(t, path, edit)

		if _, _, _, err := Read(path, exportPass); err == nil {
			t.Errorf("%s: read", name)
		}
	}

	if err := os.WriteFile(path, make([]byte, maxSize+1), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, _, _, err := Read(path, exportPass); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Errorf("oversized: %v", err)
	}

	// A vault isn't an export, even with the right password.
	vaultPath := filepath.Join(dir, "vault.keydepot")
	v, err := vault.Open(fakeSecrets{}, vaultPath)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := v.Protect(vault.ModePassword, testPassword); err != nil {
		t.Fatal(err)
	}

	if _, _, _, err := Read(vaultPath, testPassword); err == nil {
		t.Error("read a vault as an export")
	}
}
