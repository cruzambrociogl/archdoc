package serve

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/core"
	"github.com/cruzambrociogl/archdoc/internal/render"
	"github.com/cruzambrociogl/archdoc/internal/store"
)

// The published site (surface-spec §3): the same app, built as static files, with one version's
// data baked in beside it. Its data is produced by asking this server's own handlers for every
// response the app reads, so the published files have exactly the live API's shapes — there is no
// second implementation to drift.

// StaticName is the file a published site reads an API path from. web/app/src/staticName.ts mirrors it.
//
//	/api/docs/x.md             → data/docs/x.md
//	/api/svg?view=container    → data/svg@view=container.svg
//	/api/scene?view=c&version=3 → data/scene@version=3,view=c.json   (query keys sorted)
var fileSafe = strings.NewReplacer("/", "_", ":", "~")

func StaticName(apiPath string) string {
	p, rawQuery, _ := strings.Cut(strings.TrimPrefix(apiPath, "/api/"), "?")
	if strings.HasPrefix(p, "docs/") {
		return "data/" + p
	}
	ext := ".json"
	if p == "svg" {
		ext = ".svg"
	}
	name := strings.ReplaceAll(p, "/", "_")
	if q, err := url.ParseQuery(rawQuery); err == nil && len(q) > 0 {
		keys := make([]string, 0, len(q))
		for k := range q {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			// A component view's name holds an element ID — "component:app:packages/cli" — and a
			// file name may hold neither a slash nor, on every system, a colon.
			parts = append(parts, k+"="+fileSafe.Replace(q.Get(k)))
		}
		name += "@" + strings.Join(parts, ",")
	}
	return "data/" + name + ext
}

// Export returns the published site's data, keyed by StaticName: the latest version, and the
// baseline "what changed" is measured against — the version given, or the one before the latest.
// The network-run log travels as a summary only: the payloads stay on the publisher's machine.
//
// With no history — a fresh clone in CI, where history.db is never committed — the site is built
// from the committed record instead: .archdoc/model.json, laid out by the same engine, as a single
// version. Nothing is regenerated, so model-written labels in model.json survive and no key is
// needed (surface-spec §3.2).
// svgRef finds a diagram a document names: a file name ending .svg, with no folder.
var svgRef = regexp.MustCompile(`([A-Za-z0-9_.-]+\.svg)\b`)

func Export(root string, since int64) (map[string][]byte, error) {
	return ExportFrom(root, since, nil)
}

// Fixed is what a site built from a record carries in place of what only the repository could
// say: the commit and the remote its citations open on, and the two answers that read the
// repository itself — what rules.yaml did, and which of the person's sections exist. The record
// was made where the repository is (Captured); the site is built where it is not.
type Fixed struct {
	Commit, Remote      string
	Rules, Completeness []byte
}

