# Proyecto de Seminario — Notas de Ideación

> Documento de traspaso. Contiene la idea principal seleccionada, sus argumentos de
> defensa, la propuesta técnica, y el registro de ideas descartadas o pausadas.

---

## 1. Criterios de selección (filtro del proyecto)

Después de explorar múltiples direcciones, el filtro quedó definido así:

1. **No tiene que ser útil ni socialmente significativo.** Puede ser una herramienta,
   un servicio, un juego, o algo "aburrido".
2. **Sí tiene que ser algo que yo usaría** — no por utilidad objetiva, sino porque
   necesito que me importe lo suficiente para terminarlo.
3. **No puede ser algo que una IA resuelva con un prompt.** Si un modelo frontera lo
   hace de una sola pasada, es un *wrapper*, no un proyecto.
4. Si ya existe una herramienta similar, hay que **agregar valor real**, no replicar.

**Consecuencia técnica del criterio 3:** la dificultad tiene que vivir donde un modelo
no llega — latencia, corrección garantizada, estado de largo plazo, escala,
concurrencia, o un modelo entrenado con datos propios.

---

## 2. IDEA PRINCIPAL (seleccionada)

### Documentación y diagramas asistidos por IA para la comprensión de sistemas de software

#### Texto de una página

**Contexto.** La generación de código dejó de ser el cuello de botella del desarrollo.
Con cerca del 85% de los desarrolladores usando herramientas de IA a diario y
aproximadamente el 46% del código nuevo generado por IA, hoy se produce software más
rápido de lo que un equipo humano puede leer, verificar y comprender. La capacidad de
generación escaló; la capacidad de comprensión no. El resultado es una asimetría
creciente: sistemas que funcionan, pero que nadie termina de entender.

A esto se suma un fenómeno más silencioso. Stack Overflow funcionó durante quince años
como un archivo involuntario del *razonamiento* detrás del código: los intentos
fallidos, las alternativas descartadas, las razones de cada decisión. Ese archivo se
está apagando: el volumen mensual de preguntas cayó desde un pico cercano a 200,000 en
2014 hasta menos de 50,000 a finales de 2025 —una caída superior al 75%, a niveles no
vistos desde 2009— mientras la comunidad de quienes respondían se vació en paralelo.
Los problemas resueltos entre 2024 y 2026 ya no se documentan públicamente a esa
escala. Dentro de las organizaciones ocurre lo mismo: el razonamiento sucede en una
sesión de chat efímera, el código llega a producción y la conversación se descarta.
Git registra *qué* cambió, nunca *por qué*.

**Propuesta.** Este proyecto plantea una herramienta que toma como entrada el
repositorio de código, los planes y especificaciones del proyecto, y una conversación
de refinamiento con el usuario, para generar automáticamente diagramas de arquitectura
y documentación comprensible para humanos. El objetivo no es sustituir la
documentación manual, sino cerrar la brecha entre la velocidad a la que se escribe el
código y la velocidad a la que un equipo puede entenderlo.

**Alcance.** Para mantener el proyecto acotado y verificable, la implementación se
limita a un repositorio local. El sistema realiza análisis estático del código para
extraer una estructura verificable —módulos, dependencias, puntos de entrada, accesos
a datos— y utiliza un modelo de lenguaje únicamente para la capa que el análisis
estático no puede resolver: agrupar, abstraer, nombrar e interpretar la intención.
Cada elemento del diagrama conserva su trazabilidad al archivo del que proviene, de
modo que la herramienta no puede inventar componentes inexistentes. El usuario refina
el resultado mediante conversación, y esas correcciones se conservan como reglas que
sobreviven a la regeneración.

**Trabajo futuro.** La dirección natural de la herramienta es preservar el
razonamiento, no solo la estructura: capturar las decisiones de diseño y sus
alternativas durante el desarrollo, vincularlas a las partes del sistema que afectan,
y detectar desviaciones arquitectónicas entre versiones. Ese es un problema de mayor
alcance que excede este proyecto, pero define la trayectoria de la propuesta.

**Aporte.** El código nunca fue el conocimiento; el conocimiento era la conversación
que lo produjo, y esa conversación hoy se está borrando. Este proyecto convierte esa
conversación en un artefacto estructurado, verificable y trazable.

---

## 3. Argumentos de defensa

