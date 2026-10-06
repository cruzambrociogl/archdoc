package archdoc

import "fmt"

// Provenance is where a fact came from: the file that proves it, and the line within that
// file. Every fact carries exactly one — the file that won the merge.
//
// O-8 established that compose-go discards source positions, so extraction runs two passes
// and reconciles them by key path. See docs/decisions.md, 2026-08-26.
type Provenance struct {
	// Origin says what kind of evidence this is. The zero value is Extraction, because a
	// fact read from a file is the ordinary case and the one every parser produces.
	Origin Origin `json:"origin,omitempty"`

	File   string `json:"file"`             // repository-relative
	Line   int    `json:"line"`             // 1-indexed; 0 means unknown
	Column int    `json:"column,omitempty"` // 1-indexed; 0 means unknown

	// Note carries the evidence for an origin that has no line: which catalog entry matched,
	// or which model run proposed it. A catalog fact is traceable — just not to a file.
	Note string `json:"note,omitempty"`
}

// String renders provenance the way an editor expects: path/to/file.yml:13. A fact with no
// line renders as its origin and note instead — "catalog: postgres".
func (p Provenance) String() string {
	if p.File == "" && p.Note != "" {
		return string(p.origin()) + ": " + p.Note
	}
	if p.Line == 0 {
		return p.File
	}
	return fmt.Sprintf("%s:%d", p.File, p.Line)
}

func (p Provenance) origin() Origin {
	if p.Origin == "" {
		return Extraction
	}
	return p.Origin
}

// Known reports whether this provenance actually points somewhere. A fact whose provenance is
// unknown must not be emitted — P1 requires every element to be traceable.
//
// What counts as traceable depends on the origin. A file said it, so it needs a line. A catalog
// or the model supplied it, so it needs to name what supplied it — AC-1 admits catalog
// provenance explicitly, and demanding a line for something no file stated would make the
// criterion unsatisfiable rather than strict.
func (p Provenance) Known() bool {
	switch p.origin() {
	case Catalog, Semantic:
		return p.Note != ""
	default:
		return p.File != "" && p.Line > 0
	}
}

// EvidenceKind records how strongly a node is attested. §3 of the product definition:
// declared and referenced are both real evidence, distinguished so the reader knows which
// boxes are proven infrastructure and which are declared intentions. Inferred must not exist
// in R1.
type EvidenceKind string

const (
	// Declared means the repository defines it — a service in a compose file.
	Declared EvidenceKind = "declared"
	// Referenced means the repository points at it without defining it — a host in an
	// environment variable. Proves the system talks to it, and nothing more.
	Referenced EvidenceKind = "referenced"
)

// Service is a container-level component found in configuration.
type Service struct {
	Name     string       `json:"name"`  // the compose service key, and the canonical identity
	Image    string       `json:"image"` // empty when the service is built rather than pulled
	Evidence EvidenceKind `json:"evidence"`
	Prov     Provenance   `json:"provenance"`

	// DependsOn is every service this one names as a dependency (EXT/MDL-02).
	DependsOn []Dependency `json:"depends_on,omitempty"`

	// Ports are the published ones only. An unpublished port is internal plumbing; a
	// published one is declared evidence that something outside reaches in, which is where
	// actors come from.
	Ports []Port `json:"ports,omitempty"`

	// Endpoints are network locations named in this service's environment (MDL-04).
	Endpoints []Endpoint `json:"endpoints,omitempty"`

	// Networks this service is attached to (MDL-09). Membership is a declared boundary: two
	// services on no common network cannot reach each other, whatever else the file says.
	Networks []NetworkRef `json:"networks,omitempty"`

	// Aliases are the other names this service answers to — container_name, and any network
	// aliases. A gateway's routing table names upstreams by hostname, and the hostname is not
	// always the service key: Supabase routes to `realtime-dev.supabase-realtime`, which the
	// compose file declares as that service's container_name. Resolving through aliases is
	// what stops a real service being drawn twice, once as itself and once as a stranger.
	Aliases []string `json:"aliases,omitempty"`

	// Mounts are the files and directories the repository hands to this container. They
	// matter because a gateway's routing table arrives this way, and the mount line is the
	// repository saying where its own configuration lives.
	Mounts []Mount `json:"mounts,omitempty"`
}

