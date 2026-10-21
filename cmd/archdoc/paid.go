package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/cruzambrociogl/archdoc"
)

// The two commands that send anything anywhere. Each says what it will send and what that should
// cost before sending, asks unless told not to, and can show the requests instead (--dry-run) —
// built by the same functions the real requests are, so what is shown is what would go.

// consent is how a paid run may go ahead: asked at a terminal, or told to with --yes; never with no
// one to ask. dry prints the requests and stops.
type consent struct{ yes, dry bool }

// dollars is an estimate in words: a figure to the cent, never "$0.00".
func dollars(usd float64) string {
	if usd < 0.01 {
		return "under a cent"
	}
	return fmt.Sprintf("about $%.2f", usd)
}

// ask says what a run will send and what it should cost, and goes ahead only with consent. A dry
// run prints every request instead and stops; so does a "no". Nothing has been written by then.
func (c consent) ask(e env) archdoc.Consent {
	return func(r archdoc.Request) error {
		cost := "cost unknown for this model"
		if r.EstimateKnown {
			cost = dollars(r.Estimate) + ", estimated from its size"
		}
		requests := fmt.Sprintf("%d request(s)", len(r.Plans))
		if c.dry {
			for i, p := range r.Plans {
				fmt.Fprintf(e.out, "── request %d of %d · %s · %d bytes ──\n[system]\n%s\n\n[user]\n%s\n\n", i+1, len(r.Plans), p.Element, p.Bytes(), p.System, p.Prompt)
			}
			fmt.Fprintf(e.out, "dry run — %s would go to %s, %d bytes, %s. Nothing sent, nothing written.\n", requests, r.Model, r.Bytes, cost)
			return errStopped
		}
		if strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")) == "" {
			return fmt.Errorf("%s sends to Anthropic, and ANTHROPIC_API_KEY is not set — set it, or see what would be sent with --dry-run", r.What)
		}
		fmt.Fprintf(e.out, "%s: %s to %s, %d bytes — %s.\n", r.What, requests, r.Model, r.Bytes, cost)
		fmt.Fprintf(e.out, "Sends %s. A corrected answer is asked for again and costs more.\n", r.Sends)
		if c.yes {
			return nil
		}
		if !e.terminal || e.in == nil {
			return usagef("not sent: there is no one to ask here — run again with --yes to send, or --dry-run to see it first")
		}
		fmt.Fprint(e.out, "Send? [y/N] ")
		line, err := bufio.NewReader(e.in).ReadString('\n')
		if a := strings.ToLower(strings.TrimSpace(line)); a == "y" || a == "yes" {
			return nil
		}
		if err != nil {
			fmt.Fprintln(e.out) // the answer ended without a newline of its own
		}
		fmt.Fprintln(e.out, "Nothing sent, nothing written.")
		return errStopped
	}
}

func label(e env, args []string) error {
	fs := flags("label")
	model := fs.String("model", archdoc.LabelModel, "the `model` to ask")
	yes := fs.Bool("yes", false, "send without asking — for scripts; what it costs is still printed")
	dry := fs.Bool("dry-run", false, "print exactly what would be sent, and stop: nothing sent, nothing written")
	verbose := fs.Bool("verbose", false, "list every file written")
	positional, err := parse(e, fs, args, map[string]string{"label-model": "renamed: --model"})
	if err != nil {
		return err
	}
	root, err := writePath(fs, positional)
	if err != nil {
		return err
	}
	return document(e, root, archdoc.GenerateOptions{Verbose: *verbose,
		Label: &archdoc.LabelOptions{Model: *model, Consent: consent{yes: *yes, dry: *dry}.ask(e)}})
}

func explain(e env, args []string) error {
	fs := flags("explain")
	only := fs.String("only", "", "ask only about components whose id contains `text`")
	limit := fs.Int("limit", 0, "ask about at most `n` components")
	yes := fs.Bool("yes", false, "send without asking — for scripts; what it costs is still printed")
	dry := fs.Bool("dry-run", false, "print exactly what would be sent, and stop: nothing sent, nothing written")
	verbose := fs.Bool("verbose", false, "list every file written")
	positional, err := parse(e, fs, args, map[string]string{
		"explain-only":  "renamed: --only",
		"explain-limit": "renamed: --limit",
	})
	if err != nil {
		return err
	}
	root, err := writePath(fs, positional)
	if err != nil {
		return err
	}
	return document(e, root, archdoc.GenerateOptions{Verbose: *verbose,
		Explain: &archdoc.ExplainOptions{Only: *only, Limit: *limit, DryRun: *dry, Consent: consent{yes: *yes, dry: *dry}.ask(e)}})
}
