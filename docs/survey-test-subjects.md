# Test-subject survey — extraction reality check

> Empirical input to the stack conversation (§13, O-2/O-4/O-5 of `product-definition.md`).
> Read-only pass over the test subjects: what config actually exists, in what dialects,
> where contract files sit, and what would break a naive parser.
>
> **This document makes no stack recommendation.** It is evidence, gathered before the
> decision, so the decision has something to stand on.

## Method

Repos cloned, not run. Every claim below cites `file:line` in the pinned revision.

| Subject | Repo | Pinned revision |
|---|---|---|
| Immich (simple) | `github.com/immich-app/immich` | `cbf5d83a693d0328282ddb0f5d398c351d92558e` |
| Supabase (complex) | `github.com/supabase/supabase` | `34454037d31a817654e514f9748590989496cf0b` |

Order: cold-start discovery pass first (catalog what is findable *before* reading deeply,
since that is what DSC must do), then a deep read of every config file found.

---

# Part 1 — Immich

## 1.1 Headline finding

**Immich's service inventory is declared. Its wiring is not.**

The four services are cleanly declared in `docker/docker-compose.yml`. But of the three
protocol edges between them, **zero are declared in configuration**. All three are
defaults compiled into the application's TypeScript source:

| Edge | Where the target is actually declared | Kind |
|---|---|---|
| `immich-server` → `immich-machine-learning` | `server/src/dtos/config.dto.ts:624` — `'http://immich-machine-learning:3003'` | TS default |
| `immich-server` → `redis` | `server/src/repositories/config.repository.ts:203` — `dto.REDIS_HOSTNAME \|\| 'redis'` | TS default |
| `immich-server` → `database` | `server/src/repositories/config.repository.ts:237` — `dto.DB_HOSTNAME \|\| 'database'` | TS default |

`docker/example.env` (22 lines, read in full) contains **no hostnames, no URLs, no ports** —
only storage paths, a version pin, and Postgres credentials. There is no `.env.example`
carrying `DATABASE_URL=postgres://…`-style evidence of the kind §3 uses as its worked
example.

What configuration *does* offer is `depends_on: [redis, database]`
(`docker/docker-compose.yml:27-29`). That is startup ordering, not a protocol edge — it
happens to coincide with two real edges and **misses the server → machine-learning edge
entirely**, since ML has no `depends_on` entry anywhere in the release compose file.

### Why this matters to the definition

§0 states: *"In a service-based system the architecture is declared — in compose files,
proxy config, manifests, and interface contracts — not inferable from source."*

On Immich that is **half true, and the half that fails is the half that draws the arrows.**
Nodes: declared. Edges: in source. A config-only extractor run against Immich produces
4 correct boxes and, if it trusts `depends_on`, 2 of 3 arrows — with the missed arrow being
the one that distinguishes Immich from a generic web app.

This does not invalidate the approach. It sharpens a question the stack conversation should
carry: **where do edges come from when `depends_on` is the only config signal?** Options, in
increasing cost: catalog-inferred edges from image + exposed ports; `rules.yaml` by hand
(and AC-5 already assumes rules carry real weight); or pulling R1.b's static analysis
forward. Worth noting that the env var *names* are themselves declared and greppable —
`DB_HOSTNAME`, `REDIS_HOSTNAME`, `IMMICH_MACHINE_LEARNING_URL` are all schema-declared in
`server/src/dtos/env.dto.ts:73-88`, which is a narrower and more tractable target than
general static analysis.

## 1.2 Discovery surface — which file *is* the architecture?

Cold-start glob for architecture-declaring files returned **six compose files**, describing
**four different systems**:

| File | Services | What it describes |
|---|---|---|
| `docker/docker-compose.yml` | 4 | The release topology — what users actually deploy |
| `docker/docker-compose.prod.yml` | 6 | Adds `immich-prometheus`, `immich-grafana` |
| `docker/docker-compose.dev.yml` | 7 | Adds `immich-web`, `immich-init`, `immich-app-base` |
| `docker/docker-compose.rootless.yml` | 4 | Rootless variant |
| `e2e/docker-compose.yml` | 3 | Test harness |
| `e2e/docker-compose.dev.yml` | — | Test harness variant |

**Discovery cannot punt on this.** Picking `docker-compose.dev.yml` yields a diagram with a
web service and an init container that no user ever runs; picking `prod.yml` adds an
observability stack absent from the real deployment. The answer changes the documented
architecture, not just its detail level.

This is a **DSC design requirement** (§4.1), and it is not solved by a glob. Immich hints at
the answer through naming convention (unsuffixed = canonical) and location (`docker/` over
`e2e/`), but that is a heuristic, and `rules.yaml` needs a way to override it.

### Files that look like compose and are not

Two traps in the same directory:

- **`docker/hwaccel.ml.yml` and `docker/hwaccel.transcoding.yml`** — these have a top-level
  `services:` key and parse cleanly as compose, but define hardware-acceleration *fragments*
  named `cpu`, `armnn`, `rknn`, `cuda`, `rocm`, `openvino`. A discovery stage that globs
  `*.yml` and checks for `services:` will emit **boxes named "cpu" and "armnn"**. The
  `services:` key is not a sufficient test.
- **`docker/prometheus.yml`** — not compose (no `services:` key, so it fails safe), but see
  §1.4: it carries real edge facts and should be read by a *different* parser.

### Terraform that is not this system's architecture

