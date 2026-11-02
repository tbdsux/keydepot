package view

import (
	"fmt"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"keydepot/ui/widget"
)

const (
	minPasswordLen    = 8
	strongPasswordLen = 12
	// An export leaves the machine, so it is the file most likely to be stolen.
	minExportPasswordLen = strongPasswordLen
	confirmCodeLen       = 4
)

// passwordAsk is the dialog that collects a password or a recovery code.
type passwordAsk struct {
	title, hint, action, label string
	// plain shows what is typed: a recovery code is read from elsewhere.
	plain bool
	// minLen, when set, asks for a new password: typed twice, and this long.
	minLen int
	// work runs off the UI thread, then then on it after a success.
	work     func(value string) error
	then     func()
	canceled func()
	forgot   func()
}

func (a *App) askFor(ask *passwordAsk) {
	a.ask, a.askOpen, a.askValue, a.askConfirm, a.askErr, a.askBusy = ask, true, "", "", "", false
}

func (a *App) closeAsk() {
	a.ask, a.askOpen, a.askValue, a.askConfirm, a.askErr, a.askBusy = nil, false, "", "", "", false
}

func (a *App) submitAsk() {
	ask, value := a.ask, a.askValue
	if ask == nil || a.askBusy {
		return
	}

	switch {
	case value == "":
		a.askErr = "Enter it to continue."
		return

	case ask.minLen > 0 && utf8.RuneCountInString(value) < ask.minLen:
		a.askErr = fmt.Sprintf("Use at least %d characters.", ask.minLen)
		return

	case ask.minLen > 0 && value != a.askConfirm:
		a.askErr = "The passwords don't match."
		return
	}

	a.askErr, a.askBusy = "", true

	var err error

	a.run(func() { err = ask.work(value) }, func() {
		a.askBusy = false

		if err != nil {
			a.askErr = err.Error()
			return
		}

		a.closeAsk()

		if ask.then != nil {
			ask.then()
		}
	})
}

func (a *App) askDialog(c *ui.Context) {
	if a.ask == nil {
		return
	}

	// Dismissing mid-work could drop a recovery code the vault now depends on.
	if a.askBusy {
		a.askOpen = true
	}

	if !a.askOpen {
		ask := a.ask

		a.closeAsk()

		if ask.canceled != nil {
			ask.canceled()
		}

		return
	}

	t := c.Theme()
	ask := a.ask

	ui.Modal(c, &a.askOpen, func() {
		ui.Column(c).Width(widget.DialogWidth(c, 0.4, 380)).Gap(16).Children(func() {
			ui.Column(c).Gap(3).Children(func() {
				ui.Text(c, ask.title).FontSize(19).FontWeight(700)
				ui.Text(c, ask.hint).TextColor(t.TextMuted)
			})

			input := ui.TextInput(c.Key(ask.label), &a.askValue).
				Label(ask.label).
				Placeholder(ask.label).
				AutoFocus().
				FillWidth().
				OnSubmit(a.submitAsk)

			if !ask.plain {
				input.Password()
			}

			if ask.minLen > 0 {
				ui.TextInput(c, &a.askConfirm).
					Label("Confirm password").
					Placeholder("Confirm password").
					Password().
					FillWidth().
					OnSubmit(a.submitAsk)

				if n := utf8.RuneCountInString(a.askValue); n > 0 && n < strongPasswordLen {
					ui.Text(c, "Short passwords are easier to crack if your vault file is stolen.").
						FontSize(14).
						TextColor(t.TextMuted)
				}
			}

			if a.askErr != "" {
				ui.Text(c, a.askErr).FontSize(15).TextColor(t.Danger)
			}

			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				if ask.forgot != nil {
					ui.Button(c, "Forgot password?").OnClick(ask.forgot)
				}

				ui.Spacer(c)
				ui.Button(c, "Cancel").OnClick(func() { a.askOpen = false })

				action := ask.action

				if a.askBusy {
					action = "Working…"
				}

				ui.PrimaryButton(c, action).OnClick(a.submitAsk)
			})
		})
	})
}
