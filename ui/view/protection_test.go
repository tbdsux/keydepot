package view

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"keydepot/lib/auth"
	"keydepot/lib/vault"
)

func TestFirstLaunchOffersOnlyWhatTheComputerHas(t *testing.T) {
	_, _, _, _, tt := newVaultApp(t, true)
	if !tt.HasText("Touch ID or Mac password") || tt.HasText("Add your first key") {
		t.Errorf("first launch shows %q", tt.Texts())
	}

	_, _, _, _, tt = newVaultApp(t, false)
	if tt.HasText("Touch ID or Mac password") || tt.HasText("Mac password") || !tt.HasText("KeyDepot password") || !tt.HasText("No extra prompt") {
		t.Errorf("without a system prompt: %q", tt.Texts())
	}
}

func TestFirstLaunchSystemPrompt(t *testing.T) {
	a, v, f, prompts, tt := newVaultApp(t, true)
	click(t, tt, "Continue")

	if v.NeedsSetup() || f[vault.KeychainItem] == nil {
		t.Fatalf("after Continue: setup %v, keychain %v", v.NeedsSetup(), f)
	}

	if !tt.HasText("Add your first key") {
		t.Fatalf("no empty state in %q", tt.Texts())
	}

	// Adding needs no prompt in this mode, and neither does a second key.
	click(t, tt, "Add your first key")

	a.draftName, a.draftKey = "Deploy", testKey

	tt.Frame()
	click(t, tt, "Save key")

	if len(*prompts) != 0 || len(a.store.Entries) != 1 {
		t.Fatalf("after adding: prompts %v, entries %v", *prompts, a.store.Entries)
	}

	click(t, tt, "Reveal")

	if len(*prompts) != 1 || !tt.HasText(testKey) {
		t.Fatalf("prompts %v, texts %q", *prompts, tt.Texts())
	}

	// A failed prompt shows nothing, and leaves no data key in memory.
	a.Lock()

	a.authenticate = func(string) error { return auth.ErrCanceled }

	tt.Frame()
	click(t, tt, "Reveal")

	if tt.HasText(testKey) || !v.Locked() {
		t.Errorf("after declining: revealed %v, key in memory %v", tt.HasText(testKey), !v.Locked())
	}
}

func TestNoExtraPromptNeverCallsTheSystem(t *testing.T) {
	a, _, _, prompts, tt := newVaultApp(t, true)
	a.setupMode = vault.ModeNone

	tt.Frame()
	click(t, tt, "Continue")
	click(t, tt, "Add your first key")

	a.draftName, a.draftKey = "Deploy", testKey

	tt.Frame()
	click(t, tt, "Save key")
	click(t, tt, "Reveal")

	if len(*prompts) != 0 || !tt.HasText(testKey) {
		t.Errorf("prompts %v, texts %q", *prompts, tt.Texts())
	}
}

func setUpPassword(t *testing.T, a *App, tt *ui.Tester) string {
	t.Helper()

	a.setupMode = vault.ModePassword

	tt.Frame()
	click(t, tt, "Continue")

	if !tt.HasText("Choose a KeyDepot password") {
		t.Fatalf("no password dialog in %q", tt.Texts())
	}

	a.askValue, a.askConfirm = "short", "short"

	tt.Frame()
	click(t, tt, "Set password")

	if !tt.HasText("Use at least 8 characters.") {
		t.Errorf("short password accepted: %q", tt.Texts())
	}

	a.askValue, a.askConfirm = testPassword, "different password"

	tt.Frame()
	click(t, tt, "Set password")

	if !tt.HasText("The passwords don't match.") {
		t.Errorf("mismatch accepted: %q", tt.Texts())
	}

	a.askValue, a.askConfirm = testPassword, testPassword

	tt.Frame()
	click(t, tt, "Set password")

	return confirmCode(t, a, tt)
}

func confirmCode(t *testing.T, a *App, tt *ui.Tester) string {
	t.Helper()

	code := a.recoveryCode
	if code == "" || !tt.HasText(vault.FormatCode(code)) || !tt.HasText("If you lose both the password and this code, your keys are gone.") {
		t.Fatalf("no recovery code in %q", tt.Texts())
	}

	// It can't be dismissed, or passed without proof it was seen.
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	click(t, tt, "I've saved it")

	if a.recoveryCode == "" {
		t.Fatal("confirmed without the code")
	}

	a.recoveryTyped = strings.ToLower(code[len(code)-confirmCodeLen:])

	tt.Frame()
	click(t, tt, "I've saved it")

	if a.recoveryCode != "" {
		t.Fatalf("not confirmed: %q", tt.Texts())
	}

	return code
}

