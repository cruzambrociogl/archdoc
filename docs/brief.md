# archdoc — brief

> **If you are here to design, brand, or write the report: read this file and follow its
> pointers. You do not need the rest of the repository.**
>
> This is the entry point for the non-code workstream, the way `CLAUDE.md` is the entry point
> for the code. It holds identity and structure — things that rarely change. It deliberately
> holds **no numbers, dates, or measured results**: those live in the files it points at, so
> there is only ever one copy to keep true.

---

## What the project is

**archdoc reads an existing codebase and produces the architecture documentation it should
have had — accurate, traceable to the code, and regenerable as the system changes.**

Point it at a repository. It reads the configuration that already describes how the system is
deployed, builds a validated model, and writes C4 diagrams and arc42 documentation where every
element links back to the file and line that proves it exists.

It is **not** a drawing tool, and **not** an enforcement tool. It describes what is there.

## Comparison matrix

`docs/comparison-matrix.md` — the full feature list and the matrix against the eight known
alternatives (Archify, GitDiagram, DeepWiki, Google Code Wiki, the AI diagram editors, Structurizr,
Compose visualisers). In Spanish, because it feeds the report's functional-feasibility section. Every
row is marked implemented or planned; do not promote a planned row to the present tense.

## How to describe archdoc — the canonical wording

One place for the description, so the report, the slides and the branding sheet cannot drift.
**Two versions, and which one to use depends on tense, not on taste:** the first describes what
runs today and is the only one to use at a review or in a submitted report; the second describes
the direction of `docs/vision.md`, and becomes the first once code reading ships.

### Today — what runs (use this for anything graded)

> ArchDoc lee los archivos de configuración de un repositorio (Docker Compose, configuración de
> proxy, variables de entorno, contratos de API) y genera automáticamente diagramas de
> arquitectura en formato C4 y documentación en formato arc42.
>
> Cada servicio y cada conexión del diagrama se extraen directamente de la configuración e
> indican el archivo y la línea que los respaldan. La IA solo se utiliza para nombrar y describir
> elementos ya detectados: no puede agregar componentes que no existan en el proyecto.

> archdoc reads a repository's configuration files — Docker Compose, proxy configuration,
> environment variables, API contracts — and automatically generates C4 architecture diagrams and
> arc42 documentation. Every service and every connection is extracted from that configuration and
> states the file and line that back it. The AI is used only to name and describe elements already
> detected: it cannot add components that do not exist in the project.

### The direction — once code reading ships (`docs/vision.md` §2.2)

> ArchDoc es la alternativa local y verificable a los wikis de código generados por IA. Lee un
> repositorio — su código fuente y sus archivos de configuración (Docker Compose, configuración de
> proxy, variables de entorno, contratos de API) — y genera la documentación de arquitectura que
> ese proyecto debería haber tenido: diagramas C4, documentación arc42 y explicaciones del sistema
> (sus funcionalidades, sus flujos y sus datos).
>
> Cada elemento y cada relación se extraen del repositorio e indican el archivo y la línea que los
> respaldan; lo que no se puede resolver se marca como no resuelto, en lugar de omitirse. El código
> nunca tiene que salir de tu máquina y la documentación vive en tu repositorio, no en el sitio web
> de otra empresa. La IA solo nombra, agrupa y describe elementos ya detectados, y cada afirmación
> cita la evidencia en la que se basa: no puede agregar componentes que no existan en el proyecto.

> archdoc is the local, verifiable alternative to hosted AI code wikis. Point it at any repository
> and it produces the architecture documentation that repository should have had — diagrams, module
> walkthroughs, data models, flows, and answers to questions about them. Every element is extracted
> from the code and cites the file and line that proves it, the model can never add something that
> does not exist, the code never has to leave the machine, and the documentation lives in the
> repository rather than on someone else's website.

### One line

> ArchDoc convierte un repositorio en documentación de arquitectura verificable: cada elemento
> señala el archivo y la línea que lo prueban, y la IA nunca inventa componentes.

> archdoc turns a repository into architecture documentation you can verify: every element points
> at the file and line that prove it, and the AI never invents a component.

**The sentence that never gets cut.** *La IA no puede agregar componentes que no existan en el
proyecto.* Every other clause may be shortened or dropped; this one is what separates archdoc from
Google Code Wiki, DeepWiki, Archify and from simply asking a chatbot. Keep it even in the shortest
form, and never promise code reading in the present tense before it exists.

