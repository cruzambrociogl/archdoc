package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/cruzambrociogl/archdoc"
	"github.com/cruzambrociogl/archdoc/server"
)

// openHere is a bare `archdoc` (decisions, 18 Nov 2026): document the project it is run in — free,
// offline, writing only .archdoc/ unless the documents are already there — and open it in this
// machine's one archdoc, starting it if none is running. Run in a folder of repositories, it
// documents each and opens the library; in the home folder, only the library.
func openHere(e env, dir string) error {
	where, err := archdoc.Locate(dir)
	if err != nil {
		return err
	}
	page := "/"
	switch where.Kind {
	case "repository":
		if err := documentForTheApp(e, where.Root); err != nil {
			return err
		}
		page = server.Address(where.Root)
	case "folder":
		if err := documentFolder(e, where); err != nil {
			return err
		}
	default:
		fmt.Fprintf(e.out, "%s is not a project: run archdoc inside one. Opening the library.\n", where.Root)
	}
	return openInMachine(e, page)
}

// documentForTheApp is generate as the app needs it: the model and its history, and the documents
// only where they were written before.
func documentForTheApp(e env, root string) error {
	_, err := archdoc.Generate(context.Background(), root, archdoc.GenerateOptions{Log: e.out, History: true, DocsIfThere: true, By: "archdoc"})
	if err != nil {
		return err
	}
	if e.remember {
		archdoc.Remember(root) // the library's list; a failure to keep it is no failure of this command
	}
	return nil
}

// documentFolder documents each repository in a folder of them, and remembers the folder: the
// library shows it as one system. A repository with nothing to document is said and passed over.
func documentFolder(e env, where archdoc.Where) error {
	fmt.Fprintf(e.out, "%d repositories in %s\n", len(where.Repos), where.Root)
	done := 0
	for _, repo := range where.Repos {
		fmt.Fprintf(e.out, "\n— %s\n", filepath.Base(repo))
		_, err := archdoc.Generate(context.Background(), repo, archdoc.GenerateOptions{Log: e.out, History: true, DocsIfThere: true, By: "archdoc"})
		if err != nil {
			fmt.Fprintf(e.out, "passed over: %v\n", err)
			continue
		}
		done++
	}
	if done == 0 {
		return fmt.Errorf("nothing to document in any repository in %s", where.Root)
	}
	if e.remember {
		archdoc.Remember(where.Root)
	}
	fmt.Fprintln(e.out)
	return nil
}

// openInMachine opens page in the archdoc already serving this machine, or starts it here and
// serves until Ctrl-C — or until `archdoc restart` takes over from another terminal.
func openInMachine(e env, page string) error {
	if !e.machine {
		// Tests and scripts: the documentation is the result; nothing is served.
		return nil
	}
	fmt.Fprintln(e.out)
	if m, ok := server.Running(); ok {
		url := m.URL() + page
		fmt.Fprintf(e.out, "archdoc is already running on this machine — open %s\n", url)
		if this := archdoc.Build().String(); m.Build != this {
			fmt.Fprintf(e.out, "it is %s, and this is %s — 'archdoc restart' runs this one instead\n", m.Build, this)
		}
		openBrowser(e, url)
		return nil
	}
	return serveMachine(e, server.MachinePort, func(base string) {
		url := base + page
		if page == "/" {
			fmt.Fprintf(e.out, "archdoc is open at %s — the library of every project on this machine\n", url)
		} else {
			fmt.Fprintf(e.out, "archdoc is open at %s\nthe library of every project on this machine is %s\n", url, base)
		}
		fmt.Fprintln(e.out, "Ctrl-C stops it. A bare archdoc in another project opens that one here too.")
		openBrowser(e, url)
	})
}

// serveMachine serves the library in the foreground until Ctrl-C, or until another archdoc takes over.
func serveMachine(e env, port int, ready func(base string)) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	err := server.ServeMachine(ctx, port, ready)
	if errors.Is(err, server.ErrHandedOver) {
		fmt.Fprintln(e.out, "archdoc restarted from another terminal — it runs there now")
		return nil
	}
	return err
}

// restart stops this machine's archdoc and starts it again here, on the same port, reading this
// shell's environment: what to run after setting ANTHROPIC_API_KEY, or after updating archdoc.
func restart(e env, args []string) error {
	fs := flags("restart")
	positional, err := parse(e, fs, args, nil)
	if err != nil {
		return err
	}
	if len(positional) > 0 {
		return usagef("restart takes no path: it restarts this machine's archdoc, whichever projects it has open")
	}
	if !e.machine {
		return nil // tests and scripts never stop the archdoc a person is using
	}
	port, was := server.MachinePort, false
	if m, ok := server.Running(); ok {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := m.Stop(ctx); err != nil {
			return err
		}
		port, was = m.Port, true
		fmt.Fprintf(e.out, "stopped the archdoc running since %s\n", m.Since.Local().Format("15:04"))
	} else {
		fmt.Fprintln(e.out, "no archdoc was running on this machine — starting one")
	}
	if strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")) != "" {
		fmt.Fprintln(e.out, "ANTHROPIC_API_KEY is set here: Enrich with AI can use it")
	} else {
		fmt.Fprintln(e.out, "ANTHROPIC_API_KEY isn’t set here: everything but Enrich with AI works without it")
	}
	return serveMachine(e, port, func(base string) {
		fmt.Fprintf(e.out, "archdoc is open at %s — Ctrl-C stops it\n", base)
		if was {
			fmt.Fprintln(e.out, "pages already open in a browser go on working; reload one to see what changed")
			return
		}
		openBrowser(e, base)
	})
}

// libraryCommand opens the library: every project this machine has documented, in this machine's
// archdoc, starting it if none is running.
func libraryCommand(e env, args []string) error {
	fs := flags("library")
	if _, err := parse(e, fs, args, nil); err != nil {
		return err
	}
	return openInMachine(e, "/")
}

func openBrowser(e env, url string) {
	if err := browse(url); err != nil {
		fmt.Fprintf(e.out, "could not open a browser (%v) — open %s yourself\n", err, url)
	}
}
