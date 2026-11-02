// Package durable writes files that survive a crash whole.
package durable

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// WriteFile replaces path so that a crash leaves the old file or the
// new one. File.Sync is F_FULLFSYNC on macOS.
func WriteFile(path string, data []byte) error {
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}

	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}

	if err := errors.Join(err, f.Close()); err != nil {
		os.Remove(tmp)
		return err
	}

	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}

	// Windows can't sync a folder.
	if runtime.GOOS == "windows" {
		return nil
	}

	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}

	defer dir.Close()

	return dir.Sync()
}