// ExportFrom is Export, for a root that may hold only the record's files: with fixed, what needs
// the repository comes from it instead, so nothing reads code that is not there.
func ExportFrom(root string, since int64, fixed *Fixed) (map[string][]byte, error) {
	s, cleanup, err := exportServer(root, fixed)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	latest, err := s.latest()
	if err != nil {
		return nil, err
	}
	h, err := s.open()
	if err != nil {
		return nil, err
	}
	all, err := h.Versions(1000)
	h.Close()
	if err != nil {
		return nil, err
	}
	var base int64
	for _, v := range all {
		if since != 0 && v.ID == since {
			base = v.ID
		}
		if since == 0 && v.ID < latest.ID && v.ID > base {
			base = v.ID
		}
	}
	if since != 0 && base == 0 {
		return nil, fmt.Errorf("no version %d to compare with", since)
	}

	out := map[string][]byte{}
	get := s.fetch
	put := func(p string, optional bool) error {
		b, err := get(p, optional)
		if err == nil && b != nil {
			out[StaticName(p)] = b
		}
		return err
	}

	paths := []string{"/api/summary", "/api/model", "/api/rules", "/api/docs", "/api/views", "/api/runs"}
	views := render.Views(latest.Model)
	for _, view := range views {
		q := url.QueryEscape(view.Name)
		paths = append(paths, "/api/scene?view="+q, "/api/svg?view="+q,
			// What Download offers: the figure in archdoc's look too, and the view as text.
			"/api/svg?view="+q+"&look=archdoc", "/api/export?view="+q+"&format=mermaid")
		if render.HasText(view) {
			paths = append(paths, "/api/export?view="+q+"&format=structurizr", "/api/export?view="+q+"&format=plantuml")
		}
	}
	// What breaks without each part of Apps and services, for One part.
	for _, n := range latest.Model.Container().Nodes {
		if render.HasBreaks(latest.Model, n.ID) {
			paths = append(paths, "/api/breaks?part="+url.QueryEscape(n.ID))
		}
	}
	versions := []int64{latest.ID}
	var optional []string // the baseline may not have every view the latest has
	if base != 0 {
		versions = append(versions, base)
		paths = append(paths, fmt.Sprintf("/api/diff?from=%d&to=%d", base, latest.ID))
		for _, view := range views {
			optional = append(optional, fmt.Sprintf("/api/scene?view=%s&version=%d", url.QueryEscape(view.Name), base))
		}
	}
	for _, v := range versions {
		paths = append(paths, fmt.Sprintf("/api/model?version=%d", v))
	}
	for _, sec := range render.Sections() {
		if sec.Owner == render.Human {
			paths = append(paths, "/api/questions?section="+strconv.Itoa(sec.Number))
		}
	}
	for _, p := range paths {
		if err := put(p, false); err != nil {
			return nil, err
		}
	}
	for _, p := range optional {
		if err := put(p, true); err != nil {
			return nil, err
		}
	}
	if err := put("/api/coverage", true); err != nil {
		return nil, err
	}
	// The explanation, when there is one of the latest version's architecture.
	if err := put("/api/interpretation", true); err != nil {
		return nil, err
	}
	// Completeness by size only: a published site is built from a fresh checkout, where every
	// modification time is the checkout's, so staleness would mean nothing there.
	comp, err := get("/api/completeness?basis=size", false)
	if err != nil {
		return nil, err
	}
	out[StaticName("/api/completeness")] = comp

	// The documents, as the reader opens them.
	var docs struct {
		Generated []string `json:"generated"`
	}
	json.Unmarshal(out[StaticName("/api/docs")], &docs)
	for _, f := range docs.Generated {
		if err := put("/api/docs/"+url.PathEscape(f), false); err != nil {
			return nil, err
		}
	}
	// And the diagrams they show: each SVG a document names, from archdoc's own folder. One named
	// and not generated is left out, as the live app would answer it.
	named := map[string]bool{}
	for _, f := range docs.Generated {
		for _, m := range svgRef.FindAllSubmatch(out[StaticName("/api/docs/"+url.PathEscape(f))], -1) {
			named[string(m[1])] = true
		}
	}
	for _, name := range slices.Sorted(maps.Keys(named)) {
		if err := put("/api/docs/"+url.PathEscape(name), true); err != nil {
			return nil, err
		}
	}

	// Published versions: the two the site shows, and nothing between.
	var vs []map[string]any
	b, err := get("/api/versions", false)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(b, &vs)
	kept := []map[string]any{}
	for _, v := range vs {
		id, _ := v["id"].(float64)
		if int64(id) == latest.ID || int64(id) == base {
			kept = append(kept, v)
		}
	}
	out[StaticName("/api/versions")], _ = json.MarshalIndent(kept, "", "  ")

	// The summary and the session say this is a published site, from which commit, and where its
	// citations open.
	var sum map[string]any
	json.Unmarshal(out[StaticName("/api/summary")], &sum)
	sum["versions"] = len(kept)
	sum["root"] = ""
	out[StaticName("/api/summary")], _ = json.MarshalIndent(sum, "", "  ")
	out[StaticName("/api/session")], _ = json.MarshalIndent(map[string]any{
		"mode": "published", "remote": s.remote(), "commit": latest.Commit,
		"generated_at": latest.CreatedAt, "version": latest.ID, "baseline": base,
	}, "", "  ")
	return out, nil
}

