# archdoc — Stack definition

> **What this document is:** the definition of what archdoc is built with, what each piece
> does, and how the pieces fit together. A reference to come back to.
>
> **What it is not:** the definition of the product. That is `product-definition.md`, which
> stays deliberately free of technology. The evidence behind these choices is
> `survey-test-subjects.md`, cited below as *the survey*.
>
> Decided 2026-08-23.

### Where to look

You are not meant to read this end to end. Sections 1–4 describe **what the stack is**;
5–8 record **why, what it costs, and what is unresolved**.

| If you want to know… | Read |
|---|---|
| What are we using, in one screen | §1 |
| How do the pieces fit together conceptually | §2 → *The map* |
| What is this library and why is it here | §2.3 |
| What exactly is the web app | §2.5 |
| What happens when I run the thing | §3 |
| How the repository and packages are laid out | §3 → *How the code is organised* |
| How does a diagram get drawn, and will it look good | §3 → *How a diagram actually gets drawn* |
| What do I actually get out of it | §3 → *What lands on disk* |
| Why Go and not something else | §5 |
| What do I need to learn | §6 |
| What is still undecided | §8 |

**Boundary with the other documents.** Anything about *what the product is* lives in
`product-definition.md` and is only pointed at here — where the two disagree, the definition
wins. Where a claim rests on evidence, that evidence is in `survey-test-subjects.md`.

---

## 1. The stack at a glance

| Concern | Choice | In one line |
|---|---|---|
| Engine language | **Go** | Compiles to one file that runs anywhere with nothing installed |
| Reading compose files | **`compose-spec/compose-go`** | The same code Docker uses, so it is right by construction |
| Reading files with line numbers | **`gopkg.in/yaml.v3`** | Gives the line each fact came from, for provenance |
| Storing the model | **SQLite** via **`modernc.org/sqlite`** | A database that is just a file — nothing to install or start |
| Arranging diagrams | **`goccy/go-graphviz`** | Decades-old solution to "where do the boxes go", bundled in |
| Talking to Claude | **`anthropic-sdk-go`** | Official client; used only for labels and prose, never facts |
| Bundling assets | Go's **`embed`** | Puts the catalog data and web page inside the binary |
| Local web server | Go's **`net/http`** | Standard library; no framework, no external server |
| Web page | **TypeScript** | Browsers run JavaScript — there is no alternative here |
| Shipping | **A single static binary** | `archdoc` is one file; no runtime, no container, no service |

Nine of these ten are either the standard library or a single library with a single job.
There is no framework anywhere in this stack.

---

## 2. What each piece is, in plain words

Before the piece-by-piece detail, the shape of the whole thing.

### The map

There is **one engine**. Storage and drawing are *parts of it*, not things beside it. What
sits outside the engine is the two ways you talk to it, and the single file it all ships in.

```
   ┌─ archdoc-core — THE ENGINE ──────────────────────────────────┐
   │                                                              │
   │   1. Extract     read config files → facts, with line numbers│
   │   2. Refine      apply your rules.yaml                       │
   │   3. Label       ← the LLM lives HERE, and only here         │
   │   4. Validate    reject anything self-contradictory          │
   │   5. Store       save a numbered version         (SQLite)    │
   │   6. Render      draw from what was stored       (Graphviz)  │
   │                                                              │
   └──────────────────────────────────────────────────────────────┘
              ▲                                    ▲
          the CLI                             the web app
              └───────────── surfaces ─────────────┘
                               │
                    one binary — how it ships
```

**Two things this picture is meant to make obvious.**

*The LLM is one optional step in the middle, not part of drawing.* It never draws and never
decides what exists. It suggests a name, a description, a grouping — those get validated and
stored like any other value, and the renderer later draws whatever the stored model says. It
is downstream of the facts and upstream of the picture, touching neither directly. This is
what §1 of the product definition means by *"the LLM is never responsible for facts."*

*Only step 3 uses the network.* Steps 1, 2, 4, 5 and 6 are pure local computation. Remove
step 3 entirely and you still get a complete, correct, fully traceable diagram — just with
duller labels.

| Layer | What it is | Covered in |
|---|---|---|
| **The engine** | All six steps above. The product itself | §2.1, §2.3 |
| **Surfaces** | The CLI and the web app — how you talk to the engine | §2.1 |
| **Distribution** | The single binary — how it reaches your machine | §3, §4 |
| **Artifacts** | What the engine produces and keeps between runs | §2.4 |

### 2.1 The three things we write

