# Testing — the engine

How the engine is tested and what must pass before a change lands. A map, not a copy: the
criteria are defined in `docs/product-definition.md` §12 and their status is kept in `PROGRESS.md`.

## What runs

| Check | Command | Where |
|---|---|---|
| Formatting | `gofmt -l .` prints nothing | Locally and CI |
| Vet | `go vet ./...` | Locally and CI |
| Tests, race detector on, no cache | `go test -race -count=1 ./...` | Locally and CI |
| The network boundary — only `internal/semantic` may make outbound calls (hard rule 3, AC-8) | the `network-boundary` job in `.github/workflows/ci.yml` | CI |
| The public API is enough: `cmd/archdoc` imports nothing from `internal/`, and `examples/library` — a module of its own — documents a fixture through it | the `public-api` job | CI |
| The web app — both packages — typechecks and builds | `cd web && npm run build`; the `web` job | Locally and CI |
| Every screen renders the same, live and published | headless Chrome over each screen, text compared with the previous build — by hand, as for step 2 (`docs/decisions.md`, 21 Oct) | Before a release |

## The acceptance criteria

Nine, defined in `docs/product-definition.md` §12, tracked in `PROGRESS.md`. The self-checking ones
run as tests on every change:

- **AC-4** — the validator rejects every model in the fault-injection suite.
- **AC-5** — rules survive regeneration, measured by running the whole pipeline twice from disk.
- **AC-6** — drift detection over the synthetic drift set.
- **AC-7** — determinism: five runs, byte-identical output. **Sort map keys before iterating
  whenever the result reaches output** (hard rule 1); an unsorted map is the most likely way to
  break it.
- **AC-8** — egress: structure-only runs transmit zero file contents, measured on the wire against
  a local fake of the API. Tests never call the real API and cost nothing.

AC-1, AC-3 and AC-9 need a subject or a person, and are measured at reviews.

## Test data

- **`testdata/`** — fixtures standing in for *other people's* repositories (hard rule 6). Nothing in
  it is archdoc's own output.
- **The test subjects** — Immich, Supabase, Mastodon and others, cloned outside this repository at
  the commits pinned in `docs/survey-test-subjects.md` §Method. Every claim about them is anchored
  to those commits.
- **`docker compose config`** is a development-time oracle only; archdoc itself runs with no
  container runtime (hard rule 7).

## Paid runs

Anything calling the Anthropic API — `--label`, `--explain` — costs money and is never part of the
test suite. It is run by hand, by whoever holds the key.
