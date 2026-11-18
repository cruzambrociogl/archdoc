package archdoc

import (
	"context"
	"errors"
	"fmt"

	"github.com/cruzambrociogl/archdoc/internal/semantic"
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
	// DryRun asks the Consent, then stops with ErrStopped: nothing sent.
	DryRun bool
	// WriteModel is the model the writing request goes to: WriteModel (Haiku) when empty; Sonnet for
	// a fuller answer at about twenty times the price.
	WriteModel string
	// Cap is the most the run may cost, in US dollars: a run estimated above it is not sent. Zero is
	// no cap.
	Cap float64
}

// ErrOverCap is a run whose estimate is above its cap. Nothing was sent.
var ErrOverCap = errors.New("not sent: the estimate is above the cap")

const interpretSends = "names, kinds and technologies of the containers and their connections; components with file " +
	"names; tables with their columns; entry points with their paths, handlers and the code's own one-line " +
	"summaries; and the steps of a few flows, as class and method names — no code"

// Interpret explains m. It says what it will send and what that should cost before sending — the
// second request is sized as if every seed were a group of its own, an upper bound — and returns the
// explanation with what it cost. On a failure, what was spent so far still comes back.
func Interpret(ctx context.Context, m Model, o InterpretOptions) (*Interpretation, error) {
	writer := o.WriteModel
	if writer == "" {
		writer = semantic.WriteModel
	}
	group, write, groupOut, writeOut := semantic.InterpretPlans(m)
	g, gKnown := semantic.Cost(semantic.GroupModel, group.Tokens(), groupOut)
	w, wKnown := semantic.Cost(writer, write.Tokens(), writeOut)
	estimate := g + w
	if o.Cap > 0 && estimate > o.Cap {
		return nil, fmt.Errorf("%w: about $%.3f, cap $%.3f", ErrOverCap, estimate, o.Cap)
	}
	if o.Consent == nil {
		return nil, errNoConsent
	}
	err := o.Consent(Request{
		What: "interpret", Model: models(semantic.GroupModel, writer), Sends: interpretSends,
		Plans: []Plan{group, write}, Bytes: group.Bytes() + write.Bytes(),
		Estimate: estimate, EstimateKnown: gKnown && wKnown,
	})
	if err != nil {
		return nil, err
	}
	if o.DryRun {
		return nil, ErrStopped
	}
	return semantic.Interpret(ctx,
		semantic.ClaudeWith(semantic.GroupModel, semantic.GroupSchema, 16000, "medium", nil), semantic.GroupModel,
		semantic.ClaudeWith(writer, semantic.WriteSchema, 16000, "medium", nil), writer,
		m)
}

// models names the models a run asks, once each.
func models(group, write string) string {
	if group == write {
		return group
	}
	return group + ", then " + write
}