`deployment/` contains Terraform + Terragrunt (`deployment/modules/cloudflare/docs/*.tf`) —
but it provisions **Cloudflare hosting for the documentation site**, not Immich. A
discovery stage that treats "IaC files" as architecture evidence will merge a docs CDN into
the photo-management system's container diagram.

The lesson generalizes: *file type does not imply relevance.* Discovery needs a notion of
which deployment target a config describes, or a way for rules to exclude a subtree.

## 1.3 Parser surface — dialect features actually in use

Everything below appears in Immich's compose files and must be handled or deliberately
rejected:

| Feature | Example | Note |
|---|---|---|
| Env interpolation with default | `${IMMICH_VERSION:-release}` | `docker-compose.yml:15` |
| Env interpolation, no default | `${UPLOAD_LOCATION}:/data` | `:21` — unresolvable without `.env` |
| Top-level project name | `name: immich` | `:10` |
| `depends_on` short form | list of names | `:27` |
| `healthcheck: disable: true/false` | `:31-32` | non-obvious shape |
| Same-file `extends` | `extends: { service: immich-app-base }` | `dev.yml:37-38` |
| Cross-file `extends` (commented) | `extends: { file: hwaccel.transcoding.yml, service: cpu }` | `docker-compose.yml:16-18` |
| **`profiles` with the `!reset` YAML tag** | `profiles: !reset []` | `dev.yml:40` — **Compose-specific custom YAML tag; a generic YAML loader errors or mishandles it** |
| Abstract base service | `immich-app-base` with `profiles: ['_base']` | `dev.yml:17-19` — must **not** become a box |
| Named volumes | `model-cache:`, `sveltekit:` | `:75-76` |
| Bind mount from env var | `${DB_DATA_LOCATION}:/var/lib/…` | `:69` |
| `shm_size` | `128mb` | `:70` |
| Git submodule | `e2e/test-assets` | `.gitmodules` — a path that exists in config but not on disk after a plain clone |

**`profiles: !reset []` is the sharpest parser-surface finding.** `!reset` is a Compose
extension to YAML, not YAML itself. Whatever runtime is chosen, the question "does its YAML
library let me register unknown tags without dying?" is now a concrete, testable
requirement rather than a hypothetical.

**No `networks:` key appears in any Immich compose file.** All services sit on the implicit
default network. §3 promises "network and deployment boundaries" in R1 output — on Immich
that extraction yields **zero boundaries**. Boundary rendering needs a defined behaviour for
the common case where nothing is declared.

## 1.4 Edge evidence outside compose

`docker/prometheus.yml` is a Prometheus scrape config carrying two genuine edges:

```
scrape_configs:
  - job_name: immich_api            targets: ['immich-server:8081']   # :6-8
  - job_name: immich_microservices  targets: ['immich-server:8082']   # :10-12
```

Two useful facts: `immich-prometheus` → `immich-server` is a real edge found in a
non-compose file, and it reveals **ports 8081/8082 that appear nowhere in the release
compose file's `ports:` list** (which publishes only `2283`).

This is the design's "proxy config and other declarations" thesis working — but it also
means the parser inventory is open-ended: every tool with a config format is a potential
edge source, and coverage is a curation decision, not a completeness claim.

## 1.5 O-4 — contract-file matching

**Immich answers O-4 clearly, and the answer is "neither of the two easy options."**

The spec is at `open-api/immich-openapi-specs.json` — 833 KB, OpenAPI 3.0.0, 190 paths.

- **Directory proximity fails.** The spec sits in a top-level `open-api/` directory. The
  service that serves it, `immich-server`, is built from `server/`. They are siblings, not
  parent-child; proximity would attach the spec to nothing, or to the wrong thing.
- **Self-identification fails.** `"servers": [{"url": "/api"}]` — a relative path with no
  host and no port. The spec does not say who serves it.
- **Config reference fails on the primary compose file.** The chain that *would* work is
  compose service → `build.context` → `server/` → the code that emits the spec
  (`server/src/utils/misc.ts:335`). But `docker/docker-compose.yml` has **no `build:` key at
  all** — it consumes a prebuilt image `ghcr.io/immich-app/immich-server`. The chain exists
  only in `dev.yml`/`prod.yml`, which are not the canonical file.

**Conclusion: on the canonical compose file, a rule is required.** No automatic heuristic
attaches this spec to its service. That is a finding worth carrying into O-4's resolution —
"directory proximity or config reference" may not be an exhaustive menu, and `rules.yaml`
carries more weight than §4 implies.

## 1.6 O-5 — image inventory and what the catalog must recognize

13 distinct image strings across all compose files:

| Category | Count | Examples |
|---|---|---|
| Recognizable common infrastructure | 3 | `grafana/grafana`, `prom/prometheus`, `docker.io/valkey/valkey` |
| Infrastructure under a **private namespace** | 1 | `ghcr.io/immich-app/postgres:14-vectorchord0.4.3-pgvectors0.2.0` |
| First-party application images | 3 | `ghcr.io/immich-app/immich-server`, `…/immich-machine-learning` |
| Locally-built, no registry | 6 | `immich-server-dev:latest`, `immich-web-dev:latest`, … |

Three catalog problems, all concrete:

1. **`ghcr.io/immich-app/postgres` is Postgres**, but a catalog keyed on well-known image
   names will not match it. Only the substring `postgres` in a private repo path saves you —
   and substring matching on image paths is exactly the kind of heuristic that produces
   confident wrong answers.
2. **`docker.io/valkey/valkey:9` backs a service named `redis`.** Valkey is a Redis fork.
   The catalog needs the equivalence, or the diagram labels a Redis-protocol store "Valkey"
   while the service name says "redis" and the connecting code says `REDIS_HOSTNAME`.
