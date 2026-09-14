# archdoc — Vision: beyond configuration

> **What this document is:** a proposal for where archdoc goes after R1.a — the problem as it
> is now understood, the direction, and every decision that direction forces, each with options
> and a recommendation. Written to be refined, then folded into `product-definition.md`.
>
> **What it is not:** settled. Until a decision here is taken and recorded in `decisions.md`,
> **`product-definition.md` still wins** wherever the two disagree.
>
> Drafted 2026-09-14, after R1.a's engine and web app were complete. Horizon: end of 2026.

### Where to look

| If you want… | Read |
|---|---|
| Why this exists at all | §1 |
| Why not just ask the AI that wrote the code | §1.5 |
| The direction in one page | §2.1–§2.3 |
| What stays exactly as it is | §4, §5 |
| What must be decided before building | §6 — the core of this document |
| What to do first | §8 |

---

## 1. The problem

### 1.1 The original problem, restated

AI now writes code faster than any person can read it. The understanding that used to form
while typing never forms, and it is not written down afterwards because nobody ever had it.
archdoc exists to give that understanding back: **to help people understand software an AI
built for them.**

Architecture was chosen as the first target because it sits between the concept and the code —
the layer where a person can hold the whole system in their head. That instinct was right. What
R1.a got wrong was not the layer, but the evidence it builds that layer from.

### 1.2 Who it is for — how much people know about what was built

Two things vary between people building with AI: how much of the **concept** they hold, and how
much **control over the code** they keep. AI erodes the second for almost everyone; what differs
is how much of the first survives.

| Who | Concept | Code | What archdoc must give them |
|---|---|---|---|
| **Planned team** — requirements, specs, design docs, often QA | Full. They know what they are building and why | Losing some — the AI writes more than they review | Confirmation that the code still matches the design they know, and what changed |
| **Developer with plans** — `.md` plans the AI follows | Requirements, and part of the concept | Partial. The AI follows the plan; a technical developer can spot-check it | Plan → code: where each part of the plan lives, and where the AI deviated |
| **Technical prompter** — no plan, technical vocabulary | Drifts: nothing anchors it, and the AI may miss the concept | Low. Without a plan, control over the code goes | The concept that *actually emerged*, and where it differs from what was meant |
| **Vibe coder** | Maybe the requirements | None, and no technical knowledge | Everything, top down: what it does, how data moves, what it touches |
| **Expert using AI as reviewer or assistant** | Full | Full | Speed: the delta, and a map to navigate it — not an explanation |

The common gap in every row is the bridge between concept and code.

### 1.3 What R1.a proved, and where it stops

R1.a works. It reads Compose files, env files and gateway configuration; every element cites a
file and line; the structure is identical with the model on and off (AC-2); seven of nine
acceptance criteria are met. Its engineering is the foundation of everything below.

Its limits are the input, not the engine:

1. **Architecture from configuration is useful only for multi-service systems.** A React app, a
   CRUD backend or a standalone script has no Compose file, so archdoc says nothing about it.
2. **Only YAML is read.** Most of what an AI generates never passes through a Compose file.
3. **Container level says little even when it works.** For Immich, archdoc draws five boxes
   joined by "depends on", with `immich-machine-learning` unconnected.

Immich at the pinned revision holds far more, all of it provable at a line:

| In the code | Count / location |
|---|---|
| Server controllers | 46 |
| Server services | 54 |
| Database tables | 64 |
| Web routes | 55 |
| Machine-learning service | a FastAPI app, `machine-learning/immich_ml/main.py` |
| The edge the diagram is missing | `server/src/dtos/config.dto.ts:624` — `http://immich-machine-learning:3003` |

The gap between those two pictures — five boxes against how Immich actually works — is the
problem this document addresses.

### 1.4 The landscape

| | Reads | Who decides what is drawn | Can a claim be checked? |
|---|---|---|---|
| **DeepWiki** (Cognition) — the closest reference | Any repository | An LLM, from the code | Partly: pages cite source files, but the model chooses the citations and draws the diagrams, and nothing checks either |
| **Archify** — an agent skill | Whatever the agent read | The agent authors the diagram IR, layout included | No: sources are optional decoration chosen by the agent |
| **archdoc R1.a** | Compose, env, gateways | A parser | Yes — every element opens at its line |

