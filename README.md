# ArchDoc

**Generates architecture documentation from a repository's own configuration and code — every
fact traceable to the file and line that proves it.**

---

## The problem

AI now writes most of the code while a person directs it. That works, and it is fast — but the
understanding that normally forms while typing never forms. Reviewing confirms the result
works; it does not build a mental model. Weeks later the person who directed every decision is
as lost as a stranger would be, with nobody to ask, because nobody ever knew.

Documentation debt assumes someone knew and failed to write it down. **This is knowledge that
never existed.**

## The approach

Point it at a repository. It reads the configuration — Compose files, `.env`, gateway configs —
and the code of each application it finds, with a real parser: files and imports, routes, table
classes, what calls what. From those it builds a validated model, and renders C4 diagrams, arc42
documentation and a web app where **every element links back to the line that proves it
exists**. Run it again after new commits and it reports what changed architecturally.

Two rules govern the design:

1. **The diagram is never the source of truth — a validated model is.**
2. **The LLM is never responsible for facts.** It supplies labels and prose on top of facts
   extraction already established. Remove it entirely and the tool still produces a complete,
   correct, traceable diagram — just with duller labels.

## Status

**R1.a is built, the surface was redesigned on 4–5 October 2026, and since 5 October archdoc reads
code as well as configuration** — TypeScript, JavaScript, Svelte, Python, Java and C#, and Dart
for its files and imports — with NestJS, Express, FastAPI, SvelteKit, Next.js, React Router,
TanStack Router, Spring and ASP.NET Core understood by their conventions, and tables read from
TypeORM-style classes, SQLModel, SQLAlchemy, Prisma schemas, JPA entities and EF Core contexts.
Since 9 October a Java module is found by its `pom.xml` or `build.gradle`, a .NET project by its
`.csproj`, and both are read with their `application.yml` or `appsettings.json`. What it does not read
yet it says so, in the coverage report. Where the work stands is in [`PROGRESS.md`](PROGRESS.md);
the reasoning is in [`docs/vision.md`](docs/vision.md). Delivery is 11 December 2026.

What it does now:

| | |
|---|---|
| `archdoc generate` | C4 context and container diagrams; inside each container whose code it reads, its components and its tables; every route, page, command and background job; what each route or job sets off; the packages each container depends on and where it uses them; twelve arc42 sections and a coverage report — every element cited at its line |
| `archdoc generate`, on something small | A folder of scripts or a single page, with no manifest, is documented from its files; a project of one application and a few files gets one page instead of twelve chapters |
| `archdoc explain` | Asks a model what each component does — the one of two commands that cost money, and it says what it will send and what that should cost, and asks, first (`--dry-run` prints the requests). It is sent names, paths and the code's own one-line route summaries, never code — logged as `structure-and-summaries`; every sentence must cite the facts it rests on or it is refused; answers are remembered, so nothing is asked twice. `--limit N` and `--only <id>` ask about fewer; a component of one file with no route or table is not asked about |
| `archdoc label` | The other: asks a model for descriptions and relationship labels, never structure — one request, about two cents on Immich. `--model` chooses the model |
| `archdoc status` | Whether the documentation still matches the code — the code is read again and compared by structure — and what is open: stale explanations, gaps, what the network has cost |
| `archdoc show <path> <thing>` | One element, route or file in the terminal, every line cited: a route with its flow as numbered steps, a component with what it uses and the tables it queries, a file with the component it is in |
| `archdoc scan` | What `generate` would find, written nowhere |
| `archdoc diff <path> <commit> [<commit>]` | What changed in the architecture between two commits, or since one, said as a reader would — "cache renamed to store"; `--detail` lists every change under its class: added, removed, renamed, re-bounded, protocol-changed, changed. Needs git |
| `archdoc init` | A `.archdoc/rules.yaml` to correct the model in, every line a comment until you write one |
| `archdoc serve` | The web app: the diagrams drawn interactively from context down to components and data, features and their flows, every value's citation, what changed between two versions, the documents, what archdoc could not see |
| `archdoc export --site` | The same app as a static site a team opens without archdoc — GitHub Pages, any static host |

