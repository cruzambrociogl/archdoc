package archdoc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cruzambrociogl/archdoc/internal/arrange"
	"github.com/cruzambrociogl/archdoc/internal/core"
	"github.com/cruzambrociogl/archdoc/internal/extract"
	"github.com/cruzambrociogl/archdoc/internal/model"
	"github.com/cruzambrociogl/archdoc/internal/render"
	"github.com/cruzambrociogl/archdoc/internal/rules"
	"github.com/cruzambrociogl/archdoc/internal/semantic"
	"github.com/cruzambrociogl/archdoc/internal/store"
	"github.com/cruzambrociogl/archdoc/internal/validate"
)

// Where output lands inside the repository being documented. §8: the markdown is the
// deliverable and is committed; model.json is machine truth and is committed alongside it.
const (
	docsDir  = "docs/architecture"
	stateDir = ".archdoc"
	modelOut = stateDir + "/model.json"
	// coverageOut is the coverage report as data, committed beside model.json.
	coverageOut = stateDir + "/" + render.CoverageJSONFile
)

// GenerateOptions are what a run of the pipeline does beyond reading and writing. The zero value
// writes the documentation, records nothing, says nothing and sends nothing.
type GenerateOptions struct {
	// Log receives what the run has to say, line for line what 'archdoc generate' prints. Nil is
	// silence.
	Log io.Writer
	// IndexOnly writes nothing: the index goes to Log and to Generated.Index instead (--stdout).
	IndexOnly bool
	// Gaps lists every gap in the log rather than counting them (--gaps).
	Gaps bool
	// Site also writes mkdocs.yml, so the documents build as a static site (--site).
	Site bool
	// Verbose lists every file written.
	Verbose bool
	// History records this model as a version in .archdoc/history.db, and draws from the layout
	// stored there when the architecture is unchanged. The command sets it; a caller with a
	// throwaway checkout, whose history is kept elsewhere, leaves it off.
	History bool
	// Label asks a model for the system's descriptions and relationship labels; Explain asks what
	// each component does. Each sends a request, costs money, and goes ahead only if its Consent
	// agrees. Nil is never.
	Label   *LabelOptions
	Explain *ExplainOptions
	// Progress is told what the run is doing as it goes — the applications found, each one as its
	// code is read, the diagrams drawn — for a screen to show while a large repository is read.
	// Nil is no one.
	Progress func(Progress)
}

// LabelOptions is a label run: the model to ask, and who agrees to sending.
type LabelOptions struct {
	Model   string
	Consent Consent
}

// ExplainOptions is an explain run: which components to ask about, and who agrees to sending.
// DryRun stops with ErrStopped, nothing sent, when there is nothing to ask either.
type ExplainOptions struct {
	Only    string
	Limit   int
	DryRun  bool
	Consent Consent
}

// Generated is what a run produced.
type Generated struct {
	// Root is the repository, as an absolute path.
	Root string
	// Model is the validated model the documentation was written from.
	Model Model
	// Index is the index document; with IndexOnly, the only thing produced.
	Index string
	// Files are the files written, relative to Root, in the order they were written.
	Files []string
	// Version is the history version this model is, and Changed whether this run recorded it.
	// Both are zero without History, or when history was unavailable.
	Version int64
	Changed bool
}

