---
name: archdoc
description: Understand a repository's architecture before changing it, and say what a change did to it, from archdoc's documentation — services, components, routes, tables and the flows through them, every fact cited to a file and line, read from the code rather than guessed. Use when the repository has a .archdoc/ folder or the user mentions archdoc; before changing a part you did not write ("what does this touch?", "who calls this?", "what writes this table?"); and after a change that adds, removes or rewires routes, services, tables or dependencies.
---

# archdoc

archdoc reads a repository's code and configuration and records its architecture in
`.archdoc/model.json`: every service, component, table and route, the flows from each route through
the classes it calls to the tables it touches, and the line of code that proves each one. It is parsed,
not written by a model: where a sentence was written by one, it is marked ◇.

## How to ask

If the archdoc MCP tools are connected, use them. If not, the CLI prints the same answers; run it
from the repository's root.

| Question | MCP tool | CLI |
|---|---|---|
| What is this system? | `overview` | `archdoc status .` and `archdoc show . <service>` |
| What is X, and where is it proven? | `show` — a name, id, class, `"METHOD /path"` or file | `archdoc show . "POST /api/assets"` |
| Every route, table, component, job, outside service | `list`, with `filter` | `archdoc show . <part of a name>` lists what matches |
| What does changing X touch? | `impact` — a service, component, table or class | `archdoc show . <X>` — its "used by" |
| What did my change do to the architecture? | `changes` with `since: "HEAD"` | `archdoc diff . HEAD` |
| Are the docs current? | `status` | `archdoc status .` |

A name that matches several things comes back as a list of ids: ask again with one of them.

## Before changing code

1. Ask `show` or `impact` about the part you are about to change, and read what uses it.
2. Open the cited files to confirm before you edit. archdoc says where to look; the code is the truth.
3. `impact` is built from the flows archdoc followed, and is complete only as far as they go. "Affects
   no flow" does not mean nothing uses it: check `show`'s "used by", and search the code.

## After changing code

1. Ask `changes` with `since: "HEAD"` — or run `archdoc diff . HEAD` — for what your uncommitted
   change did to the architecture. Nothing needs generating first.
2. Tell the user in a line or two, in archdoc's words: "adds POST /api/albums/share in api; api now
   uses redis". Say nothing when it reports no change to the architecture.
3. If the repository keeps its documentation current, run `archdoc generate .` so it records the
   change. It is offline and free, and writes only `docs/architecture/*.generated.*` and `.archdoc/`.

## Rules

- **Cite what you rely on**: archdoc's `path:line`, so the user can check it.
- **The documentation is as of the last `archdoc generate`.** If `status` says the code has changed
  since, treat answers as possibly stale; `changes` with `since` reads the code as it is now.
- **Never run `archdoc label` or `archdoc explain`** unless the user asks: they send to Anthropic and
  cost money. Everything else is offline and free.
- **Don't edit what archdoc writes** — `*.generated.md`, `.archdoc/model.json`. Regenerate instead. A
  correction to what archdoc found goes in `.archdoc/rules.yaml` (`archdoc init .` creates it).
- **Not documented yet?** `.archdoc/model.json` is missing: suggest `archdoc generate .` to the user.
