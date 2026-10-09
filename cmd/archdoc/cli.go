package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

// env is where a command reads and writes: its result to out, and — for a run that costs money —
// the answer to "send?" from in, when in is a person at a terminal.
type env struct {
	out      io.Writer
	in       io.Reader
	terminal bool
}

// command is one thing archdoc does. The table below is the only list of them: the usage text, a
// command's own help and the suggestion for a mistyped one are all read from it.
type command struct {
	name    string
	args    string // as written in usage: "<path>", "<path> <commit> [<commit>]"
	summary string // one line, for the list
	about   string // more, for 'archdoc help <command>'
	group   string
	run     func(e env, args []string) error
}

// The groups commands are listed in: what they may do is the first thing a person needs to know.
const (
	groupWrite = "Write the documentation into the repository — offline, free"
	groupRead  = "Read — write nothing, send nothing"
	groupPaid  = "Ask Claude — sends to Anthropic and costs money; says what first, and asks"
)

var groups = []string{groupWrite, groupRead, groupPaid}

var commands []command

// Set in init: the help command reads the table it is in.
func init() {
	commands = []command{
		{"init", "<path>", "create .archdoc/ with a rules.yaml to correct the model in",
			"Creates archdoc's own directory in the repository: a rules.yaml whose every line is a comment until\n" +
				"you write a correction, and the .gitignore that keeps the local history out of git. Nothing that\n" +
				"exists is overwritten. generate does not need it; it is where corrections go.",
			groupWrite, initCommand},
		{"generate", "<path>", "read the repository and write its architecture documentation",
			"Reads the code and configuration, builds the model, and writes docs/architecture/ and .archdoc/.\n" +
				"Never uses the network: descriptions and explanations Claude wrote before are reused while what they\n" +
				"describe is unchanged. A run that changes nothing records no new version.",
			groupWrite, generate},
		{"export", "<path> --site", "build the published site: the web app as static files, for a team",
			"Builds .archdoc/site from the committed record — the app, read-only, with the latest version's data —\n" +
				"for any static host. Never deploys; never calls a model.",
			groupWrite, export},
		{"status", "<path>", "is the documentation up to date with the code, and what is open",
			"The latest version and the commit it was read at; whether the code has changed the architecture since\n" +
				"(the code is read again, offline); stale explanations, open gaps, and what the network has cost.",
			groupRead, status},
		{"scan", "<path>", "preview what generate would find, writing nothing",
			"Reads the repository exactly as generate does — applications, components, routes, tables, flows — and\n" +
				"prints what it found. Nothing is written and nothing is recorded.",
			groupRead, scan},
		{"show", "<path> <element | route | file>", "one element, route or file, with every line that proves it",
			"A container, component or table by name or id; a route as \"POST /api/assets\", with its flow as\n" +
				"numbered, cited steps; or a file, with the component it belongs to. Read from .archdoc/model.json,\n" +
				"so generate first. A name that matches several things lists them.",
			groupRead, show},
		{"diff", "<path> <commit> [<commit>]", "what changed in the architecture between two commits, or since one",
			"Reads the repository at each commit — or at one, and as the files are now — and says what changed:\n" +
				"elements, relationships, ways in, renames and moves. Needs git. Writes nothing.",
			groupRead, diff},
		{"history", "<path>", "every version archdoc has recorded", "", groupRead, history},
		{"runs", "<path>", "every run that used the network, what it sent and what it cost", "", groupRead, runs},
		{"serve", "<path>", "open the web app for a repository archdoc has documented",
			"Serves the app on this machine only. Ctrl-C stops it.", groupRead, serveCommand},
		{"label", "<path>", "ask Claude for the system's descriptions and relationship labels",
			"One request with the names, kinds, technologies and relationships of the containers — never a path or\n" +
				"a line of code. Says what it will send and what it should cost, and asks before sending; --dry-run\n" +
				"prints the request and stops. Then writes the documentation, as generate does. Needs ANTHROPIC_API_KEY.",
			groupPaid, label},
		{"explain", "<path>", "ask Claude what each component does, every sentence cited",
			"One request per component whose facts have no remembered answer: names, paths, counts and the code's\n" +
				"own route summaries — never code. A sentence that cites nothing it was given is refused. Says how many\n" +
				"requests and what they should cost, and asks first; --dry-run prints them and stops. Needs ANTHROPIC_API_KEY.",
			groupPaid, explain},
		{"version", "", "print build information", "", "", func(e env, _ []string) error { return version(e) }},
		{"help", "[<command>]", "this list, or one command's flags", "", "", helpCommand},
	}
}

