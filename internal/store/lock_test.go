package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// One run holds a repository; what it is doing can be read; a second waits, told who for, until
// the first lets go.
func TestOneUpdateAtATime(t *testing.T) {
	root := t.TempDir()
	first, err := Take(context.Background(), root, "archdoc generate", nil)
	if err != nil {
		t.Fatal(err)
	}
	first.Report(core.Progress{Doing: "reading NestJS in server"})
	if b, ok := Running(root); !ok || b.By != "archdoc generate" || b.Doing != "reading NestJS in server" || b.PID != os.Getpid() {
		t.Fatalf("running: %+v %v", b, ok)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	var waitedFor string
	if _, err := Take(ctx, root, "the app", func(b Busy) { waitedFor = b.By }); err == nil || waitedFor != "archdoc generate" {
		t.Fatalf("a second run did not wait: %v, told %q", err, waitedFor)
	}

	go func() { time.Sleep(200 * time.Millisecond); first.Release() }()
	second, err := Take(context.Background(), root, "the app", nil)
	if err != nil {
		t.Fatal(err)
	}
	second.Release()
	if _, ok := Running(root); ok {
		t.Error("released, and still running")
	}
}

// A lock whose run is gone is taken over, not waited on for ever.
func TestALockLeftByADeadRunIsTakenOver(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".archdoc"), 0o755)
	os.WriteFile(filepath.Join(root, LockFile), []byte(`{"pid": 2147483000, "by": "archdoc generate", "since": "2026-10-22T09:00:00Z"}`), 0o644)
	if _, ok := Running(root); ok {
		t.Fatal("a dead run counted as running")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	l, err := Take(ctx, root, "archdoc generate", func(Busy) { t.Error("waited for a dead run") })
	if err != nil {
		t.Fatal(err)
	}
	l.Release()
}
