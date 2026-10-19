# archdoc — Platform plan: the development environment

> **What this document is:** the plan for the step after the freeze — setting up everything needed
> to develop archdoc as a platform. Not the platform itself: the repositories, the engine's public
> API, the stack, and one request going end to end, so feature work has somewhere to land.
>
> **Why it exists:** `hosted-proposal.md` §8 settled the direction on 2026-10-19 — local and
> hosted, a business as an aim, "verifiable, wherever it runs". This is its first concrete step.
>
> **Status:** planned 2026-10-19, not started. The code is frozen for new features until this plan's
> exit criteria (§5) are met; the work below is allowed under the freeze because it restructures and
> sets up, and adds no capability.

---

## 1. The stack

| Layer | Choice | Why |
|---|---|---|
| Backend | **Go** | Imports the engine directly; one language from engine to API |
| Database | **Postgres** | The platform's own data — users, organisations, repositories, runs, jobs, later billing — is relational and needs joins and transactions |
| Flexible data | **`JSONB` columns** | The engine's output (`model.json`, coverage, history) is a document whose shape belongs to the engine and grows with it. Stored whole, queried into when needed, no migration when a field is added |
| Large files | **Object storage** — S3-compatible; MinIO locally | Generated sites and SVGs are files, not rows |
| Driver, queries | **pgx**, **sqlc** | Plain SQL, type-safe Go generated from it; no ORM |
| Migrations | **goose** | Numbered SQL files. Not knowing the schema yet is answered by making schema changes cheap, not by having no schema |
| Job queue | **River** | Runs inside Postgres — no Redis, no extra service. Postgres + object storage is the whole stack |
| Frontend | **React 19 + TypeScript + Vite** — the same base as today's app | Keeps every screen already built; the mainstream choice |
| Routing | **TanStack Router** | Type-safe URLs, once there are organisations, repositories, runs and settings |
| Server data | **TanStack Query** | Caching, retries and live job status without hand-written fetch code |
| Components | **Radix primitives** (shadcn/ui on top where it helps), styled with archdoc's own design system | Accessible menus, dialogs and forms; the look stays the Claude Design project's |
| Forms | **React Hook Form + Zod** | Sign-up, settings, connecting repositories |
| Diagrams | **React Flow** (`@xyflow/react`) — as today | The standard for this |
| Rendering | **A single-page app served as static files**; Go serves the API | No Node server in production. If public gallery pages must be found by search engines, Go pre-renders those few pages — search does not decide the stack of the whole app |
| Local dev stack | **Docker Compose** — Postgres, MinIO | The engine's "no services" rule is the engine's. The platform is a service |

**Considered and not chosen.** *MongoDB* — the schema would still exist, but in Go code and
unchecked; accounts, permissions and billing are its weakest fit; and one more service to run.
*Vue* — a rewrite of ~5,700 lines and of the diagram layer, for no gain. *Next.js* — a Node server in
production beside Go, to solve search for a handful of pages.

---

## 2. Repositories

| Repository | Visibility | Licence | Holds |
|---|---|---|---|
| `archdoc` — this one, `github.com/cruzambrociogl/archdoc` | **Public** (step 3) | **FSL** — free to read, use, modify and self-host; no competing service; each version becomes Apache 2.0 after two years | The engine, its public API, the CLI, the local web app, the shared viewer, the GitHub Action later |
| `archdoc-platform` — new | **Private** | Closed | API server, worker, platform web app, migrations, deployment |

- **Local development across both:** a Go workspace (`go.work`) in the platform repository points at a
  sibling checkout of the engine, so the platform builds against local engine changes without a
  release per change. `go.work` is not committed; CI builds against a tagged engine version.
- **Engine versions:** semantic version tags (`v0.x.y`) on the engine; the platform pins one.
- **The viewer** (step 2) is published from the engine repository as a package the platform installs.

---

## 3. Steps

Ordered by dependency. Each says what is done when it is done.

### Step 1 — The engine's public API

Go forbids importing `internal/` from another module, and the pipeline lives in `package main`:
`document()` in `cmd/archdoc/generate.go` is ~370 lines, with more logic in `diff.go`, `status.go`,
`show.go` and `export.go`. Nothing in it can be called from outside.

- A public package — the module root, `import "github.com/cruzambrociogl/archdoc"` — with the
  entry points the CLI and the platform both need: `Generate`, `Diff`, `Status`, `ExportSite`.
- Options and results as plain structs; progress reported through a callback or `io.Writer`, never
  printed by the library.
- The public types the API returns: either type aliases of `internal/archdoc` types or the types
  moved out. Decided in the step; aliases are the smaller change.
- The CLI becomes a client of the API and keeps no pipeline logic.

**Done when:** the CLI calls only the public API; `generate` on Immich and Supabase at the pinned
SHAs is byte-identical to before (AC-7); the network-boundary check passes unchanged; an example
program outside the module generates a repository through the API.

### Step 2 — The shared viewer

The diagram explorer, flows, components, citations, coverage and documents screens are needed by the
local app and the platform alike. Two copies would drift.

