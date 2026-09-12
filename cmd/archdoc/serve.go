package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os/signal"
	"syscall"

	"github.com/cruzambrociogl/archdoc/internal/serve"
)

// serveCommand starts the web app for one repository (SUR-07). The listening happens in
// internal/serve, so this package still never imports an HTTP client — the CI network rule
// stays about imports, not exceptions.
func serveCommand(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	port := fs.Int("port", 7474, "local port to serve on")

	flags, positional := partitionArgs(fs, args)
	if err := fs.Parse(flags); err != nil {
		return err
	}
	root := "."
	if len(positional) > 0 {
		root = positional[0]
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return serve.ListenAndServe(ctx, root, *port, func(url string) {
		fmt.Fprintf(out, "archdoc serving %s at %s — this machine only. Ctrl-C to stop.\n", root, url)
	})
}
