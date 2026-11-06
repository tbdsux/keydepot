// Package autolock locks KeyDepot once the computer has sat idle a while.
package autolock

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"keydepot/lib/durable"
)

// Default is how long the computer sits idle before KeyDepot locks, unless
// the user chose otherwise.
const Default = 15 * time.Minute

// maxMinutes caps a hand-edited file, whose huge value would overflow into never.
const maxMinutes = 24 * 60

// checkEvery bounds how late a lock comes.
const checkEvery = 10 * time.Second

// AutoLock holds the user's choice, which the UI changes while Watch reads it.
type AutoLock struct {
	path  string
	after atomic.Int64
}

type file struct {
	// LockAfterMinutes is 0 for never.
	LockAfterMinutes int `json:"lockAfterMinutes"`
}

// Open falls back to Default when the file can't be read: it holds only a
// preference.
func Open(path string) (*AutoLock, error) {
	l := &AutoLock{path: path}
	l.after.Store(int64(Default))

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return l, nil
	}

	if err != nil {
		return l, err
	}

	saved := file{LockAfterMinutes: int(Default / time.Minute)}
	if err := json.Unmarshal(data, &saved); err != nil {
		return l, fmt.Errorf("%s: %w", path, err)
	}

	l.after.Store(int64(min(max(saved.LockAfterMinutes, 0), maxMinutes)) * int64(time.Minute))

	return l, nil
}

// After is how long the computer sits idle before KeyDepot locks, or 0 for never.
func (l *AutoLock) After() time.Duration { return time.Duration(l.after.Load()) }

// SetAfter keeps d for this run even when it can't be saved.
func (l *AutoLock) SetAfter(d time.Duration) error {
	l.after.Store(int64(d))

	data, err := json.MarshalIndent(file{LockAfterMinutes: int(d / time.Minute)}, "", "  ")
	if err != nil {
		return err
	}

	return durable.WriteFile(l.path, data)
}

// Watch calls lock, off the UI thread, once per stretch of idle time longer
// than After. Systems that don't report idle time never lock this way.
func (l *AutoLock) Watch(lock func()) {
	go func() {
		locked := false

		for range time.Tick(checkEvery) {
			after := l.After()
			idle := after > 0 && mygo.Power.IdleTime() >= after

			if idle && !locked {
				lock()
			}

			locked = idle
		}
	}()
}
