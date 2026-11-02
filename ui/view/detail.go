package view

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"keydepot/lib/store"
	"keydepot/ui/icon"
	"keydepot/ui/widget"
)

func (a *App) detail(c *ui.Context, e store.Entry) {
	t := c.Theme()
	bar := c.TitleBar()

	ui.Column(c).Fill().Children(func() {
		ui.Row(c).Height(max(bar.Height, 28)).DragWindow()

		ui.Scroll(c).Grow(1).Children(func() {
			pad := widget.PagePadding(c)

			ui.Column(c).Padding(8, pad, pad, pad).Gap(20).FillWidth().Children(func() {
				ui.Row(c).Wrap().GapX(14).GapY(12).AlignItems(ui.Center).Children(func() {
					ui.Row(c).Gap(14).Grow(1).Shrink(1).AlignItems(ui.Center).Children(func() {
						ui.Row(c).Size(48, 48).Center().Radius(13).Background(t.Accent.Alpha(0.12)).Children(func() {
							ui.Icon(c, icon.Key).FontSize(24).TextColor(t.Accent)
						})

						ui.Column(c).Gap(2).Grow(1).Shrink(1).Children(func() {
							ui.Text(c, e.Name).FontSize(22).FontWeight(700).LetterSpacing(-0.3).Selectable()
							ui.Text(c, e.Type+" private key").TextColor(t.TextMuted)
						})
					})

					ui.Row(c).Gap(8).Children(func() {
						ui.Button(c, "").Label("Update key").OnClick(func() { a.updateKey(e) }).Children(func() {
							ui.Icon(c, icon.Pencil).FontSize(15)
							ui.Text(c, "Update").SingleLine()
						})

						ui.Button(c, "").
							Label("Delete key").
							OnClick(func() { a.confirmDelete = true }).
							Children(func() {
								ui.Icon(c, icon.Trash).FontSize(15).TextColor(t.Danger)
								ui.Text(c, "Delete").TextColor(t.Danger).SingleLine()
							})
					})
				})

				a.secretCard(c, e)

				widget.Card(c, func() {
					ui.Row(c).Padding(10, 14, 10, 18).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "Project").TextColor(t.TextMuted).Width(widget.InfoLabelWidth)

						label := projectLabel(e.Project)
						chosen := label
						ui.Select(c, &chosen, a.projectOptions()).Label("Move to project").Grow(1).Shrink(1)

						if chosen != label {
							if err := a.store.MoveKey(e.ID, projectFromLabel(chosen)); err != nil {
								a.notice = err.Error()
							}
						}
					})

					ui.Divider(c)
					widget.InfoRow(c, "Added", e.AddedAt.Format("2 January 2006, 15:04"))

					if !e.UpdatedAt.IsZero() {
						ui.Divider(c)
						widget.InfoRow(c, "Key updated", e.UpdatedAt.Format("2 January 2006, 15:04"))
					}

					ui.Divider(c)
					widget.InfoRow(c, "Stored in", vaultName)
				})
			})
		})
	})
}

func (a *App) secretCard(c *ui.Context, e store.Entry) {
	t := c.Theme()
	shown := a.revealedID == e.ID
	services := c.Services()

	widget.Card(c, func() {
		ui.Row(c).Wrap().Padding(14, 14, 14, 18).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Private key").FontWeight(600).Grow(1).SingleLine()

			ic, label := icon.Eye, "Reveal"

			if shown {
				ic, label = icon.EyeOff, "Hide"
			}

			ui.Button(c, "").Label(label).OnClick(func() {
				a.toggleReveal(e)
			}).Children(func() {
				ui.Icon(c, ic).FontSize(15)
				ui.Text(c, label).SingleLine()
			})

			ui.Button(c, "").Label("Copy").OnClick(func() {
				a.askCopy(services, e)
			}).Children(func() {
				ui.Icon(c, icon.Copy).FontSize(15)
				ui.Text(c, "Copy").SingleLine()
			})
		})

		ui.Divider(c)

		ui.Column(c).Padding(18).Background(t.Surface).Children(func() {
			if shown {
				ui.Text(c, a.revealed).Font("monospace").FontSize(13).LineHeight(1.5).Selectable()
				return
			}

			ui.Text(c, strings.Repeat("•", 36)).TextColor(t.TextMuted).LetterSpacing(1).SingleLine().Clip()
		})
	})
}
