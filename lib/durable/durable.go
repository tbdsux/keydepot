// Package durable writes files that survive a crash whole.
package durable

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// WriteFile replaces path so that a crash leaves the old file or the
// new one. File.Sync is F_FULLFSYNC on macOS. The temporary file is new and
// only the user's, so nothing planted beside path, as in a shared folder,
// receives the data.
func WriteFile(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}

	tmp := f.Name()

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