### Argumento 1 — La asimetría de velocidad
El cuello de botella se movió. La generación escaló; la comprensión no.
~85% de desarrolladores usan IA a diario, ~46% del código nuevo es generado por IA,
pero la capacidad humana de leer y verificar no mejoró nada. Cada avance en generación
ensancha la brecha. La herramienta vive del lado que dejó de escalar.

### Argumento 2 — Evaporación del conocimiento (el argumento más fuerte)
Stack Overflow era un archivo involuntario de *razonamiento*, no solo de respuestas.

Datos:
- Pico ~200,000 preguntas mensuales en 2014.
- Menos de 50,000 a finales de 2025 (caída >75%, niveles de 2009).
- Datos de inicio de 2026: ~300 mensuales, colapso cercano al 99% desde el pico.
- La comunidad de *answerers* se vació en paralelo.

La consecuencia clave: la IA se entrenó con los años dorados (2008–2020), pero los
problemas de 2024–2026 ya no se documentan públicamente a esa escala — **el grafo de
conocimiento deja de crecer**.

A nivel organizacional: Git registra *qué* cambió, nunca *por qué*. El "por qué" vivía
en hilos de SO, discusiones de PR, y la memoria de un colega. Ahora vive en una sesión
de chat que se descarta.

> **El commit sobrevive; el razonamiento se evapora.**

El *bus factor* empeora: no es que una persona sabía y se fue — es que **nadie lo supo
nunca**.

### Argumento 3 — Por qué ahora
El problema no existía hace tres años. Es consecuencia directa, fechada y medible de un
cambio posterior a 2022. Responde la pregunta real del jurado: "¿por qué vale la pena
estudiar esto ahora?"

### Objeciones anticipadas

| Objeción | Respuesta |
|---|---|
| "¿No lo hace ya Doxygen / Swimm / Mintlify?" | Documentan *qué es* el código, desde la sintaxis. No expresan el *por qué* y no consumen specs, planes ni chat, que es donde vive la intención. Además operan a nivel de función, no de arquitectura. |
| "¿Por qué no pedirle la documentación a Claude Code?" | One-shot, no verificable, sin *ground truth*, sin persistencia, obsoleto al día siguiente. El aporte es el *pipeline*: hechos del análisis estático, significado del modelo, procedencia en cada elemento, regeneración por commit. El LLM es un componente, no el producto. |
| "¿La IA no resolverá esto sola?" | La comprensión no es delegable. El humano sigue siendo responsable de decisiones de arquitectura, auditorías y onboarding. No se puede firmar un sistema que no se entiende. |
| "¿El alcance no es muy grande?" | MVP: un repositorio, un nivel de abstracción, diagrama de arquitectura + trazabilidad. Todo lo demás es roadmap. |

### Frase de cierre
> "El código nunca fue el conocimiento. El conocimiento era la conversación que lo
> produjo — y esa conversación ahora se está borrando."

---

## 4. Propuesta técnica

### Principio rector
**El diagrama nunca es la fuente de verdad — lo es una especificación estructurada.**
Y: **el LLM nunca es responsable de los hechos.**

- Herramientas mecánicas (grafos de dependencias, UML de IDE, Doxygen) → precisas pero
  ilegibles: ven sintaxis, no intención.
- LLM one-shot → legible pero posiblemente incorrecto, sin forma de verificar.
- **Este sistema:** hechos del compilador, significado del modelo.

### Ciclo principal

```
Usuario (texto/voz)
  → Orquestador backend
  → Análisis estático (ground truth)
  → RAG (patrones de arquitectura)
  → LLM devuelve un DIFF ESTRUCTURADO, no una imagen
  → Validador (schema + integridad referencial, loop de reparación)
  → Actualiza el grafo canónico en DB
  → Push al frontend → render
  → Usuario edita (chat O canvas) → vuelve al grafo
```

### Representación del diagrama

**Decisión recomendada:** grafo JSON propio como canónico, Mermaid como formato de
exportación.

```json
{
  "nodes": [{ "id": "", "type": "", "label": "", "tech": "", "meta": {} }],
  "edges": [{ "from": "", "to": "", "label": "", "protocol": "" }]
}
```

- **Mermaid** — el LLM ya conoce la sintaxis, compacto, versionable. Poco control de
  layout y metadata.
