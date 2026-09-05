# ArchDoc

**Generates architecture documentation from a repository's own configuration — every fact
traceable to the file and line that proves it.**

---

## The problem

AI now writes most of the code while a person directs it. That works, and it is fast — but the
understanding that normally forms while typing never forms. Reviewing confirms the result
works; it does not build a mental model. Weeks later the person who directed every decision is
as lost as a stranger would be, with nobody to ask, because nobody ever knew.

Documentation debt assumes someone knew and failed to write it down. **This is knowledge that
never existed.**

## The approach

Point it at a repository. It reads the configuration — Compose files, `.env`, gateway configs,
interface contracts — builds a validated model, and renders C4 diagrams and arc42
documentation where **every element links back to the file that proves it exists**. Run it
again after new commits and it reports what changed architecturally.

Two rules govern the design:

1. **The diagram is never the source of truth — a validated model is.**
2. **The LLM is never responsible for facts.** It supplies labels and prose on top of facts
   extraction already established. Remove it entirely and the tool still produces a complete,
   correct, traceable diagram — just with duller labels.

## Status

**Early.** Release R1.a is in progress, targeted at 9 October 2026. One command works today:

```console
$ archdoc scan ./immich

database                 ghcr.io/immich-app/postgres:14-vectorchord0.4.3…   docker/docker-compose.yml:57
immich-machine-learning  ghcr.io/immich-app/immich-machine-learning:v3      docker/docker-compose.yml:34
immich-server            ghcr.io/immich-app/immich-server:v3                docker/docker-compose.yml:13
redis                    docker.io/valkey/valkey:9@sha256:3acc0687f2a2e10…  docker/docker-compose.yml:50

4 services from docker/docker-compose.yml · 10 compose file(s) considered, 7 deployable
```

No edges, no model, no diagram yet — four services and the lines that prove them. That is the
part everything else rests on.

### Why discovery is harder than a glob

```console
$ archdoc scan ./immich --explain

Discovery:
    .devcontainer/mobile/container-compose-overrides.yml  deployable — 2 services with an image or build
    .devcontainer/server/container-compose-overrides.yml  deployable — 2 services with an image or build
    docker/docker-compose.dev.yml                         deployable — 6 services with an image or build
    docker/docker-compose.prod.yml                        deployable — 6 services with an image or build
    docker/docker-compose.rootless.yml                    deployable — 4 services with an image or build
  → docker/docker-compose.yml                             selected — deployable — 4 services with an image or build
    docker/hwaccel.ml.yml                                 fragment — no service declares an image or a build
    docker/hwaccel.transcoding.yml                        fragment — no service declares an image or a build
    e2e/docker-compose.dev.yml                            fragment — no service declares an image or a build
    e2e/docker-compose.yml                                deployable — 4 services with an image or build
```

Immich contains **ten** Compose files describing four different systems. A filename glob misses
the two under `.devcontainer/` — the glob written for this project's own survey missed them.
Content-sniffing finds them but then admits `hwaccel.ml.yml`, whose "services" are named `cpu`,
`armnn` and `rknn` and are hardware snippets rather than anything deployable.

Recall comes from sniffing; precision comes from rejecting services that declare neither an
image nor a build. `--explain` exists because *"trust me, I picked the right file"* is not good
enough for a tool whose entire premise is traceability.

## Built on evidence

Before any code was written, two production repositories were read and measured — self-hosted
[Immich](https://github.com/immich-app/immich) and
[Supabase](https://github.com/supabase/supabase), at pinned revisions. Findings that changed
the design:

- **Immich declares none of its three service connections in configuration.** All three live in
  TypeScript source. A configuration-driven extractor will miss them, and the accuracy target
  for that subject is expected to fall short as a result — reported as a measured limitation of
  declaration-based extraction rather than treated as a defect.
- **Compose needs a specification-grade parser.** Seven parameter-expansion forms, nesting,
  concatenation, and the `!override` / `!reset` tags — where mishandling `!override` appends a
  list instead of replacing it and yields a *silently wrong* diagram.
- **`depends_on` is not an edge source.** It captures 1 of 7 gateway edges in Supabase.

The full survey is in [`docs/survey-test-subjects.md`](docs/survey-test-subjects.md).

## Building

Requires Go 1.27. Contributors need Node for the web frontend; users do not.

```console
git clone https://github.com/cruzambrociogl/archdoc.git
cd archdoc
go test ./...
go install ./cmd/archdoc
```

## Documentation

| | |
|---|---|
| [`docs/how-it-works.md`](docs/how-it-works.md) | **The pipeline on one screen** — start here |
| [`docs/product-definition.md`](docs/product-definition.md) | What the product is — 109 capabilities, the output contract, nine acceptance criteria. Technology-free by design |
| [`docs/stack-decision.md`](docs/stack-decision.md) | What it is built with, and why, including the rejected alternatives |
| [`docs/survey-test-subjects.md`](docs/survey-test-subjects.md) | The evidence both rest on |
| [`docs/delivery-schedule.md`](docs/delivery-schedule.md) | The plan through 9 October |
| [`docs/decisions.md`](docs/decisions.md) | Decision log — read before proposing anything that seems obvious |
| [`PROGRESS.md`](PROGRESS.md) | What is done, and which gate is next |

New to the repository? [`CLAUDE.md`](CLAUDE.md) opens with a reading order.

---

*Seminar project · Universidad Galileo · 2026*
