package archdoc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"unicode/utf8"

	"github.com/cruzambrociogl/archdoc/internal/arrange"
	"github.com/cruzambrociogl/archdoc/internal/core"
	"github.com/cruzambrociogl/archdoc/internal/render"
	"github.com/cruzambrociogl/archdoc/internal/serve"
	"github.com/cruzambrociogl/archdoc/internal/validate"
	"github.com/cruzambrociogl/archdoc/web"
)

// A record is what archdoc made of a repository, in one file, with none of the repository in it:
// the model, the coverage report, the documents archdoc wrote, and the few answers that needed the
// repository to give. It is made where the code is — a person's machine, or their CI — and a site
// is built from it anywhere else, by any archdoc that reads its format, with no checkout.
//
// What it carries of the code itself is names, paths and line numbers, and short excerpts the
// model cites as written — a docstring's first line, an option's value, a manifest's description —
// each bounded to MaxExcerpt characters. No file's contents, and no function's body.

// RecordFormat is the shape of a record this archdoc writes. It changes whenever the model's JSON
// does (TestModelShapeIsPinned); MinRecordFormat is the oldest this archdoc still reads. A change
// that only adds fields keeps the minimum; one that renames or removes raises it.
const (
	RecordFormat    = 1
	MinRecordFormat = 1
)

// MaxExcerpt bounds every string a record's model carries, in characters. The longest archdoc
// writes itself are its notes, under 250; what is longer came from the code, and is cut.
const MaxExcerpt = 300

// RecordFile is where 'archdoc export --record' writes the record: inside archdoc's own
// directory, and ignored by git, since CI makes it afresh.
const RecordFile = stateDir + "/record.json"

// Record is one record. Its JSON is the file.
type Record struct {
	Format int `json:"format"`
	// Engine is the archdoc that made it.
	Engine string `json:"engine"`
	// Commit is the commit it was made at; Remote the repository's web address, where its
	// citations open — empty for a host archdoc does not know.
	Commit string `json:"commit"`
	Remote string `json:"remote,omitempty"`
	Model  Model  `json:"model"`
	// Coverage is .archdoc/coverage.json: what was read, what could not be resolved.
	Coverage json.RawMessage `json:"coverage"`
	// Documents are the documents archdoc wrote that the app shows, by file name. Diagrams are
	// not carried: a site draws them from the model, as generate does.
	Documents map[string]string `json:"documents"`
	// Rules and Completeness are the app's answers that read the repository itself: what
	// rules.yaml did, and which of the person's own sections exist.
	Rules        json.RawMessage `json:"rules"`
	Completeness json.RawMessage `json:"completeness"`
	// Layout and Views are layout.yaml and views.yaml: the diagrams as people arranged them.
	Layout string `json:"layout,omitempty"`
	Views  string `json:"views,omitempty"`
}

// Errors a record is refused with, besides a malformed one.
var (
	ErrRecordTooNew = errors.New("this record was made by a newer archdoc than this one reads — update archdoc")
	ErrRecordTooOld = errors.New("this record was made by an archdoc too old for this one to read — update archdoc where the record is made")
)