// Mount is one bind mount: a path in the repository, and where the container sees it.
type Mount struct {
	Source string     `json:"source"` // repository-relative
	Target string     `json:"target"` // the path inside the container
	Prov   Provenance `json:"provenance"`
}

// Route is a routing rule read from a gateway's own configuration (MDL-03/EXT-03).
//
// depends_on says a gateway starts after a service; a route says traffic actually reaches it.
// Only the second justifies an arrow, which is why routes are extracted separately rather than
// inferred from the compose file.
type Route struct {
	Gateway string `json:"gateway"` // the service whose configuration declared it
	Target  string `json:"target"`  // the host it routes to
	Path    string `json:"path,omitempty"`

	// Config is the file the rule was read from, repository-relative. It is not the same file
	// as the compose file, and a reader following the citation needs to land in the right one.
	Config string     `json:"config"`
	Prov   Provenance `json:"provenance"`
}

// NetworkRef is one service's membership of one network.
type NetworkRef struct {
	Name string     `json:"name"`
	Prov Provenance `json:"provenance"`
}

// Network is a network the file declares, and what it declares about it.
type Network struct {
	Name string `json:"name"`

	// Internal is Compose's own `internal: true` — the network has no outbound external
	// connectivity. It is the one trust boundary configuration states outright, so MDL-11
	// can record it without inferring anything.
	Internal bool `json:"internal,omitempty"`

	Prov Provenance `json:"provenance"`
}

// Dependency is one entry of a service's depends_on.
type Dependency struct {
	Service string     `json:"service"`
	Prov    Provenance `json:"provenance"`
}

// Port is a published port mapping: the host side is what makes it reachable from outside.
type Port struct {
	Published string     `json:"published"` // string, because Compose allows ranges and "8080"
	Target    int        `json:"target"`
	Protocol  string     `json:"protocol,omitempty"` // tcp when unstated
	Prov      Provenance `json:"provenance"`
}

// Endpoint is a network location named by an environment value — REDIS_URL, S3_ENDPOINT,
// DATABASE_HOST. It is how a repository points at something it does not itself declare.
//
// Only the location is kept. The raw environment map is deliberately not carried in the
// FactSet: Compose environments hold credentials, and the FactSet is written to disk. Carrying
// only parsed locations means there is nothing to redact later.
type Endpoint struct {
	Var    string     `json:"var"`              // the variable that named it
	Scheme string     `json:"scheme,omitempty"` // postgres, redis, https, s3 — "" when bare host
	Host   string     `json:"host"`
	Port   int        `json:"port,omitempty"`
	Prov   Provenance `json:"provenance"`
}

// FactSet is everything extraction found, and the contract between the deterministic half of
// the pipeline and everything downstream. It is produced by reading files only — no guessing
// and no model involved. See §11 of the product definition.
type FactSet struct {
	Root string `json:"root"` // absolute path to the repository that was scanned

	// Name is the system being documented — Compose's own project name where the file
	// declares one, and the repository's directory otherwise. It is what the context view
	// puts on the box that everything declared collapses into.
	Name string `json:"name"`

	// Source is the compose file discovery selected, repository-relative.
	Source string `json:"source"`

	// Considered lists every file discovery examined, whether or not it was chosen. Recorded
	// so a reader can see what was skipped rather than having to trust that nothing was.
	Considered []Candidate `json:"considered"`

	Services []Service `json:"services"`

	// Networks the file declares at the top level, in name order.
	Networks []Network `json:"networks,omitempty"`

	// Routes read from gateway configuration the compose file mounts (MDL-03).
	Routes []Route `json:"routes,omitempty"`

	// Apps are the applications the repository holds, found by their manifests (F-02, D-5) —
	// every manifest discovery read, including the ones that are not a running part of the
	// system (libraries, tests, docs, tooling), each with its role and why.
	Apps []App `json:"apps,omitempty"`

	// Interpolation is the dotenv file Compose's ${VARIABLES} were filled from, repository-
	// relative, or empty when none was found. Sample is set when it is a sample — example.env,
	// .env.example, .env.sample — whose values are defaults a real deployment may override.
	Interpolation *EnvSource `json:"interpolation,omitempty"`

	// Sources are the code of each application that runs as a container, read with a parser
	// (F-03): its files and what each one imports. Applications in a language archdoc does not
	// read yet have none, and coverage says so.
	Sources []Source `json:"sources,omitempty"`
}

