package view

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"keydepot/lib/store"
	"keydepot/ui/widget"
)

const projectPagePrefix = "project:"

// projectPage is the sidebar page listing a project's keys, or every key for
// no project.
func projectPage(project string) string {
	if project == "" {
		return "keys"
	}

	return projectPagePrefix + project
}

// project is the project whose page is showing, if any.
func (a *App) project() string {
	project, _ := strings.CutPrefix(a.page, projectPagePrefix)
	if project == a.page {
		return ""
	}

	return project
}

func projectLabel(project string) string {
	if project == "" {
		return store.NoProject
	}

	return project
}

func projectFromLabel(label string) string {
	if label == store.NoProject {
		return ""
	}

	return label
}

func (a *App) projectOptions() []string {
	return append([]string{store.NoProject}, a.store.Projects...)
}

func (a *App) projectDialog(c *ui.Context) {
	t := c.Theme()

	if !a.addingProject {
		a.draftProjectName, a.projectErr = "", ""
	}

	ui.Modal(c, &a.addingProject, func() {
		ui.Column(c).Width(widget.DialogWidth(c, 0.4, 340)).Gap(16).Children(func() {
			ui.Text(c, "New project").FontSize(19).FontWeight(700)
			ui.TextInput(c, &a.draftProjectName).
				Label("Project name").
				Placeholder("Project name").
				AutoFocus().
				FillWidth().
				OnSubmit(a.saveProject)

			if a.projectErr != "" {
				ui.Text(c, a.projectErr).FontSize(15).TextColor(t.Danger)
			}

			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				ui.Button(c, "Cancel").OnClick(func() { a.addingProject = false })
				ui.PrimaryButton(c, "Create project").OnClick(a.saveProject)
			})
		})
	})
}

func (a *App) saveProject() {
	name, err := a.store.AddProject(a.draftProjectName)
	if err != nil {
		a.projectErr = err.Error()
		return
	}

	a.addingProject, a.page, a.chosen = false, projectPage(name), 0
}
