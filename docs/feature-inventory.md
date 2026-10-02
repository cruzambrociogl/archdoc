# archdoc — feature inventory

> **What this document is:** every candidate feature for the extension phase, scored, so the
> schedule can be built from a chosen list instead of a guess. **It is a menu, not a plan.**
>
> **How to use it:** mark each row **in** or **out**. The ones marked in become capability IDs in
> `product-definition.md` §4 and rows in `schedule.csv`. The recommendation column is my opinion,
> not a decision.
>
> Drafted 2026-10-02. To be decided in the planning week, 13–17 Oct. Delivery 11 Dec 2026.
> Direction and positioning: `docs/vision.md`. Fixed obligations (measurement, report, slides) are
> listed in §8 — they are not features and are not optional.

## The decisions that shape the list

| | |
|---|---|
| **Audience** | Technical readers losing control of code an AI wrote. Primary, chosen 2 Oct. Vibe-coder-facing work is deprioritised, not deleted |
| **Posture** | The local, verifiable alternative to hosted AI code wikis (`vision.md` §2.2) |
| **Order** | Evidence before building: the pilot and the competitor comparison run first and may redirect this list |
| **Veto** | A feature whose output cannot be traced to evidence does not ship, however good it looks |

## How to read the columns

- **Audience** — **T** technical (primary) · **V** vibe coder · **A** agents · **P** panel/report
- **Evidence** — *have* (already extracted) · *new parser* (needs tier 2/3 code reading) · *new extractor* (a new file type) · *none* (would need invention → veto)
- **Verifiable** — can every element it shows be traced? **yes** · **cited** (interpretation with citations) · **no** (veto)
- **Parity** — **CW** matches something Google Code Wiki does · **✦** they structurally cannot
- **Cost** — **S** ≤2 days · **M** 3–5 days · **L** 1–2 weeks · **XL** >2 weeks
- **Rec** — **Must** (incoherent without it) · **Should** (parity, high value) · **Evid** (proves the thesis) · **Nice** (first to cut) · **Defer**

---

## A. Foundations — reading code at all

Nothing in B or C exists without these.

| ID | Feature | Aud | Evidence | Verif | Parity | Cost | Rec |
|---|---|---|---|---|---|---|---|
| F-01 | **Syntax parsing in the binary** — tree-sitter as WASM, no cgo, no extra install for users (D-4) | T | new parser | yes | CW | L | **Must** |
| F-02 | **Application discovery** — find the apps in a repo from manifests (`package.json`, `pyproject.toml`, …) at any depth, instead of "find the Compose file" (D-5) | T | new extractor | yes | CW | M | **Must** |
| F-03 | **Module and import graph** — files, modules and what imports what, per language. The generic baseline for *any* repository | T | new parser | yes | CW | M | **Must** |
| F-04 | **Framework pack: one backend stack** — routes, handlers, injected services (NestJS or Express first) | T | new parser | yes | CW | L | **Must** |
| F-05 | **Framework pack: one frontend stack** — pages and routes (React/Vite or SvelteKit) | T | new parser | yes | CW | M | **Should** |
| F-06 | **Framework pack: Python** — FastAPI/Flask routes, and plain scripts | T | new parser | yes | CW | M | **Should** |
| F-07 | **ORM and schema reading** — tables, columns, relations (Prisma/TypeORM/SQLAlchemy, SQL migrations) | T | new parser | yes | CW | M | **Must** |
| F-08 | **Outbound call detection** — HTTP clients, SDK imports, URLs built from env values | T | new parser | yes | CW | M | **Should** |
| F-09 | **Type-aware resolution (optional upgrade)** — better call resolution when a type checker is present, recorded as *resolved by type* vs *by name* (D-4) | T | new parser | yes | — | L | Nice |

## B. Lenses — what the reader sees