Both alternatives cover any repository because nothing they draw has to be true. archdoc's
position is the opposite: **everything it draws can be checked**. The direction below keeps that
position and removes the coverage ceiling.

### 1.5 What archdoc gives that asking the AI does not

The AI that wrote the code can also explain it — any coding agent will walk a person through a
repository on request. R1 argued archdoc is not a wrapper because of determinism and
persistence. Both still hold, but for the people this vision adds they are not the strongest
reasons. In order of strength:

1. **It shows what you did not know to ask.** An answer covers the question; a map covers the
   system. A vibe coder cannot ask about a table they do not know exists, a service nobody told
   them about, or a call to a paid API they never approved. The map shows all of it without being
   asked. This is the single strongest argument for the product.
2. **It cannot invent structure.** A chat answer may describe a component that does not exist,
   and the person asking has no way to tell. archdoc cannot draw one: existence comes only from
   extraction.
3. **It tells you what changed.** A chat has no memory of what the system looked like last week.
   archdoc keeps every version and reports the difference — the natural unit of understanding
   when code arrives faster than it can be read.
4. **It is the same answer every time.** Asked twice, a chat answers twice, differently. The map
   is regenerated from the code and changes only when the code does.
5. **It stays on the machine by default.** Structure only leaves unless the user decides
   otherwise, and every byte that leaves is logged.

None of these depends on the reader being able to read code. That matters for §3's point on
provenance as a guarantee.

---

## 2. The vision

### 2.1 Identity unchanged, scope widened

**Architecture documentation remains the product.** The identity sentence stands:

> archdoc reads an existing codebase and produces the architecture documentation it should have
> had — accurate, traceable to the code, and regenerable as the system changes.

What changes is the evidence: **the code, and configuration when present** — not configuration
only. The new capabilities are not a drift from architecture documentation; they are what
architecture documentation looks like when it is done properly (§2.2).

**"Architecture" is meant in arc42's sense, not the narrow one.** Not only the boxes and the
lines between them, but how the system behaves at runtime (§6), the concepts that cut across it
— its domain model, its data — (§8), and the vocabulary it uses (§12). The identity holds only
under this definition: for a single React app, most of the value lies in features, flows and
data, which the narrow sense would exclude and arc42 does not.

### 2.2 One model, many lenses, several zoom levels

One validated model, looked at through lenses. Each lens answers one question a person asks,
and each already has a home in arc42 and C4 — homes that are empty or thin today. Archify's five
diagram types fall into the same lenses.

| Lens | The question | Evidence | arc42 | C4 | Archify type |
|---|---|---|---|---|---|
| **Boundaries** | What does it talk to? | SDK imports, HTTP clients, env values, Compose | §3 Context and scope | System context | Architecture |
| **Containers** | What are the running pieces? | Manifests, frameworks, Compose | §5 level 1 | Container | Architecture |
| **Structure** | How is it organised? | Modules, imports, framework conventions | §5 level 2 | **Component** | Architecture |
| **Features** | What does it *do*? | Entry points: routes, pages, CLI commands, `main`, exported APIs | §1 (the requirements overview, extracted) | — | — |
| **Flow** | What happens when…? | Call paths from an entry point | §6 Runtime view | **Dynamic** | Sequence, Workflow |
| **Data** | What does it store, in what shape, in what states? | ORM models, schemas, migrations, types, status enums | §8 Cross-cutting (domain model), §12 Glossary | — | Data flow, Lifecycle |
| **Deployment** | Where does it run? | Compose, CI files | §7 Deployment view | **Deployment** | Workflow (CI) |
| **Change** | What did the AI just do? | Versions, git history | — (the version diff) | — | Delta mode |
| **Intent vs actual** | Did it build what was asked? | Plans the user nominates (§6, D-9) | §1, §10 | — | — |

A side finding: **Compose mostly describes deployment.** R1.a used it for §5; with code as
evidence it moves to §7, where it belongs, and corroborates what the code shows.

**Zoom levels.** Every lens reads at several depths: conceptual (what a vibe coder needs),
system, container, and component. **The code level — functions and variables — is out of scope
for this version.** Parsing the code is still required to build the levels above it.

