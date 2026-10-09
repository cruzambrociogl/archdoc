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

**2026-09-14 — Scope reopened: architecture stays the product, the evidence widens to code**
R1.a's output showed its ceiling on Immich: five boxes from Compose, while the code holds 46
controllers, 64 tables, and the one edge the diagram lacks (`server/src/dtos/config.dto.ts:624`).
Decided: architecture documentation remains the product, in arc42's broad sense — runtime,
domain, data, not only boxes. Configuration-only extraction is the limit to remove, not the
identity to change. The four interfaces stay: CLI, Markdown, storage, web app. Horizon for the
next phase: end of 2026. Everything else — lenses, explanations, D-1 to D-17 — is a proposal in
`docs/vision.md`, not taken; the product definition wins until each is recorded here (O-11).

**2026-10-02 — Positioning: the alternative, not the rival**
Google Code Wiki (Gemini, hosted, free, any language) ships most of what `docs/vision.md` §2.3–2.8
proposes: code reading, sequence diagrams, module walkthroughs, chat over the result, refreshed per
commit. Decided not to compete on breadth and not to retreat either: offer the same usefulness on a
foundation they cannot — extraction that can be checked, determinism, local and offline operation,
documentation the user owns in their own repository, and `rules.yaml` corrections that survive
regeneration. Parity list, build order and non-goals in §2.2; canonical wording in `docs/brief.md`.
The line that holds: every feature we match must have a verifiable counterpart, or we have built a
worse Code Wiki.

**2026-10-02 — Delivery 11 Dec; 9 Oct is a progress review; evidence before building**
Four planning decisions. Review 2 was held on 25 Sep and went well. **9 Oct is a progress review,
not the delivery** — so the report and slides leave October, and the 2 Oct code freeze is void; the
freeze moves to 4 Dec and delivery to **11 Dec 2026**. The phase after the review is ordered
**evidence first**: the comprehension pilot and the comparison against Code Wiki and DeepWiki run
before any parser work, because their result can redirect it. The audience to optimise for is
**technical readers losing control of code an AI wrote** — which deprioritises the vibe-coder-facing
work (plain language, guided tours, ask-instead-of-read) and promotes components, data, flows, the
change lens and a publishable site. Candidate features are scored in `docs/feature-inventory.md`,
to be marked in or out in the planning week (13–17 Oct); the calendar is built from that list.

**2026-10-02 — Deployment view enriched, coverage published, output made proportionate, site emitted**
Four deliverable-level changes, all from facts already extracted and then discarded. §7 now carries
what Compose actually states — the image each container runs, published ports, networks, and what the
repository mounts in — because a compose file *is* a deployment descriptor and §5 is the awkward
projection of it. A new `coverage.generated.md` publishes what was read (including the candidate
files discovery passed over, and why), how much is known, every validator gap grouped by rule, and
the fixed limits of reading configuration. No competitor in this space publishes what it missed; a
reader who cannot see the edge of the map has no way to know they are standing at it. Output is now
proportionate: a one-service repository with no networks, ports or relationships gets six sections
instead of twelve, and the index names the omissions so they read as a decision. `--site` writes
`mkdocs.yml` into the documentation directory (not the repository root, which a project may own),
completing layer 3 of §8 — the answer to "share it with my team" that is not a hosted service.

**2026-10-04 — Run sizing measured, and the local-model shortlist**
Measured rather than guessed: a whole-repository labelling request for Mastodon is ~3,579 chars
(≈1,000 tokens), ~83 tokens per element, matching the one live Supabase run at ~5 KB. Full-run
estimates are cents for ordinary repositories and about a dollar for an Immich-scale one; thinking,
billed as output, dominates. **Time is the binding constraint, not money** — NFR-2's three minutes
breaks long before the budget does, which is the strongest argument for interpretation memory (D-1)
landing in the first build block. Numbers and levers in `docs/vision.md` §2.9.
For the eventual local path (NFR-4): cloud stays the default, local is opt-in. Shortlist for a 24 GB
machine — **Granite 4.2 8B** first, because it is tuned for tool calls and strict JSON, which is
exactly the shape archdoc asks for; **Qwen3 14B** as the quality baseline; a 30B mixture-of-experts
only if quality demands it. Dense 32B models are rejected: they run but make the machine unusable.
What makes a small model viable is schema-constrained decoding (Ollama can constrain generation to
the JSON schema we already send) plus the validator, which refuses anything off-policy regardless of
which model produced it. Verify the shortlist when implementing — the local field moves monthly.