| ID | Feature | Aud | Evidence | Verif | Parity | Cost | Rec |
|---|---|---|---|---|---|---|---|
| F-10 | **Component view** — C4 level 3 inside each container, grouped by framework convention first (D-2) | T | F-03/04 | yes | CW | M | **Must** |
| F-11 | **Data model view** — entities, fields, relations, as a diagram and a table | T | F-07 | yes | CW | M | **Must** |
| F-12 | **Flow view** — entry point → handler → service → store/outbound, as a sequence diagram (arc42 §6) | T | F-04/07/08 | yes | CW | L | **Must** |
| F-13 | **Feature list** — what the system does, from its real entry points: routes, pages, commands | T,V | F-04/05 | yes | CW | S | **Should** |
| F-14 | **Dependency map** — third-party packages, grouped, with what uses them | T | F-02/03 | yes | CW | S | **Should** |
| F-15 | **Deployment view** — move Compose where it belongs (arc42 §7), plus CI files | T | have | yes | CW | S | **Should** |
| F-16 | **Data-flow view** — where data enters, is stored, and leaves; sensitivity labels as interpretation | T | F-07/08 | cited | CW | M | Nice |
| F-17 | **Lifecycle view** — entity states from status enums and the lines that assign them | T | F-07 | cited | CW | M | Nice |
| F-18 | **CI/workflow view** — lanes and gates from `.github/workflows` | T | new extractor | yes | CW | M | Nice |

## C. Output — documents and pages

| ID | Feature | Aud | Evidence | Verif | Parity | Cost | Rec |
|---|---|---|---|---|---|---|---|
| F-19 | **A page per component** — the "module walkthrough" shape, generated, every claim cited | T | F-10 | cited | CW | M | **Should** |
| F-20 | **More arc42 sections filled** — §5 level 2, §6 runtime, §8 domain/data, §12 glossary from real vocabulary | T,P | B lenses | yes | — | M | **Must** |
| F-21 | **Output sized to the project** — one page for a script, full arc42 for a system (D-13) | T | have | yes | — | S | **Should** |
| F-22 | **Publishable static site** — the `mkdocs.yml` already planned, so "share it with my team" has an answer that is not a hosted service | T | have | yes | CW | M | **Should** |
| F-23 | **Self-contained HTML export** of a diagram or explanation (D-10, NFR-12) | T | have | yes | CW | S | Nice |
| F-24 | **Coverage report** — "what archdoc could not see", per repository (D-6) | T,P | have | yes | ✦ | S | **Must** |

## D. Surfaces

| ID | Feature | Aud | Evidence | Verif | Parity | Cost | Rec |
|---|---|---|---|---|---|---|---|
| F-25 | **Web app at scale** — search, zoom levels, focus on one element, upstream/downstream tracing, a legend that teaches proven / interpreted / unresolved | T | have | yes | CW | L | **Should** |
| F-26 | **Ask the map** — grounded question answering over the validated model, citations attached, local-first (D-17) | T,V | have | cited | CW | L | Nice |
| F-27 | **Agent interface (MCP)** — the model as queryable tools for the coding agent already in the loop | A | have | yes | ✦ | M | Nice |
| F-28 | **`archdoc init`, `diff`, `export`** — the three CLI commands in the catalog that were never built | T | have | yes | — | S | **Should** |
| F-29 | **Watch mode** — regenerate on save, inside `serve` (D-15) | T | have | yes | CW | S | Nice |

## E. Trust, stability and privacy — the roots

| ID | Feature | Aud | Evidence | Verif | Parity | Cost | Rec |
|---|---|---|---|---|---|---|---|
| F-30 | **Interpretation memory** — store every model-written value against a fingerprint of the facts behind it; re-ask only what changed (D-1). Makes heavy LLM use deterministic, cheap and diff-stable | T | have | yes | ✦ | M | **Must** |
| F-31 | **Identity for code elements** — identity by what a thing *is* (route, table, module, symbol), not by path, so refactors are renames rather than churn (D-3) | T | F-03 | yes | ✦ | M | **Must** |
| F-32 | **Unresolved as a first-class state** — "calls something at `${URL}`", shown, never dropped (D-6) | T | F-08 | yes | ✦ | S | **Must** |
| F-33 | **Rules for the new element kinds** — corrections to components, entities and flows that survive regeneration | T | have | yes | ✦ | M | **Should** |
| F-34 | **Local model path** — Ollama driver, schema-constrained output, run log showing zero egress (NFR-4) | T | have | yes | ✦ | M | **Should** |
| F-35 | **Egress rules for code symbols** — write the symbol boundary down exactly and test it (D-8, AC-8) | T,P | have | yes | ✦ | S | **Must** |
| F-36 | **Validator rules for cited claims** — every sentence's citations must resolve, or the claim is refused (vision §3) | T,P | have | yes | ✦ | M | **Must** |

## F. Change over time

