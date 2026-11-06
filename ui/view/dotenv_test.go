package view

import (
	"os"
	"path/filepath"
	"testing"
)

func TestViewEnvImport(t *testing.T) {
	a, _, _, _, tt := newVaultApp(t, true)
	click(t, tt, "Continue")

	if _, err := a.store.AddProject("Web"); err != nil {
		t.Fatal(err)
	}

	a.page = projectPage("Web")
	tt.Frame()

	dir := t.TempDir()
	first, second := filepath.Join(dir, "a.env"), filepath.Join(dir, ".env")
	os.WriteFile(first, []byte("API_KEY=abc\nEMPTY=\n"), 0o600)
	os.WriteFile(second, []byte("PEM=\"-----BEGIN-----\\nx\\n-----END-----\"\n"), 0o600)
	a.pickEnv = func() ([]string, error) { return []string{first, second}, nil }

	a.openEnvImport()
	tt.Frame()
	click(t, tt, "Choose files…")

	if !tt.HasText("2 keys found. 1 empty values are skipped.") {
		t.Fatalf("after choosing files: %q", tt.Texts())
	}

	click(t, tt, "Import")

	if a.envOpen || a.envDraft != "" || len(a.store.Entries) != 2 {
		t.Fatalf("open %v, draft %q, entries %v", a.envOpen, a.envDraft, a.store.Entries)
	}

	for _, e := range a.store.Entries {
		if e.Project != "Web" {
			t.Errorf("%s filed in %q", e.Name, e.Project)
		}
	}
}

func TestEnvImportRefusesBadText(t *testing.T) {
	a, _, _, _, tt := newVaultApp(t, true)
	click(t, tt, "Continue")

	a.openEnvImport()
	a.envDraft = "API_KEY=abc\nnot a line"
	tt.Frame()

	if !tt.HasText("Line 2 has no “=”.") {
		t.Errorf("no parse error shown: %q", tt.Texts())
	}

	a.saveEnvImport()

	if len(a.store.Entries) != 0 || !a.envOpen {
		t.Errorf("imported from bad text: %v", a.store.Entries)
	}

	a.Lock()

	if a.envOpen || a.envDraft != "" {
		t.Error("locking kept the .env draft")
	}
}