### 2.3 Two families of output from one model

```
        Evidence: configuration + code (three tiers, §2.4)
                              │
             One validated model — provenance on everything
                              │
          ┌───────────────────┴───────────────────┐
  REFERENCE DOCUMENTATION                  EXPLANATIONS
  C4 + arc42, Markdown, committed          interactive, guided, exportable
  "what the system is"                     "how the product works"
  engineers, future maintainers            whoever has to understand it now
```

- **Both are projections of the same model**, so they cannot contradict each other (P3).
- **Both cite file and line** for every element.
- **Reference documentation stays the core deliverable.**
- **Explanations are the new value**: they explain the product to its user the way Archify
  does — sequence, data flow, lifecycle and workflow diagrams, and guided stories — with nothing
  hallucinated.

What the web app already has maps onto what Archify's viewer offers:

| Archify viewer | archdoc |
|---|---|
| Semantic Passport | The inspector — built |
| Route probe, upstream/downstream reach | Traversal over proven edges only — cheap |
| Semantic lenses | The lenses of §2.2 |
| Reading depth by zoom | The zoom levels of §2.2 |
| Before/Delta/After | The version diff — built |
| Stories (guided chapters) | **New:** a guided tour whose every chapter cites model elements |

### 2.4 Evidence in three tiers

| Tier | Reads | Gives | Cost |
|---|---|---|---|
| **1 — Configuration** | Compose, env, gateway configs, CI files | Containers, deployment, declared wiring | Built (R1.a) |
| **2 — Generic code** | Any supported language: files, modules, imports, entry points | A baseline structure for *every* repository | One parser per language |
| **3 — Framework packs** | Routes, pages, ORM models, HTTP clients, dependency injection | The meaning: features, data, flows | One pack per framework |

A property of the target helps: **AI-generated code concentrates on a few popular stacks** —
React / Next / Vite, Express / NestJS, FastAPI / Flask / Django, Prisma / TypeORM / SQLAlchemy.
A small number of framework packs covers most of what people will point archdoc at.

### 2.5 Views are earned by evidence

A container diagram with one box is noise. A view — and a document section — appears only when
the evidence supports it.

| Project | The views that carry the meaning |
|---|---|
| Multi-service system | Context, containers, deployment, cross-service flows |
| Front / back / database | Containers (three), components per container, data model, flows from UI action to API to database |
| React app | Context (the APIs it calls), components (pages, feature areas), flows (action → state → API) |
| CRUD backend | Data model at the centre, routes as features, request flows |
| Script | Context (files, APIs and environment it reads and writes), a pipeline flow of its steps |

### 2.6 What each audience gets first

| Who | Starts with |
|---|---|
| Vibe coder | Explanations: a guided story, the feature map, flows |
| Technical prompter | The emerged concept — features and structure — and where it drifted |
| Developer with plans | Intent vs actual; flows to spot-check |
| Planned team | Reference documentation, the change lens, conformance to their design |
| Expert | The change lens and the navigable map |

### 2.7 Immich, as it would be

- **Containers:** server (NestJS), web (SvelteKit), mobile (Flutter), machine-learning
  (FastAPI), Postgres, Redis — connected, with the server → machine-learning edge cited at
  `config.dto.ts:624`.
- **Components:** the server's feature areas — assets, albums, auth, jobs… — from its controllers
  and services.
- **Data:** a domain model from its 64 tables.
- **Flows:** "upload a photo" — web → controller → service → job queue → machine-learning →
  database.
- **Deployment:** the Compose file, where it belongs.

---

## 3. Principles — kept, and refined

The two governing principles and the five properties of proper documentation (P1–P5) hold
unchanged. The larger role for the LLM requires the second principle to be made precise:

1. **Existence and relationships are extraction-only.** A feature, module, table or call exists
   in the model only if a file proves it.
2. **Meaning may come from the LLM, but only as cited claims.** Every sentence of an
   explanation points at model elements; the validator rejects a claim whose citations do not
   resolve. This is today's fencing of labels, extended to prose.
3. **Interpretation is always visibly marked.**
4. **Layout is presentation, not fact.** The LLM may *propose* an arrangement; the validator
   guarantees every box maps to a model element and that nothing is added or removed (D-12).
