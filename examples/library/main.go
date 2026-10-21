// Command library documents a repository through archdoc's public API, from outside its module —
// the way the hosted platform's worker does. Go forbids importing internal/ from another module, so
// this compiling is the proof that the API is enough on its own.
//
//	go run . <path-to-a-repository>
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cruzambrociogl/archdoc"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: library <path-to-a-repository>")
		os.Exit(2)
	}
	if err := run(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "library:", err)
		os.Exit(1)
	}
}

func run(root string) error {
	ctx := context.Background()

	// Written into the repository, with no history kept and nothing said: a throwaway checkout.
	g, err := archdoc.Generate(ctx, root, archdoc.GenerateOptions{})
	if err != nil {
		return err
	}
	fmt.Printf("%s — %d files written\n", g.Model.Name, len(g.Files))
	archdoc.Count(g.Model).Print(os.Stdout, g.Model.Source)

	// The model as stored, compared with itself: what a platform does with two stored versions.
	stored, ok, err := archdoc.CommittedModel(root)
	if err != nil || !ok {
		return fmt.Errorf("no committed model after generating: %v", err)
	}
	fmt.Printf("%d change(s) between the model returned and the model written\n", len(archdoc.Compare(g.Model, stored).Drift))

	// The published site, in memory: a platform uploads these files instead of writing them.
	site, err := archdoc.BuildSite(ctx, root, 0)
	if err != nil {
		return err
	}
	fmt.Printf("published site: %d files\n", len(site.Files))
	return nil
}