3. **Reference-form variety.** Present in this one repo: `name:tag`, `name:tag@sha256:…`,
   `name@sha256:…` (digest, no tag — `prom/prometheus@sha256:508729e0…`), bare `name`
   (`prom/prometheus`), registry-qualified and bare, and `name:${VAR:-default}`. Naive
   string dedup yields two distinct Grafanas (`10.3.3-ubuntu` and `12.4.8-ubuntu@sha256:…`).

Answering O-2's sub-question — *"whether the catalog needs entries beyond common
infrastructure images"* — **yes, on the evidence of the simple subject.** 10 of 13 images
here are not identifiable from a common-image catalog alone.

## 1.7 Scale

| Metric | Canonical compose | All files combined |
|---|---|---|
| Services | 4 | ~13 distinct |
| Declared edges (`depends_on`) | 2 | — |
| True protocol edges | 3 | — |
| Networks | 0 | 0 |
| Named volumes | 1 | ~12 |

Comfortably inside §7's "dozens to low hundreds of nodes." The embedded-storage premise is
not challenged by the simple subject.

---

# Part 2 — Supabase

Pinned revision: `34454037d31a817654e514f9748590989496cf0b`.

## 2.1 Headline finding

**Supabase is the mirror image of Immich: here the wiring *is* declared, in exactly the
form §3 assumes.**

Eleven services in `docker/docker-compose.yml`, and the edges appear as URL-valued
environment variables with resolvable hosts and ports:

```
docker-compose.yml:33   STUDIO_PG_META_URL:     http://meta:8080
docker-compose.yml:51   SUPABASE_URL:           http://api-gw:8000
docker-compose.yml:136  GOTRUE_DB_DATABASE_URL: postgres://…@${POSTGRES_HOST}:${POSTGRES_PORT}/…
docker-compose.yml:263  PGRST_DB_URI:           postgres://…@${POSTGRES_HOST}:${POSTGRES_PORT}/…
docker-compose.yml:361  POSTGREST_URL:          http://rest:3000
docker-compose.yml:388  IMGPROXY_URL:           http://imgproxy:5001
docker-compose.yml:565  DATABASE_URL:           ecto://…@${POSTGRES_HOST}:${POSTGRES_PORT}/_supabase
```

`${POSTGRES_HOST}` resolves from `.env.example:111` (`POSTGRES_HOST=db`) — so the
substitution chain the design assumes is real and works here.

**The two subjects bracket the problem.** Immich declares nodes but not edges; Supabase
declares both. A tool that only works on Supabase-shaped repos will look excellent on the
complex subject and produce a near-edgeless diagram on the simple one. Any accuracy claim
(AC-1, AC-3) needs to be reported per-subject, not averaged.

## 2.2 The identity problem, demonstrated

**This is the strongest empirical support in either subject for the identity registry
(§4.3), and it is worth citing directly in the definition.**

Every Supabase service carries at least two names, and different config files reference
different ones:

| Compose service | `container_name` | Network aliases | Image | Referenced by Envoy as |
|---|---|---|---|---|
| `api-gw` | `supabase-envoy` | **`envoy`, `kong`** | `envoyproxy/envoy` | — (is the gateway) |
| `auth` | `supabase-auth` | — | `supabase/**gotrue**` | `auth` |
| `realtime` | **`realtime-dev.supabase-realtime`** | — | `supabase/realtime` | **`realtime-dev.supabase-realtime`** |
| `functions` | `supabase-edge-functions` | — | `supabase/edge-runtime` | `functions` |
| `supavisor` | `supabase-pooler` | — | `supabase/supavisor` | — |
| `db` | `supabase-db` | — | `supabase/postgres` | — |

Three distinct failure modes for a naive extractor:

1. **Envoy references `realtime` by its `container_name`, not its service name**
   (`volumes/api/envoy/cds.yaml:82` → `realtime-dev.supabase-realtime`), while referencing
   all six other services by service name. Key on service name alone and you get a phantom
   node plus an unresolved edge. The compose file even apologises for it:
   *"This container name looks inconsistent but is correct because realtime constructs
   tenant id by parsing the subdomain"* (`docker-compose.yml:286`).
2. **`api-gw` answers to `envoy` and `kong` via network aliases**
   (`docker-compose.yml:74-79`), deliberately, so that configs referencing either hostname
   resolve to whichever gateway is active. One node, four names.
3. **`supavisor` / `supabase-pooler` share no substring**, so string-similarity fallback
   does not save you.

§4.3's canonical-identity-with-alias-history design is the right shape. What this adds is
the **source ranking** question: when compose says `realtime`, the container says
`realtime-dev.supabase-realtime`, the image says `supabase/realtime` and an OpenAPI spec
says `gotrue` — which is canonical, and which are aliases? The registry needs a defined
precedence, not just the ability to hold aliases.

## 2.3 Discovery — and a machine-readable answer to "which file"

Fifteen compose files. But unlike Immich, **Supabase declares which composition is active**:

```
.env.example:11   COMPOSE_FILE=docker-compose.yml
.env.example:3    # Native docker compose COMPOSE_FILE: colon-separated list, base file first.
.env.example:8    #   COMPOSE_FILE=docker-compose.yml:docker-compose.pg17.yml
```

`COMPOSE_FILE` is a **native Docker Compose environment variable**, colon-separated, base
file first. `docker/run.sh` manages it (`run.sh:85-95` rewrites it in `.env`).

