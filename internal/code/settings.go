package code

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Settings: where a Java or .NET application keeps the addresses it connects to. Spring reads them
// from application.properties or application.yml under src/main/resources; ASP.NET from
// appsettings.json beside the project. Both are configuration a person writes, so every URL in
// them is an endpoint the application is set up to reach — and a database connection string names
// its server too. Each file is read for its hosts only, line by line, so every one is cited at its
// line; one file per profile (application-docker.yml, appsettings.Development.json) is read like
// the rest, since each describes a way the application is run.

var springSettings = regexp.MustCompile(`^(application|bootstrap)([-.][\w-]+)?\.(properties|ya?ml)$`)
var dotnetSettings = regexp.MustCompile(`^appsettings(\.[\w-]+)?\.json$`)

func settings(repo, dir string, java bool) []archdoc.SourceFile {
	base := dir
	if java {
		base = path.Join(dir, "src", "main", "resources")
	}
	entries, err := os.ReadDir(filepath.Join(repo, filepath.FromSlash(base)))
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && (java && springSettings.MatchString(e.Name()) || !java && dotnetSettings.MatchString(e.Name())) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	var out []archdoc.SourceFile
	for _, name := range names {
		rel := path.Join(base, name)
		content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		f := archdoc.SourceFile{Path: rel, Lines: lines(content)}
		switch path.Ext(name) {
		case ".properties":
			f.Language, f.Hosts = "Properties", propertiesHosts(string(content), rel)
		case ".json":
			f.Language, f.Hosts = "JSON", jsonHosts(string(content), rel)
		default:
			f.Language, f.Hosts = "YAML", yamlHosts(string(content), rel)
		}
		out = append(out, f)
	}
	return out
}

func propertiesHosts(content, file string) []archdoc.HostRef {
	var out []archdoc.HostRef
	for i, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' || line[0] == '!' {
			continue
		}
		k := strings.IndexAny(line, "=:")
		if k < 0 {
			continue
		}
		out = append(out, settingHosts(strings.TrimSpace(line[:k]), strings.TrimSpace(line[k+1:]), archdoc.Provenance{File: file, Line: i + 1})...)
	}
	return out
}

var yamlLine = regexp.MustCompile(`^(\s*)(?:-\s+)?([\w.\-/\[\]]+)\s*:\s*(.*)$`)

func yamlHosts(content, file string) []archdoc.HostRef {
	type level struct {
		indent int
		key    string
	}
	var stack []level
	var out []archdoc.HostRef
	for i, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") || strings.TrimSpace(line) == "---" {
			if strings.TrimSpace(line) == "---" {
				stack = stack[:0]
			}
			continue
		}
		m := yamlLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		indent := len(m[1])
		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}
		value := strings.TrimSpace(m[3])
		if j := strings.Index(value, " #"); j >= 0 {
			value = strings.TrimSpace(value[:j])
		}
		if value == "" {
			stack = append(stack, level{indent, m[2]})
			continue
		}
		parts := make([]string, 0, len(stack)+1)
		for _, s := range stack {
			parts = append(parts, s.key)
		}
		key := strings.Join(append(parts, m[2]), ".")
		out = append(out, settingHosts(key, strings.Trim(value, `"'`), archdoc.Provenance{File: file, Line: i + 1})...)
	}
	return out
}

var jsonPair = regexp.MustCompile(`"([^"]+)"\s*:\s*"((?:[^"\\]|\\.)*)"`)