// MakeRecord makes the record of the repository at root from what 'archdoc generate' wrote there:
// it reads archdoc's own files and nothing of the code, but for the two answers that need it.
func MakeRecord(root string) (*Record, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(filepath.Join(abs, filepath.FromSlash(modelOut)))
	if err != nil {
		return nil, fmt.Errorf("no %s — run 'archdoc generate %s' first", modelOut, root)
	}
	var m Model
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("%s: %w", modelOut, err)
	}
	coverage, err := os.ReadFile(filepath.Join(abs, filepath.FromSlash(coverageOut)))
	if err != nil {
		return nil, fmt.Errorf("no %s — run 'archdoc generate %s' with this archdoc first", coverageOut, root)
	}
	docs := map[string]string{}
	for _, name := range recordDocuments() {
		b, err := os.ReadFile(filepath.Join(abs, docsDir, name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		docs[name] = string(b)
	}
	if _, ok := docs[render.IndexFile]; !ok {
		return nil, fmt.Errorf("no documents in %s — a record carries them; run 'archdoc generate %s' first", docsDir, root)
	}
	fixed, err := serve.Captured(abs)
	if err != nil {
		return nil, err
	}
	rec := &Record{
		Format: RecordFormat, Engine: core.Build().String(),
		Commit: fixed.Commit, Remote: fixed.Remote,
		Model: m, Coverage: bytes.TrimSpace(coverage), Documents: docs,
		Rules: bytes.TrimSpace(fixed.Rules), Completeness: bytes.TrimSpace(fixed.Completeness),
	}
	rec.Layout, err = optionalFile(filepath.Join(abs, arrange.Dir, arrange.LayoutFile))
	if err != nil {
		return nil, err
	}
	rec.Views, err = optionalFile(filepath.Join(abs, arrange.Dir, arrange.ViewsFile))
	if err != nil {
		return nil, err
	}
	BoundExcerpts(&rec.Model)
	return rec, nil
}

// Write puts the record at root's RecordFile, and returns its size in bytes.
func (r *Record) Write(root string) (int, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return 0, err
	}
	return len(b) + 1, write(root, RecordFile, string(b)+"\n")
}

