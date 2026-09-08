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
- [ ] `docs/architecture/` output has no provenance for catalog facts — a node's technology
      comes from the image table and nothing records that. Blocks AC-1; `PRV-02` is the fix
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

*(cleared 2026-09-06 — the previous cycle's entry was routed to `docs/decisions.md`)*
