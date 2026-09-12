// Package serve is `archdoc serve`: one local process serving a small JSON API over the stored
// model, and the web app that displays it (SUR-07 to SUR-15).
//
// The app is thin in authority (docs/stack-decision.md §2.5): every endpoint here reads what the
// engine already computed and stored. Nothing is laid out, derived or decided in the browser, and
// the canvas draws the very SVG the engine writes into the repository — which is what keeps the
// browser picture and the committed one identical.
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
	root string
	mux  *http.ServeMux
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

	s := &Server{root: abs, mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /api/summary", s.summary)
	s.mux.HandleFunc("GET /api/versions", s.versions)
	s.mux.HandleFunc("GET /api/model", s.model)
	s.mux.HandleFunc("GET /api/svg", s.svg)
	s.mux.HandleFunc("GET /api/diff", s.diff)
	s.mux.HandleFunc("GET /api/runs", s.runs)
	s.mux.HandleFunc("GET /api/runs/{id}", s.run)
	s.mux.HandleFunc("GET /api/rules", s.rules)
	s.mux.HandleFunc("GET /api/docs", s.docs)
	s.mux.HandleFunc("GET /api/docs/{name}", s.doc)
	s.mux.HandleFunc("GET /api/completeness", s.completeness)
	s.mux.Handle("/", s.frontend())
	return s, nil
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
	s.mux.ServeHTTP(w, r)
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

func (s *Server) open() (*store.Store, error) { return store.Open(s.root) }

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
	})
}

// svg draws a view exactly as the committed SVG draws it: from the stored layout when there is a
// current one, computing one only when the stored layout is missing or from an older engine.
func (s *Server) svg(w http.ResponseWriter, r *http.Request) {
	v, err := s.version(r, "version")
	if err != nil {
		fail(w, err)
		return
	}

	name := r.URL.Query().Get("view")
	var view archdoc.Model
	switch name {
	case "context":
		view = v.Model.Context()
	case "container", "":
		name, view = "container", v.Model.Container()
	default:
		fail(w, fmt.Errorf("unknown view %q", name))
		return
	}

	l, ok := v.Layouts[name]
	if !ok || l.Version != render.LayoutVersion {
		l, err = render.Layout(r.Context(), view, name == "container")
		if err != nil {
			fail(w, err)
			return
		}
	}

	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.Write([]byte(render.SVG(view, l)))
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

	send(w, map[string]any{"file": f.Path, "exists": len(f.Rules) > 0, "rules": list, "operations": ops, "findings": found})
}

// docs lists what generate wrote, and the human sections by name only (SUR-12).
func (s *Server) docs(w http.ResponseWriter, r *http.Request) {
	var generated []string
	for _, name := range append([]string{render.IndexFile}, generatedSections()...) {
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
	states, err := render.SectionStates(s.root, docsDir, v.CreatedAt)
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
