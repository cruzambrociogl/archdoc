# Progress

Tracked against the capability catalog (`docs/product-definition.md` §4) and the acceptance
criteria (§12). These are the project's own work breakdown — no parallel TODO list.

**Status key:** `·` not started · `~` in progress · `✓` done · `⊘` superseded, see note

Last updated: 2026-10-09 (end of the 8–9 Oct session — see "Where this stands" below)

---

## Where this stands — 9 Oct 2026, to resume from

Everything is on `develop`, pushed, 193 commits ahead of `main`. `main` holds the initial commit
only; no pull request has been opened, by choice. `go test ./internal/... ./cmd/...` passes; the
web app builds (`cd web && npm run build`), and `go build -o archdoc ./cmd/archdoc` embeds it.

**What the product does now**

| | State |
|---|---|
| A system (Immich) | Context, containers, components, data, features, flows, dependencies and cited explanations, all working. The explorer reads as a C4 picture at all three levels — by inspection, not by a measured score |
| Something small | A folder of scripts, a single page, or a small app is documented on one page and told as a story. Since 9 Oct; before that a project with no manifest was refused |
| "What the AI did" | Not built. `archdoc diff` compares two commits and classifies every change; nothing yet turns that into a summary a person reads, and nothing reads a plan file |

**Acceptance criteria:** seven of nine met — AC-1, 2, 4, 5, 6, 7, 8. AC-3 waits for the hand-drawn
Immich reference (Cruz). AC-9 waits for Supabase, which is not cloned.

**Against the vision** (`docs/vision.md`; the check itself is in `docs/decisions.md`, 9 Oct): on
path for a system; back on path for small projects; not started on the change view per session and
on plan against code. The order agreed: (1) small projects accepted ✓, (2) the small app right ✓,
(3) a change summary a person reads, built on `archdoc diff`, (4) plan against code. **3 is next.**

**Test subjects on this machine** — in `../subjects/`, outside the repository:

| Subject | State |
|---|---|
| `immich` | Pinned `cbf5d83a693d0328282ddb0f5d398c351d92558e`, one commit deep. Generated, labelled, explained: 15 elements, 17 relationships, 179 components, 68 tables, 442 ways in, 165 explanations, version 30 |
| `small-script`, `small-page`, `small-app` | Written for the purpose on 9 Oct, generated, never labelled or explained |
| Mastodon, the FastAPI template, Supabase | Not on this machine. The first two were deleted; Supabase was never cloned |

**Money spent on Immich so far:** $3.29 across eleven runs, logged in its `.archdoc/history.db`
(`archdoc runs ../subjects/immich`), plus $0.02 on a throwaway copy. The first two runs, on Claude
Opus 5 before the prompt and the model were changed, are $2.24 of it. A full `--explain` of Immich
on `claude-sonnet-5-5` is about $0.45 and a `--label` about $0.02. **Paid runs are asked for first.**

**To see it**

```
cd archdoc && go build -o archdoc ./cmd/archdoc
./archdoc serve ../subjects/immich          # http://localhost:7474
./archdoc serve ../subjects/small-page      # the small-project story
./archdoc diff ../subjects/immich HEAD      # what changed since a commit
```

**Designed and not built:** 34 pieces of the surface spec and the Claude Design files — the §11 controls, a flows index,
deployment, the correction composer, commits and session summaries in the app, ask the map, intent
vs actual — are dashed *planned* placeholders in the local app, listed on its "Not built yet" page
from `web/src/planned.ts`. That file is the list; this paragraph only points at it.

**Where to read what happened:** the tables under "Code as evidence" below list every piece of work
of 8–9 Oct with what it measured and its commit; `docs/decisions.md` has the reason for each, dated;
"Not done" at the end of that section is the full list of what is open.

---

## Acceptance criteria — the definition of done

Four of these need no human judgement and no external subjects. They are the ones to run
constantly, not just at a review.

