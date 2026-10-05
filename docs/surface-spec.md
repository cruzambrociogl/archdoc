# archdoc — the surface: capabilities, screens and data

> **What this document is:** the specification of archdoc's web surface for the vision in
> `docs/vision.md` — every capability it must offer, every screen, what each shows, where its data
> comes from, and the states it must handle. Written to be handed to **Claude Design** as the brief
> for a full redesign, then brought back here and implemented against.
>
> **What it is not:** a decision on the feature list. Status columns follow the provisional working
> set in `docs/feature-inventory.md`; the planning week (13–17 Oct) can still move rows. The design
> covers the whole vision on purpose, so later features land in a slot that already exists rather
> than being bolted on.
>
> Drafted 2026-10-04. **Built 2026-10-05** except the §11 run, publish, git and settings controls and
> the correction composer — the status marks below are as drafted; `PROGRESS.md` holds what shipped.

### Where to look

| If you want… | Read |
|---|---|
| The product in one page, for someone who has never seen it | §0 |
| The bar we are designing to — Code Wiki, DeepWiki, Archify | §1 |
| The rules every screen obeys | §2 |
| Live app versus published site — one app, two modes | §3 |
| The layout and navigation | §4 |
| Every screen, one by one | §5 |
| The shared components the design system needs | §6 |
| The data each screen reads, and what is new | §7 |
| What to hand Claude Design, and in what order | §8 |
| What was decided, and what is still open | §9 |
| How the app draws diagrams | §10 |
| What the app may do — run, arrange, correct, publish; never git | §11 |

### Status key

| Mark | Meaning |
|---|---|
| ✅ | Built: the data exists and a view exists (it gets redesigned, not invented) |
| ◐ | The data exists; the view is new |
| 🔜 B1 / B2 / B3 | Needs a build block first (`delivery-schedule.md` §7) |
| ○ | Out of this phase. **Design the slot, do not build it** |

---

## 0. archdoc in one page — for the designer

archdoc reads a code repository and produces the architecture documentation it should have had:
diagrams (C4 context, containers, components, deployment), the data model, the main flows, a feature
list, and the twelve arc42 documentation sections. It runs locally as one binary.

**What makes it different is a guarantee, not a look.** Every box, arrow, table and sentence on the
screen is extracted from the code and cites the file and line that proves it. An AI model may name,
describe and group things, but it can never add something that does not exist, and its writing is
always visibly marked. When archdoc sees evidence of something it cannot identify, it shows it as
**unresolved**; it does not drop it and it does not guess.

So everything on screen is in one of three **truth states**, and the design must make them
unmistakable everywhere:

| State | Means | Example |
|---|---|---|
| **Proven** | Extracted by a parser; cites file and line | `postgres` container — `docker-compose.yml:12` |
| **Interpreted** | Written by the model, on top of proven facts, with citations | "Handles media uploads and thumbnail jobs" |
| **Unresolved** | Evidence that something exists, which archdoc could not identify | "calls something at `${IMMICH_MACHINE_LEARNING_URL}`" — `config.dto.ts:624` |

**Who reads it.** Primary: technical people losing control of code an AI wrote for them — they need
the map, the change since last time, and a one-click path to the line. Secondary: people who cannot
read code (vibe coders), for whom the citation is not a link but a guarantee that nothing was
invented. The design serves the first and does not shut out the second.

**Where it is seen.** In two places, from one application (§3): locally with `archdoc serve`, and as
a static site a team publishes from their own repository — the answer to "share it with my team"
that is not a hosted service.

---

## 1. The bar — what the competitors ship

### 1.1 What each does on screen

| | DeepWiki (Cognition) | Google Code Wiki | Archify |
|---|---|---|---|
| **Shape** | Hosted wiki per repository | Hosted wiki per repository, refreshed every commit | One self-contained HTML file per diagram, produced by an agent |
| **Layout** | Left: page tree. Centre: page. Right: "On this page". Bottom: the Ask box | Wiki navigation, pages, a chat side panel | Full-screen canvas with an overlaid toolbar |
| **Pages** | Overview, then one page per subsystem; each opens with a collapsible **"Relevant source files"** list; paragraphs end in *Sources: file:lines* | Repository overview, module walkthroughs; sections link to files, classes, functions | — |
| **Diagrams** | Mermaid, inside pages | Architecture, class and sequence diagrams, regenerated as code changes | Architecture, workflow, sequence, data flow, lifecycle |
| **Diagram interaction** | Static | Elements link to code | Find node (`/`), **path probe** between two nodes, **lens** by semantic role, upstream/downstream reach, zoom-dependent detail, minimap radar, presentation stage, deep links (`#focus=`, `#route=`, `#lens=`), keyboard-first |
| **Change** | Re-index; no diff | Refreshed; no structural diff | **Before / Delta / After** with added, removed, changed, moved |
| **Ask** | Ask Devin, plus Deep Research mode | Gemini chat with code links | — |
| **Export** | Share link | Share link | PNG, share cards (1200×630) for a route or a reach, light/dark, visual styles |

### 1.2 What we take, and what we refuse

**Take — the parity floor.** A reader moving from DeepWiki or Code Wiki must not feel archdoc is
poorer:

- the wiki shape: page tree, readable pages, an on-this-page outline, sources at the top of every
  page;
- diagrams at several levels, with sequence and data-model views, every element clickable;
- Archify's canvas interactions: find, focus, upstream/downstream, path probe, lenses, minimap,
  deep links, keyboard shortcuts, export;
- Archify's before/delta/after — which archdoc already has as a version diff;
- a global search that reaches everything.

**Refuse:**

- anything the model drew or claimed without a citation that resolves;
- decorative motion that changes what a picture says, or that enters an export;
- "regenerated, no idea what changed" — every view can show the change since a chosen version;
- dependence on a server, a CDN or an account to read a published site.