func TestViewPasswordModeUnlockAndRecovery(t *testing.T) {
	a, v, f, prompts, tt := newVaultApp(t, true)
	code := setUpPassword(t, a, tt)

	if len(f) != 0 {
		t.Errorf("password mode used the Keychain: %v", f)
	}

	click(t, tt, "Add your first key")

	a.draftName, a.draftKey = "Deploy", testKey

	tt.Frame()
	click(t, tt, "Save key")

	if len(a.store.Entries) != 1 || a.adding {
		t.Fatalf("after adding: %+v, adding %v", a.store.Entries, a.adding)
	}

	// Adding while locked asks, over the dialog, and keeps the draft on a wrong try.
	a.Lock()
	tt.Frame()
	click(t, tt, "New key")

	a.draftName, a.draftKey = "Second", testKey

	tt.Frame()
	click(t, tt, "Save key")

	if !tt.HasText("Enter your KeyDepot password") {
		t.Fatalf("no unlock in %q", tt.Texts())
	}

	a.askValue = "wrong password"

	tt.Frame()
	click(t, tt, "Unlock")

	if !tt.HasText("That password isn't right.") || len(a.store.Entries) != 1 {
		t.Errorf("after a wrong password: %q", tt.Texts())
	}

	a.askValue = testPassword

	tt.Frame()
	click(t, tt, "Unlock")

	if len(a.store.Entries) != 2 || a.adding || !a.unlocked {
		t.Fatalf("after unlocking: %+v, adding %v, unlocked %v", a.store.Entries, a.adding, a.unlocked)
	}

	// Canceling the prompt shows nothing, and asks again next time.
	a.Lock()
	tt.Frame()
	click(t, tt, "Reveal")
	click(t, tt, "Cancel")

	if tt.HasText(testKey) || a.authPending || a.ask != nil {
		t.Errorf("after canceling: pending %v, ask %v", a.authPending, a.ask)
	}

	// Recovery, abandoned at the new password, leaves everything as it was.
	click(t, tt, "Reveal")
	click(t, tt, "Forgot password?")

	a.askValue = "NOT THE CODE"

	tt.Frame()
	click(t, tt, "Continue")

	if !tt.HasText("That recovery code isn't right.") {
		t.Errorf("wrong code: %q", tt.Texts())
	}

	a.askValue = vault.FormatCode(code)

	tt.Frame()
	click(t, tt, "Continue")

	if !tt.HasText("Choose a new password") {
		t.Fatalf("no new password: %q", tt.Texts())
	}

	click(t, tt, "Cancel")

	if !v.Locked() || a.unlocked || tt.HasText(testKey) {
		t.Errorf("abandoned recovery left: key %v, unlocked %v", !v.Locked(), a.unlocked)
	}

	// Finished, it sets a new password and a new code.
	click(t, tt, "Reveal")
	click(t, tt, "Forgot password?")

	a.askValue = code

	tt.Frame()
	click(t, tt, "Continue")

	a.askValue, a.askConfirm = "brand new password", "brand new password"

	tt.Frame()
	click(t, tt, "Set password")

	second := confirmCode(t, a, tt)
	if second == code {
		t.Error("recovery kept the code")
	}

	if !a.unlocked || !tt.HasText(testKey) {
		t.Errorf("not revealed after recovery: %q", tt.Texts())
	}

	if len(*prompts) != 0 {
		t.Errorf("password mode used the system prompt: %v", *prompts)
	}
}

func TestViewSettingsChangesProtection(t *testing.T) {
	a, v, f, prompts, tt := newVaultApp(t, true)
	click(t, tt, "Continue")

	if _, err := a.store.Add("Deploy", testKey, ""); err != nil {
		t.Fatal(err)
	}

	click(t, tt, "Settings")

	if !tt.HasText("Current") || tt.HasText("Change password") {
		t.Errorf("settings show %q", tt.Texts())
	}

	// A declined prompt changes nothing.
	a.authenticate = func(string) error { return auth.ErrCanceled }

	click(t, tt, "Use No extra prompt")

	if v.ShownMode() != vault.ModeSystem {
		t.Fatalf("mode %q after declining", v.ShownMode())
	}

	// Changing always asks the current method, even when unlocked.
	a.authenticate = func(r string) error { *prompts = append(*prompts, r); return nil }
	a.unlocked = true

	click(t, tt, "Use No extra prompt")

	if len(*prompts) != 1 || v.ShownMode() != vault.ModeNone {
		t.Fatalf("prompts %v, mode %q", *prompts, v.ShownMode())
	}

	click(t, tt, "Use KeyDepot password")

	a.askValue, a.askConfirm = testPassword, testPassword

	tt.Frame()
	click(t, tt, "Set password")

	first := confirmCode(t, a, tt)

	if !v.PasswordProtected() || f[vault.KeychainItem] != nil {
		t.Fatalf("password %v, keychain %v", v.PasswordProtected(), f)
	}

	click(t, tt, "Change password")

	a.askValue = testPassword

	tt.Frame()
	click(t, tt, "Unlock")

	a.askValue, a.askConfirm = "brand new password", "brand new password"

	tt.Frame()
	click(t, tt, "Set password")

	changed := confirmCode(t, a, tt)
	if a.ask != nil || a.vault.UnlockPassword("brand new password") != nil || changed == first {
		t.Fatalf("still asking %q, or the code didn't change", tt.Texts())
	}

	click(t, tt, "New recovery code")

	a.askValue = "wrong password"

	tt.Frame()
	click(t, tt, "Continue")

	if !tt.HasText(vault.ErrWrongPassword.Error()) {
		t.Errorf("wrong password: %q", tt.Texts())
	}

	a.askValue = "brand new password"

	tt.Frame()
	click(t, tt, "Continue")

	if second := confirmCode(t, a, tt); second == changed {
		t.Error("the code didn't change")
	}

	click(t, tt, "Use Touch ID or Mac password")

	a.askValue = "brand new password"

	tt.Frame()
	click(t, tt, "Unlock")

	if v.PasswordProtected() || v.ShownMode() != vault.ModeSystem || f[vault.KeychainItem] == nil {
		t.Fatalf("after leaving password mode: %q, %v", v.ShownMode(), f)
	}

	if got, _ := a.store.Get(a.store.Entries[0].ID); got != testKey {
		t.Errorf("key after every switch: %q", got)
	}
}

