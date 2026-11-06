package view

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"keydepot/lib/dotenv"
	"keydepot/lib/store"
	"keydepot/ui/widget"
)

// openEnvImport files the keys in the project showing, as a new key would be.
func (a *App) openEnvImport() {
	a.envOpen, a.envProject = true, projectLabel(a.project())
}

func (a *App) envDialog(c *ui.Context) {
	t := c.Theme()
	_, h := c.Size()

	vars, parseErr := dotenv.Parse(a.envDraft)
	n := withValues(vars)

	ui.Modal(c, &a.envOpen, func() {
		ui.Column(c).Width(widget.DialogWidth(c, 0.55, 420)).Gap(16).Children(func() {
			ui.Column(c).Gap(3).Children(func() {
				ui.Text(c, "Import .env").FontSize(19).FontWeight(700)
				ui.Text(c, "Paste .env files or choose them. Each variable becomes a key named after it.").
					TextColor(t.TextMuted)
			})

			ui.Column(c).Gap(6).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Children(func() {
					ui.Text(c, ".env contents").FontSize(14).FontWeight(600).Grow(1)
					ui.Button(c, "Choose files…").OnClick(a.chooseEnvFiles)
				})

				ui.TextArea(c, &a.envDraft).
					Label(".env contents").
					Placeholder("API_KEY=…").
					Font("monospace").
					FontSize(13).
					Height(max(160, h*0.35)).
					AutoFocus().
					FillWidth()

				switch {
				case parseErr != nil:
					ui.Text(c, parseErr.Error()).FontSize(14).TextColor(t.Danger)

				case len(vars) > 0:
					status := fmt.Sprintf("%d keys found.", n)
					if skipped := len(vars) - n; skipped > 0 {
						status += fmt.Sprintf(" %d empty values are skipped.", skipped)
					}

					ui.Text(c, status).FontSize(14).TextColor(t.TextMuted)
				}
			})

			if len(a.store.Projects) > 0 {
				ui.Column(c).Gap(6).Children(func() {
					ui.Text(c, "Project").FontSize(14).FontWeight(600)
					ui.Select(c, &a.envProject, a.projectOptions()).Label("Project").FillWidth()
				})
			}

			if a.envErr != "" {
				ui.Text(c, a.envErr).FontSize(15).TextColor(t.Danger)
			}

			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				ui.Button(c, "Cancel").OnClick(func() { a.envOpen = false })
				ui.PrimaryButton(c, "Import").Disabled(parseErr != nil || n == 0).OnClick(a.saveEnvImport)
			})
		})
	})
}

func withValues(vars []dotenv.Var) int {
	n := 0

	for _, v := range vars {
		if v.Value != "" {
			n++
		}
	}

	return n
}

func (a *App) chooseEnvFiles() {
	var text string
	var err error

	a.run(func() {
		var paths []string
		if paths, err = a.pickEnv(); err == nil {
			text, err = dotenv.ReadFiles(paths)
		}
	}, func() {
		// Closed while picking: what was read must not come back.
		if !a.envOpen {
			return
		}

		if err != nil {
			a.envErr = err.Error()
			return
		}

		if text == "" {
			return
		}

		if a.envDraft != "" {
			a.envDraft = strings.TrimRight(a.envDraft, "\n") + "\n"
		}

		a.envDraft, a.envErr = a.envDraft+text, ""
	})
}

func (a *App) saveEnvImport() {
	vars, err := dotenv.Parse(a.envDraft)
	if err != nil {
		a.envErr = err.Error()
		return
	}

	project := projectFromLabel(a.envProject)

	a.writable("import keys", func() {
		// The dialog may have been canceled while the prompt was open.
		if !a.envOpen {
			return
		}

		var index store.Index

		keys := map[string]string{}

		for i, v := range vars {
			if v.Value == "" {
				continue
			}

			id := strconv.Itoa(i)
			index.Keys = append(index.Keys, store.Entry{ID: id, Name: v.Name, Project: project})
			keys[id] = v.Value
		}

		n, err := a.store.ImportKeys(index, keys, nil)
		if err != nil && n == 0 {
			a.envErr = err.Error()
			return
		}

		a.envOpen = false

		if err != nil {
			a.notice = fmt.Sprintf("Imported %d keys, then stopped: %v", n, err)
			return
		}

		a.notice = fmt.Sprintf("Imported %d keys.", n)
	})
}
