package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/cruzambrociogl/archdoc"
)

// export builds the published site (SUR-05, surface-spec §3): the web app as static files, with
// the latest version's data — and the change since a baseline — beside it. Or the record: what
// archdoc made of the repository, in one file, for a site to be built from somewhere else.
func export(e env, args []string) error {
	fs := flags("export")
	site := fs.Bool("site", false, "build the published site into .archdoc/site")
	record := fs.Bool("record", false, "write the record into .archdoc/record.json: the model, the coverage report and the documents, in one file, with none of the code")
	since := fs.Int64("since", 0, "the `version` 'what changed' is measured against; 0 is the one before the latest")

	positional, err := parse(e, fs, args, nil)
	if err != nil {
		return err
	}
	if *site == *record {
		return usagef("export makes one thing, and says which: archdoc export <path> --site, or --record")
	}
	root, err := writePath(fs, positional)
	if err != nil {
		return err
	}
	out := e.out

	if *record {
		r, err := archdoc.MakeRecord(root)
		if err != nil {
			return err
		}
		size, err := r.Write(root)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "wrote %s — %.1f MB: the model, the coverage report and %d documents, with none of the code\n",
			archdoc.RecordFile, float64(size)/1e6, len(r.Documents))
		return nil
	}

	s, err := archdoc.BuildSite(context.Background(), root, *since)
	if err != nil {
		return err
	}
	files, size, err := s.Write(root)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "wrote %s — %d files, %.1f MB\n", archdoc.SiteDir, files, float64(size)/1e6)
	fmt.Fprintf(out, "Deploy the folder to any static host (GitHub Pages works), or preview it with a static server,\n")
	fmt.Fprintf(out, "e.g. 'python3 -m http.server 8080 -d %s', then http://localhost:8080.\n", filepath.Join(root, archdoc.SiteDir))
	fmt.Fprintf(out, "Browsers will not load its data from a file:// page.\n")
	return nil
}
