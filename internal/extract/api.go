package extract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// An API description is how a repository says, in one file, what one of its containers offers:
// an OpenAPI document. Who calls that API is said in two ways, and both are read:
//
//   - A package whose source is made of the document's paths — a generated client checked in,
//     like a TypeScript SDK. Counted, not assumed: most of the paths must be there.
//   - A command that generates a client from the document into a directory — openapi-generator
//     with its -i and -o — where the client itself is built and never committed.
//
// Which container the document describes is decided later, against the routes the code declares
// (internal/model): this file only reads.

const (
	maxAPIFile    = 32 << 20
	maxClientFile = 4 << 20
	// A client holds most of the API. Half is the line: a test suite that calls a few endpoints,
	// or a front end with a handful of hand-written fetches, is below it.
	clientShare = 2
	minPaths    = 5
)

var (
	apiFileName  = regexp.MustCompile(`(?i)(openapi|swagger|api-spec)`)
	httpMethods  = map[string]bool{"get": true, "put": true, "post": true, "delete": true, "patch": true, "head": true, "options": true}
	generatorCmd = regexp.MustCompile(`openapi-generator(?:-cli)?\b.*\bgenerate\b`)
	generatorIn  = regexp.MustCompile(`(?:^|\s)(?:-i|--input-spec)[\s=]+("[^"]+"|'[^']+'|\S+)`)
	generatorOut = regexp.MustCompile(`(?:^|\s)(?:-o|--output)[\s=]+("[^"]+"|'[^']+'|\S+)`)
	clientSource = map[string]bool{".ts": true, ".js": true, ".mjs": true, ".dart": true, ".py": true, ".go": true, ".kt": true, ".swift": true, ".java": true, ".cs": true, ".rb": true}
	commandFile  = map[string]bool{".sh": true, ".bash": true, ".yml": true, ".yaml": true, ".json": true, ".mk": true, "": true}
)

// APIs reads every API description in the repository and finds its clients.
func APIs(root string, apps []archdoc.App) []archdoc.API {
	var out []archdoc.API
	var commands []string // files that may hold a generator command
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && (appSkipDirs[d.Name()] || strings.HasPrefix(d.Name(), ".") && d.Name() != ".github") {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		ext := strings.ToLower(path.Ext(d.Name()))
		if info, err := d.Info(); err == nil && commandFile[ext] && info.Size() < 256<<10 {
			commands = append(commands, rel)
		}
		if (ext == ".json" || ext == ".yaml" || ext == ".yml") && apiFileName.MatchString(d.Name()) {
			if api, ok := apiDocument(root, rel); ok {
				out = append(out, api)
			}
		}
		return nil
	})
	if len(out) == 0 {
		return nil
	}
	sort.Slice(out, func(i, j int) bool { return out[i].File < out[j].File })

	// The application a path belongs to: the deepest one whose directory holds it.
	owner := func(rel string) (archdoc.App, bool) {
		var best archdoc.App
		found := false
		for _, a := range apps {
			if a.Role == archdoc.RoleWorkspace {
				continue
			}
			if a.Dir == "." || rel == a.Dir || strings.HasPrefix(rel, a.Dir+"/") {
				if !found || len(a.Dir) > len(best.Dir) || best.Dir == "." {
					best, found = a, true
				}
			}
		}
		return best, found
	}

	for i := range out {
		api := &out[i]
		seen := map[string]bool{}
		add := func(c archdoc.APIClient) {
			if !seen[c.App] {
				seen[c.App] = true
				api.Clients = append(api.Clients, c)
			}
		}
		for _, c := range codeClients(root, apps, *api) {
			add(c)
		}
		for _, c := range generatedClients(root, commands, *api, owner) {
			add(c)
		}
		sort.Slice(api.Clients, func(a, b int) bool { return api.Clients[a].App < api.Clients[b].App })
	}
	return out
}

// apiDocument reads an OpenAPI or Swagger document: its paths and the methods on each.
func apiDocument(root, rel string) (archdoc.API, bool) {
	full := filepath.Join(root, filepath.FromSlash(rel))
	if info, err := os.Stat(full); err != nil || info.Size() > maxAPIFile {
		return archdoc.API{}, false
	}
	content, err := os.ReadFile(full)
	if err != nil {
		return archdoc.API{}, false
	}
	var doc struct {
		OpenAPI string                    `json:"openapi" yaml:"openapi"`
		Swagger string                    `json:"swagger" yaml:"swagger"`
		Paths   map[string]map[string]any `json:"paths" yaml:"paths"`
	}
	if strings.HasSuffix(rel, ".json") {
		err = json.Unmarshal(content, &doc)
	} else {
		err = yaml.Unmarshal(content, &doc)
	}
	if err != nil || (doc.OpenAPI == "" && doc.Swagger == "") || len(doc.Paths) == 0 {
		return archdoc.API{}, false
	}
	api := archdoc.API{File: rel, Prov: archdoc.Provenance{File: rel, Line: max(lineOf(content, "paths"), 1)}}
	for p, methods := range doc.Paths {
		for m := range methods {
			if httpMethods[strings.ToLower(m)] {
				api.Operations = append(api.Operations, archdoc.Operation{Method: strings.ToUpper(m), Path: p})
			}
		}
	}
	sort.Slice(api.Operations, func(i, j int) bool {
		if api.Operations[i].Path != api.Operations[j].Path {
			return api.Operations[i].Path < api.Operations[j].Path
		}
		return api.Operations[i].Method < api.Operations[j].Method
	})
	return api, len(api.Operations) > 0
}

