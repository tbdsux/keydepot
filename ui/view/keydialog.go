package view

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"keydepot/lib/store"
	"keydepot/ui/widget"
)

func (a *App) newKey() {
	a.adding, a.draftProject = true, projectLabel(a.project())
}

func (a *App) addDialog(c *ui.Context) {
	t := c.Theme()
	editing := a.editingID != ""
	title, hint := "New key", "It's encrypted before it's saved."
	keyLabel, placeholder, action := "Private key", "-----BEGIN OPENSSH PRIVATE KEY-----", "Save key"

	if editing {
		title, hint = "Update key", "Rename it, or paste a rotated key to replace it."
		keyLabel, placeholder, action = "New private key", "Leave empty to keep the current key", "Save changes"
	}

	_, h := c.Size()

	ui.Modal(c, &a.adding, func() {
		ui.Column(c).Width(widget.DialogWidth(c, 0.55, 420)).Gap(16).Children(func() {
			ui.Column(c).Gap(3).Children(func() {
				ui.Text(c, title).FontSize(19).FontWeight(700)
				ui.Text(c, hint).TextColor(t.TextMuted)
			})

			ui.Column(c).Gap(6).Children(func() {
				ui.Text(c, "Name").FontSize(14).FontWeight(600)
				ui.TextInput(c, &a.draftName).Label("Name").Placeholder("Production deploy key").AutoFocus().FillWidth()
			})

			hasKey := strings.TrimSpace(a.draftKey) != ""

			ui.Column(c).Gap(6).Children(func() {
				ui.Row(c).Children(func() {
					ui.Text(c, keyLabel).FontSize(14).FontWeight(600).Grow(1)

					if hasKey {
						ui.Text(c, store.DetectKeyType(a.draftKey)).FontSize(13).TextColor(t.Accent).FontWeight(600)
					}
				})

				ui.TextArea(c, &a.draftKey).
					Label(keyLabel).
					Placeholder(placeholder).
					Font("monospace").
					FontSize(13).
					Height(max(120, h*0.3)).
					FillWidth()
			})

			if editing && hasKey {
				ui.Text(c, "This replaces the current key. You can't undo this.").FontSize(15).TextColor(t.Danger)
			}

			if !editing && len(a.store.Projects) > 0 {
				ui.Column(c).Gap(6).Children(func() {
					ui.Text(c, "Project").FontSize(14).FontWeight(600)
					ui.Select(c, &a.draftProject, a.projectOptions()).Label("Project").FillWidth()
				})
			}

			if a.formErr != "" {
				ui.Text(c, a.formErr).FontSize(15).TextColor(t.Danger)
			}

			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				ui.Button(c, "Cancel").OnClick(func() { a.adding = false })
				ui.PrimaryButton(c, action).OnClick(a.saveDraft)
			})
		})
	})
}

func (a *App) updateKey(e store.Entry) {
	a.adding, a.editingID, a.draftName = true, e.ID, e.Name
}

// saveUpdate asks only to replace the key, which can't be undone.
func (a *App) saveUpdate() {
	id, name, value := a.editingID, a.draftName, a.draftKey
	if strings.TrimSpace(value) == "" {
		value = ""
	}

	if err := store.CheckKey(name, value); err != nil {
		a.formErr = err.Error()
		return
	}

	apply := func() {
		// The dialog may have been canceled while the prompt was open.
		if !a.adding || a.editingID != id {
			return
		}

		if err := a.store.Update(id, name, value); err != nil {
			a.formErr = err.Error()
			return
		}

		a.adding = false
		a.hide()
	}

	if value == "" {
		apply()
		return
	}

	a.authorized(fmt.Sprintf("replace the key in “%s”", strings.TrimSpace(name)), apply)
}

func (a *App) saveDraft() {
	if a.editingID != "" {
		a.saveUpdate()
		return
	}

	if err := store.CheckKey(a.draftName, a.draftKey); err != nil {
		a.formErr = err.Error()
		return
	}

	a.writable(fmt.Sprintf("add “%s”", strings.TrimSpace(a.draftName)), func() {
		// The dialog may have been canceled while the prompt was open.
		if !a.adding {
			return
		}

		project := projectFromLabel(a.draftProject)
		if _, err := a.store.Add(a.draftName, a.draftKey, project); err != nil {
			a.formErr = err.Error()
			return
		}

		a.query, a.adding = "", false
		// On a project's page, follow the key to where it went.
		if a.page != "keys" {
			a.page = projectPage(project)
		}

		a.chosen = len(a.visible()) - 1
	})
}
