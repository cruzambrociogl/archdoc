# Seminar Project — Working Notes

> This is my own reference document. Not a submission. The point is to be able to sit
> down, read it, and hold the whole idea in my head again — what it is, why it matters,
> what makes it hard, and what to build first.
>
> The original Spanish notes are kept in `proyecto-seminario-notas.md`.

---

## 1. The idea in one paragraph

A tool that reads a codebase — its infrastructure config, its source code, and its
written specs — and produces professional-grade architecture documentation and diagrams
from it. Not a one-time generation: it keeps a **canonical model** of the system in a
database, regenerates it on every commit, and reports what changed architecturally. Every
box and every arrow on the diagram carries a pointer back to the file and line it came
from, so it cannot invent components that don't exist. I refine the output by chatting
with it, and my corrections persist as rules that survive regeneration.

The short version: **the diagram is never the source of truth — a validated model is. And
the LLM is never responsible for facts.**

---

## 2. Why this exists — the real problem

### The thing I actually experience

AI writes almost everything now, and I guide it. That works, and it's fast. But it's fast
in a way that breaks something.

When you *write* code, knowledge sediments as you type — why this module calls that one,
what you tried first and abandoned, where the real boundary of the system is. When you
*direct* code, that doesn't happen. Reviewing confirms the result works; it does not build
the mental model. So a few weeks later I'm in the same position as a new developer on a
system I designed — except there's nobody to ask, because nobody ever knew it either.

This is the core insight and it's worth stating precisely:

> The problem isn't that documentation is missing. It's that **the understanding never
> formed in the first place** — not even in the person who built it.

That's different from the usual "we should document more" complaint. Documentation debt
assumes someone knew and didn't write it down. This is knowledge that never existed.

### Why it's structural, not a discipline problem

It's not that I'm lazy about docs. It's arithmetic: production accelerated, reading did
not. Industry-wide, roughly 85% of developers use AI tools daily and around 46% of new
code is AI-generated,¹ while human reading speed is exactly what it was. Every improvement
in generation widens the gap. **This tool lives on the side that stopped scaling** — which
means it gets *more* useful as models get better, not less.

### The quieter half of it

Stack Overflow used to be an accidental archive of *reasoning* — the failed attempts, the
rejected alternatives, the why behind each decision. That archive is shutting down:
monthly question volume fell more than 75% from its 2014 peak of ~200,000, down to under
50,000 by late 2025, with the answering community emptying out in parallel.

The consequence: models were trained on the golden years (2008–2020), but problems solved
from 2024 onward aren't being documented publicly at that scale. The knowledge graph
stopped growing. And inside a company the same thing happens at smaller scale — the
reasoning happens in a chat session, the code ships, the session is discarded.

**Git records what changed. It never records why.**

¹ *These two figures came from my earlier notes and I haven't verified the sources. If I
ever use them in front of anyone, find the citations first.*

---

## 3. The draw.io experiment — what it actually proved

Before settling on this, I ran a real test. I hand-wrote a detailed natural-language spec
of a system I know well and gave it to draw.io's AI assistant to generate the architecture
diagram.

**The system, described generically:** a reverse proxy routing six subdomains; a React
SPA; an identity provider with its own database, issuing JWTs consumed by four different
services; an orchestration layer running scripts against a document store, a queue, and an
object store; a GPU worker on a remote machine reachable over a VPN, pulling from the queue
and spawning an isolated container per job; a local model-serving service on that same
machine; and an annotation tool patched to authenticate against the identity provider.
Seven logical groupings, about twelve components, roughly twenty protocol-carrying edges.

The result was partially correct and useless. And it failed in **two distinct ways** — which
turned out to be exactly the two things this project has to solve.

### Failure 1 — I had to be the extractor

The tool contributed zero knowledge about the system. Every fact in that diagram — the
routes, the queue name, the model service port, the blocking-pull pattern — came out of my
head. **The diagram's quality was capped by what I happened to remember that day.**

And I can prove the cap, because it shows in my own prompt:

- One component — a WebSocket gateway for real-time job progress — appears as the
  *destination* of a connection but is **never defined as a component**. It's in none of
  the seven groupings.
- **The edge that feeds it was never written.** Nothing in the spec says who publishes
  progress events to it. The whole real-time path is left dangling.
- Two more elements (the docs site and the landing page) are listed as proxy routes but
  never defined either.

