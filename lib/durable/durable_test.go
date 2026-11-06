package durable

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileIgnoresAPlantedTempFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "export.keydepot")
	planted := filepath.Join(dir, "planted")

	// The fixed name an older WriteFile used, pointing elsewhere.
	if err := os.Symlink(planted, path+".tmp"); err != nil {
		t.Fatal(err)
	}

	if err := WriteFile(path, []byte("sealed")); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(planted); !os.IsNotExist(err) {
		t.Errorf("wrote through the planted link: %v", err)
	}

	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}

	if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		t.Errorf("file mode %v", info.Mode())
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		t.Errorf("left behind: %v", entries)
	}
}
