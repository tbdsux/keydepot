# Architecture

KeyDepot is a Go app using [MyGo](https://github.com/egoist/mygo)'s native UI, with no web view and no cgo. `AGENTS.md` holds the project layout and coding rules, and this page explains how the pieces work together.

## Layers

Dependencies point one way: `app` → `ui` → `lib`.

| Layer | Job |
|---|---|
| `app/` | Starts up and wires dependencies |
| `ui/view` | The window: `App` holds the state, and its views are rebuilt from it |
| `ui/tray`, `ui/update` | Tray icon and close-to-tray, and update checks |
| `ui/widget`, `ui/theme`, `ui/icon` | Shared building blocks, colors and font, icons |
| `lib/store` | The key list and projects |
| `lib/vault` | The encrypted file and protection modes |
| `lib/crypt` | AES-GCM and Argon2id |
| `lib/export` | `.keydepot` export files |
| `lib/dotenv` | Reading and writing `.env` text |
| `lib/pubkey` | The public half and fingerprint of an SSH private key |
| `lib/autolock` | Locking after the computer sits idle, and saving that choice |
| `lib/auth` | The macOS system prompt, called through `purego`; other systems don't offer it |
| `lib/durable` | Crash-safe file writes |
| `lib/shortcut`, `lib/paths` | Saved shortcuts, and where files live |

## Startup

[`app/main.go`](../app/main.go) does the following:

1. Takes the single-instance lock. A second copy brings the first to the front and exits, because two copies would overwrite each other's vault changes.
2. Starts the updater.
3. When the app is ready, opens the vault and then the store. An unreadable file shows an error rather than starting empty, because the next save would overwrite it.
4. Creates the window and the tray icon, and locks on sleep, screen lock and idle time.

The app then shows one of three screens:

- the load error, if a file couldn't be read
- the protection picker, if there's no vault yet, which leaves out the system prompt where it isn't available
- the main window

## Store and vault

Two files hold the data, and each has its own job:

- **`keys.json`** (`store.Store`) lists names, types and projects, unencrypted, so the app can show them while locked.
- **`vault.keydepot`** (`vault.Vault`) holds the encrypted keys and their notes. `store.Secrets` is the interface between them.

The vault keeps its data key in memory from unlocking until `Lock`. Its whole file is rewritten on each change. That's why individual keys are capped at 1 MiB.

## Unlocking

Actions that need a key go through three checks in [`ui/view`](../ui/view/unlock.go):

- **`authorized`** passes if the app is already unlocked, and otherwise prompts.
- **`reauth`** locks first and then prompts, for sensitive actions such as exporting.
- **`writable`** only prompts in password mode, because the other modes can get the data key from the Keychain without one. Adding and importing keys use it.

`prompt` picks what to ask for based on the vault's mode.

## Writing safely

[`durable.WriteFile`](../lib/durable/durable.go) writes to a temporary file, syncs it, renames it over the original and syncs the folder, so a crash leaves either the old file or the new one.

Vault and store changes are also ordered so a failure can't lose a key:

- **Adding:** the vault is written first, then the list. If the list can't be saved, the new item is removed. A crash between the two writes can leave an unlisted encrypted item in the vault, which is harmless.
- **Replacing:** the old key is restored if the list can't be saved.
- **Deleting:** the vault item goes first, so a failure leaves a key you can still see and retry, never one hidden in the vault.
- **Importing:** every key is checked before any is added.

## Threading

The view runs on the UI thread and can rebuild several times for one event, so views must not have side effects.

Slow work, such as Argon2id and system prompts, goes through `App.run`, which does the work on a goroutine and then returns to the UI thread with `Window.Update`. Rules for element lifetimes and identity are in [`.agents/skills/mygo-maintenance/SKILL.md`](../.agents/skills/mygo-maintenance/SKILL.md).

## Tests

`CGO_ENABLED=0 go test ./...` covers `lib/autolock`, `lib/crypt`, `lib/dotenv`, `lib/export`, `lib/pubkey`, `lib/shortcut`, `lib/store`, `lib/vault`, `ui/update` and `ui/view`. The view tests drive the UI with MyGo's `ui.NewTester`.

CI runs them on macOS, Windows and Linux. `lib/durable` and `lib/auth` have no tests of their own, and the system prompt and each system's secret store need checking by hand on a real computer.
