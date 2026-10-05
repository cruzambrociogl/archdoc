package serve

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/render"
)

// The published site (surface-spec §3): the same app, built as static files, with one version's
// data baked in beside it. Its data is produced by asking this server's own handlers for every
// response the app reads, so the published files have exactly the live API's shapes — there is no
// second implementation to drift.

// StaticName is the file a published site reads an API path from. web/src/api.ts mirrors it.
//
//	/api/docs/x.md             → data/docs/x.md
//	/api/svg?view=container    → data/svg@view=container.svg
//	/api/scene?view=c&version=3 → data/scene@version=3,view=c.json   (query keys sorted)
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
			parts = append(parts, k+"="+q.Get(k))
		}
		name += "@" + strings.Join(parts, ",")
	}
	return "data/" + name + ext
}

// Export returns the published site's data, keyed by StaticName: the latest version, and the
// baseline "what changed" is measured against — the version given, or the one before the latest.
// The network-run log travels as a summary only: the payloads stay on the publisher's machine.
func Export(root string, since int64) (map[string][]byte, error) {
	s, err := New(root)
	if err != nil {
		return nil, err
	}
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
	get := func(p string, optional bool) ([]byte, error) {
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
		// Nothing about the publisher's machine travels: absolute paths become repository-relative.
		b := bytes.ReplaceAll(rec.Body.Bytes(), []byte(s.root+"/"), nil)
		return bytes.ReplaceAll(b, []byte(s.root), []byte(".")), nil
	}
	put := func(p string, optional bool) error {
		b, err := get(p, optional)
		if err == nil && b != nil {
			out[StaticName(p)] = b
		}
		return err
	}

	paths := []string{"/api/summary", "/api/model", "/api/rules", "/api/docs", "/api/completeness", "/api/views", "/api/runs"}
	for _, view := range []string{"context", "container"} {
		paths = append(paths, "/api/scene?view="+view, "/api/svg?view="+view)
	}
	versions := []int64{latest.ID}
	if base != 0 {
		versions = append(versions, base)
		paths = append(paths, fmt.Sprintf("/api/diff?from=%d&to=%d", base, latest.ID))
		for _, view := range []string{"context", "container"} {
			paths = append(paths, fmt.Sprintf("/api/scene?view=%s&version=%d", view, base))
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
	if err := put("/api/coverage", true); err != nil {
		return nil, err
	}

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
		"mode": "published", "remote": render.Remote(s.root), "commit": latest.Commit,
		"generated_at": latest.CreatedAt, "version": latest.ID, "baseline": base,
	}, "", "  ")
	return out, nil
}