// Source is what archdoc read of one application's own code.
type Source struct {
	App  string `json:"app"`  // the application's directory, repository-relative
	Root string `json:"root"` // where its code starts — src/, its Python package, or the directory itself
	// Files are in path order; tests, type declarations and other applications nested inside are
	// not read.
	Files []SourceFile `json:"files"`
}

// SourceFile is one file and what it imports.
type SourceFile struct {
	Path     string   `json:"path"` // repository-relative
	Language string   `json:"language"`
	Lines    int      `json:"lines"`
	Imports  []Import `json:"imports,omitempty"`
	// Classes are the classes the file declares, with their decorators and what their constructor
	// takes — how NestJS says what a class is and what is injected into it. Functions decorated
	// at the top level (FastAPI's @router.get) are recorded as methods of a class with no name.
	Classes []Class `json:"classes,omitempty"`
	// Hosts are network locations the code names in a literal: a URL, or a host property's value.
	Hosts []HostRef `json:"hosts,omitempty"`
	// Calls are outbound HTTP calls whose target is computed at run time — kept, unresolved (D-6).
	Calls []Call `json:"calls,omitempty"`
	// Prefix is a global route prefix the file sets — NestJS's app.setGlobalPrefix('api').
	Prefix *Literal `json:"prefix,omitempty"`
	// Constants are string enum members the file declares, by qualified name — RouteKey.Asset is
	// "assets" — so a decorator that names one can be resolved by name.
	Constants []Constant `json:"constants,omitempty"`
	// Partial is set when the parser recovered from something it could not read in the file;
	// what it did read is still reported, and the gap is coverage, not a guess.
	Partial bool `json:"partial,omitempty"`
}

// Resolution is how an import was tied to what it names (vision D-4: the provenance of a link
// records how it was resolved, because a path and a convention are not equally strong evidence).
type Resolution string

const (
	// ByPath: a relative import, joined to the importing file's directory.
	ByPath Resolution = "path"
	// ByAlias: a path alias the application's configuration declares (tsconfig paths, baseUrl)
	// or its framework defines ($lib in SvelteKit).
	ByAlias Resolution = "alias"
	// ByModule: a Python absolute import, found as a module of the application's own package.
	ByModule Resolution = "module"
	// ByPackage: not the application's own code — a dependency, the standard library, or a
	// framework's virtual module. Package names it.
	ByPackage Resolution = "package"
	// NoMatch: it looks like the application's own code and no file matches. Kept, so the gap
	// is visible rather than silently dropped (vision D-6).
	NoMatch Resolution = "unresolved"
)

// Class is a class a file declares.
type Class struct {
	Name       string      `json:"name"`
	Extends    []string    `json:"extends,omitempty"` // the classes it extends, by name
	Decorators []Decorator `json:"decorators,omitempty"`
	// Options are a Python class's keyword arguments — SQLModel's table=True — as written.
	Options map[string]string `json:"options,omitempty"`
	// Fields are its declared fields: a TypeScript property with its decorators, a Python
	// annotated assignment with the call that defines it (SQLModel's Field, SQLAlchemy's Column).
	Fields []Field `json:"fields,omitempty"`
	// Injects are the types its constructor takes — in NestJS, what the container injects — and
	// Params the same parameters by name: this.albumRepository is an AlbumRepository.
	Injects []Literal  `json:"injects,omitempty"`
	Params  []Param    `json:"params,omitempty"`
	Methods []Method   `json:"methods,omitempty"`
	Prov    Provenance `json:"provenance"`
}

