# Progress

Tracked against the capability catalog (`docs/product-definition.md` §4) and the acceptance
criteria (§12). These are the project's own work breakdown — no parallel TODO list.

**Status key:** `·` not started · `~` in progress · `✓` done · `⊘` superseded, see note

Last updated: 2026-10-05 (surface redesign)

---

## Acceptance criteria — the definition of done

Four of these need no human judgement and no external subjects. They are the ones to run
constantly, not just at a review.

| | Criterion | Threshold | Self-checking | Status |
|---|---|---|---|---|
| AC-1 | Nodes and edges carry extraction or catalog provenance | ≥ 95% | no | ✓ **100% on all three subjects** (24 nodes, 34 edges, 8 Sep). `PRV-02` closed the catalog hole: a technology now cites the catalog entry that supplied it. Re-measure after the semantic layer, which is where the number can fall |
| AC-2 | Container diagram produced with the LLM disabled | renders + validates, both subjects | **yes** | ✓ **met, 11 Sep.** Renders and validates on all three subjects with no model involved. The live on/off comparison held on Supabase across four versions: identical elements, kinds and relationships — only words changed |
| AC-3 | Structural accuracy vs hand-drawn reference | ≥ 0.85 on Immich | no — needs the reference | · **reference due 5–16 Oct (Cruz)**, scored in the evidence sprint 20–24 Oct. Also the answer key for the Code Wiki / DeepWiki comparison |
| AC-4 | Validator rejects malformed models | 100% of fault-injection suite | **yes** | ✓ **13 of 13 rejected, 8 Sep.** Every fault in the suite is a way the draw.io experiment failed, or a way the model could lie without a reader noticing. Grow the suite as new failure modes appear |
| AC-5 | Rule persistence | 10 rules survive regeneration | **yes** | ✓ **10 of 10, 8 Sep.** Measured by running the whole pipeline from disk twice, not by re-applying a cached model — surviving *regeneration* is the criterion |
| AC-6 | Drift detection | 100% of synthetic drift set | **yes** | · scheduled — Build 3, 24 Nov – 3 Dec (F-37 diff between commits, F-39 drift set) |
| AC-7 | Determinism | 5 runs, byte-identical FactSets | **yes** | ✓ **holds end to end** — 5 identical runs on all three subjects, measured on the full generated document rather than the FactSet alone. Covered by tests in four packages |
| AC-8 | Egress | `structure-only` transmits zero file contents | **yes** | ✓ **met, 12 Sep.** Measured on the wire: the real SDK against a local fake of the API, the recorder holding exactly the bytes the server received, with no path, line, provenance or key among them. Every run is logged, failures included; `archdoc runs --show` prints it unformatted |
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
| `MEM` | Memory and diff | 8 | ~5 | AC-6 — versions compare structurally in the app (MEM-04/06/07); diff between two *commits* and MEM-05's rename/re-bound classes remain |
| `VIE` | Views and rendering | 10 | ~7 | AC-2 |
| `SUR` | Surfaces — CLI and web app | 22 | ~19 | AC-8 — web views redesigned and SUR-16–22 added and done 5 Oct (see below); `init`, `diff`, and `export` formats other than `--site` remain |
| `OUT` | Output and deliverables | 12 | ~11 | — coverage report and the MkDocs site done 2 Oct; OUT-11 published site and OUT-12 coverage as data 5 Oct; OUT-09 from size and mtime (no conformance % yet); OUT-10 approximated by mtime, not by commit |
| `ANS` | Answer surface | 7 | 0 | — *(R1.c, stretch)* |
| | **Total** | **125** | **~84** | |

109 are R1.a; the 7 `ANS` capabilities are R1.c.

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
| 4 | Data model from table classes and migrations | F-07, F-11 | ✓ 5 Oct — from table classes (TypeORM-style decorators, SQLModel, SQLAlchemy); Immich: 68 tables, 65 foreign keys, in PostgreSQL; explorer Data level, `data-*.svg`, Mermaid erDiagram. Migrations not read: they say how the schema got here, not what it is |
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

Not done: a compact `model.json` (2.6 MB on Immich, a third of it flows); flows through a Python
class's methods, events and queues; migrations as a schema source; a fresh `--explain` run since
route summaries were allowed into it — Immich shows its 42 earlier answers, marked stale.

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
| O-10 | Does R1.a resolve service-to-gateway calls by matching route paths? | fuller `MDL-03` | **open.** Routes are read, but a caller's URL names one endpoint and the bridge cannot tell which. Only actors bridge today, so a service calling a gateway loses that edge. Matching `lds` route prefixes against caller URLs would close it — decide before code freeze |
| O-11 | Where does archdoc go beyond configuration? | The roadmap after R1.a | **direction closed 2 Oct, features provisional.** Positioning: the local, verifiable alternative (`vision.md` §2.2). Audience: technical readers losing control. Evidence before building. Features scored in `docs/feature-inventory.md`, confirmed 13–17 Oct. Original note: Proposal in `docs/vision.md`: code as evidence, one model with many lenses, reference docs plus explanations. Seventeen decisions (D-1–D-17) to take or amend; the first step is a cheap comprehension pilot (D-14). Schedule unchanged until agreed |
| O-9 | Which acceptance criteria gate sprint 2? | Sprint 2 gate | ✓ **closed 6 Sep — §4.3 wins: AC-2, AC-4, AC-5.** The disagreement was a symptom, not a judgement call: the calendar had no rows for the validator or for rules, so nothing in it could have passed AC-4 or AC-5, and the gate had been quietly reconciled to the rows. Six missing rows added instead |

---

## Manual work — does not compress

Verification-bound items that AI assistance does not accelerate.

| | Needed for | Status |
|---|---|---|
| Hand-drawn Immich reference architecture | AC-3 — it is the answer key | · not on the calendar, by decision. Owned by Cruz, no date |