So: a spec written *deliberately*, by the person who built the system, about a system he
knows, still left a component half-specified and an edge missing. That's not carelessness.
That's the exact failure mode described in §2, caught in the act.

**This is the most convincing evidence I have for the whole project**, because it's
verifiable in the artifact — it doesn't require believing a statistic.

### Failure 2 — Even with perfect input, the translation loses information

Separate from what was missing: the assistant also dropped and misread things that *were*
in the prompt.

The reason is architectural. draw.io is a **drawing tool with an LLM bolted on**. It has no
domain model of architecture. It doesn't know that an edge needs a protocol, that a
boundary means something specific (network? trust? deployment?), or that a component needs
a stated responsibility. To that tool, an architecture diagram and an org chart are the
same object: boxes and arrows.

That's the gap. I'm not building a drawing tool. I'm building something with an actual
architecture domain model.

### What this directly gives me

A schema with referential integrity would have **refused to render** that diagram and
pointed at the undefined component. So the validation rules aren't design preferences —
they come straight out of observed failures:

1. No referenced component may be left undefined.
2. Every edge declares protocol, direction, and sync/async.
3. Every component belongs to exactly one boundary.
4. Every component declares a responsibility and a technology.

---

## 4. Why this isn't just a prompt

This is the question I have to be able to answer instantly, because it's the obvious
objection. Claude can draw me a Mermaid architecture diagram of a repo today, in one shot.
That's real.

But that diagram:

- is **wrong by Thursday**, and has no way to tell me it's wrong;
- has **no ground truth** — nothing distinguishes a component that exists from one the
  model inferred;
- **can't tell me what changed**, because it has no memory of the previous version.

That last one is the important one. My actual pain isn't "I need a diagram." It's **"I lost
track of a system that keeps moving."** So the center of the project is the *living* part:
regenerate on every commit, and surface the **architectural diff**.

> "This PR added a direct call from the API layer to the database, skipping the service
> layer."

No prompt produces that, ever, because it requires persistent state across time. That's the
structural argument: the difficulty lives where a language model can't reach.

Provenance and the validator are what make the output **trustworthy**. The architectural
diff is what makes the tool **irreplaceable**.

### Other objections worth having an answer for

| Objection | Answer |
|---|---|
| "Doesn't Doxygen / Swimm / Mintlify do this?" | They document *what the code is*, from syntax, at function level. Critically, **they don't read infrastructure config** — which is where the architecture of a modern service-based system actually lives. They come from the era when "architecture" meant class relationships. |
| "Doesn't draw.io / Excalidraw AI already do this?" | Tested it. See §3. Two simultaneous failure modes, because it's a drawing tool with no domain model. |
| "Why not just ask Claude Code for the docs?" | Produces something plausible, unverifiable, and stale the next day. The contribution is the *pipeline*: facts from config and static analysis, meaning from the model, provenance on every element, structural validation, regeneration per commit. The LLM is a component, not the product. |
| "Won't AI just solve this itself?" | Understanding isn't delegable. A human still signs off on architecture decisions, audits, and onboarding. You can't approve a system you don't understand. |
| "Isn't the scope too big?" | See §7. One repo, architecture + flow views, traceability, architectural diff. Everything else is roadmap, explicitly. |
| "How do we know the diagram is right?" | The right question, and it has a numeric answer: % of elements traceable to real file and line, measured automatically. See §8. |

---

## 5. What "professional grade" actually means

I need this to be concrete, not a vibe. It anchors to standards that already exist:

- **C4 model** — abstraction levels: context → container → component → code.
- **arc42** — the industry-standard documentation template, with defined sections. Gives
  me a verifiable conformance target: does the generated document fill the arc42
  structure?
- **ADR** (Architecture Decision Records) — the standard format for recording a decision
  and its alternatives. This is where the long-term direction of the project points.

From those, the concrete model requirements:

| Requirement | What it means |
|---|---|
| Each element defined exactly once | Name, responsibility, technology, boundary |
| Typed edges | Protocol, direction, sync or async |
| Explicit boundaries | Network / trust / deployment — not decorative grouping |
| Consistent notation | The same concept is drawn the same way everywhere |
| Traceability | Source file and line on every node and edge |
| Versioned and diffable | The model lives in the DB with full history |

### And the one that organizes everything else

**One canonical model, many consistent views.**

In the draw.io test, if I'd separately asked for a deployment diagram, nothing would have
guaranteed it described the same system as the architecture one — two independent
generations, two independent hallucinations. With a single validated graph and views as
*projections* of it, consistency across views is **structural, not hoped for**.

