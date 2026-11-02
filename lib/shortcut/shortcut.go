// Package shortcut parses, checks and saves keyboard shortcuts.
package shortcut

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Shortcut is a key and the exact modifiers it takes; the zero value is none.
type Shortcut struct {
	Mods ui.Modifiers
	Key  ui.Key
}

// keyNames lists the keys a shortcut can use, by the name saved for them.
var keyNames = func() map[ui.Key]string {
	names := map[ui.Key]string{
		ui.KeyEnter: "Enter", ui.KeyBackspace: "Backspace", ui.KeyDelete: "Delete",
		ui.KeySpace: "Space", ui.KeyTab: "Tab",
		ui.KeyHome: "Home", ui.KeyEnd: "End", ui.KeyPageUp: "PageUp", ui.KeyPageDown: "PageDown",
		ui.KeyLeft: "Left", ui.KeyRight: "Right", ui.KeyUp: "Up", ui.KeyDown: "Down",
		ui.KeyMinus: "-", ui.KeyEqual: "=", ui.KeyComma: ",", ui.KeyPeriod: ".", ui.KeySlash: "/", ui.KeySemicolon: ";",
		ui.KeyQuote: "'", ui.KeyBracketLeft: "[", ui.KeyBracketRight: "]", ui.KeyBackslash: "\\", ui.KeyBackquote: "`",
	}

	for i := range 26 {
		names[ui.KeyA+ui.Key(i)] = string(rune('A' + i))
	}

	for i := range 10 {
		names[ui.Key0+ui.Key(i)] = string(rune('0' + i))
	}

	for i := range 12 {
		names[ui.KeyF1+ui.Key(i)] = fmt.Sprintf("F%d", i+1)
	}

	return names
}()

// modNames is in the order macOS's menus show modifiers.
var modNames = []struct {
	mod          ui.Modifiers
	name, symbol string
}{{ui.Ctrl, "Ctrl", "⌃"}, {ui.Alt, "Alt", "⌥"}, {ui.Shift, "Shift", "⇧"}, {ui.Super, "Super", "⌘"}}

// String is how a shortcut is saved, as "Super+Shift+K".
func (s Shortcut) String() string {
	var parts []string

	for _, m := range modNames {
		if s.Mods&m.mod != 0 {
			parts = append(parts, m.name)
		}
	}

	return strings.Join(append(parts, keyNames[s.Key]), "+")
}

// Parse reads what String writes, reporting whether it could.
func Parse(text string) (Shortcut, bool) {
	parts := strings.Split(text, "+")

	var s Shortcut

	for _, part := range parts[:len(parts)-1] {
		known := false

		for _, m := range modNames {
			if m.name == part {
				s.Mods, known = s.Mods|m.mod, true
			}
		}

		if !known {
			return Shortcut{}, false
		}
	}

	for key, name := range keyNames {
		if name == parts[len(parts)-1] {
			s.Key = key
			return s, true
		}
	}

	return Shortcut{}, false
}

var keySymbols = map[ui.Key]string{
	ui.KeyEnter: "↩", ui.KeyBackspace: "⌫", ui.KeyDelete: "⌦", ui.KeyTab: "⇥",
	ui.KeyLeft: "←", ui.KeyRight: "→", ui.KeyUp: "↑", ui.KeyDown: "↓",
}

// Label is how the interface shows a shortcut: ⇧⌘K on macOS, Ctrl+Shift+K elsewhere.
func (s Shortcut) Label() string {
	if s.Key == ui.KeyUnknown {
		return "Not set"
	}

	if runtime.GOOS != "darwin" {
		return s.String()
	}

	var b strings.Builder

	for _, m := range modNames {
		if s.Mods&m.mod != 0 {
			b.WriteString(m.symbol)
		}
	}

	if symbol, ok := keySymbols[s.Key]; ok {
		b.WriteString(symbol)
	} else {
		b.WriteString(keyNames[s.Key])
	}

	return b.String()
}

// Problem says why a shortcut can't be used, if so.
func (s Shortcut) Problem() string {
	if s.Key >= ui.KeyF1 && s.Key <= ui.KeyF12 {
		return ""
	}

	// Without these, the key would type into text fields instead.
	if s.Mods&(ui.Super|ui.Ctrl) == 0 {
		if runtime.GOOS == "darwin" {
			return "Include ⌘ or ⌃, so the shortcut doesn't type text."
		}

		return "Include Ctrl or Super, so the shortcut doesn't type text."
	}

	reserved := s == Shortcut{ui.Cmd | ui.Shift, ui.KeyZ}

	if s.Mods == ui.Cmd {
		switch s.Key {
		// Text fields' editing keys, and what the system's menus keep.
		case ui.KeyA, ui.KeyC, ui.KeyV, ui.KeyX, ui.KeyY, ui.KeyZ, ui.KeyQ, ui.KeyW, ui.KeyH, ui.KeyM, ui.KeyEqual, ui.KeyMinus, ui.Key0:
			reserved = true
		}
	}

	if reserved {
		return s.Label() + " is kept for the system."
	}

	return ""
}

// Known reports whether a shortcut can use key.
func Known(key ui.Key) bool {
	_, ok := keyNames[key]

	return ok
}
