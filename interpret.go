package archdoc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cruzambrociogl/archdoc/internal/render"
	"github.com/cruzambrociogl/archdoc/internal/semantic"
	"github.com/cruzambrociogl/archdoc/internal/store"
)

// The explanation of a whole system — what it is and how it is organised, its features, concepts and
// areas, key scenarios and a review — written by a model over the facts archdoc proves, each claim
// citing them and marked. Two requests: one groups what archdoc seeded by rule, one writes; both on
// Haiku unless told otherwise. No code is sent.

// The explanation, and what it holds. Aliases of the engine's own types.
type (
	Interpretation = semantic.Interpretation
	Claim          = semantic.Claim
	Group          = semantic.Group
	Scenario       = semantic.Scenario
	ScenarioStep   = semantic.ScenarioStep
	Point          = semantic.Point
	Call           = semantic.Call
)

// The models an interpretation asks, and the marks its claims carry.
const (
	GroupModel  = semantic.GroupModel
	WriteModel  = semantic.WriteModel
	Haiku       = semantic.Haiku
	Sonnet      = semantic.Sonnet
	Proven      = semantic.Proven
	WrittenByAI = semantic.WrittenByAI
)

// InterpretOptions are what an interpretation may spend, and who agrees to it.
type InterpretOptions struct {
	// Consent is asked before anything is sent, with both requests and what they should cost. A nil
	// Consent refuses.
	Consent Consent
	// DryRun asks the Consent, then stops with ErrStopped: nothing sent. (Interpret only; Generate
	// stops a dry run on its own.)
	DryRun bool
	// WriteModel is the model the writing request goes to: WriteModel (Haiku) when empty; Sonnet for
	// a fuller answer at about twenty times the price.
	WriteModel string
	// Cap is the most the run may cost, in US dollars: a run estimated above it is not sent. Zero is
	// no cap.
	Cap float64
	// Again asks anew even when the repository's interpretation is of this very architecture.
	// (Generate only: Interpret always asks.)
	Again bool
}

// InterpretationFile is where Generate keeps the interpretation, beside the model it explains: one
// file, replaced by each new one, and committed like the model, so no one pays for it twice.
// (interpretations.json, beside it, is explain's memory of each component's sentences.)
const InterpretationFile = stateDir + "/interpretation.json"

// ErrOverCap is a run whose estimate is above its cap. Nothing was sent.
var ErrOverCap = errors.New("not sent: the estimate is above the cap")

const interpretSends = "names, kinds and technologies of the containers and their connections; components with file " +
	"names; tables with their columns; entry points with their paths, handlers and the code's own one-line " +
	"summaries; the diagrams' names; and the steps of a few flows, as class and method names — no code"

func (o InterpretOptions) writer() string {
	if o.WriteModel == "" {
		return semantic.WriteModel
	}
	return o.WriteModel
}

// Interpret explains m. It says what it will send and what that should cost before sending — the
// second request is sized as if every seed were a group of its own, an upper bound — and returns the
// explanation with what it cost. On a failure, what was spent so far still comes back.
// GenerateOptions.Interpret does the same inside a run, and keeps the result.
func Interpret(ctx context.Context, m Model, o InterpretOptions) (*Interpretation, error) {
	if err := askInterpret(m, o); err != nil {
		return nil, err
	}
	if o.DryRun {
		return nil, ErrStopped
	}
	return sendInterpret(ctx, m, o.writer(), nil)
}

// askInterpret refuses a run above its cap, then puts both requests to the Consent.
func askInterpret(m Model, o InterpretOptions) error {
	writer := o.writer()
	group, write, groupOut, writeOut := semantic.InterpretPlans(m)
	g, gKnown := semantic.Cost(semantic.GroupModel, group.Tokens(), groupOut)
	w, wKnown := semantic.Cost(writer, write.Tokens(), writeOut)
	estimate := g + w
	if o.Cap > 0 && estimate > o.Cap {
		return fmt.Errorf("%w: about $%.3f, cap $%.3f", ErrOverCap, estimate, o.Cap)
	}
	if o.Consent == nil {
		return errNoConsent
	}
	return o.Consent(Request{
		What: "interpret", Model: models(semantic.GroupModel, writer), Sends: interpretSends,
		Plans: []Plan{group, write}, Bytes: group.Bytes() + write.Bytes(),
		Estimate: estimate, EstimateKnown: gKnown && wKnown,
	})
}