**This is an actionable DSC finding.** Reading `COMPOSE_FILE` from `.env` is a
standards-based way to resolve "which file is the architecture" — not a heuristic, not a
convention, but a declaration the deployment tooling itself relies on. It should be the
first thing discovery looks for; the filename heuristics Immich forces on us are the
fallback for when it is absent.

### Overlays make the architecture combinatorial

The remaining files are composition layers, not alternative systems:

| Overlay | Effect |
|---|---|
| `docker-compose.kong.yml` | Replaces the `api-gw` service in place — Envoy → Kong |
| `docker-compose.envoy.yml` | The default, made explicit |
| `docker-compose.nginx.yml` / `.caddy.yml` | Adds an outer reverse proxy |
| `docker-compose.pg15.yml` / `.pg17.yml` | Swaps the `db` image |
| `docker-compose.s3.yml` / `.rustfs.yml` | Adds object storage |
| `docker-compose.pgbouncer.yml` | Adds connection pooling |
| `docker-compose.logs.yml` | Adds `analytics` + `vector` (2 services) |

So the documented system is *base + selected overlays*, and the merge is not a naive deep
merge — see the YAML tags below. Node count ranges from 11 to ~15 depending on
composition. **R1 must state which composition it documented**, and `model.json` should
record the `COMPOSE_FILE` value as provenance. This is a real output-contract requirement
(§8) that neither §4.1 nor §8 currently anticipates.

## 2.4 Parser surface — the hard requirement, now verified

**Both Compose-specific YAML tags appear in both subjects.** Counted across Supabase's
compose files: `!override` × 5, `!reset` × 3.

I attempted to parse `docker/docker-compose.yml` with a stock YAML loader. **It fails.**
The parse only succeeds after explicitly registering constructors for `!override` and
`!reset`. This is no longer a hypothesis about the parser surface — it is a reproduced
failure.

```
docker-compose.kong.yml:16   test: !override ["CMD", "kong", "health"]
docker-compose.kong.yml:20   ports: !override
docker-compose.kong.yml:23   volumes: !override
docker-compose.kong.yml:28   environment: !override
docker-compose.kong.yml:49   entrypoint: !override [...]
immich  docker-compose.dev.yml:40   profiles: !reset []
```

`!override` changes list-merge semantics from *append* to *replace* when overlays are
merged. A parser that strips or ignores the tag produces a **silently wrong merged model** —
Kong's ports appended to Envoy's rather than replacing them. That is worse than a crash.

**Concrete stack requirement:** the chosen runtime's YAML library must support registering
handlers for unknown tags, and the compose merge implementation must honour `!override` /
`!reset` semantics. This is now a testable acceptance question to put to any candidate.

### Other dialect features (Supabase-only, additional to Immich's list)

| Feature | Example |
|---|---|
| **Nested interpolation, 2 levels** | `${API_GW_HTTP_PORT:-${KONG_HTTP_PORT:-8000}}:8000/tcp` — `:91` |
| Nested interpolation in env value | `${JWT_JWKS:-${JWT_SECRET}}` — `:275` |
| Port with protocol suffix | `8000/tcp` |
| `depends_on` **long form** with `condition: service_healthy` | `:87-88`, `:290-293` |
| Service-level `networks:` with `aliases:` | `:74-79` |
| Healthcheck containing an **embedded URL** | `fetch('http://localhost:3000/api/…')` — `:23` |
| SELinux volume flags | `:z`, `:ro,z` |
| `${VAR:-}` (empty default) | `:99-102` |

The repo's own header comment flags the nesting as a portability hazard:
*"Nested variable interpolation (`${A:-${B}}`) requires podman-compose >= 1.6.0"*
(`docker-compose.yml:8`). A regex-based interpolation pass will mis-resolve this.

### False-positive sources for a URL scanner

Two traps, both of which would create phantom nodes:

- **Healthcheck URLs** — `http://localhost:3000/api/platform/profile` (`:23`),
  `http://localhost:4000/api/tenants/…` (`:298`). Self-directed, not edges.
- **External-facing URLs** — `SUPABASE_PUBLIC_URL=http://localhost:8000` and
  `SITE_URL=http://localhost:3000` in `.env.example:97,171`. These describe how a *browser*
  reaches the system, not a service-to-service edge.

A URL-shaped string is not an edge. Distinguishing them needs the host to resolve to a
known service name or alias — which loops back to the identity registry.

## 2.5 Edge evidence in gateway config

`volumes/api/envoy/cds.yaml` is a clean, machine-readable routing table — **seven
gateway→service edges with exact ports**:

| Cluster | Address | Port |
|---|---|---|
| `auth` | `auth` | 9999 |
| `rest` | `rest` | 3000 |
| `realtime` | `realtime-dev.supabase-realtime` | 4000 |
| `storage` | `storage` | 5000 |
| `functions` | `functions` | 9000 |
| `meta` | `meta` | 8080 |
| `studio` | `studio` | 3000 |

`volumes/api/kong.yml` describes **the same seven services** through ~20 route entries in a
completely different schema (`url: http://auth:9999/verify`, …). Two gateway configs, one
topology, two parsers.

**Compare against `depends_on`:** the compose file declares `api-gw → studio` only. So
`depends_on` captures **1 of 7** gateway edges. Combined with the Immich result, the
conclusion is consistent across both subjects: **`depends_on` is not an edge source.** It is
a startup-ordering hint that sometimes coincides with an edge.

## 2.6 O-4 — contract-file matching, second data point

Six OpenAPI specs, all in `apps/docs/spec/` — the **documentation website's** subtree, with
no path relationship whatsoever to `docker/`, where the services live.

