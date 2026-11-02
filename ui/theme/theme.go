// Package theme holds KeyDepot's colors and font.
package theme

import (
	_ "embed"
	"log"

	"github.com/egoist/mygo/ui"
)

//go:embed fonts/Figtree.ttf
var figtree []byte

func init() {
	if err := ui.RegisterFont(figtree, "Figtree"); err != nil {
		log.Fatal(err)
	}
}

// New is KeyDepot's look, in dark or light.
func New(dark bool) *ui.Theme {
	t := ui.LightTheme()
	t.Background, t.Surface = ui.Hex("#fbfbfa"), ui.Hex("#f2f2f0")
	t.SurfaceHover, t.SurfacePressed = ui.Hex("#e9e9e6"), ui.Hex("#deded9")
	t.Border, t.Text, t.TextMuted = ui.Hex("#e3e3df"), ui.Hex("#171a19"), ui.Hex("#676d6a")
	t.Accent, t.AccentText = ui.Hex("#0f766e"), ui.Hex("#ffffff")
	t.AccentHover, t.AccentPressed = ui.Hex("#115e59"), ui.Hex("#134e4a")
	t.Selection, t.Focus = ui.Hex("#0f766e").Alpha(0.2), ui.Hex("#0f766e").Alpha(0.5)

	if dark {
		t = ui.DarkTheme()
		t.Background, t.Surface = ui.Hex("#111413"), ui.Hex("#1b1f1d")
		t.SurfaceHover, t.SurfacePressed = ui.Hex("#252a28"), ui.Hex("#2f3532")
		t.Border, t.Text, t.TextMuted = ui.Hex("#2a2f2d"), ui.Hex("#f1f3f2"), ui.Hex("#9aa19e")
		t.Accent, t.AccentText = ui.Hex("#2dd4bf"), ui.Hex("#04201d")
		t.AccentHover, t.AccentPressed = ui.Hex("#5eead4"), ui.Hex("#14b8a6")
		t.Selection, t.Focus = ui.Hex("#2dd4bf").Alpha(0.25), ui.Hex("#2dd4bf").Alpha(0.5)
	}

	// Spacing is the unit of every control's padding: 5 is the toolkit's "roomy".
	t.Font, t.FontSize, t.Radius, t.Spacing = "Figtree", 15, 10, 5

	return t
}
