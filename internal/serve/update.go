package serve

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cruzambrociogl/archdoc/internal/store"
)

// Updating, as the app sees it (platform-surface §4.3, §4.8): whether a run is updating the
// repository now — from the terminal, or from the app — who, since when, and what it is doing,
// read from the lock the run holds (store.Take); and updating from the app, which writes only what
// generate writes.
//
// Two updates (decisions, 23 Oct): Update — free, offline, one click — and Update with AI, which
// first shows what each paid run would send and roughly what it costs, from a dry run of the same
// pipeline, and sends only after the person agrees. What they agreed to is a ceiling: a run that
// would send more requests, or cost much more, than they were shown stops before sending.
//
// The pipeline itself is package archdoc's, which this package cannot import; package server gives
// it as an Updater. A server given none — a published site's — has updating switched off.

// Updater runs generate on the repository at root for the app.
type Updater func(ctx context.Context, root string, r Run) (Outcome, error)

// Run is one update the app asks for.
type Run struct {
	// Label and Explain ask a model, each only if Agree agrees to what it would send.
	Label, Explain bool
	// DryRun puts each paid run to Agree and stops with ErrStopped: nothing sent, nothing written.
	DryRun bool
	Agree  func(Ask) error
	// Log receives what generate prints.
	Log io.Writer
}

// Ask is what one paid run would send, as Agree is shown it.
type Ask struct {
	What     string  `json:"what"` // "label" or "explain"
	Model    string  `json:"model"`
	Sends    string  `json:"sends"`
	Requests int     `json:"requests"`
	Bytes    int     `json:"bytes"`
	USD      float64 `json:"usd"`
	Known    bool    `json:"known"`
}

// Outcome is what an update recorded.
type Outcome struct {
	Version int64
	Changed bool
}

// ErrStopped is an Updater stopped before sending or writing: a dry run, or a run Agree refused.
var ErrStopped = errors.New("stopped")

// keyVar is where the person's key is read from: the environment archdoc serve started in.
const keyVar = "ANTHROPIC_API_KEY"

// switchedOff is what the action says where the server was given no Updater.
const switchedOff = "updating isn’t possible here — run archdoc generate in a terminal"

// slack is how far an estimate may grow between the preview and the run before the run stops:
// labels written first can change what explain asks about.
const slack = 1.25

// finished is the last update the app ran, for the page that started it.
type finished struct {
	AI       bool      `json:"ai"`
	Started  time.Time `json:"started"`
	Finished time.Time `json:"finished"`
	Error    string    `json:"error,omitempty"`
	Version  int64     `json:"version,omitempty"`
	Changed  bool      `json:"changed"`
	// Log is the end of what the run printed.
	Log string `json:"log,omitempty"`
}

// updates is the app's own runs on one repository: at most one at a time, and the last one's end.
type updates struct {
	mu      sync.Mutex
	running *time.Time
	ai      bool
	last    *finished
}

func (s *Server) registerUpdate() {
	s.mux.HandleFunc("GET /api/updating", s.updating)
	s.mux.HandleFunc("/api/update/preview", s.action(s.previewUpdate))
	s.mux.HandleFunc("/api/update", s.action(s.startUpdate))
}

// updating is the run updating the repository now, if any, and the last one the app ran.
func (s *Server) updating(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	out := map[string]any{"running": false, "can": s.update != nil}
	s.app.mu.Lock()
	if s.app.last != nil {
		out["last"] = s.app.last
	}
	mine, ai := s.app.running, s.app.ai
	s.app.mu.Unlock()
	if busy, ok := store.Running(s.root); ok {
		out["running"], out["by"], out["since"], out["doing"], out["found"] = true, busy.By, busy.Since, busy.Doing, busy.Found
	} else if mine != nil {
		// Started here, and not holding the repository yet: reading what it will document.
		out["running"], out["by"], out["since"], out["doing"] = true, "the app", mine.UTC(), "reading the repository"
	}
	if mine != nil {
		out["mine"], out["ai"] = true, ai
	}
	send(w, out)
}