| Spec | Format | Self-identification | Maps to |
|---|---|---|---|
| `auth_v1_openapi.json` | **Swagger 2.0** | `host: localhost:9999`, `title: gotrue` | service `auth` |
| `storage_v0_openapi.json` | OpenAPI 3.0.3 | **none** — no `servers`, no `host` | service `storage` |
| `functions_v0_openapi.json` | — | — | service `functions` |
| `analytics_v0_openapi.json` | — | — | `analytics` (**logs overlay only**) |
| `api_v1` / `api_v2_openapi.json` | — | — | **no self-hosted service** — platform API |

Findings:

- **Directory proximity fails again**, more severely than in Immich — the specs are in a
  different application of the monorepo entirely.
- **Format is not uniform.** `auth_v1` is Swagger 2.0 (`host`/`basePath`); `storage_v0` is
  OpenAPI 3.0.3 (`servers`). Two schemas, and the older one is the *only* one that
  self-identifies.
- **A port-matching heuristic exists and works, once.** `auth_v1`'s `host: localhost:9999`
  matches Envoy's `auth:9999`. That is a real attachment signal — but only one spec of six
  offers it.
- **Filename convention is the strongest available signal** (`auth_v1_…` ↔ `auth`) and it
  is genuinely fragile: `api_v1`/`api_v2` match no local service, and `analytics_v0` matches
  a service that exists only under an overlay.
- **Derived duplicates**: `apps/docs/spec/transforms/*_deparsed.json` are generated copies.
  A glob for `*openapi*` double-counts every spec.

## 2.7 O-5 — image inventory

23 distinct images across all Supabase compose files:

| Category | Count | Notes |
|---|---|---|
| Recognizable infrastructure | ~11 | `caddy:2`, `kong/kong`, `envoyproxy/envoy`, `postgrest/postgrest`, `timberio/vector`, `darthsim/imgproxy`, `edoburu/pgbouncer`, `inbucket/inbucket`, `jonasal/nginx-certbot`, `rustfs/rustfs` |
| Infrastructure under a vendor namespace | 3 | **`supabase/postgres:17.6.1.136`**, `cgr.dev/chainguard/minio`, `cgr.dev/chainguard/minio-client` |
| First-party application images | 9 | `supabase/gotrue`, `supabase/studio`, `supabase/realtime`, `supabase/storage-api`, `supabase/edge-runtime`, `supabase/postgres-meta`, `supabase/logflare`, `supabase/supavisor` |

Two patterns confirmed across both subjects:

1. **Well-known infrastructure hides under vendor namespaces.** `supabase/postgres` here,
   `ghcr.io/immich-app/postgres` in Immich. Both are Postgres; neither matches a catalog
   keyed on official image names. This is not an edge case — it is how both subjects ship
   their database.
2. **`cgr.dev/chainguard/minio` is MinIO**, and nothing but the trailing path segment says
   so. Chainguard/distroless rebuilds of common software are increasingly normal.

**Convention differences between subjects:** Immich pins digests aggressively
(`@sha256:…` on 4 images); Supabase uses **no digests at all**. Both spellings must parse.

## 2.8 Scale

Parsed from the canonical `docker-compose.yml` (after registering the custom tags):

| Metric | Base | With overlays |
|---|---|---|
| Services | **11** | up to ~15 |
| `depends_on` entries | 10 | — |
| True gateway edges (Envoy CDS) | 7 | — |
| URL-declared edges in env | 9 | — |
| Published port mappings | 3 | more |
| Volume mounts | 20 | — |
| Top-level named volumes | 2 | — |
| **Top-level `networks:` declared** | **0** | 0 |

Note the last row: like Immich, Supabase declares **no top-level networks**. The only
network configuration anywhere is `api-gw`'s service-level `networks: default:` block,
present solely to attach the `envoy`/`kong` aliases (`:74-79`).

**Both subjects yield zero network boundaries.** §3 lists "network and deployment
boundaries" as R1 output and §7 lists boundaries in the model — on the evidence of both
subjects, that extraction returns nothing, and boundary rendering needs a defined behaviour
for the overwhelmingly common "everything on the implicit default network" case.

Scale is comfortably within §7's "dozens to low hundreds of nodes." The embedded,
file-based storage premise is **not challenged by either subject**, including the complex
one. §7's scale argument holds.

---

# Part 3 — Synthesis

## 3.1 The central result

**Configuration reliably declares *what exists*. It does not reliably declare *what talks
to what*.**

| | Immich | Supabase |
|---|---|---|
| Services declared in config | ✅ 4, cleanly | ✅ 11, cleanly |
| Edges declared in config | ❌ **0 of 3** — all in TS source | ✅ ~9 as URL env vars + 7 in gateway config |
| `.env.example` carries edge evidence | ❌ no hostnames at all | ✅ `POSTGRES_HOST=db` |
| Networks / boundaries declared | ❌ 0 | ❌ 0 |
| Contract files attachable automatically | ❌ rule required | ❌ filename heuristic only |
| Active-composition declared | ❌ 6 files, no signal | ✅ `COMPOSE_FILE` in `.env` |

§3's claim that container level is where "services already are the boxes" is **confirmed on
both subjects** — node extraction is as easy as the definition predicts. The arrows are the
hard part, and their difficulty varies enormously by repo.

This is a genuinely good outcome for the project: it means the *node* half of P1
(everything traceable to a declaration) is solid, and it localises the risk to edges, where
it can be managed explicitly rather than discovered late.