## The problem it exists for

AI now writes most of the code while a person directs it. That works, and it is fast — but the
understanding that normally forms while typing never forms. Reviewing confirms the result
works; it does not build a mental model. Weeks later the person who directed every decision is
as lost as a stranger would be, with nobody to ask, because nobody ever knew.

> **Documentation debt assumes someone knew and failed to write it down.
> This is knowledge that never existed.**

That sentence is the centre of the project. If one line has to carry the pitch, it is that one.

## Audience

| | |
|---|---|
| **Primary** | Developers and technical leads on systems assembled quickly, often with AI, where nobody holds the whole picture |
| **Secondary** | Anyone inheriting a codebase — new joiners, consultants, maintainers |
| **For this delivery** | An academic panel judging a seminar project. They read the report and watch the presentation |

## Tone

The product's entire pitch is *"it does not make things up."* The voice has to match that or it
undercuts itself.

**Measured, precise, quietly confident.** Claims are specific and checkable. Limits are stated
plainly rather than hidden — the project deliberately publishes a criterion it expects to miss,
and explains why that is a finding rather than a defect.

**Avoid:** hype, superlatives, "revolutionary", "AI-powered" as a headline, exclamation marks,
and any claim the tool cannot demonstrate. A tool about provenance cannot afford a slogan it
cannot cite.

## Visual metaphors the subject actually offers

Two are native to the work rather than borrowed. Both are stronger than the obvious
architecture clichés — blueprints, gears, building blocks — which are generic and say nothing
specific about this tool.

**1 · Cartography, and zoom.** C4's own framing is maps. Its author opens by pinch-zooming
Google Maps from an island to a coastline, and calls the idea *"diagrams as maps."* archdoc
produces exactly this: the same system at two levels of zoom, one model seen from different
distances. Contour lines, map keys, scale bars, survey markers, the moment a shape resolves as
you zoom in.

**2 · Citation, and provenance.** Every element carries the file and line that proves it. That
is closer to a footnote than to engineering — a claim with its source attached. Margin notes,
reference marks, the apparatus of a text that can be checked.

**A third, weaker but true:** *surveying.* The project's research phase was literally a survey
of two production repositories, measuring rather than assuming. Instruments, measurement,
evidence gathered before conclusions.

## The name

**archdoc** — architecture documentation. Lowercase as an identifier: the binary, the module,
the commands. **ArchDoc** in prose and as a display name. The name is plain on purpose; the
project's character is precision, not cleverness.

## What the report needs to cover

Most of it is already written somewhere in this repository. The report is **assembly, not
invention** — each section below names where its material already lives.

| Section | Source |
|---|---|
| Problem statement | `product-definition.md` §0 |
| Related work and standards | `product-definition.md` §14 — C4, arc42, and what each is used for. Also the C4 lecture transcript in this directory |
| Objectives and scope | `product-definition.md` §1–§3 |
| Methodology | `survey-test-subjects.md` — the evidence-first approach, and what reading two real repositories changed |
| Design | `how-it-works.md` — the pipeline, the C4 mapping, one model with many views |
| Technology choices | `stack-decision.md` §5 — the decision, and the three alternatives rejected |
| Implementation | The repository itself; `stack-decision.md` §3 for structure |
| Validation | `product-definition.md` §12 — nine acceptance criteria, written before any code |
| Results and limitations | `PROGRESS.md` for scores; `decisions.md` for why certain things were excluded |
| Conclusions and future work | `product-definition.md` §2 — R1.b, R1.c, R2 |
| Decision record | `decisions.md` — every significant choice, dated, with its reasoning |

**The strongest material for a panel** is not the feature list. It is the findings: a survey
that contradicted an assumption the project rested on, a negative result that reshaped the
extractor, and a criterion published in advance that the tool is expected to miss. Those show
method.

## What the presentation should demonstrate

One thing above all: **the same diagram, generated with the language model switched on, and
then switched off.**

If it is identical in structure and differs only in labels, the central claim is proved in a
single slide — the model interprets facts, it never produces them. That is the demonstration
worth building the talk around.

---

## Keeping this file honest

Everything above is identity or a pointer. If you find yourself wanting to add a count, a date,
or a measured result, it belongs in the file that owns it — and this one should link there
instead. That is what stops it drifting out of date.

`/checkpoint` sweeps this file along with the rest.
