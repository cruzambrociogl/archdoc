# archdoc — características y matriz comparativa

> **Qué es este documento:** el insumo para la *matriz comparativa* de la sección de factibilidad
> funcional del informe. Está en español porque alimenta el informe; el resto de `docs/` está en
> inglés.
>
> **Regla de honestidad:** cada característica está marcada ✅ **implementada** (medida sobre
> Immich, Supabase y Mastodon) o 🔜 **planeada**, con el bloque del plan que la contiene
> (`delivery-schedule.md` §7). En una defensa, afirmar en presente algo que todavía no corre es el
> único error que no se perdona.
>
> Redactado 2026-10-04. Fuentes de la competencia al final.

---

## 1. Características de archdoc

### A. Extracción y evidencia

| | Característica | Estado |
|---|---|---|
| A1 | Lee la configuración que ya describe el sistema: Docker Compose, `.env` y `env_file`, configuración de gateways (Envoy, Kong) y contratos de API | ✅ |
| A2 | **Doble pasada:** `compose-go` resuelve interpolación y las reglas de *merge*; `yaml.v3` recupera archivo y línea, que `compose-go` descarta | ✅ |
| A3 | Descubrimiento: decide cuál de varios archivos Compose es el despliegue real, y registra los que descartó y por qué | ✅ |
| A4 | Sigue los *bind mounts* del propio Compose para encontrar la tabla de rutas de un gateway | ✅ |
| A5 | Lee el código fuente de la aplicación: módulos, imports, rutas, modelos de datos, llamadas salientes | 🔜 Build 1–2 |
| A6 | Descubre *aplicaciones* por sus manifiestos (`package.json`, `pyproject.toml`, `go.mod`), no solo por el Compose | 🔜 Build 1 |
| A7 | Nunca ejecuta nada: no necesita Docker instalado ni el sistema corriendo | ✅ |

### B. Modelo, validación y trazabilidad

| | Característica | Estado |
|---|---|---|
| B1 | **Cada elemento y cada relación cita archivo y línea.** Medido: 100% en los tres sujetos de prueba | ✅ |
| B2 | Trazabilidad por *valor*, no por elemento: el nombre, la descripción, la tecnología y la etiqueta llevan su propia cita | ✅ |
| B3 | Tipos de evidencia: **declarado** (el repo lo define) y **referenciado** (el repo lo nombra). **Inferido no existe**: la herramienta no puede inventar un componente | ✅ |
| B4 | Origen de cada dato: extracción, catálogo, regla humana o modelo de lenguaje | ✅ |
| B5 | Validador con 8 reglas; rechaza el modelo completo antes de escribir nada. Medido: 13 de 13 fallos inyectados rechazados | ✅ |
| B6 | Nunca se persiste un modelo inválido ni parcial | ✅ |
| B7 | Identidad estable de los elementos: un renombrado es un cambio, no un borrado más un alta | ✅ |
| B8 | Cada afirmación en prosa deberá citar la evidencia que la respalda, verificada por el validador | 🔜 Build 2 |

### C. Correcciones humanas que sobreviven

| | Característica | Estado |
|---|---|---|
| C1 | `rules.yaml`: renombrar, reclasificar, describir, excluir, agregar o quitar relaciones | ✅ |
| C2 | Las correcciones se reaplican en cada ejecución desde cero. Medido: 10 de 10 sobreviven a la regeneración | ✅ |
| C3 | Las reglas ganan sobre el modelo de lenguaje: se compilan antes y se aplican después | ✅ |
| C4 | Reporta reglas que no coincidieron con nada, o que pisan a otra | ✅ |

### D. Vistas y diagramas