**2026-10-04 — The surface: one app, two modes, diagrams drawn by the app, built early**
The web app is redesigned as archdoc's main surface, specified in `docs/surface-spec.md`. Four
decisions (S-1 to S-4). **The published app is the site:** `archdoc export --site` builds the same app
with one version's data baked in as JSON; `mkdocs.yml` stays as the plain fallback. The site reads the
**model**, not the Markdown — both are projections of one model, and the Markdown flattens citations,
truth states and diagrams. Human-owned sections are linked on the repository host, never copied in,
because copying is reading (hard rule 2). **The app draws its diagrams** from a scene the engine emits
— positions included, so layout stays deterministic and stored — and the committed SVG and Mermaid
become exports. This reverses the "same SVG, byte for byte" property of the current canvas: the
guarantee is the elements and their layout, not the bytes. **Citations open the line**: the editor
locally, a permalink at the commit when published. **The shell and today's screens are built before
Build 1**, moving F-25 and F-22 forward. Then S-5 to S-7, the same day: **React Flow** draws the graph
diagrams, our own component the sequences, layout stays in the engine. **People can drag a diagram into
shape and keep it** — `.archdoc/layout.yaml`, committed and re-applied every run like `rules.yaml` — and
**save named views** in `.archdoc/views.yaml`. Position is presentation, not fact, so neither can
falsify anything; they are the first files the app writes, so they carry ANS-04/05's write safety.
Interpretations are committed beside `model.json`, so a clone or CI shows everything without
regenerating and without a key (`surface-spec.md` §3.2).

**2026-10-04 — What the app may control (C-1 to C-5)**
The published site controls nothing; the live app is a second front end over the same engine, and
every action it offers is a CLI command with the same effect. It **runs `scan` and `generate`**,
with the model only after a confirmation showing egress, model and estimated cost (C-1). It **writes
`rules.yaml`**, append-only after a preview — archdoc's configuration, not a documentation section
(C-2). **Settings stay read-only** for now (C-3). It **never acts on git**: it shows archdoc's
uncommitted files and their diff, and the command to copy (C-4). Control from the published site is
**deferred**, not refused (C-5). Because the app now acts, every action is a `POST` requiring the
app's own Origin and a per-session token: the Host check stops cross-site reads, not cross-site form
posts. `docs/surface-spec.md` §11.

**2026-10-05 — React Flow confirmed at Immich scale (S-5)**
Measured, not assumed: a synthetic repository of 300 Compose services across 8 networks went
through the real pipeline (301 elements, 571 relationships, Graphviz layout in 12 s at generate
time), and the app drew the stored scene with React Flow in 177 ms, every route included. Panning,
wheel-zooming and panning zoomed in held 60 fps (95th-percentile frame 16.7 ms) in headless Chrome
without a GPU, with no errors. The renderer is not the limit at this size; legibility is — fitting
300 boxes needs 10% zoom — which is what D-11's capped overviews and focus views are for.

**2026-10-05 — The published site is served, not opened from disk**
surface-spec §3 said the published site could be "a folder opened from disk". It cannot, cheaply:
browsers refuse to load the app's ES-module scripts and its fonts from a file:// page, not only its
data, so supporting it would mean building the whole app — the live one included — as classic
scripts, and still losing the fonts. The site exists to be hosted (GitHub Pages, any static host,
built by CI), and previewing it locally is one command (`python3 -m http.server 8080 -d …`), which
export prints. The spec is corrected. A single self-contained HTML export (vision D-10, F-23) remains
the answer for a file to attach or open offline, if it is ever wanted.

