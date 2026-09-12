# Notes

The capture inbox. Anything that does not have a home yet — an idea, a reminder, a question,
something noticed in passing, a thing to try later.

**Add without thinking about where it goes.** That is the whole point: a note that costs
effort to file does not get written down. Routing happens later.

`PROGRESS.md` tracks work that was *planned* — capability IDs and acceptance gates. This file
holds everything that was not.

---

## How it drains

`/checkpoint` sweeps this file and gives every open item one of four outcomes. Nothing is
meant to sit here indefinitely.

| Marker | Meaning |
|---|---|
| `[ ]` | Open — still an inbox item |
| `[→ path]` | **Promoted.** Its real home now holds it; the line stays one cycle, then goes |
| `[x]` | Done, and nothing further was needed |
| `[-]` | Dropped — keep the reason on the line |

Where things get promoted to:

| A note about… | Goes to |
|---|---|
| Why we chose X over Y | `docs/decisions.md` |
| A capability finished, a gate passed | `PROGRESS.md` |
| A rule that must hold in every session | `CLAUDE.md`, Hard rules |
| Something true about a test subject | `docs/survey-test-subjects.md` |
| Scope or timing | `docs/delivery-schedule.md` |
| What the product *is* | `docs/product-definition.md` — rare, flag it |

**Precedence still applies:** if a note contradicts `docs/product-definition.md`, it is
raised, not quietly promoted over it.

---

## Open

- [ ] `web/dist/index.html` is a tracked placeholder; confirm the real build overwrites it
      cleanly rather than colliding with the `.gitignore` exception
- [ ] OUT-09 completeness — the index cannot say whether a human section is still the stub
      archdoc wrote, because OUT-03 forbids reading it. Needs content hashes in `.archdoc/`,
      which is also what OUT-10 staleness needs
- [ ] An `architecture.generated.md` from before the arc42 layout is orphaned in any repository
      generated with an older build. archdoc writes its own files and never removes ones it no
      longer emits — decide whether that is a defect
- [ ] The version fingerprint is a hash of the model's JSON, so adding a field to the model
      (as 11–12 Sep did) records a new version for every repository on its next run, though no
      architecture changed. Either fingerprint architecture only, or accept it and say so
- [ ] A layout recomputed for an unchanged architecture (engine version changed) is not stored,
      because Save returns the existing version untouched. Harmless — Supabase lays out in
      milliseconds — but the stored layout stays stale until the architecture next changes
- [ ] PRV-05 — the evidence tables now mark every model-written name, label, description and
      technology with its own citation. The Mermaid diagram itself does not yet distinguish them
- [ ] Labelling takes ~2 minutes on Supabase with adaptive thinking at default effort. Worth
      measuring `effort: low`/`medium` — labelling is the kind of work that often holds quality there
- [ ] Gateway route *paths* are not read. `lds.template.yaml` holds prefix-to-cluster rules;
      without them a service calling a gateway cannot be resolved to what it actually calls.
      Tracked as O-10 in `PROGRESS.md`
- [ ] Immich's `example.env` and Mastodon's `.env.production.sample` are read for interpolation
      but never reported. A reader cannot tell which values were filled from a sample file —
      worth surfacing in the generated document

---

## Recently resolved

Staging, not an archive. An item lands here when it is resolved, and is deleted at the
**next** checkpoint after that. Nothing stays under *Open* once it has a marker.

- [→ PRV-02] Catalog facts had no provenance. `Provenance.Origin` now records extraction /
      catalog / rules / model, and a technology cites the catalog entry that supplied it.
      AC-1 measures 100% on all three subjects — 2026-09-08