| | Criterion | Threshold | Self-checking | Status |
|---|---|---|---|---|
| AC-1 | Nodes and edges carry extraction or catalog provenance | ≥ 95% | no | ✓ **100% on Immich with its code read — 1,333 of 1,333 elements and relationships, 9 Oct**, and on every fixture but the one where a rule adds an element (4 of 5, as it should be). Counted on every run now: the first row of the coverage report. The validator already refuses an element nothing vouches for (VAL-05), so what this number can show is the share a rule or a model supplied; `--label` and `--explain` add values and sentences, never elements, and leave it at 100% |
| AC-2 | Container diagram produced with the LLM disabled | renders + validates, both subjects | **yes** | ✓ **met, 11 Sep.** Renders and validates on all three subjects with no model involved. The live on/off comparison held on Supabase across four versions: identical elements, kinds and relationships — only words changed |
| AC-3 | Structural accuracy vs hand-drawn reference | ≥ 0.85 on Immich | no — needs the reference | · **reference due 5–16 Oct (Cruz)**, scored in the evidence sprint 20–24 Oct. Also the answer key for the Code Wiki / DeepWiki comparison |
| AC-4 | Validator rejects malformed models | 100% of fault-injection suite | **yes** | ✓ **13 of 13 rejected, 8 Sep.** Every fault in the suite is a way the draw.io experiment failed, or a way the model could lie without a reader noticing. Grow the suite as new failure modes appear |
| AC-5 | Rule persistence | 10 rules survive regeneration | **yes** | ✓ **10 of 10, 8 Sep.** Measured by running the whole pipeline from disk twice, not by re-applying a cached model — surviving *regeneration* is the criterion |
| AC-6 | Drift detection | 100% of synthetic drift set | **yes** | ✓ **16 of 16 detected and classified, 9 Oct.** `archdoc diff <path> <commit> [<commit>]` (F-37) reads the repository at each revision; the drift set (F-39, `cmd/archdoc/drift_test.go` over `testdata/drift/base`) changes it one way per case — services, dependencies, protocols, routes, columns, renames, a boundary crossing, and a change that is none — and each must be reported under its class with nothing else beside it |
| AC-7 | Determinism | 5 runs, byte-identical FactSets | **yes** | ✓ **holds end to end** — 5 identical runs on all three subjects, measured on the full generated document rather than the FactSet alone. Covered by tests in four packages |
| AC-8 | Egress | `structure-only` transmits zero file contents | **yes** | ✓ **met, 12 Sep, for `--label`**; measured on the wire: the real SDK against a local fake of the API, the recorder holding exactly the bytes the server received, with no path, line, provenance or key among them. **`--explain` is a second mode, named as one since 9 Oct** — `structure-and-summaries`: it also sends paths and each route's own one-line summary, which is text from a file, by decision (8 Oct). Tested on the wire the same way: the summaries arrive, and no line number, column type, description or key does. Until 9 Oct its runs were logged as `structure-only` and announced as "names only", which was not true |
| AC-9 | Performance | NFR-1 and NFR-2 met on Supabase | no | · scheduled — 5–8 Oct, free to run; re-measured on a code-heavy repo after Build 1 |

> **AC-3 is predicted to miss its threshold.** Immich declares none of its three service
> connections in configuration — all live in TypeScript source. Reporting that as a measured
> limitation of declaration-based extraction is the intended outcome, not a defect to fix.
> See `docs/delivery-schedule.md` §7.

---

## Capabilities

