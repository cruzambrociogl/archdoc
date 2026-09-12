# Decision log

Dated, terse. Newest last. A reversed decision is **edited to point at its reversal**, never
deleted — the reasoning is the record.

Full rationale for the larger decisions lives in `docs/stack-decision.md`; this is the index
and the short form.

---

**2026-08-23 — Identity model: the registry (O-1)**
Canonical identities with alias history, resolved from stable IDs at extraction time, rules
able to pin an alias by hand. Renames are alias additions, not delete-plus-add.
The one open question that could not wait for the stack.


**2026-08-24 — Empirical survey before the stack conversation**
Read Immich and Supabase rather than choosing a stack from assumptions. Justified itself
immediately: Immich declares **none** of its three service edges in configuration, and a
naive discovery glob missed two real compose files. Both would have been found late.
→ `docs/survey-test-subjects.md`


**2026-08-24 — Orchestrator manifests out of R1 scope**
Kubernetes, Helm and Kustomize deferred, R2 the likely home. Neither test subject contains a
single k8s or application-level Terraform file, so the capability would be built with nothing
to validate it against — contradicting the evidence-first posture. `EXT-05` is superseded.
Consequence: extraction sources are Compose, `.env`, gateway/proxy config, interface
contracts.


**2026-08-24 — O-4 contract-file matching: the answer is rules**
Four attachment strategies tested against six specs across both subjects. Directory proximity
fails on both. Self-identification works once in six. The build-context chain breaks because
Immich's canonical compose file has no `build:` key. `rules.yaml` is therefore the **primary**
mechanism — which makes the rules layer load-bearing rather than merely refining, and
strengthens AC-5.


**2026-08-24 — Stack: Go for the engine, TypeScript for the web app (O-2)**
Decided on an asymmetry in failure modes: a mishandled Compose `!override` produces a
*silently wrong* model — the draw.io failure this project exists to prevent — while weak
layout produces an ugly diagram, which is visible and which §1 already subordinates to the
model. `compose-go` is the Compose specification's reference implementation, so the
highest-risk input becomes a solved problem rather than one re-derived. Rejected: TypeScript
(re-implementation on the fact path), Rust (same, without the one-language benefit), Python
(weakest distribution).
→ `docs/stack-decision.md` §5


**2026-08-24 — Storage: SQLite via `modernc.org/sqlite` (O-3)**
Pure Go, cgo-free, chosen over `mattn/go-sqlite3` to preserve cross-compilation and the
single-static-binary property that justified Go. Slower; irrelevant at dozens of nodes.
Stays behind §7's driver interface.


**2026-08-24 — Layout: `goccy/go-graphviz`, and the frontend never lays out**
Graphviz as WebAssembly — no cgo, no system install. Supersedes `VIE-03`'s `dagre`/`elkjs`
hint. Layout is computed once in the engine and persisted (`VIE-04`), so the web canvas
renders stored coordinates. That makes `VIE-10`'s fidelity guarantee structural rather than a
matter of keeping two layout engines in agreement.


**2026-08-24 — One repository, packages by seam not by catalog chapter**
Mapping §4's capability groups onto directories was rejected: it breaks on `PRV`, which is a
property every fact carries rather than a stage, and on `FactSet`, which four groups would
share. Six packages, each justified by an architectural seam. Making the network boundary
structural means AC-8 is provable by an import test rather than a hand audit.
→ `docs/stack-decision.md` §3


**2026-08-24 — Types are settled before directories**
Directory names refactor in minutes; `Fact`, `Provenance`, `FactSet`, `Node`, `Edge`, `Model`
and `Version` propagate into every signature and into `model.json`, which is a *committed
deliverable*. O-8's answer gates this — it decides whether a `Fact` carries one source
position or two.


**2026-08-24 — Delivery: R1.a complete, committed for 9 Oct**
Scope tiered by what each piece is *limited* by, not by volume. Implementation-bound work
(parsers, model, renderers, web app) compresses heavily under AI assistance;
verification-bound work (acceptance measurement, the hand-drawn reference) does not;
research-bound work (R1.b semantic clustering) does not compress at all and is **not
committed**. Web app is in scope. R1.c and R1.b static analysis are stretch.
→ `docs/delivery-schedule.md` §2


