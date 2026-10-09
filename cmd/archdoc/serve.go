package main

import (
	"context"
	"fmt"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/cruzambrociogl/archdoc/internal/serve"
)

// serveCommand starts the web app for one repository (SUR-07). The listening happens in
// internal/serve, so this package still never imports an HTTP client — the CI network rule
// stays about imports, not exceptions.
func serveCommand(e env, args []string) error {
	fs := flags("serve")
	port := fs.Int("port", 7474, "local `port` to serve on; only this machine can connect")
	open := fs.Bool("open", false, "open the app in the browser once it is serving")

	positional, err := parse(e, fs, args, nil)
	if err != nil {
		return err
	}
	root, err := readPath(fs, positional)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return serve.ListenAndServe(ctx, root, *port, func(url string) {
		fmt.Fprintf(e.out, "archdoc serving %s at %s — this machine only. Ctrl-C to stop.\n", root, url)
		if *open {
			if err := browse(url); err != nil {
				fmt.Fprintf(e.out, "could not open a browser (%v) — open %s yourself\n", err, url)
			}
		}
	})
}

// browse hands a local address to the system's own opener. Nothing leaves the machine: the page is
// served from here.
func browse(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
	return exec.Command("xdg-open", url).Start()
}
