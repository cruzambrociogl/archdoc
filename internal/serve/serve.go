// Package serve is `archdoc serve`: one local process serving a small JSON API over the stored
// model, and the web app that displays it (SUR-07 to SUR-15).
//
// The app is thin in authority (docs/stack-decision.md §2.5): every endpoint here reads what the
// engine already computed and stored. Nothing is laid out, derived or decided in the browser: the
// explorer draws the scene the engine stored — the same layout the committed SVG is drawn from —
// which is what keeps the browser picture and the committed one identical.
//
// The few endpoints that run or write something are actions, and every one passes the gate in
// guard.go (surface-spec §11.5).
//
// This package listens; it does not call out. CI exempts it from the network rule because it
// serves localhost, and the only outbound request it can make is the dev build's proxy to the
// frontend development server, also on localhost.
package serve

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
	"github.com/cruzambrociogl/archdoc/internal/arrange"
	"github.com/cruzambrociogl/archdoc/internal/extract"
	"github.com/cruzambrociogl/archdoc/internal/model"
	"github.com/cruzambrociogl/archdoc/internal/render"
	"github.com/cruzambrociogl/archdoc/internal/rules"
	"github.com/cruzambrociogl/archdoc/internal/store"
	"github.com/cruzambrociogl/archdoc/web"
)

// docsDir mirrors the directory generate writes into.
const docsDir = "docs/architecture"

// Server answers for one repository.
type Server struct {
	root    string
	history string // where the version history is kept: the repository, or a temporary seed (export.go)
	mux     *http.ServeMux
	token   string // this run's session token; see guard.go
}

// New prepares a server for the repository at root. It needs history to exist: the app displays
// what generate stored, so a repository archdoc has never documented has nothing to show yet.
func New(root string) (*Server, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(abs, store.File)); err != nil {
		return nil, fmt.Errorf("no archdoc history in %s — run 'archdoc generate %s' first", abs, root)
	}
	return newServer(abs, abs), nil
}

// newServer answers for the repository at root, reading version history from history's
// .archdoc/history.db — the repository itself, except when export seeds history from model.json.
func newServer(root, history string) *Server {
	abs := root
	s := &Server{root: abs, history: history, mux: http.NewServeMux(), token: newToken()}
	s.mux.HandleFunc("GET /api/session", s.session)
	s.mux.HandleFunc("GET /api/summary", s.summary)
	s.mux.HandleFunc("GET /api/versions", s.versions)
	s.mux.HandleFunc("GET /api/model", s.model)
	s.mux.HandleFunc("GET /api/svg", s.svg)
	s.mux.HandleFunc("GET /api/scene", s.scene)
	s.mux.HandleFunc("GET /api/diff", s.diff)
	s.mux.HandleFunc("GET /api/runs", s.runs)
	s.mux.HandleFunc("GET /api/runs/{id}", s.run)
	s.mux.HandleFunc("GET /api/rules", s.rules)
	s.mux.HandleFunc("GET /api/docs", s.docs)
	s.mux.HandleFunc("GET /api/docs/{name}", s.doc)
	s.mux.HandleFunc("GET /api/completeness", s.completeness)
	s.mux.HandleFunc("GET /api/questions", s.questions)
	s.mux.HandleFunc("GET /api/views", s.views)
	s.mux.HandleFunc("GET /api/coverage", s.coverage)

	// Actions — each one behind the gate (guard.go).
	s.mux.HandleFunc("/api/layout", s.action(s.saveLayout))
	s.mux.HandleFunc("/api/layout/reset", s.action(s.resetLayout))
	s.mux.HandleFunc("/api/views/save", s.action(s.saveView))
	s.mux.Handle("/", s.frontend())
	return s
}

// ServeHTTP refuses any request not addressed to this machine by name. A page on another site
// can make a browser send requests to localhost; checking the Host header is what stops that
// page reading someone's architecture through a rebound DNS name.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	if h, _, err := net.SplitHostPort(r.Host); err == nil {
		host = h
	}
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		http.Error(w, "archdoc serves this machine only", http.StatusForbidden)
		return
	}
	compressed(s.mux).ServeHTTP(w, r)
}

