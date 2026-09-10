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

**Early.** Release R1.a is in progress, targeted at 9 October 2026. It draws:

```console
$ archdoc generate ./immich

wrote ./immich/docs/architecture/architecture.generated.md
wrote ./immich/docs/architecture/context.mmd
wrote ./immich/docs/architecture/container.mmd
wrote ./immich/.archdoc/model.json

5 elements, 3 relationships, from docker/docker-compose.yml
```

The generated markdown holds a **system context** diagram, a **container** diagram, and the
evidence for both:

| Element | Type | Technology | Evidence | Declared at |
|---|---|---|---|---|
| User | Person | — | declared | `docker/docker-compose.yml:26` |
| immich-machine-learning | Container | — | declared | `docker/docker-compose.yml:34` |
| immich-server | Container | — | declared | `docker/docker-compose.yml:13` |
| database | Container (data store) | PostgreSQL 14 | declared | `docker/docker-compose.yml:57` |
| redis | Container (data store) | Valkey 9 | declared | `docker/docker-compose.yml:50` |

Open any of those lines and the fact is there. That is the whole claim.

Note what is *absent*: `immich-machine-learning` appears as a box and takes part in no
relationship at all. Immich reaches it over a URL assembled at runtime, so its own
configuration never declares the link. The diagram is right to leave the arrow out, and saying
so plainly is more useful than drawing a line nothing supports.

**No language model is involved in any of the above**, and none will be: labels come from a
lookup table, and an image the table does not know gets an empty technology rather than a
guess. The model's job, when it arrives, is to make those labels read well — never to produce
them.

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
go build -o ./archdoc ./cmd/archdoc
```

The built binary is git-ignored, so it can sit in the working copy.

## Running it on the test subjects

The three subjects are **not** in this repository — they are other people's code. Clone them
yourself, at the revisions pinned in
[`docs/survey-test-subjects.md`](docs/survey-test-subjects.md) §Method, and keep them beside
this directory rather than inside it:

```
University/SP2/
├── archdoc/          this repository
└── subjects/
    ├── immich/
    ├── mastodon/
    └── supabase/
```

**Look before writing.** `--stdout` prints the whole document and touches nothing:

```console
./archdoc generate ../subjects/immich   --stdout
./archdoc generate ../subjects/mastodon --stdout
./archdoc generate ../subjects/supabase --stdout
```

**Then write it in,** which is what a real user would run. Four files land in the subject:
`docs/architecture/architecture.generated.md`, two `.mmd` files beside it, and
`.archdoc/model.json`.

```console
./archdoc generate ../subjects/supabase
```

To see the diagrams, open the generated markdown in VS Code and press `⇧⌘V` — Mermaid renders
natively, with no build step and no site generator. That is the point of emitting Mermaid:
GitHub renders it the same way.

**Start with Supabase.** It is the only subject with an external system, the only one with an
excluded gateway, and the only one where the distinction between "declares a dependency" and
"declares that traffic flows" is visible.

### Checking it rather than trusting it

```console
./archdoc scan ../subjects/immich --explain   # why that file, and not the other nine
./archdoc scan ../subjects/supabase --json    # the raw FactSet, every fact with its line
./archdoc history ../subjects/supabase        # every version, and when the architecture moved
```

Pick any citation from an evidence table and open it. Supabase's
`docker/docker-compose.yml:166` reads `GOTRUE_SMTP_HOST: ${SMTP_HOST}` — the diagram shows
`supabase-mail`, resolved from `.env`, and the citation still lands on text a person can read.
That is the two-pass extractor working: one pass knows what is true, the other knows where it
was written.

**Determinism** — five runs must be byte-identical (AC-7):

```console
for i in 1 2 3 4 5; do ./archdoc generate ../subjects/supabase --stdout | md5; done | sort -u
```

One line of output means five identical runs.

### What each subject shows

| | What to look for |
|---|---|
| **Immich** | `immich-machine-learning` has **no edges at all**. Every service reaches it over a URL built at runtime, so configuration never declares the link. Correct, and the finding the whole survey turns on |
| **Mastodon** | A clean five-container diagram, and an empty context diagram. Its external dependencies are real and live in `.env.production.sample`, which the compose file references and which is not in the repository |
| **Supabase** | `api-gw` excluded as infrastructure and named under *Not shown*; `supabase-mail` as a referenced external system; relationships labelled `connects to [postgres]` where a URL knew the protocol and `depends_on` did not |

One line covers all three: **the container view is as good as the compose file, and the context
view is as good as the environment — which is usually somewhere else.**

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
