package main

import (
	"fmt"

	"github.com/cruzambrociogl/archdoc"
)

// initCommand is SUR-01: archdoc's own directory in a repository, with the file a person corrects
// the model in (archdoc.Init). Nothing that exists is overwritten.
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
	files, err := archdoc.Init(root)
	for _, f := range files {
		if f.Created {
			fmt.Fprintf(e.out, "created %s — %s\n", f.Path, f.What)
		} else {
			fmt.Fprintf(e.out, "kept    %s — already there\n", f.Path)
		}
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(e.out, "\nNext: 'archdoc generate %s' writes the documentation; 'archdoc scan %s' previews it first.\n", root, root)
	return nil
}