// Field is one declared field of a class. A Python field's defining call is recorded as its one
// decorator — Field(foreign_key="user.id") reads like @Column({ foreignKey: … }) — and Value is a
// plain literal it is assigned, as in __tablename__ = "users".
type Field struct {
	Name       string      `json:"name"`
	Type       string      `json:"type,omitempty"` // the annotation, as written
	Decorators []Decorator `json:"decorators,omitempty"`
	Value      string      `json:"value,omitempty"`
	Prov       Provenance  `json:"provenance"`
}

// Param is a constructor parameter: its name and its type.
type Param struct {
	Name string     `json:"name"`
	Type string     `json:"type"`
	Prov Provenance `json:"provenance"`
}

// Method is a method: its decorators — a route handler's, a job handler's — what it invokes on
// its own object, and the tables it queries. EndLine closes the span its calls are found in.
type Method struct {
	Name       string       `json:"name"`
	Decorators []Decorator  `json:"decorators,omitempty"`
	Invokes    []Invocation `json:"invokes,omitempty"`
	Queries    []Query      `json:"queries,omitempty"`
	EndLine    int          `json:"end_line,omitempty"`
	Prov       Provenance   `json:"provenance"`
}

// Invocation is a call a method makes on its own object: this.albumRepository.getAll(…) is Object
// albumRepository and Method getAll; this.requireAccess(…) has no Object.
type Invocation struct {
	Object string     `json:"object,omitempty"`
	Method string     `json:"method"`
	Prov   Provenance `json:"provenance"`
}

// Query is a table a method's query builder names: .selectFrom('album') reads album.
type Query struct {
	Table string     `json:"table"`
	Op    string     `json:"op"` // "reads", "writes", "updates", "deletes"
	Prov  Provenance `json:"provenance"`
}

// Decorator is one decorator as written: @Get(':id') is Get with argument ":id". Summary is a
// `summary:` the decorator's options state — Swagger's @ApiOperation, or a project's own — which
// is the code describing itself, cited, not interpretation.
type Decorator struct {
	Name    string `json:"name"`          // "Get", "Controller", "router.get"
	Arg     string `json:"arg,omitempty"` // the first argument, when it is a string literal
	HasArg  bool   `json:"has_arg,omitempty"`
	ArgExpr string `json:"arg_expr,omitempty"` // the first argument as written, when it is not
	// Target is the class a first argument like () => AssetTable names — a relation's other end.
	Target string `json:"target,omitempty"`
	// Options are the literal values of its options object or keyword arguments, as written:
	// { nullable: true, type: 'text' }, foreign_key="user.id".
	Options     map[string]string `json:"options,omitempty"`
	Summary     string            `json:"summary,omitempty"`
	SummaryProv Provenance        `json:"summary_provenance,omitempty"`
	Prov        Provenance        `json:"provenance"`
}

// Constant is a named string value: an enum member.
type Constant struct {
	Name  string     `json:"name"` // "RouteKey.Asset"
	Value string     `json:"value"`
	Prov  Provenance `json:"provenance"`
}

// Literal is a value as written, at its line.
type Literal struct {
	Value string     `json:"value"`
	Prov  Provenance `json:"provenance"`
}

// HostRef is a network location the code names: "http://immich-machine-learning:3003", or the
// 'redis' in `host: env.REDIS_HOSTNAME || 'redis'`.
type HostRef struct {
	Host   string     `json:"host"`
	Port   string     `json:"port,omitempty"`
	Scheme string     `json:"scheme,omitempty"`
	Value  string     `json:"value"`            // the literal
	Called bool       `json:"called,omitempty"` // the literal is the target of an HTTP call itself
	Prov   Provenance `json:"provenance"`
}

