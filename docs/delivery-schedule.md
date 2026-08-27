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
> | [Google Sheet](https://docs.google.com/spreadsheets/d/1Oww6pPHneW_KQmUZ-D7qXBrFHuELsR00YziVmCSmmYg/edit) | The submitted schedule deliverable | Regenerate the CSV and create a new Sheet — the Drive connector can set a file's title and location but **cannot rewrite its contents** |
> | [Artifact page](https://claude.ai/code/artifact/1bff2613-a2a6-4e3e-8d30-1ae30e4ff4a8) | Presentation view — Gantt, sprint cards | Republish the same file path to keep the URL |
>
> A superseded first-revision Sheet also exists in Drive
> (`1gy3L8PcC9UzzVEBSlkenmJ8MzgBrLhmJgvcw2Ma9H8A`) and should be trashed.

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

Stated so they can be corrected rather than silently relied on.

1. **Reviews are fortnightly Fridays:** Aug 28 · Sep 11 · Sep 25 · Oct 9.
2. **Claude Code writes the implementation.** The human work is deciding, directing,
   reviewing and verifying — see §6.
3. **Roughly two to three working sessions a week.** If the real budget differs materially,
   the tier boundaries in §2 move before the dates do.
4. **Go is being learned while building — but not written by hand.** This affects how fast
   generated code can be *reviewed and understood*, not how fast it is produced. Week 0 is
   light for that reason, not because implementation is slow.
5. ~~Test subject #3 may stay unresolved.~~ **Closed 26 Aug — Mastodon**, chosen because
   neither other subject declares a `networks:` block or references a genuine external
   managed service (O-7, `decisions.md`).

---

## 4. The plan

Four days, then three two-week sprints. **Each review makes a stronger claim than the last** —
that arc is the point of the sequencing, not just a way to divide the work.

```
 Aug 24─28         Aug 31────Sep 11        Sep 14────Sep 25        Sep 28─────Oct 9
  foundations        sprint 1                sprint 2                sprint 3
       ▲REVIEW 1          ▲REVIEW 2               ▲REVIEW 3              ▲DELIVERY
  "we know what      "it reads real          "it produces            "it is measured
   we're building"    repositories"           documentation"          against all nine"
```

### Week 0 · Aug 24 – 28 · Foundations

| Day | Work |
|---|---|
| **Mon 24** | Agree this schedule. Install Go, `go mod init`. An hour or two on the language — enough to read what gets written |
| **Tue 25** | **O-8 spike** — does file-and-line provenance survive the Compose merge? Load `docker-compose.kong.yml` and `container-compose-overrides.yml` through `compose-go` and inspect. Decides whether extraction is one pass or two |
| **Wed 26** | Repository skeleton: package layout, CLI entry point, tests in CI. **O-7** decided or deferred |
| **Thu 27** | *Stretch:* walking skeleton — `archdoc scan` printing Immich's services with provenance. Review preparation |
| **Fri 28** | **Review 1** |

> **Gate:** O-8 answered with evidence; `go test ./...` green in CI.

The walking skeleton is a stretch rather than a commitment — not because it is much work, but
because Friday is four days out and a broken demo is worse than none. If it lands, show it.

> ### ✅ Week 0 outcome — recorded 27 Aug
>
> **Gate met, and the stretch item landed.** Both open questions closed on evidence: O-8 with a
> negative result that reshaped the extractor, O-7 against a measured criterion that ruled out
> three candidates. Go 1.27, module, CLI, CI including an import check that enforces the
> network boundary, and `archdoc scan` working on both subjects — Immich 4 services from 10
> candidates, Supabase 11 from 15. Five consecutive scans byte-identical.
>
> **How far ahead this puts us: less than it looks.** The skeleton is a thin slice — roughly
> `DSC` 2 of 4, `EXT` 3 of 15, `PRV` 1 of 6. It extracts a name, an image and a line, and
> nothing else: no ports, volumes, networks, `depends_on`, or environment-derived references.
> Call it two days into Sprint 1's Week A, plus a stretch item banked. Not a week.

### Sprint 1 · Aug 31 – Sep 11 · It reads real repositories

The half of the product that cannot be faked, end to end.

*Week A — extraction*
- Discovery (`DSC`) — the multi-file problem the survey found: eight compose files in Immich,
  fifteen in Supabase, `COMPOSE_FILE` in `.env` as the resolver
- Compose and `.env` extraction via `compose-go`, carrying file and line
- Identity registry (`MDL`) — service name vs `container_name` vs alias, which Supabase's
  `realtime` proves is not optional
- FactSet emitted as JSON, snapshot-tested

*Week B — model and memory*
- Model construction, boundaries, external systems by evidence kind
- Validator (`VAL`), every rule traceable to a failure seen in the draw.io experiment
- SQLite storage and versioning; `model.json`
- Rules (`RUL`) — load-bearing, since O-4 made them the primary mechanism for contract attachment
- Architectural diff (`MEM`) between two commits
- Thu 10: review preparation

> **Gates:** AC-7 — five consecutive scans of Supabase produce byte-identical FactSets.
> AC-4 — validator rejects 100% of a fault-injection suite. AC-5 — ten rules survive
> regeneration from scratch.

**Review 2, Fri Sep 11 — the claim:** *"It reads real production repositories, every fact
points at the line that proves it, and it can tell you what changed between two commits."*

*In parallel, and not code:* **hand-draw the Immich reference architecture** — AC-3's answer
key. Verification-bound work, so it does not compress. Scheduled now because nothing blocks
it and it is otherwise remembered in October.

### Sprint 2 · Sep 14 – 25 · It produces documentation

*Week A — rendering and output*
- Layout via `go-graphviz`, positions persisted per version
- The C4 SVG renderer — ours, drawing from stored coordinates — plus Mermaid export
- arc42 output: twelve sections, fact-backed ones filled, the rest stubbed with contextual
  questions; the generated/human file-ownership boundary enforced

*Week B — meaning and the workbench*
- Semantic layer (`SEM`) via `anthropic-sdk-go` with strict tool schemas — names,
  descriptions, groupings, arc42 prose
- Egress reporting and `structure-only` mode
- **The web app** — canvas, inspector, version timeline, rules viewer, docs preview,
  completeness view. Implementation-bound, and thin by design: it renders persisted
  coordinates and holds no logic
- Thu 24: review preparation

> **Gates:** AC-2 — a container diagram renders and validates **with the LLM disabled**.
> AC-6 — 100% of a synthetic drift set detected and classified. AC-8 — `structure-only`
> transmits zero file contents.

**Review 3, Fri Sep 25 — the claim:** *"It produces real architecture documentation, and it
does so without a language model."* **This is the review that demonstrates the thesis** — AC-2
is the central claim made visible.

### Sprint 3 · Sep 28 – Oct 9 · It is measured, and it goes further

*Week A — to code freeze, Fri 2 Oct*
- Remaining gateway parsers; anything carried from sprint 2
- **Stretch, in this order:** R1.c's three write surfaces (one shared safety design — content
  hashing, conflict refusal, never-during-generation) → R1.b static analysis → a **clustering
  prototype against hand-built fixtures**, which O-6 argues for doing early precisely because
  it is unsolved
- **Code freeze Friday 2 October**

*Week B — measurement*
- AC-1 provenance coverage · AC-3 against the hand-drawn reference · AC-9 performance
- The R1.a report: what was built, what each criterion scored, what was missed and why

> **Gate:** all nine criteria evaluated and written up. A criterion that failed is a finding,
> not a hidden defect.

**Delivery, Fri Oct 9 — the claim:** *"Here is the result, measured against criteria written
before the code existed."*

### Where a lead goes — and where it does not

Being ahead does not move the delivery date. **The reviews are fixed** — 11 Sep, 25 Sep,
9 Oct — so time gained early cannot be spent by finishing sooner. It can only be spent on
scope or on risk, and deciding which in advance is what stops it being absorbed invisibly.

Pulling dates forward is explicitly **not** the answer: it manufactures slack that then
disappears into whatever the current task happens to be.

| Priority | Where it goes | Why |
|---|---|---|
| 1 | **Raise the current sprint's ambition** rather than end it early | Sprint 1's gate is AC-7, AC-4 and AC-5; AC-7 already holds on the walking skeleton. Edges (`MDL`) were Week B work — with the foundation in place, a first rendered diagram inside Sprint 1 becomes plausible |
| 2 | **The web app's canvas and inspector** | Currently in the deferred tier. The two views that carry the click-to-source story |
| 3 | **A clustering prototype against fixtures** | The only item that de-risks a phase not yet scheduled. O-6 argues for trying it early precisely because it is unsolved |
| 4 | Remaining gateway parsers, exports, the deployment view | In the order §5 gives up |

*Test subject #3 was on this list and is now closed — Mastodon, O-7.*

### Mid-sprint self-checks

Fortnightly reviews mean twice the room to drift before anyone notices. On the off Fridays —
**Sep 4, Sep 18, Oct 2** — ten minutes against that sprint's gate. Not a meeting. AC-2, AC-4,
AC-6 and AC-7 need no human judgement and no external subjects, so they answer *"is this
actually working?"* at any moment without waiting for a review.

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

## 7. Risks

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

## 8. This Friday, 28 August

**What to present.**

1. **The three documents** — definition, survey, stack decision.
2. **The research result.** Two production repositories read and measured. The findings that
   a paper design would have missed: Immich declares none of its service connections in
   configuration; Compose files need a specification-grade parser (`!override`, seven
   variable forms, nesting); a naive file glob misses real config files — demonstrated by
   the glob written for the survey missing two.
3. **The stack decision**, with its rejected alternatives and the reason: a mishandled
   Compose merge produces a *silently wrong* model, so the reference implementation belongs
   on the fact path.
4. **This schedule**, including the tiering in §2 and the AC-3 prediction in §7.
5. **O-8's answer** — a research finding, which fits the phase: provenance does *not* survive
   the Compose merge, so extraction is two passes.
6. **A live demo.** `archdoc scan` on both subjects, and `--explain` showing discovery reject
   the hwaccel fragments while finding the two `.devcontainer` files a filename glob misses —
   including the glob written for the survey itself.

**What not to over-claim.** The scan reports a name, an image and a line. There are no edges,
no model and no diagram. Say that plainly — it is four boxes and their sources, and it is the
part everything else rests on.

**The framing.** Four weeks produced a specification, an empirical survey and a technology
decision; two weeks were lost to examinations; construction starts now against a schedule
with named gates. That is a better account than four weeks of code without a plan — and it
is what actually happened.