## 3.2 `depends_on` is not an edge source

Consistent across both subjects:

| | Real edges | Captured by `depends_on` | Phantom edges added |
|---|---|---|---|
| Immich | 3 | 2 (coincidentally) | 0 |
| Supabase gateway | 7 | 1 | — |

`depends_on` expresses startup ordering. It overlaps with real edges by coincidence and
misses the most architecturally interesting ones (Immich's ML call; six of seven Supabase
gateway routes). **If R1 renders `depends_on` as edges, it renders a diagram that is wrong
in a way the reader cannot detect** — which is precisely the draw.io failure mode from §0
that this project exists to prevent.

Recommendation for the stack conversation to carry: treat `depends_on` as a **distinct,
labelled evidence kind** (a fourth alongside declared/referenced/inferred, or a
lower-confidence edge attribute), never as a plain protocol edge.

## 3.3 A URL-shaped string is not an edge

Naive URL extraction produces confident garbage. Measured:

| Source | URLs found | Real edges |
|---|---|---|
| Immich compose, `https://` | 24 (16 in comments, **8 GitHub build-metadata env vars**) | **0** |
| Supabase compose, `https://` | 21 (mostly doc links in comments) | ~0 |
| Supabase healthcheck URLs | several `http://localhost:…` | 0 (self-directed) |
| `SUPABASE_PUBLIC_URL`, `SITE_URL` | 2 | 0 (browser-facing) |

Immich's `IMMICH_REPOSITORY_URL`, `IMMICH_SOURCE_URL`, `IMMICH_BUILD_URL`,
`IMMICH_BUILD_IMAGE_URL` are all `https://github.com/...`. A scanner applying §3's
"referenced" evidence rule would emit **github.com as an external system in Immich's
architecture**. It is build provenance, not topology.

§3 offers `S3_BUCKET=training-data` as the model case for referenced evidence, and that case
is sound. But the symmetric failure is real and common: **URL-valued env vars are more often
metadata than topology.** The discriminator that works on this evidence is whether the host
resolves to a known service name or alias — which makes the identity registry a
*prerequisite* for edge extraction, not a parallel concern.

### Protocol does not always come free from the scheme

§3: *"Protocol usually comes free from the URL scheme (`postgres://`, `redis://`, `amqp://`,
`s3://`, `https://`)."* Schemes actually observed:

| Scheme | Count | Reality |
|---|---|---|
| `postgres://` / `postgresql://` | 5 | Same protocol, two spellings |
| **`ecto://`** | 1 | Elixir framework scheme — **is** Postgres, not in §3's list |
| **`pg-functions://`** | 4 | Supabase Auth hooks — **not a network protocol at all**; refers to a Postgres function |

So the catalog needs **scheme aliases** (`ecto` → postgres, `postgresql` → postgres) and a
**scheme denylist** (`pg-functions` is not an edge). "Comes free" needs a small curated
table behind it.

## 3.4 Answers to the open items

### O-4 — contract-file matching: **resolved, and the answer is "rules"**

Four attachment strategies tested against six specs across two repos:

| Strategy | Immich | Supabase | Verdict |
|---|---|---|---|
| Directory proximity | ❌ own top-level dir | ❌ in the docs app | **Not viable on either subject** |
| Spec self-identification | ❌ `servers: [{url: "/api"}]` | ⚠️ 1 of 6 (`host: localhost:9999`) | Rare |
| Config reference (build context) | ❌ canonical compose has no `build:` | ❌ none | Not viable |
| Filename ↔ service name | n/a (single spec) | ⚠️ works, with false matches | Heuristic only |

**Recommendation: `rules.yaml` is the primary mechanism for contract attachment in R1**,
with filename-matching and port-matching offered as *suggestions the human confirms*. This
raises the weight of the rules layer relative to how §4 currently frames it — worth
reflecting back into the definition, and it strengthens AC-5.

### O-5 — catalog seeding: **the catalog must go well beyond common images**

Combined inventory: 36 distinct image references. Only ~14 are identifiable from a catalog
of well-known public images. The recurring pattern is decisive:

> **Both subjects ship their database as a vendor-namespaced image**
> (`supabase/postgres`, `ghcr.io/immich-app/postgres`), and Supabase ships MinIO as
> `cgr.dev/chainguard/minio`.

A catalog keyed on official image names identifies neither project's Postgres. The catalog
needs, at minimum: registry-path normalisation, a **substring/heuristic layer with
confidence** (the `…/postgres` tail), image→software aliases (valkey ≡ redis-compatible,
gotrue = the `auth` service), and a way for rules to pin an identification by hand.

Answering O-2's sub-question directly: **yes, entries beyond common infrastructure images
are required.**

### O-3 — storage engine: **premise confirmed, decision unchanged**

Largest subject: 11–15 nodes, ~26 edges, 2 named volumes, 0 networks. Nowhere near any
threshold that would justify a database server. §7's scale argument survives contact with
the complex subject.

## 3.5 What this hands to the stack conversation

Four requirements that are now **testable against a candidate runtime** rather than
matters of taste:

1. **YAML with custom-tag support.** `!override` and `!reset` are Compose extensions
   appearing in both subjects. A loader that cannot register unknown tags fails outright
   (reproduced); one that silently strips them produces a wrong merged model, which is
   worse. *Test: parse `supabase/docker/docker-compose.kong.yml` and honour `!override`
   list-replacement semantics.*
2. **Correct recursive interpolation.** `${API_GW_HTTP_PORT:-${KONG_HTTP_PORT:-8000}}`
   is two levels deep inside a port mapping. Regex will not do it. *Test: resolve that
   string against `.env.example`.*