// staticPart is a path up to its first parameter: /albums/{id}/users is /albums/.
func staticPart(p string) string {
	if i := strings.IndexAny(p, "{:"); i >= 0 {
		p = p[:i]
	}
	return p
}

// codeClients are the applications and packages whose own source holds most of the API's paths as
// string literals — what a generated client looks like.
func codeClients(root string, apps []archdoc.App, api archdoc.API) []archdoc.APIClient {
	want := map[string]bool{}
	for _, op := range api.Operations {
		if s := staticPart(op.Path); len(s) > 1 {
			want[s] = true
			want[strings.TrimSuffix(s, "/")] = true
		}
	}
	distinct := map[string]bool{}
	for _, op := range api.Operations {
		if s := strings.TrimSuffix(staticPart(op.Path), "/"); len(s) > 1 {
			distinct[s] = true
		}
	}
	if len(distinct) < minPaths {
		return nil
	}
	nested := map[string]bool{}
	for _, a := range apps {
		nested[a.Dir] = true
	}
	var out []archdoc.APIClient
	for _, a := range apps {
		if a.Role == archdoc.RoleWorkspace || a.Dir == "." {
			continue
		}
		found := map[string]bool{}
		perFile := map[string]int{}
		base := filepath.Join(root, filepath.FromSlash(a.Dir))
		filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			rel, _ := filepath.Rel(root, p)
			rel = filepath.ToSlash(rel)
			if d.IsDir() {
				if p != base && (appSkipDirs[d.Name()] || strings.HasPrefix(d.Name(), ".") || nested[rel]) {
					return filepath.SkipDir
				}
				return nil
			}
			if !clientSource[strings.ToLower(path.Ext(d.Name()))] {
				return nil
			}
			if info, err := d.Info(); err != nil || info.Size() > maxClientFile {
				return nil
			}
			content, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			for _, lit := range pathLiterals(content) {
				lit = strings.TrimSuffix(lit, "/")
				if want[lit] && !found[lit] {
					found[lit] = true
					perFile[rel]++
				}
			}
			return nil
		})
		if len(found)*clientShare < len(distinct) {
			continue
		}
		// Cited at the file that holds the most of them; the first by name on a tie.
		files := make([]string, 0, len(perFile))
		for f := range perFile {
			files = append(files, f)
		}
		sort.Slice(files, func(i, j int) bool {
			if perFile[files[i]] != perFile[files[j]] {
				return perFile[files[i]] > perFile[files[j]]
			}
			return files[i] < files[j]
		})
		out = append(out, archdoc.APIClient{App: a.Dir, Dir: path.Dir(files[0]),
			How:  fmt.Sprintf("its code names %d of the %d paths in %s", len(found), len(distinct), api.File),
			Prov: archdoc.Provenance{File: files[0], Line: 1}})
	}
	return out
}

// pathLiterals are the string literals in a source file that start like a path: what follows a
// quote and a slash, up to where the literal ends or a value is spliced in.
func pathLiterals(content []byte) []string {
	var out []string
	for i := 0; i+1 < len(content); i++ {
		if c := content[i]; (c != '"' && c != '\'' && c != '`') || content[i+1] != '/' {
			continue
		}
		j := i + 1
		for j < len(content) && !bytes.ContainsRune([]byte("\"'`${?\n "), rune(content[j])) {
			j++
		}
		if j-i > 2 {
			out = append(out, string(content[i+1:j]))
		}
		i = j - 1
	}
	return out
}

// generatedClients are the directories a generator command writes a client of this API into:
// openapi-generator-cli generate -i ./spec.json -o ../mobile/generated/openapi.
func generatedClients(root string, commands []string, api archdoc.API, owner func(string) (archdoc.App, bool)) []archdoc.APIClient {
	unquote := func(s string) string { return strings.Trim(s, `"'`) }
	var out []archdoc.APIClient
	for _, file := range commands {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil || !bytes.Contains(content, []byte("openapi-generator")) {
			continue
		}
		for n, line := range strings.Split(string(content), "\n") {
			if !generatorCmd.MatchString(line) {
				continue
			}
			in, to := generatorIn.FindStringSubmatch(line), generatorOut.FindStringSubmatch(line)
			if in == nil || to == nil || path.Base(unquote(in[1])) != path.Base(api.File) {
				continue
			}
			// The command runs from the script's directory or one above it; the one from which its
			// input is this document is the one it is written for.
			for base := path.Dir(file); ; base = path.Dir(base) {
				if path.Clean(path.Join(base, unquote(in[1]))) == api.File {
					dir := path.Clean(path.Join(base, unquote(to[1])))
					if app, ok := owner(dir); ok && !strings.HasPrefix(dir, "..") {
						out = append(out, archdoc.APIClient{App: app.Dir, Dir: dir,
							How:  fmt.Sprintf("a client generated from %s into %s", api.File, dir),
							Prov: archdoc.Provenance{File: file, Line: n + 1}})
					}
					break
				}
				if base == "." || base == "/" {
					break
				}
			}
		}
	}
	return out
}
