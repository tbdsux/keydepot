// Package update keeps KeyDepot up to date from its signed releases.
package update

import (
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater/native"
)

// Start checks for updates in the background and offers them in a window. A
// beta follows later betas, while a stable version only takes stable ones.
func Start() {
	mygo.Updater.SetPrereleases(isPrerelease(mygo.App.Version()))
	mygo.Use(native.Plugin)
}

// isPrerelease reads a semantic version, where only build metadata follows a "+".
func isPrerelease(version string) bool {
	version, _, _ = strings.Cut(version, "+")

	return strings.Contains(version, "-")
}