// exportServer answers for root as a published site: from its history, or — with none — from a
// history seeded from its model.json. cleanup removes the seed.
func exportServer(root string, fixed *Fixed) (*Server, func(), error) {
	if fixed == nil {
		if s, err := New(root, nil); err == nil { // a published site never updates
			return s, func() {}, nil
		} else if _, serr := os.Stat(filepath.Join(root, ".archdoc", "model.json")); serr != nil {
			return nil, nil, fmt.Errorf("%v; and no committed .archdoc/model.json to build from instead: %v", err, serr)
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, nil, err
	}
	commit := render.Commit(abs)
	if fixed != nil {
		commit = fixed.Commit
	}
	seed, cleanup, err := seedHistory(abs, commit)
	if err != nil {
		return nil, nil, fmt.Errorf("no history, and no .archdoc/model.json to build from instead: %v", err)
	}
	s := newServer(abs, seed, nil)
	s.fixed = fixed
	return s, cleanup, nil
}

// fetch asks the server for p as the app would, and returns the answer with nothing about this
// machine in it: absolute paths become repository-relative. An optional path that is not found
// is nil, nil.
func (s *Server) fetch(p string, optional bool) ([]byte, error) {
	req := httptest.NewRequest("GET", p, nil)
	req.Host = "localhost"
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code == http.StatusNotFound && optional {
		return nil, nil
	}
	if rec.Code != http.StatusOK {
		return nil, fmt.Errorf("%s: %d %s", p, rec.Code, strings.TrimSpace(rec.Body.String()))
	}
	b := bytes.ReplaceAll(rec.Body.Bytes(), []byte(s.root+"/"), nil)
	return bytes.ReplaceAll(b, []byte(s.root), []byte(".")), nil
}

// remote is where the site's citations open: the repository's origin, or the record's.
func (s *Server) remote() string {
	if s.fixed != nil {
		return s.fixed.Remote
	}
	return render.Remote(s.root)
}

// Captured is what a record needs from the repository beyond its files: the commit and remote,
// and the answers of the two endpoints that read the repository itself, exactly as a published
// site built here would carry them.
func Captured(root string) (*Fixed, error) {
	s, cleanup, err := exportServer(root, nil)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	rules, err := s.fetch("/api/rules", false)
	if err != nil {
		return nil, err
	}
	// By size only, as Export asks: the site is read far from this checkout's modification times.
	comp, err := s.fetch("/api/completeness?basis=size", false)
	if err != nil {
		return nil, err
	}
	return &Fixed{Commit: render.Commit(s.root), Remote: render.Remote(s.root), Rules: rules, Completeness: comp}, nil
}

// seedHistory builds a one-version history from the committed model.json, in a temporary
// directory, so a clone with no history.db can still be published. The caller removes it.
func seedHistory(root, commit string) (string, func(), error) {
	b, err := os.ReadFile(filepath.Join(root, ".archdoc", "model.json"))
	if err != nil {
		return "", nil, err
	}
	var m core.Model
	if err := json.Unmarshal(b, &m); err != nil {
		return "", nil, fmt.Errorf(".archdoc/model.json: %w", err)
	}
	dir, err := os.MkdirTemp("", "archdoc-export-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { os.RemoveAll(dir) }

	layouts := map[string]core.Layout{}
	for _, view := range render.Views(m) {
		l, err := render.Layout(context.Background(), view.Model, view.Group)
		if err != nil {
			cleanup()
			return "", nil, err
		}
		layouts[view.Name] = l
	}
	h, err := store.Open(dir)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	defer h.Close()
	if _, _, err := h.Save(m, layouts, commit, "export"); err != nil {
		cleanup()
		return "", nil, err
	}
	return dir, cleanup, nil
}
