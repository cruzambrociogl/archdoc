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

- [ ] Decide the first commit's contents — scaffolding only, or scaffolding plus `go.mod`?
- [ ] `web/dist/index.html` is a tracked placeholder; confirm the real build overwrites it
      cleanly rather than colliding with the `.gitignore` exception
- [ ] Third test subject (O-7) — look for a repo with real declared infrastructure that is
      neither a photo app nor a BaaS. Something in a different shape would test discovery harder
- [ ] Worth checking whether `compose-go` exposes profile resolution, since Immich's dev
      stack leans on `profiles:` and `!reset`

---

## Recently resolved

*(cleared each checkpoint after one cycle — promoted items do not accumulate here)*

- [→ docs/decisions.md] Packages follow seams, not catalog chapters — 2026-08-24
- [→ docs/stack-decision.md §3] Build needs Node for contributors, not for users — 2026-08-24
- [x] Move the ideation notes somewhere durable — now `docs/origin/` — 2026-08-25
