package view

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestViewNote(t *testing.T) {
	s, _ := newTestStore(t)

	e, err := s.Add("Deploy", testKey, "")
	if err != nil {
		t.Fatal(err)
	}

	a := newApp(s, func(_ string, done func(error)) { done(nil) })
	tt := ui.NewTester(a.View, 1040, 680)

	click(t, tt, "Show notes")

	if a.noteID != e.ID {
		t.Fatalf("note not shown: %q", tt.Texts())
	}

	// Filled in directly, as in the other dialogs' tests.
	a.noteDraft = "Opens the web server"

	tt.Frame()
	click(t, tt, "Save notes")

	if note, _ := s.Note(e.ID); note != "Opens the web server" {
		t.Errorf("saved note %q", note)
	}

	a.Lock()
	tt.Frame()

	if a.noteID != "" || a.noteDraft != "" || tt.HasText("Opens the web server") {
		t.Error("note still shown after locking")
	}
}
