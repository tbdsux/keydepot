package view

import (
	"errors"
	"fmt"

	"github.com/egoist/mygo/ui"
	"keydepot/lib/shortcut"
	"keydepot/lib/store"
	"keydepot/ui/icon"
)

// command is what the palette lists and a shortcut can trigger.
type command struct {
	id, title string
	icon      *ui.SVG
	fallback  shortcut.Shortcut
	// needsKey commands act on the chosen key, so they wait for one.
	needsKey bool
	run      func(a *App, c *ui.Context, e store.Entry)
}

var commands []command

// Set here since the commands refer to what refers to them.
func init() {
	commands = []command{
		{
			id:       "palette",
			title:    "Open command palette",
			icon:     icon.Search,
			fallback: shortcut.Shortcut{Mods: ui.Cmd, Key: ui.KeyK},
			run:      func(a *App, _ *ui.Context, _ store.Entry) { a.paletteOpen = true },
		},
		{
			id:       "newKey",
			title:    "New key",
			icon:     icon.Plus,
			fallback: shortcut.Shortcut{Mods: ui.Cmd, Key: ui.KeyN},
			run:      func(a *App, _ *ui.Context, _ store.Entry) { a.newKey() },
		},
		{
			id:       "newProject",
			title:    "New project",
			icon:     icon.Folder,
			fallback: shortcut.Shortcut{Mods: ui.Cmd | ui.Shift, Key: ui.KeyN},
			run:      func(a *App, _ *ui.Context, _ store.Entry) { a.addingProject = true },
		},
		{
			id:       "importEnv",
			title:    "Import .env",
			icon:     icon.Plus,
			fallback: shortcut.Shortcut{Mods: ui.Cmd | ui.Shift, Key: ui.KeyI},
			run:      func(a *App, _ *ui.Context, _ store.Entry) { a.openEnvImport() },
		},
		{
			id:       "copyEnv",
			title:    "Copy keys as .env",
			icon:     icon.Copy,
			fallback: shortcut.Shortcut{Mods: ui.Cmd | ui.Shift, Key: ui.KeyE},
			run:      func(a *App, c *ui.Context, _ store.Entry) { a.copyEnv(c.Services()) },
		},
		{
			id:       "search",
			title:    "Search keys",
			icon:     icon.Search,
			fallback: shortcut.Shortcut{Mods: ui.Cmd, Key: ui.KeyF},
			run: func(a *App, c *ui.Context, _ store.Entry) {
				if a.page == "settings" {
					a.page = "keys"
				}

				a.searchHandle.Focus(c)
			},
		},
		{
			id:       "keys",
			title:    "Show all keys",
			icon:     icon.Key,
			fallback: shortcut.Shortcut{Mods: ui.Cmd, Key: ui.Key1},
			run:      func(a *App, _ *ui.Context, _ store.Entry) { a.page, a.query, a.chosen = "keys", "", 0 },
		},
		{
			id:       "settings",
			title:    "Open settings",
			icon:     icon.Settings,
			fallback: shortcut.Shortcut{Mods: ui.Cmd, Key: ui.KeyComma},
			run:      func(a *App, _ *ui.Context, _ store.Entry) { a.page = "settings" },
		},
		{
			id:       "lock",
			title:    "Lock KeyDepot",
			icon:     icon.Lock,
			fallback: shortcut.Shortcut{Mods: ui.Cmd, Key: ui.KeyL},
			run: func(a *App, _ *ui.Context, _ store.Entry) {
				a.Lock()
				a.notice = "Locked."
			},
		},
		{
			id:       "copy",
			title:    "Copy key",
			icon:     icon.Copy,
			fallback: shortcut.Shortcut{Mods: ui.Cmd | ui.Shift, Key: ui.KeyC},
			needsKey: true,
			run:      func(a *App, c *ui.Context, e store.Entry) { a.askCopy(c.Services(), e) },
		},
		{
			id:       "reveal",
			title:    "Reveal or hide key",
			icon:     icon.Eye,
			fallback: shortcut.Shortcut{Mods: ui.Cmd | ui.Shift, Key: ui.KeyR},
			needsKey: true,
			run:      func(a *App, _ *ui.Context, e store.Entry) { a.toggleReveal(e) },
		},
		{
			id:       "update",
			title:    "Update key",
			icon:     icon.Pencil,
			fallback: shortcut.Shortcut{Mods: ui.Cmd, Key: ui.KeyE},
			needsKey: true,
			run:      func(a *App, _ *ui.Context, e store.Entry) { a.updateKey(e) },
		},
		{
			id:       "delete",
			title:    "Delete key",
			icon:     icon.Trash,
			fallback: shortcut.Shortcut{Mods: ui.Cmd, Key: ui.KeyBackspace},
			needsKey: true,
			run:      func(a *App, _ *ui.Context, _ store.Entry) { a.confirmDelete = true },
		},
	}
}

func (a *App) shortcutFor(cmd command) shortcut.Shortcut {
	if s, ok := a.shortcuts[cmd.id]; ok {
		return s
	}

	return cmd.fallback
}

// bindShortcuts declares the commands' shortcuts, which dialogs block.
func (a *App) bindShortcuts(c *ui.Context, target *store.Entry) {
	for _, cmd := range commands {
		s := a.shortcutFor(cmd)
		if s.Key == ui.KeyUnknown || cmd.needsKey && target == nil {
			continue
		}

		c.OnShortcut(s.Mods, s.Key, func() {
			var e store.Entry

			if target != nil {
				e = *target
			}

			cmd.run(a, c, e)
		})
	}
}

// setShortcut gives cmd a shortcut, or says why it can't have it.
func (a *App) setShortcut(cmd command, s shortcut.Shortcut) error {
	if s.Key != ui.KeyUnknown {
		if problem := s.Problem(); problem != "" {
			return errors.New(problem)
		}

		for _, other := range commands {
			if other.id != cmd.id && a.shortcutFor(other) == s {
				return fmt.Errorf("%s is already used by “%s”.", s.Label(), other.title)
			}
		}
	}

	if s == cmd.fallback {
		delete(a.shortcuts, cmd.id)
	} else {
		a.shortcuts[cmd.id] = s
	}

	if a.shortcutsPath == "" {
		return nil
	}

	return shortcut.Save(a.shortcutsPath, a.shortcuts)
}