| | Característica | Estado |
|---|---|---|
| D1 | C4 nivel 1 — contexto del sistema | ✅ |
| D2 | C4 nivel 2 — contenedores, con fronteras de red anidadas y sistemas externos punteados | ✅ |
| D3 | Vista de despliegue: imagen de cada contenedor, puertos publicados, redes y qué monta el repositorio | ✅ |
| D4 | SVG propio con las convenciones C4 (el *layout* lo calcula Graphviz; el dibujo es nuestro) | ✅ |
| D5 | Posiciones guardadas por versión: dos ejecuciones iguales no reacomodan el diagrama | ✅ |
| D6 | Orientación elegida automáticamente para que el diagrama no quede ilegiblemente ancho | ✅ |
| D7 | Mermaid como fuente de texto, portable y *diffeable*, junto al SVG | ✅ |
| D8 | C4 nivel 3 — componentes dentro de cada contenedor | 🔜 Build 2 |
| D9 | Vista dinámica: diagramas de secuencia de un flujo real | 🔜 Build 2 |
| D10 | Modelo de datos: entidades, campos y relaciones | 🔜 Build 2 |
| D11 | Mapa de dependencias de terceros, y lista de funcionalidades por puntos de entrada | 🔜 Build 2 |

### E. Documentación

| | Característica | Estado |
|---|---|---|
| E1 | **arc42 completo en doce secciones**, no solo diagramas | ✅ |
| E2 | Cinco secciones generadas de hechos; siete son del autor | ✅ |
| E3 | **Nunca lee ni escribe las secciones humanas.** Se crean una vez y no se tocan nunca más | ✅ |
| E4 | Los *stubs* preguntan sobre *este* sistema («dos contenedores son alcanzables desde fuera: ¿cuál es el objetivo de disponibilidad de cada uno?»), no un formulario genérico | ✅ |
| E5 | Salida proporcionada al repositorio: seis secciones en vez de doce si no hay con qué llenarlas, y el índice dice qué omitió y por qué | ✅ |
| E6 | Cuatro capas de salida: `model.json`, Markdown + diagramas, sitio estático, aplicación web | ✅ |
| E7 | Sitio estático publicable (`--site` escribe `mkdocs.yml`) | ✅ |
| E8 | **La salida se lee con archdoc desinstalado**, y se versiona en git como texto | ✅ |
| E9 | Una página por componente y más secciones arc42 llenas desde el código | 🔜 Build 2–3 |

### F. Honestidad: decir qué no se vio

| | Característica | Estado |
|---|---|---|
| F1 | **Reporte de cobertura**: qué archivos se leyeron, cuáles se descartaron y por qué, cuánto se sabe, y cada hueco agrupado por regla | ✅ |
| F2 | Reporta las relaciones sin protocolo y los elementos sin descripción, en lugar de rellenarlos | ✅ |
| F3 | Marca en el diagrama lo escrito por el modelo (cursiva), distinto de lo extraído | ✅ |
| F4 | Dibuja un elemento sin conexiones cuando la configuración no las declara, en vez de unirlo adivinando | ✅ |
| F5 | Lista explícita de lo que la configuración **no puede** decir, y qué haría falta para saberlo | ✅ |
| F6 | Tercer estado «no resuelto»: «llama a algo en `${URL}`», citado y visible | 🔜 Build 1 |

### G. Memoria: el sistema a través del tiempo

| | Característica | Estado |
|---|---|---|
| G1 | Historial de versiones en SQLite; una ejecución que no cambia nada no registra nada | ✅ |
| G2 | `model.json` versionado en git como registro legible; la base de datos es solo un caché reconstruible | ✅ |
| G3 | Diff estructural entre versiones: qué apareció, qué desapareció y qué se reescribió, separando lo uno de lo otro | ✅ |
| G4 | Diff arquitectónico entre dos *commits* de git | 🔜 Build 3 |
| G5 | Resumen de cambios por *commit* o por sesión de IA | 🔜 Build 3 |

### H. El papel de la IA