| Group | What it covers | Count | Done | Gate |
|---|---|---|---|---|
| `DSC` | Discovery — which files are the architecture | 4 | ~3 | — |
| `EXT` | Extraction — parsers, provenance, the FactSet | 15 | ~7 | AC-7 |
| `MDL` | Model construction — nodes, edges, boundaries, identity | 17 | ~11 | — |
| `VAL` | Validation | 8 | ~8 | AC-4 |
| `RUL` | Rules — `rules.yaml` | 6 | ~6 | AC-5 |
| `SEM` | Semantic layer — the LLM | 10 | ~5 | — |
| `PRV` | Provenance | 6 | ~3 | AC-1 |
| `MEM` | Memory and diff | 8 | ~7 | AC-6 ✓ — versions compare in the app; `archdoc diff` between two commits; changes classified, renames and boundary crossings among them (9 Oct). MEM-08, the optional prose, is not built |
| `VIE` | Views and rendering | 10 | ~7 | AC-2 |
| `SUR` | Surfaces — CLI and web app | 22 | ~19 | AC-8 — web views redesigned and SUR-16–22 added and done 5 Oct (see below); `diff` built 9 Oct; `init` and `export` formats other than `--site` remain |
| `OUT` | Output and deliverables | 12 | ~11 | — coverage report and the MkDocs site done 2 Oct; OUT-11 published site and OUT-12 coverage as data 5 Oct; OUT-09 from size and mtime (no conformance % yet); OUT-10 approximated by mtime, not by commit |
| `ANS` | Answer surface | 7 | 0 | — *(R1.c, stretch)* |
| | **Total** | **125** | **~84** | |

109 are R1.a; the 7 `ANS` capabilities are R1.c. The counts were last tallied on 5 Oct; only the `MEM` and `SUR` rows have been corrected since, and most of 8–9 Oct's work is in the feature tables below, which are keyed on the inventory's F-numbers, not on this catalog.

### Surface redesign — 5 Oct, pulled forward from Build 3

Decided 4–5 Oct (`docs/decisions.md`, S-1 to S-7, C-1 to C-5); specified in `docs/surface-spec.md`;
designed in Claude Design. Built on `develop`, every screen driven in a browser.

| Feature | What shipped |
|---|---|
| F-25 web app at scale | ✓ explorer drawn with React Flow from the stored scene; find, focus, legend, minimap, zoom; checked at 300 elements, 60 fps. Upstream/downstream reach and path probe remain |
| F-55 clickable relationships | ✓ arrows selectable, with their own passport |
| F-22 publishable site | ✓ `archdoc export --site` — the same app, published mode, citations on the repository host |
| F-24 coverage report | ✓ also as `.archdoc/coverage.json` and a screen |
| S-6, S-7 | ✓ arrangements in `layout.yaml` applied to app and SVG; saved views in `views.yaml` |
| §5.12 change lens | ✓ comparison screen and overlay on the diagram |
| §5.17 search | ✓ ⌘K over elements, relationships, files, documents, views |
| §11.5 action gate | ✓ Origin + session token on every action |
| §11 run, publish, git, settings controls; correction composer (C-2) | · not started |

### Loose ends closed — 5 Oct

| | |
|---|---|
| F-50 | ✓ a version is a change of architecture; the fingerprint is recomputed, so upgrades mint nothing |
| F-51 | ✓ a same-architecture run refreshes the version's citations, layouts and commit |
| F-52 | ✓ generated files archdoc no longer writes are removed |
| F-53 | ✓ the interpolation dotenv file is reported, and a sample is called one |
| `rules.yaml` | ✓ moved to `.archdoc/`; the root one is still read, with a note |
| Completeness | ✓ "unknown" where archdoc has no record of the stub; a published site judged by size |
| Export in CI | ✓ builds from the committed `model.json` when there is no history |

### Code as evidence — the working plan, from 5 Oct

The vision's build order (`vision.md` §2.2, D-2 to D-8 as recommended there), as vertical slices:
each runs from parser to screen and ends with something visible in the app. Measured on Immich and
the FastAPI template. Worked through in order, without dates, until the 9 Oct demo is called.