- **JSON propio** — más trabajo, pero da validación propia, metadata arbitraria
  (protocolos, almacenes, fronteras) y exportación a Mermaid / PlantUML / draw.io XML.

### Stack

| Capa | Opciones |
|---|---|
| Análisis estático | tree-sitter, LSP (AST, imports, call edges, rutas, accesos a DB) |
| Render | **React Flow** (editable, nodos/edges 1:1 con el JSON) + `dagre`/`elkjs` para layout; Mermaid.js si solo lectura; Cytoscape.js alternativa |
| LLM | API (Anthropic/OpenAI/Gemini) u **Ollama** local. Usar **structured outputs / JSON schema** y **tool calling** (`add_component`, `connect`, `rename`, `remove`, `group_into_boundary`) |
| RAG | Embeddings → **pgvector** / Qdrant / Chroma. Corpus: catálogos de patrones, modelo C4, arquitecturas de referencia cloud. LangChain / LlamaIndex o retriever propio |
| Voz (opcional) | **Whisper** (`faster-whisper` o API), o Web Speech API. El transcript entra al mismo pipeline — nunca un code path separado |
| Backend | **FastAPI** (Python) o Node/NestJS (TS). **WebSockets** para streaming del diagrama en vivo |
| Persistencia | PostgreSQL + **historial de versiones** del grafo (cada diff → undo, time-travel, evolución del diseño) |

### Dónde vive la profundidad de ingeniería (decir esto en la defensa)

1. **Diffs incrementales, no regeneración.** El modelo emite *operaciones* contra el
   grafo existente. Regenerar todo pierde el layout, es lento e inestable.
2. **Loop validar → reparar.** Validación de schema, integridad referencial (sin edges
   a nodos inexistentes), reglas de ciclos. Si es inválido, se retroalimenta el error
   al modelo y se reintenta. El modelo *no puede* emitir un diagrama roto.
3. **Sincronización bidireccional.** Una edición en el canvas actualiza el grafo *y* el
   contexto de la conversación, para que el modelo no contradiga lo que el usuario
   acaba de cambiar.
4. **Estabilidad de layout.** Al agregar un nodo, el resto no debe saltar. Posiciones
   sticky + layout incremental.