5. **Three states of truth, shown everywhere:** *proven* (extracted), *interpreted* (the LLM,
   cited), and *unresolved* — evidence that something exists that archdoc could not identify.
   **Unresolved is never silently dropped** (D-6).

### Provenance serves two ways

A citation is only useful to someone who can read what it points at. **The people who most need
archdoc — vibe coders — cannot.** They also receive the most interpreted content: code with
little structure offers framework conventions little to extract (D-2), so grouping and meaning
lean hardest on the LLM exactly where the reader can least check it.

So provenance does two jobs, one per kind of reader:

| Reader | Provenance is… | What makes it visible |
|---|---|---|
| Can read code | **A link** — follow it, see the line | `file:line` on every element, opening in the editor |
| Cannot read code | **A guarantee** — nothing on this map was invented | The three truth states; a coverage report of what archdoc could not see; plain-language framing of both |

The guarantee is the same property as the link; only its presentation differs. How each
audience is shown it is D-17.

---

## 4. Interfaces

The four interfaces stay. Their roles shift.

| Interface | Today | With this vision | Change needed |
|---|---|---|---|
| **CLI** | `scan`, `generate`, `history`, `runs`, `serve` | The same commands | Flags to choose lenses and scope |
| **Markdown** | The deliverable | Still the deliverable, for reference documentation | Explanations need their own export (D-10) |
| **Storage** | SQLite cache + committed `model.json` | The same design | Scale: §7's "dozens to low hundreds of nodes" no longer holds (D-11) |
| **Web app** | The workbench | The main surface for explanations | §8's "the app is the workbench" is reworded |
| **Agent** *(not built)* | — | An MCP server or skill over the model | Timing depends on D-17. `model.json` as the contract keeps it cheap |

The people archdoc is for already work inside Claude Code, Cursor or similar. An agent interface
meets them there — "how does signup work?" answered from the model, with citations. It is R2b
arriving earlier; nothing now should make it harder.

---

## 5. What does not change

- **Determinism** of extraction (NFR-5, AC-7).
- **Human-owned sections** of the generated documentation are never read or written (OUT-02,
  OUT-03, NFR-11).
- **Provenance** on every element and every value.
- **Rules** that survive regeneration.
- **Validation** before anything is stored; nothing partial is ever written (NFR-7).
- **One binary, zero services**, no container runtime (NFR-9, NFR-10).
- **Egress is declared, minimal by default, and logged byte for byte** (NFR-3, AC-8).
- **The deterministic output is complete without the LLM** — AC-2's thesis extends: every lens
  renders from extraction alone; the LLM only makes it more legible.

---

## 6. Decisions required

Each decision: the problem, the options, a recommendation, and what it touches. Grouped by
concern. None is taken until recorded in `decisions.md`.

### A. Truth and stability

#### D-1 — Stability under a much larger LLM role *(the most important)*

**Problem.** The change lens needs elements to keep their identity from run to run (MDL-13).
Prose regenerated on every run churns committed documents and fills the diff with noise.
Labelling ~25 Supabase elements takes about two minutes today; thousands of elements plus
stories would break NFR-2 (generate under 3 minutes) and cost far more.

**Options.**
- (a) Regenerate every interpretation on every run.
- (b) **Interpretation memory:** store every LLM output keyed by a fingerprint of the facts it
  was based on; re-ask only for elements whose facts changed.
- (c) Interpret only on explicit request, element by element.

**Recommendation: (b).** It makes output stable across runs, makes the change lens meaningful,
cuts cost to the size of the change, and keeps NFR-2 reachable. (c) remains useful on top of it
in the web app. The store already fingerprints versions and keeps layouts per version; this
extends the same idea to interpretation.

**Touches:** MDL-13, MEM-05, SEM-*, NFR-2, NFR-6, `internal/store`.

#### D-2 — How files become components

**Problem.** Turning many files into a few meaningful boxes is the legibility problem (P5).
§3 argued container level made it nearly free because services are already the boxes; code
evidence brings it back. It is the research problem R1.b named.

**Options.**
- (a) Directories only.
- (b) **Framework conventions first** — NestJS modules, Django apps, Next.js route segments,
  directories — as deterministic groups, with LLM regrouping as a stored refinement on top (D-1).