**2026-08-25 — Context system: three layers, adapted rather than adopted**
A five-layer scheme from another workspace was considered. Its workspace spine, project table
and cross-project TODO tagging solve multi-repo problems this project does not have, and its
machine-local memory layer contradicts its own "repo over memory" principle. Kept: token
discipline, load-on-demand, the vocabulary table, announce-before-writing, and `/checkpoint`.
Added what it lacked — a **pruning rule**, since routing without deletion makes auto-loaded
context grow without bound, which is the failure the token discipline exists to prevent.


**2026-08-25 — No `TODO.md`. `PROGRESS.md` for planned work, `NOTES.md` for the rest**
The capability catalog is already the work breakdown, so a free-form todo list would drift
from it immediately. But planned work is not everything: unplanned ideas, reminders and
questions had no home, and a note that costs effort to file does not get written down.
`.claude/NOTES.md` is the low-friction inbox; `/checkpoint` drains it. "add a TODO" routes
there, so the habit does not need retraining.


**2026-08-25 — Meta files placed by audience, not by type**
`PROGRESS.md` at the root because it answers the question a supervisor asks and hidden folders
are not browsed. `.claude/NOTES.md` hidden because it is a private working inbox.
`docs/decisions.md` moved *out* of `.claude/` — an architecture decision record is a project
artifact, not tooling config. Rule: root is what a visitor should see, `.claude/` is what only
the tooling needs.


**2026-08-25 — Branch discipline by convention, not by GitHub ruleset**
`main` is left unprotected on GitHub. Rulesets are gated behind paid plans for private
repositories, and the alternative — making the repository public purely to unlock a setting —
is a poor reason to publish before there is anything to show. The discipline is written into
`CLAUDE.md` instead, where a session will actually read it: work on `develop`, `main` advances
only by pull request, never force-push a published branch. **Nothing enforces this**, which is
stated explicitly so no session assumes the platform is guarding it. Revisit if the repository
goes public. The PR trail is kept deliberately — it doubles as a dated record of what landed.


**2026-08-25 — Repository name: `archdoc`, lowercase**
Renamed from `ArchDoc`. Everywhere else the name is already lowercase — the binary, the CLI
verbs, `internal/archdoc`, and all running text in `docs/`. Go package names must be lowercase
regardless, so capitals in the repository name made it the outlier rather than the standard.
Concretely, an uppercase module path is escaped as `!arch!doc` in the module cache for the
life of the project. The display name stays **ArchDoc** in the README title and in prose.
Module path: `github.com/cruzambrociogl/archdoc`.


**2026-08-26 — O-7 closed: test subject #3 is Mastodon, not archdoc itself**
Chosen against a criterion rather than by preference. Measured what the existing two subjects
leave untested: **neither Immich nor Supabase declares a single top-level `networks:`**, and
neither references a genuine external managed dependency — Supabase's `SMTP_HOST=supabase-mail`
points at an internal service. So `MDL-09` (network boundaries), `MDL-11` (declared trust
boundaries), `MDL-15`/`MDL-16` (evidence kinds, external-system nodes) and `EXT-09` had
effectively **zero coverage**.

Mastodon fills exactly that gap. Its compose declares two networks with
`internal_network: {internal: true}`; `db` and `redis` sit only on the internal one while
`web`, `streaming` and `sidekiq` span both — a real trust boundary, declared. Its
`.env.production.sample` carries `S3_BUCKET=files.example.com`, `SMTP_SERVER` and `ES_HOST`,
which is **§3's worked example for referenced evidence, occurring naturally**. Five services
places it between Immich's four and Supabase's eleven.

Pinned at `47ac677a9b9392833d7cecab8fccbce34c738b83`. Candidates rejected on the same measure:
Outline, Paperless-ngx and Sentry self-hosted all declare zero networks.

