package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo"
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

func newTestStore(t *testing.T) (*Store, fakeSecrets) {
	t.Helper()

	f := fakeSecrets{}
	s, err := Open(f, filepath.Join(t.TempDir(), "keys.json"))
	if err != nil {
		t.Fatal(err)
	}

	return s, f
}

func TestStoreKeepsKeysOutOfTheIndex(t *testing.T) {
	s, f := newTestStore(t)
	e, err := s.Add(" Deploy ", testKey, "")
	if err != nil {
		t.Fatal(err)
	}

	if e.Name != "Deploy" || e.Type != "OpenSSH" {
		t.Errorf("entry %+v", e)
	}

	if got, _ := s.Get(e.ID); got != testKey {
		t.Errorf("got %q, want the key unchanged", got)
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(data), "BEGIN") {
		t.Errorf("index holds the key: %s", data)
	}

	reopened, err := Open(f, s.path)
	if err != nil || len(reopened.Entries) != 1 {
		t.Fatalf("reopened %v, %v", reopened, err)
	}

	if err := reopened.Remove(e.ID); err != nil {
		t.Fatal(err)
	}

	if len(f) != 0 || len(reopened.Entries) != 0 {
		t.Errorf("after remove: secrets %v, entries %v", f, reopened.Entries)
	}
}

func TestStoreRejectsBadInput(t *testing.T) {
	s, f := newTestStore(t)

	for _, c := range [][2]string{{"", testKey}, {"a", "  \n"}, {"a", strings.Repeat("x", maxKeySize+1)}} {
		if _, err := s.Add(c[0], c[1], ""); err == nil {
			t.Errorf("add(%q, %d bytes) succeeded", c[0], len(c[1]))
		}
	}

	if len(f) != 0 {
		t.Errorf("rejected keys reached the keychain: %v", f)
	}
}

func TestStoreRollsBackWhenIndexCannotSave(t *testing.T) {
	s, f := newTestStore(t)
	s.path = filepath.Join(s.path, "missing", "keys.json")
	if _, err := s.Add("a", testKey, ""); err == nil {
		t.Fatal("add succeeded with an unwritable index")
	}

	if len(f) != 0 || len(s.Entries) != 0 {
		t.Errorf("left behind: secrets %v, entries %v", f, s.Entries)
	}
}

func TestOpenStoreRefusesCorruptIndex(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keys.json")
	if err := os.WriteFile(path, []byte("{oops"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Open(fakeSecrets{}, path); err == nil {
		t.Fatal("opened a corrupt index")
	}
}

func TestStoreRemoveKeepsEntryWhenKeychainFails(t *testing.T) {
	s, _ := newTestStore(t)
	e, err := s.Add("a", testKey, "")
	if err != nil {
		t.Fatal(err)
	}

	s.secrets = failingDelete{s.secrets}
	if err := s.Remove(e.ID); err == nil {
		t.Fatal("remove succeeded")
	}

	if len(s.Entries) != 1 {
		t.Errorf("entries %v", s.Entries)
	}
}

type failingDelete struct{ Secrets }

func (failingDelete) Delete(string) error { return errors.New("locked") }

func TestOpenStoreReadsIndexFromBeforeProjects(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keys.json")
	legacy := `[{"id":"A","name":"Deploy","type":"OpenSSH","addedAt":"2026-01-02T03:04:05Z"}]`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}

	s, err := Open(fakeSecrets{}, path)
	if err != nil {
		t.Fatal(err)
	}

	if len(s.Entries) != 1 || s.Entries[0].Name != "Deploy" || s.Entries[0].Project != "" {
		t.Fatalf("entries %+v", s.Entries)
	}

	if _, err := s.AddProject("Work"); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(fakeSecrets{}, path)
	if err != nil || len(reopened.Entries) != 1 || len(reopened.Projects) != 1 {
		t.Fatalf("reopened %+v, %v", reopened, err)
	}
}

func TestStoreProjects(t *testing.T) {
	s, f := newTestStore(t)
	if _, err := s.AddProject(" Work "); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"", "  ", "work", "no project"} {
		if _, err := s.AddProject(name); err == nil {
			t.Errorf("addProject(%q) succeeded", name)
		}
	}

	inside, err := s.Add("a", testKey, "Work")
	if err != nil {
		t.Fatal(err)
	}

	outside, err := s.Add("b", testKey, "")
	if err != nil {
		t.Fatal(err)
	}

	if s.ProjectSize("Work") != 1 {
		t.Errorf("entries %+v", s.Entries)
	}

	if err := s.MoveKey(outside.ID, "Work"); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(f, s.path)
	if err != nil || reopened.ProjectSize("Work") != 2 {
		t.Fatalf("reopened %+v, %v", reopened, err)
	}

	if err := reopened.RemoveProject("Work"); err != nil {
		t.Fatal(err)
	}

	if len(reopened.Projects) != 0 || len(reopened.Entries) != 2 || reopened.Entries[0].Project != "" || reopened.Entries[1].Project != "" {
		t.Errorf("after removing the project: %+v", reopened)
	}

	if got, _ := reopened.Get(inside.ID); got != testKey {
		t.Errorf("secret after removing the project: %q", got)
	}
}

