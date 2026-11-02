# KeyDepot

Desktop app built with [MyGo](https://github.com/egoist/mygo) native UI. It keeps private keys encrypted.

## Go development

Before any Go coding, review, debugging, troubleshooting, or setup task, load the `samber/cc-skills-golang@golang-how-to` skill first — it routes to whichever other Go skills the task needs.

For MyGo lifetime, identity and threading rules, read `.agents/skills/mygo-maintenance/SKILL.md` before changing anything in `ui/`.

## Project layout

```
app/          package main: start up and wire dependencies, nothing else
lib/          logic with no knowledge of any screen
  auth/         system prompt, Touch ID
  crypt/        sealing, key wrapping, Argon2id costs
  durable/      crash-safe file writes
  export/       .keydepot export files
  paths/        where KeyDepot's files live
  shortcut/     parsing, checking and saving shortcuts
  store/        key list, projects, importing
  vault/        encrypted vault, protection modes, recovery codes
ui/           everything on screen
  icon/         embedded SVG icons
  theme/        colors and embedded font
  tray/         tray icon, closing to the tray
  update/       checking for and offering updates
  widget/       stateless building blocks
  view/         App, its state and its screens
resources/    files shipped with the app; MyGo requires this at the project root
docs/         how the app works, for users and contributors; update it when behavior changes
```

## Separation of concerns

- One folder per concern, one concern per file. A file doing two jobs gets split; a new concern gets a new folder.
- Dependencies point one way: `app` → `ui` → `lib`. Nothing in `lib` imports `ui` or `app`. MyGo's own `ui` types are allowed where a value is one, as in `lib/shortcut`.
- `app/main.go` only wires dependencies. Logic belongs in `lib` or `ui`.
- Define an interface where it is used, as `store.Secrets` and `vault.Keychain` are.
- Keep Go's `embed` next to the files: `go:embed` can't reach up with `..`, so assets live inside the package that embeds them.
- Everything drawn from `App`'s state stays in `ui/view`: Go methods must live in their type's package, so split that package by file, not by folder.
- Export only what another package calls. Never export something just for a test: test through the public API, or put the test in the package.

## Writing style

Keep logic blocks apart with blank lines. Code must not read as one solid wall.

- Blank line before and after every `if`, `for`, `switch`, `select` and multi-line statement.
- Blank line before `return`, unless the block has two statements or fewer.
- Blank line when the kind of statement changes: declaration, assignment, call, `defer`.
- Keep a line attached only when it uses what the line above defined:

```go
key, err := crypt.Unwrap(w, password, crypt.RolePassword)
if err != nil {
	return err
}

v.unlocked(key)

return nil
```

- A line over 120 columns gets broken: chains after each `.`, calls and literals one field per line. Long user-facing strings may stay whole.
- Composite literals from another package use field names.
- Exported names get a doc comment, and each package one `// Package x` comment.
- User-facing error messages are full sentences because the UI shows them as written; that is why they are capitalized.

## Build and verify

The main package is set by `"main": "./app"` in `mygo.json`. Run MyGo from the project root, never from `app/`.

```bash
go tool mygo dev
```

```bash
go tool mygo build
```

```bash
CGO_ENABLED=0 go test ./...
```

```bash
go tool mygo vet .
```