func lookupCommand(name string) (command, bool) {
	for _, c := range commands {
		if c.name == name {
			return c, true
		}
	}
	return command{}, false
}

// errHelped ends a command that printed its help: nothing failed.
var errHelped = errors.New("help shown")

// errStopped ends a command that stopped on purpose — a dry run, or a "no" to sending — with
// nothing written: nothing failed.
var errStopped = errors.New("stopped")

// usageError is a command used wrongly: exit code 2, and a pointer to its help.
type usageError struct{ msg string }

func (u usageError) Error() string { return u.msg }

func usagef(format string, a ...any) error { return usageError{fmt.Sprintf(format, a...)} }

// Exit codes: 0 done (help and stopping on purpose included), 1 failed, 2 used wrongly.
func exitCode(err error) int {
	var u usageError
	switch {
	case err == nil, errors.Is(err, errHelped), errors.Is(err, errStopped):
		return 0
	case errors.As(err, &u):
		return 2
	}
	return 1
}

func usage(out io.Writer) {
	fmt.Fprint(out, "archdoc — architecture documentation, read from a repository's code and configuration\n\n")
	fmt.Fprint(out, "usage:\n  archdoc <command> <path> [flags]\n  archdoc help <command>      a command's flags and what it does\n")
	for _, g := range groups {
		fmt.Fprintf(out, "\n%s:\n", g)
		for _, c := range commands {
			if c.group != g {
				continue
			}
			head := c.name + " " + c.args
			if len(head) > 22 {
				fmt.Fprintf(out, "  %s\n  %-22s %s\n", head, "", c.summary)
				continue
			}
			fmt.Fprintf(out, "  %-22s %s\n", head, c.summary)
		}
	}
	fmt.Fprint(out, "\n  version, help\n\nExit codes: 0 done, 1 failed, 2 used wrongly.\n")
}

func helpCommand(e env, args []string) error {
	if len(args) == 0 {
		usage(e.out)
		return nil
	}
	c, ok := lookupCommand(args[0])
	if !ok {
		return unknownCommand(args[0])
	}
	if c.name == "help" || c.name == "version" {
		fmt.Fprintf(e.out, "archdoc %s — %s\n", c.name, c.summary)
		return nil
	}
	return c.run(e, []string{"--help"})
}

func unknownCommand(name string) error {
	var names []string
	for _, c := range commands {
		names = append(names, c.name)
	}
	if near := nearest(name, names); near != "" {
		return usagef("unknown command %q — did you mean %q? 'archdoc help' lists them", name, near)
	}
	return usagef("unknown command %q — 'archdoc help' lists them", name)
}

// flags is a command's flag set, quiet: archdoc reports its own errors and prints its own help.
func flags(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	return fs
}

// parse reads a command's arguments: its help when asked for, a flag that has moved or been renamed
// pointed to where it went, a mistyped one answered with the nearest real one. It returns the words
// that are not flags.
func parse(e env, fs *flag.FlagSet, args []string, moved map[string]string) ([]string, error) {
	for _, a := range args {
		name := strings.TrimLeft(a, "-")
		if i := strings.Index(name, "="); i >= 0 {
			name = name[:i]
		}
		if !strings.HasPrefix(a, "-") {
			continue
		}
		if name == "h" || name == "help" {
			commandHelp(e.out, fs)
			return nil, errHelped
		}
		if where, ok := moved[name]; ok {
			return nil, usagef("%s %s — %s", fs.Name(), flagName(name), where)
		}
	}
	flagArgs, positional := partitionArgs(fs, args)
	if err := fs.Parse(flagArgs); err != nil {
		return nil, flagError(fs, err, moved)
	}
	return positional, nil
}