**`archdoc-core` — the engine.**
All of the actual work: find the config files, read them, work out the services and
connections, check the result for contradictions, save it, draw it. It has no screen and no
buttons. **This is the product**; everything else is a way of talking to it.

**The CLI.**
A thin shell. It turns what you type — `archdoc scan` — into a call to the engine, and
prints what comes back. It contains no logic of its own.

**The web app.**
`archdoc serve` starts a local web application in your browser — the diagram, an inspector,
a version timeline, a docs preview and more. Clicking a box jumps to the file that proves
that service exists. It holds no logic of its own, but it is a substantial piece of software.
§2.5 describes it properly.

> **Why one engine and two thin shells?** So the command line and the web page can never
> disagree — they are literally the same code underneath. If they were two programs, they
> would slowly drift apart and you would have two versions of the truth. This was decided in
> §7 of the product definition, before the stack, and holds regardless of it.

### 2.2 The two languages

**Go — for the engine.**
A deliberately small, plain language. Its relevant property here: `go build` produces **one
executable file** containing everything, which runs on a machine with nothing installed. For
a tool that people point at their own repositories, that is the whole install story.

**TypeScript — for the local web app.**
Browsers run JavaScript, so this part has to be written in it. TypeScript is JavaScript with
type checking.

Be careful with the word *thin* here. The web app is thin in **authority** — it holds no
business logic, computes no layout, and decides nothing about your architecture; it asks the
engine and displays the answer. It is **not** thin in surface area: it is a real
single-page application with seven views. See §2.5.

### 2.3 The five libraries

Each one exists to avoid writing something hard. That is the whole reason each is here.

---

**`compose-go` — reading docker-compose files**

*What it is:* the reference implementation of the Compose file format, maintained by the
Compose specification project. It is the code Docker Compose itself runs.

*Its job here:* turn `docker-compose.yml` and its override files into a correct list of
services, images, ports, volumes and dependencies.

*Why not write it ourselves:* because compose files look simple and are not. The survey
found all of this in the two test repositories:

- A value can contain a variable with a fallback inside another fallback:
  `${API_GW_HTTP_PORT:-${KONG_HTTP_PORT:-8000}}`
- There are **seven** different variable forms, and `${V:-x}` and `${V-x}` mean different
  things — the colon changes the meaning
- Files layer on top of each other, and a special marker `!override` means *"replace this
  list, do not add to it"*
- Get that one marker wrong and you silently produce a **wrong** diagram — not a broken one,
  a confidently wrong one

*What to know:* this is the single most important choice in the stack. Everything else could
be swapped later; this one is here because being wrong about facts is the failure this whole
project exists to prevent.

---

**`yaml.v3` — reading files with line numbers**

*What it is:* Go's standard YAML library. It can hand back not just the values in a file but
the **line and column each value came from**.

*Its job here:* provenance. The product promises that every box and arrow links back to the
exact file and line that proves it exists. That requires knowing where each value was found.

*Why both this and `compose-go`:* `compose-go` gives the *correct meaning* of a compose file
after all the layering and variable substitution — but that process may lose track of where
each value originally came from. So the engine likely reads each file twice: once with
`compose-go` for correct meaning, once with `yaml.v3` for line numbers, then matches them up.

*What to know:* whether this second pass is actually necessary is the one unanswered question
in this document. See §8.

---

**SQLite (via `modernc.org/sqlite`) — remembering things**

*What it is:* a database that lives entirely inside **one file**. There is no server to
install, no service to start, no port, no password. Programs open the file the way they would
open any other file.

*Its job here:* remember what the architecture looked like at each commit, so the tool can
answer *"what changed architecturally since last month?"* — which needs stored history, not
just a current snapshot.

*Why SQLite and not plain files:* the tool needs to ask real questions of that history —
follow chains of connections, search text, compare two versions. Those are database
questions. SQLite answers them without being a server.

*Why `modernc.org/sqlite` specifically:* there are two ways to use SQLite from Go. The common
one wraps the original C code, which means the build needs a C compiler and loses the
"compiles for any machine from any machine" property. `modernc.org/sqlite` is SQLite
**translated into pure Go**, so the single-file-binary property survives. It is somewhat
slower. At this project's size — dozens of services, megabytes of history, one user — that
does not matter.

---

**`go-graphviz` — arranging the diagram**

*What it is:* Graphviz, the long-standing graph-drawing program, packaged so it runs **inside**
the Go program instead of being installed separately.