**Exceed — what they structurally cannot show,** and so what the design should lead with:

1. The **three truth states**, visible on every element, sentence and arrow.
2. **Coverage** — a first-class screen on what archdoc could not see.
3. **The change lens** — what the AI just did, architecturally.
4. **Corrections that survive** — what a human fixed, and that it stays fixed.
5. **Egress, byte for byte** — what left the machine, if anything.
6. *(stretch)* **Intent versus actual** — did it build what was asked.

---

## 2. Rules for every screen

1. **Every visible fact has a way back to its evidence.** A box, a row, an arrow label, a sentence.
   One click (or hover) shows the citation; a second opens the line. No exceptions in the design —
   if a mock-up shows a value, it shows where the value's citation lives.
2. **Truth state is always encoded, never only by colour.** Proven, interpreted and unresolved
   differ in form as well (solid / italic and marked / dashed or hatched), so they survive
   greyscale, print and colour blindness.
3. **Views are earned by evidence** (`vision.md` §2.6). A screen with nothing behind it does not show
   an empty diagram; it says why it is absent ("No data models found — archdoc reads Prisma, TypeORM
   and SQLAlchemy; none were present") and links to Coverage. The navigation hides or dims what was
   not earned.
4. **Same answer every time.** Nothing on screen is randomised or computed differently on reload:
   layout positions are stored per version, lists are sorted, and the app computes nothing the
   engine did not store (today's rule in `web/src/api.ts`).
5. **Readable at Immich scale** — about 300 elements. No view tries to show everything at once; an
   overview holds roughly a dozen or two elements, and focus views handle the rest.
6. **Works offline and published.** No runtime CDN, no web fonts fetched at view time, no account.
   Light and dark themes both first-class.
7. **Technical first, plain language one click away.** The default is dense and precise. A reader
   can switch explanations to plain language (§6.10), and citations are never removed in that mode.
8. **The surface never writes to the repository** in this phase. Where a human would correct
   something, the app produces the `rules.yaml` snippet to paste (§5.13); it does not save it.

---

## 3. One app, two modes

Today there are two surfaces that share nothing: the web app (`archdoc serve`) and an MkDocs
configuration (`--site`) that needs Python to build and shows only the Markdown. **The proposal:
one application, run in two modes.**

| | **Live** — `archdoc serve` | **Published** — `archdoc export --site` |
|---|---|---|
| What it is | The app, served from the binary, reading the store through `/api` | The same app, built as static files with the data baked in as JSON |
| Who sees it | The developer, on their machine | Their team, from GitHub Pages or any static host; or a folder opened from disk |
| Data | Every version in the store | The current version, plus the change since a chosen earlier version |
| "Open the line" | `vscode://` to the local file (today's behaviour) | A link to the file and line at the commit on the repository's host, when a remote is known; otherwise the path as text |
| Rules screen | Full: rules, operations, findings | Read-only list of the corrections in force |
| Network runs | Full log, payloads included | Omitted by default; a summary ("no data left the machine" / "1 run, 5 KB, structure only") |
| Live refresh | Later, with watch mode (○ F-29) | — |

This answers D-10 and F-22/F-23 together: explanations get a home that renders with archdoc
uninstalled, and the site stops being a second, poorer product. The Markdown in
`docs/architecture/` stays the reference deliverable; MkDocs can remain as a fallback or go (§9).

**Implementation consequence (for later, not for the designer):** `api.ts` gains a data source
switch — `/api/...` live, `data/*.json` published — and every screen has to work from a single
version's data, with history-dependent parts degrading cleanly.

### 3.1 Where the site's content comes from

**From the model, not from the Markdown.** The Markdown is one projection of the validated model;
the site is another. Both are produced by the same run from the same model, so they cannot disagree
(P3) — and neither is built from the other.

```
                    validated model (+ stored layout, interpretations, coverage)
                                         │
             ┌───────────────────────────┼────────────────────────────┐
   docs/architecture/*.generated.md   site bundle: data/*.json      `archdoc serve`
   + .mmd + .svg (committed)          + the app (static)            /api/* (same shapes)
   GitHub, MkDocs, code review        the published site            the live app
```

Why not render the site from the `.md` files: Markdown flattens what makes the site worth having.
A citation becomes text, a truth state becomes italics, a diagram becomes a picture, and the
inspector, focus, search and delta overlay have nothing to hold on to. Parsing our own output back
into structure would also be a fragile second parser of something we already hold as data.

What goes in the bundle — the same JSON the live API serves, for one version:

| File | Holds |
|---|---|
| `summary.json` | Repository, commit, counts, coverage %, interpreted share, egress summary |
| `model.json` | Elements, relationships, provenance on every value |
| `views/*.json` | One scene per diagram — elements, groups, edges, positions, truth states (§10) |
| `pages/*.json` | Component pages, flow pages, entity pages: prose as **sentences with citation ids**, never as rendered text |
| `coverage.json`, `diff.json`, `rules.json`, `search.json` | The structured forms of Coverage, Changes (against the chosen baseline), Corrections and the search index |
| `docs/*.md` | The **generated** arc42 sections, for the Documents reader (§5.10) |

**The human-owned sections are the one exception, and hard rule 2 decides it.** archdoc never reads
them — `serve` already refuses them at the API. So the bundle cannot copy them in, since copying
is reading. In the published site the Documents reader lists them with their completeness state
and **links to them on the repository host**, which renders Markdown on its own (the same
citation mechanism as S-3). MkDocs keeps working as the fallback precisely because *MkDocs* reads
them, not archdoc. An option for later, which keeps the rule: when the site is deployed alongside
the repository's files, the reader's browser fetches and renders a human section itself at view
time; archdoc still never opens it.

### 3.2 What is stored, and what is computed when you look

**Generating is the expensive step and happens once per change. Looking is cheap and recomputes
nothing.** The JSON the app reads is not "content being generated" — it is the stored content put
into the shape a screen wants, in milliseconds.

| Step | When | What it does | Cost |
|---|---|---|---|
| `archdoc generate` | When the code changed | Reads the repository, builds and validates the model, lays out every view, asks the model only for what changed (D-1), **stores the result** | Seconds to minutes; the only step that may cost money |
| `archdoc serve` | Whenever you want to look | Reads what `generate` stored and serves it to the app. No parsing, no model, no network | Instant |
| `archdoc export --site` | When you want to publish | Reads the same stored result and writes it as files beside the app | Seconds |

**Where it is stored** — three places, each with one job:

| Where | Holds | Committed? | Why |
|---|---|---|---|
| `.archdoc/history.db` (SQLite) | Every version: model and layouts; the network-run log | **No** — a local cache | Fast history and diffs on this machine. A binary in git conflicts on every parallel run |
| `.archdoc/` text files — `model.json` today; interpretations, `layout.yaml`, `views.yaml` with this vision | The current version's facts and interpretations, and the human arrangements and saved views | **Yes** | The durable record: travels with the code, readable in review, and lets a teammate or CI **see everything without regenerating and without an API key** |
| `docs/architecture/` | The Markdown, `.mmd` and `.svg` | **Yes** | The reference documentation, readable on GitHub with archdoc uninstalled |

Scenes, pages, the search index and the published bundle are **not** stored. They are cheap
projections of the above, rebuilt whenever they are asked for, which keeps them from going stale.

**What this means in practice:**

- Run `generate` once; open `serve` as often as you like, today or next month — it shows the last
  result immediately.
- Run `generate` again only after the code changes. If the architecture did not change, no new
  version is recorded. If it did, only the changed parts are re-interpreted (D-1), so a second run
  costs a fraction of the first.
- A teammate who clones the repository gets the committed record: `serve` and `export --site` work
  for them with no regeneration and no key. Their history database starts empty and fills from
  their own runs. Rebuilding older versions from git history is a possible later addition.
- CI publishes the site from the committed record. It never calls a model, so it needs no
  secrets.

**Consequence for the engine (for us):** interpretations — labels today, component and flow prose
with this vision — must be committed alongside `model.json`, not kept only in `history.db`.
Otherwise every clone and every CI run would have to pay to regenerate them, and the site would
differ from what the author saw. This is D-1's interpretation memory, made durable.

**Where it is written and how it is published.** `archdoc export --site` writes the bundle to
`.archdoc/site/` (inside the closed write set; already MkDocs's `site_dir`). It is a build product,
not committed. Publishing is a CI step — run `export --site`, deploy the folder to Pages — and the
baseline for "what changed" comes from the previous commit's committed `model.json`, so CI needs no
history database.

---

## 4. Layout and navigation

### 4.1 The shell

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ archdoc · immich   ⌁ commit 4f2a1c · v12 ▾   [ ⌘K  Search everything…    ]  ◐ │  top bar
├───────────────┬──────────────────────────────────────────────┬───────────────┤
│ Overview      │                                              │ On this page  │
│ Architecture  │              main content                    │   or          │
│   Context     │        (page, diagram, table, diff)          │ Inspector     │
│   Containers  │                                              │ (selected     │
│   Components  │                                              │  element)     │
│   Deployment  │                                              │               │
│ Components ▸  │                                              │               │
│ Features      │                                              │               │
│ Flows ▸       │                                              │               │
│ Data model ▸  │                                              │               │
│ Dependencies  │                                              │               │
│ Changes  •3   │                                              │               │
│ Coverage 87%  │                                              │               │
│ Documents ▸   │                                              │               │
│ ─────────     │                                              │               │
│ Corrections   │                                              │               │
│ Network runs  │                                              │               │
└───────────────┴──────────────────────────────────────────────┴───────────────┘
```

- **Top bar:** product mark, repository name, commit, version picker (live) or "generated from
  commit … on …" (published), global search, theme toggle, a mode badge (*Local* / *Published*), and
  the truth-state legend reachable from one click.
- **Left navigation:** the wiki tree. Entries for unearned views are hidden, not empty (rule 3).
  Groups expand to their pages (each component, each flow, each entity). Badges carry counts that
  matter: changes since the compared version, coverage percentage.
- **Right rail:** "On this page" on reading pages; the **inspector** when something is selected on
  a diagram. Collapsible.
- **Ask panel ○:** a slot docked to the bottom of the main column, as DeepWiki does. Designed now,
  built when F-26 is.
- **Responsive:** the rails collapse to drawers below ~1100 px; the diagram screens remain usable on
  a laptop; the reading pages remain readable on a phone.

### 4.2 Navigation behaviour

- Every state is a URL: page, selected element, diagram level, focus, path, lens, compared version.
  Copying the address bar reproduces the view (Archify's `#focus=`, `#route=`, `#lens=`, extended).
- Keyboard: `⌘K` / `/` search, `?` shortcuts, `g o` overview, `g a` architecture, `[` `]` previous
  and next page, `Esc` clears selection.
- Back and forward work through every selection and every focus change.

---

## 5. Screens

Each screen: what question it answers, what it shows, how it behaves, its states, and its status.

### 5.1 Overview — the landing page

**Answers:** what is this system, at a glance, and how much of it can I trust?

- One-paragraph system summary (interpreted, cited) and the names of the applications discovered.
- **Stat strip:** applications, containers, components, entities, flows, features, external
  systems — each a link to its screen.
- **The trust strip:** coverage percentage, unresolved count, share of text that is interpreted
  rather than proven, and egress status ("nothing left this machine" / last run). This is the
  guarantee made visible for readers who cannot check a citation.
- The main architecture diagram at its most useful level, clickable into the explorer.
- **What changed** since the compared version: a short list with counts (added, removed, changed),
  linking to Changes.
- **Where to start** — three reading paths: "Understand it from the top" (context → containers →
  key flows), "What changed" (Changes), "Check the evidence" (Coverage).

**States:** first run (no previous version — "what changed" says so); no LLM used (the summary is
replaced by the extracted facts; the trust strip says "0% interpreted").
**Status:** ◐ for configuration-level repositories now; full content 🔜 B2.

### 5.2 Architecture explorer — the canvas

**Answers:** what are the pieces and how do they connect, at the depth I choose?

- **Levels:** System context · Containers · Components 🔜 B2 · Deployment ✅ (data) / ◐ (view).
  Switching level keeps the selection when the element exists at the new level.
- **Semantic zoom:** zooming out reduces boxes to names; zooming in reveals technology and
  description (Archify's reading depth by zoom).
- **Find** (`/`): type a name, the match is focused and the rest fades.
- **Focus:** selecting an element dims everything not connected to it; **Upstream** and
  **Downstream** extend the highlight through proven edges only.
- **Path probe:** choose two elements, see every route between them, step by step, each step cited.
- **Lenses:** filter or colour by kind (data stores, external systems, entry points), by
  technology, by truth state, by change since the compared version.
- **Containers open:** double-click a container to descend to its components.
- **Arrows are first-class:** clickable, with their own inspector (today they are not — F-55).
- **Delta overlay:** with a compared version, added / removed / changed elements and edges are
  marked on the diagram itself (before / delta / after).
- Minimap, zoom controls, fit, full-screen presentation mode.
- **Legend:** always one click away; it teaches proven / interpreted / unresolved, declared /
  referenced, and the C4 notation.
- **Export:** SVG (the committed one), PNG, and a share card of the current focus or path.

**Unresolved on the canvas:** an unresolved reference appears as a dashed stub leaving its source
("→ something at `${URL}`"), never as an invented target box.

**States:** one-element diagram (the level is not earned — say so, suggest the useful level); very
large level (open on the overview cap with a "show all N" escape and a search prompt).
**Status:** ✅ context and containers (redesign); interactions ◐; components 🔜 B2.

### 5.3 Inspector — an element's passport

**Answers:** what exactly is this, how do we know, and what touches it?

Opens in the right rail on selection; also a full page per element (shareable URL).

- **Header:** name, kind (with icon), technology, truth state, evidence class (*declared* — the
  repository defines it; *referenced* — the repository only names it).
- **Facts,** each with its own citation chip: name, description, technology, image, ports,
  networks, parent. The chip shows origin — extraction, catalog, rule, model — as today, redesigned.
- **Relationships:** incoming and outgoing, grouped by kind, each with label, protocol and
  citations. Unresolved references listed with the same weight as resolved ones.
- **Where it lives:** the source files and lines, one click to open each.
- **Touches** 🔜 B2: entities it reads or writes, flows it takes part in, features it serves.
- **History:** when it appeared, what changed about it across versions (◐ — computed from stored
  versions).
- **Corrections:** rules that apply to it, and **"Correct this…"**, which composes a `rules.yaml`
  snippet (rename, reclassify, describe, exclude) to copy (§5.13).

**Status:** ✅ (redesign) plus ◐ history and correction snippet; "touches" 🔜 B2.

### 5.4 Component pages — the wiki

**Answers:** what does this part of the system do, and where is it in the code? (F-19, the module
walkthrough Code Wiki and DeepWiki are known for.)

- **Relevant source files** — collapsible list at the top, files and line ranges (DeepWiki parity,
  but extracted, not chosen by a model).
- **Summary** — interpreted prose in which **every sentence carries citation markers**; hovering a
  marker previews the cited element; a sentence whose citations could not be validated is never
  shown (F-36).
- A focused component diagram: this component, its neighbours one hop out.
- **Entry points** it exposes (routes, handlers, commands) → Features and Flows.
- **Data** it owns or touches → Data model.
- **Outbound calls,** including unresolved ones.
- **Dependencies** (third-party packages) it uses.
- Previous / next component; "on this page" outline in the right rail.

**Status:** 🔜 B2–B3.

### 5.5 Features — what the system does

**Answers:** what can a user of this system actually do? (F-13 — arc42 §1, extracted.)

- Every real entry point — HTTP routes, pages, CLI commands, scheduled jobs, exported APIs —
  grouped by component or area, filterable by kind and by method.
- Each row: the entry point, its handler (cited), the component, and a link to its flow when one
  exists.
- An optional plain-language description per group (interpreted, cited).

**Status:** 🔜 B2.

### 5.6 Flows — what happens when…

**Answers:** what happens, step by step, when this entry point is called? (F-12 — arc42 §6, C4
dynamic view.)

- Flow index: one per traced entry point, searchable, grouped by feature area.
- **Sequence diagram** with lanes for the participants (UI, controller, service, queue, store,
  external system). Lanes and time lay themselves out — deterministic.
- **Step list** beside it: each step is a cited call; clicking a step highlights its arrow and
  shows its line.
- **Step-through:** advance one step at a time — a guided read, not decoration; respects reduced
  motion and never enters an export.
- **Unresolved steps** shown in the sequence as dashed arrows to an unknown participant, with the
  reason ("dependency injection not resolved", "URL built from a string").
- Switch to the same flow drawn over the architecture diagram (the path highlighted on the canvas).

**Status:** 🔜 B2.

### 5.7 Data model

**Answers:** what does this system store, in what shape, and who uses it? (F-11 — arc42 §8.)

- **Entity–relationship diagram,** clustered by component or schema, with focus and search as on the
  canvas. Immich has 64 tables: the overview shows clusters; a cluster opens to its entities.
- **Entity table:** name, fields count, relations, owning component, source (ORM model or
  migration, cited).
- **Entity page:** fields with type, nullability, keys, defaults; relations; which components and
  flows read and write it; status fields and their values ○ (lifecycle view, F-17, out).

**Status:** 🔜 B1 (reading) / B2 (view).

### 5.8 Dependencies

**Answers:** what third-party code does this rely on, and where? (F-14)

- Packages per application, grouped (framework, database driver, HTTP client, SDK of an external
  service, testing, build), with version and the manifest line.
- For each, which components import it. External services' SDKs link to their external-system box.

**Status:** 🔜 B1–B2.

### 5.9 Deployment

**Answers:** where and how does it run? (F-15 — arc42 §7, C4 deployment.)

- Deployment diagram and table: container, image, published ports, networks, mounts from the
  repository; CI workflow summary ○ (F-18, out).

**Status:** ✅ data / ◐ view (today it is only a generated Markdown section).

### 5.10 Documents — the arc42 reader

**Answers:** show me the documentation itself, as it is committed.

- The twelve arc42 sections (or fewer, when output is proportionate), in order, rendered: diagrams
  inline and interactive, citations as links.
- Each section marked **generated** (archdoc owns it, regenerated every run) or **yours** (human-
  owned, never read or written by archdoc), with its completeness state: *not started*, *written*,
  *may be stale* (architecture changed after it was last edited).
- For human sections not started: the **contextual stub questions** archdoc generated for this
  system ("two containers are reachable from outside — what is the availability target of each?"),
  and a link to open the file in the editor.
- The completeness meter across the twelve (today's Completeness tab, merged here).
- Download: the Markdown folder; print-friendly view.

**Status:** ✅ (Documents and Completeness, redesigned and merged).

### 5.11 Coverage — what archdoc could not see

**Answers:** how much of this can I rely on, and where are the gaps? (F-24 — the honesty screen.
None of the competitors has one; give it real weight in the design.)

- **Headline:** coverage percentage, with what it is a percentage *of*.
- **What was read:** files and applications, by kind; files passed over and why.
- **Unresolved references,** grouped by reason, each cited — the list a reader works through.
- **Gaps by rule:** relationships without protocol, elements without description, and so on.
- **Fixed limits:** what the evidence types used here cannot say, and what would be needed.
- Sample-file values: which values were filled from `example.env`-style files (F-53 ○).

**Status:** ✅ data (as `coverage.generated.md`) / ◐ structured view (needs the coverage as JSON).

### 5.12 Changes — what the AI just did

**Answers:** what changed in the architecture since then? (The change lens; F-37, F-38.)

- **Timeline** of versions (live) with commit, date and a one-line change count per version.
- **Compare:** pick any two versions (live) or two commits 🔜 B3. Result as:
  - a **summary** — appeared, disappeared, renamed, rewired, re-described, with the two kinds kept
    apart: *structural* change versus *only words changed* (the model relabelled something);
  - the **diagram in delta mode** (§5.2);
  - a **list** of every change, each with before and after and both citations.
- **Change summary per commit or session** 🔜 B3 — "this session added X, touched Y, introduced
  Z" — interpreted, cited.
- In the published site: the change since the version the publisher chose as baseline.

**Status:** ✅ version diff (redesign); commits and summaries 🔜 B3.

### 5.13 Corrections — `rules.yaml`

**Answers:** what has a human corrected, and is every correction still landing?

- The rules in force, each with what it matched and what it changed.
- **Findings:** rules that match nothing, rules that override another — today's warnings,
  redesigned.
- **Compose a correction:** from the inspector's "Correct this…", a small form produces the YAML
  to paste into `.archdoc/rules.yaml`, with an explanation that it survives every regeneration.
  The app copies; it does not write.

**Status:** ✅ list and findings (redesign); composer ◐.

### 5.14 Network runs — what left the machine

**Answers:** did anything leave my machine, what exactly, and what did it cost?

- Runs: time, model, egress mode, requests, bytes, tokens, cost (when the price is known), status.
- A run: every request **exactly as sent**, byte for byte — with a short plain-language reading
  above it ("names of 6 elements and 8 relationships; no file contents").
- Empty state as a feature: "archdoc has never sent anything from this repository."

**Status:** ✅ (redesign). Live only; summarised in the published site.

### 5.15 Intent versus actual ○ stretch

**Answers:** did it build what I asked for? (F-40, F-41.)

- The plan files the user nominated, requirement by requirement: **found** (where in the code,
  cited), **missing**, and **built but not asked for**.
- Design the slot and one example screen; build only if Build 2 lands early (13 Nov checkpoint).

### 5.16 Ask the map ○

**Answers:** a question I have, in my words. (F-26.)

- Docked input; answers drawn only from the validated model, each sentence cited like a component
  page; the honest answer "not resolved — here is what archdoc saw" is a designed state, not an
  error. Local-model badge when nothing leaves the machine.
- Design the slot and the answer card; not built this phase.

### 5.17 Global search

- One index over elements, components, entities, fields, routes, flows, dependencies, documents and
  files. Results grouped by type, keyboard-navigable, each showing its truth state.
- Searching a file path answers "what does archdoc know about this file?" — the elements it proves.

**Status:** ◐ over today's data; grows with each block.

---

## 6. Shared components — what the design system must define

| # | Component | Notes |
|---|---|---|
| 6.1 | **Truth-state treatment** | Proven / interpreted / unresolved, for boxes, arrows, table rows, inline text and sentences. Form plus colour (rule 2). The single most important visual decision |
| 6.2 | **Citation chip** | `file:line` with origin (extraction, catalog, rule, model); hover previews, click opens. A compact form for tables and a sentence-marker form for prose |
| 6.3 | **Element kind icons** | Person, system, external system, container, component, data store, queue, entry point, entity, flow, dependency |
| 6.4 | **C4 diagram style** | Boxes, boundaries, nested networks, dashed external systems — in both themes, drawn by the app (§10); the committed SVG is the export |
| 6.5 | **Sequence diagram style** | Lanes, activations, dashed unresolved calls, step numbering tied to the step list |
| 6.6 | **ER diagram style** | Entities, keys, relation cardinality, clusters |
| 6.7 | **Delta marks** | Added, removed, changed, moved — on diagrams, lists and inline values; distinct from truth states |
| 6.8 | **Empty / not-earned state** | Why the view is absent, what archdoc reads, link to Coverage |
| 6.9 | **Meters and stat tiles** | Coverage, completeness, counts — with what the number is out of |
| 6.10 | **Plain-language toggle** | Switches interpreted text between technical and plain versions; citations stay |
| 6.11 | **Inspector panel** | Shared by every diagram screen |
| 6.12 | **Code reference** | Path with line range, monospace, opens the line (live: editor; published: repository host) |
| 6.13 | **Keyboard shortcut sheet** | `?` overlay |
| 6.14 | **Share/export menu** | SVG, PNG, share card, copy link to this view |

---

## 7. Data — what each screen reads

The app computes nothing the engine did not store (rule 4). So every screen needs an endpoint
(live) and a JSON file (published) with the same shape.

| Data | Today | Needed for | New work |
|---|---|---|---|
| Summary, versions | `/api/summary`, `/api/versions` | Shell, Overview, Changes | Add counts per lens, coverage %, interpreted share |
| Model + views | `/api/model` (model, context, container) | Explorer, Inspector | Components, deployment as view projections 🔜 |
| Diagram | `/api/svg` | Explorer | Layout as JSON (positions per element) if the app draws the diagram itself — §9 |
| Diff | `/api/diff` | Changes, delta overlay | Commit-to-commit 🔜 B3 |
| Rules | `/api/rules` | Corrections | — |
| Documents | `/api/docs`, `/api/docs/{name}` | Documents | — |
| Completeness | `/api/completeness` | Documents | — |
| Runs | `/api/runs`, `/api/runs/{id}` | Network runs | — |
| Coverage | Markdown only | Coverage, Overview | `/api/coverage` as structured JSON ◐ |
| Element history | — | Inspector | `/api/element/{id}/history` ◐ |
| Search index | — | Global search | `/api/search` or a prebuilt index ◐ |
| Components, features, flows, entities, dependencies | — | §5.4–5.8 | 🔜 B1–B2, as the lenses land |
| Published bundle | — | Published mode | `archdoc export --site` writes the app plus every JSON above for one version ◐ |

---

## 8. Handing this to Claude Design

**Give it:** §0 to §6 of this document, §10.4 (what customising a diagram covers), §11.7 (the controls), and screenshots
of the current app on Mastodon as the "before". The rest of §7, §9 and §10 is for us. Tell it the
diagrams are drawn as interactive components (nodes and edges it styles), not as static images.

**Ask for, in this order** — the first three set the visual language everything else reuses:

1. **The design system:** truth states (6.1), citation chip (6.2), kind icons (6.3), C4 style
   (6.4), delta marks (6.7), both themes.
2. **The shell** (§4) with the **Architecture explorer** (§5.2) and the **Inspector** (§5.3) —
   container level of a mid-size system, one element selected, one unresolved reference visible.
3. **Overview** (§5.1).
4. **A component page** (§5.4) — the wiki look, citations in prose.
5. **A flow** (§5.6) — sequence diagram with step list and an unresolved step.
6. **Coverage** (§5.11) and **Changes** with the delta overlay (§5.12).
7. **Data model** (§5.7), **Features** (§5.5), **Documents** (§5.10).
8. Corrections, Network runs, Search, Dependencies, Deployment.
9. The slots: Ask the map (§5.16), Intent versus actual (§5.15).

**Use realistic content.** Immich as described in `vision.md` §2.8 is the best sample — server
(NestJS), web (SvelteKit), machine-learning (FastAPI), Postgres, Redis; the server → machine-learning
edge at `config.dto.ts:624`; feature areas assets, albums, auth, jobs; the "upload a photo" flow.
Mastodon (6 elements, 8 relationships) is the small case and the empty-state case.

**Tone:** a precise technical instrument — calm, dense, legible; closer to a good code editor or
API reference than to a marketing site. The vision's own line: *not prettier than Google, but good
enough that nobody picks the alternative for its looks alone* — the design target is "at their
level", the identity is "you can check every claim".

**Side use:** the same mock-ups are the "paper mock-ups of the new lenses" the comprehension pilot
needs (`vision.md` D-14, F-42, 20–24 Oct). Designing now feeds the evidence sprint as well.

---

## 9. Decisions

### Taken — 2026-10-04

| | Decision |
|---|---|
| S-1 | **The published app is the site.** `mkdocs.yml` stays as the plain fallback. The site's content is the model's JSON, not the Markdown (§3.1) |
| S-2 | **The app draws the diagrams itself,** interactively, from a scene the engine emits — for today's views and every planned one. The committed SVG and the Mermaid text become exports (§10) |
| S-3 | **Citations open the line:** the editor when live; a permalink at the commit on the repository host when published and a remote is known; the path as text otherwise |
| S-4 | **The surface is built early:** the shell and today's screens before Build 1, each new screen as its lens lands. Moves F-25 and F-22 forward; the planning week re-cuts the calendar around it |
| S-5 | **React Flow draws the graph diagrams**; our own component draws sequences; layout stays in the engine (Graphviz, ELK if nested boundaries demand it). A spike on Immich-sized data comes before the explorer — it confirms the approach, it does not reopen the choice |
| S-6 | **People can rearrange a diagram and keep it.** Drag boxes and groups; the arrangement is saved to `.archdoc/layout.yaml` and re-applied on every run, like `rules.yaml`. Committed, so the team sees the same arrangement and the published site uses it. Position is presentation, not fact (`vision.md` §3.4), so an arrangement cannot falsify anything. A new element with no saved position is placed by the engine beside its neighbours and marked as new; a saved position whose element is gone is reported, as an unmatched rule is. **It is the first thing the app writes**, so it carries ANS-04/05's write safety: refuse if the file changed on disk since it was loaded, or while a run is in flight. Live app only |
| S-7 | **Saved views:** named combinations of level, focus, filter, lens and arrangement, kept in `.archdoc/views.yaml`, committed, and listed in the navigation of both the live app and the published site. Shares S-6's write path |

### Still open

None for the surface. The planning week places S-4 to S-7 in the calendar.

---

## 10. Diagrams — how the app draws them

### 10.1 One scene format, several renderers, several exports

Every diagram, today's and planned, is described by the engine as a **scene**: plain JSON that the
app renders and that every export is produced from.

```
              model ──► view projection ──► layout (engine) ──► scene.json
                                                                   │
                ┌──────────────────────┬───────────────────────────┼──────────────────────┐
         app: graph renderer    app: sequence renderer      export: SVG (committed)   export: Mermaid
         context, containers,   flows                       docs/architecture/*.svg   *.mmd, inside the
         components, deployment,                                                      Markdown — GitHub
         data model, dependencies                                                     and MkDocs render it
```

A scene holds, for each element: id, kind, name, technology, description, **truth state**, evidence
class, citation ids, parent group, and **position and size**. For each edge: endpoints, label,
protocol, truth state, citations, and its route. For groups: boundaries and networks. For a sequence:
participants in order and numbered steps. **Nothing in a scene is computed by the app** — rule 4 —
so the app and every export show the same thing.

### 10.2 Why not Mermaid in the app

Mermaid is the right **export** and stays one: GitHub and MkDocs render it with no build step, and
it diffs in review. It is the wrong **renderer** for the app:

- **It lays the diagram out itself.** Positions are not ours, are not stored, and can move when the
  Mermaid version changes — the opposite of "same answer every time" and of the positions stored per
  version (D5 in the comparison matrix).
- **It returns an SVG string.** Focus, path probe, delta overlay, semantic zoom, clickable arrows and
  hover citations all have to be bolted onto markup it controls.
- **Its styling is a theme, not a design system.** The truth states, citation chips and kind icons
  of §6 cannot be expressed in it properly.
- **It is large** (about a megabyte of JavaScript) for what the app would use of it.

### 10.3 What draws them instead — recommendation, to confirm with a spike (S-5)

| Diagram family | Renderer | Why |
|---|---|---|
| **Graphs** — context, containers, components, deployment, data model, dependencies | **React Flow** (`@xyflow/react`, MIT) | Nodes and edges are React components, so §6's design system is used as-is. Pan, zoom, minimap, fit, selection, groups and clickable edges come built in. It takes positions from outside, which is exactly what the engine provides. Bundled, so no network |
| **Sequences** — flows | **Our own React component** | Lanes and time lay themselves out (`vision.md` D-12), so there is no layout problem to buy a library for; a few hundred lines gives per-step citations, unresolved arrows and step-through, which no off-the-shelf sequence renderer does well |
| **Layout** | **Stays in the engine** — Graphviz, already embedded and deterministic | Positions are computed once, stored per version, and reused by the committed SVG. If Graphviz's nested boundaries look poor at component level, **ELK** (also deterministic, stronger on nested groups) is the alternative, run in the engine. The spike decides |

Considered and set aside: **Cytoscape.js** (excellent at scale, but draws to a canvas and styles
through its own stylesheet, so §6 would be rebuilt in a second language); **AntV X6 / G6** (capable,
heavier, a smaller ecosystem in English); **D3** (everything by hand); **JointJS** (the useful parts
are commercial).

**The spike, before building the explorer:** render Immich-sized scenes (≈300 nodes, nested groups)
in React Flow with Graphviz positions. Pass: it pans smoothly, the boundaries read correctly, and
the focus view and delta overlay are straightforward to build on it.

### 10.4 What "customisable" covers

| | What | Phase |
|---|---|---|
| Themes | Light, dark and print, from the design system's tokens | With the shell |
| Reading controls | Level, semantic zoom, focus, upstream/downstream, path, lenses, delta — all in the URL | With the explorer |
| Arrangement | Drag boxes and groups to make a crowded diagram readable; keep the arrangement for everyone (S-6). "Reset to automatic layout" always available | With the explorer, together with the write path |
| Saved views | Named views in the navigation, shared through the repository (S-7) | Right after S-6 |
| Export | SVG, PNG, Mermaid, share card, link to this view | With the explorer |

---

## 11. Control — what you can do from the app

> **Decided 2026-10-04** — C-1 to C-5 in §11.6. C-3 and C-5 are deliberately provisional.

### 11.1 The principle

- **The published site controls nothing.** It is static files: it shows, and where an action would
  help it shows the command to run.
- **The live app is a second front end over the same engine** (`product-definition.md` §7, "one
  engine, two thin frontends"). Every action it offers is a CLI command with the same name and the
  same effect. It gains nothing the CLI lacks, so the app is a convenience, never a second way into
  the repository.
- **Three lines it never crosses:**
  1. It never acts on git — no commit, push, pull, branch or checkout. It may *read* git.
  2. It never writes outside the closed write set (hard rule 2).
  3. Nothing that sends data or costs money happens without a confirmation showing what will be
     sent and roughly what it will cost.

### 11.2 The control map

| Area | Action | Live app | Published site | CLI | Status |
|---|---|---|---|---|---|
| **Read** | Browse, inspect, search, compare stored versions | Yes | One version, plus the chosen baseline | — | ✅ |
| **Freshness** | "The code changed since this was generated" — compares the version's commit with HEAD and the working tree | Status pill in the top bar | "Generated from commit … on …" | `archdoc status` (new) | ◐ |
| **Run** (C-1) | `scan` — free, offline, seconds | One click | — | `scan` | ◐ |
| | `generate` without the model — free, offline, deterministic | One click | — | `generate` | ◐ |
| | `generate` with the model — sends structure, costs money | After a confirmation: egress mode, model, items, estimated bytes, tokens and cost. Logged in Network runs like any run | — | `generate --label` | ◐ |
| | Progress | Stages streamed live; cancel; one run at a time | — | — | ◐ |
| | Watch mode — regenerate on save (F-29) | A toggle. Deterministic only: **never calls the model on its own** | — | `serve --watch` | ○ |
| | Re-describe one element — a single-element model call (D-1 option c) | Same confirmation, scoped to one element | — | — | ○ |
| **Arrange** | Drag a diagram, save named views (S-6, S-7) | Yes | Shown, read-only | — | 🔜 explorer |
| **Correct** | Add a correction to `rules.yaml` — append-only, after a preview of the exact lines (C-2) | Yes | Read-only list | edit the file | ◐ |
| **Document** | Write the human-owned arc42 sections | **No** — the answer surface (ANS), a later phase | — | your editor | ○ |
| **History** | Compare two commits — the engine reads them from git without a checkout (F-37) | Pick two commits | — | `archdoc diff` | 🔜 B3 |
| | Delete or prune cached versions | No — the database is a cache, deleting it is harmless | — | — | — |
| **Git** | Which of archdoc's files are uncommitted, **their diff against the last commit**, and the command to commit them (C-4). Only archdoc's own files: a human section's diff would be reading it (hard rule 2) | Read-only card | — | `git status`, `git diff` | ◐ |
| | Commit, push, pull | **Never** | — | git | — |
| **Publish** | Build the site into `.archdoc/site/` | One click | — | `export --site` | ◐ |
| | Deploy it | **Never from archdoc.** Only `internal/semantic` may make outbound calls (hard rule 3); deploying is CI's job. The app shows the workflow to copy, because `.github/workflows/` is outside the write set | — | CI | ◐ |
| **Settings** | Egress mode, model, provider, effort | Read-only for now (C-3) | Shows which mode produced the content | flags / config | ◐ |
| | API key | **Never entered in the browser.** It stays in the environment | — | env | — |

### 11.3 What "sync" means — three kinds, three owners

| Sync between | Owner | How |
|---|---|---|
| **Documentation ↔ code** | archdoc | The freshness pill says when they drift; Regenerate (or watch mode, later) closes the gap |
| **You ↔ your team** | git, which means you | archdoc writes committed files (§3.2). You commit them and teammates pull. The app tells you what is uncommitted, and does nothing about it |
| **Repository ↔ published site** | CI | On push, CI runs `export --site` from the committed record and deploys. No model calls, no secrets |

### 11.4 Versions, made concrete

- A **version** is recorded by `generate` only when the architecture changed. Versions live in the
  local cache (`history.db`), and the **committed `model.json` at each commit** is the durable copy.
- In the app you **view** any version, **compare** any two, and choose the **baseline** for "what
  changed" — by default the version at the last commit.
- The published site carries one version and its diff against a baseline the publisher chooses
  (`export --site --since <commit>`).
- **Rebuilding history from git** — replaying committed `model.json` files into an empty cache, so a
  fresh clone has history too — is a natural later addition and needs no regeneration.

### 11.5 Safety for anything that runs or writes

- **The Host check is not enough for writes.** It stops another website from *reading* the app
  through a rebound DNS name (decision of 12 Sep). It does not stop that site from *submitting a
  form* to localhost, which arrives with a valid Host. So every action is a `POST` that requires the
  app's own `Origin` **and** a per-session token embedded in the page when `serve` starts.
- **One run at a time,** through a lock the CLI shares: a `generate` started in the terminal shows in
  the app and blocks its writes (ANS-05).
- **Every write checks the file is unchanged since it was loaded** (ANS-04); otherwise it refuses and
  offers to reload.
- **Every network request is logged,** whichever front end started it — already true of runs.

### 11.6 Decisions — taken 2026-10-04

| | Decision |
|---|---|
| C-1 | **The live app runs the engine:** `scan` and `generate` with one click; `generate` with the model only after the confirmation of §11.2, logged like any run |
| C-2 | **The app writes `rules.yaml`** — append-only, after a preview of the exact lines added, with the hash check of §11.5. `rules.yaml` is archdoc's configuration, written by a person, not a documentation section, so hard rule 2 does not cover it. Appending text, never re-serialising, keeps the author's comments and order |
| C-3 | **Settings are read-only for now.** Revisit once the app is in use; nothing here should make editing them harder later |
| C-4 | **The app never acts on git.** It shows which of archdoc's files are uncommitted and their diff, read-only, with the command to copy |
| C-5 | **The published site has no control for now.** Deferred, not refused: decided when a concrete need appears, bearing in mind that anything beyond static files needs a server archdoc does not have |

### 11.7 What the design needs from this

For Claude Design, together with §4:

- a **status pill** in the top bar: *Up to date* · *Code changed* · *Running…* · *Failed* — and, in
  the published site, *Generated from commit … on …*;
- a **Run dialog** with the egress and cost preview, and a **progress drawer** with stages and cancel;
- an **uncommitted files** card with the command to copy;
- a **Publish dialog**: build the site, and the CI workflow to copy;
- **disabled states** for every action while a run is in flight;
- a **read-only settings panel**.
