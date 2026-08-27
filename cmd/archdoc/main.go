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
	"text/tabwriter"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
	"github.com/cruzambrociogl/archdoc/internal/extract"
)

const usage = `archdoc — architecture documentation generated from a repository's configuration

usage:
  archdoc <command> [flags]

commands:
  scan <path>    extract services from a repository's configuration
  version        print build information

flags for scan:
  --json         emit the FactSet as JSON instead of a table
  --explain      show every file discovery considered, and why
`

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "archdoc:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		fmt.Fprint(out, usage)
		return nil
	}

	switch cmd := args[0]; cmd {
	case "scan":
		return scan(args[1:], out)

	case "version":
		fmt.Fprintln(out, archdoc.Build())
		return nil

	case "help", "-h", "--help":
		fmt.Fprint(out, usage)
		return nil

	default:
		return fmt.Errorf("unknown command %q — run 'archdoc help'", cmd)
	}
}

func scan(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	asJSON := fs.Bool("json", false, "emit the FactSet as JSON")
	explain := fs.Bool("explain", false, "show every file considered")

	// Go's flag package stops parsing at the first non-flag argument, so "scan ./repo --json"
	// would silently ignore the flag. Separating them first means flags work on either side of
	// the path, which is what anyone typing the command will expect.
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

	if *asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(facts)
	}

	report(out, facts, *explain)
	return nil
}

func report(out io.Writer, f *archdoc.FactSet, explain bool) {
	if f.Source == "" {
		fmt.Fprintf(out, "No deployable Compose file found in %s\n", f.Root)
		if len(f.Considered) > 0 {
			fmt.Fprintf(out, "\n%d file(s) considered:\n", len(f.Considered))
			for _, c := range f.Considered {
				fmt.Fprintf(out, "  %-52s %s\n", c.File, c.Reason)
			}
		}
		return
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	for _, s := range f.Services {
		image := s.Image
		if image == "" {
			image = "(built locally)"
		}

		// The provenance column is the point of the whole exercise: every service names the
		// file and line that proves it exists.
		fmt.Fprintf(w, "%s\t%s\t%s\n", s.Name, image, s.Prov)
	}
	w.Flush()

	deployable := 0
	for _, c := range f.Considered {
		if c.Chosen || len(c.Reason) > 10 && c.Reason[:10] == "deployable" {
			deployable++
		}
	}

	fmt.Fprintf(out, "\n%d services from %s · %d compose file(s) considered, %d deployable\n",
		len(f.Services), f.Source, len(f.Considered), deployable)

	if explain {
		fmt.Fprintln(out, "\nDiscovery:")
		for _, c := range f.Considered {
			mark := " "
			if c.Chosen {
				mark = "→"
			}
			fmt.Fprintf(out, "  %s %-52s %s\n", mark, c.File, c.Reason)
		}
	}
}

// partitionArgs splits arguments into flags and positional values, so flags may appear before
// or after the path.
func partitionArgs(args []string) (flags, positional []string) {
	for _, a := range args {
		if len(a) > 1 && a[0] == '-' {
			flags = append(flags, a)
			continue
		}
		positional = append(positional, a)
	}
	return flags, positional
}