*Its job here:* given twelve boxes and twenty arrows, decide where to put each box so the
diagram is readable and the arrows cross as little as possible. Then produce the SVG image.

*Why not write it ourselves:* this is a genuinely hard problem with a long research history.
Graphviz has been the reference answer since the 1990s.

*What to know:* positions are calculated **once, in the engine, and saved** with each version.
The web page does not calculate anything — it draws boxes where the engine said. This is what
guarantees the picture in the browser and the picture committed to the repository are
identical, and it is why the frontend stays thin.

---

**`anthropic-sdk-go` — asking Claude for names and wording**

*What it is:* Anthropic's official Go client for the Claude API.

*Its job here:* a deliberately narrow one. Suggest a friendly name for a service the catalog
does not recognise, write a one-sentence description of what a service does, group services
into logical layers, and draft prose for the documentation sections.

*What it explicitly does not do:* decide what exists. Services, connections and ports all
come from configuration files. §6 of the product definition makes this measurable — remove
the model entirely and the tool still produces a full, correct, traceable diagram, just with
duller labels.

*What to know:* it is a remote API call over the network. It is never bundled, never
required, and if it fails the tool degrades rather than breaks.

### 2.4 The three artifacts the stack has to handle

The product definition owns the full list of what archdoc produces (§10 data model, §11
pipeline, §8 output contract). Three matter here because they determine what the libraries in
§2.3 are actually for:

| Thing | Plain description | Which library holds it |
|---|---|---|
| **FactSet** | What was found by reading files only — services, connections, and the file and line proving each one. No guessing, no model involved. The dividing line between the certain half of the system and the rest | produced by `compose-go` + `yaml.v3` |
| **The model** | The FactSet refined by your rules, labelled, validated, and **saved as a numbered version**. Cannot be saved in a self-contradictory state | stored in SQLite |
| **The catalog** | A bundled reference list: this image is Postgres, that one is Redis, this URL scheme means a database. Shipped inside the binary, overridable by you | bundled by `embed` |

Everything else — `rules.yaml`, `model.json`, version history, the generated documents — is
defined in `product-definition.md` and is not a stack concern.

---

### 2.5 The web app, up close

`archdoc serve` starts one process that serves a local API and the pre-built page assets on
a localhost port. What it serves is not a static page and not two pages — it is a
single-page application with seven views, specified as `SUR-07` through `SUR-15` in the
product definition:

| View | What it shows |
|---|---|
| **Canvas** | The C4 diagram, with a switch between context and container level |
| **Inspector** | Click any box or arrow: what it is, and the file and line that prove it |
| **Version timeline** | Every past scan, and a diff viewer between any two |
| **Rules viewer** | What your `rules.yaml` changed, and where it took effect |
| **Docs preview** | The generated documentation as it will look |
| **Completeness view** | Which human-written sections are empty, filled, or stale — each linking out to the file in your editor |
| **Run reporting** | Cost and tokens per run, and exactly what left your machine |

**Why it is still called thin.** Every view is a projection of the stored model. The canvas
draws boxes at coordinates the engine already computed and saved; it never lays anything out
itself. That is what keeps the browser picture and the committed SVG identical, and it is why
the app cannot drift away from the CLI — there is nothing in it to drift.

**What it is not.** It is not where the documentation lives — see *What lands on disk* below
— and it is not a text editor. A deliberately narrow editing surface is planned, but it is
`ANS` in **phase R1.c, not the first build**, and its constraints are specified in §4.12 of
the product definition.

---

## 3. How the pieces fit together

### What is inside the single binary

```
archdoc  (one executable file)
├── the engine            reading, checking, storing, drawing
├── compose-go            compose file semantics
├── yaml.v3               line numbers for provenance
├── SQLite (pure Go)      the stored model and its history
├── Graphviz (as WASM)    diagram layout
├── the catalog           bundled reference data
└── the web page          pre-built HTML/CSS/JS assets
```

Nothing on that list is installed separately. Nothing runs unless you run it.

### What happens when you run `archdoc scan`

```
your repo
   │
   ├─ 1. find the config files          which file is the real architecture?
   ├─ 2. read them into facts           compose-go for meaning, yaml.v3 for line numbers
   ├─ 3. apply your rules.yaml          your corrections
   ├─ 4. ask Claude for labels          optional — skip it and everything still works
   ├─ 5. check for contradictions       reject rather than store something invalid
   ├─ 6. save a new version             into the SQLite file
   └─ 7. draw it                        Graphviz for layout, then SVG + Mermaid + arc42
```