**2026-10-05 — Parsing code: gotreesitter, pure Go (F-01, D-4)**
The binary must stay one static file with no cgo (NFR-10), and every Go tree-sitter binding but one
needs cgo. gotreesitter is a pure-Go tree-sitter runtime that loads the upstream grammars' parse
tables. Its `grammars` package embeds all 206 grammars and takes over ten minutes to compile, so
`internal/code` imports one package per grammar instead (`grammars/typescript`, `tsx`, `javascript`,
`python`, `svelte`): a plain `go build` in seconds, no build tags, and +11 MB on the binary (35 → 46
MB). Measured before choosing:
Immich's server, 543 TypeScript files (3.4 MB), parsed in 1.0 s, finding 47 controllers and 301
routes each at its line; 7 files carry one known grammar gap (a tagged template with a type argument,
Kysely's ``sql<T>`…` ``), recovered locally inside SQL bodies archdoc never reads. Python: Immich's
machine-learning service (31 files) and the FastAPI template's backend (40 files) parse with no errors,
every route found. The alternative — an importable fork of Microsoft's typescript-go — would add type
information but only for TypeScript, from an unofficial fork of internal packages; it stays the
candidate for D-4's later "resolved by type" upgrade. Syntax is the baseline, and provenance records
how a link was resolved. A second gap, found following flows: a call with type arguments after `await` —
`await this.predict<T>(a, b)` — is read as two comparisons; `internal/code` recognises that shape as
the call it is (`misreadGenericCall`), since a comparison is never followed by an argument list.

**2026-10-06 — What leaves the machine about code: names, not text (D-8, F-35)**
An explanation request (`--explain`) carries a component's facts as a numbered list, and the list is
names only: the component's, its files' paths, the components it uses and is used by with import
counts, its routes by method, path and handler, its tables and their column names. It carries no line
of code, no string the code holds — not the route summaries its decorators state, not docstrings,
though both would help the prose — and no provenance. Strings in code are where people write
things they did not mean to publish; names are what a reader of the repository's tree already sees.
`internal/semantic/explain_test.go` holds the line. Revisit only with the person who owns the data.
*Amended 8 Oct, by Cruz:* a route's summary — what a decorator's `summary` or a docstring's first line
says the route does — is now sent with the route. It is the code describing itself, and the prose is
much the poorer without it. Everything else above stands: no other string, no code, no provenance.
The fact wording changed, so remembered answers are asked for again on the next `--explain`.

**2026-10-08 — The model stays `claude-opus-5`**
Asked whether to move the default to a newer Opus: no. Labels and explanations were run and priced
on `claude-opus-5`; `internal/semantic` keeps it as the default and the only priced model.
*Superseded the same day — see "Two models, by what the job needs" below.*

**2026-10-08 — A component is a responsibility, not a folder (D-2, F-10)**
Looking at Immich in the explorer, Cruz saw that the component level showed how the code is filed —
`controllers`, `services`, `repositories` — not what C4 means by a component. It had been built on
directories for speed, against D-2's "conventions first". Now: where files are named by what they
are for and what they do (`album.controller`, `album.service`, `album.repository`, `album.table`),
a component is a name that spans those roles. Roles are found by counting suffixes, not from a
list; a longer name joins the feature it extends; a lone file that does something in a main role is
a component of its own; the rest stays with its folder. It applies only where the convention
carries the application (three features, three files in ten); otherwise folders remain the
components. The folders stay as a second view, "by folder" — a `module` kind, a `structure:` view —
because how the code is filed is also true, and is the compact picture. Everything is still by
name, so every membership can be read off a file name; nothing is grouped by a model. Immich's
server: 75 components where there were 14 folders.

**2026-10-08 — External systems: a configured URL, or a client library — never a link**
The context view of Immich was a user and a box. External systems now come from two kinds of
evidence in the code. A URL is one when the code calls it, or when a configuration key holds it
whole (`versionCheck: { url: '…' }`) outside a template file; a URL in a sentence, a link, an email
template, or one the code only starts (`` `https://github.com/…/${version}` ``) draws nothing.
And a well-known client library names the system it talks to — `nodemailer` an SMTP server,
`openid-client` an identity provider, `huggingface_hub` the Hub — by a lookup table, cited as a
lookup the way an image's technology is: the import is read, the meaning is looked up, and the two
are told apart. Immich: seven external systems, none of them a link.

**2026-10-08 — Two models, by what the job needs; trivial components are not asked about**
The first full `--explain` run cost $1.52 for 43 components, and Immich now has 122. Measured on
five components after the prompt was fixed (no retries): 1,220 tokens in and 337 out each, $0.0145
on Claude Opus 5 — two thirds of it output. Decided with Cruz: `--explain` moves to
`claude-sonnet-5-5` ($2 / $10 per million against $5 / $25) at low effort, since turning a fact list
into three cited sentences does not need the largest model and there is a request per component;
`--label`, one request for the whole system, moves to `claude-opus-5-5`, newer and a fifth cheaper
than Opus 5. Effort is stated on every request, because its default differs by model and reasoning
is billed as output. A component of one file with no route and no table is not asked about at
all — its name is what there is to say. Not done: the batch API (half price, results later), which
needs the run reworked.

**2026-10-08 — `--label` defaults to `claude-sonnet-5-5` too**
Tried on Immich the same evening: one request, 27 operations accepted on the first attempt, $0.018,
and the 16 descriptions and 11 edge labels read correctly. Cruz chose it as the default over
`claude-opus-5-5`, which the entry above had named. `--label-model` asks another model; no
side-by-side run against Opus 5.5 was made.

**2026-10-08 — `model.json` is laid out, not thinned; the server compresses**
Immich's model was 3.3 MB. A third of that was layout: a citation took five lines, and 2,128 values
carried an empty provenance meaning "same as the element". Both are gone — a value that fits in 140
characters is written on one line, an empty provenance is not written — and the file is 2.6 MB with
the same content, still a member per line where it matters for a diff. What remains is content
(364 flows are a third of it) and was kept: the file is the reviewable record, and dropping derived
parts would make the published site recompute them. The cost a person feels was the browser's
download, so `archdoc serve` now gzips text: `/api/model` goes from 3.4 MB to 226 KB.

**2026-10-08 — Flows follow what runs out of line, by name**
A flow stopped at `eventRepository.emit(…)` and `jobRepository.queue(…)`, where most of what a
route sets off begins. Now: a call that emits (`emit…`, `publish…`, `dispatch…`) with a literal
first argument — or a choice between two — continues in the methods `@OnEvent({ name })` marks,
each step noted "matched by name"; an object literal named after a job (`{ name: JobName.X }`,
resolved through the enum) is a step to that job, not followed — it runs later and has a flow of
its own, which the lifeline opens. The same word handed to any other method is not an event:
`serverSend('ConfigUpdate')` matched at first and drew a fan-out that does not happen here. In
Python a method is followed where the code states the object's class — `self`, a field `__init__`
assigns from a typed parameter or a constructor, a typed parameter, a local built from a class —
and nowhere else: no inference from use. On Immich: 68 flows now reach a queued job, 37 an event
listener, and `/predict` reaches `InferenceModel.load`. Not done: Celery, RQ and other Python queues.

**2026-10-08 — SQL migrations are the schema where the code declares none**
Until now tables came only from what the code declares — ORM classes, a Prisma schema — on the
reasoning that migrations say how a schema got here, not what it is. That left a container with no
ORM, or in a language archdoc does not read, with no data view at all. Now the `.sql` files under an
application's directory are folded in path order — `CREATE TABLE`, `ALTER TABLE` (add, drop, rename,
type, nullability, keys), `DROP TABLE` — into the schema they leave: each table cited at its
`CREATE TABLE`, each column at the statement that last defined it. The way back is not applied
(`.down.sql`, below `-- +goose Down`). Declarations still win: a container with one declared table
ignores its migrations, so nothing is counted twice and Immich is unchanged. Read for any
language — a Go service gets a data view from its migrations alone, and coverage says its code
was not read. Not read: migrations in a tool's own DSL (Alembic, Knex, Rails' `schema.rb`).

