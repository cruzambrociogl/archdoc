package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
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
)

func generate(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	toStdout := fs.Bool("stdout", false, "print the index instead of writing files")
	gaps := fs.Bool("explain-gaps", false, "list what the configuration does not state")
	label := fs.Bool("label", false, "ask Claude for names, descriptions and edge labels")

	flags, positional := partitionArgs(fs, args)
	if err := fs.Parse(flags); err != nil {
		return err
	}

	root := "."
	if len(positional) > 0 {
		root = positional[0]
	}

	facts, err := extract.Scan(root)
	if err != nil {
		return err
	}
	if facts.Source == "" {
		return fmt.Errorf("no deployable Compose file found in %s", facts.Root)
	}

	m := model.Derive(facts)

	// RUL-04 — corrections apply after extraction and before validation. Rules run on every
	// generate rather than being applied once to an output file a later run overwrites.
	rf, err := rules.Load(facts.Root)
	if err != nil {
		return err
	}

	// Rules are compiled against the model as extracted, before anything relabels it: a rule
	// matching `name: api` must still match after the model has renamed api to "API".
	ops, ruleFindings := rf.Compile(facts, m)
	if !ruleFindings.OK() {
		return fmt.Errorf("rules.yaml is not usable, nothing written\n%s", ruleFindings.Error())
	}

	// The semantic layer is opt-in and runs before the rules are applied, so a person's
	// correction always overrides a model's suggestion. Without --label no request is made
	// and the diagram is complete anyway — that is AC-2.
	if *label {
		rec := &semantic.Recorder{}
		started := time.Now()
		labelled, rep, err := semantic.Label(context.Background(), semantic.Claude(semantic.Model, rec), semantic.Model, m)

		// AC-8 — logged whether or not labelling succeeded. A request that left the machine is
		// in the log; a failed run is exactly the one someone goes looking for.
		runID := logRun(facts.Root, rec, rep, started, err, out)
		if err != nil {
			return fmt.Errorf("labelling failed, nothing written: %w", err)
		}
		m = labelled

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
			return fmt.Errorf("rules.yaml produced an invalid model, nothing written\n%s",
				applied.Error())
		}
	}

	// VAL-08: nothing is written unless the whole model is sound. A documentation generator
	// that emits a diagram it knows to be wrong is worse than one that emits nothing, because
	// the reader cannot tell.
	result := validate.Model(m)
	if !result.OK() {
		return fmt.Errorf("model failed validation, nothing written\n%s", result.Error())
	}

	meta := render.Meta{
		Tool:   archdoc.Build().String(),
		Commit: render.Commit(facts.Root),
		Source: facts.Source,
		Rules:  citations(rf),
	}

	// VIE-03/04 — positions are computed once per architecture and stored with its version. A
	// run that finds the architecture unchanged draws from the stored coordinates, so the
	// picture cannot shift between identical runs. --stdout writes nothing, so it draws nothing.
	var layouts map[string]archdoc.Layout
	if !*toStdout {
		layouts = layoutViews(facts.Root, m, out)
		meta.Pictures = layouts != nil
	}

	// Which human-owned sections already exist. Asked of the filesystem rather than by
	// opening the file: OUT-03 forbids reading them, and their existence is all the index
	// needs to report completeness.
	existing := render.State{}
	for _, s := range render.Sections() {
		if s.Owner != render.Human {
			continue
		}
		_, err := os.Stat(filepath.Join(facts.Root, docsDir, s.File()))
		existing[s.File()] = err == nil
	}

	index := render.Index(m, meta)

	if *toStdout {
		_, err := io.WriteString(out, index)
		return err
	}

	// Tool-owned output, overwritten in full.
	generated := map[string]string{
		render.IndexFile: index,
		"context.mmd":    render.Mermaid(m.Context(), false),
		"container.mmd":  render.Mermaid(m.Container(), true),
	}
	if meta.Pictures {
		generated["context.svg"] = render.SVG(m.Context(), layouts["context"])
		generated["container.svg"] = render.SVG(m.Container(), layouts["container"])
	}
	for name, content := range render.Arc42(m, meta) {
		generated[name] = content
	}

	for _, name := range sortedKeys(generated) {
		if err := write(facts.Root, filepath.Join(docsDir, name), generated[name]); err != nil {
			return err
		}
		fmt.Fprintf(out, "wrote %s\n", filepath.Join(docsDir, name))
	}

	if err := write(facts.Root, modelOut, encode(m)); err != nil {
		return err
	}
	fmt.Fprintf(out, "wrote %s\n", modelOut)

	// OUT-02 and OUT-03 — the regeneration boundary. A human-owned section is created once,
	// with questions derived from this model, and after that archdoc neither reads nor writes
	// it. A documentation generator that eats someone's writing gets uninstalled once.
	created := 0
	stubs := render.Stubs(m, meta)
	for _, name := range sortedKeys(stubs) {
		if existing[name] {
			continue
		}
		if err := write(facts.Root, filepath.Join(docsDir, name), stubs[name]); err != nil {
			return err
		}
		// Remember the stub's size and time, so completeness can later tell "still the stub"
		// from "written" by asking the filesystem, never by opening the file (OUT-03).
		if err := render.RecordStub(facts.Root, docsDir, name); err != nil {
			fmt.Fprintf(out, "completeness tracking unavailable for %s: %v\n", name, err)
		}
		created++
	}

	// MEM-01 — history is recorded after the model is known good, never before. A version
	// nothing validated is a version nobody can trust to diff against.
	if err := record(facts.Root, m, layouts, meta, out); err != nil {
		return err
	}

	fmt.Fprintf(out, "\n%d elements, %d relationships, from %s\n",
		len(m.Nodes), len(m.Edges), m.Source)

	if created > 0 {
		fmt.Fprintf(out, "%d section(s) created for you to write — see %s\n",
			created, filepath.Join(docsDir, render.IndexFile))
	}

	if n := len(rf.Rules); n > 0 {
		fmt.Fprintf(out, "%d rule(s) applied from %s\n", n, rules.Name)
	}
	for _, f := range ruleFindings.Warnings() {
		fmt.Fprintf(out, "  %s: %s — %s\n", f.Rule, f.Element, f.Message)
	}

	// Completeness, not correctness. The model is sound; these are the things configuration
	// does not state and the semantic layer exists to fill.
	if w := result.Warnings(); len(w) > 0 {
		fmt.Fprintf(out, "%d gap(s) — run with --explain-gaps to list them\n", len(w))
		if *gaps {
			for _, f := range w {
				fmt.Fprintf(out, "  %s: %s — %s\n", f.Rule, f.Element, f.Message)
			}
		}
	}

	return nil
}