*Replaces* the original intent of documenting archdoc itself, which §3's own vantage-point
argument predicted would be weak — a CLI tool with no declared services yields roughly one box.


**2026-08-26 — O-8 closed: provenance does NOT survive the Compose merge. Extraction is two passes.**
Tested against the two hardest files the survey found, at the pinned revisions:
`supabase/docker/docker-compose.kong.yml` (`!override` across a merge) and
`immich/.devcontainer/server/container-compose-overrides.yml` (`!reset`, plus two expansions
concatenated in one value).

*What works.* `compose-go` v2.14.0 gets the semantics right. `!override` replaced api-gw's
ports rather than appending them (8000, 8443 — exactly two), and swapped the image to Kong.
`!reset []` emptied `profiles`. The concatenated expansion
`${UPLOAD_LOCATION:-default}${UPLOAD_LOCATION:+/photos}` resolved correctly both ways —
`upload-devcontainer-volume` when unset, `/mnt/photos/photos` when set. The library earns its
place; this was the argument the stack decision rested on and it holds.

*What does not.* `types.ServiceConfig` has **100 fields and none carries a source position** —
no Line, Column, or File. `types.Project` exposes `ComposeFiles`, but that is which files
contributed to the project, not where any single fact came from. Nowhere near PRV-01.

*The fallback is proven, not assumed.* A raw `gopkg.in/yaml.v3` parse recovers positions and
they are addressable by key path — `services.api-gw.image` resolves to line 72 in the base
file and line 14 in the overlay. **The last file in which a key appears is the one that won
the merge**, which is exactly the provenance PRV-01 needs.

**Consequence:** extraction runs `compose-go` for semantics and `yaml.v3` for positions,
reconciled by key path. More work, no more risk, and the same would be required in any
language. It also settles the core type: a **`Fact` carries one source position** — the file
that won — not two.


**2026-08-27 — Discovery: sniff for recall, reject fragments for precision**
The walking skeleton forced the discovery rule to be settled. A filename glob is not enough —
Immich keeps two real Compose files at `.devcontainer/server/container-compose-overrides.yml`,
matching no conventional pattern, and the glob written for the survey missed both. But
content-sniffing alone admits `docker/hwaccel.ml.yml`, whose services are named `cpu`, `armnn`
and `rknn` and are hardware snippets rather than anything deployable.

**Rule: sniff every YAML for a top-level `services:` key, then reject any file where no service
declares an image or a build.** Recall from the sniff, precision from the fragment test. On
Immich that yields 10 candidates, 7 deployable, 3 fragments.

Selection between deployables prefers the declared answer over convention: `COMPOSE_FILE` in a
neighbouring `.env` or `.env.example` is a native Compose variable holding a colon-separated
list, base file first — Supabase's own `run.sh` maintains it, so reading it is standards-based
rather than a guess. Only when absent does convention decide: unsuffixed name beats suffixed,
shallower path beats deeper, and `.devcontainer`/`e2e`/`test` paths are penalised.

`--explain` prints every candidate and the reason for its verdict. A tool whose premise is
traceability cannot answer "which file did you use?" with "trust me".


**2026-09-06 — C4: transparent intermediaries stay in the model, leave the container view**
Compose describes *deployment*; a C4 container diagram must not show deployment concepts, and
Brown is explicit that gateways are *"typically wrong"* on one — most exist only in production.
The live cases are Supabase's `api-gw` (Envoy/Kong) and `supavisor`, a connection pooler.

Both are separately deployable applications, so the literal definition would admit them. They
are excluded anyway because they are **transparent**: applications talk *through* them without
depending on them semantically. Drawing them turns `auth → db` into `auth → supavisor → db`,
which hides the real coupling behind an intermediary — Brown's own complaint about modelling a
message bus as one box: *"everything looks like a hub and spoke architecture... we're missing
the couplings between the individual services."*

