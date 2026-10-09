package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cruzambrociogl/archdoc/internal/rules"
)

// initCommand is SUR-01: archdoc's own directory in a repository, with the file a person corrects
// the model in. generate needs none of it — it creates what it writes — so init is where a person
// starts who means to correct things. Nothing that exists is overwritten, and nothing outside
// .archdoc/ is touched: the write set stays closed (hard rule 2).
func initCommand(e env, args []string) error {
	fs := flags("init")
	positional, err := parse(e, fs, args, nil)
	if err != nil {
		return err
	}
	root, err := writePath(fs, positional)
	if err != nil {
		return err
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return fmt.Errorf("%s is not a directory", root)
	}

	for _, f := range []struct{ rel, content, what string }{
		{rules.Name, rulesTemplate, "corrections that survive regeneration; every line is a comment until you write one"},
		{stateDir + "/.gitignore", ignoreFile, "keeps the local history and the built site out of git"},
	} {
		if _, err := os.Stat(filepath.Join(root, f.rel)); err == nil {
			fmt.Fprintf(e.out, "kept    %s — already there\n", f.rel)
			continue
		}
		if err := write(root, f.rel, f.content); err != nil {
			return err
		}
		fmt.Fprintf(e.out, "created %s — %s\n", f.rel, f.what)
	}
	fmt.Fprintf(e.out, "\nNext: 'archdoc generate %s' writes the documentation; 'archdoc scan %s' previews it first.\n", root, root)
	return nil
}

const rulesTemplate = `# Corrections to what archdoc reads from this repository — applied on every run, so they survive
# regeneration, and always over anything a model suggested. Every value a rule sets cites its line.
#
# A rule matches elements, a relationship or routes, and sets, excludes or removes:
#
# rules:
#   # Say what something is when the code does not.
#   - match: { name: api }
#     set: { description: Serves the public REST API, technology: Go }
#
#   # A container that is infrastructure, not part of the architecture.
#   - match: { image: "*/vector*" }
#     exclude: true
#
#   # Treat a service as a proxy, so arrows are drawn through it.
#   - match: { name: pgbouncer }
#     set: { kind: proxy }
#
#   # A component inside one container only: by its name and the container's.
#   - match: { name: services, in: immich-server }
#     set: { description: Business logic behind every route }
#
#   # A relationship between two Compose services that nothing in the files states — or, with
#   # remove: true, one that is stated and wrong.
#   - edge: { from: web, to: api }
#     set: { label: calls over HTTPS }
#
#   # Leave out routes, or describe one; * matches any run of characters.
#   - route: "GET /api/health*"
#     exclude: true
#
# match takes name, image (both accept *), kind, in and id — the id the app's inspector shows.
# set takes name, description, technology, kind and group.
`