| # | Slice | Features | Status |
|---|---|---|---|
| 0 | Setup; parsing inside the binary — gotreesitter vs. typescript-go, measured on Immich | F-01 | ✓ 5 Oct — gotreesitter, five grammar packages |
| 1 | Applications from manifests; Compose as deployment evidence | F-02 | ✓ 5 Oct — Immich: server and ML tied by build line, web, mobile, CLI on their own |
| 2 | Module and import graph → components by convention; identity by what a thing is | F-03, F-10, F-31 | ✓ 5 Oct — Immich: 43 components in 5 containers, 145 uses, no import unresolved; explorer Components level, committed SVGs, coverage of the code |
| 3 | NestJS pack: routes, services, injection; outbound calls; unresolved as a state | F-04, F-08, F-13, F-32 | ✓ 5 Oct — Immich: 301 routes (274 described by the code), server → ML from `config.dto.ts:624`, 11 calls unresolved; Features screen and `features.generated.md` |
| 4 | Data model from table classes and migrations | F-07, F-11 | ✓ 5 Oct — from table classes (TypeORM-style decorators, SQLModel, SQLAlchemy); Immich: 68 tables, 65 foreign keys, in PostgreSQL; explorer Data level, `data-*.svg`, Mermaid erDiagram. SQL migrations read since 8 Oct, where the code declares no table |
| 5 | Flows from an entry point, as sequence diagrams | F-12 | ✓ 5 Oct — 298 of Immich's 301 routes followed to tables and outbound calls; sequence diagram per route on Features; arc42 §6 drawn from the code (eight widest flows, by rule) |
| 6 | SvelteKit and React pages; FastAPI and Python | F-05, F-06 | ✓ 6 Oct — FastAPI routes with full prefixes (settings resolved by name, conditional includes marked) and Python flows; SvelteKit pages by file, TanStack by `createFileRoute`. Template: 23 routes, 8 pages, 18 flows; Immich: 304 routes, 55 pages |
| 7 | Component pages and cited claims, remembered — paid runs, asked first | F-19, F-30, F-36 | ✓ 8 Oct — `components.generated.md`, inspector, `--explain`, VAL-09, `.archdoc/interpretations.json`. Live on Immich: 42 of 43 components explained, 152 cited sentences, $1.52 (after a first run lost $0.72 of answers to one empty reply — fixed) |

**After the plan, 8 Oct** — built the same day the plan closed:

| What | Features |
|---|---|
| Changes and versions see routes, pages and columns; a run that changes nothing records nothing again (a network-less service had been minting a version every run) | F-37 in part |
| Rules for components, tables and routes (`in:`, `id:`, `route:`); excluding a container takes its parts with it | F-33 |
| Shared components said, not drawn — the component picture is readable | F-10 |
| Flows follow plain function calls that lead somewhere; jobs have flows | F-12 |
| Unresolved calls on the canvas, as a count on the box | F-32 |
| A page per component in the app; explanations outlive their facts, marked stale; the memory only grows | F-19, F-30 |
| Dependencies: manifest line, imports, components | F-14 |
| Express, Next.js, React Router; commands and jobs; Prisma and SQLAlchemy; Dart files and imports | F-04–F-07, F-13 |
| **Components are features, not folders**, where file names carry roles; folders as a second view; shared and wiring parts said, not drawn | F-10 |
| **A fuller context**: external systems from configured URLs and from known client libraries, never from links | F-08 |

**The evening of 8 Oct** — cost, then the three known gaps. Every row is on `develop`; the reasons
are in `docs/decisions.md` under the same date.

