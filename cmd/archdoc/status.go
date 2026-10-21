package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/cruzambrociogl/archdoc"
)

// status answers "is the documentation up to date, and what is open?" — the question the app's
// top bar asks. The committed model is compared with the code as it is now, read again offline, by
// structure: a description reworded by a model is not the code changing. Writes nothing.
func status(e env, args []string) error {
	fs := flags("status")
	asJSON := fs.Bool("json", false, "print the status as JSON")
	positional, err := parse(e, fs, args, nil)
	if err != nil {
		return err
	}
	root, err := readPath(fs, positional)
	if err != nil {
		return err
	}

	r, err := archdoc.Status(context.Background(), root)
	if err != nil {
		return err
	}
	if !r.Documented {
		if *asJSON {
			return printJSON(e.out, r)
		}
		fmt.Fprintf(e.out, "%s is not documented yet — 'archdoc scan %s' previews it, 'archdoc generate %s' writes it.\n", r.Root, root, root)
		return nil
	}
	if *asJSON {
		return printJSON(e.out, r)
	}
	out := e.out
	line := func(head, format string, a ...any) { fmt.Fprintf(out, "%-14s %s\n", head, fmt.Sprintf(format, a...)) }
	fmt.Fprintf(out, "%s — %s\n\n", r.Name, r.Root)
	switch {
	case r.Version > 0:
		line("documented", "version %d, recorded %s%s", r.Version, r.Recorded, atCommit(r.Commit))
	default:
		line("documented", "%s is committed; this clone has no local history — 'archdoc generate %s' records it", archdoc.ModelFile, root)
	}
	switch {
	case r.Changes < 0:
		line("the code now", "could not be read: %s", firstLine(r.CodeError))
	case r.Changes == 0:
		line("the code now", "%sthe architecture is as documented", headAt(r.Head))
	default:
		line("the code now", "%s%d change(s) to the structure since — 'archdoc generate %s' records them", headAt(r.Head), r.Changes, root)
		if r.Commit != "" {
			line("", "'archdoc diff %s %s' says what they are", root, r.Commit)
		}
	}
	if r.Labelled > 0 {
		line("descriptions", "%d written by a model, marked as such — 'archdoc label %s' asks again", r.Labelled, root)
	} else {
		line("descriptions", "none written by a model — 'archdoc label %s' asks for them", root)
	}
	if r.Components > 0 {
		x := fmt.Sprintf("%d of %d components", r.Explained, r.Components)
		if r.Stale > 0 {
			x += fmt.Sprintf(", %d of them about facts that have since changed", r.Stale)
		}
		if r.WouldAsk > 0 {
			x += fmt.Sprintf(" — 'archdoc explain %s' would ask about %d", root, r.WouldAsk)
			if r.AskEstimateKnown {
				x += ", " + dollars(r.AskEstimate)
			}
		}
		line("explanations", "%s", x)
	}
	if r.Gaps > 0 {
		line("gaps", "%d — 'archdoc generate %s --gaps' lists them", r.Gaps, root)
	} else {
		line("gaps", "none")
	}
	switch {
	case r.Runs == 0:
		line("network", "never used")
	default:
		x := fmt.Sprintf("%d run(s), $%.2f in all", r.Runs, r.Spent)
		if r.UnknownCosts > 0 {
			x += fmt.Sprintf(" and %d of unknown cost", r.UnknownCosts)
		}
		line("network", "%s — 'archdoc runs %s' lists them", x, root)
	}
	return nil
}

func atCommit(c string) string {
	if c == "" {
		return ""
	}
	if len(c) > 7 {
		c = c[:7]
	}
	return " at commit " + c
}

func headAt(c string) string {
	if c == "" {
		return ""
	}
	return strings.TrimSpace(atCommit(c)) + " — "
}

func firstLine(s string) string {
	first, _, _ := strings.Cut(s, "\n")
	return first
}
