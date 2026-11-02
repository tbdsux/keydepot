package shortcut

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestShortcutNamesRoundTrip(t *testing.T) {
	for key := range keyNames {
		want := Shortcut{ui.Ctrl | ui.Shift | ui.Alt | ui.Super, key}
		if got, ok := Parse(want.String()); !ok || got != want {
			t.Errorf("%q parsed as %v, %v", want, got, ok)
		}
	}

	if len(keyNames) != 26+10+12+24 {
		t.Errorf("%d key names: keys overlap", len(keyNames))
	}

	for _, text := range []string{"", "Super+", "Hyper+K", "Super+Nope"} {
		if s, ok := Parse(text); ok {
			t.Errorf("%q parsed as %v", text, s)
		}
	}
}