// flagError rewrites the flag package's message — "flag provided but not defined: -labels" — into
// one a person can act on.
func flagError(fs *flag.FlagSet, err error, moved map[string]string) error {
	msg := err.Error()
	help := fmt.Sprintf("'archdoc help %s' lists its flags", fs.Name())
	if name, ok := strings.CutPrefix(msg, "flag provided but not defined: "); ok {
		name = strings.TrimLeft(name, "-")
		var known []string
		fs.VisitAll(func(f *flag.Flag) { known = append(known, f.Name) })
		for old := range moved {
			known = append(known, old)
		}
		if near := nearest(name, known); near != "" {
			if where, ok := moved[near]; ok {
				return usagef("%s has no flag %s — did you mean %s? %s", fs.Name(), flagName(name), flagName(near), where)
			}
			return usagef("%s has no flag %s — did you mean %s? %s", fs.Name(), flagName(name), flagName(near), help)
		}
		return usagef("%s has no flag %s — %s", fs.Name(), flagName(name), help)
	}
	if name, ok := strings.CutPrefix(msg, "flag needs an argument: "); ok {
		return usagef("%s needs a value — %s", flagName(strings.TrimLeft(name, "-")), help)
	}
	return usagef("%s — %s", msg, help)
}

func flagName(name string) string {
	if len(name) == 1 {
		return "-" + name
	}
	return "--" + name
}

// commandHelp prints one command's usage, what it does, and its flags, read from its flag set.
func commandHelp(out io.Writer, fs *flag.FlagSet) {
	c, _ := lookupCommand(fs.Name())
	fmt.Fprintf(out, "usage: archdoc %s %s [flags]\n\n%s.\n", c.name, c.args, strings.ToUpper(c.summary[:1])+c.summary[1:])
	if c.about != "" {
		fmt.Fprintf(out, "\n%s\n", c.about)
	}
	if c.group == groupPaid {
		fmt.Fprintf(out, "\nEvery request is logged with what it cost: 'archdoc runs <path> --show <run>' prints it as sent.\n")
	}
	var lines [][2]string
	fs.VisitAll(func(f *flag.Flag) {
		kind, use := flag.UnquoteUsage(f)
		head := flagName(f.Name)
		if kind != "" {
			head += " <" + kind + ">"
		}
		if f.DefValue != "" && f.DefValue != "false" && f.DefValue != "0" {
			use += " (default " + f.DefValue + ")"
		}
		lines = append(lines, [2]string{head, use})
	})
	if len(lines) == 0 {
		return
	}
	fmt.Fprint(out, "\nflags:\n")
	for _, l := range lines {
		if len(l[0]) > 20 {
			fmt.Fprintf(out, "  %s\n  %-20s %s\n", l[0], "", l[1])
			continue
		}
		fmt.Fprintf(out, "  %-20s %s\n", l[0], l[1])
	}
}

// writePath is the one path a command that writes into a repository is given. It is required: a
// command that writes must never land in whatever directory it happened to be run from.
func writePath(fs *flag.FlagSet, positional []string) (string, error) {
	switch len(positional) {
	case 0:
		return "", usagef("%s writes into a repository, so it needs one: archdoc %s <path> — '.' is this folder", fs.Name(), fs.Name())
	case 1:
		return positional[0], nil
	}
	return "", usagef("%s takes one path, not %d: %s", fs.Name(), len(positional), strings.Join(positional, " "))
}

// readPath is the path of a command that only reads, which may be left out for this folder.
func readPath(fs *flag.FlagSet, positional []string) (string, error) {
	switch len(positional) {
	case 0:
		return ".", nil
	case 1:
		return positional[0], nil
	}
	return "", usagef("%s takes one path, not %d: %s", fs.Name(), len(positional), strings.Join(positional, " "))
}

func absolute(p string) string {
	if a, err := filepath.Abs(p); err == nil {
		return a
	}
	return p
}

// nearest is the candidate a mistyped word most likely meant: within two edits, or one the word
// starts or is the start of. Ties go to the first in sorted order, so the answer is stable.
func nearest(word string, candidates []string) string {
	sorted := append([]string(nil), candidates...)
	sort.Strings(sorted)
	best, bestD := "", 3
	for _, c := range sorted {
		d := distance(word, c)
		if d < bestD {
			best, bestD = c, d
		}
	}
	if best != "" {
		return best
	}
	for _, c := range sorted {
		if len(word) >= 3 && (strings.HasPrefix(c, word) || strings.HasPrefix(word, c)) {
			return c
		}
	}
	return ""
}

// distance is the Levenshtein distance between two words.
func distance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}
