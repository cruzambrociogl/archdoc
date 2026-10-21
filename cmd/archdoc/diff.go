package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cruzambrociogl/archdoc"
)

// diff is F-37: what changed in the architecture between two commits, or between a commit and
// the files as they are now — every change classified (MEM-05), whether or not archdoc was ever
// run at either. Each side is read the way generate reads it, corrections included and no model
// asked, so the comparison is of what the repository states and nothing else.
func diff(e env, args []string) error {
	fs := flags("diff")
	asJSON := fs.Bool("json", false, "print the comparison as JSON")
	detail := fs.Bool("detail", false, "list every change by its class and id, after the summary")

	positional, err := parse(e, fs, args, nil)
	if err != nil {
		return err
	}
	if len(positional) < 2 || len(positional) > 3 {
		return usagef("archdoc diff <path> <commit> [<commit>] — one commit compares it with the files as they are")
	}
	out := e.out
	root, from, to := positional[0], positional[1], ""
	if len(positional) == 3 {
		to = positional[2]
	}

	c, err := archdoc.Diff(context.Background(), root, from, to)
	if err != nil {
		return err
	}
	drift := c.Drift
	if *asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(struct {
			From  string          `json:"from"`
			To    string          `json:"to"`
			Drift []archdoc.Drift `json:"drift"`
			Diff  archdoc.Delta   `json:"diff"`
		}{c.From, c.To, append([]archdoc.Drift{}, drift...), c.Delta})
	}

	fmt.Fprintf(out, "%s → %s\n\n", c.From, c.To)
	if len(drift) == 0 {
		fmt.Fprintln(out, "no change to the architecture")
		return nil
	}
	structural, words := 0, 0
	for _, r := range c.Summary() {
		if r.Words {
			words++
		} else {
			structural++
		}
		fmt.Fprintf(out, "%s %s\n", r.Mark, r.Text)
	}
	fmt.Fprintf(out, "\n%d change(s) to the structure, %d to wording only", structural, words)
	if !*detail {
		fmt.Fprintln(out, " — --detail lists every one by class and id")
		return nil
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out)
	counts := map[string]int{}
	for _, ch := range drift {
		counts[ch.Class]++
		fmt.Fprintf(out, "%-17s %s", ch.Class, ch.Element)
		if ch.Detail != "" {
			fmt.Fprintf(out, " — %s", ch.Detail)
		}
		fmt.Fprintln(out)
	}
	fmt.Fprintln(out)
	for _, class := range []string{archdoc.Added, archdoc.Removed, archdoc.Renamed, archdoc.Rebounded, archdoc.ProtocolChanged, archdoc.Changed} {
		if counts[class] > 0 {
			fmt.Fprintf(out, "%d %s  ", counts[class], class)
		}
	}
	fmt.Fprintln(out)
	return nil
}