**Rule: a Compose service becomes a C4 container when it is an application or a data store.
Proxies, gateways, poolers and sidecars remain in the model with `kind` recorded, and the
container view omits them — annotating the edge they mediate, `auth → db (via supavisor)`.**
That annotation is Brown's suggested treatment for a queue between two services. Nothing is
lost: the intermediary is still in `model.json`, and `rules.yaml` can force it back in.

Consequence: Supabase renders 9–10 containers rather than 11, and its real service-to-service
edges become visible for the first time. `Docker` never appears as a technology on a container
diagram — the technology is what runs *inside*, which is what the catalog is for.


**2026-09-06 — Actors are derived from published ports, not invented**
C4's context level needs people and external callers. Nothing in a Compose file says a person
exists, and §3 forbids inferred nodes in R1 — so an actor guessed by the model would be
inadmissible.

**A published port is declared evidence that something outside reaches in.** `immich-server`
exposes `2283:2283` at `docker-compose.yml:26`; Supabase's `api-gw` exposes `8000`. That is a
fact at a line, not a guess. What it does not say is *who* — a person, a mobile client, another
service.

So the actor is built in three steps, each staying inside the evidence rule: extraction emits a
generic **External client** node carrying the port's provenance; the semantic layer **names** it
more usefully (*"Photographer"*, *"Mobile app"*) — labelling a node that already exists rather
than inventing one; `rules.yaml` corrects it when the guess is wrong.

Mastodon is the case that proves the rule is doing real work: `web` binds `127.0.0.1:3000:3000`,
localhost only. Honestly read, that declares *"expects a reverse proxy in front"* rather than
*"a person talks to this directly"* — a distinction only available by reading what the file
actually says.

**2026-09-06 — Adviser feedback: bring visible output forward, add a second workstream**
Five changes follow from a review conversation.

*Scope may grow, but not this week.* The adviser confirmed AI-assisted implementation means
scope can expand. Correct for the project; wrong for the next five days, where one finished
visible artifact beats three partial ones.

*Tangible output moves to the front.* A terminal table does not communicate what this project
is. The cheapest path to a picture is **Mermaid**: text output, no layout engine, and GitHub
renders it natively inside markdown. SVG, layout and stored coordinates stay where they were.
The Friday target is a generated markdown file — diagram plus provenance table — committed into
each test subject as a worked example.

*The report and the presentation are graded deliverables*, and had no time allocated anywhere.
They now have a workstream and an entry point, `brief.md`.

*Two people, two tracks.* The split follows the package seams and meets at the FactSet — which
§11 already defines as the contract between the deterministic and probabilistic halves. Using
it as the contract between two people costs nothing, and lets the drawing side build against a
fixture before real data exists.

*The LLM waits.* The adviser asked for it; it is still deferred past Friday. The stronger
demonstration is the diagram produced with **no model involved** — that is AC-2, the central
claim — followed by the labelling layer afterwards. A half-wired API call would weaken the
argument it was meant to strengthen.

**2026-09-06 — One entry point per workstream, not one per audience**
`brief.md` is to the design and report work what `CLAUDE.md` is to the code: a map, not a
library. One file rather than separate brand and report documents, because two files means two
things to keep current and two things to read.

It carries only **identity** (name, tone, audience, metaphors — which rarely change) and
**pointers** (where each fact already lives). No counts, dates or measured results, so there is
never a second copy to drift. Rot resistance comes from the structure, not from discipline.

The visual metaphors are drawn from the subject rather than from the category: **cartography**,
because C4's own framing is maps and zoom, and **citation**, because every element carries the
line that proves it. Both beat the generic architecture imagery of blueprints and gears.


**2026-09-06 — Derivation gets its own package**
`internal/extract` reads files; `internal/render` draws. Turning facts into a graph is neither,
and it is the seam the two workstreams meet at, so it became `internal/model` rather than
hiding inside one side of the boundary it defines.

The alternative was putting it in `internal/archdoc` beside the types. Rejected: that package
is the vocabulary both tracks import, and giving it behaviour would make every change to
derivation a change to the shared contract.

