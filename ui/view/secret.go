package view

import (
	"fmt"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"keydepot/lib/store"
)

const clipboardLifetime = 60 * time.Second

func (a *App) hide() { a.revealed, a.revealedID = "", "" }

func (a *App) toggleReveal(e store.Entry) {
	if a.revealedID == e.ID {
		a.hide()
		return
	}

	a.authorized(fmt.Sprintf("reveal “%s”", e.Name), func() { a.reveal(e) })
}

func (a *App) askCopy(services ui.Services, e store.Entry) {
	a.authorized(fmt.Sprintf("copy “%s”", e.Name), func() { a.copy(services, e) })
}

func (a *App) reveal(e store.Entry) {
	value, err := a.store.Get(e.ID)
	if err != nil {
		a.notice = err.Error()
		return
	}

	a.revealed, a.revealedID = value, e.ID
}

func (a *App) copy(services ui.Services, e store.Entry) {
	value, err := a.store.Get(e.ID)
	if err != nil {
		a.notice = err.Error()
		return
	}

	a.copySecret(services, value, "Copied")
}

func (a *App) copySecret(services ui.Services, value, message string) {
	services.WriteClipboard(value)

	a.notice = fmt.Sprintf("%s. The clipboard clears in %d seconds.", message, int(clipboardLifetime.Seconds()))

	// Leaves the clipboard alone if the user has copied something else since.
	time.AfterFunc(clipboardLifetime, func() {
		if mygo.Clipboard.ReadText() == value {
			mygo.Clipboard.WriteText("")
		}
	})
}
