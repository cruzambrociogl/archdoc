package render

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const docs = "docs/architecture"

// writeStubs creates every human section as archdoc would, and records it.
func writeStubs(t *testing.T, root string) {
	t.Helper()
	for name, content := range Stubs(fixture(), meta()) {
		path := filepath.Join(root, docs, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := RecordStub(root, docs, name); err != nil {
			t.Fatal(err)
		}
	}
}

func stateOf(t *testing.T, root, file string, changed time.Time) Completeness {
	t.Helper()
	states, err := SectionStates(root, docs, changed)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range states {
		if s.File == file {
			return s.State
		}
	}
	t.Fatalf("%s not reported", file)
	return ""
}

func TestUntouchedStubIsNotStarted(t *testing.T) {
	root := t.TempDir()
	writeStubs(t, root)

	if got := stateOf(t, root, "01-introduction-and-goals.md", time.Time{}); got != NotStarted {
		t.Errorf("an untouched stub reads %q", got)
	}
}

// A person writing in the section is detected without archdoc ever opening the file.
func TestEditedSectionIsWritten(t *testing.T) {
	root := t.TempDir()
	writeStubs(t, root)

	path := filepath.Join(root, docs, "01-introduction-and-goals.md")
	if err := os.WriteFile(path, []byte("# 1. Introduction\n\nMine now.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := stateOf(t, root, "01-introduction-and-goals.md", time.Time{}); got != Written {
		t.Errorf("an edited section reads %q", got)
	}
}

// Written before the architecture last changed: it may describe a system that no longer exists.
func TestSectionWrittenBeforeAChangeMayBeStale(t *testing.T) {
	root := t.TempDir()
	writeStubs(t, root)

	path := filepath.Join(root, docs, "04-solution-strategy.md")
	if err := os.WriteFile(path, []byte("We chose microservices.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(path, past, past); err != nil {
		t.Fatal(err)
	}

	if got := stateOf(t, root, "04-solution-strategy.md", time.Now().Add(-time.Hour)); got != MayBeStale {
		t.Errorf("a section written before the architecture changed reads %q", got)
	}
}

func TestDeletedSectionIsMissing(t *testing.T) {
	root := t.TempDir()
	writeStubs(t, root)
	if err := os.Remove(filepath.Join(root, docs, "09-architecture-decisions.md")); err != nil {
		t.Fatal(err)
	}

	if got := stateOf(t, root, "09-architecture-decisions.md", time.Time{}); got != Missing {
		t.Errorf("a deleted section reads %q", got)
	}
}

// Every human section is reported, and only human sections.
func TestAllSevenHumanSectionsAreReported(t *testing.T) {
	root := t.TempDir()
	writeStubs(t, root)

	states, err := SectionStates(root, docs, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(states) != 7 {
		t.Errorf("got %d sections, want 7", len(states))
	}
	for _, s := range states {
		if s.Section.Owner != Human {
			t.Errorf("a generated section was reported: %s", s.File)
		}
	}
}
