package store

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// One update of a repository at a time (surface-spec §11.5): a run that writes takes the lock — a
// file in .archdoc saying who, since when, and what it is doing — and a second waits for it, and
// says so. The app reads the same file, and says an update is running in the terminal (§4.8). A
// lock left by a run that died — its process gone — is taken over.

// LockFile is where the lock is kept, beside the history.
const LockFile = ".archdoc/updating.json"

// Busy is the run updating a repository now.
type Busy struct {
	PID   int       `json:"pid"`
	By    string    `json:"by"` // "archdoc generate", "the app"
	Since time.Time `json:"since"`
	core.Progress
}

// Lock is a run's hold on a repository, until Release.
type Lock struct {
	path string
	busy Busy
}

// Take waits until no other run updates root, then holds it. waiting is told, once, who it waits
// for; ctx ending stops the wait.
func Take(ctx context.Context, root, by string, waiting func(Busy)) (*Lock, error) {
	path := filepath.Join(root, LockFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	l := &Lock{path: path, busy: Busy{PID: os.Getpid(), By: by, Since: time.Now().UTC()}}
	told := false
	for {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			f.Close()
			return l, l.write()
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		other, ok := Running(root)
		if !ok {
			os.Remove(path) // its run is gone: the lock is stale
			continue
		}
		if !told && waiting != nil {
			waiting(*other)
			told = true
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// Report records what the run is doing, for whoever looks: the app, a waiting run.
func (l *Lock) Report(p core.Progress) {
	l.busy.Progress = p
	l.write()
}

// Release lets the next run in.
func (l *Lock) Release() { os.Remove(l.path) }

func (l *Lock) write() error {
	b, err := json.Marshal(l.busy)
	if err != nil {
		return err
	}
	tmp := l.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, l.path)
}

// Running is the run updating root now, if there is one whose process is alive.
func Running(root string) (*Busy, bool) {
	b, err := os.ReadFile(filepath.Join(root, LockFile))
	if err != nil {
		return nil, false
	}
	var busy Busy
	if json.Unmarshal(b, &busy) != nil || busy.PID == 0 || !alive(busy.PID) {
		return nil, false
	}
	return &busy, true
}
