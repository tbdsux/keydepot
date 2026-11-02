package view

import (
	"github.com/egoist/mygo/ui"
	"keydepot/ui/icon"
	"keydepot/ui/widget"
)

func (a *App) sidebar(c *ui.Context) {
	t := c.Theme()
	bar := c.TitleBar()

	ui.Column(c).Fill().Background(t.Surface).Children(func() {
		ui.Row(c).Height(max(bar.Height, 28)).DragWindow()

		ui.Row(c).Gap(10).Padding(8, 16, 16, 16).AlignItems(ui.Center).DragWindow().Children(func() {
			ui.Row(c).Size(30, 30).Center().Radius(9).Background(t.Accent).Children(func() {
				ui.Icon(c, icon.Lock).FontSize(16).TextColor(t.AccentText)
			})

			ui.Text(c, "KeyDepot").FontSize(18).FontWeight(700).LetterSpacing(-0.2)
		})

		a.paletteButton(c)

		ui.Sidebar(c, &a.page, func() {
			ui.SidebarItem(c, "keys", icon.Key, "Keys").Children(func() { widget.Count(c, len(a.store.Entries)) })
			ui.SidebarItem(c, "settings", icon.Settings, "Settings")

			if len(a.store.Projects) > 0 {
				ui.SidebarSection(c, "Projects", nil, func() {
					for _, p := range a.store.Projects {
						ui.SidebarItem(c.Key(p), projectPage(p), icon.Folder, p).Children(func() {
							widget.Count(c, a.store.ProjectSize(p))
						})
					}
				})
			}
		}).Grow(1).FillWidth().OnChange(func() { a.chosen = 0 })

		ui.Column(c).Padding(0, 10).Children(func() {
			ui.Button(c, "").
				Label("New project").
				FillWidth().
				OnClick(func() { a.addingProject = true }).
				Children(func() {
					ui.Icon(c, icon.Plus).FontSize(15)
					ui.Text(c, "New project").SingleLine()
				})
		})

		ui.Row(c).Gap(8).Padding(14, 16).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icon.Shield).FontSize(15).TextColor(t.Accent)
			ui.Text(c, vaultName).FontSize(13).TextColor(t.TextMuted)
		})
	})
}