Steps 1, 2, 3, 5 and 6 involve no network and no model. Step 4 is the only one that does, and
it is optional.

### What happens when you run `archdoc serve`

One process starts. It serves the local API and the bundled web page on a localhost port.
You stop it with Ctrl-C. It is a foreground program like any development server — not a
background service, not a container, and nothing is left running.

---

### How a diagram actually gets drawn

This is the part most likely to be misunderstood, because it involves two separate steps that
are easy to collapse into one.

```
   the stored model
         │
         ▼
   ┌──────────────────────────────┐
   │ 1. LAYOUT     Graphviz       │   "where does each box go?"
   │               → coordinates  │   pure geometry, no appearance
   └──────────────────────────────┘
         │
         ▼   coordinates saved with this version  (VIE-04)
         │
   ┌──────────────────────────────┐
   │ 2. RENDER     our own code   │   "what does each box look like?"
   │               → SVG          │   C4 styling, entirely ours
   └──────────────────────────────┘
         │
         ├──▶  SVG embedded in the committed markdown
         └──▶  the same coordinates sent to the web canvas
```

**Graphviz decides positions. It does not decide appearance.** Its famously plain default
output never reaches anyone. What the reader sees — service name, technology and description
stacked inside each box, boundaries drawn as containers, external systems greyed out,
protocols labelled on the arrows — is drawn by our own SVG writer from the stored
coordinates. The quality ceiling is set by that renderer, not by Graphviz.

Two consequences worth stating:

- **The browser and the committed file show the identical picture**, because both draw the
  same saved coordinates. That is `VIE-10`'s requirement, and it is structural rather than a
  matter of keeping two renderers in agreement.
- **The layout engine is the cheapest thing in the stack to replace.** It emits numbers.
  Nothing downstream depends on which library produced them.

**The honest risk.** Graphviz's `dot` is weakest at exactly what C4 leans on: nested
boundaries, and arrows routed around them. With a dozen services and a few boundaries it is
very likely fine; if boundary-heavy diagrams come out awkward, the fallbacks are laying out
each boundary separately and composing the results, or swapping the layout engine — cheap,
per the point above.

**One quality limit that is not ours to fix.** Alongside the SVG, the tool emits a Mermaid
version of each diagram for portability and diffing. Mermaid does its **own** layout, so the
Mermaid picture will not match the app's, and its C4 support is still experimental. The
product definition accepts this knowingly — that is why it emits both formats rather than
choosing one: SVG for fidelity, Mermaid for hand-editing and for rendering natively on
GitHub.

### How the code is organised