| ID | Feature | Aud | Evidence | Verif | Parity | Cost | Rec |
|---|---|---|---|---|---|---|---|
| F-37 | **Diff between two commits** — the promised AC-6, not just between stored versions | T,P | have | yes | ✦ | M | **Must** |
| F-38 | **Per-commit or per-session change summary** — "this session added X, touched Y, introduced Z" | T | F-37 | cited | CW | M | **Should** |
| F-39 | **Drift set** — the synthetic fixtures AC-6 is scored against | P | have | yes | — | S | **Must** |

## G. Intent versus actual

| ID | Feature | Aud | Evidence | Verif | Parity | Cost | Rec |
|---|---|---|---|---|---|---|---|
| F-40 | **Intent sources** — the user names their plan files; archdoc reads only those (D-9) | T | new extractor | yes | ✦ | S | **Should** |
| F-41 | **Plan → code mapping** — where each stated requirement lives, what is missing, what nobody asked for | T | F-40 + B | cited | ✦ | L | **Should** |

## H. Evidence and measurement — what the thesis rests on

| ID | Feature | Aud | Evidence | Verif | Parity | Cost | Rec |
|---|---|---|---|---|---|---|---|
| F-42 | **Comprehension pilot** — three conditions: code alone, asking a coding agent, archdoc. Includes questions about things the participant was never told exist (D-14) | P | — | — | ✦ | M | **Evid** |
| F-43 | **Competitor comparison on Immich** — archdoc vs Code Wiki vs DeepWiki against the hand-drawn reference: what each missed, what each invented | P | needs the reference | — | ✦ | M | **Evid** |
| F-44 | **Hand-drawn Immich reference** — the answer key for F-43 and AC-3. **Owner: Cruz. Nobody else can do it** | P | — | — | — | M | **Evid** |
| F-45 | **Structure-only quality check** — are explanations good enough when only names and structure leave the machine? (D-8) | P | have | — | ✦ | S | **Evid** |
| F-46 | **Local model quality check** — the same labels from Granite 4.2 8B or Qwen3 14B versus Opus, on the same subjects | P | F-34 | — | ✦ | S | **Evid** |
| F-47 | **Performance measurement** — AC-9: NFR-1 (<30 s scan) and NFR-2 (<3 min generate), measured on Supabase and on a large code repo | P | have | — | — | S | **Evid** |
| F-48 | **New test subjects** — a React app, a CRUD backend, a script; a couple deliberately vibe-coded, since that is what the audience ships | P | — | — | — | M | **Evid** |

## I. Loose ends already on the books

Small, and each one is a real defect or gap. Mostly from `.claude/NOTES.md`.

| ID | Item | Cost | Rec |
|---|---|---|---|
| F-49 | Gateway route paths — resolve a service's call to a gateway by matching route prefixes (O-10) | M | **Should** |
| F-50 | Fingerprint only architecture, so a schema change stops minting versions for every repo | S | **Should** |
| F-51 | Store a recomputed layout when the engine version changes | S | Nice |
| F-52 | Remove generated files archdoc no longer emits (the orphaned `architecture.generated.md`) | S | **Should** |
| F-53 | Report which values came from a sample env file (`example.env`, `.env.production.sample`) | S | Nice |
| F-54 | Click the web app through in a browser on all three subjects | S | **Must** |
| F-55 | Clickable relationships on the canvas (arrows are not groups in the SVG) | S | Nice |
| F-56 | Tune labelling effort — measure `low`/`medium` against default | S | Nice |
| F-57 | Upgrade the cloud model to `claude-opus-5-5` (cheaper per token; effort defaults change) | S | **Should** |

## J. Deferred on purpose

Not rejected — just not this phase. Each has a reason.

| Item | Why not now |
|---|---|
| Vibe-coder presentation: plain language, guided stories, three-state framing for non-readers | Audience chosen is technical (D-17). Revisit after the pilot |
| R1.c write surfaces: chat refinement, editable canvas, answer surface | A whole phase of its own, and a write-safety design we would rush |
| LLM-proposed layout (D-12) | An experiment, not a feature. Cheap to try once the diagrams exist |
| More languages (Go, Java, Ruby, PHP) | The generic tier (F-03) covers them shallowly; depth does not pay yet |
| Kubernetes and Helm | Out of R1 scope by decision, and unchanged by this phase |
| Multi-repo composition | Clean extension later; needs declared-vs-referenced, which already exists |
| R2a conformance and `archdoc check` | Needs an intended model; F-40/41 is the descriptive first step |