- Split `web/` into a **viewer** package (screens that show a model) and the **local app** (the
  shell around it: navigation, corrections, network runs — what only a local user does).
- The viewer takes its data through an interface, not from `archdoc serve`'s endpoints directly, so
  the platform can feed it from its own API.
- An npm workspace in the engine repository; the viewer published as a package.

**Done when:** the local app is built from the viewer and works as before; `export --site` output is
unchanged; the viewer builds and renders a stored model on its own.

### Step 3 — The engine goes public

- `LICENSE` with the FSL; the README states what the licence allows.
- Review what is in the repository and its history before it is public — `docs/origin/`,
  `docs/brief.md`, the report material, anything naming people or courses — and decide what stays.
- First tagged version, `v0.1.0`.

**Done when:** the repository is public, licensed and tagged, and `go get` of the tag works from a
clean machine.

### Step 4 — The platform repository

```
archdoc-platform/
  cmd/api/          HTTP API
  cmd/worker/       runs jobs: clone, generate, export, store
  internal/         platform packages
  db/migrations/    goose
  db/queries/       sqlc
  web/              the platform app — imports the viewer
  compose.yaml      Postgres, MinIO
  .env.example
  CLAUDE.md
```

- `compose.yaml` brings up Postgres and MinIO; one command starts the whole stack.
- A task runner with the few commands anyone needs — up, migrate, generate (sqlc), test, run.

**Done when:** a fresh clone with Docker and Go goes from nothing to a running API and worker with
one documented command.

### Step 5 — The data foundation

Only the tables the first slice needs. Accounts, organisations and billing wait for piece D.

- `repositories` — owner, name, URL.
- `runs` — repository, commit, status, timings, the model and coverage as `JSONB`, the site's
  object-storage key.
- River's own tables, through its migrations.

**Done when:** migrations apply up and down cleanly; sqlc generates without diff in CI.

### Step 6 — One request, end to end

The walking skeleton: proof that every layer connects, not a feature.

1. `POST /runs` with a public GitHub URL.
2. A River job: the worker shallow-clones at the default branch's commit into a throwaway
   directory, calls the engine's `Generate` and `ExportSite`, uploads the site to object storage,
   stores the model, and **deletes the clone, on success or failure**.
3. `GET /runs/{id}` reports status; when done, the viewer shows the result.
4. A second request for the same `repo@commit` is served from what is stored (AC-7 makes that safe).

**Done when:** pasting Immich's URL in a local environment shows its documentation, and the clone is
gone afterwards.

### Step 7 — The platform web app

- Vite + React + TypeScript, TanStack Router and Query, Radix, the design system's tokens.
- Three screens only: paste a link; a run in progress; the result, through the viewer.
- **Readable before complete** (`hosted-proposal.md` §8): every screen answers one question in plain
  words first, with the evidence one step behind. This is set from the first screen, not retrofitted.

**Done when:** the step 6 flow works through the browser, not only through the API.

### Step 8 — Continuous integration

- **Engine:** as today — gofmt, vet, build, race tests, the network boundary — plus the API example
  and the viewer build.
- **Platform:** gofmt, vet, tests against a Postgres service, migrations up and down, sqlc diff,
  web typecheck and build.

**Done when:** both are green on `develop` and block a broken pull request.

### Step 9 — Project context

- The platform repository gets its own `CLAUDE.md`, `PROGRESS.md` and `docs/decisions.md`, small
  from the start — it must not inherit the engine's 4,750 lines of documents.
- The engine's `CLAUDE.md` and `PROGRESS.md` describe the engine as one part of a larger product.
- `product-definition.md` §7 and `vision.md` §2.2 amended, as the 2026-10-19 decision requires.

**Done when:** a new session in either repository orients itself from its own context file alone.

---

## 4. Out of scope for this phase

Accounts and sign-in, organisations, private repositories, the GitHub App, billing, AI on the hosted
path, deployment to a public host, and every S- feature of `hosted-proposal.md` §7. Each needs this
environment first; none of it is needed to have it.

---

## 5. Exit criteria — when the freeze lifts

1. The CLI runs on the engine's public API, with output byte-identical to before.
2. The local app and the platform show models through one shared viewer.
3. The engine repository is public, FSL-licensed and tagged.
4. A fresh clone of the platform runs locally with one command.
5. Pasting a public repository's URL in the local platform shows its documentation, end to end.
6. CI is green and enforcing in both repositories.
7. Each repository has its own context, and the definition and vision are amended.

---

## 6. Open questions

1. **Public types** — aliases of `internal/archdoc`, or moved out (step 1)?
2. **The viewer package** — published to npm, to GitHub Packages, or consumed from git (step 2)?
3. **What stays out of the public repository** — the report, the origin notes, the brief (step 3)?
4. **The product's name and domain** — does the platform stay "archdoc"?
5. **Hosting** — Fly.io, Render, Railway, a cloud provider. Not needed until deployment, which is
   outside this phase; worth knowing early because it shapes object storage and Postgres choices.
6. **Authentication** — build it or use a provider. Outside this phase; decided when piece D starts.
