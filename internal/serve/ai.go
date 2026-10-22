package serve

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cruzambrociogl/archdoc/internal/core"
	"github.com/cruzambrociogl/archdoc/internal/extract"
	"github.com/cruzambrociogl/archdoc/internal/model"
	"github.com/cruzambrociogl/archdoc/internal/render"
	"github.com/cruzambrociogl/archdoc/internal/semantic"
	"github.com/cruzambrociogl/archdoc/internal/store"
)

// What the app asks a model, on this machine (platform-surface §6.3, §6.5): Ask, and Did it build
// what I asked? Both say what they would send and roughly what it costs before anything leaves —
// a GET, free — and send only when the person says so — an action, behind the gate (rule 19). Both
// remember an answer against what it was asked, so asking again costs nothing; and both record
// every request in the run log, What left this machine, like --label and --explain.
//
// The key is the person's own, from the environment archdoc serve started in; archdoc never asks
// for it, stores it or shows it.

// plansFile names the plan files a person pointed archdoc at (F-40): a setting, committed with the
// rest of .archdoc, and the only way a plan is read.
const plansFile = ".archdoc/plans.json"

// completer makes the client a request goes through; a test puts a fake in its place.
type completer func(model string, schema map[string]any, maxTokens int64, effort string, rec *semantic.Recorder) semantic.Completer

func defaultCompleter(model string, schema map[string]any, maxTokens int64, effort string, rec *semantic.Recorder) semantic.Completer {
	return semantic.ClaudeWith(model, schema, maxTokens, effort, rec)
}

// estimate is roughly what a request costs: its bytes as tokens, at about 3.5 bytes a token, with
// the schema's and the answer's tokens on top. Rough by design — the run log has the real figure.
func estimate(model string, bytes int, answerTokens int64) map[string]any {
	in := int64(bytes)*10/35 + 400
	usd, known := semantic.Cost(model, in, answerTokens)
	return map[string]any{"usd": usd, "known": known, "tokens_in": in, "tokens_out": answerTokens}
}

func (s *Server) registerAI() {
	s.mux.HandleFunc("GET /api/ask", s.askPreview)
	s.mux.HandleFunc("/api/ask/send", s.action(s.askSend))
	s.mux.HandleFunc("GET /api/plan", s.planPreview)
	s.mux.HandleFunc("/api/plan/files", s.action(s.planFiles))
	s.mux.HandleFunc("/api/plan/check", s.action(s.planCheck))
}

// ——— Ask ———

func (s *Server) latestModel() (core.Model, *store.Store, error) {
	h, err := s.open()
	if err != nil {
		return core.Model{}, nil, err
	}
	v, err := h.Latest()
	if err != nil || v == nil {
		h.Close()
		return core.Model{}, nil, errors.New("no version recorded")
	}
	return v.Model, h, nil
}

// askPreview says what a question would send and cost, or gives the answer already remembered.
func (s *Server) askPreview(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		refuse(w, http.StatusBadRequest, "no question")
		return
	}
	m, h, err := s.latestModel()
	if err != nil {
		fail(w, err)
		return
	}
	defer h.Close()
	facts := semantic.QuestionFacts(m, q)
	prompt := semantic.AskPrompt(q, facts)
	out := map[string]any{"question": q, "facts": len(facts), "model": semantic.ExplainModel, "sends": semantic.StructureAndQuestion,
		"bytes": len(semantic.AskSystem) + len(prompt), "estimate": estimate(semantic.ExplainModel, len(semantic.AskSystem)+len(prompt), 400)}
	if body, at, err := h.Recall("ask", semantic.AnswerKey(q, facts)); err == nil && body != nil {
		out["remembered"], out["remembered_at"] = json.RawMessage(body), at
	}
	send(w, out)
}