3. **Multi-format config parsing.** Compose (+ merge semantics), `.env`, Envoy YAML, Kong
   declarative YAML, Prometheus YAML, Swagger 2.0 **and** OpenAPI 3.x, with Terraform/HCL
   present-but-to-be-excluded. Contract parsing is two schemas, not one.
4. **Ordered, stable data structures.** AC-7 requires byte-identical FactSets across five
   runs. Every map iteration in extraction is a determinism risk; the runtime's default
   ordering behaviour is a real selection criterion.

## 3.6 Suggested amendments to `product-definition.md`

Findings that contradict or extend the definition, worth folding back in:

| § | Current text | Evidence |
|---|---|---|
| §0 | *"architecture is declared … not inferable from source"* | **Half-false on Immich** — nodes declared, all 3 edges live in TS source. Worth softening to "declared for nodes; declared for edges in some repos" and naming the consequence. |
| §3 | Protocol "comes free" from URL scheme | `ecto://` and `pg-functions://` say otherwise. Needs a scheme alias table + denylist. |
| §3 | Referenced-evidence example `S3_BUCKET=…` | Sound, but needs the counter-rule: build-metadata URLs (`IMMICH_BUILD_URL`) must not become external systems. |
| §3 / §7 | "network and deployment boundaries" as R1 output | **Both subjects declare zero networks.** Define the no-boundaries-declared behaviour. |
| §4.1 (DSC) | Discovery unspecified on multi-compose repos | Immich has 6 compose files / 4 topologies; Supabase 15. `COMPOSE_FILE` in `.env` is a standards-based resolver worth specifying. |
| §4.3 (registry) | Canonical identity + aliases | **Strongly confirmed.** Add a *precedence* rule: service name vs `container_name` vs alias vs image name. Supabase's `realtime` is the worked example. |
| §8 (output) | — | `model.json` should record which composition was documented (the `COMPOSE_FILE` value) as provenance. |
| §4.9 (RUL) | Rules as refinement | O-4 makes rules **load-bearing** for contract attachment, not just refinement. |

## 3.7 Confidence and limits

- Everything above is from **reading two repositories**. Neither was run; no extraction code
  was written. Claims about what a parser "would" produce are reasoned from the file
  contents, not measured against an implementation.
- Two subjects is a small sample, deliberately bracketed (simplest / most complex). The
  Immich-vs-Supabase split on edge declaration is the main result, and a third subject could
  easily land outside that bracket.
- No hand-drawn reference architecture was built for either subject, so **AC-3 remains
  unaddressed**. That is the next artifact needed, and it is independent of the stack.
- Image and edge counts are from `grep`/`yaml` passes over the pinned revisions and are
  accurate to those revisions only. Both projects move quickly.

---

# Part 4 — Second-pass additions

A verification pass before closing the survey turned up material I had missed. Recorded
separately rather than folded in, so the record shows what a first pass misses.

## 4.1 My own discovery glob failed — the way DSC is predicted to fail

Part 1 reported six Immich compose files, found by globbing `docker-compose*.y*ml` and
`compose*.y*ml`. **That glob missed two real ones:**

```
.devcontainer/server/container-compose-overrides.yml
.devcontainer/mobile/container-compose-overrides.yml
```

Both are genuine Compose files overriding the dev stack. Neither filename matches any
conventional compose pattern. Immich has **eight** compose files, not six.

Content-sniffing instead (top-level `services:` key) finds **ten** files — the eight real
ones plus the two `hwaccel.*.yml` fragments that are not deployable stacks.

| Strategy | Real files found | False positives | Missed |
|---|---|---|---|
| Filename glob | 6 | 0 | **2** |
| Content sniff (`services:`) | 8 | **2** | 0 |
| Both + disambiguation | 8 | 0 | 0 |

**Neither strategy alone is correct.** DSC needs content sniffing to achieve recall, plus a
disambiguation rule to reject fragments — and the fragments are only distinguishable by
semantics (services with no `image` and no `build`, referenced only via `extends`), not by
name or shape. This is a concrete algorithm requirement, and it is worth noting that the
failure reproduced itself on the first try, against the simple subject.

## 4.2 Interpolation is close to full POSIX parameter expansion

The missed devcontainer file contains forms not present anywhere I had looked:

```
:16  ${UPLOAD_LOCATION:-upload-devcontainer-volume}${UPLOAD_LOCATION:+/photos}
:27  ${DB_PASSWORD-postgres}
```

Two new operators, and a new structural case:

| Form | Semantics | Seen in |
|---|---|---|
| `${VAR}` | plain | both subjects |
| `${VAR:-default}` | default if unset **or empty** | both |
| **`${VAR-default}`** | default if unset **only** — empty string is kept | immich devcontainer `:27` |
| **`${VAR:+alt}`** | alt if set, else empty | immich devcontainer `:16` |
| **Concatenation of two expansions in one value** | — | immich devcontainer `:16` |
| Nesting, 2 levels | — | supabase `:91`, `:275` |

`${VAR:-x}` vs `${VAR-x}` differ only by a colon and mean different things. Combined with
nesting and concatenation, this is effectively POSIX parameter expansion, not a
find-and-replace. **This materially raises requirement #2 from §3.5** — it is not "handle
nested defaults," it is "implement the expansion grammar."

## 4.3 Compose custom tags appear in Immich too, more widely than reported