// Call is an outbound HTTP call — fetch, axios, requests — whose target is an expression, not a
// literal: "calls something at new URL('predict', url)".
type Call struct {
	Callee string     `json:"callee"` // "fetch", "axios.post", "requests.get"
	Target string     `json:"target"` // the expression, as written, shortened
	Prov   Provenance `json:"provenance"`
}

// Import is one thing a file imports, at the line that imports it.
type Import struct {
	Spec    string     `json:"spec"`              // as written: "./album.service", "src/utils/misc", ".core"
	Target  string     `json:"target,omitempty"`  // the repository file it resolves to
	Package string     `json:"package,omitempty"` // for ByPackage: "@nestjs/common", "fastapi"
	How     Resolution `json:"how"`
	Prov    Provenance `json:"provenance"`
}

// AppRole is what a manifest's package is, judged from what it depends on and declares.
type AppRole string

const (
	// RoleService is a server: a web framework, a worker. A C4 container.
	RoleService AppRole = "service"
	// RoleWeb is a front end served to a browser. A C4 container.
	RoleWeb AppRole = "web"
	// RoleMobile is an app installed on a phone. A C4 container.
	RoleMobile AppRole = "mobile"
	// RoleCLI is a command-line tool the repository ships. A C4 container.
	RoleCLI AppRole = "cli"
	// RoleLibrary is code other parts import; it runs inside them, not on its own.
	RoleLibrary AppRole = "library"
	// RoleTest is a test suite — end-to-end tests and their helpers.
	RoleTest AppRole = "test"
	// RoleDocs is a documentation site: about the system, not part of it.
	RoleDocs AppRole = "docs"
	// RoleWorkspace is a monorepo root that only gathers other packages.
	RoleWorkspace AppRole = "workspace"
	// RoleTooling is anything else: scripts, configuration, build helpers.
	RoleTooling AppRole = "tooling"
)

// Container reports whether an application of this role runs as a part of the system — a C4
// container — rather than something about it or inside it.
func (r AppRole) Container() bool {
	return r == RoleService || r == RoleWeb || r == RoleMobile || r == RoleCLI
}

// App is one manifest and what it says about the package it describes.
type App struct {
	Name     string `json:"name"`     // the manifest's package name, or its directory's
	Dir      string `json:"dir"`      // repository-relative; "." for the root
	Manifest string `json:"manifest"` // repository-relative
	Language string `json:"language,omitempty"`
	// Framework is what the role was judged from — NestJS, SvelteKit, FastAPI, Flutter — and
	// FrameworkProv is the line in the manifest that names it.
	Framework     string     `json:"framework,omitempty"`
	FrameworkProv Provenance `json:"framework_provenance,omitempty"`
	Role          AppRole    `json:"role"`
	Why           string     `json:"why"` // the role, in words: what decided it
	Prov          Provenance `json:"provenance"`
	// Deployed names the Compose service that runs this application, when some Compose file in
	// the repository builds that service from this application's directory; the build line is the
	// evidence. Without it, the application is a container of its own.
	Deployed *Deployment `json:"deployed,omitempty"`
}

// Deployment ties an application to the Compose service that runs it.
type Deployment struct {
	Service string     `json:"service"`
	Prov    Provenance `json:"provenance"`
	// ByName is set when no build line ties them and the match is an exact, unique name. The
	// provenance then cites the service and says so.
	ByName bool `json:"by_name,omitempty"`
}

// EnvSource is the dotenv file interpolation read, and whether it is a sample.
type EnvSource struct {
	File   string `json:"file"`
	Sample bool   `json:"sample"`
}

// Candidate is a file discovery looked at, and what it decided about it.
type Candidate struct {
	File   string `json:"file"`
	Reason string `json:"reason"` // why it was chosen, or why it was not
	Chosen bool   `json:"chosen"`
}