| | Característica | Estado |
|---|---|---|
| H1 | **La IA nunca produce hechos.** Solo nombra, describe y agrupa sobre elementos ya extraídos | ✅ |
| H2 | Es opcional: `--label`. Sin esa bandera no se envía nada y la documentación sale igual | ✅ |
| H3 | Devuelve un diff de operaciones con esquema estricto, nunca un modelo completo | ✅ |
| H4 | El validador acota lo que la IA puede cambiar: no puede excluir elementos ni crear relaciones | ✅ |
| H5 | **Medido: la estructura es idéntica con la IA encendida y apagada.** Solo cambian las palabras | ✅ |
| H6 | Memoria de interpretación: se vuelve a preguntar solo por lo que cambió | 🔜 Build 1 |
| H7 | Modelo local (Ollama) para operación 100% privada | 🔜 flexible |

### I. Privacidad y control de egress

| | Característica | Estado |
|---|---|---|
| I1 | Por defecto sale **solo estructura**: nombres, tipos y relaciones. **Cero contenido de archivos** | ✅ |
| I2 | Registro byte a byte de cada petición, tal como salió al cable, incluyendo las que fallaron | ✅ |
| I3 | Coste y tokens por ejecución; el coste solo se muestra si el precio del modelo se conoce | ✅ |
| I4 | Un único paquete puede hacer llamadas de red; lo verifica la integración continua, no una revisión manual | ✅ |
| I5 | El código nunca tiene que salir de la máquina; nada se sube a un servicio | ✅ |

### J. Superficies

| | Característica | Estado |
|---|---|---|
| J1 | CLI: `scan`, `generate`, `history`, `runs`, `serve` | ✅ |
| J2 | Aplicación web local (`archdoc serve`) con siete vistas: diagrama, inspector, historial, reglas, documentos, completitud y ejecuciones de red | ✅ |
| J3 | El diagrama del navegador es **el mismo SVG** que se versiona en el repositorio | ✅ |
| J4 | Clic en un elemento → el archivo y la línea se abren en el editor (`vscode://`) | ✅ |
| J5 | Vista de completitud: estado de cada sección humana **sin abrir el archivo** (tamaño y fecha) | ✅ |
| J6 | Búsqueda, zoom y foco para repositorios de cientos de elementos | 🔜 Build 3 |
| J7 | Preguntas y respuestas sobre el modelo, y acceso para agentes (MCP) | 🔜 fuera de esta fase |

### K. Operación

| | Característica | Estado |
|---|---|---|
| K1 | **Un solo binario**, sin dependencias en tiempo de ejecución; el usuario no instala Node ni Python | ✅ |
| K2 | **Cero servicios**: un proceso al servir, ninguno el resto del tiempo. Sin base de datos, sin contenedores | ✅ |
| K3 | **Determinismo absoluto**: cinco ejecuciones idénticas producen documentos byte a byte iguales | ✅ |
| K4 | Degrada con aviso en vez de caer: si el *layout* falla, los documentos salen con Mermaid | ✅ |
| K5 | Multiplataforma (macOS, Linux, Windows), sin interfaz gráfica obligatoria | ✅ |

---

## 2. Matriz comparativa

Columnas: **AD** archdoc · **AF** Archify · **GD** GitDiagram · **DW** DeepWiki · **CW** Google Code
Wiki · **IA‑D** draw.io / Lucidchart / FigJam / Eraser DiagramGPT · **ST** Structurizr · **CV**
docker-compose-viz / docker-diagrams.

