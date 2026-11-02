package view

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"keydepot/lib/shortcut"
	"keydepot/lib/store"
)

func commandByID(t *testing.T, id string) command {
	t.Helper()

	for _, cmd := range commands {
		if cmd.id == id {
			return cmd
		}
	}

	t.Fatalf("no command %q", id)

	return command{}
}

func TestDefaultShortcutsAreUsable(t *testing.T) {
	seen := map[shortcut.Shortcut]string{}

	for _, cmd := range commands {
		if problem := cmd.fallback.Problem(); problem != "" {
			t.Errorf("%s: %s", cmd.id, problem)
		}

		if other, dup := seen[cmd.fallback]; dup {
			t.Errorf("%s and %s share %s", cmd.id, other, cmd.fallback)
		}

		seen[cmd.fallback] = cmd.id
	}
}

func TestSetShortcutRefusesWhatCantWork(t *testing.T) {
	a := newApp(nil, nil)
	cmd := commandByID(t, "copy")

	for name, s := range map[string]shortcut.Shortcut{
		"typing":  shortcut.Shortcut{Mods: 0, Key: ui.KeyJ},
		"shifted": shortcut.Shortcut{Mods: ui.Shift, Key: ui.KeyJ},
		"alt":     shortcut.Shortcut{Mods: ui.Alt, Key: ui.KeyJ},
		"paste":   shortcut.Shortcut{Mods: ui.Cmd, Key: ui.KeyV},
		"quit":    shortcut.Shortcut{Mods: ui.Cmd, Key: ui.KeyQ},
		"taken":   commandByID(t, "newKey").fallback,
	} {
		if err := a.setShortcut(cmd, s); err == nil {
			t.Errorf("%s: %v accepted", name, s)
		}
	}

	if len(a.shortcuts) != 0 {
		t.Errorf("refused shortcuts kept: %v", a.shortcuts)
	}

	if err := a.setShortcut(cmd, shortcut.Shortcut{Mods: 0, Key: ui.KeyF5}); err != nil {
		t.Errorf("a function key alone: %v", err)
	}

	if err := a.setShortcut(cmd, cmd.fallback); err != nil || len(a.shortcuts) != 0 {
		t.Errorf("reset: %v, left %v", err, a.shortcuts)
	}
}

func TestShortcutsPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shortcuts.json")
	if got, err := shortcut.Load(path); err != nil || len(got) != 0 {
		t.Fatalf("no file: %v, %v", got, err)
	}

	a := newApp(nil, nil)
	a.shortcutsPath = path
	if err := a.setShortcut(commandByID(t, "copy"), shortcut.Shortcut{Mods: ui.Cmd, Key: ui.KeyJ}); err != nil {
		t.Fatal(err)
	}

	if err := a.setShortcut(commandByID(t, "lock"), shortcut.Shortcut{}); err != nil {
		t.Fatal(err)
	}

	got, err := shortcut.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if got["copy"] != (shortcut.Shortcut{Mods: ui.Cmd, Key: ui.KeyJ}) {
		t.Errorf("copy reloaded as %v", got["copy"])
	}

	if s, ok := got["lock"]; !ok || s != (shortcut.Shortcut{}) {
		t.Errorf("removed shortcut reloaded as %v, %v", s, ok)
	}

	if err := os.WriteFile(path, []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got, err := shortcut.Load(path); err == nil || got == nil {
		t.Errorf("corrupt file: %v, %v", got, err)
	}
}

func paletteApp(t *testing.T) (*App, *store.Store, *ui.Tester) {
	t.Helper()

	s, _ := newTestStore(t)
	if _, err := s.AddProject("Work"); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"Alpha", "Beta"} {
		if _, err := s.Add(name, testKey, ""); err != nil {
			t.Fatal(err)
		}
	}

	a := newApp(s, func(_ string, done func(error)) { done(nil) })

	return a, s, ui.NewTester(a.View, 1040, 680)
}

func TestPaletteRunsWhatItFinds(t *testing.T) {
	a, _, tt := paletteApp(t)
	tt.Key(ui.Cmd, ui.KeyK)

	if !a.paletteOpen || !tt.HasText("New key") {
		t.Fatalf("palette closed: %q", tt.Texts())
	}

	// Choosing a key goes to it.
	a.paletteQuery = "beta"

	tt.Frame()

	if tt.HasText("Lock KeyDepot") {
		t.Errorf("filter left a command in: %q", tt.Texts())
	}

	tt.Key(0, ui.KeyEnter)

	if a.paletteOpen || a.paletteQuery != "" || a.chosen != 1 {
		t.Errorf("after Enter: open %v, query %q, chosen %d", a.paletteOpen, a.paletteQuery, a.chosen)
	}

	// Arrows wrap around.
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Key(0, ui.KeyUp)

	if a.paletteChosen == 0 {
		t.Error("Up from the first row stayed there")
	}

	tt.Key(0, ui.KeyDown)

	if a.paletteChosen != 0 {
		t.Errorf("Down from the last row went to %d", a.paletteChosen)
	}

	// Nothing found: Enter does nothing and the palette stays.
	a.paletteQuery = "zzzz"

	tt.Frame()

	if !tt.HasText("No results") {
		t.Errorf("no empty state: %q", tt.Texts())
	}

	tt.Key(0, ui.KeyEnter)

	if !a.paletteOpen {
		t.Error("Enter on no results closed the palette")
	}

	// A click runs a row, and its own shortcut closes it.
	a.paletteQuery = ""

	tt.Frame()

	if err := tt.Click("Open settings"); err != nil {
		t.Fatal(err)
	}

	if a.paletteOpen || a.page != "settings" {
		t.Errorf("after click: open %v, page %q", a.paletteOpen, a.page)
	}

	tt.Key(ui.Cmd, ui.KeyK)
	tt.Key(ui.Cmd, ui.KeyK)

	if a.paletteOpen {
		t.Error("the palette's shortcut didn't close it")
	}
}

