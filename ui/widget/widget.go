// Package widget holds the stateless building blocks the views share.
package widget

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

// Count is the number at the end of a sidebar row.
func Count(c *ui.Context, n int) {
	ui.Spacer(c)
	ui.Text(c, fmt.Sprint(n)).FontSize(13).TextColor(c.Theme().TextMuted).FontFeatures("tnum")
}

// PagePadding tightens the page's sides as the window narrows.
func PagePadding(c *ui.Context) float32 {
	if w, _ := c.Size(); w < 1100 {
		return 28
	}

	return 36
}

// DialogWidth gives a dialog a share of the window, for those too narrow to spare one.
func DialogWidth(c *ui.Context, share, floor float32) float32 {
	w, _ := c.Size()
	return max(w*share, floor)
}

// Card is a bordered group of rows.
func Card(c *ui.Context, children func()) {
	t := c.Theme()
	ui.Column(c).FillWidth().Radius(14).Border(1, t.Border).Background(t.Background).Clip().Children(children)
}

// InfoLabelWidth lines up the labels of a card's rows.
const InfoLabelWidth = 108

// InfoRow shows a label and its value.
func InfoRow(c *ui.Context, label, value string) {
	ui.Row(c).Padding(14, 18).Children(func() {
		ui.Text(c, label).TextColor(c.Theme().TextMuted).Width(InfoLabelWidth)
		ui.Text(c, value).Grow(1).Shrink(1)
	})
}

// SettingRow describes a setting, with action building its control.
func SettingRow(c *ui.Context, ic *ui.SVG, title, description string, action func()) {
	t := c.Theme()

	ui.Row(c).Padding(16, 18).Gap(14).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, ic).FontSize(19).TextColor(t.TextMuted)

		ui.Column(c).Gap(2).Grow(1).Shrink(1).Children(func() {
			ui.Text(c, title).FontWeight(600)
			ui.Text(c, description).FontSize(14).TextColor(t.TextMuted)
		})

		action()
	})
}

// Keycap shows keys as a small raised label.
func Keycap(c *ui.Context, text string, fg, bg, border ui.Color) {
	ui.Row(c).Padding(2, 7).Radius(6).Background(bg).Border(1, border).Shrink(0).Children(func() {
		ui.Text(c, text).FontSize(13).TextColor(fg).SingleLine()
	})
}