- (c) LLM clustering from scratch.

**Recommendation: (b).** Conventions give stable, explainable groups for free; the LLM refines,
the validator checks every file is placed exactly once, and the refinement is remembered.
`rules.yaml` corrects either.

**Touches:** MDL-*, SEM-*, RUL-*, the R1.b research scope.

#### D-3 — Element identity when code moves

**Problem.** AI-written code is refactored constantly; files move and rename. Identity by path
makes every move a removal plus an addition.

**Options.**
- (a) Identity by path.
- (b) Identity by what the element *is* — route path, table name, exported symbol, framework
  module name — with the path as provenance only.
- (c) Similarity matching between versions.

**Recommendation: (b), with (c) later** for what (b) cannot name. MEM-06 (a rename is not a
delete plus add) depends on it.

**Touches:** MDL-13, MEM-05, MEM-06, AC-6.

### B. Evidence

#### D-4 — Precision versus the single binary

**Problem.** A syntax parser such as tree-sitter gives structure, not types. Resolving "this
controller calls that service" in NestJS needs type information — in practice the TypeScript
compiler, which needs Node. NFR-10 says users install one package and nothing else.

**Options.**
- (a) Syntax only (tree-sitter, run as WASM like Graphviz is today): heuristic resolution.
- (b) Require Node and the TypeScript compiler.
- (c) **Syntax as the baseline; type-aware indexing as an optional precision upgrade** when the
  runtime is present.

**Recommendation: (c).** The binary stays self-sufficient; precision improves where it can.
Provenance records *how* a link was resolved — by name or by type — so the reader sees the
difference.

**Touches:** NFR-10, `docs/stack-decision.md`, EXT-*.

#### D-5 — Discovery: applications, not Compose files

**Problem.** Discovery today means "find the right Compose file". Immich is five applications in
one repository — server, web, mobile, machine-learning, CLI — most invisible to Compose.

**Recommendation:** discover **applications** from manifests (`package.json`, `pyproject.toml`,
`pubspec.yaml`, `go.mod`…) at any depth, then attach Compose services to them as deployment
evidence. Compose becomes one signal among several.

**Touches:** DSC-*, `internal/extract`.

#### D-6 — Unresolved links

**Problem.** Static analysis will miss dependency injection, events, and URLs built from
strings. Dropping what cannot be resolved repeats the Immich machine-learning failure at scale.

**Recommendation:** represent unresolved references as first-class — "calls something at
`${IMMICH_MACHINE_LEARNING_URL}`", cited — rendered in the third truth state (§3), and report
**coverage per repository**: what archdoc resolved, and what it could not see.

**Touches:** MDL-*, PRV-*, VIE-*, a new acceptance criterion (D-14).

#### D-7 — First languages and frameworks

**Recommendation:** TypeScript/JavaScript and Python first — they cover React, Express/NestJS,
FastAPI/Flask/Django and scripts, which is most of what AI generates. The first framework pack
is chosen by the new subjects (§8): **NestJS on Immich** is the natural first target, with a
concrete bar — recover the server → machine-learning edge and the server's feature areas.

### C. Privacy

#### D-8 — Egress for code

**Problem.** Structure-only today means names, paths and edges, no file contents (NFR-3). Route
paths, table and column names, and function names are names — but they come out of file
contents. The boundary must be exact, because AC-8 is measured on it.

**Options.**
- (a) Structure-only covers symbols: names of files, modules, routes, tables, columns and
  exported functions — never bodies, literals or comments.
- (b) `excerpts` (already in NFR-3) as an explicit opt-in, per run, logged byte for byte.
- (c) `local` (already in NFR-4): a local model, nothing leaves.

**Recommendation: all three, as NFR-3 already shaped them — (a) as the default.** Write the
symbol rule down exactly and extend AC-8 to test it. Treat the local mode as first-class, not
an afterthought: this is people's private code.

**Touches:** NFR-3, NFR-4, AC-8, `internal/semantic`.

#### D-9 — Intent sources

**Problem.** Intent vs actual needs the plans and requirements people wrote — human prose. The
spirit of hard rule 2 is that archdoc does not consume people's writing, and structure-only
egress forbids sending it.

