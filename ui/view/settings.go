package view

import (
	"github.com/egoist/mygo/ui"
	"keydepot/lib/paths"
	"keydepot/ui/widget"
)

func (a *App) settings(c *ui.Context) {
	t := c.Theme()
	bar := c.TitleBar()

	ui.Column(c).Fill().Children(func() {
		ui.Row(c).Height(max(bar.Height, 28)).DragWindow()

		ui.Scroll(c).Grow(1).Children(func() {
			pad := widget.PagePadding(c)

			ui.Column(c).Padding(8, pad, pad, pad).Gap(20).FillWidth().Children(func() {
				ui.Text(c, "Settings").FontSize(26).FontWeight(700).LetterSpacing(-0.3)

				if a.vault != nil {
					a.protectionCards(c)
				}

				a.shortcutsCard(c)

				widget.Card(c, func() {
					ui.Column(c).Padding(16, 18).Gap(5).Children(func() {
						index, _ := paths.Index()
						vaultFile, _ := paths.Vault()

						ui.Text(c, "Files").FontWeight(600)
						ui.Text(c, "The vault holds your encrypted keys. The key list holds names, types and projects, unencrypted, never the keys themselves.").
							FontSize(14).
							TextColor(t.TextMuted)
						ui.Text(c, vaultFile).FontSize(13).Font("monospace").TextColor(t.TextMuted).Selectable()
						ui.Text(c, index).FontSize(13).Font("monospace").TextColor(t.TextMuted).Selectable()
					})
				})
			})
		})
	})
}
