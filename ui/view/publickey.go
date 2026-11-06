package view

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"keydepot/lib/pubkey"
	"keydepot/lib/store"
	"keydepot/ui/icon"
	"keydepot/ui/widget"
)

// hasPublicKey is whether e's type can be an SSH key; PGP and encrypted
// PKCS#8 keys can't be read without more.
func hasPublicKey(e store.Entry) bool {
	switch e.Type {
	case "OpenSSH", "RSA", "EC", "PKCS#8":
		return true
	}

	return false
}

// derivePublic reads the private key, so it needs the same prompt as
// revealing it.
func (a *App) derivePublic(e store.Entry, then func()) {
	if a.publicID == e.ID {
		then()
		return
	}

	a.authorized(fmt.Sprintf("read the public key of “%s”", e.Name), func() {
		value, err := a.store.Get(e.ID)
		if err != nil {
			a.notice = err.Error()
			return
		}

		key, err := pubkey.Derive(value)
		if err != nil {
			a.notice = err.Error()
			return
		}

		a.public, a.publicID = key, e.ID

		then()
	})
}

func (a *App) publicKeyCard(c *ui.Context, e store.Entry) {
	t := c.Theme()
	shown := a.publicID == e.ID
	services := c.Services()

	widget.Card(c, func() {
		ui.Row(c).Wrap().Padding(14, 14, 14, 18).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Public key").FontWeight(600).Grow(1).SingleLine()

			if !shown {
				ui.Button(c, "").Label("Show public key").OnClick(func() {
					a.derivePublic(e, func() {})
				}).Children(func() {
					ui.Icon(c, icon.Eye).FontSize(15)
					ui.Text(c, "Show").SingleLine()
				})
			}

			// The public key isn't secret, so the clipboard keeps it.
			ui.Button(c, "").Label("Copy public key").OnClick(func() {
				a.derivePublic(e, func() {
					services.WriteClipboard(a.public.Authorized)
					a.notice = "Copied the public key."
				})
			}).Children(func() {
				ui.Icon(c, icon.Copy).FontSize(15)
				ui.Text(c, "Copy").SingleLine()
			})
		})

		if !shown {
			return
		}

		ui.Divider(c)
		widget.InfoRow(c, "Fingerprint", a.public.Fingerprint)
		ui.Divider(c)

		ui.Column(c).Padding(18).Background(t.Surface).Children(func() {
			ui.Text(c, a.public.Authorized).Font("monospace").FontSize(13).LineHeight(1.5).Selectable()
		})
	})
}
