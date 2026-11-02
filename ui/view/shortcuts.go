package view

import (
	"slices"

	"github.com/egoist/mygo/ui"
	"keydepot/lib/shortcut"
	"keydepot/ui/widget"
)

func (a *App) shortcutsCard(c *ui.Context) {
	t := c.Theme()

	widget.Card(c, func() {
		ui.Column(c).Padding(16, 18, 12, 18).Gap(3).Children(func() {
			ui.Text(c, "Keyboard shortcuts").FontWeight(600)
			ui.Text(c, "Choose a shortcut to change it.").FontSize(14).TextColor(t.TextMuted)
		})

		for _, cmd := range commands {
			ui.Divider(c.Key("divider " + cmd.id))

			ui.Row(c.Key(cmd.id)).Padding(9, 14, 9, 18).Gap(12).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, cmd.icon).FontSize(16).TextColor(t.TextMuted)
				ui.Text(c, cmd.title).Grow(1).Shrink(1)

				s := a.shortcutFor(cmd)
				if s != cmd.fallback {
					ui.Button(c, "Reset").Label("Reset " + cmd.title).OnClick(func() {
						if err := a.setShortcut(cmd, cmd.fallback); err != nil {
							a.notice = err.Error()
						}
					})
				}

				ui.Button(c, s.Label()).Label("Change shortcut for " + cmd.title).OnClick(func() {
					a.recordOpen, a.recordingID, a.recordErr = true, cmd.id, ""
				})
			})
		}
	})
}

// shortcutDialog takes the next key pressed as a command's new shortcut.
func (a *App) shortcutDialog(c *ui.Context) {
	if !a.recordOpen {
		a.recordingID, a.recordErr = "", ""
		return
	}

	i := slices.IndexFunc(commands, func(cmd command) bool { return cmd.id == a.recordingID })
	if i < 0 {
		a.recordOpen = false
		return
	}

	cmd := commands[i]
	t := c.Theme()

	apply := func(s shortcut.Shortcut) {
		if err := a.setShortcut(cmd, s); err != nil {
			a.recordErr = err.Error()
			return
		}

		a.recordOpen = false
	}

	ui.Modal(c, &a.recordOpen, func() {
		ui.Column(c).Width(widget.DialogWidth(c, 0.4, 400)).Gap(16).Children(func() {
			ui.Column(c).Gap(3).Children(func() {
				ui.Text(c, "Change shortcut").FontSize(19).FontWeight(700)
				ui.Text(c, cmd.title).TextColor(t.TextMuted)
			})

			capture := ui.Column(c).
				FillWidth().
				Center().
				Gap(6).
				Padding(24).
				Radius(12).
				Background(t.Surface).
				Border(1, t.Accent).
				Focusable().
				AutoFocus().
				Label("Press the new shortcut")

			capture.HandleInput(func(ev ui.InputEvent) bool {
				if ev.Kind != ui.InputKeyDown || ev.Repeat {
					return false
				}

				if !shortcut.Known(ev.Key) {
					return false // a modifier by itself, or Escape to close
				}

				if ev.Key == ui.KeyTab && ev.Mods&^ui.Shift == 0 {
					return false // leaves the keyboard a way to the buttons
				}

				apply(shortcut.Shortcut{Mods: ev.Mods, Key: ev.Key})

				return true
			})

			capture.Children(func() {
				ui.Text(c, "Press the new shortcut").FontWeight(600)
				ui.Text(c, "Now: "+a.shortcutFor(cmd).Label()).FontSize(14).TextColor(t.TextMuted)
			})

			if a.recordErr != "" {
				ui.Text(c, a.recordErr).FontSize(14).TextColor(t.Danger)
			}

			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				if a.shortcutFor(cmd).Key != ui.KeyUnknown {
					ui.Button(c, "Remove shortcut").OnClick(func() { apply(shortcut.Shortcut{}) })
				}

				ui.Spacer(c)
				ui.Button(c, "Cancel").OnClick(func() { a.recordOpen = false })
			})
		})
	})
}