**2026-10-08 — A language with no framework cites its manifest**
Found while testing the above: a service whose manifest named no framework archdoc knows — any
plain Go module — had the technology "Go" with no provenance, failed VAL-05, and nothing was
written. The manifest is what says the language, so it is what the technology cites.

**2026-10-09 — Comparing two commits runs git; nothing else does**
`render.Commit` reads `.git/HEAD` by hand so that archdoc works where git is not installed. A diff
between commits (F-37) needs the files of another commit, which live packed and delta-compressed in
git's object store. Reading that is git's job: `archdoc diff` runs `git archive`, unpacks it into a
temporary directory named as the repository is, and reads it as `generate` would — rules applied, no
model asked, nothing written. A pure-Go git library was the alternative and a large dependency for
one command. `generate`, `serve` and `export` still need no git.

**2026-10-09 — A changed ID is matched on what stayed the same, or not at all**
Identity is the ID, and a service's ID is its Compose name — so renaming one read as a removal, an
addition and every arrow rewired, and for a container with code as hundreds of components, tables
and routes appearing and disappearing. `model.Compare` now pairs a removed element with an added one
when exactly one on each side shares what identifies it: a container's directory, or its kind,
technology and relationships; a table's columns; a component's files. A pair is reported once, as
renamed, and everything attached is compared under the new ID. The same name inside the system on
one side and outside it on the other is re-bounded: a service that left the repository and is still
called. An ambiguous match is not made — two honest changes beat one guessed rename.