| Criterio | AD | AF | GD | DW | CW | IA‑D | ST | CV |
|---|---|---|---|---|---|---|---|---|
| **Quién decide qué aparece en el diagrama** | un *parser* | el agente de IA | un LLM | un LLM | un LLM | la persona + IA | la persona (DSL) | un *parser* |
| **Cada elemento cita archivo y línea** | sí, obligatorio | opcional, la elige el agente | enlaza al archivo | cita archivos, las elige el modelo | cita archivos, las elige el modelo | no | no (el DSL es la fuente) | no |
| **¿Puede aparecer algo que no existe?** | **imposible por diseño** | sí | sí | sí | sí | sí | sí (si lo escribe el autor) | no |
| **Salida byte a byte idéntica entre ejecuciones** | sí (criterio de aceptación) | no | no | no | no | no | sí | sí |
| **Validación que rechaza un modelo inválido** | sí, 8 reglas, todo o nada | valida el esquema del diagrama | no | no | no | no | valida el DSL | no |
| **Correcciones humanas que sobreviven a regenerar** | sí (`rules.yaml`) | se reescribe el IR | no | no | no | se pierde al regenerar | el DSL *es* la corrección | no |
| **Documentación estructurada además del diagrama** | arc42 de 12 secciones | no | explicación en prosa | wiki | wiki | no | vistas + documentación ADR | no |
| **Niveles C4** | 1, 2, despliegue (3 y dinámica planeadas) | libre, no C4 | uno, libre | libre | libre | libre | 1–3 + despliegue y dinámica | uno |
| **Respeta lo que escribió el humano** | **nunca lee ni escribe sus secciones** | n/a | n/a | se regenera todo | se regenera todo | n/a | el humano escribe todo | n/a |
| **Dice qué no pudo ver (cobertura)** | **sí, página dedicada** | no | no | no | no | no | no | no |
| **Distingue probado / interpretado** | sí, visible | no | no | no | no | no | n/a | n/a |
| **Historial y diff arquitectónico** | sí, por versión (por *commit* planeado) | comparación antes/después | no | se actualiza, sin diff | se actualiza por *commit*, sin diff estructural | no | vía git del DSL | no |
| **Funciona con repositorios privados sin subir código** | sí | sí (local) | no (repos públicos) | no | no | n/a | sí | sí |
| **Opera 100% sin red** | sí (la IA es opcional) | no (el agente usa red) | no | no | no | no | sí | sí |
| **Registro de qué salió de la máquina** | **sí, byte a byte** | no | no | no | no | no | n/a | n/a |
| **La documentación vive en tu repositorio** | sí, texto versionado | archivo HTML | enlace del servicio | en su sitio | en su sitio | archivo de la herramienta | sí, el DSL | imagen |
| **Lee la configuración de despliegue real (Compose, gateways)** | sí, y es su base | solo lo que el agente leyó | metadatos y árbol de archivos | sí, con un LLM | sí, con un LLM | no | no, se modela a mano | sí, solo Compose |
| **Esfuerzo del usuario** | un comando | describir el sistema al agente | pegar una URL | pegar una URL | pegar una URL | dibujar o *promptear* | **escribir y mantener el modelo** | un comando |
| **Requisitos** | un binario | Node + un agente de IA | servicio web | servicio web | servicio web | cuenta en el servicio | Java / Docker | PHP o Python |
| **Coste** | el de la API si se usa `--label`; cero sin ella | el del agente | gratis (hospedado) | gratis / de pago | gratis en *preview* | suscripción | gratis (Lite) / de pago (cloud) | gratis |

---

## 3. Lectura por categoría

**a) Generadores de diagramas con IA (Archify, GitDiagram).** Dibujan bien y cubren cualquier
repositorio, porque nada de lo que dibujan tiene que ser verdad: el diagrama lo redacta un modelo.
Archify incluso concluyó que «el *layout* es el producto» y deja que el modelo coloque cada caja.
Son más atractivos y más rápidos de adoptar; no tienen mecanismo alguno que impida una relación
inventada, y GitDiagram trabaja además sobre metadatos y árbol de archivos, no sobre el despliegue
declarado.

**b) Plataformas de documentación con IA (DeepWiki, Google Code Wiki).** Son los competidores
reales y son muy buenos: cualquier lenguaje, cero configuración, actualización en cada *commit* y
un chat encima. Code Wiki enlaza sus explicaciones a archivos; pero **el modelo elige esas citas y
nada comprueba que lo descrito exista**, la wiki vive en su sitio y no en el repositorio, y se
regenera por completo, de modo que no hay noción de «esta sección la escribí yo».