func TestViewExportAndImport(t *testing.T) {
	a, _, _, prompts, tt := newVaultApp(t, true)
	click(t, tt, "Continue")
	click(t, tt, "Settings")
	click(t, tt, "Export keys")

	if !tt.HasText("There are no keys to export.") && len(*prompts) != 0 {
		t.Errorf("empty export: prompts %v", *prompts)
	}

	if _, err := a.store.Add("Deploy", testKey, ""); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "out.keydepot")
	a.pickSave = func() (string, error) { return path, nil }
	a.pickOpen = func() (string, error) { return path, nil }

	a.unlocked = true

	click(t, tt, "Export keys")

	if len(*prompts) != 1 {
		t.Errorf("export didn't ask again: %v", *prompts)
	}

	a.askValue, a.askConfirm = "only eleven", "only eleven"

	tt.Frame()
	click(t, tt, "Export")

	if !tt.HasText("Use at least 12 characters.") {
		t.Errorf("weak export password accepted: %q", tt.Texts())
	}

	a.askValue, a.askConfirm = exportPass, exportPass

	tt.Frame()
	click(t, tt, "Export")

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("no export: %v, %q", err, tt.Texts())
	}

	click(t, tt, "Import keys")

	a.askValue = "wrong password"

	tt.Frame()
	click(t, tt, "Import")

	if !tt.HasText("That password isn't right.") || len(a.store.Entries) != 1 {
		t.Errorf("wrong password: %q", tt.Texts())
	}

	a.askValue = exportPass

	tt.Frame()
	click(t, tt, "Import")

	if len(a.store.Entries) != 2 {
		t.Fatalf("after import: %+v", a.store.Entries)
	}

	if got, _ := a.store.Get(a.store.Entries[1].ID); got != testKey || a.store.Entries[0].ID == a.store.Entries[1].ID {
		t.Errorf("imported %q, entries %+v", got, a.store.Entries)
	}

	// A canceled file dialog does nothing.
	a.pickSave = func() (string, error) { return "", nil }

	click(t, tt, "Export keys")

	if a.ask != nil {
		t.Errorf("asked after canceling the file dialog: %q", tt.Texts())
	}
}

// Renders each new screen for review when KEYDEPOT_SHOTS names a folder.
func TestRenderScreens(t *testing.T) {
	dir := os.Getenv("KEYDEPOT_SHOTS")
	if dir == "" {
		t.Skip("KEYDEPOT_SHOTS isn't set")
	}

	shot := func(tt *ui.Tester, name string) {
		t.Helper()

		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}

		defer f.Close()

		tt.Advance(1e9)
		tt.Frame()

		if err := png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
	}

	a, _, _, _, tt := newVaultApp(t, true)
	shot(tt, "setup")

	a.setupMode = vault.ModePassword

	tt.Frame()
	click(t, tt, "Continue")

	a.askValue, a.askConfirm = "short", "short"

	shot(tt, "new-password")

	a.askValue, a.askConfirm = testPassword, testPassword

	tt.Frame()
	click(t, tt, "Set password")
	shot(tt, "recovery-code")

	a.recoveryTyped = a.recoveryCode[len(a.recoveryCode)-confirmCodeLen:]

	tt.Frame()
	click(t, tt, "I've saved it")

	if _, err := a.store.Add("Production deploy", testKey, ""); err != nil {
		t.Fatal(err)
	}

	click(t, tt, "Settings")
	shot(tt, "settings")
	click(t, tt, "Keys")
	a.Lock()
	tt.Frame()
	click(t, tt, "Reveal")
	shot(tt, "unlock")
}
