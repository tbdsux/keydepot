package view

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"keydepot/lib/store"
	"keydepot/ui/icon"
	"keydepot/ui/widget"
)

// showNote reads the note, so it needs the same prompt as revealing the key.
func (a *App) showNote(e store.Entry) {
	a.authorized(fmt.Sprintf("show the note on “%s”", e.Name), func() {
		note, err := a.store.Note(e.ID)
		if err != nil {
			a.notice = err.Error()
			return
		}

		a.noteID, a.noteSaved, a.noteDraft = e.ID, note, note
	})
}

func (a *App) saveNote(e store.Entry) {
	draft := a.noteDraft

	a.writable(fmt.Sprintf("save the note on “%s”", e.Name), func() {
		// The note may have been hidden while the prompt was open.
		if a.noteID != e.ID {
			return
		}

		if err := a.store.SetNote(e.ID, draft); err != nil {
			a.notice = err.Error()
			return
		}

		a.noteSaved = draft
	})
}

func (a *App) hideNote() { a.noteID, a.noteSaved, a.noteDraft = "", "", "" }

func (a *App) noteCard(c *ui.Context, e store.Entry) {
	t := c.Theme()
	shown := a.noteID == e.ID

	widget.Card(c, func() {
		ui.Row(c).Wrap().Padding(14, 14, 14, 18).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(2).Grow(1).Shrink(1).Children(func() {
				ui.Text(c, "Notes").FontWeight(600).SingleLine()
				ui.Text(c, "Encrypted with the key.").FontSize(14).TextColor(t.TextMuted)
			})

			if !shown {
				ui.Button(c, "").Label("Show notes").OnClick(func() { a.showNote(e) }).Children(func() {
					ui.Icon(c, icon.Eye).FontSize(15)
					ui.Text(c, "Show").SingleLine()
				})

				return
			}

			ui.Button(c, "").Label("Hide notes").OnClick(a.hideNote).Children(func() {
				ui.Icon(c, icon.EyeOff).FontSize(15)
				ui.Text(c, "Hide").SingleLine()
			})

			if a.noteDraft != a.noteSaved {
				ui.PrimaryButton(c, "Save").Label("Save notes").OnClick(func() { a.saveNote(e) })
			}
		})

		if !shown {
			return
		}

		ui.Divider(c)

		ui.Column(c).Padding(12).Background(t.Surface).Children(func() {
			ui.TextArea(c, &a.noteDraft).
				Label("Notes").
				Placeholder("What this key opens, where it's installed, when to rotate it").
				Height(120).
				FillWidth()
		})
	})
}