## Capacity — what actually fits

Build window: **27 Oct → 3 Dec, about five and a half weeks**, after the planning week and the
evidence sprint, before the freeze on 4 Dec.

That is **roughly 12 to 16 features**, not forty. The Musts in A, B, C and E alone come to about
four weeks, which leaves little room. So the real choice is which **Shoulds** survive.

**My recommended cut, in order of what I would drop first:** F-18, F-17, F-16, F-29, F-27, F-26,
F-09, F-23. Dropping all eight buys back roughly two weeks and loses nothing a technical reader
needs on 11 Dec.

**The one I would fight to keep despite its cost:** F-41 (plan → code). It is the clearest thing
archdoc does that no competitor can, and it speaks directly to "the AI built something; is it what
I asked for?"

## §8 Fixed obligations — not features

| | When |
|---|---|
| Progress review | 9 Oct |
| Planning week — this document decided, D-1 to D-17 settled, catalog and schedule rewritten | 13–17 Oct |
| Evidence sprint — F-42 to F-48 | 20–24 Oct |
| Code freeze | 4 Dec |
| Acceptance measurement — all nine criteria | 4–8 Dec |
| Report and slides, from measured results | 7–10 Dec |
| **Delivery** | **11 Dec** |

## Provisional working set — so there is a path before the planning week

**Status: provisional, chosen by me on 2 Oct, to be confirmed or amended 13–17 Oct.** The calendar
is built on this so that work can start; nothing here is a commitment, and every row stays
negotiable until the planning week closes.

**In — the build (27 Oct → 3 Dec)**

| Block | Features |
|---|---|
| **Build 1 · foundations and stability** · 27 Oct – 13 Nov | F-01 syntax parsing in the binary · F-02 application discovery · F-03 module and import graph · F-04 one backend framework pack · F-07 ORM and schema reading · F-31 identity for code elements · F-32 unresolved as a state · F-30 interpretation memory · F-35 egress rules for symbols |
| **Build 2 · lenses and output** · 10 – 21 Nov | F-10 component view · F-11 data model view · F-12 flow view · F-13 feature list · F-14 dependency map · F-15 deployment view · F-20 more arc42 sections · F-24 coverage report · F-36 validator rules for cited claims · F-33 rules for new element kinds |
| **Build 3 · surfaces and change** · 24 Nov – 3 Dec | F-25 web app at scale · F-22 publishable static site · F-19 component pages · F-21 output sized to the project · F-37 diff between two commits · F-39 drift set · F-38 change summary · F-28 the three missing CLI commands · F-54 browser pass · loose ends F-49, F-50, F-52, F-57 |
| **Wherever they fit** | F-05 frontend pack · F-06 Python pack · F-34 local model path · F-40 intent sources |

**Evidence (20–24 Oct, and again in December):** F-42 comprehension pilot · F-43 competitor
comparison · F-44 the hand-drawn reference *(yours, starts now)* · F-45 structure-only quality ·
F-46 local model quality · F-47 performance · F-48 new test subjects.

**Stretch — only if Build 2 lands early:** F-41 plan → code mapping. It is the feature with no
competitor counterpart, so it is the first thing to promote if there is room, and the mid-phase
checkpoint on 13 Nov is where that is decided.

**Out for this phase:** F-09 type-aware resolution · F-16 data-flow view · F-17 lifecycle view ·
F-18 CI view · F-23 HTML export · F-26 ask the map · F-27 agent interface · F-29 watch mode ·
F-51, F-53, F-55, F-56. None is rejected; each is cheap to revive once the foundations exist.

**The honest arithmetic.** Added up as ideal days, the Musts alone exceed the window. The reason to
try anyway is measured: between 6 Sep and 12 Sep this project built extraction, the model, the
validator, rules, storage, layout, the C4 renderer, arc42 output, the semantic layer and the whole
web app. Velocity with AI assistance is several times what these estimates assume. **But the buffer
is thin, so the cut list above is agreed in advance** — when something slips, we drop from the
bottom of Build 3 rather than renegotiate.

## How to decide

1. Go down each table and mark **in** or **out**. Ignore my recommendations where you disagree.
2. Where two features conflict for the same week, say which one wins.
3. Anything you want that is not listed, add it — the list is certainly incomplete.
4. Then I turn the chosen rows into capability IDs and a calendar, and we stop planning.
