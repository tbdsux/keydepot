package update

import "testing"

func TestIsPrerelease(t *testing.T) {
	for version, want := range map[string]bool{
		"":                   false,
		"0.1.0":              false,
		"1.0.0+build-5":      false,
		"0.1.0-beta.1":       true,
		"1.0.0-rc.1+build-5": true,
	} {
		if got := isPrerelease(version); got != want {
			t.Errorf("isPrerelease(%q) = %v, want %v", version, got, want)
		}
	}
}
