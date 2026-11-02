package view

import (
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
	"keydepot/lib/shortcut"
	"keydepot/lib/store"
	"keydepot/ui/icon"
	"keydepot/ui/widget"
)

func (a *App) paletteButton(c *ui.Context) {
	t := c.Theme()
	cmd := commands[0]

	ui.Column(c).Padding(0, 10, 10, 10).Children(func() {
		ui.Button(c, "").
			Label("Search and commands").
			FillWidth().
			OnClick(func() { a.paletteOpen = true }).
			Children(func() {
				ui.Icon(c, icon.Search).FontSize(15).TextColor(t.TextMuted)
				ui.Text(c, "Search…").TextColor(t.TextMuted).SingleLine().Grow(1).Shrink(1)

				if s := a.shortcutFor(cmd); s.Key != ui.KeyUnknown {
					widget.Keycap(c, s.Label(), t.TextMuted, t.Background, t.Border)
				}
			})
	})
}

// paletteItem is one row of the palette.
type paletteItem struct {
	title, detail string
	icon          *ui.SVG
	keys          shortcut.Shortcut
	run           func()
}

// paletteItems lists what can be run now, in order: commands, keys, projects.
func (a *App) paletteItems(c *ui.Context, target *store.Entry) []paletteItem {
	var items []paletteItem

	for _, cmd := range commands {
		if cmd.id == "palette" || cmd.needsKey && target == nil {
			continue
		}

		item := paletteItem{title: cmd.title, icon: cmd.icon, keys: a.shortcutFor(cmd)}

		var e store.Entry

		if cmd.needsKey {
			e, item.detail = *target, target.Name
		}

		item.run = func() { cmd.run(a, c, e) }
		items = append(items, item)
	}

	for i, e := range a.store.Entries {
		detail := e.Type + " key"

		if e.Project != "" {
			detail += " · " + e.Project
		}

		items = append(items, paletteItem{title: e.Name, detail: detail, icon: icon.Key, run: func() {
			a.page, a.query, a.chosen = "keys", "", i
			a.list.ScrollIntoView(i)
		}})
	}

	for _, p := range a.store.Projects {
		items = append(items, paletteItem{title: p, detail: "Project", icon: icon.Folder, run: func() {
			a.page, a.query, a.chosen = projectPage(p), "", 0
		}})
	}

	return filterPalette(items, a.paletteQuery)
}

// filterPalette keeps the items holding every word of the query.
func filterPalette(items []paletteItem, query string) []paletteItem {
	words := strings.Fields(strings.ToLower(query))
	return slices.DeleteFunc(items, func(item paletteItem) bool {
		text := strings.ToLower(item.title + " " + item.detail)
		return slices.ContainsFunc(words, func(word string) bool { return !strings.Contains(text, word) })
	})
}

func (a *App) commandPalette(c *ui.Context, target *store.Entry) {
	if !a.paletteOpen {
		a.paletteQuery, a.paletteChosen = "", 0
		return
	}

	t := c.Theme()
	items := a.paletteItems(c, target)
	a.paletteChosen = max(min(a.paletteChosen, len(items)-1), 0)

	runItem := func(i int) {
		if i < len(items) {
			a.paletteOpen = false

			items[i].run()
		}
	}

	move := func(by int) {
		if len(items) > 0 {
			a.paletteChosen = (a.paletteChosen + by + len(items)) % len(items)
			a.paletteList.ScrollIntoView(a.paletteChosen)
		}
	}

	w, h := c.Size()

	ui.DialogBase(c, &a.paletteOpen, func(back, panel ui.Element) {
		back.Background(ui.RGBA(0, 0, 0, 0.32)).Justify(ui.Start).Padding(h*0.12, 0, 0, 0)
		panel.Width(min(640, w-48)).
			Height(min(420, h*0.76)).
			Radius(16).
			Background(t.Background).
			Border(1, t.Border).
			Clip()
		panel.Shadow(0, 18, 50, 0, ui.RGBA(0, 0, 0, 0.35))

		ui.Row(c).Padding(16, 20).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icon.Search).FontSize(18).TextColor(t.TextMuted)

			input := ui.TextInputBase(c, &a.paletteQuery).
				Label("Search keys and commands").
				Placeholder("Search keys and commands…").
				FontSize(18).
				Grow(1).
				AutoFocus()

			input.OnChange(func() {
				a.paletteChosen = 0
				a.paletteList.ScrollIntoView(0)
			})

			input.OnSubmit(func() { runItem(a.paletteChosen) })
			input.OnShortcut(0, ui.KeyDown, func() { move(1) })
			input.OnShortcut(0, ui.KeyUp, func() { move(-1) })

			if s := a.shortcutFor(commands[0]); s.Key != ui.KeyUnknown {
				input.OnShortcut(s.Mods, s.Key, func() { a.paletteOpen = false })
			}
		})

		ui.Divider(c)

		a.paletteList.Selected = &a.paletteChosen
		a.paletteList.Label = func(i int) string { return items[i].title }

		ui.List(c, &a.paletteList, len(items), func(i int) {
			paletteRow(c, items[i], i == a.paletteChosen, func() { runItem(i) })
		}).
			Grow(1).
			Gap(2).
			Padding(8).
			OnSubmit(func() { runItem(a.paletteChosen) }).
			Children(func() {
				if len(items) == 0 {
					ui.Text(c, "No results").TextColor(t.TextMuted).Padding(16, 12)
				}
			})

		ui.Divider(c)

		ui.Row(c).Padding(10, 20).Gap(16).AlignItems(ui.Center).Background(t.Surface).Children(func() {
			for _, hint := range [][2]string{{"↑↓", "Navigate"}, {"↩", "Run"}, {"esc", "Close"}} {
				ui.Row(c.Key(hint[1])).Gap(6).AlignItems(ui.Center).Children(func() {
					widget.Keycap(c, hint[0], t.TextMuted, t.Background, t.Border)
					ui.Text(c, hint[1]).FontSize(13).TextColor(t.TextMuted)
				})
			}
		})
	})
}

func paletteRow(c *ui.Context, item paletteItem, selected bool, run func()) {
	t := c.Theme()
	// The list paints the chosen row in the accent color.
	text, muted, accent, capBg, capBorder := t.Text, t.TextMuted, t.Accent, t.Surface, t.Border

	if selected {
		text, muted, accent = t.AccentText, t.AccentText.Alpha(0.8), t.AccentText
		capBg, capBorder = t.AccentText.Alpha(0.16), t.AccentText.Alpha(0.2)
	}

	ui.Row(c).Padding(10, 12).Gap(12).AlignItems(ui.Center).OnClick(run).Children(func() {
		ui.Icon(c, item.icon).FontSize(16).TextColor(accent)
		ui.Text(c, item.title).FontWeight(500).TextColor(text).SingleLine().Ellipsis("…").Grow(1).Shrink(1)

		if item.detail != "" {
			ui.Text(c, item.detail).FontSize(13).TextColor(muted).SingleLine().Ellipsis("…").Shrink(1)
		}

		if item.keys.Key != ui.KeyUnknown {
			widget.Keycap(c, item.keys.Label(), muted, capBg, capBorder)
		}
	})
}
