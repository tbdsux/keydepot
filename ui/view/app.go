// Package view builds KeyDepot's window from the app's state.
package view

import (
	"errors"

	"github.com/egoist/mygo/ui"
	"keydepot/lib/auth"
	"keydepot/lib/pubkey"
	"keydepot/lib/shortcut"
	"keydepot/lib/store"
	"keydepot/lib/vault"
)

// vaultName is how the interface names where keys live.
const vaultName = "Encrypted vault"

// App is the state the window shows. Its view builds the interface from it.
type App struct {
	store   *store.Store
	loadErr error
	page    string
	query   string

	list   ui.ListState
	chosen int

	// Each pane's share of the room it splits, so panes also follow the window.
	sidebarPercent float32
	listPercent    float32

	revealed   string
	revealedID string
	public     pubkey.Key
	publicID   string

	adding bool
	// editingID is the key the dialog updates, or empty when it adds one.
	editingID    string
	draftName    string
	draftKey     string
	draftProject string
	formErr      string

	addingProject    bool
	draftProjectName string
	projectErr       string

	confirmDelete        bool
	confirmDeleteProject bool

	// auth asks the user to prove who they are, then calls done on the UI thread.
	auth        func(reason string, done func(error))
	authPending bool
	unlocked    bool // passed the prompt since the app started or last locked

	vault *vault.Vault
	// authenticate is the system prompt.
	authenticate func(reason string) error
	systemAuth   bool // whether the system prompt can be offered
	touchID      bool // whether the system prompt includes Touch ID
	// run does slow work off the UI thread, then calls then on it.
	run func(work, then func())
	// pickSave and pickOpen ask for a file, returning "" when canceled.
	pickSave, pickOpen func() (string, error)
	setupMode          string

	ask                *passwordAsk
	askOpen, askBusy   bool
	askValue           string
	askConfirm, askErr string
	recoveryCode       string
	recoveryTyped      string
	recoveryErr        string

	paletteOpen   bool
	paletteQuery  string
	paletteChosen int
	paletteList   ui.ListState
	searchHandle  ui.Handle

	// shortcuts holds the user's changes to the commands' default shortcuts.
	shortcuts     map[string]shortcut.Shortcut
	shortcutsPath string
	recordOpen    bool
	recordingID   string
	recordErr     string

	// notice waits for the next frame: a Context ignores calls made between
	// frames, which is when auth answers.
	notice string
}

func newApp(s *store.Store, auth func(string, func(error))) *App {
	return &App{
		store:          s,
		page:           "keys",
		auth:           auth,
		sidebarPercent: 20,
		listPercent:    34,
		shortcuts:      map[string]shortcut.Shortcut{},
		run: func(work, then func()) {
			work()
			then()
		},
	}
}

// authorized runs then, asking the user to pass the system prompt unless they
// have since the app last locked.
func (a *App) authorized(reason string, then func()) {
	if a.unlocked {
		then()
		return
	}

	if a.authPending {
		return
	}

	a.authPending = true

	a.auth(reason, func(err error) {
		a.authPending = false

		switch {
		case err == nil:
			a.unlocked = true

			then()

		case !errors.Is(err, auth.ErrCanceled):
			a.notice = err.Error()
		}
	})
}

// Lock takes keys off the screen and forgets the data key.
func (a *App) Lock() {
	a.unlocked = false
	a.hide()

	// The draft may hold a private key, and a hidden window builds no frame to clear it.
	a.adding = false
	a.draftName, a.draftKey, a.formErr, a.editingID = "", "", "", ""

	if a.vault != nil {
		a.vault.Lock()
	}
}

// NewKey opens the new key dialog, leaving one already open as it is.
func (a *App) NewKey() {
	if !a.adding {
		a.newKey()
	}
}

// OpenSettings shows the settings page.
func (a *App) OpenSettings() { a.page = "settings" }

// Options is what the window's App is opened with.
type Options struct {
	Store *store.Store
	Vault *vault.Vault
	// LoadErr says why Store is nil.
	LoadErr error
	// Authenticate is the system prompt.
	Authenticate func(reason string) error
	// SystemAuth is whether the system prompt can be offered, and TouchID
	// whether it includes Touch ID.
	SystemAuth, TouchID bool
	// Run does slow work off the UI thread, then calls then on it.
	Run func(work, then func())
	// PickSave and PickOpen ask for a file, returning "" when canceled.
	PickSave, PickOpen func() (string, error)
	// ShortcutsPath is empty when the user's shortcuts can't be kept.
	ShortcutsPath string
}

// New opens the App main shows.
func New(o Options) *App {
	a := newApp(o.Store, nil)
	a.auth = a.prompt
	a.vault, a.loadErr = o.Vault, o.LoadErr
	a.authenticate, a.systemAuth, a.touchID = o.Authenticate, o.SystemAuth, o.TouchID
	a.run, a.pickSave, a.pickOpen = o.Run, o.PickSave, o.PickOpen

	// Losing shortcuts is no reason to stop: the defaults work.
	if o.ShortcutsPath != "" {
		a.shortcutsPath = o.ShortcutsPath

		var err error
		if a.shortcuts, err = shortcut.Load(o.ShortcutsPath); err != nil {
			a.notice = "Your shortcuts couldn't be read, so the defaults are in use."
		}
	}

	return a
}
