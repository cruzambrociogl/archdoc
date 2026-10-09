package main

import (
	"fmt"
	"text/tabwriter"

	"github.com/cruzambrociogl/archdoc/internal/extract"
)

// history lists what archdoc has recorded for a repository.
//
// The point is not the list. It is that a run which changes nothing adds nothing — so the number
// of versions is the number of times the architecture actually moved, not the number of times
// somebody ran the tool.
func history(e env, args []string) error {
	fs := flags("history")
	limit := fs.Int("n", 20, "list the latest `count` versions")
	asJSON := fs.Bool("json", false, "print the versions as JSON")

	positional, err := parse(e, fs, args, nil)
	if err != nil {
		return err
	}
	root, err := readPath(fs, positional)
	if err != nil {
		return err
	}
	out := e.out

	facts, err := extract.Scan(root)
	if err != nil {
		return err
	}

	h, ok, err := openHistory(facts.Root)
	if err != nil {
		return err
	}
	if !ok {
		if *asJSON {
			return printJSON(out, []any{})
		}
		fmt.Fprintf(out, "No history yet for %s — 'archdoc generate %s' records the first version.\n", facts.Root, root)
		return nil
	}
	defer h.Close()

	versions, err := h.Versions(*limit)
	if err != nil {
		return err
	}

	if *asJSON {
		list := []map[string]any{}
		for _, v := range versions {
			list = append(list, map[string]any{"version": v.ID, "recorded": v.CreatedAt, "commit": v.Commit, "source": v.Source})
		}
		return printJSON(out, list)
	}
	if len(versions) == 0 {
		fmt.Fprintf(out, "No history yet for %s — 'archdoc generate %s' records the first version.\n", facts.Root, root)
		return nil
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "VERSION\tRECORDED\tCOMMIT\tSOURCE")
	for _, v := range versions {
		commit := v.Commit
		if commit == "" {
			commit = "—"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
			v.ID, v.CreatedAt.Format("2006-01-02 15:04"), commit, v.Source)
	}
	w.Flush()

	all, err := h.Versions(1 << 30)
	if err != nil {
		return err
	}
	if len(versions) < len(all) {
		fmt.Fprintf(out, "\nThe latest %d of %d versions — -n shows more. A run that changes nothing records nothing.\n", len(versions), len(all))
		return nil
	}
	fmt.Fprintf(out, "\n%d version(s). A run that changes nothing records nothing.\n", len(versions))
	return nil
}
