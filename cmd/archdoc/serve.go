package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/cruzambrociogl/archdoc"
	"github.com/cruzambrociogl/archdoc/server"
)

// serveCommand starts the web app for one repository (SUR-07), or for a folder of them as one
// system (platform-surface §4.6). The listening happens in package server, so this package still
// never imports an HTTP client — the CI network rule stays about imports, not exceptions.
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

	return server.ListenAndServe(ctx, root, *port, func(url string) {
		if e.remember {
			archdoc.Opened(root) // Home's list: opened now, at these versions
		}
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
	case "linux":
		// WSL opens Windows' browser; a server over SSH or a container has nothing to open one on.
		if os.Getenv("WSL_DISTRO_NAME") != "" {
			return exec.Command("cmd.exe", "/c", "start", "", url).Start() // Windows' browser, from WSL
		}
		if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
			return errors.New("no display here")
		}
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
	return exec.Command("xdg-open", url).Start()
}
