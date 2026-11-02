package view

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"keydepot/lib/auth"
	"keydepot/lib/store"
)

// The view runs without a window in tests, which click and type as a user
// would.
func TestViewAddRevealDelete(t *testing.T) {
	s, f := newTestStore(t)
	a := newApp(s, func(_ string, done func(error)) { done(nil) })
	tt := ui.NewTester(a.View, 1040, 680)
	if !tt.HasText("Add your first key") {
		t.Fatalf("no empty state in %q", tt.Texts())
	}

	if err := tt.Click("Add your first key"); err != nil {
		t.Fatal(err)
	}

	if err := tt.Click("Save key"); err != nil {
		t.Fatal(err)
	}

	if !tt.HasText("Give the key a name.") {
		t.Errorf("no validation error in %q", tt.Texts())
	}

	// Filled in directly: the error above moved the fields, and focus is on Save.
	a.draftName, a.draftKey = "Deploy", testKey

	tt.Frame()

	if err := tt.Click("Save key"); err != nil {
		t.Fatal(err)
	}

	if len(f) != 1 || a.adding || a.draftKey != "" {
		t.Fatalf("after save: secrets %v, adding %v, draft %q", f, a.adding, a.draftKey)
	}

	if !tt.HasText("OpenSSH private key") {
		t.Errorf("no detail in %q", tt.Texts())
	}

	if tt.HasText(testKey) {
		t.Error("key shows before Reveal")
	}

	if err := tt.Click("Reveal"); err != nil {
		t.Fatal(err)
	}

	if !tt.HasText(testKey) {
		t.Errorf("key hidden after Reveal: %q", tt.Texts())
	}

	if err := tt.Click("Hide"); err != nil {
		t.Fatal(err)
	}

	if tt.HasText(testKey) || a.revealed != "" {
		t.Error("key still held after Hide")
	}

	if err := tt.Click("Delete"); err != nil {
		t.Fatal(err)
	}

	tt.Key(0, ui.KeyEnter) // the alert's default button

	if len(f) != 0 || len(s.Entries) != 0 {
		t.Errorf("after delete: secrets %v, entries %v", f, s.Entries)
	}
}

func TestViewProjects(t *testing.T) {
	s, f := newTestStore(t)
	if _, err := s.Add("Loose", testKey, ""); err != nil {
		t.Fatal(err)
	}

	a := newApp(s, func(_ string, done func(error)) { done(nil) })
	tt := ui.NewTester(a.View, 1040, 680)

	if err := tt.Click("New project"); err != nil {
		t.Fatal(err)
	}

	if err := tt.Click("Create project"); err != nil {
		t.Fatal(err)
	}

	if !tt.HasText("Give the project a name.") {
		t.Errorf("no validation error in %q", tt.Texts())
	}

	a.draftProjectName = "Work"

	tt.Frame()

	if err := tt.Click("Create project"); err != nil {
		t.Fatal(err)
	}

	if a.addingProject || a.project() != "Work" || tt.HasText("Loose") {
		t.Fatalf("after creating: page %q, texts %q", a.page, tt.Texts())
	}

	// A key made on the project's page lands in the project, and only there.
	if err := tt.Click("New key"); err != nil {
		t.Fatal(err)
	}

	a.draftName, a.draftKey = "Inside", testKey

	tt.Frame()

	if err := tt.Click("Save key"); err != nil {
		t.Fatal(err)
	}

	if len(s.Entries) != 2 || s.Entries[1].Project != "Work" || a.visible()[a.chosen].Name != "Inside" {
		t.Fatalf("after save: %+v, chosen %d", s.Entries, a.chosen)
	}

	// A project is optional: the dialog can file a key outside it.
	if err := tt.Click("New key"); err != nil {
		t.Fatal(err)
	}

	a.draftName, a.draftKey, a.draftProject = "Outside", testKey, store.NoProject

	tt.Frame()

	if err := tt.Click("Save key"); err != nil {
		t.Fatal(err)
	}

	if s.Entries[2].Project != "" || a.page != "keys" || a.visible()[a.chosen].Name != "Outside" {
		t.Fatalf("after save: %+v, page %q, chosen %d", s.Entries, a.page, a.chosen)
	}

	// Move the chosen key with its picker.
	if err := tt.Click("Move to project"); err != nil {
		t.Fatal(err)
	}

	// By keyboard: "Work" also names the sidebar item under a click.
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEnter)
	tt.Frame()

	if s.Entries[2].Project != "Work" {
		t.Errorf("after moving: %+v", s.Entries)
	}

	// Revealing stops at the sidebar: another page has another chosen key.
	if err := tt.Click("Reveal"); err != nil {
		t.Fatal(err)
	}

	if a.revealed == "" {
		t.Fatal("not revealed")
	}

	if err := tt.Click("Work"); err != nil {
		t.Fatal(err)
	}

	tt.Frame()

	if a.project() != "Work" || a.chosen != 0 || a.revealed != "" {
		t.Errorf("page %q, chosen %d, revealed %q", a.page, a.chosen, a.revealed)
	}

	// Deleting the project keeps its keys.
	if err := tt.Click("Delete project"); err != nil {
		t.Fatal(err)
	}

	tt.Key(0, ui.KeyEnter)
	tt.Frame()

	if len(s.Projects) != 0 || a.page != "keys" || len(s.Entries) != 3 || len(f) != 3 || s.Entries[1].Project != "" {
		t.Errorf("after deleting: projects %v, page %q, entries %+v, secrets %d", s.Projects, a.page, s.Entries, len(f))
	}
}