func TestPaletteSearchFocusesField(t *testing.T) {
	a, _, tt := paletteApp(t)
	tt.Key(ui.Cmd, ui.KeyK)

	a.paletteQuery = "search keys"

	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.TypeKey(0, ui.KeyB, "b")
	tt.Frame()

	if a.paletteOpen || a.query != "b" {
		t.Errorf("open %v, query %q", a.paletteOpen, a.query)
	}
}

func TestShortcutsRespectContext(t *testing.T) {
	a, _, tt := paletteApp(t)
	tt.Key(ui.Cmd, ui.KeyN)

	if !a.adding {
		t.Fatal("new key shortcut did nothing")
	}

	// Behind a dialog, shortcuts wait.
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Key(ui.Cmd, ui.KeyComma)

	if a.paletteOpen || a.page == "settings" {
		t.Errorf("ran behind a dialog: palette %v, page %q", a.paletteOpen, a.page)
	}

	tt.Key(0, ui.KeyEscape)

	// Reveal goes through the same prompt as the button, and Settings shows no key to act on.
	tt.Key(ui.Cmd|ui.Shift, ui.KeyR)

	if a.revealed != testKey {
		t.Errorf("reveal shortcut: %q", a.revealed)
	}

	tt.Key(ui.Cmd|ui.Shift, ui.KeyR)

	if a.revealed != "" {
		t.Error("reveal shortcut didn't hide")
	}

	tt.Key(ui.Cmd, ui.KeyComma)
	tt.Key(ui.Cmd, ui.KeyBackspace)
	tt.Key(ui.Cmd, ui.KeyE)

	if a.confirmDelete || a.adding {
		t.Errorf("acted on an unseen key: delete %v, update %v", a.confirmDelete, a.adding)
	}

	// Locking asks again.
	a.unlocked = true

	tt.Key(ui.Cmd, ui.KeyL)

	if a.unlocked {
		t.Error("lock shortcut left it unlocked")
	}
}

func TestChangeShortcutInSettings(t *testing.T) {
	a, _, tt := paletteApp(t)
	a.page = "settings"

	tt.Frame()

	if err := tt.Click("Change shortcut for " + commandByID(t, "palette").title); err != nil {
		t.Fatal(err)
	}

	if !a.recordOpen || a.recordingID != "palette" {
		t.Fatalf("not recording: %v, %q", a.recordOpen, a.recordingID)
	}

	tt.Key(0, ui.KeyJ)

	if !a.recordOpen || a.recordErr == "" {
		t.Errorf("a typing key: open %v, error %q", a.recordOpen, a.recordErr)
	}

	tt.Key(ui.Cmd, ui.KeyN)

	if !a.recordOpen || a.recordErr == "" {
		t.Errorf("a taken key: open %v, error %q", a.recordOpen, a.recordErr)
	}

	tt.Key(ui.Cmd, ui.KeyJ)

	if a.recordOpen || a.shortcuts["palette"] != (shortcut.Shortcut{Mods: ui.Cmd, Key: ui.KeyJ}) {
		t.Fatalf("a good key: open %v, %v", a.recordOpen, a.shortcuts)
	}

	tt.Key(ui.Cmd, ui.KeyK)

	if a.paletteOpen {
		t.Error("the old shortcut still works")
	}

	tt.Key(ui.Cmd, ui.KeyJ)

	if !a.paletteOpen {
		t.Error("the new shortcut doesn't work")
	}

	tt.Key(0, ui.KeyEscape)

	// Escape cancels, and a removed shortcut shows as such and does nothing.
	a.recordOpen, a.recordingID = true, "lock"

	tt.Frame()
	tt.Key(0, ui.KeyEscape)

	if a.recordOpen || len(a.shortcuts) != 1 {
		t.Errorf("after Escape: open %v, %v", a.recordOpen, a.shortcuts)
	}

	a.recordOpen, a.recordingID = true, "lock"

	tt.Frame()

	if err := tt.Click("Remove shortcut"); err != nil {
		t.Fatal(err)
	}

	if !tt.HasText("Not set") {
		t.Errorf("removed shortcut: %q", tt.Texts())
	}

	if err := tt.Click("Reset " + commandByID(t, "lock").title); err != nil {
		t.Fatal(err)
	}

	if tt.HasText("Not set") {
		t.Errorf("reset shortcut: %q", tt.Texts())
	}
}
