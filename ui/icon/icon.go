// Package icon holds the interface's icons, embedded.
package icon

import (
	"embed"

	"github.com/egoist/mygo/ui"
)

//go:embed svg/*.svg
var files embed.FS

func load(name string) *ui.SVG {
	data, err := files.ReadFile("svg/" + name + ".svg")
	if err != nil {
		panic(err)
	}

	return ui.MustParseSVG(data)
}

// The interface's icons.
var (
	Key         = load("key-round")
	Settings    = load("settings")
	Plus        = load("plus")
	Eye         = load("eye")
	EyeOff      = load("eye-off")
	Copy        = load("copy")
	Trash       = load("trash-2")
	Shield      = load("shield-check")
	Fingerprint = load("fingerprint")
	Lock        = load("lock-keyhole")
	Folder      = load("folder")
	Pencil      = load("pencil")
	Search      = load("search")
)