**2026-09-06 — The FactSet carries parsed endpoints, never the raw environment**
Compose environments hold passwords, JWT secrets and API keys — all four sit in plain sight in
two of the three subjects — and the FactSet is written to disk as `model.json`.

So extraction reads the environment and keeps only what parses as a network location. Nothing
downstream has to remember to redact, because there is nothing left to redact; `url.Hostname()`
drops the credential even when the URL that named the host carried one. A test asserts the
fixture's password reaches no output.

Two shapes are recognised: a value containing a URL, and a value in a variable whose name says
it holds a host. Nothing is guessed from a value alone — that is what keeps `MDL-04`
deterministic rather than a heuristic.

**2026-09-06 — Edges record whether traffic flows, and the bridge requires it**
*Refined the same day — see "Only reachability evidence may originate a bridge" below. Traffic
at both hops turned out to be necessary and not sufficient.*
Found by reading real output rather than by testing. Supabase's container diagram claimed
*"User reaches studio"* and *"functions connects to studio"*. Both were bridged through
`api-gw`, and both hops came from `depends_on`.

`depends_on` is start-up order, not routing. A gateway that waits for an admin console to be
healthy does not thereby route users to it. Edges now carry `Traffic`, set by endpoints and
published ports and not by `depends_on`, and a bridge across an excluded proxy requires it at
both hops. Two false arrows disappeared.

The same flag decides which label survives a merge: where `depends_on` and a configured URL
describe one pair, *"connects to postgres"* beats *"depends on"* — the stronger evidence names
the relationship.

Where a route is now missing, it is declared in the gateway's own configuration. That is
`MDL-03`, a source archdoc does not read yet, and the generated document says so.

**2026-09-06 — Mermaid flowchart, not Mermaid's C4 syntax**
Mermaid does provide `C4Context` and `C4Container`. They are still marked experimental, and
GitHub's renderer lags upstream — §8 already flagged the risk.

A diagram that does not draw is the one failure a reader cannot work around, so the C4
vocabulary lives in the labels, where it is visible and cannot break. `[Container: PostgreSQL
14]`, `[External System]`, `[Person]`. Revisit when the syntax stabilises; the model is
unaffected either way, which is the whole point of one model and many views.

**2026-09-06 — The image catalog matches exact names, never substrings**
`supabase/postgres-meta` is an application that manages a database, and `darthsim/imgproxy`
transforms images rather than proxying them. Any rule loose enough to catch `postgres` inside
the first also misfiles the second.

So: strip registry, namespace, tag and digest, then match the remaining name exactly. An image
not in the table is an **application with no technology** — empty says *"not known from
configuration"*, which is honest and is precisely the case `MDL-08` hands to the semantic layer.
Inventing a stack for an unknown image would put a guess on a diagram that claims not to guess.

**2026-09-06 — Fixtures are written by hand, not trimmed from the subjects**
Closes the open note asking what goes in `testdata/`. Trimmed copies of real compose files
looked cheaper and are worse: they carry irrelevant detail, they go stale against pinned
revisions, and a reader cannot tell which line the test is actually about.

`testdata/endpoints/` is instead written so every service exercises one rule, and every trap in
it was found in a subject first — a unix socket in `POSTGRES_HOST`, a `0.0.0.0` bind address, a
flag named `SELF_HOST`, a `localhost` URL naming the reader's own machine. The subjects stay
what they are: evidence, read at pinned revisions, never vendored.

**2026-09-06 — Friday shows a live preview, and the hand-drawn reference leaves the calendar**
The sprint 1 gate said *"rendering on GitHub"*. Rendering on GitHub was never the requirement —
the requirement is that the output reads with archdoc absent and with no build step, and a
markdown preview in the editor demonstrates that as well as GitHub does. It also sidesteps the
awkwardness that the worked examples live in clones of other people's repositories.

Separately, the hand-drawn Immich reference architecture comes off the schedule at Cruz's
request. It stays in `PROGRESS.md` under manual work, with an owner and no date, because it is
the answer key for AC-3 — dropping it from the plan entirely would quietly drop the criterion.