| What | Measured | Commit |
|---|---|---|
| `--explain` asks `claude-sonnet-5-5` at low effort; `--label` asks it too, at medium, and `--label-model` asks another | Per component $0.0145 on Opus 5 → about $0.005; `--label` on a copy of Immich: one request, 27 operations, $0.018 | `b096a87`, `ec4b283`, `2af9d48` |
| A component of one file with no route, page or table is not asked about | Immich: 25 of 122 skipped | `b096a87` |
| An answer in the agreed shape with no sentence in it is asked again, and named if it stays empty | Seen live: 3 of 83, silently unexplained before | `aaf486c` |
| Full `--explain` on Immich | 97 components explained, every sentence cited; $0.41 + $0.06 of tests + $0.02 for the three empty ones | runs 4–7 |
| `model.json` laid out compactly; an empty provenance is not written | Immich: 3.3 MB → 2.6 MB, same content | `6ae1b73` |
| `archdoc serve` gzips text | `/api/model`: 3.4 MB → 226 KB | `6ae1b73` |
| Flows follow an emitted event into its `@OnEvent` listeners, and a queued job to a lifeline that opens the job's own flow | Immich: 37 flows reach a listener, 68 a queued job | `bb29ba9` |
| Flows follow a Python class's methods where the code states the object's class | Immich: `/predict` reaches `InferenceModel.load` | `bb29ba9` |
| SQL migrations are the schema where the code declares no table, for a container in any language; coverage names the schema files | A Go service with only migrations gets a data view; Immich unchanged, byte for byte | `012c70b` |
| Fixed: a service whose manifest names no known framework failed validation and nothing was written | Any plain Go module | `012c70b` |

**9 Oct** — the acceptance criteria that could be scored without outside subjects:

| What | Measured | Commit |
|---|---|---|
| `archdoc diff` between two commits, or a commit and the working tree; needs git, the only command that does | Immich, one commit against its files: no change, 3 s | `7ddd3c9` |
| Renames and boundary crossings are one change (MEM-05, MEM-06): matched on a directory, relationships, columns or files, only when the match is unique; a container's parts and routes follow it | A renamed container with code: 1 change, not dozens | `7ddd3c9` |
| The drift set | AC-6: 16 of 16 | `7ddd3c9` |
| The Changes screen lists renames and crossings | — | `7ddd3c9` |
| AC-1 counted in the coverage report | Immich 1,333 of 1,333 | `05c0040` |
| `--explain` runs logged and announced as `structure-and-summaries`, with a wire test of what that mode sends | — | `05c0040` |

**9 Oct, later** — the explorer as a C4 picture of Immich, after looking at it level by level:

| What | Measured on Immich | Commit |
|---|---|---|
| Clients call the API they are clients of: an OpenAPI document, tied to a container by its routes, and the packages and generated directories that are clients of it | web, mobile and CLI → server, each cited; 274 of 274 operations matched | `9905416` |
| A person reaches web, mobile and command-line applications | 3 new arrows; context unchanged | `9905416` |
| A package that running code depends on is a library | `@immich/plugin-sdk` no longer a container: 15 elements, 17 relationships | `9905416` |
| A container with more than 24 components opens on its 16 main ones; all of them and by-folder one click away | server: 16 of 75; mobile, with 18, shows all | `187b7d1`, then `f988124` |
| Labels remembered in `.archdoc/labels.json` and applied on every run | `--label` on the clone: 26 operations, $0.018; a plain run after it keeps all 15 descriptions and records no version | `980e42f` |
| Element order made total (it depended on what else was in the model) | 9 explanations no longer stale for no reason | `980e42f` |
| Small diagrams keep technology and description when fitted; description text sized as the engine sizes boxes | container view readable at 82% | `980e42f` |
| The system box has a description; component boxes show the first sentence of their explanation; a rank too wide is folded; labels and notes wrap as the engine sizes them | Immich: context, containers and the server's 16 main components all read with descriptions at fit zoom; `--label` again, $0.018 | `54e6b6d` |
| A front end's components are features found across its layers, where no suffix convention applies; the marker on a wrapped arrow label sits with its text | web: 15 folders → 41 components, 26 of them features; mobile 18 → 54; server unchanged | `2e40636` |
| A main view draws each component's two strongest uses; `--explain` for the new web and mobile components | mobile: 32 arrows where there were none, of 155 uses; 87 components asked, $0.44, 165 explained in all | `5e1e6e3` |

