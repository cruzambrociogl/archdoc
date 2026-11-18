package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// gitFixture is a fixture repository copied somewhere writable, with a .git so it is a repository.
func gitFixture(t *testing.T, name, dst string) string {
	t.Helper()
	src := copyFixture(t, name)
	if dst == "" {
		dst = src
	} else if err := os.Rename(src, dst); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dst, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dst
}

func exists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	return err == nil
}

// A bare archdoc, from anywhere in a repository, documents the repository and writes only
// .archdoc/: a first look leaves no documents in the working tree.
func TestBareArchdocWritesOnlyItsOwnFolder(t *testing.T) {
	root := gitFixture(t, "endpoints", "")
	sub := filepath.Join(root, "deep", "er")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run([]string{sub}, &out); err != nil {
		t.Fatalf("archdoc %s: %v\n%s", sub, err, out.String())
	}
	if !exists(t, filepath.Join(root, ".archdoc", "model.json")) || !exists(t, filepath.Join(root, ".archdoc", "history.db")) {
		t.Errorf("no model or history in the repository's root:\n%s", out.String())
	}
	if exists(t, filepath.Join(root, "docs")) {
		t.Errorf("a bare archdoc wrote documents into the repository:\n%s", out.String())
	}
	if exists(t, filepath.Join(sub, ".archdoc")) {
		t.Error("documented the subfolder it was run in, not the repository")
	}
	if !strings.Contains(out.String(), "no documents in docs/architecture") {
		t.Errorf("did not say where the documents come from:\n%s", out.String())
	}
}

// Documents someone asked for once stay current: a bare archdoc writes them where they are.
func TestBareArchdocKeepsDocumentsThatAreThere(t *testing.T) {
	root := gitFixture(t, "endpoints", "")
	if err := run([]string{"generate", root}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	coverage := filepath.Join(root, "docs", "architecture", "coverage.generated.md")
	if err := os.Remove(coverage); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run([]string{root}, &out); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	if !exists(t, coverage) {
		t.Errorf("the documents that were there were not kept current:\n%s", out.String())
	}
}

// In a folder of repositories, each is documented — one with nothing in it is passed over.
func TestBareArchdocInAFolderOfRepositories(t *testing.T) {
	dir := t.TempDir()
	a := gitFixture(t, "endpoints", filepath.Join(dir, "a"))
	b := gitFixture(t, "gateway", filepath.Join(dir, "b"))
	empty := filepath.Join(dir, "empty")
	if err := os.MkdirAll(filepath.Join(empty, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run([]string{dir}, &out); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	for _, repo := range []string{a, b} {
		if !exists(t, filepath.Join(repo, ".archdoc", "model.json")) {
			t.Errorf("%s was not documented:\n%s", repo, out.String())
		}
		if exists(t, filepath.Join(repo, "docs")) {
			t.Errorf("%s got documents it did not ask for", repo)
		}
	}
	if !strings.Contains(out.String(), "passed over") {
		t.Errorf("the empty repository was not said to be passed over:\n%s", out.String())
	}
}
