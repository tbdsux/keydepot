package vault

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"keydepot/lib/crypt"
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

const (
	testPassword = "correct horse"
	exportPass   = "a long export password"
)

func newTestVault(t *testing.T, mode string) (*Vault, fakeSecrets) {
	t.Helper()

	f := fakeSecrets{}
	v, err := Open(f, filepath.Join(t.TempDir(), "vault.keydepot"))
	if err != nil {
		t.Fatal(err)
	}

	if !v.NeedsSetup() {
		t.Fatal("new vault doesn't need setup")
	}

	if _, err := v.Protect(mode, testPassword); err != nil {
		t.Fatal(err)
	}

	return v, f
}

func mustSet(t *testing.T, v *Vault, name, value string) {
	t.Helper()

	if err := v.Set(name, []byte(value)); err != nil {
		t.Fatal(err)
	}
}

func wantValue(t *testing.T, v *Vault, name, want string) {
	t.Helper()

	got, err := v.Get(name)
	if err != nil || string(got) != want {
		t.Errorf("Get(%q) = %q, %v, want %q", name, got, err, want)
	}
}

func editFile(t *testing.T, path string, edit func(*File)) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var f File
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

func TestVaultEncryptsOnDiskAndReopens(t *testing.T) {
	for _, mode := range []string{ModeSystem, ModeNone} {
		v, f := newTestVault(t, mode)
		mustSet(t, v, "key.A", testKey)

		data, err := os.ReadFile(v.path)
		if err != nil {
			t.Fatal(err)
		}

		if strings.Contains(string(data), "BEGIN") {
			t.Errorf("%s: vault holds the key in the clear", mode)
		}

		if _, err := os.Stat(v.path + ".tmp"); err == nil {
			t.Errorf("%s: temp file left behind", mode)
		}

		reopened, err := Open(f, v.path)
		if err != nil || reopened.NeedsSetup() {
			t.Fatalf("%s: reopened %v, %v", mode, reopened, err)
		}

		got, err := reopened.Load()
		if err != nil || got != mode {
			t.Errorf("load = %q, %v, want %q", got, err, mode)
		}

		wantValue(t, reopened, "key.A", testKey)

		if err := reopened.Delete("key.A"); err != nil {
			t.Fatal(err)
		}

		if _, err := reopened.Get("key.A"); err == nil {
			t.Error("deleted key still reads")
		}
	}
}

func TestVaultHoldsKeysTheKeychainLimitWouldNot(t *testing.T) {
	v, _ := newTestVault(t, ModeNone)
	large := strings.Repeat("x", 16*1024)
	mustSet(t, v, "key.A", large)
	wantValue(t, v, "key.A", large)
}

func TestVaultRefusesSwappedOrChangedValues(t *testing.T) {
	v, f := newTestVault(t, ModeNone)
	mustSet(t, v, "key.A", "aaa")
	mustSet(t, v, "key.B", "bbb")

	editFile(t, v.path, func(file *File) {
		file.Items["key.A"], file.Items["key.B"] = file.Items["key.B"], file.Items["key.A"]
	})

	swapped, err := Open(f, v.path)
	if err != nil {
		t.Fatal(err)
	}

	if got, err := swapped.Get("key.A"); err == nil {
		t.Errorf("swapped value read as %q", got)
	}

	editFile(t, v.path, func(file *File) { file.Items["key.B"][20] ^= 1 })

	changed, err := Open(f, v.path)
	if err != nil {
		t.Fatal(err)
	}

	if got, err := changed.Get("key.B"); err == nil {
		t.Errorf("changed value read as %q", got)
	}
}

func TestVaultRefusesUnreadableFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.keydepot")

	for _, content := range []string{"{oops", `{"version":99,"items":{}}`, `{"version":0}`} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		if _, err := Open(fakeSecrets{}, path); err == nil {
			t.Errorf("opened %q", content)
		}
	}
}

func TestMissingKeychainItemFailsClosed(t *testing.T) {
	v, f := newTestVault(t, ModeSystem)
	mustSet(t, v, "key.A", testKey)
	delete(f, KeychainItem)

	reopened, err := Open(f, v.path)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := reopened.Load(); err == nil {
		t.Error("loaded without the Keychain item")
	}

	if _, err := reopened.Get("key.A"); err == nil {
		t.Error("read a key without the Keychain item")
	}

	if reopened.NeedsSetup() {
		t.Error("a vault with keys went back to setup, inviting a new key that opens nothing")
	}
}