**Options.**
- (a) Never read intent.
- (b) **Opt-in:** the user names intent files in `.archdoc/config.yaml`; they are read only then,
  and sent to a model only under `excerpts` or `local`.
- (c) Detect plan files automatically.

**Recommendation: (b).** Hard rule 2 keeps its exact scope — the human sections of archdoc's own
output are never read — and intent becomes an input the user explicitly hands over.

**Touches:** hard rule 2 (clarified, not weakened), NFR-3, a new capability group.

### D. Outputs

#### D-10 — Where explanations live

**Problem.** NFR-12 says generated output renders with archdoc uninstalled. Explanations that
exist only in the web app break it.

**Options.**
- (a) Web app only.
- (b) **Self-contained HTML export** of each explanation — what Archify ships — plus a static
  fallback in the Markdown (a sequence diagram as SVG, a flow as a table).

**Recommendation: (b).** §8 is reworded: Markdown is the deliverable for reference
documentation; explanations are delivered as self-contained HTML; the web app is where both are
explored.

**Touches:** §8, NFR-12, OUT-*, `internal/render`.

#### D-11 — Scale: diagrams and storage

**Problem.** Immich at component level means hundreds of elements. One diagram cannot hold them,
and one `model.json` per version in git becomes large and its diffs unreadable.

**Recommendation:** overview diagrams capped at a readable size (C4's own guidance is roughly a
dozen or two elements), with focus views for the rest; `model.json` split per application when a
threshold is crossed — measure before choosing the threshold.

**Touches:** §7 scale check, VIE-*, OUT-*, `internal/store`.

#### D-12 — LLM-proposed layout

**Problem.** Archify concluded that "layout is the product" and lets the model place every box.
archdoc lays out with Graphviz, which is correct and plain. Position is not a fact, so a model
proposing arrangement does not violate the second principle.

**Recommendation:** keep deterministic layout for the structured diagrams (sequence, lifecycle,
workflow — lanes and time lay themselves out); **experiment** with model-proposed arrangement for
the free-form architecture graph, validated so every box maps to an element and nothing is added
or dropped, and remembered under D-1. Decide on evidence from the experiment.

**Touches:** VIE-03 (already superseded once), `internal/render`.

#### D-13 — Output that scales with the project

**Problem.** Twelve arc42 sections for a 200-line script is absurd.

**Recommendation:** extend "views are earned by evidence" (§2.5) to documents: a single page for
a small project, full arc42 for a system. §8 already holds that partial output is the honest
output; this applies it to size.

**Touches:** OUT-*, §8.

### E. Scope and process

#### D-14 — How success is measured, and the hypothesis the concept rests on

**Problem.** Today's acceptance criteria measure correctness. The goal is now human
understanding, which they do not measure. More than that: **the whole vision rests on one
unproven hypothesis — that a verified map helps a person understand software an AI built.** If
it is false, correctness does not matter.

**Recommendation:** keep the correctness criteria and extend them — coverage and unresolved
rate per repository; flow accuracy against hand-traced flows — and treat the **comprehension
test** as the concept's central experiment, not as one metric among several:

- people answer questions about an unfamiliar, AI-built app, measured on correctness and time;
- three conditions: **the code alone**, **asking a coding agent**, and **archdoc** — the second is
  the real competitor (§1.5), and a comparison that leaves it out proves little;
- include questions about things the participant was never told exist, which is where §1.5
  predicts archdoc wins;
- run a small pilot **early**, on R1.a's output and paper mock-ups of the new lenses, before
  building much. It is cheap, and it is the result most able to redirect the work.

It is also the result the report most needs.

**Touches:** §12, AC-3 (its hand-drawn reference becomes one of several answer keys), D-17.

#### D-15 — When archdoc runs

**Problem.** AI produces code continuously; a manual `generate` is friction, and the change lens
is most useful per commit or per AI session.

**Options:** manual; a file watcher inside `serve` (§7 already allows it); a git hook; an agent
calling archdoc after each task.

**Recommendation:** manual plus the watcher first; the hook next; the agent with the agent
interface (§4). All four run the same engine.

#### D-16 — The roadmap

**Problem.** R1.b, R1.c and R2 were drawn for a configuration-only product.

**Recommendation, to refine:**

