package main

import (
	"flag"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/cruzambrociogl/archdoc/internal/extract"
	"github.com/cruzambrociogl/archdoc/internal/store"
)

// runs is SUR-13 and SUR-14: every run that used the network, what it cost, and — with --show —
// exactly what it sent.
//
// The payload is printed as it left the machine, not reformatted. A pretty-printed version would
// be easier to read and would no longer be the thing that was sent, which is the one property
// this command exists to have.
func runs(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("runs", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	show := fs.Int("show", 0, "print exactly what one run sent")
	limit := fs.Int("n", 20, "how many runs to list")

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

	h, err := store.Open(facts.Root)
	if err != nil {
		return err
	}
	defer h.Close()

	if *show > 0 {
		return showRun(h, int64(*show), out)
	}

	list, err := h.Runs(*limit)
	if err != nil {
		return err
	}
	if len(list) == 0 {
		fmt.Fprintln(out, "No run has used the network. Without --label, archdoc sends nothing.")
		return nil
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "RUN\tWHEN\tMODE\tREQUESTS\tBYTES SENT\tTOKENS IN/OUT\tCOST\tSTATUS")
	for _, r := range list {
		fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%d\t%d / %d\t%s\t%s\n",
			r.ID, r.StartedAt.Format("2006-01-02 15:04"), r.EgressMode, r.Requests, r.BytesSent,
			r.TokensIn, r.TokensOut, costText(r), short(r.Status))
	}
	w.Flush()

	fmt.Fprintf(out, "\n'archdoc runs %s --show <run>' prints exactly what a run sent.\n", root)
	return nil
}

func showRun(h *store.Store, id int64, out io.Writer) error {
	r, err := h.Run(id)
	if err != nil {
		return err
	}
	if r == nil {
		return fmt.Errorf("no run %d", id)
	}

	fmt.Fprintf(out, "run %d · %s · %s · model %s\n", r.ID, r.StartedAt.Format("2006-01-02 15:04:05 MST"), r.EgressMode, r.Model)
	fmt.Fprintf(out, "%d request(s), %d bytes sent · %d tokens in, %d out · %s\n", r.Requests, r.BytesSent, r.TokensIn, r.TokensOut, costText(*r))
	fmt.Fprintf(out, "status: %s\n", r.Status)

	for i, ex := range r.Exchanges {
		fmt.Fprintf(out, "\n── request %d of %d · %s %s · %d bytes · HTTP %d ──\n", i+1, len(r.Exchanges), ex.Method, ex.URL, len(ex.Body), ex.Status)
		fmt.Fprintln(out, ex.Body)
	}
	return nil
}

func costText(r store.Run) string {
	if !r.CostKnown {
		return "unknown"
	}
	return fmt.Sprintf("$%.4f", r.CostUSD)
}

func short(s string) string {
	if len(s) > 40 {
		return s[:39] + "…"
	}
	return s
}