func TestEmptyVaultWithoutKeychainItemChoosesAgain(t *testing.T) {
	v, f := newTestVault(t, ModeSystem)
	delete(f, KeychainItem)

	reopened, err := Open(f, v.path)
	if err != nil {
		t.Fatal(err)
	}

	reopened.Load()

	if !reopened.NeedsSetup() {
		t.Error("empty vault with no key doesn't offer setup")
	}
}

// The mode comes from the Keychain: editing the file can't turn the prompt off.
func TestEditingTheFileCannotDowngrade(t *testing.T) {
	v, f := newTestVault(t, ModeSystem)
	editFile(t, v.path, func(file *File) { file.Mode = ModeNone })

	reopened, err := Open(f, v.path)
	if err != nil {
		t.Fatal(err)
	}

	if got, err := reopened.Load(); err != nil || got != ModeSystem {
		t.Errorf("load = %q, %v", got, err)
	}
}

func TestPasswordMode(t *testing.T) {
	f := fakeSecrets{}
	v, err := Open(f, filepath.Join(t.TempDir(), "vault.keydepot"))
	if err != nil {
		t.Fatal(err)
	}

	code, err := v.Protect(ModePassword, testPassword)
	if err != nil || code == "" {
		t.Fatalf("protect: %q, %v", code, err)
	}

	if len(f) != 0 {
		t.Errorf("password mode used the Keychain: %v", f)
	}

	mustSet(t, v, "key.A", testKey)

	reopen := func() *Vault {
		t.Helper()

		r, err := Open(f, v.path)
		if err != nil {
			t.Fatal(err)
		}

		return r
	}

	r := reopen()
	if _, err := r.Get("key.A"); !errors.Is(err, ErrLocked) {
		t.Errorf("Get while locked: %v", err)
	}

	if err := r.Set("key.B", []byte("x")); !errors.Is(err, ErrLocked) {
		t.Errorf("Set while locked: %v", err)
	}

	if err := r.UnlockPassword("wrong"); !errors.Is(err, ErrWrongPassword) {
		t.Errorf("wrong password: %v", err)
	}

	if err := r.UnlockRecovery("WRONG"); !errors.Is(err, errWrongCode) {
		t.Errorf("wrong code: %v", err)
	}

	if err := r.UnlockPassword(testPassword); err != nil {
		t.Fatal(err)
	}

	wantValue(t, r, "key.A", testKey)

	// A copy of the vault from before each change, with the secret that
	// opened it, must open nothing saved after.
	before, err := os.ReadFile(v.path)
	if err != nil {
		t.Fatal(err)
	}

	// Changing the password replaces the code, and the key under both.
	changed, err := r.Protect(ModePassword, "a new password")
	if err != nil {
		t.Fatal(err)
	}

	mustSet(t, r, "key.B", "after the change")

	r = reopen()
	if err := r.UnlockPassword(testPassword); err == nil {
		t.Error("old password still works")
	}

	if err := r.UnlockRecovery(code); err == nil {
		t.Error("old code works after changing the password")
	}

	if err := r.UnlockRecovery(strings.ToLower(FormatCode(changed))); err != nil {
		t.Errorf("new code, as a person might type it: %v", err)
	}

	wantValue(t, r, "key.A", testKey)
	wantOldCopyOpensNothingNew(t, before, testPassword, r.file.Items["key.B"], "key.B")

	// A new code ends the old one, and needs the password.
	if _, err := r.RenewRecoveryCode("wrong"); !errors.Is(err, ErrWrongPassword) {
		t.Errorf("renew with the wrong password: %v", err)
	}

	if before, err = os.ReadFile(v.path); err != nil {
		t.Fatal(err)
	}

	second, err := r.RenewRecoveryCode("a new password")
	if err != nil {
		t.Fatal(err)
	}

	r = reopen()
	if err := r.UnlockRecovery(changed); err == nil {
		t.Error("old code works after making a new one")
	}

	if err := r.UnlockRecovery(second); err != nil {
		t.Fatal(err)
	}

	mustSet(t, r, "key.C", "after the renewal")
	wantOldCopyOpensNothingNew(t, before, "a new password", r.file.Items["key.C"], "key.C")

	// Stopping after the code changes nothing: the old password and code work.
	r.Lock()

	r = reopen()
	if err := r.UnlockRecovery(second); err != nil {
		t.Errorf("code after an interrupted recovery: %v", err)
	}

	r.Lock()

	if err := r.UnlockPassword("a new password"); err != nil {
		t.Errorf("password after an interrupted recovery: %v", err)
	}

	// Finishing recovery ends the old password and code.
	if err := r.UnlockRecovery(second); err != nil {
		t.Fatal(err)
	}

	third, err := r.Protect(ModePassword, "another password")
	if err != nil {
		t.Fatal(err)
	}

	wantValue(t, r, "key.A", testKey)

	r = reopen()
	if err := r.UnlockRecovery(second); err == nil {
		t.Error("code works after recovery used it")
	}

	if err := r.UnlockPassword("a new password"); err == nil {
		t.Error("old password works after recovery")
	}

	if err := r.UnlockRecovery(third); err != nil {
		t.Errorf("new code: %v", err)
	}

	wantValue(t, r, "key.A", testKey)
}

