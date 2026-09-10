package main

import (
	"flag"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/cruzambrociogl/archdoc/internal/extract"
	"github.com/cruzambrociogl/archdoc/internal/store"
)

// history lists what archdoc has recorded for a repository.
//
// The point is not the list. It is that a run which changes nothing adds nothing — so the number
// of versions is the number of times the architecture actually moved, not the number of times
// somebody ran the tool.
func history(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("history", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	limit := fs.Int("n", 20, "how many versions to list")

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

	h, err := store.Open(facts.Root)
	if err != nil {
		return err
	}
	defer h.Close()

	versions, err := h.Versions(*limit)
	if err != nil {
		return err
	}

	if len(versions) == 0 {
		fmt.Fprintf(out, "No history yet for %s — run 'archdoc generate' first.\n", facts.Root)
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

	fmt.Fprintf(out, "\n%d version(s). A run that changes nothing records nothing.\n", len(versions))
	return nil
}
