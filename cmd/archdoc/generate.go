package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
	"github.com/cruzambrociogl/archdoc/internal/extract"
	"github.com/cruzambrociogl/archdoc/internal/model"
	"github.com/cruzambrociogl/archdoc/internal/render"
	"github.com/cruzambrociogl/archdoc/internal/rules"
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

	flags, positional := partitionArgs(args)
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

	ops, ruleFindings := rf.Compile(facts, m)
	if !ruleFindings.OK() {
		return fmt.Errorf("rules.yaml is not usable, nothing written\n%s", ruleFindings.Error())
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
		created++
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