This is the real reason to keep a custom JSON graph as canonical rather than generating
Mermaid directly. Much stronger than "arbitrary metadata."

---

## 6. How it works

### The three extraction tiers — the key technical decision

The insight from the draw.io test: **in a modern service-based system, the architecture
lives in the configuration, not in the source code.** A pure AST analyzer would have
produced a *worse* diagram than my handwritten prompt.

So extraction is ordered by decreasing confidence:

| Tier | Source | What it yields | Confidence |
|---|---|---|---|
| **1. Config / IaC** | docker-compose, Dockerfiles, Traefik/nginx config, k8s manifests, CI/CD, `.env` schemas | Service inventory, ports, networks, routes, deployment boundaries | **High** — declarative, zero inference |
| **2. Static code analysis** | tree-sitter / LSP: imports, call edges, clients, queue names, bucket names, endpoints | What talks to what, inside and between services | Medium — needs heuristics |
| **3. LLM** | Output of 1 and 2, plus specs and plans in the repo's `.md` files | Semantic grouping, naming, responsibilities, prose | Low — never emits facts, only interprets |

Why this ordering is a win in four separate ways:

1. **Tier 1 is the cheapest to build and pays the most.** Parsing a `docker-compose.yml`
   and a set of proxy routing rules reconstructs most of the topology with no inference at
   all.
2. **It raises the floor on the traceability metric.** Tier 1 facts are 100% traceable by
   construction, so a large share of the diagram is verifiable without trusting the model.
3. **It cuts the language dependency.** The MVP supports any stack that ships declarative
   config, which widens the pool of test repos considerably.
4. **It's precisely the competition's blind spot.** None of the doc tools read
   infrastructure.

### The main loop

```
Me (chat)
  → backend orchestrator
  → Tier 1: config / IaC parsers        ─┐
  → Tier 2: static code analysis         ├─→ ground truth
  → Tier 3: LLM (groups, names, explains)─┘
  → LLM returns a STRUCTURED DIFF, not an image
  → Validator (schema + referential integrity, repair loop)
  → Update the canonical graph in the DB (+ new version)
  → Project the views (architecture, flows, deployment)
  → Push to frontend → render
  → I edit (chat OR canvas) → back into the graph
```

The LLM returning a **diff of operations** rather than a whole diagram is important —
regenerating everything loses the layout, is slow, and is unstable between runs.

### The canonical model

Custom JSON graph as canonical; Mermaid, PlantUML, and draw.io XML as export formats.

```json
{
  "nodes": [{
    "id": "", "type": "", "label": "", "responsibility": "",
    "tech": "", "boundary": "",
    "provenance": [{ "file": "", "lines": "", "tier": 1 }]
  }],
  "edges": [{
    "from": "", "to": "", "label": "",
    "protocol": "", "sync": "sync|async",
    "provenance": [{ "file": "", "lines": "", "tier": 1 }]
  }],
  "boundaries": [{ "id": "", "label": "", "kind": "network|trust|deployment" }]
}
```

The `tier` field on provenance does double duty: it lets me compute the evaluation metric
straight off the model, and it lets the UI visually distinguish **verified fact** from
**model interpretation**. That second one matters more than it sounds — it's what makes the
output safe to trust selectively.

### Where the hard engineering actually is

This is the list to point at when someone asks what's difficult about this.

1. **Incremental diffs, not regeneration.** The model emits *operations* against the
   existing graph.
2. **Validate → repair loop.** Schema, referential integrity, boundary rules. Invalid
   output gets fed back to the model with the error and retried. **The model literally
   cannot emit a broken diagram** — which is exactly what draw.io did.
3. **Multi-level semantic clustering.** Turning 400 files into 8 boxes that actually mean
   something, and keeping those groupings coherent across C4 levels. This is the genuinely
   hard problem in the project.
4. **Projecting mutually consistent views** from a single model.
5. **Bidirectional sync.** A canvas edit updates the graph *and* the conversation context,
   so the model doesn't contradict what I just changed by hand.
6. **Layout stability.** Adding one node must not make everything else jump. Sticky
   positions plus incremental layout.
7. **Persistent corrections as rules** — e.g. "treat `/utils` as infrastructure, not a
   component" — that survive regeneration.

### Features that fall out of the architecture (and can't be prompted)

