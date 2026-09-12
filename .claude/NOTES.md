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

- [ ] An `architecture.generated.md` from before the arc42 layout is orphaned in any repository
      generated with an older build. archdoc writes its own files and never removes ones it no
      longer emits — decide whether that is a defect
- [ ] The version fingerprint is a hash of the model's JSON, so adding a field to the model
      (as 11–12 Sep did) records a new version for every repository on its next run, though no
      architecture changed. Either fingerprint architecture only, or accept it and say so
- [ ] A layout recomputed for an unchanged architecture (engine version changed) is not stored,
      because Save returns the existing version untouched. Harmless — Supabase lays out in
      milliseconds, and `serve` recomputes a stale layout on the fly — but the stored one stays stale
- [ ] Labelling takes ~2 minutes on Supabase with adaptive thinking at default effort. Worth
      measuring `effort: low`/`medium` — labelling is the kind of work that often holds quality there
- [ ] Immich's `example.env` and Mastodon's `.env.production.sample` are read for interpolation
      but never reported. A reader cannot tell which values were filled from a sample file —
      worth surfacing in the generated document
- [ ] Web app checked with HTTP requests and unit tests only — click through it in a browser on
      all three subjects before Review 2
- [ ] Relationships are not clickable on the canvas (the SVG groups boxes, not arrows); they are
      inspected from either end in the inspector. Fine for the review; revisit if it confuses

---

## Recently resolved

Staging, not an archive. An item lands here when it is resolved, and is deleted at the
**next** checkpoint after that. Nothing stays under *Open* once it has a marker.

- [x] `web/dist/index.html` placeholder — it did collide: every local build showed it modified.
      Untracked; `web/dist/.gitkeep` is tracked instead so a fresh clone still compiles — 2026-09-12
- [→ docs/decisions.md, PROGRESS.md OUT] OUT-09 completeness — done from size and mtime recorded
      at stub creation, **not** content hashes: hashing a file is reading it — 2026-09-12
- [x] PRV-05 — the C4 SVG, which is the diagram now shown, italicises every model-written value.
      Mermaid is the fallback and stays plain — 2026-09-12
- [→ PROGRESS.md O-10] Gateway route paths are not read — tracked there, not here — 2026-09-12