**2026-10-09 — `--explain` is its own egress mode, not `structure-only`**
Since 8 Oct `--explain` sends each route's summary, a sentence from a decorator or a docstring. Its
runs were still logged as `structure-only` and announced as "names only": the label promised less
than was sent. The mode is now `structure-and-summaries`, defined beside `structure-only` in
`internal/semantic`, shown in the run log and the app, and held by a test on the wire. AC-8's
wording — `structure-only` transmits zero file contents — stays true of `--label`.

**2026-10-09 — Who calls an API is read from its description, not guessed from who exists**
Immich's container view drew its web app, mobile app and CLI with no arrow to anything: each calls
the server through a generated client, and no URL in their code names it. The repository does say
it, in three steps, each a line: an OpenAPI document; a client of that document — a package whose
code names most of its paths, or a directory an `openapi-generator` command writes into; and an
application that holds that client or depends on its package. The document is tied to a container
by counting its operations against the routes that container's code declares (Immich: 274 of 274),
so nothing is drawn from a document nobody here serves. A development dependency counts only where
the code imports it — a bundled CLI lists what it ships with there. Closes O-10 for generated
clients; hand-written calls to a computed address stay unresolved.

**2026-10-09 — A person reaches what a person runs**
The only actor evidence was a published port. A web front end, a mobile app and a command-line tool
are now reached by the person too, each arrow citing the manifest line that makes the application
what it is. This is the one arrow that rests on what kind of thing an element is rather than on a
statement about the arrow; its note says so.

**2026-10-09 — A package others run is a library, whatever command it ships**
`@immich/plugin-sdk` declares a `bin` and was drawn as a container. It also exports code, and a
package that runs depends on it: it is a library whose command is a build tool. A test suite
depending on a tool does not count, or the CLI would stop being one.

**2026-10-09 — A large container opens on its main components**
Immich's server has 75 components and 758 uses: accurate, cited, and unreadable as a picture; C4's
own advice is to split a component diagram long before that. A container with more than 24
components now has a main view — the 16 that handle the most routes, pages, commands and jobs, by
what the code declares, then by size — and the explorer opens on it, with every component and the
by-folder view one click away. Selecting something outside the main ones shows them all. The rule
is a count, not a judgement; `main-<container>.svg` is committed beside the full one.