func jsonHosts(content, file string) []archdoc.HostRef {
	var out []archdoc.HostRef
	for i, line := range strings.Split(content, "\n") {
		for _, m := range jsonPair.FindAllStringSubmatch(line, -1) {
			out = append(out, settingHosts(m[1], strings.ReplaceAll(m[2], `\\`, `\`), archdoc.Provenance{File: file, Line: i + 1})...)
		}
	}
	return out
}

// placeholderValue resolves Spring's ${NAME:default} to its default, which is what the application
// connects to when nothing overrides it.
var placeholderValue = regexp.MustCompile(`\$\{[^:}]+:([^}]*)\}`)

// settingHosts are the hosts one setting names: a URL — jdbc: and all — a connection string's
// server, or a host property's value.
func settingHosts(key, value string, prov archdoc.Provenance) []archdoc.HostRef {
	value = placeholderValue.ReplaceAllString(value, "$1")
	value = strings.TrimPrefix(value, "jdbc:")
	short := key[strings.LastIndex(key, ".")+1:]
	if h, ok := hostOf(value); ok {
		h.Key, h.Prov = key, prov
		return []archdoc.HostRef{h}
	}
	// A list of URLs: eureka's defaultZone, a gateway's routes.
	var out []archdoc.HostRef
	for _, part := range strings.Split(value, ",") {
		if h, ok := hostOf(strings.TrimSpace(part)); ok {
			h.Key, h.Prov = key, prov
			out = append(out, h)
		}
	}
	if len(out) > 0 {
		return out
	}
	if server, scheme, ok := connectionServer(value); ok {
		return []archdoc.HostRef{{Host: server, Scheme: scheme, Value: value, Key: key, Prov: prov}}
	}
	if hostKey.MatchString(short) && hostName.MatchString(value) && strings.ContainsAny(value, "abcdefghijklmnopqrstuvwxyz") {
		return []archdoc.HostRef{{Host: value, Value: value, Key: key, Prov: prov}}
	}
	return nil
}

// connectionServer is the server an ADO.NET connection string names — Server=sqlserver,1433;… or
// Host=db;Port=5432;… — and the kind of database that convention belongs to.
func connectionServer(value string) (server, scheme string, ok bool) {
	if !strings.Contains(value, ";") || !strings.Contains(value, "=") {
		return "", "", false
	}
	for _, part := range strings.Split(value, ";") {
		k, v, found := strings.Cut(part, "=")
		if !found {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(k)) {
		case "server", "data source", "address", "addr":
			scheme = "sqlserver"
		case "host":
			scheme = "postgres"
		default:
			continue
		}
		v = strings.TrimPrefix(strings.TrimSpace(v), "tcp:")
		if i := strings.IndexAny(v, ",:\\"); i >= 0 {
			v = v[:i]
		}
		if v == "" || strings.HasPrefix(v, "(") || v == "." || !hostName.MatchString(v) {
			return "", "", false // (localdb), ".": this machine
		}
		return v, scheme, true
	}
	return "", "", false
}

// LibrarySchemas are the table classes of the libraries an application is built with. A .NET or Java
// application usually keeps its data model in a class library its project references — eShopOnWeb's
// entities in ApplicationCore, its DbContext in Infrastructure — and the library is built into the
// application, so its tables are the application's. Only the files that say what a table is are kept:
// a JPA entity and the classes it extends, or an EF Core DbContext, the classes it holds a DbSet of,
// and theirs. They are schemas, not code: no component, no flow is made of them.
func LibrarySchemas(repo string, libs []archdoc.App) []archdoc.SourceFile {
	var all []archdoc.SourceFile
	for _, lib := range libs {
		src, ok := Read(repo, lib, nil)
		if ok {
			all = append(all, src.Files...)
		}
	}
	declares := map[string]int{} // class → file index
	for i, f := range all {
		for _, c := range f.Classes {
			if _, seen := declares[c.Name]; !seen && c.Name != "" {
				declares[c.Name] = i
			}
		}
	}
	keep := map[int]bool{}
	var want []string
	for i, f := range all {
		for _, c := range f.Classes {
			context := false
			for _, e := range c.Extends {
				context = context || strings.HasSuffix(e, "DbContext")
			}
			for _, d := range c.Decorators {
				if d.Name == "Entity" || d.Name == "Table" || d.Name == "MappedSuperclass" || d.Name == "Embeddable" {
					keep[i] = true
					want = append(want, c.Extends...)
				}
			}
			if !context {
				continue
			}
			keep[i] = true
			for _, fl := range c.Fields {
				if inner, ok := strings.CutPrefix(fl.Type, "DbSet<"); ok {
					want = append(want, strings.TrimSuffix(inner, ">"))
				}
			}
		}
	}
	// What a table class extends holds columns too: BaseEntity's Id.
	seen := map[string]bool{}
	for len(want) > 0 {
		name := want[0][strings.LastIndex(want[0], ".")+1:]
		want = want[1:]
		i, ok := declares[name]
		if seen[name] || !ok {
			continue
		}
		seen[name] = true
		keep[i] = true
		for _, c := range all[i].Classes {
			if c.Name == name {
				want = append(want, c.Extends...)
			}
		}
	}
	var out []archdoc.SourceFile
	for i, f := range all {
		if keep[i] {
			f.Imports, f.Hosts, f.Calls = nil, nil, nil
			out = append(out, f)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
