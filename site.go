package archdoc

import (
	"context"
	"errors"
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

// ErrNoWebApp is BuildSite in a build of archdoc made without its web app: web/ was not built
// before the Go code, so there is no app to publish.
var ErrNoWebApp = errors.New("this archdoc was built without its web app — build web/ first (see README)")

// Site is the published site (SUR-05, surface-spec §3): the web app as static files, with the
// latest version's data — and the change since a baseline — beside it. Files maps each path,
// relative to the site's root, to its content.
type Site struct {
	Files map[string][]byte
}

// BuildSite builds the published site from a repository's history, or — in a clone with none —
// from its committed model. since is the version "what changed" is measured against; 0 is the one
// before the latest. Nothing is written: Write puts the site in place, or the caller sends the
// files wherever they are served from.
func BuildSite(ctx context.Context, root string, since int64) (*Site, error) {
	assets, built := web.Assets()
	if !built {
		return nil, ErrNoWebApp
	}
	data, err := serve.Export(root, since)
	if err != nil {
		return nil, err
	}

	files := map[string][]byte{}
	err = walkAssets(assets, func(name string, b []byte) {
		if name == "index.html" {
			b = []byte(strings.Replace(string(b), "<head>", "<head>\n    "+publishedMarker, 1))
		}
		files[name] = b
	})
	if err != nil {
		return nil, err
	}
	for name, b := range data {
		files[name] = b
	}
	return &Site{Files: files}, nil
}

// Write replaces the repository's .archdoc/site with the site, whole — the directory is archdoc's
// alone, so nothing stale survives. It returns how many files it wrote and their size in bytes.
func (s *Site) Write(root string) (files, size int, err error) {
	dir := filepath.Join(root, filepath.FromSlash(siteDir))
	if err := os.RemoveAll(dir); err != nil {
		return 0, 0, err
	}
	names := make([]string, 0, len(s.Files))
	for name := range s.Files {
		names = append(names, name)
		size += len(s.Files[name])
	}
	sort.Strings(names) // AC-7: the same order, and the same report, every run
	for _, name := range names {
		if err := write(root, siteDir+"/"+name, string(s.Files[name])); err != nil {
			return 0, 0, err
		}
	}
	return len(names), size, nil
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