**9 Oct, against the vision** — checked on three small projects written for it (`docs/decisions.md`): a script and a page were refused, a six-file app was thin and partly wrong.

| What | Measured | Commit |
|---|---|---|
| A project with no manifest is read from its files: scripts, or a page and its scripts | A one-file Python script and a one-page site: both documented | `a0c3bda` |
| A tiny project gets one page, no chapters, nothing to fill in (D-13) | 21 files written → 8; the page says what it is, does, reaches and is made of | `a0c3bda` |
| Hosted backends in the catalog; a named URL that is called; Next's page rule only for Next; the root application named | The small app: Supabase drawn, 2 pages for 2 (was 4), the script's API drawn (was "computed at run time") | `a0c3bda` |
| A small project told as a story, on its page and in the app's Overview; the app without a containers level for it; a page's inline scripts read | `subjects/small-script`, `small-page`, `small-app`: each a cited story of four to six sentences | `493ef72` |

**Not done** — known, and left:

| What | Why it is open |
|---|---|
| Stale explanations on small components | Eleven of Immich's one-file components still show an earlier model's answer, marked stale, with advice to run `--explain` — which no longer asks about them. Hide them, or stop marking them: undecided |
| AC-3 and AC-9 | AC-3 waits for the hand-drawn Immich reference (Cruz). AC-9 waits for Supabase, which is not cloned — no room for it now |
| `archdoc diff` on real history | The Immich clone holds one commit, so two real commits of a real repository have not been compared; the two-commit test uses the fixture |
| Earlier `--explain` runs keep their old label | Runs 1–7 in Immich's log still read `structure-only`; the log is a record and was not rewritten |
| A person choosing a container's main components | The main view picks by count of routes, pages, commands and jobs handled. A rule in `rules.yaml` naming them would be better and is not built |
| "What the AI did" | The vision's change view: a summary of a commit or a session a person reads (F-38), and plan against code (F-40, F-41). `archdoc diff` is the base; nothing is built on it |
| A story for a system | A small project opens on a cited story; Immich still opens on counts. The vision's guided story for a large system — where to start, what matters — is not built |
| A project an AI actually built | The three small subjects were written to test with. Nothing vibe-coded has been through archdoc |
| The batch API for `--explain` | Half price, results later; needs the run reworked. Left out on purpose at $0.005 a component |
| Queues in Python | Celery, RQ: not followed. Only decorator-marked TypeScript jobs and events are |
| Migrations in a tool's own DSL | Alembic, Knex, Rails' `schema.rb`: not read. Only SQL is |
| Coverage lists only the migration files that create a table | One that only alters is read and applied, and not named |
| Readers verified on fixtures only | Express, Next.js, React Router, Prisma, SQLAlchemy, Typer and Click; SQL migrations; Python class methods beyond Immich's ML service. No real repository has been through them |
| No side-by-side of `--label` on Sonnet 5.5 against Opus 5.5 | Sonnet's output read correctly; the comparison was never made |
| The other test subjects | Mastodon and the FastAPI template were deleted and not cloned again; Supabase never was (AC-9) |
| `develop` → `main` | `main` holds the initial commit only; no pull request yet, by choice |
| The published site, and a demo walk-through | Deferred to demo preparation, by choice |
| Which of Archify's diagrams archdoc should also draw | Raised 8 Oct, not settled |

### Superseded — do not implement as written

Two catalog entries are contradicted by later decisions. Recorded here so they are not
mistaken for pending work.

| ID | Catalog text | Superseded by |
|---|---|---|
| `EXT-05` | Parse Kubernetes manifests — Deployments, Services, Ingresses… | Orchestrator manifests are **out of R1 scope**. `docs/product-definition.md` §13, Settled |
| `VIE-03` | Compute layout (`dagre` / `elkjs`) | Layout is `goccy/go-graphviz`, computed in the engine and persisted. `docs/stack-decision.md` §3 |