// record appends this model to the repository's history, unless nothing has changed.
//
// A failure here is reported and does not stop the run: the documentation is the deliverable and
// history is an index over it. Refusing to write a correct diagram because a cache could not be
// updated would be the wrong trade.
func record(root string, m archdoc.Model, layouts map[string]archdoc.Layout, meta render.Meta, out io.Writer) error {
	// archdoc's own cache should not land in someone's commit. model.json beside it is
	// committed on purpose — that is the durable record — but a binary index conflicts on
	// every parallel run and diffs as noise, and it can be rebuilt by regenerating.
	if err := write(root, stateDir+"/.gitignore", ignoreFile); err != nil {
		return err
	}

	h, err := store.Open(root)
	if err != nil {
		fmt.Fprintf(out, "history unavailable: %v\n", err)
		return nil
	}
	defer h.Close()

	id, changed, err := h.Save(m, layouts, meta.Commit, meta.Tool)
	if err != nil {
		fmt.Fprintf(out, "history unavailable: %v\n", err)
		return nil
	}

	if changed {
		fmt.Fprintf(out, "recorded version %d\n", id)
		return nil
	}
	fmt.Fprintf(out, "architecture unchanged since version %d\n", id)
	return nil
}

// ignoreFile keeps history local while leaving model.json committed.
const ignoreFile = `# Written by archdoc.
#
# history.db is a local index over the models archdoc has produced. It is rebuildable by
# regenerating, so it is not worth the merge conflicts a binary file in git causes.
#
# model.json is deliberately NOT ignored: it is the durable, reviewable record of the
# architecture at this commit, and git is the thing designed for storing that.
history.db
history.db-shm
history.db-wal
`

// logRun records one use of the network: every request exactly as it left, and what it cost. A
// failure to write the log is reported and does not stop the run — but it is reported, because a
// run log with a silent gap is not a run log.
func logRun(root string, rec *semantic.Recorder, rep semantic.Report, started time.Time, runErr error, out io.Writer) int64 {
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
		Commit: render.Commit(root), EgressMode: "structure-only", Model: rep.Model,
		BytesSent: rec.Bytes(), TokensIn: rep.InputTokens, TokensOut: rep.OutputTokens,
		CostUSD: cost, CostKnown: known,
	}
	for _, ex := range rec.Exchanges() {
		r.Exchanges = append(r.Exchanges, store.Exchange{
			Method: ex.Method, URL: ex.URL, Status: ex.Status, Body: string(ex.Body),
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

// layoutViews returns the positions for the context and container views: the stored ones when
// history already holds this exact architecture, freshly computed ones otherwise.
//
// A layout failure is reported and does not stop the run. The documentation falls back to the
// Mermaid diagrams, which need no layout — a missing picture is a gap, and a run that refused to
// write correct documentation because a drawing failed would be the wrong trade.
func layoutViews(root string, m archdoc.Model, out io.Writer) map[string]archdoc.Layout {
	if h, err := store.Open(root); err == nil {
		defer h.Close()
		if latest, err := h.Latest(); err == nil && latest != nil {
			if fp, err := store.Fingerprint(m); err == nil && fp == latest.Fingerprint {
				c1, okCtx := latest.Layouts["context"]
				c2, okCon := latest.Layouts["container"]
				// Only a layout from the running engine is reused; an older one is recomputed.
				if okCtx && okCon && c1.Version == render.LayoutVersion && c2.Version == render.LayoutVersion {
					return latest.Layouts
				}
			}
		}
	}

	ctx := context.Background()
	contextLayout, err := render.Layout(ctx, m.Context(), false)
	if err != nil {
		fmt.Fprintf(out, "layout unavailable, Mermaid diagrams only: %v\n", err)
		return nil
	}
	containerLayout, err := render.Layout(ctx, m.Container(), true)
	if err != nil {
		fmt.Fprintf(out, "layout unavailable, Mermaid diagrams only: %v\n", err)
		return nil
	}
	return map[string]archdoc.Layout{"context": contextLayout, "container": containerLayout}
}

// citations names the rules that were applied, for the stamp OUT-04 puts on every generated
// file. A reader who meets a technology they did not expect can find the line that set it.
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

func encode(m archdoc.Model) string {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b) + "\n"
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