**c) Herramientas de diagramación asistida (draw.io, Lucidchart, FigJam, Eraser DiagramGPT).** El
punto de partida de este proyecto fue exactamente este experimento: una especificación detallada
entregada al asistente de draw.io. Falló de dos maneras — todos los hechos tenían que salir de la
memoria de una persona, y la herramienta además perdió y malinterpretó cosas que *sí* estaban
escritas. No son generadores: son editores. El diagrama es la fuente de verdad, y por eso envejece.

**d) Determinísticas y «arquitectura como código» (Structurizr, visualizadores de Compose).**
Comparten con archdoc lo que más importa: determinismo y salida versionable. Structurizr es el
referente de C4 y llega más lejos en vistas, pero **el modelo lo escribe y lo mantiene una
persona**: no lee el repositorio, así que puede divergir del código sin que nada lo note. Los
visualizadores de Compose sí leen la configuración, pero producen un grafo, no documentación: sin
arc42, sin trazabilidad a línea, sin historial, sin correcciones persistentes.

---

## 4. Lo que diferencia a archdoc — cinco puntos

1. **Nada de lo que aparece puede estar inventado.** La topología la produce un *parser* y el
   validador acota lo que la IA puede tocar. Es la única de la lista que puede afirmarlo como
   propiedad del diseño y no como buena intención.
2. **Todo se puede comprobar en un clic:** archivo y línea por cada elemento *y por cada valor*.
   Para quien lee código es un enlace; para quien no, es la garantía de que nada se inventó.
3. **Dice dónde termina su conocimiento.** El reporte de cobertura publica qué no vio y qué la
   configuración no puede decir. Ninguna alternativa publica sus propios huecos.
4. **La documentación es tuya y se queda contigo:** texto versionado en tu repositorio, legible con
   archdoc desinstalado, con tus secciones intactas y tus correcciones sobreviviendo a cada
   regeneración. Las plataformas hospedadas te dan una wiki que vive en su sitio.
5. **Privado por defecto, y demostrable:** sin la bandera no sale nada; con ella sale solo
   estructura, y cada byte queda registrado. Un modelo local lo cierra por completo.

Y uno que no es una característica sino un resultado medido: **la estructura es idéntica con la IA
encendida y apagada**. Es la tesis del proyecto y ninguna herramienta de las categorías (a) y (b)
puede siquiera plantearla.

## 5. Dónde la competencia es mejor — para decirlo antes de que lo pregunten

| Alternativa | Dónde gana |
|---|---|
| Code Wiki, DeepWiki | Cobertura: cualquier lenguaje hoy, sin configuración, gratis y con chat. archdoc empieza por Compose y añade lenguajes de a uno |
| Archify, GitDiagram | Acabado visual y narrativa: animación, temas, exportaciones, vídeo explicativo |
| Structurizr | Madurez de vistas C4, ecosistema y comunidad |
| Visualizadores de Compose | Simplicidad extrema para una sola imagen |
| Todas las hospedadas | Cero instalación y un enlace para compartir |

La respuesta de archdoc no es ganarles en amplitud, sino ser **la alternativa local y verificable**:
las mismas preguntas respondidas sobre una base que ellas no pueden ofrecer.

## 6. Una línea

> archdoc convierte un repositorio en documentación de arquitectura verificable: cada elemento
> señala el archivo y la línea que lo prueban, la IA nunca inventa componentes, y la documentación
> vive en tu repositorio y no en el sitio web de otra empresa.

---

## Fuentes

- Google Code Wiki — [blog de Google Developers](https://developers.googleblog.com/introducing-code-wiki-accelerating-your-code-understanding/), [DevOps.com](https://devops.com/google-code-wiki-aims-to-solve-documentations-oldest-problem/)
- Archify — [tt-a1i/archify](https://github.com/tt-a1i/archify)
- GitDiagram — [ahmedkhaleel2004/gitdiagram](https://github.com/ahmedkhaleel2004/gitdiagram)
- Structurizr — [structurizr.com](https://structurizr.com/)
- Mediciones propias de archdoc: `PROGRESS.md` (criterios de aceptación) y `docs/survey-test-subjects.md` (sujetos de prueba y revisiones fijadas)