---

## Sprint gates

From `docs/delivery-schedule.md` §4. A sprint is done when its gate passes, not when its
items are ticked.

| | Ends | Gate | Status |
|---|---|---|---|
| Week 0 | 28 Aug | O-8 answered; `go test ./...` green in CI | ✓ **met, verified.** O-8 and O-7 closed on evidence. CI green on `a96fc1d` — both jobs, including the import check that enforces the network boundary |
| Sprint 1 | 11 Sep | `archdoc generate` produces a container **and** a context diagram for all three subjects, every element traceable to a file and line | ✓ **met, 6 Sep.** All three generated; AC-7 verified on each. (The gate here previously read *AC-7, AC-4, AC-5* — those are sprint 2's, per `docs/delivery-schedule.md` §4.3) |
| Sprint 2 | 25 Sep | AC-2, AC-4, AC-5 — **the thesis, demonstrated** (per O-9; this line read AC-2, AC-6, AC-8 until 12 Sep) | ✓ **all three met by 12 Sep**, as is AC-8. Every coding row done, web app included. Left: report draft, self-check (18 Sep), review prep (24 Sep) |
| Sprint 3 | 9 Oct | All nine evaluated and written up | ⊘ **superseded 2 Oct** — 9 Oct became a progress review; this gate moves to the extension phase below |
| **Extension** | **11 Dec** | Code as evidence: component, data and flow views on all subjects, every element cited and what cannot be resolved shown; all nine criteria scored and written up | · plan in `docs/delivery-schedule.md` §7; work chosen in `docs/feature-inventory.md` |

---

## Open questions

| | Question | Blocks | Status |
|---|---|---|---|
| O-7 | Test subject #3 | — | ✓ **closed 26 Aug — Mastodon**, pinned `47ac677`. Chosen to cover `MDL-09`/`MDL-11`/`MDL-16`/`EXT-09`, which the other two leave untested |
| O-8 | Does `file:line` provenance survive the Compose merge? | — | ✓ **closed 26 Aug — no.** Extraction is two passes; a `Fact` carries one position. See `docs/decisions.md` |
| O-6 | R1.b sequencing — analysis before clustering? | R1.b only | deferred by design, decide with a working spine |
| O-10 | Does R1.a resolve service-to-gateway calls by matching route paths? | fuller `MDL-03` | **open for gateways; closed 9 Oct for generated API clients** (an OpenAPI document matched to a container's routes). Routes are read, but a caller's URL names one endpoint and the bridge cannot tell which. Only actors bridge today, so a service calling a gateway loses that edge. Matching `lds` route prefixes against caller URLs would close it — decide before code freeze |
| O-11 | Where does archdoc go beyond configuration? | The roadmap after R1.a | **direction closed 2 Oct, features provisional.** Positioning: the local, verifiable alternative (`vision.md` §2.2). Audience: technical readers losing control. Evidence before building. Features scored in `docs/feature-inventory.md`, confirmed 13–17 Oct. Original note: Proposal in `docs/vision.md`: code as evidence, one model with many lenses, reference docs plus explanations. Seventeen decisions (D-1–D-17) to take or amend; the first step is a cheap comprehension pilot (D-14). Schedule unchanged until agreed |
| O-9 | Which acceptance criteria gate sprint 2? | Sprint 2 gate | ✓ **closed 6 Sep — §4.3 wins: AC-2, AC-4, AC-5.** The disagreement was a symptom, not a judgement call: the calendar had no rows for the validator or for rules, so nothing in it could have passed AC-4 or AC-5, and the gate had been quietly reconciled to the rows. Six missing rows added instead |

---

## Manual work — does not compress

Verification-bound items that AI assistance does not accelerate.

| | Needed for | Status |
|---|---|---|
| Hand-drawn Immich reference architecture | AC-3 — it is the answer key | · not on the calendar, by decision. Owned by Cruz, no date |