1. **Architectural diff per commit** — the headline feature. Automatic re-run and drift
   report. Requires persistent state over time.
2. **Provenance on every element** — file and lines, plus extraction tier. Click a box,
   jump to the code. No hallucinated components, and the UI separates fact from
   interpretation.
3. **Navigable C4 abstraction levels** — context → container → component → code, with
   guaranteed consistency between levels.

### Stack

| Layer | Options |
|---|---|
| Tier 1 extraction | Custom parsers over YAML/TOML/Dockerfile: compose, Traefik, nginx, k8s, GitHub Actions |
| Tier 2 extraction | tree-sitter, LSP |
| Render | **React Flow** (editable, nodes/edges 1:1 with the JSON) + `dagre`/`elkjs` for layout; Cytoscape.js as alternative |
| LLM | API (Anthropic/OpenAI/Gemini) or local **Ollama**. Use **structured outputs / JSON schema** and **tool calling** (`add_component`, `connect`, `rename`, `remove`, `group_into_boundary`) |
| RAG | Embeddings → **pgvector** / Qdrant / Chroma. Corpus: C4 catalog, arc42 template, pattern catalogs, cloud reference architectures |
| Backend | **FastAPI** (Python) or Node/NestJS (TS). **WebSockets** for live model streaming |
| Storage | PostgreSQL + **full version history** of the graph (each diff → undo, time travel, architectural diff) |

---

## 7. What to build first

### In the MVP

- **Architecture view (C4 container + component)** — the core, where the hard engineering
  lives.
- **Flow / sequence views** — derived from the same call graph Tier 2 already extracts, so
  the marginal cost is low.
- **Full traceability** with per-tier provenance.
- **Architectural diff per commit.**
- **Chat refinement** with persistent rules.
- **Export** to Mermaid / draw.io XML.

### Not in the MVP — roadmap, in order

1. **Deployment view** — the best stretch goal. Manifests are declarative, so it's almost
   fully verifiable and carries little risk. First thing to add if there's time.
2. **ADRs and reasoning capture** — the long-term direction. Capture design decisions and
   their alternatives, link them to the parts of the system they affect.
3. **Ingesting AI session transcripts** — the most novel ingredient and the biggest scope
   risk. Transcripts are ephemeral, enormous, and formatted differently per tool. **Decision:
   out of the MVP.** The tool must deliver full value from repo + `.md` files alone. Session
   history is designed as an *optional enrichment layer* that attaches rationale to nodes
   that already exist from extraction — never as a source of structure. If I build it the
   other way around, I've built a transcript parser instead of a comprehension tool, and a
   format change leaves me with nothing.

### Explicitly cut

- **Testing diagrams** — coverage-shaped, not structure-shaped. Dilutes the project
  without adding depth.
- **Voice** — cut from the MVP. Text is the backbone; voice would be a thin layer over the
  same pipeline and adds nothing to the technical argument.
- **"Any kind of diagram"** — this is an architecture tool, not a general diagramming tool.
  That generality is exactly what makes draw.io fail.

---

## 8. How I'd know it works

Three test subjects, increasing complexity, all public or my own:

| Subject | Why | Role |
|---|---|---|
| **Immich** (self-hosted) | Simple multi-service: server + ML service + Postgres + Redis, all in compose | Baseline / development |
| **Supabase** (self-hosted) | Topology very close to the reference case: edge proxy routing to multiple services, identity provider, Postgres, polyglot services — all declared in config | Complex / main case |
| **This project itself** | Dogfooding. If the tool can explain to me the system I built and forgot, that's the demo | Demo |

**Note:** the real system that inspired this is my employer's. It is **not** a test subject
and is not documented anywhere in this project. Its generalized topology is kept only as a
*target complexity spec* — the MVP should be able to reconstruct a system of that shape:
~12 components, 7 boundaries, ~20 protocol-carrying edges, including one VPN network
boundary.

### Metrics

1. **Traceability (automatic, primary).** % of nodes and edges with verifiable provenance
   to a real file and line. Reported broken down by tier — the Tier 1 share is pure fact.
2. **Structural accuracy.** Compare against hand-drawn reference architectures for the
   three subjects: precision and recall on components, edges, and boundaries. Costs me
   three hand-drawn references — bounded and known.
3. **arc42 conformance.** % of template sections the generated document fills with
   substantive content.
4. **Drift detection.** Run over the subjects' real commit history — does it catch the
   architectural changes that were actually introduced? Measurable retrospectively, no
   human subjects needed.