// ReadRecord reads a record that may come from anyone: its format first, then strictly — no field
// it does not know, every document one archdoc writes, the commit and remote in the forms archdoc
// produces — its excerpts bounded, and the model validated as generate validates it. The caller
// limits how much it reads.
func ReadRecord(in io.Reader) (*Record, error) {
	b, err := io.ReadAll(in)
	if err != nil {
		return nil, err
	}
	var head struct {
		Format int `json:"format"`
	}
	if err := json.Unmarshal(b, &head); err != nil {
		return nil, fmt.Errorf("not a record: %w", err)
	}
	switch {
	case head.Format > RecordFormat:
		return nil, ErrRecordTooNew
	case head.Format < MinRecordFormat:
		return nil, ErrRecordTooOld
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var r Record
	if err := dec.Decode(&r); err != nil {
		return nil, fmt.Errorf("not a record: %w", err)
	}
	if dec.More() {
		return nil, errors.New("not a record: more after it")
	}
	if !commitShape.MatchString(r.Commit) {
		return nil, fmt.Errorf("the record's commit is not a commit: %q", r.Commit)
	}
	if r.Remote != "" && (render.WebAddress(r.Remote) != r.Remote || !remoteShape.MatchString(r.Remote)) {
		return nil, fmt.Errorf("the record's remote is not a repository's address: %q", r.Remote)
	}
	allowed := map[string]bool{}
	for _, name := range recordDocuments() {
		allowed[name] = true
	}
	for name := range r.Documents {
		if !allowed[name] {
			return nil, fmt.Errorf("the record carries %q, which is not a document archdoc writes", name)
		}
	}
	if _, ok := r.Documents[render.IndexFile]; !ok {
		return nil, fmt.Errorf("the record carries no %s", render.IndexFile)
	}
	var cov render.CoverageReport
	strict := json.NewDecoder(bytes.NewReader(r.Coverage))
	strict.DisallowUnknownFields()
	if err := strict.Decode(&cov); err != nil {
		return nil, fmt.Errorf("the record's coverage: %w", err)
	}
	for what, raw := range map[string]json.RawMessage{"rules": r.Rules, "completeness": r.Completeness} {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(raw, &obj); err != nil {
			return nil, fmt.Errorf("the record's %s: %w", what, err)
		}
	}
	BoundExcerpts(&r.Model)
	if res := validate.Model(r.Model); !res.OK() {
		return nil, fmt.Errorf("the record's model failed validation\n%s", res.Error())
	}
	return &r, nil
}

// BuildSiteFromRecord builds the published site from a record alone: the record's files in a
// directory of their own, the diagrams drawn from its model as generate draws them, and the site
// exported from there as BuildSite exports a repository — the same server answering, so there is
// no second implementation to drift. The record should come through ReadRecord or MakeRecord.
func BuildSiteFromRecord(ctx context.Context, r *Record) (*Site, error) {
	assets, built := web.Assets()
	if !built {
		return nil, ErrNoWebApp
	}
	dir, err := os.MkdirTemp("", "archdoc-record-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	files := map[string]string{
		modelOut:    encode(r.Model),
		coverageOut: string(indented(r.Coverage)),
	}
	for name, content := range r.Documents {
		if name != path.Base(name) { // ReadRecord refused it already; a record made here never has one
			return nil, fmt.Errorf("a document name, not a path: %q", name)
		}
		files[docsDir+"/"+name] = content
	}
	if r.Layout != "" {
		files[arrange.Dir+"/"+arrange.LayoutFile] = r.Layout
	}
	if r.Views != "" {
		files[arrange.Dir+"/"+arrange.ViewsFile] = r.Views
	}
	for _, rel := range sortedKeys(files) {
		if err := write(dir, rel, files[rel]); err != nil {
			return nil, err
		}
	}
	// After layout.yaml is in place: the figures show the diagrams as people arranged them.
	for name, content := range pictures(dir, r.Model, computeLayouts(r.Model, io.Discard), io.Discard) {
		if err := write(dir, docsDir+"/"+name, content); err != nil {
			return nil, err
		}
	}

	data, err := serve.ExportFrom(dir, 0, &serve.Fixed{
		Commit: r.Commit, Remote: r.Remote, Rules: indented(r.Rules), Completeness: indented(r.Completeness),
	})
	if err != nil {
		return nil, err
	}
	return assemble(assets, data)
}

// BoundExcerpts cuts every string in m longer than MaxExcerpt characters to that length, its last
// character an ellipsis, and returns how many it cut. Names, paths and archdoc's own notes fall
// well under the bound; what reaches it is text the code carried.
func BoundExcerpts(m *Model) int {
	return bound(reflect.ValueOf(m).Elem())
}

func bound(v reflect.Value) int {
	cut := 0
	switch v.Kind() {
	case reflect.String:
		if s := v.String(); utf8.RuneCountInString(s) > MaxExcerpt && v.CanSet() {
			v.SetString(string([]rune(s)[:MaxExcerpt-1]) + "…")
			cut++
		}
	case reflect.Struct:
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				cut += bound(v.Field(i))
			}
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			cut += bound(v.Index(i))
		}
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			cut += bound(v.Elem())
		}
	case reflect.Map:
		// Map values cannot be set in place: each is copied, bounded and put back.
		iter := v.MapRange()
		for iter.Next() {
			val := reflect.New(iter.Value().Type()).Elem()
			val.Set(iter.Value())
			if n := bound(val); n > 0 {
				v.SetMapIndex(iter.Key(), val)
				cut += n
			}
		}
	}
	return cut
}

// recordDocuments are the documents a record carries: the ones the app lists and shows.
func recordDocuments() []string {
	names := []string{render.IndexFile, render.CoverageFile}
	for _, sec := range render.Sections() {
		if sec.Owner == render.Generated {
			names = append(names, sec.File())
		}
	}
	return names
}

// indented is JSON as archdoc writes and serves it, whatever spacing it travelled in.
func indented(raw json.RawMessage) []byte {
	var b bytes.Buffer
	if err := json.Indent(&b, raw, "", "  "); err != nil {
		return raw
	}
	b.WriteByte('\n')
	return b.Bytes()
}

func optionalFile(p string) (string, error) {
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	return string(b), err
}

var (
	commitShape = regexp.MustCompile(`^[0-9a-f]{0,40}$`)
	remoteShape = regexp.MustCompile(`^https://[A-Za-z0-9.-]+/[A-Za-z0-9._~/-]+$`)
)
