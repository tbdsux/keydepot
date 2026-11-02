package view

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"keydepot/lib/auth"
	"keydepot/lib/vault"
	"keydepot/ui/widget"
)

// prompt is the app's auth once a vault is open: it asks for what the vault's
// protection needs, and leaves the data key loaded only if that passed.
func (a *App) prompt(reason string, done func(error)) {
	if a.vault.PasswordProtected() {
		a.askUnlock(reason, done)
		return
	}

	var err error

	a.run(func() {
		var mode string
		if mode, err = a.vault.Load(); err == nil && mode == vault.ModeSystem {
			err = a.authenticate(reason)
		}

		if err != nil {
			a.vault.Lock()
		}
	}, func() {
		// The Keychain item and every key are gone: back to the picker.
		if a.vault.NeedsSetup() {
			err = auth.ErrCanceled
		}

		done(err)
	})
}

func (a *App) askUnlock(reason string, done func(error)) {
	a.askFor(&passwordAsk{
		title: "Enter your KeyDepot password", hint: "KeyDepot needs it to " + reason + ".",
		action: "Unlock", label: "Password",
		work:     a.vault.UnlockPassword,
		then:     func() { done(nil) },
		canceled: func() { done(auth.ErrCanceled) },
		forgot:   func() { a.askRecovery(done) },
	})
}

// askRecovery never leaves the old password or code invalid until the new
// ones are written: canceling at any step keeps them.
func (a *App) askRecovery(done func(error)) {
	a.askFor(&passwordAsk{
		title: "Enter your recovery code", hint: "You'll then choose a new password and get a new code. This one stops working.",
		action: "Continue", label: "Recovery code", plain: true,
		work: a.vault.UnlockRecovery,
		then: func() {
			a.askNewPassword("Choose a new password", func() { done(nil) }, func() {
				a.vault.Lock()
				done(auth.ErrCanceled)
			})
		},
		canceled: func() { done(auth.ErrCanceled) },
	})
}

// askNewPassword moves the vault into password mode, which issues a code.
func (a *App) askNewPassword(title string, then, canceled func()) {
	var code string

	a.askFor(&passwordAsk{
		title: title, hint: "It encrypts your keys. KeyDepot doesn't store it and can't reset it.",
		action: "Set password", label: "Password", minLen: minPasswordLen,
		work: func(password string) (err error) {
			code, err = a.vault.Protect(vault.ModePassword, password)
			return err
		},
		then: func() {
			a.recoveryCode, a.recoveryTyped, a.recoveryErr = code, "", ""

			if then != nil {
				then()
			}
		},
		canceled: canceled,
	})
}

// recoveryDialog can't be dismissed: the code is shown only this once.
func (a *App) recoveryDialog(c *ui.Context) {
	if a.recoveryCode == "" {
		return
	}

	t := c.Theme()
	services := c.Services()
	open := true

	ui.Modal(c, &open, func() {
		ui.Column(c).Width(widget.DialogWidth(c, 0.5, 440)).Gap(16).Children(func() {
			ui.Column(c).Gap(6).Children(func() {
				ui.Text(c, "Save your recovery code").FontSize(19).FontWeight(700)
				ui.Text(c, "It's your way back in if you forget your password, and KeyDepot shows it only once.").TextColor(t.TextMuted)
				ui.Text(c, "Anyone with this code and your vault file can read your keys, so keep it in a password manager, not beside the vault or in the same backup.").TextColor(t.TextMuted)
				ui.Text(c, "If you lose both the password and this code, your keys are gone.").TextColor(t.Danger)
			})

			ui.Row(c).Padding(12).Radius(9).Background(t.Surface).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, vault.FormatCode(a.recoveryCode)).
					Font("monospace").
					FontSize(14).
					Selectable().
					Grow(1).
					Shrink(1)
				ui.Button(c, "Copy code").OnClick(func() { a.copySecret(services, a.recoveryCode, "Copied") })
			})

			ui.Column(c).Gap(6).Children(func() {
				ui.Text(c, fmt.Sprintf("Type its last %d characters to confirm you saved it", confirmCodeLen)).
					FontSize(14).
					FontWeight(600)
				ui.TextInput(c, &a.recoveryTyped).Label("Last characters").FillWidth().OnSubmit(a.confirmRecovery)
			})

			if a.recoveryErr != "" {
				ui.Text(c, a.recoveryErr).FontSize(15).TextColor(t.Danger)
			}

			ui.Row(c).Justify(ui.End).Children(func() {
				ui.PrimaryButton(c, "I've saved it").OnClick(a.confirmRecovery)
			})
		})
	})
}

func (a *App) confirmRecovery() {
	if vault.NormalizeCode(a.recoveryTyped) != a.recoveryCode[len(a.recoveryCode)-confirmCodeLen:] {
		a.recoveryErr = "Those aren't its last characters."
		return
	}

	a.recoveryCode, a.recoveryTyped, a.recoveryErr = "", "", ""
}

// reauth asks again even when unlocked, for what must not happen at a computer
// someone merely walked up to.
func (a *App) reauth(reason string, then func()) {
	a.Lock()
	a.authorized(reason, then)
}

// writable runs then once the vault can encrypt, which only needs asking for
// with a password: otherwise the Keychain hands over the data key.
func (a *App) writable(reason string, then func()) {
	if a.vault != nil && a.vault.PasswordProtected() {
		a.authorized(reason, then)
		return
	}

	then()
}
