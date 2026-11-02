# KeyDepot

A desktop app that keeps your private keys encrypted, on your own computer. Store SSH, PGP and other private keys in one vault, group them by project, and copy one when you need it.

> **Beta.** KeyDepot hasn't had an independent security review. Keep another backup of any key you can't afford to lose.

## Features

- **Encrypted vault:** every key is sealed with AES-256-GCM in a single local file.
- **Your choice of protection:** Touch ID or your Mac password, a KeyDepot password with a recovery code, or no extra prompt.
- **Projects and search:** group keys, search them, and do everything from the command palette.
- **Careful with secrets:** the clipboard clears after 60 seconds, and KeyDepot locks when your Mac sleeps or locks or when you close the window.
- **Encrypted export and import:** move keys between computers in a password-protected file.
- **Signed updates:** the app checks for new versions and verifies them before installing.

KeyDepot has no account and no cloud sync. Its only network traffic is checking for and downloading updates.

## Install

KeyDepot needs macOS 12 or later. The code also compiles for Windows and Linux, but those builds are untested and aren't released.

> **KeyDepot isn't signed or notarized by Apple yet.** macOS can't confirm who made the app, so it blocks the first launch and you have to allow it yourself. Only do this with a copy downloaded from this repository's [Releases](https://github.com/tbdsux/keydepot/releases) page.

### 1. Download

Download the disk image for your Mac from [Releases](https://github.com/tbdsux/keydepot/releases):

| Your Mac | File |
|---|---|
| Apple Silicon (M1 or later) | `keydepot-<version>-arm64.dmg` |
| Intel | `keydepot-<version>-amd64.dmg` |

To check which you have, open the Apple menu and choose **About This Mac**.

You can also compare the download with the release's `SHA256SUMS` file. This catches a broken download, but `SHA256SUMS` isn't signed, so it doesn't prove who made the file:

```bash
shasum -a 256 ~/Downloads/keydepot-*.dmg
```

### 2. Install

Open the disk image and drag KeyDepot to Applications.

### 3. Allow the first launch

1. Open KeyDepot from Applications. macOS blocks it, so close the message.
2. Go to **System Settings → Privacy & Security** and scroll down to **Security**.
3. Choose **Open Anyway** next to the message about KeyDepot, then confirm with your Mac password or Touch ID.

If there's no **Open Anyway** button, or macOS says the app is damaged, remove the download's quarantine flag instead, then open the app again:

```bash
xattr -dr com.apple.quarantine /Applications/KeyDepot.app
```

You only need to do this for the first launch.

### Updating

KeyDepot offers new versions itself and checks their signature before installing them. See [Updates](docs/usage.md#updates).

Because the app isn't signed by Apple, macOS may ask again for access to KeyDepot's Keychain item after an update, unless you use a KeyDepot password. Choose **Always Allow** if you trust the update. Those modes have no recovery code, so keep a recent [export](docs/usage.md#backups-and-moving-computers) in case the vault doesn't open.

### Uninstalling

Export any keys you want to keep, then quit KeyDepot from its menu and move the app to the Trash.

Your vault stays in `~/Library/Application Support/KeyDepot/` until you delete that folder. **Deleting it can't be undone, and it removes your keys.**

In the Touch ID and No extra prompt modes, the vault's key also stays in your login Keychain, under `dev.tbdsux.keydepot`. You can remove it with Keychain Access.

## Build from source

You need Go 1.27.3 or later. Run MyGo from the project root:

```bash
go tool mygo dev
```

```bash
go tool mygo build
```

To run the checks:

```bash
CGO_ENABLED=0 go test ./...
```

```bash
go tool mygo vet .
```

## Documentation

- [Using KeyDepot](docs/usage.md): protection modes, managing keys, shortcuts, backups
- [Security](docs/security.md): how keys are encrypted, what that protects against, and what it doesn't
- [Architecture](docs/architecture.md): how the code is organized and how the app runs

## Reporting a security problem

Please don't open a public issue for a vulnerability. Use GitHub's [private vulnerability reporting](https://github.com/tbdsux/keydepot/security/advisories/new) instead.

## Credits

Built with [MyGo](https://github.com/egoist/mygo). Uses the [Figtree](https://github.com/erikdkennedy/figtree) font (SIL Open Font License) and [Lucide](https://lucide.dev) icons (ISC License).

## License

[MIT](LICENSE)