// ListenAndServe serves on addr until ctx ends. It binds to the loopback interface only; addr's
// host part is ignored so a flag cannot accidentally expose the model to the network.
func ListenAndServe(ctx context.Context, root string, port int, ready func(url string)) error {
	srv, err := New(root)
	if err != nil {
		return err
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return err
	}
	hs := &http.Server{Handler: srv, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shut, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		hs.Shutdown(shut)
	}()
	if ready != nil {
		ready(fmt.Sprintf("http://localhost:%d", ln.Addr().(*net.TCPAddr).Port))
	}
	if err := hs.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// ——— the API ———

// coverage serves the coverage report as generate computed it (.archdoc/coverage.json): what was
// read, what could not be resolved, and what the evidence cannot state. The same data the committed
// coverage page is rendered from.
func (s *Server) coverage(w http.ResponseWriter, r *http.Request) {
	b, err := os.ReadFile(filepath.Join(s.root, ".archdoc", render.CoverageJSONFile))
	if errors.Is(err, os.ErrNotExist) {
		notFound(w, "no coverage report yet — run archdoc generate with this version of archdoc")
		return
	}
	if err != nil {
		fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

// questions serves the questions a human-owned section raises, regenerated from the model of the
// requested version. The section's file is never opened (OUT-03); the stub it started from was
// built from the same function.
func (s *Server) questions(w http.ResponseWriter, r *http.Request) {
	v, err := s.version(r, "version")
	if err != nil {
		fail(w, err)
		return
	}
	n, err := strconv.Atoi(r.URL.Query().Get("section"))
	if err != nil {
		fail(w, errors.New("section must be a number"))
		return
	}
	for _, sec := range render.Sections() {
		if sec.Number == n && sec.Owner == render.Human {
			send(w, map[string]any{"section": n, "title": sec.Title, "markdown": render.Questions(n, v.Model)})
			return
		}
	}
	fail(w, fmt.Errorf("section %d is not one a person writes", n))
}

func (s *Server) open() (*store.Store, error) { return store.Open(s.history) }

func (s *Server) summary(w http.ResponseWriter, r *http.Request) {
	h, err := s.open()
	if err != nil {
		fail(w, err)
		return
	}
	defer h.Close()

	latest, err := h.Latest()
	if err != nil || latest == nil {
		fail(w, fmt.Errorf("no version recorded"))
		return
	}
	versions, _ := h.Versions(1000)
	runs, _ := h.Runs(1000)
	_, built := web.Assets()

	send(w, map[string]any{
		"name": latest.Model.Name, "root": s.root, "source": latest.Source,
		"commit": latest.Commit, "latest_version": latest.ID,
		"versions": len(versions), "runs": len(runs),
		"frontend_built": built || web.DevServer() != "",
	})
}

func (s *Server) versions(w http.ResponseWriter, r *http.Request) {
	h, err := s.open()
	if err != nil {
		fail(w, err)
		return
	}
	defer h.Close()

	list, err := h.Versions(1000)
	if err != nil {
		fail(w, err)
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, v := range list {
		out = append(out, map[string]any{
			"id": v.ID, "created_at": v.CreatedAt, "commit": v.Commit, "source": v.Source,
		})
	}
	send(w, out)
}

// version loads the version the request names, or the latest.
func (s *Server) version(r *http.Request, param string) (*store.Version, error) {
	h, err := s.open()
	if err != nil {
		return nil, err
	}
	defer h.Close()

	if raw := r.URL.Query().Get(param); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%s must be a version number", param)
		}
		v, err := h.Get(id)
		if err == nil && v == nil {
			err = fmt.Errorf("no version %d", id)
		}
		return v, err
	}
	v, err := h.Latest()
	if err == nil && v == nil {
		err = errors.New("no version recorded")
	}
	return v, err
}

func (s *Server) model(w http.ResponseWriter, r *http.Request) {
	v, err := s.version(r, "version")
	if err != nil {
		fail(w, err)
		return
	}
	send(w, map[string]any{
		"version": v.ID, "created_at": v.CreatedAt, "commit": v.Commit,
		"model": v.Model, "context": v.Model.Context(), "container": v.Model.Container(),
		"components": orEmpty(v.Model.Components()),
	})
}

// scene is a view with the layout it is drawn at — what the app draws, and what the committed SVG
// is drawn from. Both come from laidOut, so the page and the repository show one arrangement
// (surface-spec §10.1). The report says what a person placed, what is new since, and which saved
// positions name nothing; the hash is the layout.yaml the page must hand back to save.
func (s *Server) scene(w http.ResponseWriter, r *http.Request) {
	v, name, view, l, rep, hash, err := s.laidOut(r)
	if errors.Is(err, errNoView) {
		notFound(w, err.Error())
		return
	}
	if err != nil {
		fail(w, err)
		return
	}
	send(w, map[string]any{
		"version": v.ID, "view": name, "model": view, "layout": l,
		// The containers that have a component view, so a container can say it opens and the
		// component level can offer the others.
		"components": opens(v.Model, archdoc.Component, v.Model.Components()),
		// The containers whose code declares tables: they open onto a data view.
		"data": opens(v.Model, archdoc.Table, v.Model.Datas()),
		// The containers with more components than a diagram shows, which open on their main ones.
		"main": opens(v.Model, archdoc.Component, render.Mains(v.Model)),
		// The containers whose components are features, and so have a by-folder view as well.
		"structure": opens(v.Model, archdoc.Module, v.Model.Structures()),
		// How many containers the system box opens onto, in the context view.
		"containers": containers(v.Model),
		// What the code does that could not be tied to an element, for the boxes it leaves from.
		"unresolved": unresolvedOrEmpty(v.Model),
		"arrangement": map[string]any{"file": arrange.Dir + "/" + arrange.LayoutFile, "hash": hash,
			"placed": rep.Placed, "new": rep.New, "stale": rep.Stale},
	})
}

// svg draws a view exactly as the committed SVG draws it — the export of the same scene.
func (s *Server) svg(w http.ResponseWriter, r *http.Request) {
	_, _, view, l, _, _, err := s.laidOut(r)
	if errors.Is(err, errNoView) {
		notFound(w, err.Error())
		return
	}
	if err != nil {
		fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.Write([]byte(render.SVG(view, l)))
}

// laidOut resolves the requested version and view at the layout it is drawn at: the stored one
// when it is current, computed only when missing or from an older engine — then arranged as a
// person placed it in layout.yaml.
func (s *Server) laidOut(r *http.Request) (*store.Version, string, archdoc.Model, archdoc.Layout, arrange.Report, string, error) {
	none := func(err error) (*store.Version, string, archdoc.Model, archdoc.Layout, arrange.Report, string, error) {
		return nil, "", archdoc.Model{}, archdoc.Layout{}, arrange.Report{}, "", err
	}
	v, err := s.version(r, "version")
	if err != nil {
		return none(err)
	}
	name := r.URL.Query().Get("view")
	if name == "" {
		name = "container"
	}
	view, l, err := engineLayout(r.Context(), v, name)
	if err != nil {
		return none(err)
	}
	a, hash, err := arrange.LoadLayout(s.root)
	if err != nil {
		return none(err)
	}
	arranged, rep := arrange.Apply(name, l, a)
	return v, name, view, arranged, rep, hash, nil
}

// errNoView is a view the version does not have: a container with no component view, or one
// from another version.
var errNoView = errors.New("no such view")

// engineLayout is a version's view at the engine's own layout: the stored one when current.
func engineLayout(ctx context.Context, v *store.Version, name string) (archdoc.Model, archdoc.Layout, error) {
	view, ok := render.ViewOf(v.Model, name)
	if !ok {
		return archdoc.Model{}, archdoc.Layout{}, fmt.Errorf("%w: %q in version %d", errNoView, name, v.ID)
	}
	l, ok := v.Layouts[name]
	if !ok || l.Version != render.LayoutVersion {
		var err error
		if l, err = render.Layout(ctx, view.Model, view.Group); err != nil {
			return archdoc.Model{}, archdoc.Layout{}, err
		}
	}
	return view.Model, l, nil
}

// opening is a container with a component view: its name, and how many components it has.
type opening struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Components int    `json:"components"`
}

func opens(m archdoc.Model, kind archdoc.Kind, containers []string) []opening {
	count := map[string]int{}
	for _, n := range m.Nodes {
		if n.Kind == kind {
			count[n.Parent]++
		}
	}
	out := []opening{}
	for _, id := range containers {
		c, _ := m.Node(id)
		out = append(out, opening{ID: id, Name: c.Name, Components: count[id]})
	}
	return out
}

func containers(m archdoc.Model) int {
	n := 0
	for _, x := range m.Nodes {
		if x.Kind.Container() {
			n++
		}
	}
	return n
}

func unresolvedOrEmpty(m archdoc.Model) []archdoc.Unresolved {
	if m.Unresolved == nil {
		return []archdoc.Unresolved{}
	}
	return m.Unresolved
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func (s *Server) diff(w http.ResponseWriter, r *http.Request) {
	from, err := s.version(r, "from")
	if err != nil {
		fail(w, err)
		return
	}
	to, err := s.version(r, "to")
	if err != nil {
		fail(w, err)
		return
	}
	d := model.Compare(from.Model, to.Model)
	send(w, map[string]any{
		"from": from.ID, "to": to.ID, "structural": d.Structural(), "empty": d.Empty(), "diff": d,
	})
}

func (s *Server) runs(w http.ResponseWriter, r *http.Request) {
	h, err := s.open()
	if err != nil {
		fail(w, err)
		return
	}
	defer h.Close()
	list, err := h.Runs(1000)
	if err != nil {
		fail(w, err)
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, run := range list {
		out = append(out, runJSON(run))
	}
	send(w, out)
}

func (s *Server) run(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		fail(w, errors.New("run must be a number"))
		return
	}
	h, err := s.open()
	if err != nil {
		fail(w, err)
		return
	}
	defer h.Close()
	run, err := h.Run(id)
	if err != nil {
		fail(w, err)
		return
	}
	if run == nil {
		notFound(w, fmt.Sprintf("no run %d", id))
		return
	}
	out := runJSON(*run)
	out["exchanges"] = run.Exchanges // exactly as sent — the page shows them unformatted
	send(w, out)
}

func runJSON(r store.Run) map[string]any {
	return map[string]any{
		"id": r.ID, "started_at": r.StartedAt, "finished_at": r.FinishedAt, "status": r.Status,
		"commit": r.Commit, "egress_mode": r.EgressMode, "model": r.Model,
		"requests": r.Requests, "bytes_sent": r.BytesSent,
		"tokens_in": r.TokensIn, "tokens_out": r.TokensOut,
		"cost_usd": r.CostUSD, "cost_known": r.CostKnown,
	}
}

// rules shows what rules.yaml says and what it did (SUR-11): each rule, the operations it
// compiled to against the model as extracted, and the rules that matched nothing or overrode
// another.
func (s *Server) rules(w http.ResponseWriter, r *http.Request) {
	f, err := rules.Load(s.root)
	if err != nil {
		fail(w, err)
		return
	}
	facts, err := extract.Scan(s.root)
	if err != nil {
		fail(w, err)
		return
	}
	ops, findings := f.Compile(facts, model.Derive(facts))

	list := make([]map[string]any, 0, len(f.Rules))
	for _, rule := range f.Rules {
		entry := map[string]any{
			"line": rule.Line, "set": rule.Set, "exclude": rule.Exclude, "remove": rule.Remove,
			"match": map[string]string{"name": rule.Match.Name, "image": rule.Match.Image, "kind": rule.Match.Kind},
		}
		if rule.Edge != nil {
			entry["edge"] = map[string]string{"from": rule.Edge.From, "to": rule.Edge.To}
		}
		list = append(list, entry)
	}

	found := make([]map[string]any, 0, len(findings.Findings))
	for _, fd := range findings.Findings {
		found = append(found, map[string]any{
			"rule": fd.Rule, "severity": fd.Severity, "element": fd.Element, "message": fd.Message,
		})
	}

	send(w, map[string]any{"file": f.Path, "exists": len(f.Rules) > 0, "rules": list, "operations": ops, "findings": found,
		"legacy": f.Legacy, "shadowed": f.Shadowed})
}

// docs lists what generate wrote, and the human sections by name only (SUR-12).
func (s *Server) docs(w http.ResponseWriter, r *http.Request) {
	var generated []string
	for _, name := range append([]string{render.IndexFile, render.CoverageFile}, generatedSections()...) {
		if _, err := os.Stat(filepath.Join(s.root, docsDir, name)); err == nil {
			generated = append(generated, name)
		}
	}
	var human []map[string]any
	for _, sec := range render.Sections() {
		if sec.Owner == render.Human {
			human = append(human, map[string]any{"number": sec.Number, "title": sec.Title, "file": sec.File()})
		}
	}
	send(w, map[string]any{"generated": generated, "human": human, "dir": filepath.Join(s.root, docsDir)})
}

func generatedSections() []string {
	var out []string
	for _, sec := range render.Sections() {
		if sec.Owner == render.Generated {
			out = append(out, sec.File())
		}
	}
	return out
}

// doc returns one file archdoc generated. A human-owned section is refused, not read: OUT-03
// holds in the app exactly as it holds on the command line. The page links to the file in the
// editor instead.
func (s *Server) doc(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name != path.Base(name) || strings.ContainsAny(name, `/\`) {
		fail(w, errors.New("a document name, not a path"))
		return
	}
	if !(strings.HasSuffix(name, ".generated.md") || strings.HasSuffix(name, ".svg") || strings.HasSuffix(name, ".mmd")) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "this section is yours: archdoc never reads it. Open it in your editor.",
		})
		return
	}

	b, err := os.ReadFile(filepath.Join(s.root, docsDir, name))
	if errors.Is(err, fs.ErrNotExist) {
		notFound(w, "not generated yet: "+name)
		return
	}
	if err != nil {
		fail(w, err)
		return
	}
	switch {
	case strings.HasSuffix(name, ".svg"):
		w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	default:
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	}
	w.Write(b)
}

// completeness reports each human section by asking the filesystem (SUR-15, OUT-09). "May be
// stale" compares against when the latest version was recorded — the last time the architecture
// changed.
func (s *Server) completeness(w http.ResponseWriter, r *http.Request) {
	v, err := s.version(r, "version")
	if err != nil {
		fail(w, err)
		return
	}
	// basis=size is what a published site is built with: its files come from a fresh checkout,
	// where modification times mean nothing, so only sizes are compared and staleness is not
	// claimed.
	basis := render.BySizeAndTime
	if r.URL.Query().Get("basis") == "size" {
		basis = render.BySize
	}
	states, err := render.SectionStates(s.root, docsDir, v.CreatedAt, basis)
	if err != nil {
		fail(w, err)
		return
	}
	out := make([]map[string]any, 0, len(states))
	for _, st := range states {
		entry := map[string]any{
			"number": st.Section.Number, "title": st.Section.Title, "file": st.File, "state": st.State,
		}
		if !st.Modified.IsZero() {
			entry["modified"] = st.Modified
		}
		out = append(out, entry)
	}
	send(w, map[string]any{"architecture_changed": v.CreatedAt, "sections": out, "dir": filepath.Join(s.root, docsDir)})
}

// ——— the page ———

// frontend serves the built app, proxies to the development server in the dev build, or explains
// how to build the app when neither is available — never a blank page.
func (s *Server) frontend() http.Handler {
	if dev := web.DevServer(); dev != "" {
		target, _ := url.Parse(dev)
		return httputil.NewSingleHostReverseProxy(target)
	}

	assets, built := web.Assets()
	if !built {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, `<!doctype html><title>archdoc</title><body style="font-family:system-ui;max-width:40em;margin:4em auto">
<h1>The web app has not been built</h1>
<p>This archdoc binary was built without its frontend. From the archdoc repository, run:</p>
<pre>npm --prefix web install
npm --prefix web run build
go build -o ./archdoc ./cmd/archdoc</pre>
<p>The API is running regardless: <a href="/api/summary">/api/summary</a>.</p></body>`)
		})
	}

	files := http.FileServerFS(assets)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A single-page app: any path that is not a file is the app itself.
		if _, err := fs.Stat(assets, strings.TrimPrefix(path.Clean(r.URL.Path), "/")); err != nil {
			r.URL.Path = "/"
		}
		files.ServeHTTP(w, r)
	})
}

// ——— helpers ———

func send(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func fail(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func notFound(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