Every command explains itself — `archdoc help <command>` — and exits 0 when done, 1 when something
failed, 2 when used wrongly. Only `label` and `explain` use the network; every other command is
offline, the ones that write need a path, and the ones that read write nothing.

On Immich, at the revision the survey pins:

```console
$ archdoc generate ./immich
…
16 elements, 11 relationships, from docker/docker-compose.yml
122 components in 6 containers, 964 uses between them, from the code
68 tables in 1 container, 65 foreign keys between them, from the code
304 routes, 55 pages, 18 commands, 66 jobs, and 11 calls whose target is computed at run time
7 section(s) created for you to write — see docs/architecture/index.generated.md
11 gap(s) — 'archdoc generate ./immich --gaps' lists them
```

Twelve arc42 sections: five filled from facts and overwritten every run, seven created once with
questions derived from *this* model and then never read or written again. The building block
view carries the container diagram and the evidence for it:

| Element | Type | Technology | Declared at |
|---|---|---|---|
| User | Person | — | `docker/docker-compose.yml:26` |
| immich-web | Container | SvelteKit · TypeScript <sup>`web/package.json:80`</sup> | `web/package.json:2` |
| immich-machine-learning | Container | FastAPI · Python <sup>`machine-learning/pyproject.toml:10`</sup> | `docker/docker-compose.yml:34` |
| immich-server | Container | NestJS · TypeScript <sup>`server/package.json:44`</sup> | `docker/docker-compose.yml:13` |
| database | Container (data store) | PostgreSQL 14 <sup>`catalog: postgres`</sup> | `docker/docker-compose.yml:57` |
| redis | Container (data store) | Valkey 9 <sup>`catalog: valkey`</sup> | `docker/docker-compose.yml:50` |

| From | To | Relationship | Declared at |
|---|---|---|---|
| immich-server | immich-machine-learning | calls, http | `server/src/dtos/config.dto.ts:624` |
| immich-server | database | connects to | `docker/docker-compose.yml:29`, `server/src/repositories/config.repository.ts:237` |

Every value carries the citation for *that value*: a box is proven by the line declaring it, its
framework by the line of the manifest that names it, a data store's product by a lookup table that
says so. Open any of those lines and the fact is there. That is the whole claim.

A component here is not a folder. Immich's server files are named by what they are for and what
they do — `album.controller.ts`, `album.service.ts`, `album.repository.ts`, `album.table.ts` — so
`album` is a component: 13 files across five folders, 13 routes, six tables. The folders are still
there, as a second view of the same code.

The second table is why code matters. Immich's configuration never says the server talks to machine
learning — it reaches it over a URL whose default is written in TypeScript. Reading configuration
alone, archdoc drew that box unconnected and said so; reading the code, it draws the arrow and cites
the line. And where the code calls an address it computes at run time, the call is listed as
unresolved rather than given an arrow nothing supports.

**No language model is involved in any of the above.** Routes, tables, flows and their descriptions
come from the code itself — a route's description is the summary its own decorator states. A model
is asked only by `archdoc label` or `archdoc explain`, only for wording, and what it writes is marked
as interpretation wherever it appears.

### Why discovery is harder than a glob

```console
$ archdoc scan ./immich --explain

Discovery:
    .devcontainer/mobile/container-compose-overrides.yml  deployable — 2 services with an image or build
    .devcontainer/server/container-compose-overrides.yml  deployable — 2 services with an image or build
    docker/docker-compose.dev.yml                         deployable — 6 services with an image or build
    docker/docker-compose.prod.yml                        deployable — 6 services with an image or build
    docker/docker-compose.rootless.yml                    deployable — 4 services with an image or build
  → docker/docker-compose.yml                             selected — deployable — 4 services with an image or build
    docker/hwaccel.ml.yml                                 fragment — no service declares an image or a build
    docker/hwaccel.transcoding.yml                        fragment — no service declares an image or a build
    e2e/docker-compose.dev.yml                            fragment — no service declares an image or a build
    e2e/docker-compose.yml                                deployable — 4 services with an image or build
```