**One repository.** Not a monorepo — that word describes many projects sharing a repo
(Supabase's `apps/`, `packages/`, `docker/`). archdoc is *one product in two languages*,
because browsers run JavaScript and there is no way around that.

The decisive reason is §7 of the product definition: *"The CLI and the web app must never
become two implementations."* Separate repositories institutionalise exactly that drift. Three
reinforcements: `embed` needs the built frontend present at Go compile time; a change to the
model's shape touches engine and frontend in one commit; and repository splitting solves
team-boundary problems that a single author does not have.

**Packages follow architectural seams, not the specification's chapters.**

The capability catalog (§4 of the definition) groups 109 capabilities as `DSC`, `EXT`, `MDL`,
`VAL`, `SEM`, `VIE`, `PRV`, `MEM`, `RUL`, `SUR`, `OUT`. That is a planning structure, and
mapping it onto directories was rejected: it breaks immediately on `PRV`, which is not a stage
but a property every fact carries, and on `FactSet`, which four groups would need to share.
When a mapping needs exceptions before any code exists, it is the wrong mapping.

Each package below earns its place from a boundary the definition or the survey already
requires:

```
archdoc/
├── cmd/archdoc/       CLI entry point — thin
├── internal/
│   ├── archdoc/       core types: Fact, Provenance, FactSet, Node, Edge, Model, Version
│   ├── extract/       discovery + the parser registry
│   ├── store/         storage interface + the SQLite driver
│   ├── semantic/      the only package permitted outbound network access
│   ├── render/        layout, SVG, Mermaid
│   └── serve/         local HTTP + embedded web assets
├── web/               the TypeScript application
├── catalog/           embedded reference data
└── testdata/          fixtures
```

| Package | The seam it represents | Required by |
|---|---|---|
| `archdoc` | The FactSet contract between the deterministic and probabilistic halves | §11 |
| `extract` | The parser registry — five gateway formats across two repos, with a long tail | Survey §4.4 |
| `store` | The storage driver interface, kept as an escape hatch to a server-backed store | §7 |
| `semantic` | The network boundary | §6, AC-8 |
| `render` | Layout and appearance, downstream of the stored model | §4.6 |
| `serve` | Engine versus surface | §7 |

`validate`, `rules` and `output` begin as files inside the packages that use them, and split
out when coupling earns it rather than on the strength of having a section number.

**Making the network boundary structural is deliberate.** If `semantic` is the only package
that may make outbound calls, AC-8 — *"`structure-only` transmits zero file contents"* —
becomes provable by a test that inspects imports, rather than an audit performed by hand.

**Types are settled before directories.** Directory names are refactorable in minutes; the
shape of `Fact`, `Provenance`, `FactSet`, `Node`, `Edge`, `Model` and `Version` propagates
into every signature, every test, and into `model.json` — which is a *committed deliverable*,
so its shape is close to permanent. The two interfaces (storage driver, parser registry) are
settled at the same time, for the same reason.

### Two build decisions

**Building from source requires Node; using archdoc does not.** `//go:embed web/dist` will
not compile if `dist/` is absent, so the repository carries a placeholder `index.html` and a
build script that builds the frontend before the binary. Committing real build output was
rejected — generated artifacts in version control rot silently. Users are unaffected: they
receive a binary.

**The development loop uses a build tag.** Rebuilding the Go binary to see a CSS change is
untenable, so a `dev` build tag proxies to the frontend dev server with hot reload, while the
default build embeds `dist/`. Worth setting up in week 0; it bites on the first day otherwise.

---

### What lands on disk

Full contract in §8 of the product definition. What the stack has to deliver:

> **Markdown is the deliverable. The app is the workbench.**

| Layer | What | Committed? |
|---|---|---|
| 1 | `model.json` — machine truth, not meant for reading | Yes |
| 2 | **Markdown + diagrams — the primary deliverable**, in `docs/architecture/` | **Yes** |
| 3 | A static site (MkDocs Material) — optional | Usually not |
| 4 | `archdoc serve` — a running process, not a file | No |

Two rules from §8 that constrain the stack directly:

- **The output must be fully readable with archdoc absent.** Committed markdown with embedded
  SVG, plus Mermaid that GitHub renders natively — no site generator, no build step. This is
  why the renderer emits plain files rather than driving a viewer.
- **Generated files are overwritten wholesale; human-written files are never written to, and
  never even read** — only linked. No merging, so no merge step to get wrong.

---

## 4. The rules this stack had to obey

These come from §7 of the product definition and were fixed **before** the stack was chosen.
Any candidate that failed one was disqualified.

| Rule | Why it exists | How this stack satisfies it |
|---|---|---|
| **No background service, ever** | Installing and running a database server to draw a diagram is absurd | SQLite is a file |
| **No container required** | The tool reads *your* repositories and must open files in *your* editor | One native binary |
| **Nothing installed alongside it** | Every extra dependency is a reason the tool does not run | Graphviz and the web page are bundled in |
| **One engine, two thin shells** | So the CLI and web app cannot disagree | The engine is a library both call |
| **Same input, byte-identical output** | Trust depends on repeatability (AC-7) | Ordered data structures, explicit sorting — see §6 |
| **The model is never required to draw** | The product must work without an API key | Only step 4 above touches the network |

---

## 5. Why these choices

The full reasoning is condensed here rather than argued at length.

### The one argument that decided it

The hardest and most dangerous part of this project is reading configuration files
*correctly*. Everything else can be wrong in a visible way; this can be wrong in an
**invisible** way.

| If this is wrong | Result | Would you notice? |
|---|---|---|
| Compose file merging or variables | A confidently **wrong** diagram | **No** |
| Diagram layout | An ugly diagram | Immediately |

Go is the only candidate language with the Compose format's **reference implementation**
available as a library. Every other language means writing that logic yourself, in the exact
place where mistakes are invisible. That, plus the single-binary install story, decided it.

### What was rejected, and why

| Option | Its real advantage | Why not |
|---|---|---|
| **TypeScript everywhere** | One language for everything; best library for tracking line numbers; the author already knew it | The compose-reading logic would have to be written by hand, permanently owned, in the place where errors are silent. Also needs Node installed, or a 50–100 MB bundle |
| **Rust** | Safest repeatability; best tools for the odd formats (Caddyfile, nginx) | No reference compose library either, weaker diagram options, and slowest to get a first working version — which §6 of the definition wants first |
| **Python** | Familiar, good YAML tools | Worst install story of the four, and its usual advantage (machine-learning ecosystem) is irrelevant since Claude is just an API call |
| **Shelling out to `docker compose config`** | Free correct answers | Would require Docker installed **and running** to draw a diagram, contradicting the product's identity. Still useful as a **testing** reference — see §8 |

---

## 6. What has to be learned

The author had not used Go when this was decided. Recorded as a real cost rather than glossed
over. It was accepted because the alternative is not cheaper — it spends the same time writing
a compose parser by hand — and because a config parser is not this project's contribution.

This project needs a small, unexciting slice of the language: structs and maps, errors as
return values, interfaces for the storage and parser boundaries, `encoding/json`,
`filepath.WalkDir`, `embed`, and `net/http`. No concurrency design, no generics puzzles, no
performance tuning. Goroutines are essentially optional here.

> **The one real trap.** Go deliberately shuffles the order in which you get items out of a
> map, so the same input can produce differently-ordered output on each run — which would
> break the byte-identical requirement (AC-7). The fix is mechanical: sort the keys before
> looping. It is a **loud** failure rather than a subtle one, and AC-7's five-run test exists
> to catch it.

---

## 7. Costs accepted

| Cost | Detail | Why it is acceptable |
|---|---|---|
| **Two languages** | Go engine, TypeScript page | Unavoidable — browsers run JavaScript. The page is kept thin: it draws saved positions and nothing more |
| **Two odd formats written by hand** | Caddyfile and nginx.conf have no usable Go libraries; nginx also needs two-step name resolution | Contained work behind the parser boundary, not engine work |
| **Sorting discipline** | Every map loop that produces output must sort first | Mechanical, and AC-7 catches it |
| **Learning Go** | Author was new to it | One-time cost that ends; writing a parser is a cost that does not |
| **Possibly reading files twice** | For line numbers — see §8 | More work, not more risk |

---

## 8. Still open

**O-8 — do line numbers survive compose file merging?**

`compose-go` gives the correct final meaning of a set of compose files. The product promises
every fact links to the file and line that proves it. It is not yet known whether that
position information survives the merging and variable substitution.

- **If it does:** one pass, simpler than expected.
- **If it does not:** the engine reads each file twice — `compose-go` for meaning, `yaml.v3`
  for positions — and matches them up. More work, but no more risk, and the same approach
  would be needed in any language.

This should be settled before building, because the product's central promise depends on it.
The two files to test against are the hardest ones the survey found:
`supabase/docker/docker-compose.kong.yml` (the `!override` marker across a merge) and
`immich/.devcontainer/server/container-compose-overrides.yml` (two variables joined in one
value, plus `!reset`).

**Not decided here, and not stack questions:** test subject #3 (O-7) and the R1.b build
ordering (O-6) both remain open in the product definition. Which catalog entries to seed is
curation work, unblocked but not done.

---

## Appendix A — effect on the capability catalog

§13 of the product definition promised that choosing the stack would mean revisiting the
capability catalog to see what became cheaper or more expensive. That pass, against the 109
capabilities of §4:

**No capability is cut.**

**Cheaper**

| Area | Why |
|---|---|
| **EXT** — reading compose files | `compose-go` does the hardest part. The largest single saving in the catalog |
| **VIE-03 / VIE-10** — layout and SVG | `go-graphviz`, with nothing to install |
| **SUR** — the local web app | `embed` + `net/http`: one binary, one process |
| **MDL / MEM** — storage, versions, diff | SQLite handles history and queries without a server |
| **OUT** — distribution | A single binary removes the install story as a design problem |
| **SEM** — structured model output | The Go client supports strict tool schemas, so Claude's structured reply is guaranteed well-formed. The validator then spends its retries on *meaning* rather than malformed JSON |

**More expensive**

| Area | Why |
|---|---|
| **EXT** — Caddyfile, nginx.conf | Written by hand; nginx needs two-step name resolution |
| **VIE-08** — the interactive canvas | A separate TypeScript codebase, though a thin one |
| **All deterministic capabilities** | Sorting discipline wherever a map is looped over |

**New prerequisites**

- **PRV-01** may need the two-pass reading of §8.
- **A parser boundary earns its place.** Six config formats, with more beyond them — the
  survey found five gateway formats across two repositories alone. Parsers should plug into a
  common interface driven by catalog data, so a new format is an addition rather than a
  change to the engine.
