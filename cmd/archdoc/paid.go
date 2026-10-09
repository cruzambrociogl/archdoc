package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/semantic"
)

// The two commands that send anything anywhere. Each says what it will send and what that should
// cost before sending, asks unless told not to, and can show the requests instead (--dry-run) —
// built by the same functions the real requests are, so what is shown is what would go.

type labelRun struct {
	model   string
	consent consent
}

type explainRun struct {
	only    string
	limit   int
	consent consent
}

// consent is how a paid run may go ahead: asked at a terminal, or told to with --yes; never with no
// one to ask. dry prints the requests and stops.
type consent struct{ yes, dry bool }

const (
	labelSends   = "names, kinds, technologies and relationships of the containers — no path, no code"
	explainSends = "each component's name, file paths, counts and the code's own route summaries — no code"
)

// What a request costs is its tokens; before it is sent, only its size is known. Measured on
// Immich's eleven runs: a request carries about 2.5 bytes per input token (run 11: 270,139 bytes,
// 107,916 tokens); an explanation's answer is about 250 tokens (22,358 over 91 requests); a label
// run answers about one token for every five bytes it was sent (run 10: 1,290 for 7,055).
const (
	bytesPerToken = 2.5
	explainAnswer = 250
)

func labelAnswer(p semantic.Planned) int64 { return int64(p.Bytes()) / 5 }

// dollars is an estimate in words: a figure to the cent, never "$0.00".
func dollars(usd float64) string {
	if usd < 0.01 {
		return "under a cent"
	}
	return fmt.Sprintf("about $%.2f", usd)
}

// ask says what a run will send and what it should cost, and goes ahead only with consent. A dry
// run prints every request instead and stops; so does a "no". Nothing has been written by then.
func (c consent) ask(e env, what, model, sends string, plans []semantic.Planned, answerTokens int64) error {
	bytes := 0
	for _, p := range plans {
		bytes += p.Bytes()
	}
	cost := "cost unknown for this model"
	if usd, ok := semantic.Cost(model, int64(float64(bytes)/bytesPerToken), answerTokens); ok {
		cost = dollars(usd) + ", estimated from its size"
	}
	requests := fmt.Sprintf("%d request(s)", len(plans))
	if c.dry {
		for i, p := range plans {
			fmt.Fprintf(e.out, "── request %d of %d · %s · %d bytes ──\n[system]\n%s\n\n[user]\n%s\n\n", i+1, len(plans), p.Element, p.Bytes(), p.System, p.Prompt)
		}
		fmt.Fprintf(e.out, "dry run — %s would go to %s, %d bytes, %s. Nothing sent, nothing written.\n", requests, model, bytes, cost)
		return errStopped
	}
	if strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")) == "" {
		return fmt.Errorf("%s sends to Anthropic, and ANTHROPIC_API_KEY is not set — set it, or see what would be sent with --dry-run", what)
	}
	fmt.Fprintf(e.out, "%s: %s to %s, %d bytes — %s.\n", what, requests, model, bytes, cost)
	fmt.Fprintf(e.out, "Sends %s. A corrected answer is asked for again and costs more.\n", sends)
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

func label(e env, args []string) error {
	fs := flags("label")
	model := fs.String("model", semantic.Model, "the `model` to ask")
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
	return document(e, root, options{verbose: *verbose, label: &labelRun{model: *model, consent: consent{yes: *yes, dry: *dry}}})
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
	return document(e, root, options{verbose: *verbose, explain: &explainRun{only: *only, limit: *limit, consent: consent{yes: *yes, dry: *dry}}})
}