// sendInterpret sends both requests, through rec when there is one to log them.
func sendInterpret(ctx context.Context, m Model, writer string, rec *semantic.Recorder) (*Interpretation, error) {
	it, err := semantic.Interpret(ctx,
		semantic.ClaudeWith(semantic.GroupModel, semantic.GroupSchema, 16000, "medium", rec), semantic.GroupModel,
		semantic.ClaudeWith(writer, semantic.WriteSchema, 16000, "medium", rec), writer,
		m)
	if it != nil {
		it.Fingerprint, _ = Fingerprint(m)
	}
	return it, err
}

// ReadInterpretation is the interpretation Generate kept in the repository at root, if any.
func ReadInterpretation(root string) (*Interpretation, error) {
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(InterpretationFile)))
	if err != nil {
		return nil, err
	}
	var it Interpretation
	if err := json.Unmarshal(b, &it); err != nil {
		return nil, fmt.Errorf("%s: %w", InterpretationFile, err)
	}
	return &it, nil
}

// models names the models a run asks, once each.
func models(group, write string) string {
	if group == write {
		return group
	}
	return group + ", then " + write
}

// interpretIn is GenerateOptions.Interpret's work once the documentation is written: send, log the
// run, keep the result. A failure costs the documentation nothing; what was spent is logged.
func interpretIn(ctx context.Context, root, shown string, m Model, o InterpretOptions, out io.Writer) (*Interpretation, error) {
	rec := &semantic.Recorder{}
	started := time.Now()
	writer := o.writer()
	it, err := sendInterpret(ctx, m, writer, rec)
	runID := logInterpretation(root, models(semantic.GroupModel, writer), rec, it, started, err, out)
	if err != nil {
		if runID > 0 {
			fmt.Fprintf(out, "run %d logged — 'archdoc runs %s --show %d' prints exactly what was sent\n", runID, shown, runID)
		}
		return nil, fmt.Errorf("the explanation failed; the documentation is written: %w", err)
	}
	b, _ := json.MarshalIndent(it, "", "  ")
	if err := write(root, InterpretationFile, string(b)+"\n"); err != nil {
		return nil, err
	}
	fixed := 0
	for _, n := range it.Fixes {
		fixed += n
	}
	fmt.Fprintf(out, "explained by %s: %d feature(s), %d scenario(s), %d review point(s), %d diagram note(s); %d slip(s) fixed in code\n",
		models(semantic.GroupModel, writer), len(it.Features), len(it.Scenarios), len(it.Review), len(it.Notes), fixed)
	cost := "cost unknown"
	if it.CostKnown {
		cost = fmt.Sprintf("$%.4f", it.Cost)
	}
	fmt.Fprintf(out, "sent %d request(s), %d bytes — names and structure, no code · %s · wrote %s\n", len(rec.Exchanges()), rec.Bytes(), cost, InterpretationFile)
	if runID > 0 {
		fmt.Fprintf(out, "run %d logged — 'archdoc runs %s --show %d' prints exactly what was sent\n", runID, shown, runID)
	}
	return it, nil
}

// logInterpretation records an interpretation's requests in the run log, as a label or explain run's.
func logInterpretation(root, model string, rec *semantic.Recorder, it *Interpretation, started time.Time, runErr error, out io.Writer) int64 {
	h, err := store.Open(root)
	if err != nil {
		fmt.Fprintf(out, "run log unavailable: %v\n", err)
		return 0
	}
	defer h.Close()
	status := "ok"
	if runErr != nil {
		status = runErr.Error()
	}
	r := store.Run{StartedAt: started, FinishedAt: time.Now(), Status: status, Commit: render.Commit(root),
		EgressMode: semantic.StructureAndSummaries, Model: model, BytesSent: rec.Bytes()}
	if it != nil {
		for _, c := range it.Calls {
			r.TokensIn += c.InputTokens
			r.TokensOut += c.OutputTokens
		}
		r.CostUSD, r.CostKnown = it.Cost, it.CostKnown
	}
	for _, ex := range rec.Exchanges() {
		r.Exchanges = append(r.Exchanges, store.Exchange{Method: ex.Method, URL: ex.URL, Status: ex.Status, Body: string(ex.Body), Response: string(ex.Response)})
	}
	r.Requests = len(r.Exchanges)
	id, err := h.SaveRun(r)
	if err != nil {
		fmt.Fprintf(out, "run log unavailable: %v\n", err)
		return 0
	}
	return id
}