// update is the body of both actions: which paid runs, and — to start one — what was agreed to.
type update struct {
	Label   bool           `json:"label"`
	Explain bool           `json:"explain"`
	Agreed  map[string]Ask `json:"agreed"`
}

// previewUpdate says what an update with AI would send and roughly cost: a dry run of the
// pipeline, so what is shown is built by the same functions the requests would be.
func (s *Server) previewUpdate(w http.ResponseWriter, r *http.Request) {
	if s.update == nil {
		refuse(w, http.StatusForbidden, switchedOff)
		return
	}
	var body update
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !(body.Label || body.Explain) {
		refuse(w, http.StatusBadRequest, "the body asks for neither labels nor explanations")
		return
	}
	asks := []Ask{}
	agree := func(a Ask) error { asks = append(asks, a); return nil }
	_, err := s.update(r.Context(), s.root, Run{Label: body.Label, Explain: body.Explain, DryRun: true, Agree: agree, Log: io.Discard})
	if err != nil && !errors.Is(err, ErrStopped) {
		fail(w, err)
		return
	}
	send(w, map[string]any{"asks": asks, "key": strings.TrimSpace(os.Getenv(keyVar)) != ""})
}

// startUpdate starts an update — free, or with AI within what the person agreed to. It returns at
// once; the run is followed through updating.
func (s *Server) startUpdate(w http.ResponseWriter, r *http.Request) {
	if s.update == nil {
		refuse(w, http.StatusForbidden, switchedOff)
		return
	}
	var body update
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		refuse(w, http.StatusBadRequest, "the body is not an update")
		return
	}
	ai := body.Label || body.Explain
	if ai && strings.TrimSpace(os.Getenv(keyVar)) == "" {
		refuse(w, http.StatusBadRequest, keyVar+" isn’t set in the shell archdoc serve was started from — set it there and start archdoc serve again")
		return
	}
	s.app.mu.Lock()
	if s.app.running != nil {
		s.app.mu.Unlock()
		refuse(w, http.StatusConflict, "an update started here is running already")
		return
	}
	if busy, ok := store.Running(s.root); ok {
		s.app.mu.Unlock()
		refuse(w, http.StatusConflict, busy.By+" is updating this repository already")
		return
	}
	now := time.Now()
	s.app.running, s.app.ai = &now, ai
	s.app.mu.Unlock()

	go s.runUpdate(body, now)
	w.WriteHeader(http.StatusAccepted)
	send(w, map[string]any{"started": true})
}

// runUpdate is one update from the app, to its end.
func (s *Server) runUpdate(body update, started time.Time) {
	var log bytes.Buffer
	agree := func(a Ask) error { return withinAgreed(a, body.Agreed[a.What]) }
	out, err := s.update(context.Background(), s.root, Run{Label: body.Label, Explain: body.Explain, Agree: agree, Log: &syncWriter{w: &log}})
	end := &finished{AI: body.Label || body.Explain, Started: started, Finished: time.Now(), Version: out.Version, Changed: out.Changed, Log: tail(log.String(), 40)}
	if err != nil {
		end.Error = err.Error()
	}
	s.app.mu.Lock()
	s.app.running, s.app.last = nil, end
	s.app.mu.Unlock()
}

// withinAgreed lets a paid run go ahead only within what the person was shown and agreed to: no
// more requests, and no more than a little over the cost.
func withinAgreed(a, agreed Ask) error {
	if agreed.What != a.What {
		return fmt.Errorf("%s wasn’t agreed to — nothing sent for it", a.What)
	}
	if a.Requests > agreed.Requests || (a.Known && a.USD > agreed.USD*slack+0.001) {
		return fmt.Errorf("%s would now send %d request(s), about $%.2f — more than the %d, about $%.2f, agreed to; nothing sent for it. Look again for the new figure",
			a.What, a.Requests, a.USD, agreed.Requests, agreed.USD)
	}
	return nil
}

// tail is the last n lines of s.
func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// syncWriter lets the run write while nothing else reads; a run's log is read only once it ends.
type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (s *syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}
