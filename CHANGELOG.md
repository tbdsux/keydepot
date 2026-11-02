# Changelog

Each release needs a `## <version>` section matching `version` in `mygo.json`, or its build fails. The section becomes the release's description and what the update window shows.

## 0.1.0

First release.

- Keep private keys in an encrypted vault, grouped by project
- Protect them with a KeyDepot password and recovery code, or on macOS with Touch ID or your Mac password
- Search keys and run commands from the command palette, with shortcuts you can change
- Export and import keys in password-protected files
- Update from within the app, with signed updates
- Builds for macOS, Windows and Linux. Windows and Linux are untested so far, and no build is signed by Apple or Microsoft yet