Immich contains **ten** Compose files describing four different systems. A filename glob misses
the two under `.devcontainer/` — the glob written for this project's own survey missed them.
Content-sniffing finds them but then admits `hwaccel.ml.yml`, whose "services" are named `cpu`,
`armnn` and `rknn` and are hardware snippets rather than anything deployable.

Recall comes from sniffing; precision comes from rejecting services that declare neither an
image nor a build. `--explain` exists because *"trust me, I picked the right file"* is not good
enough for a tool whose entire premise is traceability.

## Built on evidence

Before any code was written, two production repositories were read and measured — self-hosted
[Immich](https://github.com/immich-app/immich) and
[Supabase](https://github.com/supabase/supabase), at pinned revisions. Findings that changed
the design:

- **Immich declares none of its three service connections in configuration.** All three live in
  TypeScript source. A configuration-driven extractor misses them — which is what led to reading
  the code: archdoc now finds all three there, each at its line.
- **Compose needs a specification-grade parser.** Seven parameter-expansion forms, nesting,
  concatenation, and the `!override` / `!reset` tags — where mishandling `!override` appends a
  list instead of replacing it and yields a *silently wrong* diagram.
- **`depends_on` is not an edge source.** It captures 1 of 7 gateway edges in Supabase.

The full survey is in [`docs/survey-test-subjects.md`](docs/survey-test-subjects.md).

## Building

Requires Go 1.27 and, to build the web app, Node 24. The app is compiled into the binary, so
*using* archdoc needs neither — only building it does.

```console
git clone https://github.com/cruzambrociogl/archdoc.git
cd archdoc
(cd web && npm ci && npm run build)   # the web app, embedded into the binary
go test ./...
go build -o ./archdoc ./cmd/archdoc
```

The built binary is git-ignored, so it can sit in the working copy. A binary built without the web
step still works on the command line; `serve` then says what to run. Working on the app itself,
`go build -tags dev` proxies to Vite (`cd web && npm run dev`) so a change shows on reload —
[`web/README.md`](web/README.md) has the rest.

## Running it on the test subjects

The three subjects are **not** in this repository — they are other people's code. Clone them
yourself, at the revisions pinned in
[`docs/survey-test-subjects.md`](docs/survey-test-subjects.md) §Method, and keep them beside
this directory rather than inside it:

```
University/SP2/
├── archdoc/          this repository
└── subjects/
    ├── immich/
    ├── mastodon/
    └── supabase/
```

**Look before writing.** `--stdout` prints the whole document and touches nothing:

```console
./archdoc generate ../subjects/immich   --stdout
./archdoc generate ../subjects/mastodon --stdout
./archdoc generate ../subjects/supabase --stdout
```

**Then write it in,** which is what a real user would run:

```console
./archdoc generate ../subjects/supabase
```

```
docs/architecture/
├── index.generated.md                  every diagram, and links to everything below
├── 01-introduction-and-goals.md        yours — questions, not a blank template
├── 03-context-and-scope.generated.md   regenerated every run
├── 05-building-block-view.generated.md
├── 06-runtime-view.generated.md        flows drawn from the code, where code was read
├── 07-deployment-view.generated.md
├── 12-glossary.generated.md
├── components.generated.md             a page per component
├── features.generated.md               every route and page
├── coverage.generated.md               what archdoc read, and what it could not resolve
├── 02, 04, 08–11                       yours
├── context.svg · container.svg         the drawn C4 diagrams, as you arranged them
├── component-*.svg · data-*.svg        a container's components, and its tables
└── *.mmd                               the same diagrams, as editable text
.archdoc/
├── model.json                          committed — the durable record
├── coverage.json                       committed — the coverage report, as data
├── interpretations.json                committed — a model's answers, remembered by the facts they were given
├── rules.yaml                          committed — your corrections, if any
├── layout.yaml · views.yaml            committed — arrangements and views saved in the app
├── history.db                          local cache, git-ignored by archdoc
└── site/                               the published site, git-ignored — CI builds it
```

Generated files archdoc no longer writes are removed on the next run; nothing else ever is.

To see the diagrams, open `index.generated.md` in VS Code and press `⇧⌘V` — Mermaid renders
natively, with no build step and no site generator.

**Write a sentence into any file without `.generated.` in its name, then regenerate.** It will
still be there. That boundary is the difference between a tool people keep and one they
uninstall.

**Start with Supabase.** It is the only subject with an external system, the only one with an
excluded gateway, and the only one where the distinction between "declares a dependency" and
"declares that traffic flows" is visible.

### Every command, on Immich

The whole CLI, run from this directory against the Immich clone, in the order a person meets it.
The output is real (9 October, Immich pinned at `cbf5d83`), shortened where marked `…`.
`./archdoc help` lists the commands; `./archdoc help <command>` shows one command's flags.

**Preview first.** `scan` reads the repository exactly as `generate` will, and writes nothing:

```console
$ ./archdoc scan ../subjects/immich
immich — …/subjects/immich

APPLICATION              ROLE       BUILT ON    WHERE
immich-ml                service    FastAPI     machine-learning
immich_mobile            mobile     Flutter     mobile
@immich/cli              cli        —           packages/cli
immich                   service    NestJS      server
immich-web               web        SvelteKit   web
…                                                              (15 manifests in all)

4 services in docker/docker-compose.yml
15 elements, 17 relationships, from docker/docker-compose.yml
179 components in 5 containers, 1694 uses between them, from the code
68 tables in 1 container, 65 foreign keys between them, from the code
304 routes, 55 pages, 17 commands, 66 jobs, and 11 calls whose target is computed at run time
364 flows followed through the code from them

Nothing written. 'archdoc generate ../subjects/immich' writes it.
```

**Where corrections go.** `init` creates `.archdoc/rules.yaml`, every line a comment until you write
a rule; it never overwrites anything:

```console
$ ./archdoc init ../subjects/immich
created .archdoc/rules.yaml — corrections that survive regeneration; every line is a comment until you write one
kept    .archdoc/.gitignore — already there
```

**Write it.** `generate` is offline and free, always. Descriptions and explanations a model wrote
before are reused while what they describe is unchanged:

```console
$ ./archdoc generate ../subjects/immich
documenting …/subjects/immich
wrote 41 files into docs/architecture and .archdoc — --verbose lists them
architecture unchanged since version 30

15 elements, 17 relationships, from docker/docker-compose.yml
179 components in 5 containers, 1694 uses between them, from the code
68 tables in 1 container, 65 foreign keys between them, from the code
304 routes, 55 pages, 17 commands, 66 jobs, and 11 calls whose target is computed at run time
364 flows followed through the code from them
10 gap(s) — 'archdoc generate ../subjects/immich --gaps' lists them
```

**Is it still true?** `status` reads the code again and compares it with what was documented, by
structure: a description a model reworded is not the code changing.

```console
$ ./archdoc status ../subjects/immich
immich — …/subjects/immich

documented     version 30, recorded 9 Oct 2026 01:10 at commit cbf5d83
the code now   at commit cbf5d83 — the architecture is as documented
descriptions   15 written by a model, marked as such — 'archdoc label ../subjects/immich' asks again
explanations   165 of 179 components, 8 of them about facts that have since changed
gaps           10 — 'archdoc generate ../subjects/immich --gaps' lists them
network        11 run(s), $3.29 in all — 'archdoc runs ../subjects/immich' lists them
```

**Ask about one thing.** `show` answers from the committed model: a route with its flow as numbered
steps, a component, a table and who queries it, or a file and the component it belongs to. Every
line carries its citation; what a model wrote is marked `◇`.

```console
$ ./archdoc show ../subjects/immich "POST /api/assets"
POST /api/assets  [http in immich-server]
  "Upload asset" — the code's own description, server/src/controllers/asset-media.controller.ts:72
  handled by AssetMediaController.uploadAsset, declared at server/src/controllers/asset-media.controller.ts:51
  in component asset

flow — 40 step(s) through 27 participant(s), followed by name through the code:
  1  AssetMediaController → AssetMediaService.uploadAsset             server/src/controllers/asset-media.controller.ts:83
  2    AssetMediaService → AssetMediaService.requireAccess            server/src/services/asset-media.service.ts:133
  3    AssetMediaService → AssetMediaService.requireQuota             server/src/services/asset-media.service.ts:140
  4    AssetMediaService → AssetRepository.create                     server/src/services/asset-media.service.ts:149
  5      AssetRepository writes asset                                 server/src/repositories/asset.repository.ts:448
  …
 21    AssetMediaService queues AssetExtractMetadata                  server/src/services/asset-media.service.ts:187
       handled later by MetadataService.handleMetadataExtraction, a flow of its own
  …

$ ./archdoc show ../subjects/immich album
"album" matches 4 things — name one:

  cmp:immich-server/album                  component in immich-server
  cmp:mobile/album                         component in immich_mobile
  cmp:web/album                            component in immich-web
  tbl:immich-server/album                  table in immich-server

$ ./archdoc show ../subjects/immich tbl:immich-server/album
album  [table in immich-server]
  id tbl:immich-server/album · declared at server/src/schema/tables/album.table.ts:18
  …
queried by:
  reads    album                                      7× · first at server/src/repositories/album.repository.ts:91
  updates  album                                      4× · first at server/src/repositories/album.repository.ts:236
  writes   album                                      1× · first at server/src/repositories/album.repository.ts:325
  …

$ ./archdoc show ../subjects/immich server/src/services/album.service.ts
server/src/services/album.service.ts
  in component album, component in immich-server
  …
```

**What changed.** `diff` reads the repository at a commit — and at another, or as the files are now
— and says what changed in the architecture. Immich's clone is one commit deep, so here it has
nothing to say; on two commits it reads like "cache renamed to store", and `--detail` lists every
change under its class:

```console
$ ./archdoc diff ../subjects/immich HEAD
HEAD → the working tree

no change to the architecture
```

**Ask Claude — the only two commands that cost money.** Each says what it will send and what that
should cost, and asks; with no one to ask it sends nothing unless given `--yes`. `--dry-run` prints
every request exactly as it would go, and stops:

```console
$ ./archdoc label ../subjects/immich
documenting …/subjects/immich
label: 1 request(s) to claude-sonnet-5-5, 7483 bytes — about $0.02, estimated from its size.
Sends names, kinds, technologies and relationships of the containers — no path, no code. A corrected answer is asked for again and costs more.
Send? [y/N]

$ ./archdoc label ../subjects/immich --dry-run
── request 1 of 1 · system · 7483 bytes ──
[system]
You label software architecture models for documentation.
…
dry run — 1 request(s) would go to claude-sonnet-5-5, 7483 bytes, about $0.02, estimated from its size. Nothing sent, nothing written.

$ ./archdoc explain ../subjects/immich --dry-run
documenting …/subjects/immich
nothing to ask: every component has a remembered answer for its facts, or is too small to ask about
dry run — nothing sent, nothing written.
```

Every component of Immich worth asking about has a remembered answer, so `explain` has nothing to
send; on a repository explained for the first time it lists one request per component, and
`--only <id>` or `--limit <n>` asks about fewer.

**What was sent, and what it cost.** `runs` lists every run that used the network; `--show <run>`
prints its requests byte for byte, and the answers:

```console
$ ./archdoc runs ../subjects/immich -n 3
RUN  WHEN              MODE                     REQUESTS  BYTES SENT  TOKENS IN/OUT   COST     STATUS
11   2026-10-09 07:14  structure-and-summaries  91        270139      107916 / 22358  $0.4394  ok
10   2026-10-09 07:04  structure-only           1         7055        2555 / 1290     $0.0180  ok
9    2026-10-09 06:53  structure-and-summaries  3         19799       7747 / 936      $0.0249  ok

11 run(s), $3.29 in all — the latest 3 listed; -n shows more
'archdoc runs ../subjects/immich --show <run>' prints exactly what a run sent.
```

**Every version.** A run that changes nothing records nothing, so this is the number of times the
architecture moved:

```console
$ ./archdoc history ../subjects/immich -n 3
VERSION  RECORDED          COMMIT        SOURCE
30       2026-10-09 07:10  cbf5d83a693d  docker/docker-compose.yml
29       2026-10-09 07:09  cbf5d83a693d  docker/docker-compose.yml
28       2026-10-09 07:09  cbf5d83a693d  docker/docker-compose.yml

The latest 3 of 30 versions — -n shows more. A run that changes nothing records nothing.
```

**Look at it, and share it.** `serve` opens the app on this machine only; `export --site` builds the
same app as a static site for a team:

```console
$ ./archdoc serve ../subjects/immich --open
archdoc serving ../subjects/immich at http://localhost:7474 — this machine only. Ctrl-C to stop.

$ ./archdoc export ../subjects/immich --site
```

Scripts and coding agents get `--json` from every command that reads, and exit codes they can test:
0 done, 1 failed, 2 used wrongly.

### Checking it rather than trusting it

```console
./archdoc scan ../subjects/immich --considered  # why that Compose file, and not the other nine
./archdoc scan ../subjects/supabase --facts     # the raw FactSet, every fact with its line
./archdoc show ../subjects/immich "POST /api/assets"   # a route and its flow, each step cited
./archdoc status ../subjects/immich             # still matches the code? what is open?
./archdoc history ../subjects/supabase          # every version, and when the architecture moved
```

Pick any citation from an evidence table and open it. Supabase's
`docker/docker-compose.yml:166` reads `GOTRUE_SMTP_HOST: ${SMTP_HOST}` — the diagram shows
`supabase-mail`, resolved from `.env`, and the citation still lands on text a person can read.
That is the two-pass extractor working: one pass knows what is true, the other knows where it
was written.

**History** — a run that changes nothing records nothing, so the version count is the number of
times the architecture actually moved:

```console
./archdoc generate ../subjects/supabase   # recorded version 1
./archdoc generate ../subjects/supabase   # architecture unchanged since version 1
./archdoc history ../subjects/supabase
```

**Determinism** — five runs must be byte-identical (AC-7):

```console
for i in 1 2 3 4 5; do ./archdoc generate ../subjects/supabase --stdout | md5; done | sort -u
```

One line of output means five identical runs.

### Exploring it in the app

```console
./archdoc serve ../subjects/supabase      # http://localhost:7474 — this machine only
```

| Screen | What it answers |
|---|---|
| **Overview** | What the system is, how much of it is interpreted, what has ever left the machine, what changed |
| **Explorer** | The diagrams, drawn from the engine's stored layout. Click a box or an arrow: every value with the line that proves it. `/` finds, *Focus* dims everything but the neighbours, *Compare with…* marks what changed since another version. Drag boxes into a readable arrangement and *Save* — it goes into `.archdoc/layout.yaml` and the committed SVG follows. *Save view…* names what you are looking at |
| **Changes** | Any two versions side by side — structural change kept apart from changed words |
| **Documents** | The arc42 sections. Generated ones render here; yours are never opened — the app shows their state and, for one nobody has started, the questions this system raises |
| **Coverage** | What archdoc could not see: every gap by rule, what nothing connects to, the files it passed over, and the limits of reading configuration at all |
| **Corrections** | What `rules.yaml` changed, and the rules that matched nothing |
| **Network runs** | Every request that ever left the machine, exactly as sent |

`⌘K` searches everything — type a file path to see what it proves. `?` lists the shortcuts.

### Correcting it

A correction is a rule in `.archdoc/rules.yaml`, applied on every run, so it survives
regeneration — and a person's correction always wins over the model's:

```yaml
rules:
  - match: { name: supavisor }
    set: { kind: proxy, description: Pools Postgres connections }
  - match: { image: "*/vector*" }
    exclude: true
```

Every value a rule sets cites the line that set it. (A `rules.yaml` at the repository root, where it
lived before 5 October, is still read; archdoc says to move it.)

### Publishing it

```console
./archdoc export --site ../subjects/supabase
python3 -m http.server 8080 -d ../subjects/supabase/.archdoc/site    # preview
```

The published site is the same app with no controls: one version, what changed since the previous
one (`--since <version>` to choose), citations opening on GitHub at the commit when the repository
has a GitHub, GitLab, Codeberg or Bitbucket origin. It needs a host — browsers will not load it from
a `file://` page. It builds from the committed `.archdoc/model.json` when there is no history, so CI
needs no key and regenerates nothing. A GitHub Pages workflow, in the documented repository:

```yaml
name: Architecture site
on: { push: { branches: [main] } }
permissions: { contents: read, pages: write, id-token: write }
jobs:
  site:
    runs-on: ubuntu-latest
    environment: { name: github-pages, url: "${{ steps.deploy.outputs.page_url }}" }
    steps:
      - uses: actions/checkout@v4
      - uses: actions/checkout@v4
        with: { repository: cruzambrociogl/archdoc, path: .archdoc-tool }
      - uses: actions/setup-go@v5
        with: { go-version-file: .archdoc-tool/go.mod }
      - uses: actions/setup-node@v4
        with: { node-version: 24 }
      - run: cd .archdoc-tool/web && npm ci && npm run build
      - run: cd .archdoc-tool && go build -o "$RUNNER_TEMP/archdoc" ./cmd/archdoc
      - run: '"$RUNNER_TEMP/archdoc" export --site .'
      - uses: actions/upload-pages-artifact@v3
        with: { path: .archdoc/site }
      - id: deploy
        uses: actions/deploy-pages@v4
```

### What each subject shows

| | What to look for |
|---|---|
| **Immich** | `immich-machine-learning` has **no edges at all**. Every service reaches it over a URL built at runtime, so configuration never declares the link. Correct, and the finding the whole survey turns on |
| **Mastodon** | A clean five-container diagram, and an empty context diagram. Its external dependencies are real and live in `.env.production.sample`, which the compose file references and which is not in the repository |
| **Supabase** | `api-gw` excluded as infrastructure and named under *Not shown*; `supabase-mail` as a referenced external system; relationships labelled `connects to [postgres]` where a URL knew the protocol and `depends_on` did not |

One line covers all three: **the container view is as good as the compose file, and the context
view is as good as the environment — which is usually somewhere else.**

## Documentation

| | |
|---|---|
| [`docs/how-it-works.md`](docs/how-it-works.md) | **The pipeline on one screen** — start here |
| [`docs/product-definition.md`](docs/product-definition.md) | What the product is — the capabilities, the output contract, nine acceptance criteria. Technology-free by design |
| [`docs/surface-spec.md`](docs/surface-spec.md) | The web app and the published site — every screen, the design system, the two modes, what the app may do |
| [`web/README.md`](web/README.md) | Working on the web app |
| [`docs/vision.md`](docs/vision.md) | Where archdoc goes next: the code as evidence |
| [`docs/stack-decision.md`](docs/stack-decision.md) | What it is built with, and why, including the rejected alternatives |
| [`docs/survey-test-subjects.md`](docs/survey-test-subjects.md) | The evidence both rest on |
| [`docs/delivery-schedule.md`](docs/delivery-schedule.md) | The plan through 11 December |
| [`docs/decisions.md`](docs/decisions.md) | Decision log — read before proposing anything that seems obvious |
| [`PROGRESS.md`](PROGRESS.md) | What is done, and which gate is next |

New to the repository? [`CLAUDE.md`](CLAUDE.md) opens with a reading order.

---

*Seminar project · Universidad Galileo · 2026*
