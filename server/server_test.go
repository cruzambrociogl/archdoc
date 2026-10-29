package server

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/cruzambrociogl/archdoc"
	"github.com/cruzambrociogl/archdoc/internal/serve"
)

// shop is a writable copy of the drift fixture.
func shop(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "shop")
	from := filepath.Join("..", "testdata", "drift", "base")
	err := filepath.WalkDir(from, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(from, p)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dir, rel), 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, rel), b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// The app's update is generate: a plain one records a version, as the command does; a dry run puts
// label's and explain's requests to Agree, as the command's --dry-run would print them, and sends
// and writes nothing; and a paid run Agree refuses sends nothing.
func TestTheAppsUpdateIsGenerate(t *testing.T) {
	ctx := context.Background()
	dir := shop(t)

	var asks []serve.Ask
	agree := func(a serve.Ask) error { asks = append(asks, a); return nil }
	if _, err := update(ctx, dir, serve.Run{Label: true, Explain: true, DryRun: true, Agree: agree, Log: io.Discard}); !errors.Is(err, serve.ErrStopped) {
		t.Fatalf("a dry run came back as %v", err)
	}
	if len(asks) != 2 || asks[0].What != "label" || asks[1].What != "explain" || asks[1].Requests == 0 || asks[0].Model == "" || !asks[0].Known {
		t.Errorf("asked %+v", asks)
	}
	if _, err := os.Stat(filepath.Join(dir, ".archdoc")); !os.IsNotExist(err) {
		t.Error("a dry run wrote .archdoc")
	}

	out, err := update(ctx, dir, serve.Run{Log: io.Discard})
	if err != nil || out.Version != 1 || !out.Changed {
		t.Fatalf("a plain update: %+v %v", out, err)
	}
	if _, ok, _ := archdoc.CommittedModel(dir); !ok {
		t.Error("a plain update wrote no model")
	}

	refuse := func(serve.Ask) error { return serve.ErrStopped }
	if _, err := update(ctx, dir, serve.Run{Label: true, Agree: refuse, Log: io.Discard}); !errors.Is(err, serve.ErrStopped) {
		t.Errorf("a refused label run came back as %v", err)
	}
	if _, err := update(ctx, dir, serve.Run{Explain: true, Log: io.Discard}); err == nil {
		t.Error("a paid run with no Agree went ahead")
	}
}
