package extract

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

const fixture = "../../testdata/repo"

func TestDiscoverClassifiesFragments(t *testing.T) {
	got, err := Discover(fixture)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	want := map[string]string{
		"docker/docker-compose.yml":     "deployable",
		"docker/docker-compose.dev.yml": "deployable",
		"e2e/docker-compose.yml":        "deployable",
		"docker/hwaccel.yml":            "fragment",
	}

	if len(got) != len(want) {
		t.Errorf("found %d candidates, want %d", len(got), len(want))
		for _, c := range got {
			t.Logf("  %s — %s", c.File, c.Reason)
		}
	}

	for _, c := range got {
		expect, ok := want[c.File]
		if !ok {
			t.Errorf("unexpected candidate %s", c.File)
			continue
		}

		// A chosen file's reason is prefixed with "selected — ".
		reason := c.Reason
		if c.Chosen {
			reason = reason[len("selected — "):]
		}

		if len(reason) < len(expect) || reason[:len(expect)] != expect {
			t.Errorf("%s classified %q, want %q", c.File, reason, expect)
		}
	}
}

// A fragment must never be selected. docker/hwaccel.yml parses cleanly and has a top-level
// services: key, which is exactly why content-sniffing alone is not enough.
func TestDiscoverSelectsCanonical(t *testing.T) {
	got, err := Discover(fixture)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	chosen := ""
	count := 0
	for _, c := range got {
		if c.Chosen {
			chosen = c.File
			count++
		}
	}

	if count != 1 {
		t.Fatalf("%d files chosen, want exactly 1", count)
	}

	if want := filepath.Join("docker", "docker-compose.yml"); chosen != want {
		t.Errorf("chose %s, want %s — the unsuffixed name at the shallower path", chosen, want)
	}
}

func TestScanProvenance(t *testing.T) {
	fs, err := Scan(fixture)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(fs.Services) != 2 {
		t.Fatalf("found %d services, want 2", len(fs.Services))
	}

	// Sorted by name, so cache precedes web.
	want := []struct {
		name  string
		image string
		line  int
	}{
		{"cache", "redis:7", 11},
		{"web", "nginx:1.27", 4},
	}

	for i, w := range want {
		got := fs.Services[i]

		if got.Name != w.name {
			t.Errorf("service %d is %q, want %q", i, got.Name, w.name)
		}
		if got.Image != w.image {
			t.Errorf("%s image is %q, want %q", w.name, got.Image, w.image)
		}
		if got.Prov.Line != w.line {
			t.Errorf("%s declared at line %d, want %d", w.name, got.Prov.Line, w.line)
		}

		// P1: a fact that cannot be traced to its source must not be emitted.
		if !got.Prov.Known() {
			t.Errorf("%s has unknown provenance %+v", w.name, got.Prov)
		}
	}
}

// AC-7 requires five consecutive scans to produce byte-identical output. Go randomises map
// iteration, so this fails without explicit sorting — which is the single most likely defect
// in this codebase.
func TestScanIsDeterministic(t *testing.T) {
	var first string

	for i := range 5 {
		fs, err := Scan(fixture)
		if err != nil {
			t.Fatalf("run %d: %v", i, err)
		}

		b, err := json.Marshal(fs)
		if err != nil {
			t.Fatalf("run %d: marshal: %v", i, err)
		}

		if i == 0 {
			first = string(b)
			continue
		}

		if string(b) != first {
			t.Fatalf("run %d differs from run 0 — output is not deterministic", i)
		}
	}
}

// Compose's !reset and !override are custom YAML tags that a stock decoder rejects. The
// position pass strips them; this proves the fixture using one still classifies.
func TestCustomTagsDoNotBreakDiscovery(t *testing.T) {
	got, err := Discover(fixture)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	for _, c := range got {
		if c.File == filepath.Join("docker", "docker-compose.dev.yml") {
			return // found and classified; the tag did not abort the parse
		}
	}

	t.Error("docker/docker-compose.dev.yml was not discovered — the !reset tag broke the parse")
}
