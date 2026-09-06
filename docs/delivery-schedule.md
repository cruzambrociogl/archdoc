# archdoc — R1 delivery schedule

> Planned 2026-08-24, second revision. **Delivery at the review of Friday 9 October 2026.**
>
> A working plan. Building will change it; the review dates will not. Depends on
> `product-definition.md` (what), `stack-decision.md` (with what) and
> `survey-test-subjects.md` (the evidence).

> ### ⚠ This document has published mirrors
>
> **This file is the source of truth. When it changes, the mirrors must be updated or they
> become confidently wrong in front of a supervisor.**
>
> | Mirror | Purpose | Update by |
> |---|---|---|
> | [`schedule.csv`](schedule.csv) | **The calendar's source of truth.** Version-controlled, ISO dates | Edit directly, commit |
> | [Google Sheet](https://docs.google.com/spreadsheets/d/1Oww6pPHneW_KQmUZ-D7qXBrFHuELsR00YziVmCSmmYg/edit) | The submitted deliverable — a mirror of the CSV | Paste the CSV in. Never replace the file: the connector cannot rewrite contents, so regenerating mints a new URL |
> | [Artifact page](https://claude.ai/code/artifact/1bff2613-a2a6-4e3e-8d30-1ae30e4ff4a8) | Presentation view — Gantt, sprint cards | Republish the same file path to keep the URL |
>
> ### Keeping the calendar in sync
>
> `docs/schedule.csv` is authoritative for **dates and rows**; this document is authoritative
> for **scope and reasoning**. If they disagree, one of them is wrong — say so rather than
> picking silently.
>
> **Local → Sheet.** Open the CSV, copy, paste over the Sheet's range. Sheets re-parses the
> ISO dates into real dates on paste.
>
> **Sheet → local.** After editing the Sheet by hand, ask Claude to read it and update the CSV.
> Sheets renders dates in locale form (`8/7/2026`); they are converted back to ISO on the way
> in, so the committed file stays sortable and diffable.
>
> **One direction at a time.** There is no merge — whichever side is written last wins. Decide
> where a change originates before making it.

> Two now-redundant Sheets remain in Drive and should be trashed:
> `1gy3L8PcC9UzzVEBSlkenmJ8MzgBrLhmJgvcw2Ma9H8A` and
> `1mJX3I9gDrN-C0WUzNdTx5MtSuKuyF5M4HD5D7tc4PHE`.

---

## 1. Where we are

The calendar shows four quiet weeks. They produced the three documents this plan rests on.

| Period | Phase | Output |
|---|---|---|
| Jul 31 – Aug 7 | **Idea definition** | Problem, governing principles, release structure |
| Aug 7 – Aug 14 | **Specification** | 109 capabilities classified, output contract, nine acceptance criteria |
| Aug 14 – Aug 21 | *Midterm preparation* | — |
| Aug 21 – Aug 24 | *Midterm examinations* | — |
| Aug 24 | **R&D and planning** | Two production repositories surveyed; stack decided on that evidence |

**Nothing is behind.** Idea definition, specification and research happened in the correct
order, and the research was empirical rather than assumed. Construction has not started, and
that is the right state on 24 August with two weeks lost to exams.

---

## 2. Scope

**The binding constraint is not writing code.** Implementation is done with Claude Code
against a specification that is already unusually complete — 109 capabilities classified by
dependency, nine objective acceptance criteria, a defined data model and pipeline, and 87% of
R1.a deterministic. That combination (tight spec, objective tests, no hidden state) is close
to the ideal case for AI-assisted building. The human work is deciding, directing, reviewing
and verifying.

So the schedule is set by what each piece of work is actually *limited* by:

| Constraint | Work | Effect |
|---|---|---|
| **Implementation-bound** | Parsers, model, validator, storage, renderers, CLI, **the web app**, R1.c's write surfaces | Compressed heavily. Schedule aggressively |
| **Verification-bound** | Acceptance measurement, testing against real repositories, the hand-drawn reference | Largely fixed cost. Must be scheduled explicitly |
| **Research-bound** | R1.b's semantic clustering | **Not compressed at all** |

### What is committed, and what is not

| Tier | Contents | Commitment |
|---|---|---|
| **R1.a — complete** | Deterministic spine, semantic layer, architectural diff, CLI **and the local web app** | **Committed for 9 Oct** |
| **R1.c — interaction** | The three write surfaces, sharing one write-safety design — content hashing, conflict refusal, never-during-generation | **Stretch.** Implementation-bound, so genuinely reachable if sprints 1 and 2 hold |
| **R1.b — static analysis** | Parsing application source, building the intra-service graph | **Stretch.** Also implementation-bound |
| **R1.b — semantic clustering** | Turning many files inside a service into a few meaningful boxes | **Not committed, and not schedulable.** §2 of the definition calls it "where the genuine research problem lives"; O-6 calls it "the one genuinely unsolved problem". It gets a prototype against fixtures, not a delivery date |

**This is close to all of R1.** What is explicitly excluded is one research problem — and
excluding it is the definition's own position, not caution about velocity. O-6 is also
explicit that R1.b's sequencing should be decided "with a working R1.a spine in front of you,
not before", which is another reason it is a spike in sprint 3 rather than a plan today.

**Definition of done for 9 October:** R1.a complete, AC-1 through AC-9 evaluated against
Immich and Supabase, results written up honestly including any criterion not met.

---

## 3. Assumptions

1. **Three graded checkpoints: 11 Sep · 25 Sep · 9 Oct.** All three are shown to the adviser.
   The first is where the project has to stop being a terminal command and become something you
   can look at.
2. **Two people, one calendar.** Two tracks meeting at the FactSet; the schedule stays single so
   neither drifts. See §4.1.
3. **Implementation is not the constraint.** Claude Code writes it; the human work is deciding,
   directing, reviewing and verifying. Estimates in this plan are set by decisions and review,
   not by typing.
4. **The report and the presentation are graded deliverables**, not by-products, and have their
   own workstream. Entry point: `brief.md`.
5. **9 October may not be the end.** The adviser has signalled a possible extension to late
   November if progress is judged good. So 9 October optimises for **demonstrated capability**
   rather than completeness, and §7 carries what would fill the extension.

---

## 4. The plan

Three two-week sprints, each ending at a **graded** checkpoint. Each review makes a stronger
claim than the last, and the first has to be *visible*.

```
 Aug 31────Sep 11        Sep 14────Sep 25        Sep 28─────Oct 9
   sprint 1                sprint 2                sprint 3
        ▲REVIEW 1               ▲REVIEW 2              ▲REVIEW 3 + DELIVERY
   "it draws"              "it documents —         "measured, and
                            without an LLM"         presented"
```

### 4.1 Two tracks, one calendar

The split follows the package seams and meets at the **FactSet** — which §11 already defines as
the contract between the deterministic and probabilistic halves. Using it as the contract
between two *people* costs nothing extra, and lets the drawing side build against a fixture
before real data exists.

```
   TRACK A — FACTS                    TRACK B — OUTPUT + DELIVERABLES
   internal/extract                   internal/render
   internal/archdoc                   arc42 emitter · web app
   edges · kinds · identity           report · slides · branding
                    ↘   FactSet   ↙
                     agreed first, mocked immediately
```

**Rules that stop this becoming two projects.** Neither track edits the other's packages.
Changes to `internal/archdoc` — the shared types — are agreed before being made, since they
break both sides. Both work on `develop`; merge often enough that conflicts stay small.

Track ownership is not recorded per row. The calendar stays single; who takes a row is decided
weekly.

### 4.2 Sprint 1 · Aug 31 – Sep 11 · **It draws**

**The goal is a picture**, committed into a real repository, where every box cites the line that
declares it.

*First, together:* settle `Node`, `Edge` and `kind` in `internal/archdoc`, and write one
hand-made FactSet fixture. Both tracks are then unblocked and neither waits.

*Track A — facts*
- Edges from `depends_on` **and** from environment values carrying URLs
- Node kinds, applying the derivation rule: applications and data stores become containers;
  proxies, gateways and poolers stay in the model but leave the container view
- Actors from published ports — declared evidence that something outside reaches in
- Evidence kinds: declared versus referenced, which *is* the C4 system boundary

*Track B — output*
- **Mermaid emitter** — the cheapest path to a visible diagram. Text, no layout engine, and
  GitHub renders it natively inside markdown
- Generated markdown: the diagram plus a provenance table
- **Both views** — container *and* context. Context is a projection over the same model, so it
  costs little once `evidence` is carried
- Committed into all three subjects as worked examples

*Track B also starts the deliverables:* report outline and branding, from `brief.md`.

> **Gate:** `archdoc generate` produces a container **and** a context diagram for all three
> subjects, rendering on GitHub, every element traceable to a file and line.

**Review 1, Fri 11 Sep — the claim:** *"It reads real production repositories and draws them,
and every element points at the line that proves it."*

*Deliberately not this sprint:* the LLM, SVG, layout, storage, the validator, the web app.

### 4.3 Sprint 2 · Sep 14 – 25 · **It documents, and does so without an LLM**

*Track A — trust*
- Validator, every rule traceable to a failure seen in the draw.io experiment
- SQLite storage and versioning; `model.json`
- Rules — `rules.yaml` corrections that survive regeneration

*Track B — polish and meaning*
- Layout via `go-graphviz`, positions persisted per version; the C4 SVG renderer
- Full arc42: twelve sections, the generated/human ownership boundary enforced
- **Semantic layer** via the Anthropic API — names, responsibilities, groupings, prose
- Report: first draft sections. Branding applied

> **Gates:** **AC-2** — a container diagram renders and validates with the LLM **disabled**.
> AC-4 — validator rejects 100% of a fault-injection suite. AC-5 — ten rules survive
> regeneration.

**Review 2, Fri 25 Sep — the claim:** *"Here is the documentation with the language model on,
and here is the same diagram with it switched off."*

That side-by-side **is** the thesis. It is the single most important demonstration in the
project, and the honest answer to *"you used AI to build this."*

### 4.4 Sprint 3 · Sep 28 – Oct 9 · **Measured, and presented**

*Track A — evidence*
- Architectural diff between two commits
- Acceptance measurement: nine criteria, three subjects
- **Code freeze Friday 2 October**

*Track B — deliverables*
- Report finalised: results, limitations, findings
- Presentation slides, built from the report rather than written fresh

> **Gate:** nine criteria evaluated and written up. A criterion that failed is a finding, not a
> hidden defect.

**Review 3, Fri 9 Oct — the claim:** *"Measured against criteria written before the code
existed — including what it cannot do, and why."*

### 4.5 Mid-sprint self-checks

On the off Fridays — **18 Sep, 2 Oct** — ten minutes against that sprint's gate. Not a meeting.
AC-2, AC-4, AC-6 and AC-7 need no human judgement, so they answer *"is this actually working?"*
at any moment.

---

## 5. What slips first

Decided now, while it is cheap.

| Order | What gives | Why acceptable |
|---|---|---|
| 1 | The stretch tier — R1.c, static analysis, the clustering prototype | Never committed; upside, not scope |
| 2 | Web app views beyond canvas and inspector | §8: *"Markdown is the deliverable. The app is the workbench."* The deliverable survives intact |
| 3 | Caddyfile and nginx parsers | Envoy or Kong alone covers Supabase's gateway |
| 4 | draw.io and PlantUML export | SVG and Mermaid satisfy the output contract |
| 5 | Deployment view, test subject #3 | §2 already calls the first a stretch; both acceptance-critical subjects are in hand |

**What must not slip, because the argument dies without it:** provenance on every fact (P1),
the validator (AC-4), determinism (AC-7), and a container diagram that renders with the LLM
switched off (AC-2). Those four are the thesis. The rest is product.

---

## 6. What changes because AI writes the code

Implementation is not the constraint, which is why §2 tiers by what each piece is *limited*
by. But the time freed from typing has to go somewhere specific rather than being assumed
away — and where it goes is review and verification.

| Activity | Share | Why |
|---|---|---|
| Deciding and specifying | High | The bottleneck is knowing what should exist |
| **Reviewing generated code** | **High — do not compress** | See below |
| Verifying against real repositories | High | Only Immich and Supabase can say whether extraction is right |
| Typing implementation | Low | The genuinely accelerated part |

**Two real risks.**

1. **Unreviewed code accumulates faster than understanding.** For a project you must defend,
   code you cannot explain is worse than code you do not have. Mitigation is structural:
   every sprint ends at an objective gate, and four of the acceptance suites need no human
   judgement — they are the check that survives not having read every line.
2. **Go is new.** Reading generated Go is the fastest way to learn it, but it must be
   *reading*, not skimming. §6 of the stack definition lists the small slice of the language
   that actually appears here.

**The one Go-specific trap to grep for deliberately:** iterating a map without sorting its
keys first. It compiles, it passes casual testing, and it silently breaks AC-7.

---

## 7. If the extension happens — the November plan

Written now and presented as **trajectory**, not as a request. Having it ready is part of what
earns the extension.

Roughly seven additional weeks would go, in priority order:

| | Work | Why this order |
|---|---|---|
| 1 | **Build-manifest extraction** — `pom.xml`, `Cargo.toml`, `go.work`, workspaces | The largest reach for the least risk. Declared structure, language-agnostic, no clustering. Opens every Java, Rust and Node project that has no Compose file |
| 2 | **R1.c — the three write surfaces** | One shared safety design; implementation-bound |
| 3 | **Orchestrator manifests** — Kubernetes, Helm | Currently out of scope for want of a test subject. Would also make a real deployment diagram worth drawing |
| 4 | **R1.b static analysis** | Per-language cost, so it starts with one ecosystem |
| 5 | **Clustering prototype** | Research. Prototype against fixtures; no delivery date, ever |

Item 1 is the strongest candidate: it is the only one that widens who can *use* the tool rather
than deepening what it does for those it already serves.

---

## 8. Risks

| Risk | Likelihood | Response |
|---|---|---|
| **O-8 forces a two-pass extractor** | Medium | Days, not weeks. Week 0 exists to find out before anything is built on it |
| **Reviewing generated Go slows sprint 1** | Medium | Affects comprehension speed, not build speed. The stretch tier absorbs it |
| **Graphviz handles C4 boundaries poorly** | Medium | Layout only emits coordinates, so the engine is replaceable; fallbacks named in the stack definition |
| **Integration surprises across ten stages** | Medium | The likeliest real source of delay — individually fast pieces that disagree at their seams. Mid-sprint self-checks exist to surface this early |
| **AC-3 scores below 0.85 on Immich** | **Expected** | See below |
| **Review preparation eats build time** | High | Now scheduled — Thursday before each review |

> **AC-3 will probably miss its threshold, and that is the honest result.** The survey found
> Immich's three real service connections live in TypeScript source, not configuration
> (`config.dto.ts:624`, `config.repository.ts:203` and `:237`). A configuration-driven
> extractor will miss them. **Predicting this now and reporting it as a measured limitation
> of declaration-based extraction is a stronger result than a tuned score** — and it is
> precisely the kind of finding R1.b exists to address.

---

## 9. This Friday, 11 September

**What to show.**

1. **`archdoc generate` on all three subjects** — container and context diagrams, generated,
   committed, rendering on GitHub. Click from a box to the line that declares it.
2. **Mastodon as the lead example.** It is the only subject with real external systems, so it is
   the only one whose context diagram says anything.
3. **`--explain`** — ten Compose files in Immich, four different systems, two that a filename
   glob misses. Discovery is a decision, and the tool shows its reasoning.
4. **The report outline and first branding**, from `brief.md`.
5. **The C4 mapping decisions** — why Supabase renders nine or ten containers rather than
   eleven, and why the gateway is excluded.

**What not to over-claim.** No validator, no stored versions, no diff, no language model. The
diagram is produced from configuration alone — which is the point, not a shortfall.

**The line worth landing:** *"Every box on this diagram cites the file and line that proves it
exists. Nothing here was inferred, and no model was involved in producing it."*