**2026-09-06 — Gateway routes are found by following bind mounts, not by guessing paths**
archdoc does not look for `nginx.conf` in the places nginx configs usually live. It reads the
compose file's `volumes:`, and whatever is mounted into a service is, by the repository's own
statement, that service's configuration. `docker-compose.yml:93` mounting `cds.yaml` into
`api-gw` is the citation for *why* that file was read at all.

Which services are gateways is then decided by what the mounted files contain rather than by
image name — the same sniff-for-recall, reject-precisely rule discovery already uses. A gateway
running an image no catalog knows is still a gateway, and a `.sql` seed file mounted beside the
routing table yields nothing.

Measured on Supabase: 7 routes, each citing a line in `cds.yaml`, which is the number the survey
predicted and the schedule row's gate.

**2026-09-06 — Hostnames resolve through container_name and network aliases**
One of Supabase's seven routes points at `realtime-dev.supabase-realtime`, which is not a
service key. It is the `realtime` service's `container_name`, declared in the compose file with
a comment explaining why.

Without resolution that route creates an external system, and a real container appears twice —
once as itself and once as a stranger the repository seems to depend on. Services therefore
carry their aliases, and every host lookup goes through them. This applies to environment
endpoints too, not only routes.

**2026-09-06 — Only reachability evidence may originate a bridge**
A refinement of the traffic rule, and again found by reading output rather than by a test.

With routes extracted, `functions` set `SUPABASE_URL=http://api-gw:8000` and the container view
drew it reaching all **seven** services behind the gateway. Both hops carried traffic, so the
earlier rule allowed it — but a gateway routes by *path*, and a bridge cannot see paths. One
call became a fan-out.

So the first hop must be evidence of **reachability**, not of a specific call. A published port
is reachability: "anything outside can reach whatever this gateway routes to" is what a public
entry point means, and it is true. An environment URL is a call to one endpoint, and which one
the configuration does not say. In practice this means only actors originate bridges.

Resolving the rest needs route paths matched against the caller's URL, which is further into
MDL-03 than R1.a goes. Supabase's user now reaches 8 services; `functions` keeps only the calls
it declares directly.

**2026-09-06 — Network membership is a fact worth drawing; sample env files are not**
Two extraction decisions taken together, because measuring them together is what separated
them.

*Networks are drawn.* Compose network membership is the one reachability claim a configuration
file states rather than implies — two services sharing no network cannot reach each other — and
`internal: true` says a network has no route out at all. Both are declared, so `MDL-11` records
a trust boundary without inferring anything. Membership often nests, so the boundary nests:
Mastodon shows `db` and `redis` inside `internal_network` and outside `external_network`.

Where two networks share members without one containing the other, **no boundary is drawn**.
Nested boxes cannot express that, and flattening would erase the distinction the file drew. A
wrong grouping claims more than no grouping.

*Sample env files are not read as fact.* `EXT-07` reads what `env_file` names, and only files
that exist. It deliberately ignores `.env.example` and `.env.production.sample`. Mastodon's
sample sets `REDIS_HOST=localhost` and `DB_HOST=/var/run/postgresql` — correct defaults for a
non-container deployment, wrong for the stack its compose file describes — and `S3_ALIAS_HOST`
is the placeholder `files.example.com`. Reading them would put fiction on the diagram.

Those files are still used for **interpolation defaults**. Filling `${VAR}` with the value a
repository suggests is a weaker claim than asserting the value is a fact about the system.

The measured result is that `EXT-07` is worth nothing on all three subjects: none ships a dotenv
file that exists. Recorded rather than hidden — the capability is right and the evidence is
absent, which is a finding about the subjects. Mastodon's real external dependencies are
declared as capability toggles (`S3_ENABLED=true`) rather than as hosts, which is `EXT-09`'s
shape, not `EXT-07`'s.

**2026-09-10 — The history database is a cache; git is the archive**
`.archdoc/history.db` records every model archdoc has produced for a repository, and archdoc
writes a `.gitignore` beside it so the file stays local.

