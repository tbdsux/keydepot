# Using KeyDepot

## Choosing a protection

On first launch, KeyDepot asks what should open your keys. Your keys are encrypted in every mode. You can change the mode later in Settings.

| Mode | What opens your keys | If you lose access |
|---|---|---|
| **Touch ID or Mac password** (macOS only) | The system prompt. The vault's key stays in the macOS Keychain. | Keys are lost if the Keychain item is lost, unless you have an export. |
| **KeyDepot password** | A password of at least 8 characters, which KeyDepot doesn't store. | Your recovery code gets you back in. If you lose the password and the code, your keys are gone. |
| **No extra prompt** | Nothing beyond being logged in. The vault's key stays in your system's secret store. Anyone using your unlocked computer can copy your keys. | Keys are lost if the secret store's item is lost, unless you have an export. |

The secret store is the Keychain on macOS, Credential Manager on Windows and the Secret Service on Linux, such as GNOME Keyring or KWallet.

### Recovery code

Password mode gives you a recovery code, and KeyDepot shows it only once. You confirm you saved it by typing its last 4 characters.

Keep the code in a password manager, away from the vault and its backups. Anyone with both the code and the vault file can read your keys.

If you forget your password, choose **Forgot password?** on the unlock prompt and enter the code. You then set a new password and get a new code, and the old code stops working. Canceling partway keeps the old password and code.

Changing your password in **Settings** also gives you a new recovery code, and **New recovery code** asks for your password. Either way the old password and code stop working. A backup copy of the vault still opens with them, but holds only the keys it had then.

## When KeyDepot asks again

Once you pass the prompt, KeyDepot stays unlocked until it locks. It locks when:

- your computer sleeps or the screen locks (on Linux, when the screen saver starts)
- the computer goes unused for 15 minutes. Change this, or turn it off, in **Settings → Lock when idle**. It counts keyboard and mouse use in any app, and some Linux desktops don't report it, so it never locks there
- you close the window, which keeps KeyDepot running in the tray or menu bar
- on macOS, you press ⌘Q, which also closes to the menu bar rather than quitting
- you choose **Lock KeyDepot** from the app or the tray icon
- you quit, using **Quit KeyDepot** from the menu or the tray icon

These always ask again, even when unlocked: changing the protection, changing the password, making a new recovery code, exporting, and copying keys as `.env`.

## Managing keys

- **Add:** paste a private key and give it a name. KeyDepot stores it exactly as pasted, up to 1 MiB, and detects its type from the PEM header: OpenSSH, RSA, EC, PKCS#8, Encrypted PKCS#8 or PGP. Anything else shows as Other.
- **Reveal or copy:** both need you to be unlocked. Locking hides a revealed key.
- **Public key:** for SSH keys (OpenSSH, RSA, EC and PKCS#8), **Show** displays the public key and its SHA256 fingerprint, the same one `ssh-keygen -l` shows, and **Copy** copies the line to paste into `authorized_keys` or GitHub. This also needs you to be unlocked, as it reads the private key. The public key isn't secret, so the clipboard keeps it. An OpenSSH key with a passphrase still works, but a PEM key with one doesn't.
- **Notes:** each key can have notes, such as what it opens or where it's installed. They're encrypted in the vault like the key, so showing and saving them needs you to be unlocked, and exports include them.
- **Clipboard:** a copied key clears after 60 seconds, or when KeyDepot locks or quits, unless you've copied something else since. Clipboard history tools may keep their own copy.
- **Update:** rename a key, or paste a rotated key to replace it. Replacing can't be undone.
- **Delete:** this removes the key from the vault and can't be undone.
- **Projects:** a key belongs to one project or none. Deleting a project keeps its keys and moves them out of it.

## Shortcuts and the command palette

Press ⌘K to open the palette, which lists commands, keys and projects. You can change or remove any shortcut in Settings, except those kept for the system, such as ⌘C and ⌘Q. The shortcuts below are shown for macOS. On Windows and Linux, use Ctrl in place of ⌘ and Shift in place of ⇧.

| Command | Default |
|---|---|
| Open command palette | ⌘K |
| New key | ⌘N |
| New project | ⇧⌘N |
| Import .env | ⇧⌘I |
| Copy keys as .env | ⇧⌘E |
| Search keys | ⌘F |
| Show all keys | ⌘1 |
| Open settings | ⌘, |
| Lock KeyDepot | ⌘L |
| Copy key | ⇧⌘C |
| Reveal or hide key | ⇧⌘R |
| Update key | ⌘E |
| Delete key | ⌘⌫ (Ctrl+Backspace) |

## Backups and moving computers

Use **Settings → Export keys** to make a `.keydepot` file with its own password of at least 12 characters. Nothing can recover that password, so keep it apart from the file.

**Import keys** only adds. Imported keys get new IDs and never overwrite existing ones, so importing the same file twice gives you duplicates. Projects with the same name are merged, ignoring case.

## .env files

**Settings → Import .env**, or **Import .env** in the command palette, turns each `NAME=value` line into a key named `NAME`. Paste the text, or choose one or more files with **Choose files…**. Keys go into the project you pick, which starts as the project showing. Comments, `export ` and quoted values are understood, including double-quoted values over several lines. Variables with empty values are skipped, and a repeated name gives two keys. Nothing is added if any line can't be read.

On a project's page, **Copy as .env** copies its keys as one `.env` file. **Copy keys as .env** in the command palette does the same, or copies every key from outside a project page. It always asks again, and the clipboard clears after 60 seconds like a copied key. A name that isn't a valid variable name becomes upper snake case, so "Deploy key" is `DEPLOY_KEY`. KeyDepot never writes a `.env` file: save one yourself only where you need it, and keep it out of git.

Copying the vault file isn't a full backup in the Touch ID and No extra prompt modes, because the key that opens it stays in your system's secret store. Make an export instead.

## Where your files are

| System | Folder |
|---|---|
| macOS | `~/Library/Application Support/KeyDepot/` |
| Windows | `%AppData%\KeyDepot\` |
| Linux | `~/.config/KeyDepot/` |

Settings shows the paths of the first two files below.

| File | Contents |
|---|---|
| `vault.keydepot` | Your encrypted keys |
| `keys.json` | Key names, types, projects and dates, **unencrypted** |
| `shortcuts.json` | Your shortcut changes |
| `autolock.json` | How long the computer sits idle before KeyDepot locks |
| `updater.json` | Update preferences |
| `window-state.json` | Window size and position |

## Updates

KeyDepot checks 10 seconds after launch and then once a day while running. You can also choose **Check for Updates…** from the app or tray menu. When a new version is out, a window offers to install it, remind you later or skip that version.

A beta build also sees later betas. A stable build only sees stable releases. The Debian package on Linux doesn't update itself.