// askSend asks: nothing when nothing matches, the remembered answer when there is one, the model
// otherwise — recorded in the run log either way it went out.
func (s *Server) askSend(w http.ResponseWriter, r *http.Request) {
	var body struct{ Q string }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Q) == "" {
		refuse(w, http.StatusBadRequest, "the body names no question")
		return
	}
	q := strings.TrimSpace(body.Q)
	m, h, err := s.latestModel()
	if err != nil {
		fail(w, err)
		return
	}
	defer h.Close()
	facts := semantic.QuestionFacts(m, q)
	key := semantic.AnswerKey(q, facts)
	if len(facts) == 0 {
		send(w, map[string]any{"answer": semantic.Answer{Question: q, Claims: []core.Claim{}}, "sent": false})
		return
	}
	if b, _, err := h.Recall("ask", key); err == nil && b != nil {
		send(w, map[string]any{"answer": json.RawMessage(b), "sent": false, "remembered": true})
		return
	}
	rec := &semantic.Recorder{}
	started := time.Now()
	ans, rep, askErr := semantic.AskQuestion(r.Context(), s.complete(semantic.ExplainModel, semantic.AskSchema(), semantic.AskMaxTokens, semantic.AskEffort, rec),
		semantic.ExplainModel, q, facts)
	run, usd := s.logRun(h, semantic.StructureAndQuestion, rec, rep, started, askErr)
	if askErr != nil {
		refuse(w, http.StatusBadGateway, askErr.Error())
		return
	}
	if b, err := json.Marshal(ans); err == nil {
		h.Keep("ask", key, b)
	}
	send(w, map[string]any{"answer": ans, "sent": true, "run": run, "usd": usd})
}

// ——— the plan ———

type plans struct {
	Files []string `json:"files"`
}

func (s *Server) loadPlans() plans {
	var p plans
	if b, err := os.ReadFile(filepath.Join(s.root, plansFile)); err == nil {
		json.Unmarshal(b, &p)
	}
	return p
}

// planInputs is everything a check would send: the plan's requirements, the facts for them, and
// what the latest version added.
func (s *Server) planInputs(h *store.Store) (items []core.PlanItem, more int, facts, adds []semantic.Fact, err error) {
	files := s.loadPlans().Files
	if len(files) == 0 {
		return nil, 0, nil, nil, nil
	}
	items, more, err = extract.Plan(s.root, files)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	versions, err := h.Versions(2)
	if err != nil || len(versions) == 0 {
		return nil, 0, nil, nil, errors.New("no version recorded")
	}
	latest, err := h.Get(versions[0].ID)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	facts = semantic.PlanFacts(latest.Model, items)
	if len(versions) == 2 {
		if before, err := h.Get(versions[1].ID); err == nil {
			d := model.Compare(before.Model, latest.Model)
			adds = semantic.Additions(d.AddedNodes, d.AddedEntries)
		}
	}
	return items, more, facts, adds, nil
}

// planPreview is the plan as archdoc reads it, what a check would send and cost, and the check
// remembered for this plan and code — or the last one, marked as for an earlier plan or code.
func (s *Server) planPreview(w http.ResponseWriter, r *http.Request) {
	h, err := s.open()
	if err != nil {
		fail(w, err)
		return
	}
	defer h.Close()
	out := map[string]any{"files": orEmpty(s.loadPlans().Files), "model": semantic.ExplainModel, "sends": semantic.StructureAndPlan, "items": []core.PlanItem{}}
	items, more, facts, adds, err := s.planInputs(h)
	if err != nil {
		out["problem"] = err.Error()
		send(w, out)
		return
	}
	prompt := semantic.PlanPrompt(items, facts, adds)
	out["items"], out["more"], out["facts"], out["additions"] = orEmptyItems(items), more, len(facts), len(adds)
	out["bytes"] = len(semantic.PlanSystem) + len(prompt)
	out["estimate"] = estimate(semantic.ExplainModel, len(semantic.PlanSystem)+len(prompt), int64(120*len(items)+300))
	if len(items) > 0 {
		if b, at, err := h.Recall("plan", semantic.CheckKey(items, facts, adds)); err == nil && b != nil {
			out["check"], out["checked_at"] = json.RawMessage(b), at
		} else if b, at, err := h.LastAnswer("plan"); err == nil && b != nil {
			out["earlier"], out["checked_at"] = json.RawMessage(b), at
		}
	}
	send(w, out)
}

