package view

import (
	"github.com/egoist/mygo/ui"
	"keydepot/lib/vault"
	"keydepot/ui/icon"
	"keydepot/ui/widget"
)

type protectionOption struct {
	mode, title, description string
	icon                     *ui.SVG
}

// protectionOptions lists what this computer can offer, the default first.
func (a *App) protectionOptions() []protectionOption {
	var options []protectionOption

	if a.systemAuth {
		options = append(options, protectionOption{
			mode:        vault.ModeSystem,
			title:       a.systemTitle(),
			description: "Asks once, until you quit or your Mac sleeps or locks. The vault's key stays in " + vault.StorageName() + ".",
			icon:        icon.Fingerprint,
		})
	}

	return append(options,
		protectionOption{
			mode:        vault.ModePassword,
			title:       "KeyDepot password",
			description: "Encrypts your keys with a password only you know, with a recovery code as the way back in.",
			icon:        icon.Lock,
		},
		protectionOption{
			mode:        vault.ModeNone,
			title:       "No extra prompt",
			description: "The weakest option: anyone using your unlocked computer can copy your keys. The vault's key stays in " + vault.StorageName() + ".",
			icon:        icon.Shield,
		},
	)
}

func (a *App) systemTitle() string {
	if a.touchID {
		return "Touch ID or Mac password"
	}

	return "Mac password"
}

// setupPage is the first launch: nothing else opens until the user chooses.
func (a *App) setupPage(c *ui.Context) {
	t := c.Theme()
	options := a.protectionOptions()

	if a.setupMode == "" {
		a.setupMode = options[0].mode
	}

	ui.Column(c).Fill().Center().Gap(18).Padding(32).DragWindow().Children(func() {
		ui.Column(c).Width(min(widget.DialogWidth(c, 0.6, 480), 620)).Gap(18).Children(func() {
			ui.Column(c).Gap(4).Children(func() {
				ui.Text(c, "How should KeyDepot protect your keys?").FontSize(24).FontWeight(700).LetterSpacing(-0.3)
				ui.Text(c, "They're always encrypted. This decides what opens them, and you can change it in Settings.").TextColor(t.TextMuted)
			})

			ui.RadioGroup(c, func() {
				ui.Column(c).Gap(12).FillWidth().Children(func() {
					for _, o := range options {
						ui.Column(c.Key(o.mode)).Gap(2).FillWidth().Children(func() {
							ui.Radio(c, &a.setupMode, o.mode, o.title)
							ui.Text(c, o.description).FontSize(14).TextColor(t.TextMuted).Padding(0, 0, 0, 26)
						})
					}
				})
			})

			ui.Row(c).Justify(ui.End).Children(func() {
				ui.PrimaryButton(c, "Continue").OnClick(func() { a.applyProtection(a.setupMode) })
			})
		})
	})
}

func (a *App) applyProtection(mode string) {
	if mode == vault.ModePassword {
		a.askNewPassword("Choose a KeyDepot password", func() { a.unlocked = true }, nil)
		return
	}

	var err error

	a.run(func() { _, err = a.vault.Protect(mode, "") }, func() {
		if err != nil {
			a.notice = err.Error()
		}
	})
}

func (a *App) protectionCards(c *ui.Context) {
	current := a.vault.ShownMode()
	options := a.protectionOptions()

	widget.Card(c, func() {
		ui.Column(c).Padding(16, 18, 4, 18).Children(func() {
			ui.Text(c, "Protection").FontWeight(600)
		})

		for i, o := range options {
			if i > 0 {
				ui.Divider(c)
			}

			widget.SettingRow(c.Key(o.mode), o.icon, o.title, o.description, func() {
				if o.mode == current {
					ui.Text(c, "Current").FontSize(13).FontWeight(600).TextColor(c.Theme().Accent).SingleLine()
					return
				}

				ui.Button(c, "Use").Label("Use " + o.title).OnClick(func() {
					a.reauth("change how your keys are protected", func() { a.applyProtection(o.mode) })
				})
			})
		}

		if current == vault.ModePassword {
			ui.Divider(c)

			widget.SettingRow(c, icon.Lock, "Password", "Changing it keeps your recovery code valid.", func() {
				ui.Button(c, "Change password").OnClick(func() {
					a.reauth("change your password", func() {
						a.askFor(&passwordAsk{
							title: "Choose a new password", hint: "Your recovery code keeps working.",
							action: "Save password", label: "New password", minLen: minPasswordLen,
							work: a.vault.ChangePassword,
							then: func() { a.notice = "Password changed." },
						})
					})
				})
			})

			ui.Divider(c)

			widget.SettingRow(c, icon.Shield, "Recovery code", "A new code replaces the old one, which stops working.", func() {
				ui.Button(c, "New recovery code").OnClick(func() {
					a.reauth("make a new recovery code", a.renewRecoveryCode)
				})
			})
		}
	})

	widget.Card(c, func() {
		widget.SettingRow(c, icon.Key, "Export and import", "An export is a file with its own password. Importing only adds keys and never overwrites.", func() {
			ui.Row(c).Gap(8).Children(func() {
				ui.Button(c, "Export keys").OnClick(a.exportKeys)
				ui.Button(c, "Import keys").OnClick(a.importKeys)
				ui.Button(c, "Import .env").OnClick(a.openEnvImport)
			})
		})
	})
}

func (a *App) renewRecoveryCode() {
	var code string
	var err error

	a.run(func() { code, err = a.vault.RenewRecoveryCode() }, func() {
		if err != nil {
			a.notice = err.Error()
			return
		}

		a.recoveryCode, a.recoveryTyped, a.recoveryErr = code, "", ""
	})
}