// Nothing opens until the prompt passes. done runs between frames, as
// Window.Update runs it in the app.
func TestViewRequiresAuth(t *testing.T) {
	s, f := newTestStore(t)
	if _, err := s.Add("Deploy", testKey, ""); err != nil {
		t.Fatal(err)
	}

	var done func(error)

	a := newApp(s, func(_ string, d func(error)) { done = d })
	tt := ui.NewTester(a.View, 1040, 680)

	answer := func(err error) {
		t.Helper()

		if done == nil {
			t.Fatal("no prompt")
		}

		d := done
		done = nil

		d(err)
		tt.Frame()
	}

	for _, err := range []error{auth.ErrCanceled, errors.New("locked out")} {
		if err := tt.Click("Reveal"); err != nil {
			t.Fatal(err)
		}

		if tt.HasText(testKey) {
			t.Fatal("key shows while the prompt is open")
		}

		answer(err)

		if tt.HasText(testKey) || a.revealed != "" {
			t.Fatalf("key revealed after %v", err)
		}

		if got, want := tt.HasText("locked out"), err != auth.ErrCanceled; got != want {
			t.Errorf("after %v: message shown %v, want %v", err, got, want)
		}
	}

	if err := tt.Click("Copy"); err != nil {
		t.Fatal(err)
	}

	answer(auth.ErrCanceled)

	if tt.Clipboard() == testKey {
		t.Error("key copied without passing the prompt")
	}

	if err := tt.Click("Delete"); err != nil {
		t.Fatal(err)
	}

	tt.Key(0, ui.KeyEnter)
	answer(auth.ErrCanceled)

	if len(f) != 1 || len(s.Entries) != 1 {
		t.Errorf("deleted without passing the prompt: %v", s.Entries)
	}

	if err := tt.Click("Copy"); err != nil {
		t.Fatal(err)
	}

	tt.Click("Reveal") // ignored: the Copy prompt is still open
	answer(nil)

	if tt.Clipboard() != testKey || a.revealed != "" {
		t.Errorf("clipboard %q, revealed %q after passing the Copy prompt", tt.Clipboard(), a.revealed)
	}

	// Passing once unlocks everything: no more prompts.
	if err := tt.Click("Reveal"); err != nil {
		t.Fatal(err)
	}

	if done != nil || !tt.HasText(testKey) {
		t.Fatalf("after unlocking: prompt %v, texts %q", done != nil, tt.Texts())
	}

	// Sleep or a locked screen locks again, and takes the key off the screen.
	a.Lock()
	tt.Frame()

	if tt.HasText(testKey) {
		t.Error("key still shows after locking")
	}

	if err := tt.Click("Reveal"); err != nil {
		t.Fatal(err)
	}

	if tt.HasText(testKey) {
		t.Error("key shows after locking, before the prompt passes")
	}

	answer(nil)

	if !tt.HasText(testKey) {
		t.Errorf("key hidden after unlocking again: %q", tt.Texts())
	}
}

func TestViewUpdate(t *testing.T) {
	s, f := newTestStore(t)
	e, err := s.Add("Deploy", testKey, "")
	if err != nil {
		t.Fatal(err)
	}

	var done func(error)

	a := newApp(s, func(_ string, d func(error)) { done = d })
	tt := ui.NewTester(a.View, 1040, 680)
	secret := func() string { return string(f[store.SecretName(e.ID)]) }

	// Renaming needs no prompt, and keeps the key.
	if err := tt.Click("Update"); err != nil {
		t.Fatal(err)
	}

	if a.draftName != "Deploy" || a.draftKey != "" {
		t.Fatalf("dialog opened with %q, %q", a.draftName, a.draftKey)
	}

	a.draftName = "Renamed"

	tt.Frame()

	if err := tt.Click("Save changes"); err != nil {
		t.Fatal(err)
	}

	if done != nil || a.adding || s.Entries[0].Name != "Renamed" || secret() != testKey {
		t.Fatalf("after rename: prompt %v, adding %v, %+v", done != nil, a.adding, s.Entries[0])
	}

	const rotated = "-----BEGIN EC PRIVATE KEY-----\nnew\n"

	replace := func() {
		t.Helper()

		if err := tt.Click("Update"); err != nil {
			t.Fatal(err)
		}

		a.draftKey = rotated

		tt.Frame()

		if !tt.HasText("This replaces the current key. You can't undo this.") {
			t.Errorf("no warning in %q", tt.Texts())
		}

		if err := tt.Click("Save changes"); err != nil {
			t.Fatal(err)
		}

		if done == nil {
			t.Fatal("no prompt to replace the key")
		}
	}

	// A declined prompt keeps the old key and the draft.
	replace()

	d := done
	done = nil

	d(auth.ErrCanceled)
	tt.Frame()

	if secret() != testKey || !a.adding {
		t.Fatalf("after declining: secret %q, adding %v", secret(), a.adding)
	}

	if err := tt.Click("Cancel"); err != nil {
		t.Fatal(err)
	}

	// Canceling the dialog behind the prompt calls the replacement off.
	replace()

	if err := tt.Click("Cancel"); err != nil {
		t.Fatal(err)
	}

	d = done
	done = nil

	d(nil)
	tt.Frame()

	if secret() != testKey {
		t.Fatalf("replaced after canceling: %q", secret())
	}

	// A passed prompt replaces it, and drops the old key from the screen.
	a.Lock()

	a.revealed, a.revealedID = testKey, e.ID

	replace()

	d = done
	done = nil

	d(nil)
	tt.Frame()

	if secret() != rotated || a.adding || a.draftKey != "" || a.revealed != "" {
		t.Fatalf("after replacing: secret %q, adding %v, draft %q, revealed %q", secret(), a.adding, a.draftKey, a.revealed)
	}

	if !tt.HasText("EC private key") || !tt.HasText("Key updated") {
		t.Errorf("detail not updated: %q", tt.Texts())
	}
}

