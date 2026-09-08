# Progress

Tracked against the capability catalog (`docs/product-definition.md` §4) and the acceptance
criteria (§12). These are the project's own work breakdown — no parallel TODO list.

**Status key:** `·` not started · `~` in progress · `✓` done · `⊘` superseded, see note

Last updated: 2026-09-06 (second checkpoint)

---

## Acceptance criteria — the definition of done

Four of these need no human judgement and no external subjects. They are the ones to run
constantly, not just at a review.

| | Criterion | Threshold | Self-checking | Status |
|---|---|---|---|---|
| AC-1 | Nodes and edges carry extraction or catalog provenance | ≥ 95% | no | ~ **extraction side holds** — every node and edge cites a line. **Catalog side does not**: technology labels come from the image table and carry no provenance at all. See the finding below |
| AC-2 | Container diagram produced with the LLM disabled | renders + validates, both subjects | **yes** | ~ **half met.** Renders on all three subjects, and there is no LLM in the tool at all. *Validates* waits on the validator (sprint 2) |
| AC-3 | Structural accuracy vs hand-drawn reference | ≥ 0.85 on Immich | no — needs the reference | · |
| AC-4 | Validator rejects malformed models | 100% of fault-injection suite | **yes** | · |
| AC-5 | Rule persistence | 10 rules survive regeneration | **yes** | · |
| AC-6 | Drift detection | 100% of synthetic drift set | **yes** | · |
| AC-7 | Determinism | 5 runs, byte-identical FactSets | **yes** | ✓ **holds end to end** — 5 identical runs on all three subjects, measured on the full generated document rather than the FactSet alone. Covered by tests in four packages |
| AC-8 | Egress | `structure-only` transmits zero file contents | **yes** | · |
| AC-9 | Performance | NFR-1 and NFR-2 met on Supabase | no | · |

> **Finding, 6 Sep — catalog facts have no provenance.** A node's `Technology` is a lookup
> from the image table, and nothing records that. AC-1 explicitly admits *catalog* provenance
> alongside extraction, so the criterion is not satisfied by extraction alone. `PRV-02` is the
> capability that fixes it — label each entry with its origin — and it is unstarted.

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
| `VAL` | Validation | 8 | 0 | AC-4 |
| `RUL` | Rules — `rules.yaml` | 6 | 0 | AC-5 |
| `SEM` | Semantic layer — the LLM | 10 | 0 | — |
| `PRV` | Provenance | 6 | ~1 | AC-1 |
| `MEM` | Memory and diff | 8 | 0 | AC-6 |
| `VIE` | Views and rendering | 10 | ~4 | AC-2 |
| `SUR` | Surfaces — CLI and web app | 15 | ~2 | AC-8 |
| `OUT` | Output and deliverables | 10 | ~2 | — |
| `ANS` | Answer surface | 7 | 0 | — *(R1.c, stretch)* |
| | **Total** | **116** | **~30** | |

109 are R1.a; the 7 `ANS` capabilities are R1.c.

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
| Sprint 2 | 25 Sep | AC-2, AC-6, AC-8 — **the thesis, demonstrated** | · |
| Sprint 3 | 9 Oct | All nine evaluated and written up | · |

---

## Open questions

| | Question | Blocks | Status |
|---|---|---|---|
| O-7 | Test subject #3 | — | ✓ **closed 26 Aug — Mastodon**, pinned `47ac677`. Chosen to cover `MDL-09`/`MDL-11`/`MDL-16`/`EXT-09`, which the other two leave untested |
| O-8 | Does `file:line` provenance survive the Compose merge? | — | ✓ **closed 26 Aug — no.** Extraction is two passes; a `Fact` carries one position. See `docs/decisions.md` |
| O-6 | R1.b sequencing — analysis before clustering? | R1.b only | deferred by design, decide with a working spine |
| O-10 | Does R1.a resolve service-to-gateway calls by matching route paths? | fuller `MDL-03` | **open.** Routes are read, but a caller's URL names one endpoint and the bridge cannot tell which. Only actors bridge today, so a service calling a gateway loses that edge. Matching `lds` route prefixes against caller URLs would close it — decide before code freeze |
| O-9 | Which acceptance criteria gate sprint 2? | Sprint 2 gate | ✓ **closed 6 Sep — §4.3 wins: AC-2, AC-4, AC-5.** The disagreement was a symptom, not a judgement call: the calendar had no rows for the validator or for rules, so nothing in it could have passed AC-4 or AC-5, and the gate had been quietly reconciled to the rows. Six missing rows added instead |

---

## Manual work — does not compress

Verification-bound items that AI assistance does not accelerate.

| | Needed for | Status |
|---|---|---|
| Hand-drawn Immich reference architecture | AC-3 — it is the answer key | · not on the calendar, by decision. Owned by Cruz, no date |
