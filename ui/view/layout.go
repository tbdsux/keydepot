package view

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"keydepot/lib/store"
	"keydepot/ui/icon"
	"keydepot/ui/theme"
)

// View builds the window from the app's state.
func (a *App) View(c *ui.Context) {
	c.SetTheme(theme.New(c.Theme().Dark))

	t := c.Theme()

	if a.notice != "" {
		c.Toast(a.notice)

		a.notice = ""
	}

	if a.store == nil {
		ui.Column(c).Fill().Center().Gap(8).Padding(32).Children(func() {
			ui.Text(c, "KeyDepot can't read its key list").FontSize(19).FontWeight(600)
			ui.Text(c, a.loadErr.Error()).TextColor(t.TextMuted).Selectable()
		})

		return
	}

	if a.vault != nil && a.vault.NeedsSetup() {
		a.setupPage(c)
		a.askDialog(c)

		return
	}

	// The draft holds a private key: it must not outlive the dialog, however it closed.
	if !a.adding {
		a.draftName, a.draftKey, a.formErr, a.editingID = "", "", "", ""
	}

	visible := a.visible()
	a.chosen = min(a.chosen, len(visible)-1)
	if a.chosen < 0 && len(visible) > 0 {
		a.chosen = 0
	}

	var current *store.Entry

	if a.chosen >= 0 {
		current = &visible[a.chosen]
	}

	if id := a.shownID(); id != "" && (current == nil || current.ID != id || a.page == "settings") {
		a.hide()
	}

	a.clampPanes(c)

	ui.SplitPercent(c, &a.sidebarPercent, func() { a.sidebar(c) }, func() {
		if a.page == "settings" {
			a.settings(c)
			return
		}

		ui.SplitPercent(c, &a.listPercent, func() { a.keyList(c, visible) }, func() {
			switch {
			case current != nil:
				a.detail(c, *current)

			case len(a.store.Entries) == 0:
				a.welcome(c)

			default:
				ui.Column(c).Fill().Center().Children(func() {
					ui.Text(c, "Choose a key").TextColor(t.TextMuted)
				})
			}
		}).Fill()
	}).Fill()

	// Settings shows no key, so commands must not act on one out of sight.
	target := current

	if a.page == "settings" {
		target = nil
	}

	a.bindShortcuts(c, target)
	a.addDialog(c)
	a.projectDialog(c)
	a.commandPalette(c, target)
	a.shortcutDialog(c)

	if project := a.project(); project != "" {
		title := fmt.Sprintf("Delete project “%s”?", project)
		if ui.AlertDialog(c, &a.confirmDeleteProject, title, "Its keys are kept and move out of the project.", "Cancel", "Delete") == 1 {
			if err := a.store.RemoveProject(project); err != nil {
				a.notice = err.Error()
			} else {
				a.page, a.chosen = "keys", 0
			}
		}
	}

	if current != nil {
		title := fmt.Sprintf("Delete “%s”?", current.Name)

		const message = "This removes it from your vault. You can't undo this."
		if ui.AlertDialog(c, &a.confirmDelete, title, message, "Cancel", "Delete") == 1 {
			id := current.ID

			a.authorized(fmt.Sprintf("delete “%s”", current.Name), func() {
				if err := a.store.Remove(id); err != nil {
					a.notice = err.Error()
				}
			})
		}
	}

	a.askDialog(c)
	a.recoveryDialog(c)
}

const (
	minSidebarWidth = 150
	minListWidth    = 200
	minDetailWidth  = 260
)

// clampPanes keeps dragged panes usable: ui.Split alone lets one shrink to 40 DIPs.
func (a *App) clampPanes(c *ui.Context) {
	w, _ := c.Size()
	if w <= 0 {
		return
	}

	sidebar := max(min(a.sidebarPercent*w/100, w-minListWidth-minDetailWidth), minSidebarWidth)
	a.sidebarPercent = sidebar / w * 100
	rest := w - sidebar
	a.listPercent = max(min(a.listPercent*rest/100, rest-minDetailWidth), minListWidth) / rest * 100
}

func (a *App) welcome(c *ui.Context) {
	t := c.Theme()

	ui.Column(c).Fill().Center().Gap(10).Padding(32).DragWindow().Children(func() {
		ui.Row(c).Size(64, 64).Center().Radius(18).Background(t.Accent.Alpha(0.12)).Children(func() {
			ui.Icon(c, icon.Lock).FontSize(28).TextColor(t.Accent)
		})

		ui.Text(c, "Keep private keys out of plain files").
			FontSize(24).
			FontWeight(700).
			LetterSpacing(-0.3).
			TextAlign(ui.Center)
		ui.Text(c, "KeyDepot encrypts each key and keeps it in your vault.").TextColor(t.TextMuted).TextAlign(ui.Center)

		ui.Column(c).Padding(10, 0, 0, 0).Children(func() {
			ui.PrimaryButton(c, "Add your first key").OnClick(a.newKey)
		})
	})
}
