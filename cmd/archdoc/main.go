// Command archdoc reads a repository and produces the architecture documentation it should
// have had — accurate, traceable to the code, and regenerable as the system changes.
//
// This is the CLI: argument parsing over the engine, and nothing else. All behaviour lives in
// internal/, so the engine stays testable with no frontend present. See docs/stack-decision.md
// §3.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

const usage = `archdoc — architecture documentation generated from a repository's configuration

usage:
  archdoc <command> [flags]

commands:
  version    print build information

Most commands are not implemented yet. See PROGRESS.md for what exists.
`

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "archdoc:", err)
		os.Exit(1)
	}
}

// run holds the CLI logic so it is testable without spawning a process, and writes to out
// rather than os.Stdout for the same reason.
func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		fmt.Fprint(out, usage)
		return nil
	}

	switch cmd := args[0]; cmd {
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