5. **Correcciones persistentes como reglas** (ej. "tratar `/utils` como
   infraestructura, no como componente") que sobreviven a la regeneración.

### Features derivadas de la arquitectura (no "prompteables")

1. **Procedencia en cada elemento** — cada nodo/edge carga los archivos y líneas de
   origen. Click en una caja → salta al código. No hay componentes alucinados.
2. **Niveles de abstracción (estilo C4)** — contexto → contenedor → componente →
   código. La ingeniería dura es el *clustering semántico*: convertir 400 archivos en
   8 cajas con sentido.
3. **Se mantiene vivo** — re-ejecución por commit y **diff arquitectónico**: "este PR
   agregó una llamada directa de la capa de API a la base de datos, saltándose la capa
   de servicio". Detección de deriva arquitectónica.

### Plan de evaluación (esto defiende muy bien)
- % de elementos del diagrama trazables a código real.
- Precisión contra una arquitectura de referencia dibujada a mano.
- Tiempo que tarda un desarrollador nuevo en responder preguntas sobre un repo
  desconocido, con y sin la herramienta.

### Pendientes por definir
- Qué tipos de diagrama cubre (recomendación: acotar a arquitectura de software, no
  "cualquier diagrama").
- Si la voz es central o una capa opcional sobre el flujo de texto
  (recomendación: texto como columna vertebral, voz como capa delgada).

---

## 5. Otras ideas (pausadas / descartadas)

### 5.1 En pausa — Plataforma de monitoreo de salud de cultivos

Estuvo cerca de ser la elegida. Se pausó porque es un *servicio para otros*, no una
herramienta de uso propio, y su ciclo de prueba es lento (recolectar imágenes de campo,
etiquetar, reentrenar, encontrar agricultores).

**Texto de tres párrafos:**

> **Plataforma para el monitoreo de la salud de cultivos y asesoría agronómica**
>
> En Guatemala, el maíz y el café sostienen tanto la seguridad alimentaria familiar
> como la economía rural; sin embargo, enfermedades como la roya y los tizones foliares
> provocan pérdidas recurrentes. Para la mayoría de productores el problema no es
> reconocer que una planta está enferma —muchos identifican los síntomas por
> experiencia—, sino saber *qué tan extendida está la infección, con qué velocidad
> avanza y qué acción se justifica.* La inspección de campo es subjetiva, no queda
> documentada y rara vez se repite de forma consistente, por lo que las decisiones se
> toman con base en impresiones y no en evidencia.
>
> Este proyecto propone una plataforma que permite a una familia o empresa monitorear
> de forma continua sus propias parcelas, a través de dos superficies conectadas. En
> campo, un asistente conversacional dirige un muestreo estructurado: genera una ruta
> con puntos definidos según el tamaño y la forma de la parcela, guía al productor a
> través de ellos y solicita fotografías en cada uno, vinculando cada imagen a su
> punto, parcela y fecha. El sistema devuelve un diagnóstico con un nivel de severidad
> objetivo y explica el tratamiento recomendado en lenguaje sencillo, con soporte de
> voz para usuarios con alfabetización limitada. Esa información alimenta un centro de
> mando web, donde el propietario o el técnico visualiza las parcelas sobre un mapa, la
> incidencia y severidad por zona, la evolución de cada ronda de muestreo, la dirección
> y velocidad de expansión de la enfermedad y las alertas cuando se superan los
> umbrales.
>
> El cultivo piloto se definirá en la fase inicial —maíz y café son los candidatos, por
> su relevancia económica y alimentaria y la disponibilidad de datos—, y el sistema se
> diseñará de forma agnóstica al cultivo. Técnicamente combina un modelo de visión por
> computadora para clasificación y estimación de severidad, una capa de inteligencia
> artificial generativa fundamentada en documentación agronómica institucional, y una
> aplicación web para visualización y alertas. El aporte no radica en identificar
> enfermedades, sino en convertir una inspección subjetiva en un registro cuantitativo,
> geolocalizado y trazable que respalde decisiones oportunas.

**Datos relevantes encontrados:**
- **Café:** JMuBEN (~58,555 imágenes, 5 clases incl. roya, cercospora, phoma);
  RoCoLe (fotos de campo con severidad de roya en niveles 1–4).
- **Maíz:** subconjunto PlantVillage (3,852 imágenes, 4 clases: gray leaf spot, common
  rust, northern leaf blight, healthy). *Caveat:* no cubre mancha de asfalto ni
  gusano cogollero, que son los problemas grandes en Guatemala.
- **Frijol:** iBean de Makerere (~1,296 imágenes de campo, 3 clases), incluido en
  TensorFlow Datasets — el arranque más rápido.

**Competencia identificada (importante):**
- **CoffeeCloud (ANACAFÉ, Guatemala)** — app nacional para café con vigilancia de roya,
  broca y ojo de gallo, muestreos, y portal web. **No tiene visión por computadora**:
  el productor llena un cuestionario manual. Ese es el hueco. Su existencia además
  *valida* la premisa (una institución nacional construyó su sistema de alerta sobre
  incidencia y severidad de muestreos estructurados).
- **Plantix** — diagnóstico global por foto, ~800 síntomas, 60 cultivos, 10M+ descargas,
  imágenes geoetiquetadas. Digital Green lo integró a un chatbot de WhatsApp.
- **Plataformas comerciales de scouting** (OneSoil, GeoPard, xarvio, EOSDA) — para
  fincas grandes mecanizadas, dirigidas por satélite/NDVI.

**Conclusión estratégica:** para café habría que diferenciarse de un incumbente
institucional gratuito y desplegado nacionalmente; para maíz no existe equivalente en
Guatemala (espacio libre).

### 5.2 Descartadas por el filtro "una IA ya lo hace"
- Notas/portapapeles inteligente con auto-clasificación y *resurfacing*.
- Sincronizador del horario y tareas de la universidad hacia Google Calendar con
  recordatorios inteligentes y detección de conflictos con el trabajo.
- Hub personal unificado (notas, deadlines, calendario, portapapeles).
- Foto de pizarrón → notas estructuradas.
- Grabación de pantalla → documentación.
- Recibos → datos de gasto estructurados.
- Lectura de labios (VSR) — descartada por complejidad y precisión insuficiente.
- LENSEGUA (lengua de señas de Guatemala) — buena idea, requiere construir el dataset.

---

## 6. Vetas alternativas no exploradas a fondo

Por si hay que pivotar. Todas pasan el filtro por construcción (la dificultad no es
lingüística sino de latencia, corrección, estado, escala o concurrencia).

### Infraestructura para agentes de IA (la veta más fuerte encontrada)
El patrón: los modelos escalaron rápido, la infraestructura para hacerlos seguros,
testeables, observables y económicos no.

1. **Seguridad de agentes** — OpenClaw acumuló 280+ advisories y 100+ vulnerabilidades
   en poco tiempo; claves filtradas, agentes secuestrados, skills maliciosas. El
   problema núcleo: el agente no distingue un comando legítimo de un prompt malicioso
   incrustado en una web o documento.
   → firewall de egress, analizador estático de skills, sandbox real.
2. **Observabilidad** — no existe forma consistente e independiente del proveedor de
   revisar planes, tool calls, aprobaciones y diffs.
   → debugger de trazas con replay.
3. **Fricción de setup** — instalar, API key, skills, configuración; peor con modelos
   locales.
4. **Confiabilidad en tareas largas** — sin checkpoint/resume; con 85% de fiabilidad
   por paso, un flujo de 10 pasos termina ~20% de las veces.
   → capa de ejecución durable.
5. **Tool poisoning en MCP** — las descripciones de tools son lenguaje natural que el
   agente lee como contexto; nada impide que un server devuelva lo que quiera. Un
   escaneo encontró 1,862 servers MCP expuestos; de 119 verificados, los 119 permitían
   listar tools sin autenticación.
   → scanner/auditor de MCP, gateway que sanitiza, registry firmado.
6. **Evaluación de agentes** — es manual; un agente puede razonar bien, elegir mal la
   tool, producir salida plausible y fallar en silencio.
   → harness de regresión, diagnóstico de fallos por traza.
7. **Memoria** — deriva entre sesiones, invisible sin evaluación longitudinal.
8. **Costo** — falta gobernanza de costo por tarea.
9. **Los fixes no se propagan** — la corrección queda en el config personal de quien la
   hizo.
10. **Human-in-the-loop** — no hay capa genérica de aprobación.
11. Coordinación multi-agente sin métricas; gestión de contexto; ergonomía de modelos
    locales.

### Otras vetas técnicas
- **Emuladores / bajo nivel:** CHIP-8, Game Boy, mini-git, mini-Docker, motor de base de
  datos, ray tracer.
- **Compiladores / lenguajes:** lenguaje propio, sistema de álgebra simbólica,
  transpilador, lenguaje de consulta para el historial de git o un codebase.
- **Simulación en tiempo real:** motor de física, fluidos, terreno procedural, n-body.
- **Juegos con ingeniería real:** netcode con rollback, motor de ajedrez propio
  (minimax/MCTS), agente de RL que aprende a jugar.
- **Audio / DSP:** sintetizador desde cero, fingerprinting estilo Shazam, detector de
  acordes por FFT, separación de fuentes.
- **Redes / distribuido:** P2P estilo BitTorrent, key-value store con Raft, editor
  colaborativo con CRDTs.
- **Modelo propio entrenado:** red neuronal desde cero con backprop, algoritmos
  genéticos, vida artificial.
- **Criptografía / seguridad:** esteganografía, blockchain propia, honeypot.
- **Parsers de formatos:** PNG, ZIP, MIDI, fuentes tipográficas, QR desde el estándar.

---

## 7. Método para desbloquear ideas (si vuelve a hacer falta)

El bloqueo no es de creatividad, es de **recuperación**. No se pueden recordar
fricciones sin un disparador enfrente.

**Método correcto:** durante 3–4 días, anotar en el teléfono cada vez que aparezca el
pensamiento *"otra vez esto"* o *"¿por qué estoy haciendo esto a mano?"*. Al final de
la semana hay 8–15 fricciones reales de donde elegir. Registrar en vez de recordar.

**Disparadores concretos** (más efectivos que "¿qué te molesta?"):
- ¿Qué hice ayer en el trabajo, paso a paso?
- ¿Cuál fue la última respuesta mediocre de una IA que tuve que arreglar a mano?
- ¿Qué pestañas tengo abiertas ahora?
- ¿Qué herramienta uso y detesto pero no he reemplazado?
