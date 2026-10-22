package main

import (
	"context"

	"github.com/cruzambrociogl/archdoc"
)

func generate(e env, args []string) error {
	fs := flags("generate")
	toStdout := fs.Bool("stdout", false, "print the index instead of writing files")
	gaps := fs.Bool("gaps", false, "list what the code and configuration do not state")
	site := fs.Bool("site", false, "also write mkdocs.yml, so the documents build as a static site")
	verbose := fs.Bool("verbose", false, "list every file written")

	positional, err := parse(e, fs, args, map[string]string{
		"label":         "it is a command of its own now, so generate never costs money: archdoc label <path>",
		"label-model":   "now: archdoc label <path> --model <model>",
		"explain":       "it is a command of its own now, so generate never costs money: archdoc explain <path>",
		"explain-only":  "now: archdoc explain <path> --only <id>",
		"explain-limit": "now: archdoc explain <path> --limit <n>",
		"explain-gaps":  "renamed: --gaps",
	})
	if err != nil {
		return err
	}
	root, err := writePath(fs, positional)
	if err != nil {
		return err
	}
	return document(e, root, archdoc.GenerateOptions{IndexOnly: *toStdout, Gaps: *gaps, Site: *site, Verbose: *verbose})
}

// document runs the pipeline the way every writing command does: speaking to the terminal, and
// recording the version in the repository's history.
func document(e env, root string, o archdoc.GenerateOptions) error {
	o.Log, o.History = e.out, true
	_, err := archdoc.Generate(context.Background(), root, o)
	if err == nil && e.remember && !o.IndexOnly {
		archdoc.Remember(root) // Home's list; a failure to keep it is no failure of this command
	}
	return err
}
