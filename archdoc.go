// Package archdoc reads a repository and produces the architecture documentation it should have
// had — accurate, traceable to the code, and regenerable as the system changes.
//
// This is the engine's public API: everything the archdoc command does, it does through this
// package, so anything else can do the same — the hosted platform's worker, a CI step, a script.
// The pipeline lives in internal/; this package is its one door.
//
// Nothing here prints on its own. What a run has to say goes to the io.Writer the caller passes
// (GenerateOptions.Log), line for line what the command prints; a caller that passes none hears
// nothing. Nothing here makes a network request either, unless the caller asks for one with
// GenerateOptions.Label or GenerateOptions.Explain and its Consent agrees — the only two things that
// send anything, through internal/semantic (AC-8).
package archdoc

import (
	"errors"

	"github.com/cruzambrociogl/archdoc/internal/core"
	"github.com/cruzambrociogl/archdoc/internal/model"
	"github.com/cruzambrociogl/archdoc/internal/semantic"
	"github.com/cruzambrociogl/archdoc/internal/store"
)

// The model and its parts. Aliases of the engine's own types, so a value from this package is the
// value the pipeline built — no copy, no conversion, and the same JSON.
type (
	Model       = core.Model
	Node        = core.Node
	Edge        = core.Edge
	Entry       = core.Entry
	Flow        = core.Flow
	Access      = core.Access
	Explanation = core.Explanation
	Provenance  = core.Provenance
	Layout      = core.Layout
	Kind        = core.Kind
	FactSet     = core.FactSet
	App         = core.App
	BuildInfo   = core.BuildInfo
	Progress    = core.Progress
)

// Kinds of element the command's own output distinguishes.
const (
	Component = core.Component
	Table     = core.Table
	Module    = core.Module
)

// Referenced is the evidence of an element the repository only points at — named in an address,
// never declared.
const Referenced = core.Referenced

// What changed between two models: the comparison, and each change classified (MEM-05).
type (
	Delta = model.Diff
	Drift = model.Drift
)

// The classes a change is put in.
const (
	Added           = model.Added
	Removed         = model.Removed
	Renamed         = model.Renamed
	Rebounded       = model.Rebounded
	ProtocolChanged = model.ProtocolChanged
	Changed         = model.Changed
)

// What history holds: a version of the architecture, and a run that used the network with every
// request it sent.
type (
	Version  = store.Version
	Run      = store.Run
	Exchange = store.Exchange
)

// Plan is one request a paid run would send, built by the same function that sends it, so what is
// shown before sending is what would go.
type Plan = semantic.Planned

// Build is this engine's version and build.
func Build() BuildInfo { return core.Build() }

// ErrStopped ends a run that stopped on purpose — a dry run, or a "no" to sending — with nothing
// written. A Consent returns it to stop; Generate returns it unchanged.
var ErrStopped = errors.New("stopped")

// Where archdoc keeps what it writes inside a repository. The markdown is the deliverable and is
// committed; the model is machine truth and is committed beside it (product-definition §8).
const (
	DocsDir   = docsDir
	StateDir  = stateDir
	ModelFile = modelOut
	SiteDir   = siteDir
)