5. **Human comprehension (optional, expensive).** How long a new developer takes to answer
   questions about an unfamiliar repo, with and without the tool. Needs 6–8 participants.
   **Decision:** only if I can get the group comfortably. Metrics 1–4 stand on their own.

---

## 9. Open questions and to-dos

- **Find real sources** for the 85% adoption and 46% AI-generated-code figures, or drop
  them.
- **Check the §3 description** of the reference system — decide whether even the
  generalized topology is too close to my employer's system. Easy to soften further: drop
  the component counts, or reduce it to a purely abstract shape.
- **Semantic clustering is the real research question.** How do you turn 400 files into 8
  meaningful boxes, reproducibly? Everything else in this project is engineering; this part
  is the one I don't yet know how to do well. Worth reading up on before committing to an
  approach.
- **Decide the repair-loop budget.** How many retries before the validator gives up and
  surfaces the failure to me instead of looping?
- **Layout stability strategy** — sticky positions sound simple but get subtle once nodes
  are removed or reparented across abstraction levels.

### Decisions already made (so I stop relitigating them)

| Decision | Why |
|---|---|
| Custom JSON graph as canonical, not Mermaid | One model, many consistent views. Mermaid is an export format. |
| Config/IaC before code analysis | That's where the architecture actually lives in service systems. |
| Architecture + flows only in the MVP | Everything else is a different extraction problem. |
| Session transcripts out of the MVP | Highest novelty, highest scope risk. Value must exist without them. |
| Voice cut entirely | Adds no technical depth. |
| Architectural diff is the headline feature | It's the part that's structurally impossible to prompt. |

---

## 10. Parked idea — crop health monitoring platform

Came close to being the pick. Parked because it's a *service for other people* rather than
a tool I'd use myself, and its test cycle is slow: collect field images, label, retrain,
find farmers.

> **Platform for crop health monitoring and agronomic advisory**
>
> In Guatemala, maize and coffee support both household food security and the rural
> economy, yet diseases like rust and leaf blights cause recurring losses. For most growers
> the problem isn't recognizing that a plant is sick — many identify symptoms from
> experience — but knowing *how widespread the infection is, how fast it's advancing, and
> what action is justified.* Field inspection is subjective, undocumented, and rarely
> repeated consistently, so decisions rest on impressions rather than evidence.
>
> The platform would let a family or company continuously monitor their own plots through
> two connected surfaces. In the field, a conversational assistant directs structured
> sampling: it generates a route with points defined by the plot's size and shape, guides
> the grower through them, and requests photographs at each one, linking every image to its
> point, plot, and date. The system returns a diagnosis with an objective severity level and
> explains the recommended treatment in plain language, with voice support for users with
> limited literacy. That data feeds a web command center where the owner or technician sees
> plots on a map, incidence and severity by zone, the evolution of each sampling round, the
> direction and speed of the disease's spread, and alerts when thresholds are crossed.
>
> The pilot crop would be defined in the initial phase — maize and coffee are the
> candidates, given their economic and food relevance and data availability — with the
> system designed crop-agnostically. Technically it combines a computer vision model for
> classification and severity estimation, a generative AI layer grounded in institutional
> agronomic documentation, and a web application for visualization and alerts. The
> contribution isn't identifying diseases, but turning a subjective inspection into a
> quantitative, geolocated, traceable record that supports timely decisions.

**Datasets found:**

- **Coffee:** JMuBEN (~58,555 images, 5 classes incl. rust, cercospora, phoma); RoCoLe
  (field photos with rust severity levels 1–4).
- **Maize:** PlantVillage subset (3,852 images, 4 classes: gray leaf spot, common rust,
  northern leaf blight, healthy). *Caveat:* doesn't cover tar spot or fall armyworm, which
  are the big problems in Guatemala.
- **Beans:** Makerere's iBean (~1,296 field images, 3 classes), included in TensorFlow
  Datasets — fastest possible start.

**Competition found (important):**

- **CoffeeCloud (ANACAFÉ, Guatemala)** — national coffee app with rust, borer, and eye-spot
  surveillance, sampling, and a web portal. **No computer vision** — the grower fills out a
  manual questionnaire. That's the gap. Its existence also *validates* the premise: a
  national institution built its alert system on incidence and severity from structured
  sampling.
- **Plantix** — global photo diagnosis, ~800 symptoms, 60 crops, 10M+ downloads,
  geotagged images. Digital Green integrated it into a WhatsApp chatbot.