The durable record is `model.json`, committed next to the documentation at every commit. The
model as of any revision is therefore already stored by the thing designed for storing
revisions, and losing the database costs speed and nothing else — regenerating rebuilds it. A
binary file in git conflicts on every parallel run and diffs as noise, which is a real cost for
no gain.

Two consequences worth stating. A diff between two commits does not need the database: check out
each revision and the committed `model.json` is right there. And `.archdoc/model.json` is
deliberately *not* ignored — it is the reviewable record, and a reviewer should see it change.

**2026-09-10 — A run that changes nothing records nothing**
AC-7 guarantees byte-identical output across runs, so recording a version per run would fill
history with entries differing only in their timestamp, and a diff between two of them would be
empty.

`Save` fingerprints the model and compares it to the latest version. Same fingerprint, no new
version. The count in `archdoc history` is therefore the number of times the architecture
actually moved, not the number of times somebody ran the tool — which is the number a reader
cares about.

The same model at a *different commit* is also not a change. A commit that touched no
architecture has no place in an architectural history.

**2026-09-10 — `internal/store` has no interface yet**
The package map says "storage interface + SQLite driver". There is one driver and the tests run
against SQLite in memory, so an interface would add a layer with nothing on the other side of
it. Recorded so the deviation is deliberate rather than forgotten: the moment a second backing
store is real, or a test needs a fake, the interface earns itself.

**2026-09-11 — The semantic layer is fenced by the type system, not by the prompt**
Three things keep the model from changing what exists, and none is an instruction it could
ignore: it is sent structure only (names, kinds, relationships — no paths, lines or file
contents); it answers in a strict JSON schema whose only verbs are labelling verbs; and every
operation passes through the same validator as a rules.yaml correction, where
`OpKind.AllowedFrom` rejects anything structural. Validation failures go back to the model
with the reasons, three times at most (VAL-07), then the run fails with nothing applied (VAL-08).

It is opt-in (`--label`). Without it no request is made and the diagram is complete — AC-2 by
construction. Rules are compiled against the model as extracted and applied *after* labelling,
so a person's correction always overrides a model suggestion, and a model renaming `api` to
"API" cannot break a rule that matches `name: api`.

Defaults: `claude-opus-5`, with server-side fallbacks in `"default"` mode so a safety-classifier
refusal is re-run on Anthropic's recommended fallback rather than returned empty. A model
value's provenance names the model but not the request id, so a relabelled run that changes
no architecture does not create a new version in history.

**2026-09-11 — First live runs: provenance for labels is separate, and labels may not assert protocol**
Three live runs on Supabase. Structure held on every one: versions 1–4 in history have the same
13 elements, kinds and 23 relationships — AC-2 on real output, not a fixture.

Two defects surfaced only because the model produced real text, and both are now fenced by code
rather than by prompt:

*A relabelled arrow looked like evidence for the arrow.* SetEdgeLabel appended the model's
citation to the edge's own provenance, and a citation with no file sorts first, so every
relationship read "Declared at: model: claude-opus-5". Edges now carry `LabelProv` and nodes
`NameProv`, beside `DescProv` and `TechProv`. `Prov` means one thing only: the evidence that the
element or relationship exists.

*A label asserted a fact.* Seven relationships were labelled "over HTTPS" when the configuration
publishes plain port 8000. The validator now rejects a model-written label naming a protocol the
edge's recorded technology does not state, which sends the reason back through VAL-07. Rules are
exempt. Descriptions are not checked — they are interpretation by definition, and marked as such.

~~Known and accepted: the seven user relationships, all bridged from one published port through
the gateway, get one generic label. Their evidence is identical, so a label distinguishing them
would be claiming more than the configuration says.~~ *Wrong — corrected 12 Sep, below.*

**2026-09-12 — A reconnected arrow takes the route's label, when one was written**
Corrects the "known and accepted" note in the entry above. The seven user arrows' evidence was
never identical: each crosses the gateway on its own route, with its own line in `cds.yaml`, and
the model had labelled each route specifically ("forwards signup, login and token requests to").
Those labels were hidden with the gateway, because bridging kept the first hop's label.

