package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/serve"
	"github.com/cruzambrociogl/archdoc/web"
)

// siteDir is where the published site is built: inside archdoc's own directory, so the write set
// stays closed, and ignored by git, since CI rebuilds it from the committed record.
const siteDir = stateDir + "/site"

// publishedMarker switches the app to published mode: it reads data/ instead of the API, offers no
// actions, and opens citations on the repository host (surface-spec §3).
const publishedMarker = `<meta name="archdoc-mode" content="published">`

// export builds the published site (SUR-05, surface-spec §3): the web app as static files, with
// the latest version's data — and the change since a baseline — beside it.
func export(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	site := fs.Bool("site", false, "build the published site into .archdoc/site")
	since := fs.Int64("since", 0, "the version 'what changed' is measured against (default: the one before the latest)")

	flags, positional := partitionArgs(fs, args)
	if err := fs.Parse(flags); err != nil {
		return err
	}
	if !*site {
		return fmt.Errorf("export writes the published site: archdoc export --site <path>")
	}
	root := "."
	if len(positional) > 0 {
		root = positional[0]
	}

	assets, built := web.Assets()
	if !built {
		return fmt.Errorf("this archdoc was built without its web app — build web/ first (see README)")
	}
	data, err := serve.Export(root, *since)
	if err != nil {
		return err
	}

	files := map[string][]byte{}
	err = walkAssets(assets, func(name string, b []byte) {
		if name == "index.html" {
			b = []byte(strings.Replace(string(b), "<head>", "<head>\n    "+publishedMarker, 1))
		}
		files[name] = b
	})
	if err != nil {
		return err
	}
	for name, b := range data {
		files[name] = b
	}

	// The site directory is archdoc's alone, so it is replaced whole: nothing stale survives.
	dir := filepath.Join(root, filepath.FromSlash(siteDir))
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	names := make([]string, 0, len(files))
	size := 0
	for name := range files {
		names = append(names, name)
		size += len(files[name])
	}
	sort.Strings(names) // AC-7: the same order, and the same report, every run
	for _, name := range names {
		if err := write(root, siteDir+"/"+name, string(files[name])); err != nil {
			return err
		}
	}

	fmt.Fprintf(out, "wrote %s — %d files, %.1f MB\n", siteDir, len(names), float64(size)/1e6)
	fmt.Fprintf(out, "Deploy the folder to any static host (GitHub Pages works), or preview it with a static server,\n")
	fmt.Fprintf(out, "e.g. 'python3 -m http.server 8080 -d %s', then http://localhost:8080.\n", filepath.Join(root, siteDir))
	fmt.Fprintf(out, "Browsers will not load its data from a file:// page.\n")
	return nil
}

func walkAssets(assets fs.FS, visit func(name string, b []byte)) error {
	return fs.WalkDir(assets, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			return err
		}
		b, err := fs.ReadFile(assets, p)
		if err != nil {
			return err
		}
		visit(p, b)
		return nil
	})
}
