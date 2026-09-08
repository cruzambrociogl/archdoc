package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// copyFixture puts a fixture repository somewhere writable. archdoc writes into the repository
// it documents, and a test that wrote into testdata/ would leave the fixture dirty for every
// run after it.
func copyFixture(t *testing.T, name string) string {
	t.Helper()

	dst := t.TempDir()
	src := filepath.Join("..", "..", "testdata", name)

	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), b, 0o644)
	})
	if err != nil {
		t.Fatalf("copy fixture: %v", err)
	}
	return dst
}

func gen(t *testing.T, root string) string {
	t.Helper()

	var out bytes.Buffer
	if err := generate([]string{root}, &out); err != nil {
		t.Fatalf("generate: %v\n%s", err, out.String())
	}
	return out.String()
}

// OUT-02 and OUT-03, and the reason hard rule 2 exists: a documentation generator that eats
// someone's writing gets uninstalled once. This is the single most important test in the
// repository from a user's point of view.
func TestRegenerationNeverTouchesHumanSections(t *testing.T) {
	root := copyFixture(t, "gateway")
	gen(t, root)

	mine := filepath.Join(root, docsDir, "01-introduction-and-goals.md")
	written := "# 1. Introduction and Goals\n\nThis sentence is mine and must survive.\n"

	if err := os.WriteFile(mine, []byte(written), 0o644); err != nil {
		t.Fatal(err)
	}

	// Twice, because a bug that overwrites on the second run is the one that would ship.
	gen(t, root)
	gen(t, root)

	got, err := os.ReadFile(mine)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != written {
		t.Errorf("a human-owned section was modified:\n%s", got)
	}
}

// A generated section is the opposite: overwritten in full, every time, so a stale claim cannot
// survive a regeneration.
func TestGeneratedSectionsAreOverwritten(t *testing.T) {
	root := copyFixture(t, "gateway")
	gen(t, root)

	generated := filepath.Join(root, docsDir, "05-building-block-view.generated.md")
	if err := os.WriteFile(generated, []byte("stale nonsense\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	gen(t, root)

	got, err := os.ReadFile(generated)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "stale nonsense") {
		t.Error("a generated section survived regeneration")
	}
}

// The stubs are written once. On a second run nothing is created, and the report says so by
// staying quiet rather than repeating itself.
func TestStubsAreCreatedOnceOnly(t *testing.T) {
	root := copyFixture(t, "gateway")

	if first := gen(t, root); !strings.Contains(first, "7 section(s) created") {
		t.Errorf("the first run did not create the human sections:\n%s", first)
	}
	if second := gen(t, root); strings.Contains(second, "section(s) created") {
		t.Errorf("the second run created sections again:\n%s", second)
	}
}

// A repository archdoc has never seen must come out complete: twelve sections, an index, and
// the diagrams beside them.
func TestFirstRunProducesTheWholeDocument(t *testing.T) {
	root := copyFixture(t, "gateway")
	gen(t, root)

	entries, err := os.ReadDir(filepath.Join(root, docsDir))
	if err != nil {
		t.Fatal(err)
	}

	// 12 sections + index + two .mmd files.
	if len(entries) != 15 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("got %d files, want 15: %v", len(entries), names)
	}
}

// AC-7 at the level a user sees: two full runs must leave byte-identical output.
func TestGeneratedOutputIsStableAcrossRuns(t *testing.T) {
	root := copyFixture(t, "rules")
	gen(t, root)

	before := readAll(t, filepath.Join(root, docsDir))
	gen(t, root)
	after := readAll(t, filepath.Join(root, docsDir))

	for name, content := range before {
		if after[name] != content {
			t.Errorf("%s changed between two identical runs", name)
		}
	}
}

func readAll(t *testing.T, dir string) map[string]string {
	t.Helper()

	out := map[string]string{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		out[e.Name()] = string(b)
	}
	return out
}