// planFiles points archdoc at the plan files to read — each one checked, inside the repository.
func (s *Server) planFiles(w http.ResponseWriter, r *http.Request) {
	var body plans
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		refuse(w, http.StatusBadRequest, "the body names no files")
		return
	}
	var files []string
	seen := map[string]bool{}
	for _, f := range body.Files {
		clean, err := extract.PlanFile(s.root, f)
		if err != nil {
			refuse(w, http.StatusBadRequest, err.Error())
			return
		}
		if !seen[clean] {
			seen[clean] = true
			files = append(files, clean)
		}
	}
	b, _ := json.MarshalIndent(plans{Files: orEmpty(files)}, "", "  ")
	if err := os.WriteFile(filepath.Join(s.root, plansFile), append(b, '\n'), 0o644); err != nil {
		fail(w, err)
		return
	}
	s.planPreview(w, r)
}

// planCheck reads the plan against the code: the remembered check when nothing changed, one
// request otherwise, recorded in the run log.
func (s *Server) planCheck(w http.ResponseWriter, r *http.Request) {
	h, err := s.open()
	if err != nil {
		fail(w, err)
		return
	}
	defer h.Close()
	items, _, facts, adds, err := s.planInputs(h)
	if err != nil {
		refuse(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(items) == 0 {
		refuse(w, http.StatusBadRequest, "no plan: name the plan files first")
		return
	}
	key := semantic.CheckKey(items, facts, adds)
	if b, _, err := h.Recall("plan", key); err == nil && b != nil {
		send(w, map[string]any{"check": json.RawMessage(b), "sent": false, "remembered": true})
		return
	}
	rec := &semantic.Recorder{}
	started := time.Now()
	check, rep, checkErr := semantic.CheckPlan(r.Context(), s.complete(semantic.ExplainModel, semantic.PlanSchema(), semantic.PlanMaxTokens, semantic.PlanEffort, rec),
		semantic.ExplainModel, items, facts, adds)
	run, usd := s.logRun(h, semantic.StructureAndPlan, rec, rep, started, checkErr)
	if checkErr != nil {
		refuse(w, http.StatusBadGateway, checkErr.Error())
		return
	}
	if b, err := json.Marshal(check); err == nil {
		h.Keep("plan", key, b)
	}
	send(w, map[string]any{"check": check, "sent": true, "run": run, "usd": usd})
}

// ——— shared ———

func (s *Server) complete(model string, schema map[string]any, maxTokens int64, effort string, rec *semantic.Recorder) semantic.Completer {
	if s.completer != nil {
		return s.completer(model, schema, maxTokens, effort, rec)
	}
	return defaultCompleter(model, schema, maxTokens, effort, rec)
}

// logRun records what left the machine, as generate does for --label and --explain: every
// request exactly as sent, and what it cost. It returns the run and its cost.
func (s *Server) logRun(h *store.Store, mode string, rec *semantic.Recorder, rep semantic.Report, started time.Time, runErr error) (int64, float64) {
	if len(rec.Exchanges()) == 0 {
		return 0, 0
	}
	status := "ok"
	if runErr != nil {
		status = runErr.Error()
	}
	usd, known := semantic.Cost(rep.Model, rep.InputTokens, rep.OutputTokens)
	run := store.Run{StartedAt: started, FinishedAt: time.Now(), Status: status, Commit: render.Commit(s.root), EgressMode: mode,
		Model: rep.Model, BytesSent: rec.Bytes(), TokensIn: rep.InputTokens, TokensOut: rep.OutputTokens, CostUSD: usd, CostKnown: known}
	for _, ex := range rec.Exchanges() {
		run.Exchanges = append(run.Exchanges, store.Exchange{Method: ex.Method, URL: ex.URL, Status: ex.Status, Body: string(ex.Body), Response: string(ex.Response)})
	}
	run.Requests = len(run.Exchanges)
	id, err := h.SaveRun(run)
	if err != nil {
		fmt.Fprintf(os.Stderr, "archdoc: run log unavailable: %v\n", err)
	}
	return id, usd
}

func orEmptyItems(items []core.PlanItem) []core.PlanItem {
	if items == nil {
		return []core.PlanItem{}
	}
	return items
}