- **Commercial scouting platforms** (OneSoil, GeoPard, xarvio, EOSDA) — aimed at large
  mechanized farms, satellite/NDVI-driven.

**Strategic conclusion:** for coffee I'd have to differentiate against a free, nationally
deployed institutional incumbent. For maize there's no Guatemalan equivalent — open space.

### Rejected by the "an AI already does this" filter

- Smart notes/clipboard with auto-classification and resurfacing.
- University schedule and task sync to Google Calendar with smart reminders and work
  conflict detection.
- Unified personal hub (notes, deadlines, calendar, clipboard).
- Whiteboard photo → structured notes.
- Screen recording → documentation.
- Receipts → structured expense data.
- Lip reading (VSR) — rejected for complexity and insufficient accuracy.
- LENSEGUA (Guatemalan sign language) — good idea, but requires building the dataset.

---

## 11. Alternative directions, if I ever need to pivot

All of these pass the filter by construction — the difficulty is latency, correctness,
state, scale, or concurrency, not language.

### AI agent infrastructure (the strongest vein found)

The pattern: models scaled fast; the infrastructure to make them safe, testable,
observable, and cheap did not.

1. **Agent security** — OpenClaw accumulated 280+ advisories and 100+ vulnerabilities in a
   short time; leaked keys, hijacked agents, malicious skills. The core problem: the agent
   can't distinguish a legitimate command from a malicious prompt embedded in a web page or
   document. → egress firewall, static skill analyzer, real sandbox.
2. **Observability** — no consistent, provider-independent way to review plans, tool calls,
   approvals, and diffs. → trace debugger with replay.
3. **Setup friction** — install, API key, skills, config; worse with local models.
4. **Long-task reliability** — no checkpoint/resume. At 85% per-step reliability, a 10-step
   flow completes ~20% of the time. → durable execution layer.
5. **MCP tool poisoning** — tool descriptions are natural language the agent reads as
   context, and nothing stops a server from returning whatever it wants. One scan found
   1,862 exposed MCP servers; of 119 checked, all 119 allowed listing tools without
   authentication. → MCP scanner/auditor, sanitizing gateway, signed registry.
6. **Agent evaluation** — it's manual. An agent can reason well, pick the wrong tool,
   produce plausible output, and fail silently. → regression harness, per-trace failure
   diagnosis.
7. **Memory** — drifts across sessions, invisible without longitudinal evaluation.
8. **Cost** — no per-task cost governance.
9. **Fixes don't propagate** — a correction stays in the personal config of whoever made
   it.
10. **Human-in-the-loop** — no generic approval layer.
11. Multi-agent coordination without metrics; context management; local model ergonomics.

### Other technical veins

- **Emulators / low level:** CHIP-8, Game Boy, mini-git, mini-Docker, a database engine, a
  ray tracer.
- **Compilers / languages:** own language, symbolic algebra system, transpiler, query
  language over git history or a codebase.
- **Real-time simulation:** physics engine, fluids, procedural terrain, n-body.
- **Games with real engineering:** rollback netcode, own chess engine (minimax/MCTS), an RL
  agent that learns to play.
- **Audio / DSP:** synthesizer from scratch, Shazam-style fingerprinting, FFT chord
  detection, source separation.
- **Networking / distributed:** BitTorrent-style P2P, key-value store with Raft,
  collaborative editor with CRDTs.
- **Train my own model:** neural net from scratch with backprop, genetic algorithms,
  artificial life.
- **Crypto / security:** steganography, own blockchain, honeypot.
- **Format parsers:** PNG, ZIP, MIDI, typefaces, QR from the spec.

---

## 12. How to unblock ideas, if I ever need to again

The block isn't creativity, it's **retrieval**. You can't remember frictions without a
trigger in front of you.

**The method that works:** for 3–4 days, note on your phone every time the thought *"this
again"* or *"why am I doing this by hand?"* shows up. By the end of the week you have 8–15
real frictions to choose from. **Record instead of remember.**

**Concrete triggers** (more effective than "what annoys you?"):

- What did I do at work yesterday, step by step?
- What was the last mediocre AI answer I had to fix by hand?
- What tabs do I have open right now?
- What tool do I use and hate but haven't replaced?

> The draw.io experiment (§3) is this method working. The idea didn't come from searching
> for a topic — it came from a real friction, noticed at the moment it happened.