**2026-10-09 — Labels are remembered, like explanations**
A description `--label` wrote lasted one run: the next plain `generate` dropped it and recorded a
new version without it. Labels are now kept in `.archdoc/labels.json`, each beside what it was
written about — an element's kind, name and technology as extracted; a relationship's ends and
protocol — and applied again on every run while that is unchanged, through the same validator as
when they were first accepted. Change the element and its label is left out until `--label` is run
again. A newer label replaces the older for the same value. This is F-30 for labels.

**2026-10-09 — The order of elements is by rank and ID, and nothing else**
`sortNodes` compared kinds before ranks, and applications, components, tables and modules share a
rank: two of different kinds were each "not before" the other, which is not an order. The result
was repeatable for one input — so five identical runs agreed (AC-7) — but adding or removing any
element reshuffled the rest. Found when removing one container from Immich's model made nine
unrelated explanations stale: a component's facts list its tables in model order, and the order
had moved. Fixed to rank, then ID. Every repository records one new version from this, once.

**2026-10-09 — A small diagram keeps its descriptions when fitted to the screen**
The app hid technology and description below 85% zoom, and a container view fitted to a window
sits below that — boxes with only names, which is not a C4 diagram. Views of twenty elements or
fewer keep them down to 50%. The app's description text is now set at the line height and width
the engine sizes boxes for, so what the layout made room for is not cut.

**2026-10-09 — The main view's rule stays a count, and starts later**
Looked at again on Immich. Ranking by routes, pages, commands and jobs handled keeps `search` and
`media` and leaves out `sync` and `shared-link`; ranking by handled, stored and size together
keeps those two and drops `search` and `media` for `stack` and `config`. Neither is what a
maintainer would pick, and no count will be: which components matter most is a judgement, and
archdoc states rules it can show. The simple rule stays — it is the one the diagram's own label
can say in a line — and every component is one click away. One real defect is fixed: a container
needs more than 24 components to get a main view, not 17, so Immich's mobile app shows all 18
instead of hiding a three-line and a four-line one. A person choosing the main components, as a
rule that survives regeneration, is the right way to do better, and is not built.

**2026-10-09 — The system has a description; a component's box says what it does**
The context view's own box had a name and nothing else: no file says what a system is for, and
the system is not an element a label could land on. The model now carries the system's
description, `--label` is asked for it under the id `system`, the validator lets that one
operation through without an element, and it is remembered until the system gains or loses a
container. A component's box shows the first sentence of its explanation, marked as a model's like
any description — in a view of 24 components or fewer, and never from an answer about an earlier
version of the code. The view carries it; the model's component does not, so nothing a model wrote
enters the architecture's fingerprint through it.

**2026-10-09 — A rank too wide to read is folded**
With descriptions on them, the sixteen main components of Immich's server laid out as two ranks ten
boxes wide — a strip in either orientation, since few arrows are drawn among them. When neither
orientation is within twice as wide as tall, a rank of more than five boxes is folded: each box past
the fifth is put a rank below the one five places before it, by an invisible constraint, and the
result is kept only if its shape is nearer a page's. Arrow labels and the note on a shared
component are now wrapped in the app where the engine wraps them, so neither runs under a box.

**2026-10-09 — A label run starts from what is remembered**
`--label` ignored remembered labels and applied only what the new answer held, while the plain run
after it applied both — so the two runs gave different models and the second recorded a version
nothing had caused. Remembered labels are now applied before a new run, which writes over them.

**2026-10-09 — A front end's components are the features its layers share**
Immich's web app has no `album.service.ts`-style convention across its code, so its components
were its folders: `lib/components`, `lib/modals`, `lib/utils`. It is laid out by layer, and a
feature's name runs across the layers: `album.service.ts`, `AlbumEditModal.svelte`,
`album-utils.ts`, `components/album-page/`, `routes/(user)/albums/`. Where the suffix rule does not
carry an application, that is now read: the layer is the role; a file's name is the longest run of
its leading words that the code itself uses as a name — the stem of a file with a counted suffix,
or a directory inside a layer — with plurals, `-page` and a router's `(group)` and `[param]`
segments set aside; a feature is a name in two layers or more, of at least three files. A name made
only of layer words (`shared-components`, `widget`) is not a feature. Everything else stays with
its folder, and the folders remain as the second view. Measured on Immich: web 15 folders → 41
components, 26 of them features holding about half its files (`asset`, `admin`, `timeline`, `user`,
`album`, `auth`, `people`, `workflow`, `shared-link`…); mobile 18 → 54. The server, which has the
suffix convention, is unchanged at 75. The rule is names, so it is as good as the naming: `people`
and `person` are two features in the web app because the code spells them two ways.

