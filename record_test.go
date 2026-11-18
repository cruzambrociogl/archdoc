package archdoc_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/cruzambrociogl/archdoc"
)

// documented is the shop, generated, in a checkout with a commit and a remote on GitHub.
func documented(t *testing.T) string {
	t.Helper()
	root := shop(t)
	git := filepath.Join(root, ".git")
	if err := os.MkdirAll(git, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(git, "HEAD"), []byte("0123456789abcdef0123456789abcdef01234567\n"), 0o644)
	os.WriteFile(filepath.Join(git, "config"), []byte("[remote \"origin\"]\n\turl = git@github.com:acme/shop.git\n"), 0o644)
	if _, err := archdoc.Generate(context.Background(), root, archdoc.GenerateOptions{}); err != nil {
		t.Fatal(err)
	}
	return root
}

// A record carries the model, the coverage report, the documents and the commit and remote — and
// none of the code: not a line of the fixture's sources is in it.
func TestARecordCarriesWhatArchdocMadeAndNoneOfTheCode(t *testing.T) {
	root := documented(t)
	r, err := archdoc.MakeRecord(root)
	if err != nil {
		t.Fatal(err)
	}
	if r.Format != archdoc.RecordFormat || r.Engine == "" {
		t.Errorf("format %d, engine %q", r.Format, r.Engine)
	}
	if r.Remote != "https://github.com/acme/shop" || !strings.HasPrefix("0123456789abcdef0123456789abcdef01234567", r.Commit) || r.Commit == "" {
		t.Errorf("commit %q, remote %q", r.Commit, r.Remote)
	}
	if r.Model.Name != "shop" || len(r.Model.Nodes) == 0 {
		t.Errorf("model %q with %d nodes", r.Model.Name, len(r.Model.Nodes))
	}
	if _, ok := r.Documents["index.generated.md"]; !ok {
		t.Errorf("documents: %v", slices.Sorted(maps.Keys(r.Documents)))
	}
	for name := range r.Documents {
		if strings.HasSuffix(name, ".svg") || strings.HasSuffix(name, ".mmd") {
			t.Errorf("the record carries %s: diagrams are drawn from the model, not carried", name)
		}
	}

	size, err := r.Write(root)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, archdoc.RecordFile))
	if err != nil || len(b) != size {
		t.Fatalf("wrote %d bytes, read %d: %v", size, len(b), err)
	}
	if bytes.Contains(b, []byte(root)) {
		t.Error("the record names a path on this machine")
	}
	err = filepath.WalkDir(filepath.Join(root, "api"), func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		src, _ := os.ReadFile(p)
		for _, line := range strings.Split(string(src), "\n") {
			line = strings.TrimSpace(line)
			if len(line) < 20 { // short lines are names and braces, which a record may share
				continue
			}
			quoted, _ := json.Marshal(line)
			if bytes.Contains(b, quoted[1:len(quoted)-1]) {
				t.Errorf("the record carries a line of %s: %s", filepath.Base(p), line)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Making a record needs what generate wrote; without it, it says so.
func TestARecordNeedsGenerateFirst(t *testing.T) {
	if _, err := archdoc.MakeRecord(shop(t)); err == nil || !strings.Contains(err.Error(), "archdoc generate") {
		t.Errorf("a record of an undocumented repository: %v", err)
	}
}

// What one archdoc writes, another reads back the same.
func TestARecordReadsBackAsWritten(t *testing.T) {
	r, err := archdoc.MakeRecord(documented(t))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(r)
	back, err := archdoc.ReadRecord(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	again, _ := json.Marshal(back)
	if !bytes.Equal(b, again) {
		t.Error("a record read back differs from the one written")
	}
}

// A record may come from anyone, so ReadRecord refuses anything archdoc would not have written.
func TestReadRecordRefusesWhatArchdocWouldNotWrite(t *testing.T) {
	r, err := archdoc.MakeRecord(documented(t))
	if err != nil {
		t.Fatal(err)
	}
	good, _ := json.Marshal(r)
	edit := func(f func(m map[string]any)) []byte {
		var m map[string]any
		json.Unmarshal(good, &m)
		f(m)
		b, _ := json.Marshal(m)
		return b
	}
	cases := []struct {
		name string
		in   []byte
		want string
	}{
		{"not JSON", []byte("<html>"), "not a record"},
		{"newer", edit(func(m map[string]any) { m["format"] = archdoc.RecordFormat + 1 }), "newer archdoc"},
		{"older", edit(func(m map[string]any) { m["format"] = 0 }), "too old"},
		{"an unknown field", edit(func(m map[string]any) { m["source_files"] = []string{"main.go"} }), "unknown field"},
		{"a path for a document", edit(func(m map[string]any) {
			m["documents"].(map[string]any)["../../etc/passwd"] = "x"
		}), "not a document archdoc writes"},
		{"a document archdoc does not write", edit(func(m map[string]any) {
			m["documents"].(map[string]any)["01-introduction-and-goals.md"] = "the person's own section"
		}), "not a document archdoc writes"},
		{"no index", edit(func(m map[string]any) { delete(m["documents"].(map[string]any), "index.generated.md") }), "index.generated.md"},
		{"a commit that is not one", edit(func(m map[string]any) { m["commit"] = "main; rm -rf /" }), "not a commit"},
		{"a remote that is a script", edit(func(m map[string]any) { m["remote"] = "javascript:alert(1)" }), "not a repository's address"},
		{"a remote on another host", edit(func(m map[string]any) { m["remote"] = "https://evil.example/acme/shop" }), "not a repository's address"},
		{"coverage of another shape", edit(func(m map[string]any) { m["coverage"] = map[string]any{"files": 1} }), "coverage"},
		{"rules that are not an answer", edit(func(m map[string]any) { m["rules"] = "yes" }), "rules"},
		{"a model that fails validation", edit(func(m map[string]any) {
			model := m["model"].(map[string]any)
			model["edges"] = append(model["edges"].([]any), map[string]any{"from": "svc:nowhere", "to": "svc:nothing"})
		}), "failed validation"},
	}
	for _, c := range cases {
		_, err := archdoc.ReadRecord(bytes.NewReader(c.in))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v — want an error saying %q", c.name, err, c.want)
		}
	}
	if _, err := archdoc.ReadRecord(bytes.NewReader(edit(func(m map[string]any) { m["format"] = archdoc.RecordFormat + 1 }))); !errors.Is(err, archdoc.ErrRecordTooNew) {
		t.Errorf("a newer record: %v, want ErrRecordTooNew", err)
	}
}

// Every string a record's model carries is bounded; what is under the bound is left as it is.
func TestExcerptsAreBounded(t *testing.T) {
	long := strings.Repeat("a docstring that goes on ", 40)
	m := archdoc.Model{
		Name:        "shop",
		Description: long,
		Nodes:       []archdoc.Node{{ID: "svc:api", Name: "api"}},
	}
	if n := archdoc.BoundExcerpts(&m); n != 1 {
		t.Errorf("cut %d strings, want 1", n)
	}
	if got := utf8.RuneCountInString(m.Description); got != archdoc.MaxExcerpt || !strings.HasSuffix(m.Description, "…") {
		t.Errorf("bounded to %d characters: %q", got, m.Description)
	}
	if m.Name != "shop" || m.Nodes[0].ID != "svc:api" {
		t.Error("a short string was changed")
	}
	if n := archdoc.BoundExcerpts(&m); n != 0 {
		t.Errorf("bounding twice cut %d more", n)
	}
}

// A site built from the record alone is the site built from the repository: the same files, and
// the same content but for when each was built.
func TestASiteFromTheRecordIsTheRepositorysSite(t *testing.T) {
	root := documented(t)
	want, err := archdoc.BuildSite(context.Background(), root, 0)
	if errors.Is(err, archdoc.ErrNoWebApp) {
		t.Skip("built without the web app")
	}
	if err != nil {
		t.Fatal(err)
	}
	r, err := archdoc.MakeRecord(root)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(r)
	read, err := archdoc.ReadRecord(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	got, err := archdoc.BuildSiteFromRecord(context.Background(), read)
	if err != nil {
		t.Fatal(err)
	}
	if w, g := slices.Sorted(maps.Keys(want.Files)), slices.Sorted(maps.Keys(got.Files)); !slices.Equal(w, g) {
		t.Fatalf("the sites differ in their files:\nfrom the repository %v\nfrom the record     %v", w, g)
	}
	// When each was built: the version's time, and what is measured against it.
	timed := map[string]bool{"data/session.json": true, "data/versions.json": true, "data/completeness.json": true}
	for name, w := range want.Files {
		if timed[name] {
			continue
		}
		if !bytes.Equal(w, got.Files[name]) {
			t.Errorf("%s differs between the two sites:\nfrom the repository %.400s\nfrom the record     %.400s", name, w, got.Files[name])
		}
	}
	var sess struct {
		Remote, Commit string
	}
	json.Unmarshal(got.Files["data/session.json"], &sess)
	if sess.Remote != "https://github.com/acme/shop" || sess.Commit != r.Commit {
		t.Errorf("the record's site cites %q at %q", sess.Remote, sess.Commit)
	}
}

// The model's JSON is what a record's format promises. When its shape changes — a field added,
// renamed, removed, or its type changed — this fails, to be answered in record.go: a change that
// only adds fields raises RecordFormat; one that renames or removes raises MinRecordFormat too.
// Then the pin below is updated.
func TestModelShapeIsPinned(t *testing.T) {
	const pinned = "fdf4f89a49a5b7da"
	sum := sha256.Sum256([]byte(shape(reflect.TypeOf(archdoc.Model{}), map[reflect.Type]bool{})))
	if got := hex.EncodeToString(sum[:8]); got != pinned {
		t.Errorf("the model's shape changed (%s, pinned %s): raise RecordFormat in record.go — and MinRecordFormat if a field was renamed or removed — then pin %s", got, pinned, got)
	}
}

// shape writes a type as its JSON sees it: each field's name and the shape of its value.
func shape(t reflect.Type, seen map[reflect.Type]bool) string {
	switch t.Kind() {
	case reflect.Pointer:
		return "*" + shape(t.Elem(), seen)
	case reflect.Slice, reflect.Array:
		return "[]" + shape(t.Elem(), seen)
	case reflect.Map:
		return "map[" + shape(t.Key(), seen) + "]" + shape(t.Elem(), seen)
	case reflect.Struct:
		if seen[t] {
			return t.Name()
		}
		seen[t] = true
		var b strings.Builder
		b.WriteString(t.Name() + "{")
		for i := range t.NumField() {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			tag := f.Tag.Get("json")
			if tag == "-" {
				continue
			}
			fmt.Fprintf(&b, "%s:%s;", tag, shape(f.Type, seen))
		}
		return b.String() + "}"
	}
	return t.Kind().String()
}