// Every control stays in the window, and panes widen with it.
func TestLayoutResponds(t *testing.T) {
	s, _ := newTestStore(t)
	if _, err := s.Add("Deploy", testKey, ""); err != nil {
		t.Fatal(err)
	}

	a := newApp(s, nil)
	tt := ui.NewTester(a.View, 720, 480)

	var narrowCard float32

	for _, width := range []int{720, 1040, 1800} {
		tt.SetSize(width, 680)

		for _, label := range []string{"New key", "Update", "Delete", "Reveal", "Copy"} {
			box, ok := tt.Find(label)
			if !ok || box.X+box.W > float32(width) {
				t.Errorf("at %d wide, %q at %+v (found %v)", width, label, box, ok)
			}
		}

		if box, _ := tt.Find("Delete"); width == 720 {
			narrowCard = box.X
		} else if box.X <= narrowCard {
			t.Errorf("at %d wide, Delete stayed at x=%v", width, box.X)
		}
	}

	box, _ := tt.Find("Delete")
	if box.X+box.W < 1800-80 {
		t.Errorf("content stops short of the window at 1800 wide: Delete ends at %v", box.X+box.W)
	}
}

func TestPanesResize(t *testing.T) {
	s, _ := newTestStore(t)
	if _, err := s.AddProject("Work"); err != nil {
		t.Fatal(err)
	}

	if _, err := s.Add("Deploy", testKey, "Work"); err != nil {
		t.Fatal(err)
	}

	a := newApp(s, nil)
	a.page = projectPage("Work")

	const w, h = 1000, 680

	tt := ui.NewTester(a.View, w, h)

	drag := func(from, to float32) {
		tt.Press(from, h/2)
		tt.Move(to, h/2)
		tt.Release(to, h/2)
		tt.Frame()
	}

	sidebar := func() float32 { return a.sidebarPercent * w / 100 }
	list := func() float32 { return a.listPercent * (w - sidebar()) / 100 }

	drag(sidebar(), 300)

	if got := sidebar(); got < 295 || got > 305 {
		t.Errorf("sidebar %v wide after dragging to 300", got)
	}

	listEnd := sidebar() + list()
	drag(listEnd, listEnd+100)

	if got := sidebar() + list(); got < listEnd+95 || got > listEnd+105 {
		t.Errorf("list ends at %v after dragging to %v", got, listEnd+100)
	}

	// Dragged to the edges, every pane keeps its minimum.
	drag(sidebar(), 0)

	if got := sidebar(); got < minSidebarWidth-1 || got > minSidebarWidth+1 {
		t.Errorf("sidebar %v wide, want the minimum", got)
	}

	drag(sidebar()+list(), w)

	if got := w - sidebar() - list(); got < minDetailWidth-1 {
		t.Errorf("detail %v wide, want at least %v", got, minDetailWidth)
	}

	drag(sidebar()+list(), 0)

	if got := list(); got < minListWidth-1 {
		t.Errorf("list %v wide, want at least %v", got, minListWidth)
	}

	// Shares follow the window, down to the smallest it can be.
	tt.SetSize(720, h)
	tt.Frame()

	for _, label := range []string{"New key", "Update", "Delete"} {
		if box, ok := tt.Find(label); !ok || box.X+box.W > 720 {
			t.Errorf("at 720 wide, %q at %+v (found %v)", label, box, ok)
		}
	}
}

func TestLockForgetsDraft(t *testing.T) {
	s, _ := newTestStore(t)
	a := newApp(s, func(_ string, done func(error)) { done(nil) })

	a.NewKey()
	a.draftKey = "private"

	a.NewKey()

	if a.draftKey != "private" {
		t.Fatal("NewKey replaced the open draft")
	}

	a.Lock()

	if a.adding || a.draftKey != "" {
		t.Fatalf("draft survived locking: adding %v, key %q", a.adding, a.draftKey)
	}
}