| Phase | Content |
|---|---|
| R1.a — done | Configuration as evidence; the whole spine; the web app |
| **R1.b — Code as evidence** | Tiers 2 and 3 (§2.4) for the first stacks; the Boundaries, Containers, Structure, Data and Features lenses; D-1 to D-9 |
| **R1.c — Explanations** | Flow, sequence, lifecycle, workflow; guided stories; HTML export; the change lens per commit |
| **R1.d — Interaction** | The former R1.c: chat refinement, editable canvas, answer surface |
| R2 — Authority | Unchanged; intent vs actual (D-9) is its descriptive first step, and the agent interface brings R2b nearer |

`docs/delivery-schedule.md` and the calendar are **not** updated by this document. They change
only when this roadmap is agreed.

### F. Audience and delivery

#### D-17 — Trust and delivery, by audience

**Problem.** Two assumptions carried over from R1 do not hold for the whole audience of §1.2:

1. *That the reader can check a citation.* A vibe coder cannot (§3, "Provenance serves two
   ways").
2. *That the reader will open documentation.* Many people building with AI never open a docs
   folder; they ask their agent. A map nobody opens helps nobody — however correct it is.

**Options.**
- (a) One presentation for everyone; documents first, the agent interface later.
- (b) **Audience-aware presentation over the same model:** technical readers get citations,
  documents and the inspector; non-technical readers get plain language, the three truth
  states and the coverage report up front, and guided stories — with citations one click away,
  never removed.
- (c) Agent-first: the agent interface becomes the primary surface, and documents become its
  by-product.

**Recommendation: (b) now, and let the comprehension pilot (D-14) decide between documents and
the agent as the primary surface for non-technical users.** Whether the map should be *read* or
*asked* is an empirical question, and the pilot's "asking a coding agent" condition answers most
of it. Either way the model, the guarantee and the citations are the same; only presentation and
delivery differ, so neither answer wastes work.

**Touches:** §4 (the agent row), §8's deliverables, the web app, the pitch of §1.5.

---

## 7. Risks and honest limits

| Risk | Why it is real | Mitigation |
|---|---|---|
| Static analysis is partial | Dynamic dispatch, events, runtime wiring | Unresolved as a first-class state (D-6); report coverage |
| Legibility is a research problem | It is the problem R1.a deliberately deferred | Framework conventions first (D-2); measure it (D-14) |
| Breadth multiplies work | Lenses × diagram types × frameworks | One stack end to end before widening (D-7) |
| Cost and time grow with the model | More elements, more prose | Interpretation memory (D-1) |
| The field moves fast | DeepWiki, Archify and others ship quickly | Compete on what they structurally cannot do: verified claims, determinism, the change lens, local operation |
| **The central hypothesis is false** | Nobody has yet shown that a verified map helps people understand AI-built software better than asking an agent | Test it first and cheaply — the comprehension pilot (D-14) |
| The neediest readers cannot verify | Citations mean nothing to someone who cannot read code | Provenance presented as a guarantee to them (§3, D-17) |
| People never open the documents | Many ask their agent instead | Let the pilot decide the primary surface (D-17) |
| Default privacy costs quality | Explanations from names and structure alone may be thin | Measure early (§8); the local model as the private path to richer output (D-8) |

---

## 8. Next steps

1. **Refine this document.** Take or amend each decision in §6; record each taken one in
   `decisions.md`.
2. **Evidence before design**, as the original survey did:
   - choose new subjects across the project types of §2.5 — including a couple deliberately
     vibe-coded, since that is the audience;
   - for each, write down the questions a person would actually ask about it *before* designing
     anything. That list decides which lenses matter first.
3. **Test the hypothesis early and cheaply** — the comprehension pilot of D-14, on R1.a's output
   and paper mock-ups of the new lenses, with "asking a coding agent" as one condition. It is
   the step most able to redirect everything after it.
4. **Measure structure-only explanations** (D-8): take one subject, write the explanations a
   model produces from names and structure alone, and judge whether they are good enough to be
   the default.
5. **One feasibility spike:** tree-sitter as WASM inside the Go binary, plus a NestJS pack on
   Immich — targets: the server → machine-learning edge and the server's feature areas.
6. **Then** update `product-definition.md`, the capability catalog and the roadmap, and flag the
   schedule's mirrors.
