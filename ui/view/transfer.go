package view

import (
	"fmt"

	"keydepot/lib/export"
	"keydepot/lib/store"
)

func (a *App) exportKeys() {
	if len(a.store.Entries) == 0 {
		a.notice = "There are no keys to export."
		return
	}

	a.reauth("export your keys", func() {
		// Read now: an export missing a key must fail, not look complete.
		keys := make(map[string]string, len(a.store.Entries))

		for _, e := range a.store.Entries {
			value, err := a.store.Get(e.ID)
			if err != nil {
				a.notice = fmt.Sprintf("“%s” can't be read, so nothing was exported: %v", e.Name, err)
				return
			}

			keys[e.ID] = value
		}

		index := store.Index{Projects: a.store.Projects, Keys: a.store.Entries}

		var path string
		var err error

		a.run(func() { path, err = a.pickSave() }, func() {
			if err != nil {
				a.notice = err.Error()
				return
			}

			if path == "" {
				return
			}

			a.askFor(&passwordAsk{
				title: "Choose a password for this export", hint: "Only this password opens the file, so keep it apart from the file. Nothing can recover it.",
				action: "Export", label: "Export password", minLen: minExportPasswordLen,
				work: func(password string) error { return export.Write(path, password, index, keys) },
				then: func() { a.notice = fmt.Sprintf("Exported %d keys.", len(keys)) },
			})
		})
	})
}

func (a *App) importKeys() {
	var path string
	var err error

	a.run(func() { path, err = a.pickOpen() }, func() {
		if err != nil {
			a.notice = err.Error()
			return
		}

		if path == "" {
			return
		}

		a.writable("import keys", func() { a.askImportPassword(path) })
	})
}

func (a *App) askImportPassword(path string) {
	var index store.Index
	var keys map[string]string

	a.askFor(&passwordAsk{
		title: "Enter the export's password", hint: "Its keys are added to the ones you have.",
		action: "Import", label: "Export password",
		work: func(password string) (err error) {
			index, keys, err = export.Read(path, password)
			return err
		},
		then: func() {
			n, err := a.store.ImportKeys(index, keys)

			switch {
			case err != nil && n > 0:
				a.notice = fmt.Sprintf("Imported %d keys, then stopped: %v", n, err)

			case err != nil:
				a.notice = err.Error()

			default:
				a.notice = fmt.Sprintf("Imported %d keys.", n)
			}
		},
	})
}