Bridging now takes the second hop's label when a person or the model wrote it — that hop is the
one that says what reaches the target — and keeps the first hop's extracted label otherwise, since
with no model "reaches" reads better on a user's arrow than "routes to". The label's citation
moves with it; the arrow's evidence is still both hops.

**2026-09-12 — archdoc draws its own C4 diagram from stored coordinates**
Graphviz (`goccy/go-graphviz`, WebAssembly — no cgo, no system install) places everything; archdoc
sizes every box for the exact text it will draw, then draws the SVG itself. Graphviz's own SVG was
rejected: the C4 conventions — name, type, technology, responsibility, dashed external systems,
italic model text — are archdoc's to apply, and a future web canvas must draw the identical picture
from the same stored coordinates (VIE-10). The markdown shows the SVG and keeps the Mermaid source
in a collapsed block, as §8 asks; if layout fails, the documents fall back to Mermaid alone.

Positions are stored with each version and reused when the architecture is unchanged, so an
identical run cannot reshuffle the picture. **Keeping boxes in place across a *changed*
architecture is out of scope:** Graphviz lays out from scratch, and position pinning is closer to
research than to a task. A stored layout carries the engine's version and is reused only by the
same engine — found the first day, when a fix to boundary labels did not appear on Immich or
Mastodon because their unchanged architectures reused layouts stored by the old code.

Cost: the binary grows from 24 MB to 31 MB. That is the price of layout without a system Graphviz.

**2026-09-12 — The run log records the wire, not the prompt**
SUR-14 says print exactly what left the machine; AC-8 says the run log accounts for every byte.
Both are met by recording the HTTP request bodies as they go onto the wire — a middleware on the
SDK client, inside `internal/semantic`, the one package allowed to make calls. A log built from
the prompt string would account for the prompt and miss whatever the client wrapped around it.

The test that gates AC-8 points the real SDK at a local fake of the Messages API and checks
that the recorder holds exactly the bytes the server received — and that no path, line number,
provenance or API key is among them.

Two rules. A run is logged **whether or not it succeeded**: a request that left the machine is
in the log, and a failed run is the one someone goes looking for. And cost is reported only for
a model whose price archdoc knows; an unknown model reports its tokens and no figure, because
NFR-6 asks for actual cost and a guessed one is worse than none. `archdoc runs --show N` prints
the payload unformatted — a prettier version would no longer be the thing that was sent.

**2026-09-12 — The web app is thin, local, and draws the committed SVG**
`archdoc serve` reads what `generate` stored and computes nothing of its own; the canvas is the
SVG from `render.SVG` over the stored layout, so the browser and the repository show one drawing
(VIE-10). React + Vite, built into `web/dist` and embedded; a `dev` build tag proxies to Vite.
It binds to 127.0.0.1 and refuses any Host header that is not this machine — a page elsewhere can
make a browser call localhost, and the Host check is what stops it reading the model through a
rebound DNS name. Human sections are refused by the API, not read: OUT-03 holds in the app too.

**2026-09-12 — Completeness from size and mtime, not content hashes**
OUT-09 needs to know whether a human section is still archdoc's stub; OUT-03 forbids reading it.
Hashing was the obvious answer and was rejected — hashing a file is reading it. `generate` records
each stub's size and modification time in `.archdoc/sections.json`; a later change to either means
someone wrote. "May be stale" compares the file's mtime with the latest version, an approximation
of OUT-10, which asks for the section's last *commit*. Cost: a `touch` reads as written.

**2026-09-12 — Version diff by element ID, wording kept apart from structure**
The timeline's diff identifies elements by ID, so a rename is one changed field, not a removal
plus an addition (MEM-06). Appeared/disappeared is reported apart from reworded: the first is
news about the system, the second about the documentation. This compares recorded versions; the
diff between two git commits (AC-6) remains Sprint 3 work.