// wantOldCopyOpensNothingNew unlocks an old copy of the vault with its old
// password, and checks that the key it gives can't open sealed.
func wantOldCopyOpensNothingNew(t *testing.T, old []byte, password string, sealed []byte, name string) {
	t.Helper()

	f, err := ParseFile(old)
	if err != nil {
		t.Fatal(err)
	}

	oldKey, err := crypt.Unwrap(f.Password, password, crypt.RolePassword)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := crypt.Unseal(oldKey, sealed, name); err == nil {
		t.Errorf("an old copy of the vault opens %s", name)
	}
}

type failingSecrets struct {
	fakeSecrets
	set, del bool
}

func (f *failingSecrets) Set(k string, v []byte) error {
	if f.set {
		return errors.New("keychain says no")
	}

	return f.fakeSecrets.Set(k, v)
}

func (f *failingSecrets) Delete(k string) error {
	if f.del {
		return errors.New("keychain says no")
	}

	return f.fakeSecrets.Delete(k)
}

// A crash between the vault write and removing the Keychain item.
func TestIntoPasswordInterruptedBeforeKeychainDelete(t *testing.T) {
	f := &failingSecrets{fakeSecrets: fakeSecrets{}, del: true}
	v, err := Open(f, filepath.Join(t.TempDir(), "vault.keydepot"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := v.Protect(ModeSystem, ""); err != nil {
		t.Fatal(err)
	}

	mustSet(t, v, "key.A", testKey)

	stale := append([]byte(nil), f.fakeSecrets[KeychainItem]...)

	if _, err := v.Protect(ModePassword, testPassword); err != nil {
		t.Fatal(err)
	}

	f.del = false
	if f.fakeSecrets[KeychainItem] == nil {
		t.Fatal("test needs the item to survive")
	}

	r, err := Open(f, v.path)
	if err != nil {
		t.Fatal(err)
	}

	if got := r.ShownMode(); got != ModePassword {
		t.Errorf("mode %q", got)
	}

	// The stale item must open nothing: the data key was replaced.
	if _, err := r.Get("key.A"); !errors.Is(err, ErrLocked) {
		t.Errorf("Get with a stale item: %v", err)
	}

	var item keychainItem
	if err := json.Unmarshal(stale, &item); err != nil {
		t.Fatal(err)
	}

	r.dataKey = item.Key
	if _, err := r.Get("key.A"); err == nil {
		t.Error("the old data key still decrypts")
	}

	r.Lock()

	if err := r.UnlockPassword(testPassword); err != nil {
		t.Fatal(err)
	}

	if _, ok := f.fakeSecrets[KeychainItem]; ok {
		t.Error("unlocking left the stale item")
	}

	wantValue(t, r, "key.A", testKey)
}

func blockWrites(t *testing.T, v *Vault) (restore func()) {
	t.Helper()

	path := v.path
	v.path = filepath.Join(path, "missing", "vault.keydepot")

	return func() { v.path = path }
}

func TestIntoPasswordWriteFailureChangesNothing(t *testing.T) {
	v, f := newTestVault(t, ModeSystem)
	mustSet(t, v, "key.A", testKey)

	restore := blockWrites(t, v)

	if _, err := v.Protect(ModePassword, testPassword); err == nil {
		t.Fatal("protect succeeded with an unwritable vault")
	}

	restore()
	wantValue(t, v, "key.A", testKey)

	r, err := Open(f, v.path)
	if err != nil {
		t.Fatal(err)
	}

	if got, err := r.Load(); err != nil || got != ModeSystem {
		t.Errorf("load = %q, %v", got, err)
	}

	wantValue(t, r, "key.A", testKey)
}

// The Keychain item is written, then the crash: password mode must still win.
func TestOutOfPasswordInterruptedKeepsPassword(t *testing.T) {
	v, f := newTestVault(t, ModePassword)
	mustSet(t, v, "key.A", testKey)

	restore := blockWrites(t, v)

	if _, err := v.Protect(ModeSystem, ""); err == nil {
		t.Fatal("protect succeeded with an unwritable vault")
	}

	restore()

	if _, ok := f[KeychainItem]; !ok {
		t.Fatal("test needs the item to be written first")
	}

	r, err := Open(f, v.path)
	if err != nil {
		t.Fatal(err)
	}

	if !r.PasswordProtected() {
		t.Fatal("interrupted switch left password mode")
	}

	if _, err := r.Get("key.A"); !errors.Is(err, ErrLocked) {
		t.Errorf("Get: %v", err)
	}

	if err := r.UnlockPassword(testPassword); err != nil {
		t.Fatal(err)
	}

	wantValue(t, r, "key.A", testKey)

	if _, ok := f[KeychainItem]; ok {
		t.Error("stale item survived unlocking")
	}
}

func TestSwitchingModesKeepsKeys(t *testing.T) {
	v, f := newTestVault(t, ModeSystem)
	mustSet(t, v, "key.A", testKey)

	for _, mode := range []string{ModeNone, ModePassword, ModeSystem, ModePassword, ModeNone} {
		if _, err := v.Protect(mode, testPassword); err != nil {
			t.Fatalf("into %s: %v", mode, err)
		}

		r, err := Open(f, v.path)
		if err != nil {
			t.Fatal(err)
		}

		if mode == ModePassword {
			if err := r.UnlockPassword(testPassword); err != nil {
				t.Fatal(err)
			}

			if _, ok := f[KeychainItem]; ok {
				t.Error("password mode left the data key in the Keychain")
			}
		} else if got, err := r.Load(); err != nil || got != mode {
			t.Errorf("load = %q, %v, want %q", got, err, mode)
		}

		wantValue(t, r, "key.A", testKey)
	}
}

func TestKeychainFailureLeavesTheVaultAlone(t *testing.T) {
	f := &failingSecrets{fakeSecrets: fakeSecrets{}}
	v, err := Open(f, filepath.Join(t.TempDir(), "vault.keydepot"))
	if err != nil {
		t.Fatal(err)
	}

	f.set = true

	if _, err := v.Protect(ModeSystem, ""); err == nil {
		t.Fatal("protect succeeded without the Keychain")
	}

	if !v.NeedsSetup() {
		t.Error("setup finished without the Keychain")
	}
}

func TestLeavingPasswordModeReplacesTheKey(t *testing.T) {
	v, _ := newTestVault(t, ModePassword)
	mustSet(t, v, "key.A", testKey)

	before, err := os.ReadFile(v.path)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := v.Protect(ModeSystem, ""); err != nil {
		t.Fatal(err)
	}

	mustSet(t, v, "key.B", "after leaving")
	wantValue(t, v, "key.A", testKey)
	wantOldCopyOpensNothingNew(t, before, testPassword, v.file.Items["key.B"], "key.B")
}

func TestWritesWhileLockedKeepNoKey(t *testing.T) {
	v, _ := newTestVault(t, ModeSystem)
	v.Lock()

	mustSet(t, v, "key.A", testKey)

	if !v.Locked() {
		t.Error("a write left the data key in memory")
	}

	wantValue(t, v, "key.A", testKey)

	if !v.Locked() {
		t.Error("a read left the data key in memory")
	}

	if _, err := v.Load(); err != nil || v.Locked() {
		t.Errorf("load: %v, locked %v", err, v.Locked())
	}
}