Part 1 found `!reset` once, in `docker-compose.dev.yml`. The devcontainer overrides use
both tags heavily — `env_file: !reset []` four times, plus `environment: !override`
(`.devcontainer/server/container-compose-overrides.yml:11,21,23,25,26`).

Both tags now confirmed in **both subjects**, in multiple files each. The parser
requirement is not a Supabase quirk.

## 4.4 Proxy config: four formats, not two

Part 2 covered Envoy CDS and Kong. Supabase ships two more gateway configs, both carrying
real edges:

```
volumes/proxy/caddy/Caddyfile:5     reverse_proxy api-gw:8000
volumes/proxy/caddy/Caddyfile:14    reverse_proxy studio:3000
volumes/proxy/nginx/supabase-nginx.conf.tpl:1    upstream api_gw_upstream { … }
volumes/proxy/nginx/supabase-nginx.conf.tpl:42   proxy_pass http://studio:3000;
volumes/proxy/nginx/supabase-nginx.conf.tpl:46   proxy_pass http://api_gw_upstream;
```

Two additional problems:

- **A Caddyfile is a bespoke format** — not YAML, not JSON, not INI. It needs a dedicated
  parser or is out of scope. There is no generic-config route to it.
- **nginx uses `upstream` indirection.** `proxy_pass http://api_gw_upstream` does not name a
  service; `api_gw_upstream` is a block defined at `:1` in the same file. Edge resolution is
  **two-hop within the file**, not a hostname lookup.

So the gateway/proxy parser inventory across both subjects is: **Envoy YAML, Kong
declarative YAML, Caddyfile, nginx.conf, Prometheus YAML** — five formats, four of them
schema-specific. Add `volumes/pooler/pooler.exs` (Elixir config) as a sixth format present
but unparsed.

**Coverage is a curation decision with a long tail.** This is the clearest argument in the
survey for §7's "catalog/parser as data, not code" — every new proxy is otherwise a code
change.

## 4.5 Templates are not config

Two files carrying edge evidence are **templates, not valid config**:

- `volumes/api/envoy/lds.template.yaml`
- `volumes/proxy/nginx/supabase-nginx.conf.tpl`

They are rendered at container start by entrypoint scripts (`docker-entrypoint.sh`,
`kong-entrypoint.sh`). A parser fed the template either fails on the placeholder syntax or
extracts placeholders as hostnames. R1 needs a stated position: **skip templates and record
the gap in provenance**, rather than extract from them and silently claim facts that depend
on unrendered variables.

## 4.6 No Kubernetes anywhere — a scope finding

Globbing both repos for `Chart.yaml`, `values*.yaml`, `kustomization*`, `k8s/`, `manifests/`:

- **Immich:** zero. The only IaC is `deployment/modules/cloudflare/*.tf` — Terraform for the
  documentation site's CDN, unrelated to the application (see §1.2).
- **Supabase:** **zero.** No Kubernetes, Helm, Kustomize, or Terraform at all.

§3 lists extraction sources as "compose files, proxy config, manifests, and interface
contracts." **"Manifests" has no coverage in the test-subject set.** Either R1 scopes
honestly to Compose + proxy + contracts, or Kubernetes support gets built with no subject to
validate it against — which contradicts the evidence-first posture the project is built on.

This also bears on the third-subject question (O-2): if k8s is to stay in R1 scope, the
third subject should be a Kubernetes-declared system, and that is a stronger argument for
its selection than self-documentation.

> **Resolved (2026-08-23).** Kubernetes, Helm and Kustomize are **out of R1 scope**, with R2
> the likely home. Recorded in `product-definition.md` §13. R1's extraction sources are
> therefore Compose, `.env`, gateway/proxy config, and interface contracts — which is exactly
> what both test subjects contain, so the scope and the evidence now agree. The third subject
> (O-2) is consequently freed from needing to be a Kubernetes system.

## 4.7 Scan cost — input to NFR-1/NFR-2 and AC-9

Never measured in the first pass; it is a real stack input.

| | Immich | Supabase |
|---|---|---|
| Tracked files | 3,432 | **16,959** |
| Candidate config files (`.yaml/.yml/.json/.toml/.conf/.tf/.hcl/.env`) | 250 | **620** |
| Working tree (excl. `.git`) | 439 MB | **3.1 GB** |

Two consequences:

- Discovery must walk ~17k paths on the complex subject to find ~620 candidates, of which
  roughly a dozen matter. **Cheap traversal and early pruning matter more than parse
  throughput** — the parse volume is tiny; the walk is not.
- §9's NFR targets and AC-9 should be stated against these numbers. "Fast on Supabase"
  now has a denominator.

## 4.8 Revised requirements for the stack conversation

Superseding §3.5:

1. **YAML with custom-tag support** — `!override`, `!reset`, both subjects, multiple files.
   Merge semantics must be honoured, not stripped. *(unchanged, now better evidenced)*
2. **POSIX-style parameter expansion** — `${V}`, `${V:-d}`, `${V-d}`, `${V:+a}`,
   concatenated and nested. *(raised in difficulty)*
3. **Six config formats minimum** — Compose, `.env`, Envoy YAML, Kong YAML, Caddyfile,
   nginx.conf; plus Prometheus YAML, Swagger 2.0, OpenAPI 3.x. Two are non-YAML bespoke
   grammars. **Parsers should be data/plugin-driven, not hardcoded.** *(new)*
4. **Fast filesystem traversal over ~17k paths**, with pruning. *(new)*
5. **Ordered, stable data structures** for AC-7 byte-identical output. *(unchanged)*
6. **A template-detection position** — skip and record, do not extract. *(new)*
