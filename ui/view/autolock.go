package view

import (
	"slices"
	"time"

	"github.com/egoist/mygo/ui"
	"keydepot/ui/icon"
	"keydepot/ui/widget"
)

var lockAfterChoices = []struct {
	label string
	after time.Duration
}{
	{"After 1 minute", time.Minute},
	{"After 5 minutes", 5 * time.Minute},
	{"After 15 minutes", 15 * time.Minute},
	{"After 30 minutes", 30 * time.Minute},
	{"After 1 hour", time.Hour},
	{"Never", 0},
}

func (a *App) autoLockCard(c *ui.Context) {
	widget.Card(c, func() {
		widget.SettingRow(c, icon.Lock, "Lock when idle", "Lock KeyDepot when the computer goes unused this long.", func() {
			labels := make([]string, len(lockAfterChoices))
			label := ""

			for i, choice := range lockAfterChoices {
				labels[i] = choice.label

				if choice.after == a.autoLock.After() {
					label = choice.label
				}
			}

			chosen := label
			ui.Select(c, &chosen, labels).Label("Lock when idle")

			if i := slices.Index(labels, chosen); chosen != label && i >= 0 {
				if err := a.autoLock.SetAfter(lockAfterChoices[i].after); err != nil {
					a.notice = "This choice applies until KeyDepot quits, as it couldn't be saved."
				}
			}
		})
	})
}
