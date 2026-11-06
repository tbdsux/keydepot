// KeyDepot is a desktop app that keeps private keys encrypted.
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"keydepot/lib/auth"
	"keydepot/lib/autolock"
	"keydepot/lib/paths"
	"keydepot/lib/store"
	"keydepot/lib/vault"
	"keydepot/ui/tray"
	"keydepot/ui/update"
	"keydepot/ui/view"
)

func main() {
	// One copy writes the vault at a time, and a second would overwrite the first's changes.
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}

	update.Start()

	var win *mygo.Window

	filters := []mygo.FileFilter{{Name: "KeyDepot export", Extensions: []string{"keydepot"}}}

	mygo.App.OnSecondInstance(func([]string, string) {
		if win != nil {
			win.Show()
			win.Focus()
		}
	})

	mygo.App.WhenReady(func() {
		v, s, err := openData()

		// Losing shortcuts is no reason to stop: the defaults work.
		shortcuts, _ := paths.Shortcuts()

		// An unreadable choice still locks, after the default time.
		var idle *autolock.AutoLock
		if path, err := paths.AutoLock(); err == nil {
			idle, _ = autolock.Open(path)
		}

		a := view.New(view.Options{
			Store:        s,
			Vault:        v,
			LoadErr:      err,
			Authenticate: auth.Authenticate,
			SystemAuth:   auth.HasSystemAuth(),
			TouchID:      auth.HasTouchID(),
			Run: func(work, then func()) {
				go func() {
					work()
					win.Update(then)
				}()
			},
			PickSave: func() (string, error) {
				return mygo.Dialog.Save(mygo.SaveDialogOptions{
					Parent:            win,
					Title:             "Export keys",
					DefaultPath:       "KeyDepot Export.keydepot",
					Filters:           filters,
					CreateDirectories: true,
				})
			},
			PickOpen: func() (string, error) {
				picked, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
					Parent:  win,
					Title:   "Import keys",
					Filters: filters,
				})
				if err != nil || len(picked) == 0 {
					return "", err
				}

				return picked[0], nil
			},
			PickEnv: func() ([]string, error) {
				return mygo.Dialog.Open(mygo.OpenDialogOptions{
					Parent:          win,
					Title:           "Import .env files",
					Multiple:        true,
					ShowHiddenFiles: true,
				})
			},
			ShortcutsPath: shortcuts,
			AutoLock:      idle,
		})

		win = mygo.NewWindow(mygo.WindowOptions{
			Title:         "KeyDepot",
			Width:         1040,
			Height:        680,
			MinWidth:      720,
			MinHeight:     480,
			StateKey:      "main",
			TitleBarStyle: mygo.TitleBarHidden,
			Content:       ui.View(a.View),
		})

		lock := func() { win.Update(a.Lock) }
		mygo.Power.OnSuspend(lock)
		mygo.Power.OnLockScreen(lock)
		// Before windows close, on the main thread: a key copied just before quitting leaves with it.
		mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { a.Lock() })

		if idle != nil {
			idle.Watch(lock)
		}

		// Without an icon closing the window quits, as there would be no way back.
		actions := tray.Actions{
			NewKey:   func() { win.Update(a.NewKey) },
			Settings: func() { win.Update(a.OpenSettings) },
			Lock:     lock,
		}

		if err := tray.Keep(win, actions); err != nil {
			log.Println("tray:", err)
		}
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// openData opens the vault, then the key list it holds the keys of.
func openData() (*vault.Vault, *store.Store, error) {
	vaultPath, err := paths.Vault()
	if err != nil {
		return nil, nil, err
	}

	indexPath, err := paths.Index()
	if err != nil {
		return nil, nil, err
	}

	v, err := vault.Open(mygo.Secrets, vaultPath)
	if err != nil {
		return nil, nil, err
	}

	s, err := store.Open(v, indexPath)
	if err != nil {
		return v, nil, err
	}

	return v, s, nil
}
