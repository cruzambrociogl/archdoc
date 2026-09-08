package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

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
	docsDir      = "docs/architecture"
	stateDir     = ".archdoc"
	documentOut  = docsDir + "/architecture.generated.md"
	contextOut   = docsDir + "/context.mmd"
	containerOut = docsDir + "/container.mmd"
	modelOut     = stateDir + "/model.json"
)

func generate(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	toStdout := fs.Bool("stdout", false, "print the document instead of writing files")
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

	// RUL-04 — corrections apply after extraction and before validation. A rule is the only
	// thing that survives regeneration, so it has to run on every generate rather than being
	// applied once to an output file someone later overwrites.
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

	document := render.Document(m)

	if *toStdout {
		_, err := io.WriteString(out, document)
		return err
	}

	files := []struct {
		path    string
		content string
	}{
		{documentOut, document},
		{contextOut, render.Mermaid(m.Context(), false)},
		{containerOut, render.Mermaid(m.Container(), true)},
		{modelOut, encode(m)},
	}

	for _, f := range files {
		if err := write(facts.Root, f.path, f.content); err != nil {
			return err
		}
		fmt.Fprintf(out, "wrote %s\n", filepath.Join(facts.Root, f.path))
	}

	fmt.Fprintf(out, "\n%d elements, %d relationships, from %s\n",
		len(m.Nodes), len(m.Edges), m.Source)

	// Completeness, not correctness. The model is sound; these are the things configuration
	// does not state and the semantic layer exists to fill.
	if n := len(rf.Rules); n > 0 {
		// RUL-06 — a reader has to be able to tell which parts of this document are
		// corrections rather than readings.
		fmt.Fprintf(out, "%d rule(s) applied from %s\n", n, rules.Name)
	}
	for _, f := range ruleFindings.Warnings() {
		fmt.Fprintf(out, "  %s: %s — %s\n", f.Rule, f.Element, f.Message)
	}

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

// write puts one generated file into the repository being documented.
//
// Only paths this file names are ever written, and each is overwritten in full. archdoc reads
// repositories it does not own; a documentation generator that edits someone's own writing gets
// uninstalled once.
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