**2026-10-09 — A main view draws each component's strongest uses**
The full component view says "used by most" on a box and leaves those arrows out, which makes
seventy-five components readable. Carried into the main view it failed: the main components of a
tightly knit application are nearly all used by most, so Immich's mobile app drew sixteen boxes and
no line, with 155 uses between them. A main view now draws, for each component, its two strongest
uses of the others — by number of imports — and says so on its boundary, with the count it chose
from. Every use is still in the inspector and in the full view. Where a container declares no
route, page or job, the boundary says the components are its largest, since that is the rule that
picked them.

**2026-10-09 — Checked against the vision: on path for a system, off it for anything small**
Cruz asked whether the product is still what the vision describes — able to explain what an AI
built whether it is a full-stack project, a script, a page or an app. Tried on three projects
written for the question, not assumed:

- A single Python script that reads a CSV and calls an API: refused — "no deployable Compose file
  and no application manifest found".
- A single HTML page with one script that fetches a list: refused, the same way.
- A six-file React app with two pages and Supabase as its backend: documented, thinly and partly
  wrongly. Supabase — the whole backend — is not drawn; four pages are reported where there are
  two; and it gets eleven arc42 documents, seven of them stubs.

Against the vision (`docs/vision.md` §1.1, §1.2, D-13): "understand software an AI built" holds for
a system — context, containers, components, data, features, flows, cited explanations, all working
on Immich. "Whatever its size" does not: a project with no manifest is rejected, and output is not
sized to the project. "What the AI did" — the change per session or commit, and plan against code
(F-38, F-40, F-41) — is not started; `archdoc diff` is its base and no more. The vibe coder, who the
vision says starts from a guided story, is not served: what exists is reference documentation.

Why: 8 and 9 Oct went into making Immich's C4 picture good. Worth doing, and it pulled the work
toward architecture documentation for large systems — the narrower product the vision was written
to move beyond. Order agreed to get back: (1) accept a project with no manifest, output sized to
it; (2) make the small app right — hosted backends recognised, pages counted once; then (3) a
change summary a person reads, on `archdoc diff`; (4) plan against code. Cruz chose 1 and 2 now.

**2026-10-09 — A project with no manifest is documented, on one page**
The first two steps back toward the vision. (1) Where no manifest describes anything that runs,
the repository is read as what its files are: Python files are a tool a person runs, a page with
scripts beside it is a web front end, scripts alone are a tool — cited at the file that starts it,
where a manifest's line would be. A script's way in is the file that says it is run directly
(`if __name__ == "__main__"`), a page's is its HTML file, with its `<title>`. (2) A project of one
application, nothing deployed beside it and thirty source files or fewer gets one page — what it
is, does, reaches, is made of, is built on — and no arc42 chapters and nothing to fill in; they
appear when it grows. "Related" in the plan no longer counts a person reaching an application,
which had quietly turned every web app into a twelve-chapter system. Also: a call to a name that
holds a URL (`API = "https://…"`, `requests.get(API)`, `fetch(API)`) is a call to that URL, resolved
within the file; hosted backends — Supabase, Firebase, Clerk, Appwrite, Convex and their kind — are
in the catalog, since they are the whole backend of most first versions; Next's `pages/` rule
applies only to a Next application; and the application at a repository's root is identified by
its name, not by ".". Measured on the three test projects: all documented, each on one page, the
script's API, the page's fetch and the app's Supabase drawn, two pages reported for two.
Not done: inline `<script>` in a page is not read; a tiny project in the app still shows the full
navigation; the one page is not yet the guided story the vision gives a vibe coder.
