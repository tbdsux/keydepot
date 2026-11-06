package view

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"keydepot/lib/store"
	"keydepot/ui/icon"
)

func (a *App) visible() []store.Entry {
	q := strings.ToLower(strings.TrimSpace(a.query))
	project := a.project()

	var out []store.Entry

	for _, e := range a.store.Entries {
		if project != "" && e.Project != project {
			continue
		}

		if strings.Contains(strings.ToLower(e.Name), q) || strings.Contains(strings.ToLower(e.Type), q) {
			out = append(out, e)
		}
	}

	return out
}

func (a *App) keyList(c *ui.Context, visible []store.Entry) {
	t := c.Theme()
	bar := c.TitleBar()

	ui.Column(c).Fill().Children(func() {
		ui.Row(c).Height(max(bar.Height, 28)).DragWindow()

		title := "Keys"

		if project := a.project(); project != "" {
			title = project
		}

		ui.Row(c).Padding(8, 16, 12, 16).Gap(6).AlignItems(ui.Center).DragWindow().Children(func() {
			ui.Text(c, title).
				FontSize(21).
				FontWeight(700).
				LetterSpacing(-0.2).
				Grow(1).
				Shrink(1).
				SingleLine().
				Ellipsis("…")

			if a.project() != "" {
				ui.Button(c, "").
					Label("Delete project").
					Padding(8, 9).
					OnClick(func() { a.confirmDeleteProject = true }).
					Children(func() {
						ui.Icon(c, icon.Trash).FontSize(15).TextColor(t.Danger)
					})

				ui.Button(c, "").
					Label("Copy as .env").
					Padding(8, 9).
					OnClick(func() { a.copyEnv(c.Services()) }).
					Children(func() {
						ui.Icon(c, icon.Copy).FontSize(15)
					})
			}

			ui.PrimaryButton(c, "").Label("New key").OnClick(a.newKey).Children(func() {
				ui.Icon(c, icon.Plus).FontSize(15)
				ui.Text(c, "New key").FontWeight(600).SingleLine()
			})
		})

		ui.Column(c).Padding(0, 16, 10, 16).Children(func() {
			ui.SearchField(c, &a.query).
				Label("Search keys").
				Placeholder("Search keys").
				FillWidth().
				Bind(&a.searchHandle).
				OnChange(func() { a.chosen = 0 })
		})

		a.list.Selected = &a.chosen
		a.list.Key = func(i int) any { return visible[i].ID }
		a.list.Label = func(i int) string { return visible[i].Name }

		ui.List(c, &a.list, len(visible), func(i int) { a.row(c, visible[i], i == a.chosen) }).
			Grow(1).
			Gap(2).
			Padding(0, 10, 10, 10).
			Children(func() {
				if len(visible) == 0 {
					msg := "No keys yet"

					if a.query != "" {
						msg = "No matching keys"
					}

					ui.Text(c, msg).TextColor(t.TextMuted).Padding(18, 8)
				}
			})
	})
}

func (a *App) row(c *ui.Context, e store.Entry, selected bool) {
	t := c.Theme()
	// The list paints the chosen row in the accent color.
	text, muted, accent, tile := t.Text, t.TextMuted, t.Accent, t.Surface

	if selected {
		text, muted, accent, tile = t.AccentText, t.AccentText.Alpha(0.75), t.AccentText, t.AccentText.Alpha(0.16)
	}

	ui.Row(c).Padding(11, 12).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Size(36, 36).Center().Radius(10).Background(tile).Children(func() {
			ui.Icon(c, icon.Key).FontSize(15).TextColor(accent)
		})

		ui.Column(c).Gap(1).Grow(1).Shrink(1).Children(func() {
			ui.Text(c, e.Name).FontWeight(600).TextColor(text).SingleLine().Ellipsis("…")
			ui.Text(c, e.Type+" · "+e.AddedAt.Format("2 Jan 2006")).
				FontSize(13).
				TextColor(muted).
				SingleLine().
				Ellipsis("…")
		})
	})
}
