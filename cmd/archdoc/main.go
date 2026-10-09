// Command archdoc reads a repository and produces the architecture documentation it should
// have had — accurate, traceable to the code, and regenerable as the system changes.
//
// This is the CLI: argument parsing over the engine, and nothing else. All behaviour lives in
// internal/, so the engine stays testable with no frontend present. See docs/stack-decision.md
// §3.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
	"github.com/cruzambrociogl/archdoc/internal/extract"
)

func main() {
	err := runIn(env{out: os.Stdout, in: os.Stdin, terminal: interactive(os.Stdin)}, os.Args[1:])
	code := exitCode(err)
	if code != 0 {
		fmt.Fprintln(os.Stderr, "archdoc:", err)
	}
	os.Exit(code)
}

// run is archdoc with no one at the keyboard: what tests and scripts get. A paid run is never
// asked about there; it needs --yes.
func run(args []string, out io.Writer) error {
	err := runIn(env{out: out}, args)
	if exitCode(err) == 0 {
		return nil
	}
	return err
}

func runIn(e env, args []string) error {
	if len(args) == 0 {
		usage(e.out)
		return nil
	}
	switch args[0] {
	case "-h", "--help":
		usage(e.out)
		return nil
	case "-v", "--version":
		return version(e)
	}
	c, ok := lookupCommand(args[0])
	if !ok {
		return unknownCommand(args[0])
	}
	return c.run(e, args[1:])
}

// interactive reports whether f is a person at a terminal, who can be asked before money is spent:
// a character device, and not /dev/null, which is one too.
func interactive(f *os.File) bool {
	info, err := f.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	if null, err := os.Stat(os.DevNull); err == nil && os.SameFile(info, null) {
		return false
	}
	return true
}

func version(e env) error {
	fmt.Fprintln(e.out, archdoc.Build())
	return nil
}

// scan is generate without the writing: what archdoc would document, found exactly the way generate
// finds it — rules applied, no model asked — and printed.
func scan(e env, args []string) error {
	fs := flags("scan")
	asJSON := fs.Bool("json", false, "print what was found as JSON")
	raw := fs.Bool("facts", false, "print everything extraction read, as JSON: the FactSet")
	considered := fs.Bool("considered", false, "list every file discovery looked at, and why it was or was not used")
	positional, err := parse(e, fs, args, map[string]string{"explain": "renamed: --considered lists the files discovery looked at"})
	if err != nil {
		return err
	}
	root, err := readPath(fs, positional)
	if err != nil {
		return err
	}

	facts, err := extract.Scan(root)
	if err != nil {
		return err
	}
	if *raw {
		return printJSON(e.out, facts)
	}
	if facts.Source == "" && !hasContainerApp(facts.Apps) {
		if *asJSON {
			return printJSON(e.out, map[string]any{"root": facts.Root, "documentable": false})
		}
		fmt.Fprintf(e.out, "Nothing to document in %s: no Compose file, no application manifest, and no Python,\n", facts.Root)
		fmt.Fprintf(e.out, "JavaScript or HTML files.\n")
		listConsidered(e.out, facts)
		return nil
	}
	m, err := corrected(facts)
	if err != nil {
		return err
	}
	if *asJSON {
		apps := []map[string]string{}
		for _, a := range facts.Apps {
			apps = append(apps, map[string]string{"name": a.Name, "dir": a.Dir, "role": string(a.Role), "framework": a.Framework, "why": a.Why})
		}
		return printJSON(e.out, map[string]any{"root": facts.Root, "documentable": true, "name": m.Name, "source": m.Source,
			"applications": apps, "counts": countModel(m)})
	}

	fmt.Fprintf(e.out, "%s — %s\n\n", m.Name, facts.Root)
	if len(facts.Apps) > 0 {
		w := tabwriter.NewWriter(e.out, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "APPLICATION\tROLE\tBUILT ON\tWHERE")
		for _, a := range facts.Apps {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.Name, a.Role, orDash(a.Framework), a.Dir)
		}
		w.Flush()
		fmt.Fprintln(e.out)
	}
	if facts.Source != "" {
		fmt.Fprintf(e.out, "%d services in %s\n", len(facts.Services), facts.Source)
	}
	countModel(m).print(e.out, m.Source)
	if *considered {
		listConsidered(e.out, facts)
	}
	fmt.Fprintf(e.out, "\nNothing written. 'archdoc generate %s' writes it.\n", root)
	return nil
}

func listConsidered(out io.Writer, f *archdoc.FactSet) {
	if len(f.Considered) == 0 {
		return
	}
	fmt.Fprintf(out, "\n%d Compose file(s) considered:\n", len(f.Considered))
	for _, c := range f.Considered {
		mark := " "
		if c.Chosen {
			mark = "→"
		}
		fmt.Fprintf(out, "  %s %-52s %s\n", mark, c.File, c.Reason)
	}
}

func printJSON(out io.Writer, v any) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// partitionArgs splits arguments into flags and positional values, so flags may appear before
// or after the path.
//
// A flag that takes a value keeps the word after it: `history -n 3 ./repo` means n=3 and the
// path is ./repo. The first version treated every flag as on/off, so the 3 became the path and
// the flag package complained that -n had no value. Boolean flags still never consume a word.
func partitionArgs(fs *flag.FlagSet, args []string) (flags, positional []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if len(a) < 2 || a[0] != '-' {
			positional = append(positional, a)
			continue
		}

		flags = append(flags, a)
		if strings.Contains(a, "=") {
			continue // -n=3 carries its own value
		}

		if f := fs.Lookup(strings.TrimLeft(a, "-")); f != nil && !isBool(f) && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return flags, positional
}

func isBool(f *flag.Flag) bool {
	b, ok := f.Value.(interface{ IsBoolFlag() bool })
	return ok && b.IsBoolFlag()
}