func TestStoreProjectChangesRollBackWhenIndexCannotSave(t *testing.T) {
	s, _ := newTestStore(t)
	if _, err := s.AddProject("Work"); err != nil {
		t.Fatal(err)
	}

	e, err := s.Add("a", testKey, "Work")
	if err != nil {
		t.Fatal(err)
	}

	s.path = filepath.Join(s.path, "missing", "keys.json")
	if _, err := s.AddProject("Other"); err == nil || len(s.Projects) != 1 {
		t.Errorf("addProject: %v, projects %v", err, s.Projects)
	}

	if err := s.MoveKey(e.ID, ""); err == nil || s.Entries[0].Project != "Work" {
		t.Errorf("moveKey: %v, entries %+v", err, s.Entries)
	}

	if err := s.RemoveProject("Work"); err == nil || len(s.Projects) != 1 || s.Entries[0].Project != "Work" {
		t.Errorf("removeProject: %v, projects %v, entries %+v", err, s.Projects, s.Entries)
	}
}

func TestDetectKeyType(t *testing.T) {
	for value, want := range map[string]string{
		testKey:                                 "OpenSSH",
		"-----BEGIN RSA PRIVATE KEY-----":       "RSA",
		"-----BEGIN EC PRIVATE KEY-----":        "EC",
		"-----BEGIN PRIVATE KEY-----":           "PKCS#8",
		"-----BEGIN ENCRYPTED PRIVATE KEY-----": "Encrypted PKCS#8",
		"-----BEGIN PGP PRIVATE KEY BLOCK-----": "PGP",
		"0xdeadbeef":                            "Other",
	} {
		if got := DetectKeyType(value); got != want {
			t.Errorf("DetectKeyType(%q) = %q, want %q", value, got, want)
		}
	}
}

func TestStoreUpdate(t *testing.T) {
	s, f := newTestStore(t)
	e, err := s.Add("Deploy", testKey, "")
	if err != nil {
		t.Fatal(err)
	}

	if err := s.Update(e.ID, " Renamed ", ""); err != nil {
		t.Fatal(err)
	}

	got := s.Entries[0]
	if got.Name != "Renamed" || !got.UpdatedAt.IsZero() || string(f[SecretName(e.ID)]) != testKey {
		t.Errorf("after rename: %+v, secret %q", got, f[SecretName(e.ID)])
	}

	const rotated = "-----BEGIN RSA PRIVATE KEY-----\nnew\n"
	if err := s.Update(e.ID, "Renamed", rotated); err != nil {
		t.Fatal(err)
	}

	got = s.Entries[0]
	if got.ID != e.ID || got.Type != "RSA" || got.UpdatedAt.IsZero() || string(f[SecretName(e.ID)]) != rotated {
		t.Errorf("after rotation: %+v, secret %q", got, f[SecretName(e.ID)])
	}

	reopened, err := Open(f, s.path)
	if err != nil || reopened.Entries[0].Type != "RSA" || reopened.Entries[0].UpdatedAt.IsZero() {
		t.Errorf("reopened %+v, %v", reopened, err)
	}

	for _, c := range [][2]string{{"", testKey}, {"a", strings.Repeat("x", maxKeySize+1)}} {
		if err := s.Update(e.ID, c[0], c[1]); err == nil {
			t.Errorf("update(%q, %d bytes) succeeded", c[0], len(c[1]))
		}
	}

	if string(f[SecretName(e.ID)]) != rotated || s.Update("missing", "a", "") == nil {
		t.Error("bad updates changed the key, or an unknown ID was accepted")
	}
}

func TestStoreUpdateRestoresKeyWhenIndexCannotSave(t *testing.T) {
	s, f := newTestStore(t)
	e, err := s.Add("Deploy", testKey, "")
	if err != nil {
		t.Fatal(err)
	}

	s.path = filepath.Join(s.path, "missing", "keys.json")
	if err := s.Update(e.ID, "Renamed", "new"); err == nil {
		t.Fatal("update succeeded with an unwritable index")
	}

	if string(f[SecretName(e.ID)]) != testKey || s.Entries[0].Name != "Deploy" || s.Entries[0].Type != "OpenSSH" {
		t.Errorf("after failure: %+v, secret %q", s.Entries[0], f[SecretName(e.ID)])
	}
}

func TestImportChecksEveryKeyBeforeAddingAny(t *testing.T) {
	s, f := newTestStore(t)
	index := Index{Keys: []Entry{{ID: "A", Name: "Good"}, {ID: "B", Name: ""}}}
	if n, err := s.ImportKeys(index, map[string]string{"A": testKey, "B": testKey}); err == nil || n != 0 {
		t.Fatalf("import: %d, %v", n, err)
	}

	if len(f) != 0 || len(s.Entries) != 0 {
		t.Errorf("partial import: %v", s.Entries)
	}

	// A project name the store refuses doesn't lose the key.
	index = Index{Projects: []string{NoProject}, Keys: []Entry{{ID: "A", Name: "Good", Project: NoProject}}}
	if _, err := s.ImportKeys(index, map[string]string{"A": testKey}); err != nil {
		t.Fatal(err)
	}

	if s.Entries[0].Project != "" || len(s.Projects) != 0 {
		t.Errorf("entries %+v, projects %v", s.Entries, s.Projects)
	}
}