// Generate is the pipeline: read, derive, correct, ask if asked to, validate, write, record.
// Nothing is written unless the whole model is sound.
func Generate(ctx context.Context, root string, o GenerateOptions) (*Generated, error) {
	out := o.Log
	if out == nil {
		out = io.Discard
	}
	res := &Generated{}
	progress := o.Progress
	if progress == nil {
		progress = func(Progress) {}
	}
	facts, err := extract.ScanReporting(root, progress)
	if err != nil {
		return nil, err
	}
	res.Root = facts.Root
	if facts.Source == "" && !hasContainerApp(facts.Apps) {
		return nil, fmt.Errorf("nothing to document in %s: no Compose file, no application manifest, and no Python, JavaScript or HTML files", facts.Root)
	}
	if !o.IndexOnly {
		fmt.Fprintf(out, "documenting %s\n", facts.Root)
	}

	m := model.Derive(facts)

	// RUL-04 — corrections apply after extraction and before validation. Rules run on every
	// generate rather than being applied once to an output file a later run overwrites.
	rf, err := rules.Load(facts.Root)
	if err != nil {
		return nil, err
	}

	// Rules are compiled against the model as extracted, before anything relabels it: a rule
	// matching `name: api` must still match after the model has renamed api to "API".
	ops, ruleFindings := rf.Compile(facts, m)
	if !ruleFindings.OK() {
		return nil, fmt.Errorf("%s is not usable, nothing written\n%s", rf.Path, ruleFindings.Error())
	}

	// The semantic layer is opt-in and runs before the rules are applied, so a person's
	// correction always overrides a model's suggestion. Without archdoc label no request is made
	// and the diagram is complete anyway — that is AC-2.
	labels, err := loadLabels(facts.Root)
	if err != nil {
		return nil, err
	}
	extracted := m
	// What a model wrote before is applied again while what it was written about is unchanged
	// (F-30): no request, no cost, and the documents do not lose their descriptions. It is applied
	// before a new label run too, which then writes over it: a label run and the plain run after it
	// must give the same model, or the second records a version nothing caused.
	if recalled := semantic.Recall(labels, extracted); len(recalled) > 0 {
		if again, res := validate.Apply(m, recalled); res.OK() {
			m = again
		} else {
			fmt.Fprintf(out, "remembered labels no longer fit the model and were left out — 'archdoc label' writes new ones\n")
		}
	}
	if o.Label != nil {
		plan, err := semantic.LabelPlan(m)
		if err != nil {
			return nil, err
		}
		if err := o.Label.Consent.ask("label", o.Label.Model, labelSends, []Plan{plan}, labelAnswer(plan)); err != nil {
			return nil, err
		}
		rec := &semantic.Recorder{}
		started := time.Now()
		labelled, rep, err := semantic.Label(ctx, semantic.Claude(o.Label.Model, rec), o.Label.Model, m)

		// AC-8 — logged whether or not labelling succeeded. A request that left the machine is
		// in the log; a failed run is exactly the one someone goes looking for.
		runID := logRun(facts.Root, semantic.StructureOnly, rec, rep, started, err, out)
		if err != nil {
			return nil, fmt.Errorf("labelling failed, nothing written: %w", err)
		}
		m = labelled
		if kept := semantic.Remember(labels, extracted, rep.Accepted); len(kept) > 0 {
			pretty, _ := json.MarshalIndent(kept, "", "  ")
			if werr := write(facts.Root, labelsOut, string(pretty)+"\n"); werr != nil {
				return nil, werr
			}
		}

		cost := "cost unknown for this model"
		if c, ok := semantic.Cost(rep.Model, rep.InputTokens, rep.OutputTokens); ok {
			cost = fmt.Sprintf("$%.4f", c)
		}
		fmt.Fprintf(out, "labelled by %s: %d operation(s) in %d attempt(s)\n", rep.Model, rep.Ops, rep.Attempts)
		fmt.Fprintf(out, "sent %d request(s), %d bytes, structure only · %d tokens in, %d out · %s\n",
			len(rec.Exchanges()), rec.Bytes(), rep.InputTokens, rep.OutputTokens, cost)
		if runID > 0 {
			fmt.Fprintf(out, "run %d logged — 'archdoc runs %s --show %d' prints exactly what was sent\n",
				runID, root, runID)
		}
	}

	if len(ops) > 0 {
		var applied validate.Result
		m, applied = validate.Apply(m, ops)
		if !applied.OK() {
			return nil, fmt.Errorf("rules.yaml produced an invalid model, nothing written\n%s",
				applied.Error())
		}
	}

	// F-19, F-30 — what each component does. Answers remembered against their facts apply on every
	// run, with no request; archdoc explain asks only about components whose facts are new or changed.
	memory, err := loadMemory(facts.Root)
	if err != nil {
		return nil, err
	}
	var asker semantic.Completer
	var rec *semantic.Recorder
	only, limit := "", 0
	if x := o.Explain; x != nil {
		only, limit = x.Only, x.Limit
		plans := semantic.ExplainPlan(m, memory, only, limit)
		if len(plans) == 0 {
			fmt.Fprintln(out, "nothing to ask: every component has a remembered answer for its facts, or is too small to ask about")
			if x.DryRun {
				fmt.Fprintln(out, "dry run — nothing sent, nothing written.")
				return nil, ErrStopped
			}
		} else {
			if err := x.Consent.ask("explain", semantic.ExplainModel, explainSends, plans, explainAnswer*int64(len(plans))); err != nil {
				return nil, err
			}
			rec = &semantic.Recorder{}
			asker = semantic.ClaudeWith(semantic.ExplainModel, semantic.ExplainSchema(), semantic.ExplainMaxTokens, "low", rec)
		}
	}
	started := time.Now()
	explained, remembered, xrep, err := semantic.ExplainSome(ctx, asker, semantic.ExplainModel, m, memory, only, limit)
	// The memory is written the moment answers are paid for, not at the end of a run something
	// later could stop — and it only grows: every answer it held stays, unless a newer answer
	// about the same element replaces it. A plain run writes it too when it has something to add:
	// which element an older answer is about.
	if merged := mergeMemory(memory, remembered); len(merged) > 0 {
		before, _ := json.Marshal(memory)
		after, merr := json.Marshal(merged)
		if merr == nil && string(before) != string(after) {
			pretty, _ := json.MarshalIndent(merged, "", "  ")
			if werr := write(facts.Root, memoryOut, string(pretty)+"\n"); werr != nil {
				return nil, werr
			}
		}
	}
	if asker != nil {
		runID := logRun(facts.Root, semantic.StructureAndSummaries, rec, xrep.Report, started, err, out)
		if err == nil {
			fmt.Fprintf(out, "explained by %s: %d component(s) asked, %d remembered, %d too small to ask about, %d sentence(s) refused for want of a citation\n",
				xrep.Model, xrep.Asked, xrep.Remembered, xrep.Skipped, xrep.Refused)
			if len(xrep.Unanswered) > 0 {
				fmt.Fprintf(out, "no usable answer for %d, left unexplained: %s\n", len(xrep.Unanswered), strings.Join(xrep.Unanswered, ", "))
			}
			if xrep.Stopped != "" {
				fmt.Fprintf(out, "asking stopped %s — what was answered before is kept; 'archdoc explain' again asks for the rest\n", xrep.Stopped)
			}
			if c, ok := semantic.Cost(xrep.Model, xrep.InputTokens, xrep.OutputTokens); ok {
				fmt.Fprintf(out, "sent %d request(s), %d bytes — names, paths and the code's own route summaries, no code · $%.4f\n", len(rec.Exchanges()), rec.Bytes(), c)
			}
			if runID > 0 {
				fmt.Fprintf(out, "run %d logged — 'archdoc runs %s --show %d' prints exactly what was sent\n", runID, root, runID)
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("explaining failed, nothing written: %w", err)
	}
	m = explained

	// VAL-08: nothing is written unless the whole model is sound. A documentation generator
	// that emits a diagram it knows to be wrong is worse than one that emits nothing, because
	// the reader cannot tell.
	result := validate.Model(m)
	if !result.OK() {
		return nil, fmt.Errorf("model failed validation, nothing written\n%s", result.Error())
	}

	meta := render.Meta{
		Tool:   core.Build().String(),
		Commit: render.Commit(facts.Root),
		Source: facts.Source,
		Rules:  citations(rf),
	}

	// VIE-03/04 — positions are computed once per architecture and stored with its version. A
	// run that finds the architecture unchanged draws from the stored coordinates, so the
	// picture cannot shift between identical runs. --stdout writes nothing, so it draws nothing.
	var layouts map[string]core.Layout
	if !o.IndexOnly {
		progress(Progress{Doing: "drawing the diagrams", Found: found(m)})
		layouts = layoutViews(facts.Root, m, o.History, out)
		meta.Pictures = layouts != nil
		progress(Progress{Doing: "writing the documents", Found: found(m)})
	}

	// Which sections this repository has evidence for. A single-service repo with no networks
	// gets a document set proportionate to it rather than twelve chapters, four of which would
	// ask about structure it does not have (OUT-01, §8's "partial output is the honest output").
	plan := render.PlanFor(m, *facts)

	// Which human-owned sections already exist. Asked of the filesystem rather than by
	// opening the file: OUT-03 forbids reading them, and their existence is all the index
	// needs to report completeness.
	existing := render.State{}
	for _, s := range plan {
		if s.Owner != render.Human {
			continue
		}
		_, err := os.Stat(filepath.Join(facts.Root, docsDir, s.File()))
		existing[s.File()] = err == nil
	}

	index := render.Index(m, plan, meta)
	res.Model, res.Index = m, index

	if o.IndexOnly {
		_, err := io.WriteString(out, index)
		return res, err
	}

	// Tool-owned output, overwritten in full.
	generated := map[string]string{
		render.IndexFile: index,
		// OUT-09's other half: a document set that says where its own evidence runs out. No
		// competitor in this space publishes what it missed.
		render.CoverageFile: render.Coverage(m, *facts, reported(result), meta),
	}
	// A project small enough for one page has its features and its parts on that page.
	if len(plan) > 0 {
		if features := render.Features(m, meta); features != "" {
			generated[render.FeaturesFile] = features
		}
		if components := render.Components(m, meta); components != "" {
			generated[render.ComponentsFile] = components
		}
	}
	views := render.Views(m)
	for _, v := range views {
		generated[v.File+".mmd"] = render.Mermaid(v.Model, v.Group)
	}
	if meta.Pictures {
		// The committed pictures show the diagrams as a person arranged them in the app
		// (layout.yaml); the engine's own layout is what is stored, so resetting loses nothing.
		arrangement, _, err := arrange.LoadLayout(facts.Root)
		if err != nil {
			fmt.Fprintf(out, "warning: %v — drawing the engine's layout instead\n", err)
			arrangement = arrange.Arrangement{}
		}
		for _, view := range views {
			stored, ok := layouts[view.Name]
			if !ok {
				continue
			}
			l, rep := arrange.Apply(view.Name, stored, arrangement)
			generated[view.File+".svg"] = render.SVG(view.Model, l)
			if len(rep.Stale) > 0 {
				fmt.Fprintf(out, "%s/%s: %d saved position(s) in the %s view match nothing: %s\n",
					arrange.Dir, arrange.LayoutFile, len(rep.Stale), view.Name, strings.Join(rep.Stale, ", "))
			}
		}
	}
	for name, content := range render.Arc42(m, *facts, plan, meta) {
		generated[name] = content
	}
	// Layer 3 of the output contract (§8), opt-in: the committed markdown already stands on its
	// own, so a site is one configuration file away.
	if o.Site {
		generated[render.MkDocsFile] = render.MkDocs(m, plan)
	}

	written := 0
	wrote := func(rel string) {
		written++
		res.Files = append(res.Files, rel)
		if o.Verbose {
			fmt.Fprintf(out, "wrote %s\n", rel)
		}
	}
	for _, name := range sortedKeys(generated) {
		if err := write(facts.Root, filepath.Join(docsDir, name), generated[name]); err != nil {
			return nil, err
		}
		wrote(filepath.Join(docsDir, name))
	}

	// Files only archdoc could have written there, that it did not write this run — a section the
	// document no longer has, an older layout's file, a picture the layout could not draw — would
	// otherwise sit beside the current set looking current. They are removed; nothing else is.
	for _, name := range orphans(facts.Root, generated) {
		if err := os.Remove(filepath.Join(facts.Root, docsDir, name)); err != nil {
			return nil, err
		}
		fmt.Fprintf(out, "removed %s — archdoc no longer writes it\n", filepath.Join(docsDir, name))
	}

	if err := write(facts.Root, modelOut, encode(m)); err != nil {
		return nil, err
	}
	wrote(modelOut)

	// The coverage report as data, beside the model: what the app and the published site show,
	// computed by the same function as the committed coverage page, so the two cannot disagree.
	coverage, err := json.MarshalIndent(render.BuildCoverage(m, *facts, reported(result)), "", "  ")
	if err != nil {
		return nil, err
	}
	if err := write(facts.Root, coverageOut, string(coverage)+"\n"); err != nil {
		return nil, err
	}
	wrote(coverageOut)

	// OUT-02 and OUT-03 — the regeneration boundary. A human-owned section is created once,
	// with questions derived from this model, and after that archdoc neither reads nor writes
	// it. A documentation generator that eats someone's writing gets uninstalled once.
	created := 0
	stubs := render.Stubs(m, plan, meta)
	for _, name := range sortedKeys(stubs) {
		if existing[name] {
			continue
		}
		if err := write(facts.Root, filepath.Join(docsDir, name), stubs[name]); err != nil {
			return nil, err
		}
		wrote(filepath.Join(docsDir, name))
		// Remember the stub's size and time, so completeness can later tell "still the stub"
		// from "written" by asking the filesystem, never by opening the file (OUT-03).
		if err := render.RecordStub(facts.Root, docsDir, name); err != nil {
			fmt.Fprintf(out, "completeness tracking unavailable for %s: %v\n", name, err)
		}
		created++
	}
	if !o.Verbose {
		fmt.Fprintf(out, "wrote %d files into %s and %s — --verbose lists them\n", written, docsDir, stateDir)
	}

	// MEM-01 — history is recorded after the model is known good, never before. A version
	// nothing validated is a version nobody can trust to diff against.
	if o.History {
		id, changed, err := record(facts.Root, m, layouts, meta, out)
		if err != nil {
			return nil, err
		}
		res.Version, res.Changed = id, changed
	}

	fmt.Fprintln(out)
	Count(m).Print(out, m.Source)

	if created > 0 {
		fmt.Fprintf(out, "%d section(s) created for you to write — see %s\n",
			created, filepath.Join(docsDir, render.IndexFile))
	}

	if n := len(rf.Rules); n > 0 {
		fmt.Fprintf(out, "%d rule(s) applied from %s\n", n, rf.Path)
	}
	if rf.Legacy {
		fmt.Fprintf(out, "rules.yaml at the repository root is read for now; move it to %s, where archdoc keeps its other files\n", rules.Name)
	}
	if rf.Shadowed {
		fmt.Fprintf(out, "rules.yaml at the repository root is ignored: %s takes precedence — remove the root one\n", rules.Name)
	}
	for _, f := range ruleFindings.Warnings() {
		fmt.Fprintf(out, "  %s: %s — %s\n", f.Rule, f.Element, f.Message)
	}

	// Completeness, not correctness. The model is sound; these are the things the code and
	// configuration do not state and the semantic layer exists to fill.
	if w := result.Warnings(); len(w) > 0 {
		if o.Gaps {
			fmt.Fprintf(out, "%d gap(s):\n", len(w))
			for _, f := range w {
				fmt.Fprintf(out, "  %s: %s — %s\n", f.Rule, f.Element, f.Message)
			}
		} else {
			fmt.Fprintf(out, "%d gap(s) — 'archdoc generate %s --gaps' lists them\n", len(w), root)
		}
	}

	return res, nil
}

// record appends this model to the repository's history, unless nothing has changed.
//
// A failure here is reported and does not stop the run: the documentation is the deliverable and
// history is an index over it. Refusing to write a correct diagram because a cache could not be
// updated would be the wrong trade.
func record(root string, m core.Model, layouts map[string]core.Layout, meta render.Meta, out io.Writer) (int64, bool, error) {
	// archdoc's own cache should not land in someone's commit. model.json beside it is
	// committed on purpose — that is the durable record — but a binary index conflicts on
	// every parallel run and diffs as noise, and it can be rebuilt by regenerating.
	if err := write(root, stateDir+"/.gitignore", ignoreFile); err != nil {
		return 0, false, err
	}

	h, err := store.Open(root)
	if err != nil {
		fmt.Fprintf(out, "history unavailable: %v\n", err)
		return 0, false, nil
	}
	defer h.Close()

	id, changed, err := h.Save(m, layouts, meta.Commit, meta.Tool)
	if err != nil {
		fmt.Fprintf(out, "history unavailable: %v\n", err)
		return 0, false, nil
	}

	if changed {
		fmt.Fprintf(out, "recorded version %d\n", id)
		return id, true, nil
	}
	fmt.Fprintf(out, "architecture unchanged since version %d\n", id)
	return id, false, nil
}

// ignoreFile keeps history local while leaving model.json committed.
const ignoreFile = `# Written by archdoc.
#
# history.db is a local index over the models archdoc has produced. It is rebuildable by
# regenerating, so it is not worth the merge conflicts a binary file in git causes.
#
# model.json is deliberately NOT ignored: it is the durable, reviewable record of the
# architecture at this commit, and git is the thing designed for storing that. Neither are
# coverage.json, layout.yaml and views.yaml beside it.
#
# site/ is what 'archdoc export --site' builds from the committed record — a build product,
# rebuilt by CI, not something to commit.
history.db
history.db-shm
history.db-wal
site/
`

// logRun records one use of the network: every request exactly as it left, and what it cost. A
// failure to write the log is reported and does not stop the run — but it is reported, because a
// run log with a silent gap is not a run log.
func logRun(root, mode string, rec *semantic.Recorder, rep semantic.Report, started time.Time, runErr error, out io.Writer) int64 {
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
	cost, known := semantic.Cost(rep.Model, rep.InputTokens, rep.OutputTokens)

	r := store.Run{
		StartedAt: started, FinishedAt: time.Now(), Status: status,
		Commit: render.Commit(root), EgressMode: mode, Model: rep.Model,
		BytesSent: rec.Bytes(), TokensIn: rep.InputTokens, TokensOut: rep.OutputTokens,
		CostUSD: cost, CostKnown: known,
	}
	for _, ex := range rec.Exchanges() {
		r.Exchanges = append(r.Exchanges, store.Exchange{
			Method: ex.Method, URL: ex.URL, Status: ex.Status, Body: string(ex.Body), Response: string(ex.Response),
		})
	}
	r.Requests = len(r.Exchanges)

	id, err := h.SaveRun(r)
	if err != nil {
		fmt.Fprintf(out, "run log unavailable: %v\n", err)
		return 0
	}
	return id
}

// layoutViews returns the positions for every view: the stored ones when history already holds
// this exact architecture, freshly computed ones otherwise.
//
// A layout failure is reported and does not stop the run. The documentation falls back to the
// Mermaid diagrams, which need no layout — a missing picture is a gap, and a run that refused to
// write correct documentation because a drawing failed would be the wrong trade.
func layoutViews(root string, m core.Model, history bool, out io.Writer) map[string]core.Layout {
	if !history {
		return computeLayouts(m, out)
	}
	if h, err := store.Open(root); err == nil {
		defer h.Close()
		if latest, err := h.Latest(); err == nil && latest != nil {
			if fp, err := store.Fingerprint(m); err == nil && fp == latest.Fingerprint {
				// Only a layout from the running engine is reused; an older one is recomputed.
				current := true
				for _, v := range render.Views(m) {
					if l, ok := latest.Layouts[v.Name]; !ok || l.Version != render.LayoutVersion || !render.Places(l, v.Model) {
						current = false
					}
				}
				if current {
					return latest.Layouts
				}
			}
		}
	}

	return computeLayouts(m, out)
}

// computeLayouts lays out every view afresh.
func computeLayouts(m core.Model, out io.Writer) map[string]core.Layout {
	ctx := context.Background()
	layouts := map[string]core.Layout{}
	for _, v := range render.Views(m) {
		l, err := render.Layout(ctx, v.Model, v.Group)
		if err != nil {
			fmt.Fprintf(out, "layout unavailable, Mermaid diagrams only: %v\n", err)
			return nil
		}
		layouts[v.Name] = l
	}
	return layouts
}

// citations names the rules that were applied, for the stamp OUT-04 puts on every generated
// file. A reader who meets a technology they did not expect can find the line that set it.
// reported turns the validator's warnings into what the coverage report prints. Warnings are
// completeness, not correctness: the model is sound and these are the things configuration does
// not state (VAL-03, VAL-06). Mapped rather than passed through, so rendering stays independent
// of the validator's types.
// hasContainerApp reports whether any manifest describes a running part of the system — enough to
// document a repository that has no Compose file.
func hasContainerApp(apps []core.App) bool {
	for _, a := range apps {
		if a.Role.Container() {
			return true
		}
	}
	return false
}

// orphans are the files in docs/architecture that only archdoc writes — by name, so a human file can
// never match — and that this run did not write. mkdocs.yml is opt-in per run, so it is left alone.
func orphans(root string, written map[string]string) []string {
	entries, err := os.ReadDir(filepath.Join(root, docsDir))
	if err != nil {
		return nil
	}
	ours := map[string]bool{"context.svg": true, "container.svg": true, "context.mmd": true, "container.mmd": true}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || (!strings.HasSuffix(name, ".generated.md") && !ours[name] && !render.ComponentFile(name)) {
			continue
		}
		if _, ok := written[name]; !ok {
			out = append(out, name)
		}
	}
	sort.Strings(out) // AC-7: the same report every run
	return out
}

func reported(r validate.Result) []render.Gap {
	w := r.Warnings()
	out := make([]render.Gap, 0, len(w))
	for _, f := range w {
		out = append(out, render.Gap{Rule: f.Rule, Element: f.Element, Message: f.Message})
	}
	return out
}

func citations(rf *rules.File) []string {
	out := make([]string, 0, len(rf.Rules))
	for _, r := range rf.Rules {
		out = append(out, fmt.Sprintf("%s:%d", rf.Path, r.Line))
	}
	return out
}

// write puts one generated file into the repository being documented.
//
// Only paths this file names are ever written, and each is overwritten in full. archdoc reads
// repositories it does not own.
func write(root, rel, content string) error {
	path := filepath.Join(root, rel)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func encode(m core.Model) string {
	b, err := m.JSON()
	if err != nil {
		return "{}"
	}
	return string(b)
}

// sortedKeys keeps the order files are written — and therefore the order they are reported —
// stable across runs. AC-7 covers what appears on the terminal too.
func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// memoryOut keeps the model's answers against the facts they were given (F-30): reused while the
// facts are unchanged, so explanations are as stable as the code they describe.
const memoryOut = stateDir + "/interpretations.json"

// mergeMemory is what was remembered before, with this run's answers: an older answer about an
// element gives way to a newer one about the same element, and nothing else is dropped.
func mergeMemory(before, now semantic.Memory) semantic.Memory {
	latest := map[string]string{}
	for fp, r := range now {
		if r.Element != "" {
			latest[r.Element] = fp
		}
	}
	out := semantic.Memory{}
	for fp, r := range before {
		if newer, ok := latest[r.Element]; ok && r.Element != "" && newer != fp {
			continue
		}
		out[fp] = r
	}
	for fp, r := range now {
		out[fp] = r
	}
	return out
}

// labelsOut keeps the labels a model wrote against what each was written about (F-30).
const labelsOut = stateDir + "/labels.json"

func loadLabels(root string) (semantic.Labels, error) {
	b, err := os.ReadFile(filepath.Join(root, labelsOut))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var l semantic.Labels
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, fmt.Errorf("%s: %w", labelsOut, err)
	}
	return l, nil
}

func loadMemory(root string) (semantic.Memory, error) {
	b, err := os.ReadFile(filepath.Join(root, memoryOut))
	if os.IsNotExist(err) {
		return semantic.Memory{}, nil
	}
	if err != nil {
		return nil, err
	}
	var m semantic.Memory
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("%s: %w", memoryOut, err)
	}
	return m, nil
}

// found says what a model holds, for a run's progress: its parts, routes and tables.
func found(m core.Model) []string {
	parts, tables := 0, 0
	for _, n := range m.Nodes {
		switch {
		case n.Kind.Container():
			parts++
		case n.Kind == core.Table:
			tables++
		}
	}
	count := func(n int, one, many string) string {
		if n == 1 {
			return "1 " + one + " found"
		}
		return fmt.Sprintf("%d %s found", n, many)
	}
	out := []string{count(parts, "app or service", "apps and services")}
	if len(m.Entries) > 0 {
		out = append(out, count(len(m.Entries), "route", "routes"))
	}
	if tables > 0 {
		out = append(out, count(tables, "table", "tables"))
	}
	return out
}
