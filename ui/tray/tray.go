// Package tray keeps KeyDepot running in the menu bar or notification area.
package tray

import (
	_ "embed"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater"
)

//go:embed icon.png
var icon []byte

// Actions is what the tray's menu asks of the app. Each may touch its state.
type Actions struct {
	NewKey, Settings, Lock func()
}

// Keep adds the tray icon and makes closing win hide it and lock, so the app
// runs on until the user quits from a menu. Without an icon nothing changes.
func Keep(win *mygo.Window, do Actions) error {
	show := func() {
		win.Show()
		win.Focus()
	}

	_, err := mygo.NewTray(mygo.TrayOptions{
		ID:             "main",
		Icon:           icon,
		IconIsTemplate: true,
		ToolTip:        "KeyDepot",
		Menu: mygo.NewMenu([]*mygo.MenuItem{
			{Label: "Open KeyDepot", Click: func(*mygo.MenuItem, *mygo.Window) { show() }},
			{Label: "New Key…", Click: func(*mygo.MenuItem, *mygo.Window) {
				do.NewKey()
				show()
			}},
			{Label: "Settings…", Click: func(*mygo.MenuItem, *mygo.Window) {
				do.Settings()
				show()
			}},
			updater.MenuItem(),
			mygo.Separator(),
			{Label: "Lock KeyDepot", Click: func(*mygo.MenuItem, *mygo.Window) { do.Lock() }},
			mygo.Separator(),
			quitItem(),
		}),
	})
	if err != nil {
		return err
	}

	hide := func() {
		win.Hide()
		do.Lock()
	}

	quitting := false

	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { quitting = true })

	win.OnClose(func(e *mygo.CloseEvent) {
		if !quitting {
			e.PreventDefault()
			hide()
		}
	})

	// The Dock icon and opening the app again from Finder land here.
	mygo.App.OnActivate(func(hasVisibleWindows bool) {
		if !hasVisibleWindows {
			show()
		}
	})

	if runtime.GOOS == "darwin" {
		mygo.App.SetMenu(macMenu(hide))
	}

	return nil
}

// macMenu is the default menu with Cmd+Q hiding the window. Refusing the quit
// itself in OnBeforeQuit would also stop the Mac logging out or shutting down.
func macMenu(hide func()) *mygo.Menu {
	return mygo.NewMenu([]*mygo.MenuItem{
		{Label: "KeyDepot", Submenu: []*mygo.MenuItem{
			{Role: mygo.RoleAbout},
			updater.MenuItem(),
			mygo.Separator(),
			{Role: mygo.RoleServices},
			mygo.Separator(),
			{Role: mygo.RoleHide},
			{Role: mygo.RoleHideOthers},
			{Role: mygo.RoleUnhide},
			mygo.Separator(),
			{
				Label:       "Close to Menu Bar",
				Accelerator: "Cmd+Q",
				Click:       func(*mygo.MenuItem, *mygo.Window) { hide() },
			},
			quitItem(),
		}},
		{Role: mygo.RoleFileMenu},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleViewMenu},
		{Role: mygo.RoleWindowMenu},
	})
}

// quitItem really quits. It isn't RoleQuit, which would show Cmd+Q beside it.
func quitItem() *mygo.MenuItem {
	return &mygo.MenuItem{
		Label: "Quit KeyDepot",
		Click: func(*mygo.MenuItem, *mygo.Window) { mygo.App.Quit() },
	}
}
