package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
	"github.com/cruzambrociogl/archdoc/internal/extract"
	"github.com/cruzambrociogl/archdoc/internal/model"
	"github.com/cruzambrociogl/archdoc/internal/rules"
	"github.com/cruzambrociogl/archdoc/internal/store"
	"github.com/cruzambrociogl/archdoc/internal/validate"
)

// diff is F-37: what changed in the architecture between two commits, or between a commit and
// the files as they are now — every change classified (MEM-05), whether or not archdoc was ever
// run at either. Each side is read the way generate reads it, corrections included and no model
// asked, so the comparison is of what the repository states and nothing else.
func diff(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	asJSON := fs.Bool("json", false, "emit the comparison as JSON")

	flags, positional := partitionArgs(fs, args)
	if err := fs.Parse(flags); err != nil {
		return err
	}
	if len(positional) < 2 || len(positional) > 3 {
		return fmt.Errorf("usage: archdoc diff <path> <commit> [<commit>] — one commit compares it with the files as they are")
	}
	root, from := positional[0], positional[1]

	before, err := modelAt(root, from)
	if err != nil {
		return err
	}
	to := "the working tree"
	var after archdoc.Model
	if len(positional) == 3 {
		to = positional[2]
		after, err = modelAt(root, to)
	} else {
		after, err = modelOf(root)
	}
	if err != nil {
		return err
	}

	d := model.Compare(before, after)
	drift := d.Drift()
	if *asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(struct {
			From  string        `json:"from"`
			To    string        `json:"to"`
			Drift []model.Drift `json:"drift"`
			Diff  model.Diff    `json:"diff"`
		}{from, to, append([]model.Drift{}, drift...), d})
	}

	fmt.Fprintf(out, "%s → %s\n\n", from, to)
	if len(drift) == 0 {
		fmt.Fprintln(out, "no change to the architecture")
		return nil
	}
	counts := map[string]int{}
	for _, c := range drift {
		counts[c.Class]++
		fmt.Fprintf(out, "%-17s %s", c.Class, c.Element)
		if c.Detail != "" {
			fmt.Fprintf(out, " — %s", c.Detail)
		}
		fmt.Fprintln(out)
	}
	fmt.Fprintln(out)
	for _, class := range []string{model.Added, model.Removed, model.Renamed, model.Rebounded, model.ProtocolChanged, model.Changed} {
		if counts[class] > 0 {
			fmt.Fprintf(out, "%d %s  ", counts[class], class)
		}
	}
	fmt.Fprintln(out)
	return nil
}

// modelAt is the model of a repository as it was at a revision.
func modelAt(root, rev string) (archdoc.Model, error) {
	dir, remove, err := store.Tree(root, rev)
	if err != nil {
		return archdoc.Model{}, err
	}
	defer remove()
	m, err := modelOf(dir)
	if err != nil {
		return m, fmt.Errorf("at %s: %w", rev, err)
	}
	return m, nil
}

// modelOf is the model generate would write for a directory: extracted, corrected by its rules,
// validated. No model is asked, and nothing is written.
func modelOf(root string) (archdoc.Model, error) {
	facts, err := extract.Scan(root)
	if err != nil {
		return archdoc.Model{}, err
	}
	if facts.Source == "" && !hasContainerApp(facts.Apps) {
		return archdoc.Model{}, fmt.Errorf("no deployable Compose file and no application manifest found")
	}
	m := model.Derive(facts)
	rf, err := rules.Load(facts.Root)
	if err != nil {
		return m, err
	}
	ops, findings := rf.Compile(facts, m)
	if !findings.OK() {
		return m, fmt.Errorf("%s is not usable\n%s", rf.Path, findings.Error())
	}
	if len(ops) > 0 {
		var applied validate.Result
		if m, applied = validate.Apply(m, ops); !applied.OK() {
			return m, fmt.Errorf("rules.yaml produced an invalid model\n%s", applied.Error())
		}
	}
	if result := validate.Model(m); !result.OK() {
		return m, fmt.Errorf("model failed validation\n%s", result.Error())
	}
	return m, nil
}
