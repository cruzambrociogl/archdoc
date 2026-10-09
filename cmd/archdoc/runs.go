package main

import (
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
func runs(e env, args []string) error {
	fs := flags("runs")
	show := fs.Int("show", 0, "print exactly what `run` sent, and what came back")
	limit := fs.Int("n", 20, "list the latest `count` runs")
	asJSON := fs.Bool("json", false, "print the runs as JSON, without what they sent")

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
		fmt.Fprintln(out, "No run has used the network. Only 'archdoc label' and 'archdoc explain' send anything.")
		return nil
	}
	defer h.Close()

	if *show > 0 {
		return showRun(h, int64(*show), out)
	}

	all, err := h.Runs(1 << 30)
	if err != nil {
		return err
	}
	list := all
	if *limit > 0 && len(list) > *limit {
		list = list[:*limit]
	}
	if *asJSON {
		rows := []map[string]any{}
		for _, r := range list {
			rows = append(rows, map[string]any{"run": r.ID, "started": r.StartedAt, "mode": r.EgressMode, "model": r.Model,
				"requests": r.Requests, "bytes_sent": r.BytesSent, "tokens_in": r.TokensIn, "tokens_out": r.TokensOut,
				"cost_usd": r.CostUSD, "cost_known": r.CostKnown, "status": r.Status})
		}
		return printJSON(out, rows)
	}
	if len(list) == 0 {
		fmt.Fprintln(out, "No run has used the network. Only 'archdoc label' and 'archdoc explain' send anything.")
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

	total, unknown := spent(all)
	fmt.Fprintf(out, "\n%d run(s), $%.2f in all", len(all), total)
	if unknown > 0 {
		fmt.Fprintf(out, ", and %d whose cost is unknown", unknown)
	}
	if len(list) < len(all) {
		fmt.Fprintf(out, " — the latest %d listed; -n shows more", len(list))
	}
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
		if ex.Response != "" {
			fmt.Fprintf(out, "\n── answer %d of %d · %d bytes ──\n", i+1, len(r.Exchanges), len(ex.Response))
			fmt.Fprintln(out, ex.Response)
		}
	}
	return nil
}

// spent is what the runs cost, and how many of them cost an unknown amount.
func spent(rs []store.Run) (float64, int) {
	total, unknown := 0.0, 0
	for _, r := range rs {
		if r.CostKnown {
			total += r.CostUSD
		} else {
			unknown++
		}
	}
	return total, unknown
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
