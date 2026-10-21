package archdoc_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc"
)

// shop is a writable copy of the drift fixture, in a directory named shop: the model is named
// after its directory, so two readings compare only if they share it.
func shop(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "shop")
	from := filepath.Join("testdata", "drift", "base")
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

// rename turns the shop's cache into a store — a rename, as a commit would.
func rename(t *testing.T, dir string) {
	t.Helper()
	p := filepath.Join(dir, "docker-compose.yml")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, r := range [][2]string{{"  cache:\n    image: redis:7", "  store:\n    image: redis:7"}, {"      - cache\n", "      - store\n"}, {"CACHE_HOST: cache", "CACHE_HOST: store"}} {
		if !strings.Contains(s, r[0]) {
			t.Fatalf("fixture changed: %q is gone", r[0])
		}
		s = strings.Replace(s, r[0], r[1], 1)
	}
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The platform's worker documents a throwaway clone and keeps history itself: without History,
// Generate writes the documentation, says nothing unless given a log, and leaves no history behind.
func TestGenerateWithoutHistoryRecordsNothing(t *testing.T) {
	dir := shop(t)
	g, err := archdoc.Generate(context.Background(), dir, archdoc.GenerateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(g.Root) || len(g.Model.Nodes) == 0 || g.Index == "" {
		t.Errorf("root %q, %d nodes, index %d bytes", g.Root, len(g.Model.Nodes), len(g.Index))
	}
	if !slices.Contains(g.Files, archdoc.ModelFile) {
		t.Errorf("files written do not include %s: %v", archdoc.ModelFile, g.Files)
	}
	if g.Version != 0 || g.Changed {
		t.Errorf("a version was reported without history: %d, %v", g.Version, g.Changed)
	}
	if _, err := os.Stat(filepath.Join(dir, ".archdoc", "history.db")); !os.IsNotExist(err) {
		t.Errorf("history.db exists without History: %v", err)
	}
	written, err := os.ReadFile(filepath.Join(dir, archdoc.ModelFile))
	if err != nil {
		t.Fatal(err)
	}
	if returned, err := g.Model.JSON(); err != nil || !bytes.Equal(written, returned) {
		t.Errorf("the model written is not the one returned (%v)", err)
	}
}

// With History, a version is recorded once: a second run over the same architecture records
// nothing (decisions, 2026-09-10).
func TestGenerateWithHistoryRecordsAVersionOnce(t *testing.T) {
	dir := shop(t)
	var log bytes.Buffer
	o := archdoc.GenerateOptions{Log: &log, History: true}
	first, err := archdoc.Generate(context.Background(), dir, o)
	if err != nil {
		t.Fatal(err)
	}
	second, err := archdoc.Generate(context.Background(), dir, o)
	if err != nil {
		t.Fatal(err)
	}
	if first.Version != 1 || !first.Changed || second.Version != 1 || second.Changed {
		t.Errorf("first %d %v, second %d %v", first.Version, first.Changed, second.Version, second.Changed)
	}
	if !strings.Contains(log.String(), "recorded version 1") || !strings.Contains(log.String(), "architecture unchanged since version 1") {
		t.Errorf("the log does not say so:\n%s", log.String())
	}
}

// A paid run goes ahead only if its Consent agrees, and is shown what it would send first. With no
// Consent, or one that stops, nothing is sent and nothing is written.
func TestPaidRunsNeedConsent(t *testing.T) {
	dir := shop(t)
	_, err := archdoc.Generate(context.Background(), dir, archdoc.GenerateOptions{Label: &archdoc.LabelOptions{Model: archdoc.LabelModel}})
	if err == nil {
		t.Fatal("a label run with no Consent went ahead")
	}

	var asked archdoc.Request
	stop := func(r archdoc.Request) error { asked = r; return archdoc.ErrStopped }
	_, err = archdoc.Generate(context.Background(), dir, archdoc.GenerateOptions{Label: &archdoc.LabelOptions{Model: archdoc.LabelModel, Consent: stop}})
	if !errors.Is(err, archdoc.ErrStopped) {
		t.Fatalf("a stop came back as %v", err)
	}
	if asked.What != "label" || len(asked.Plans) != 1 || asked.Bytes == 0 || asked.Sends == "" {
		t.Errorf("the consent was asked %+v", asked)
	}
	if _, err := os.Stat(filepath.Join(dir, archdoc.DocsDir)); !os.IsNotExist(err) {
		t.Error("a refused run wrote the documentation")
	}
}

// The plan's step 1: comparing two models given directly — as the platform holds them, with no
// clone — equals comparing the commits they came from.
func TestComparingStoredModelsEqualsDiffingTheirCommits(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	ctx := context.Background()

	repo := shop(t)
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	git("add", "-A")
	git("commit", "-q", "-m", "first")
	rename(t, repo)
	git("commit", "-q", "-am", "second")
	viaGit, err := archdoc.Diff(ctx, repo, "HEAD~1", "HEAD")
	if err != nil {
		t.Fatal(err)
	}

	// The same two states documented separately, as the worker would at each commit, and only
	// their models kept.
	stored := func(edit func(*testing.T, string)) archdoc.Model {
		dir := shop(t)
		if edit != nil {
			edit(t, dir)
		}
		g, err := archdoc.Generate(ctx, dir, archdoc.GenerateOptions{})
		if err != nil {
			t.Fatal(err)
		}
		return g.Model
	}
	direct := archdoc.Compare(stored(nil), stored(rename))

	if len(viaGit.Drift) == 0 {
		t.Fatal("the rename was not seen between the commits")
	}
	if !reflect.DeepEqual(direct.Drift, viaGit.Drift) {
		t.Errorf("models compared directly:\n  %+v\ncommits compared:\n  %+v", direct.Drift, viaGit.Drift)
	}
	if !reflect.DeepEqual(direct.Summary(), viaGit.Summary()) {
		t.Errorf("summaries differ:\n  %+v\n  %+v", direct.Summary(), viaGit.Summary())
	}
}
