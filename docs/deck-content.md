# ArchDoc — contenido del deck (revisión de avance, 9 de octubre de 2026)

> **Qué es este archivo:** el contenido completo para reconstruir la presentación en Claude Design.
> Sustituye al deck publicado, que quedó en el estado del 14 de septiembre.
>
> **Estado al 7 de octubre de 2026.** Los números salen de `PROGRESS.md`, `docs/decisions.md`,
> `docs/vision.md` y `docs/delivery-schedule.md` §7. Es una foto: si algo cambia antes del 9, se
> corrige allí primero y luego aquí.
>
> **Sistema visual:** se mantiene el del deck actual (Spectral, Public Sans, JetBrains Mono; fondo
> claro `#FBFCFC`, tinta `#15201E`, magenta `#A31F62` para etiquetas de sección; diapositivas
> oscuras para los tres momentos de énfasis). 1920 × 1080. Pie: logotipo y «NN / 16».

## Qué cambió respecto al deck anterior

| Antes (14 Sep) | Ahora (7 Oct) |
|---|---|
| El código como evidencia era «la versión siguiente» | **Está construido**: componentes, rutas, datos y flujos salen del código, medidos sobre Immich |
| Immich: cinco cajas, machine learning sin conectar | Immich: 5 contenedores, 43 componentes, 301 rutas, 68 tablas, y la arista servidor → ML recuperada en su línea |
| Referencias: DeepWiki y Archify | Aparece **Google Code Wiki**, el competidor directo. Posición: *la alternativa local y verificable* |
| Cinco perfiles, todos por igual | Audiencia principal decidida: **lectores técnicos que pierden control del código que escribió una IA** |
| «No resuelto» era una regla propuesta | Es un estado real: 11 llamadas no resueltas en Immich, visibles y citadas |
| Privacidad: una regla por escribir | Decidida el 6 Oct: **salen nombres, nunca texto** |
| Diecisiete decisiones, ninguna tomada | Dirección cerrada el 2 Oct; el resto se confirma del 13 al 17 Oct |
| Versión 1 el 9 de octubre | El 9 Oct es **revisión de avance**. Entrega: **11 de diciembre** |
| La app web: siete vistas sobre un SVG | Superficie rediseñada: una app, dos modos, y un sitio publicable |

## Por confirmar antes de presentar

1. **Criterios cumplidos: ¿seis o siete?** La tabla de `PROGRESS.md` marca seis (AC-1, 2, 4, 5, 7, 8);
   `vision.md` y `delivery-schedule.md` dicen siete. La diapositiva 07 usa la tabla. Si AC-9 se
   midió esta semana (estaba agendado del 5 al 8 Oct), pasa a siete.
2. **Corrida en vivo de las explicaciones.** Las páginas por componente con afirmaciones citadas
   están construidas y probadas con un modelo falso; no hay corrida real todavía. Las diapositivas
   10 y 12 lo dicen así. Si se hace antes del 9, se actualizan.
3. **`docs/comparison-matrix.md` quedó atrás:** todavía marca la lectura de código como planeada.
   No afecta al deck, pero sí al informe.
4. **Terminología:** las diapositivas y las notas dicen «Versión 1». No usar «R1.a» en voz alta.

---

## 01 · Portada

**Fondo claro, con la imagen de portada actual.**

- Universidad Galileo
- FISSIC · Seminario Profesional 2
- **ArchDoc**
- *Entender el software que una IA construyó*
- Presentan: Cruz Ambrocio – 20005588 · Luis Guzman – 21000542
- Asesor: Axel Benavides
- Guatemala, 2026 · Revisión de avance, 9 de octubre

**Notas.** Buenas tardes. Somos Cruz Ambrocio y Luis Guzman. ArchDoc genera la documentación de
arquitectura que un proyecto debió tener, extraída del repositorio y trazable línea por línea. Hoy
es la revisión de avance: qué está construido, qué aprendimos, y qué falta hasta la entrega del 11
de diciembre.

---

## 02 · Agenda

**Etiqueta:** Agenda

1. El problema y para quién
2. El panorama, y nuestra posición
3. Versión 1, y su techo
4. El código como evidencia
5. Qué lo hace confiable
6. La prueba que falta, y el plan

**Notas.** Seis partes. El problema y para quién es. Luego el panorama, que cambió desde la última
vez. Después lo construido: la versión 1, el límite que encontramos, y cómo lo quitamos. Cierro con
las garantías, el experimento que todavía falta, y el calendario.

---

## 03 · El problema

**Diapositiva oscura.** **Etiqueta:** 01 — El problema

**Titular:** La IA escribe código más rápido de lo que una persona puede leerlo. La comprensión que
antes se formaba al escribirlo ya no se forma.

**Cita destacada:** La deuda de documentación supone que alguien sabía y no lo escribió. Esto es
conocimiento que nunca existió.

**Cierre:** ArchDoc existe para devolver esa comprensión.

**Notas.** La IA escribe código más rápido de lo que cualquiera puede leerlo. Revisar confirma que
funciona; no construye un modelo mental. Semanas después, quien dirigió cada decisión está tan
perdido como un extraño, y no hay a quién preguntar porque nadie lo supo nunca. Esa es la frase que
sostiene el proyecto: la deuda de documentación supone que alguien sabía. Esto es distinto.

---

## 04 · Para quién

**Etiqueta:** 01 — Para quién

**Titular:** Cuánto sabe cada persona de lo que se construyó

| Perfil | Qué sabía del plan | Qué puede leer del código | Qué necesita de ArchDoc |
|---|---|---|---|
| Experto que usa IA | Diseño propio y completo | Lee todo, pero genera más de lo que alcanza a revisar | Velocidad: qué cambió y un mapa para navegarlo |
| Equipo con plan | Diseño escrito y acordado | Revisa el código, pero no todo lo que la IA generó | Confirmar que el código sigue coincidiendo con el diseño |
| Desarrollador con planes | Planes por funcionalidad, sin diseño del conjunto | Lee lo que revisó; el resto pasó sin mirarse | Plan → código: dónde vive cada parte y dónde se desvió |
| Prompter técnico | Lo que pidió, no lo que la IA terminó haciendo | Fragmentos sueltos, nunca el conjunto | El concepto que realmente emergió |
| Vibe coder | Solo el resultado en pantalla | No lee código | Todo, de arriba abajo |

**Marca sobre la tabla:** las cuatro primeras filas llevan la etiqueta **«Audiencia principal —
lectores técnicos que pierden el control»**. La última lleva **«Después del piloto»**.

**Pie:** El hueco común es el mismo en todas las filas: el puente entre el concepto y el código.

**Notas.** Cinco perfiles, de más a menos control. Dos cosas varían: cuánto del concepto sostienen
y cuánto control del código conservan. El 2 de octubre decidimos para quién optimizamos primero:
lectores técnicos que están perdiendo el control de código que escribió una IA. El vibe coder no se
descarta; se pospone hasta que el piloto diga si un mapa le sirve más que preguntarle a un agente.
Esa decisión mueve el trabajo hacia componentes, datos, flujos y cambios, y aleja las visitas
guiadas y el lenguaje llano.

---

## 05 · Por qué no basta preguntar

**Etiqueta:** 01 — La objeción obvia

**Titular:** ¿Y por qué no le preguntas a la IA que lo escribió?

**Razón principal (grande):**
**01 · Muestra lo que no sabías que preguntar.** Una respuesta cubre la pregunta; un mapa cubre el
sistema. Nadie pregunta por una tabla que no sabe que existe, por un servicio que nadie le mencionó,
o por una API de pago que nunca aprobó.

**Las otras cuatro (columna):**
- **02 · No puede inventar estructura.** Un chat puede describir un componente que no existe.
  ArchDoc no: la existencia solo viene de la extracción.
- **03 · Dice qué cambió.** Un chat no recuerda cómo era el sistema la semana pasada. ArchDoc guarda
  cada versión y reporta la diferencia.
- **04 · Es la misma respuesta siempre.** El mapa cambia solo cuando cambia el código.
- **05 · Se queda en la máquina.** Por defecto no sale nada, y cada byte que sale queda registrado.

**Notas.** La objeción obvia: la IA que escribió el código también lo explica. Cinco razones, en
orden de fuerza. La primera es el argumento más fuerte del producto: una respuesta cubre la
pregunta, un mapa cubre el sistema. Que esto sea cierto en la práctica es justamente lo que el
piloto tiene que medir; vuelvo a eso al final.

---

## 06 · El panorama

**Etiqueta:** 02 — El panorama

**Titular:** Quién decide qué se dibuja

| | Lee | Quién decide | ¿Se puede comprobar? |
|---|---|---|---|
| **Google Code Wiki** · Gemini, hospedado, gratis — el competidor directo | Cualquier repositorio público, todo su código | Un LLM | No. Enlaza archivos, pero nada comprueba que lo descrito exista |
| **DeepWiki** · Cognition | Cualquier repositorio | Un LLM | En parte. El modelo elige las citas y dibuja los diagramas; nada comprueba ninguna de las dos |
| **Archify** · una habilidad de agente | Lo que el agente leyó | El agente, layout incluido | No. Las fuentes son opcionales y las elige el agente |
| **ArchDoc** | Configuración y código del repositorio | Reglas de extracción fijas: ningún modelo decide qué existe | Sí. Cada caja y cada flecha abre en la línea que la prueba; sin línea, no se dibuja |

**Pie:** Cubren cualquier repositorio porque ningún mecanismo exige que lo dibujado exista.

**Notas.** El panorama cambió. Google publicó Code Wiki: hospedado, gratis, cualquier lenguaje, con
diagramas, recorridos por módulo y chat, actualizado en cada commit. Es casi todo lo que nuestra
visión proponía. En las tres alternativas un modelo de lenguaje decide qué aparece. Citan archivos,
pero el modelo elige las citas y ningún paso verifica que lo descrito exista. En ArchDoc la
existencia la decide un parser.

---

## 07 · Nuestra posición

**Etiqueta:** 02 — Nuestra posición

**Titular:** La alternativa local y verificable. No el rival.

**Columna izquierda — Dónde ellos son mejores**
- Cobertura: cualquier lenguaje hoy, sin configuración
- Cero instalación y un enlace para compartir
- Acabado visual y chat encima

**Columna derecha — Lo que ellos no pueden ofrecer**
- **Nada inventado:** la estructura la produce un parser, no un modelo
- **Comprobable:** archivo y línea en cada elemento y en cada valor
- **Dice qué no vio:** un reporte de cobertura publicado
- **Es tuya:** texto versionado en tu repositorio, legible sin ArchDoc instalado
- **Privada y demostrable:** el código no sale de la máquina; lo que sale queda registrado
- **Tus correcciones sobreviven** a cada regeneración (`rules.yaml`)

**Regla al pie:** Cada funcionalidad que igualemos debe tener su contraparte verificable. Si no,
construimos un Code Wiki peor.

**Notas.** No competimos con ese equipo en amplitud y no hace falta. Lo digo antes de que lo
pregunten: en cobertura, en cero instalación y en acabado, ellos ganan. Nuestra respuesta es ofrecer
la misma utilidad sobre una base que ellos no pueden ofrecer: extracción comprobable, determinismo,
operación local, y documentación que vive en el repositorio del usuario y no en el sitio de otra
empresa. La regla que nos imponemos está al pie.

---

## 08 · Versión 1: construida y medida

**Etiqueta:** 03 — Lo construido

**Titular:** La versión 1 funciona, y está medida

**Columna izquierda — el pipeline, ocho etapas**
- **1–2 · Descubrir y extraer** — qué archivo es la arquitectura, y leerlo en dos pasadas
- **FactSet** — cada hecho con su archivo y su línea *(línea divisoria)*
- **3–4 · Derivar y refinar** — de hechos a un grafo, más las correcciones de la persona
- **5 · Etiquetar** — el modelo de lenguaje, y solo aquí. La única etapa que toca la red
- **6–8 · Validar, guardar, dibujar** — nunca se escribe nada parcial

*Bajo el pipeline:* Que solo la etapa 5 salga a la red no es una promesa: la integración continua
falla si cualquier otro paquete importa un cliente HTTP.

**Columna derecha — medido sobre Immich, Supabase y Mastodon**

| Criterio | Resultado |
|---|---|
| Procedencia (AC-1) | 100 % — 24 nodos, 34 aristas, tres repositorios |
| Estructura sin modelo (AC-2) | Idéntica con el modelo apagado y encendido |
| Validador (AC-4) | 13 de 13 fallos inyectados, rechazados |
| Correcciones (AC-5) | 10 de 10 sobreviven a la regeneración |
| Determinismo (AC-7) | 5 corridas, documentos byte a byte iguales |
| Egreso (AC-8) | Cero contenido de archivos, medido en el cable |
| Exactitud contra referencia (AC-3) | Pendiente: referencia dibujada a mano, 5–16 Oct |
| Detección de cambios (AC-6) | Pendiente: noviembre |
| Rendimiento (AC-9) | Pendiente: en medición |

**Destacado:** **AC-2, la tesis.** Mismos elementos, tipos y relaciones en Supabase a lo largo de
cuatro versiones. Solo cambian las palabras: el modelo interpreta hechos, nunca los produce.

**Notas.** La versión 1 lee Compose, archivos de entorno y configuración de gateways. Cada elemento
cita archivo y línea. Los nueve criterios se escribieron antes del código; seis están cumplidos y
tres están pendientes con fecha. El más importante es AC-2: con el modelo encendido y apagado la
estructura es idéntica, verificado en Supabase en cuatro versiones. *(Aquí va la demostración: el
mismo diagrama con el modelo apagado y encendido.)*

---

## 09 · El techo

**Diapositiva oscura.** **Etiqueta:** 03 — El techo

**Titular:** El límite no era el motor. Era la evidencia que leía.

**Izquierda — Lo que la versión 1 dibuja de Immich**
Cinco cajas unidas por `depends on`, con `immich-machine-learning` sin conectar.

**Derecha — Lo que estaba en el código y no se leía**
- El servidor completo: controladores, servicios, rutas
- El modelo de datos
- Las páginas de la aplicación web
- **La arista que faltaba:** `server/src/dtos/config.dto.ts:624` → `http://immich-machine-learning:3003`

**Tres límites (fila inferior):**
- Solo sirve para sistemas de varios servicios
- Solo lee configuración
- El nivel contenedor dice poco aun cuando funciona

**Notas.** Este fue el hallazgo del 14 de septiembre. Los límites no eran del motor, eran de la
entrada. Una app de React, un backend CRUD o un script no tienen Compose, así que ArchDoc no decía
nada de ellos. Y aun cuando funcionaba, decía poco: para Immich, cinco cajas, con machine learning
desconectado. La conexión real existe y está en una línea de TypeScript que no leíamos. La decisión
fue: la arquitectura sigue siendo el producto; lo que se amplía es la evidencia.

---

## 10 · El código como evidencia — construido

**Etiqueta:** 04 — El código como evidencia

**Titular:** Immich, leído desde su código

**Antes / ahora (dos columnas grandes)**

| | Versión 1 (configuración) | Ahora (configuración + código) |
|---|---|---|
| Piezas | 5 cajas de Compose | 5 aplicaciones descubiertas por sus manifiestos |
| Organización interna | — | **43 componentes**, 145 usos entre ellos, ningún import sin resolver |
| Qué hace | — | **301 rutas** del servidor, **55 páginas** web |
| Datos | — | **68 tablas**, 65 llaves foráneas |
| Qué pasa cuando… | — | **298 de 301 rutas** seguidas hasta sus tablas y llamadas salientes, cada una como diagrama de secuencia |
| Servidor → machine learning | sin conectar | **conectado**, citado en `config.dto.ts:624` |
| Lo que no se pudo resolver | invisible | **11 llamadas**, mostradas como no resueltas |

**Fila inferior — cómo**
- Parser dentro del binario: tree-sitter en Go puro, cinco gramáticas (TypeScript, TSX, JavaScript,
  Python, Svelte). Sigue siendo un solo archivo, sin instalar nada más.
- 543 archivos TypeScript del servidor, leídos en 1 segundo.
- Paquetes por framework: NestJS, FastAPI, SvelteKit, TanStack Router; tablas de TypeORM, SQLModel y
  SQLAlchemy.
- Segundo repositorio de prueba, la plantilla de FastAPI: 23 rutas, 8 páginas, 18 flujos.

**Límites, dichos en la diapositiva:** dos lenguajes. Las migraciones no se leen. La resolución es
por nombre, no por tipo. Las explicaciones en prosa están construidas pero sin corrida real.

**Notas.** Esto es lo nuevo. Entre el 5 y el 6 de octubre construimos la lectura de código, en
rebanadas verticales, cada una del parser a la pantalla. Sobre Immich: cinco aplicaciones, 43
componentes, 301 rutas, 68 tablas, y 298 rutas seguidas hasta la base de datos. La arista que
faltaba aparece, citada en la línea 624. Y once llamadas que no pudimos resolver se muestran como no
resueltas en lugar de desaparecer. Los límites también van en la diapositiva: TypeScript y Python
por ahora, y la prosa generada todavía no se ha corrido con un modelo real.

---

## 11 · Un modelo, muchos lentes

**Etiqueta:** 04 — Un modelo, muchos lentes

**Titular:** Cada lente responde una pregunta que alguien hace de verdad

| Lente | La pregunta | Evidencia | arc42 · C4 | Estado |
|---|---|---|---|---|
| Fronteras | ¿Con qué habla? | Clientes HTTP, entorno, Compose | §3 · Contexto | Construido |
| Piezas en ejecución | ¿Qué piezas corren? | Manifiestos, Compose | §5 · Contenedor | Construido |
| Organización interna | ¿Cómo está organizado? | Módulos, imports, convenciones | §5 · Componente | Construido |
| Capacidades | ¿Qué hace? | Rutas, páginas | §1 | Construido |
| Recorridos | ¿Qué pasa cuando…? | Caminos de llamada desde una entrada | §6 · Dinámico | Construido |
| Datos | ¿Qué guarda, y en qué forma? | Clases de tabla | §8 | Construido |
| Despliegue | ¿Dónde corre? | Compose | §7 · Despliegue | Construido |
| Cambios | ¿Qué acaba de hacer la IA? | Versiones, historial de git | — | Entre versiones: construido. Entre commits: noviembre |
| Plan contra realidad | ¿Construyó lo que se pidió? | Los planes que el usuario entregue | §1 §10 | Si hay espacio |

**Pie:** Un solo modelo validado. Los documentos y la aplicación son proyecciones de él, así que no
pueden contradecirse. El nivel de código —funciones y variables— queda fuera.

**Notas.** Un solo modelo, mirado a través de lentes. Cada uno responde una pregunta real y cada uno
ya tenía su sección en arc42 y su vista en C4; estaban vacías. Siete de los nueve están construidos.
Un hallazgo lateral: Compose describe despliegue, así que se movió a la sección 7, donde pertenece.
El último lente, plan contra realidad, es el que ningún competidor tiene; entra solo si el resto
aterriza antes.

---

## 12 · La superficie

**Etiqueta:** 04 — Qué se entrega

**Titular:** Una aplicación, dos modos, y documentos que viven en el repositorio

**Tres bloques**

**En el repositorio** — Markdown, SVG y `model.json` versionados. C4 y arc42. Se leen con ArchDoc
desinstalado y se revisan en un pull request. Las secciones escritas por personas nunca se leen ni
se tocan.

**La aplicación local** — `archdoc serve`. Explorador por niveles: sistema, contenedores,
componentes, datos. Cada elemento y cada flecha abren su archivo en la línea. Comparación entre
versiones sobre el diagrama. Búsqueda. Reporte de cobertura.

**El sitio publicado** — `archdoc export --site`. La misma aplicación con una versión horneada;
se aloja en cualquier servidor estático. Las citas abren la línea en el repositorio, en el commit
exacto.

**Fila inferior — medido:** 301 elementos y 571 relaciones dibujados en 177 ms, a 60 cuadros por
segundo. A ese tamaño el límite es la legibilidad, no el dibujo.

**Aún no:** los controles para correr y publicar desde la app, y el editor de correcciones.

**Notas.** Tres maneras de llegar al mismo modelo. La documentación de referencia sigue siendo el
entregable central y vive en el repositorio. La aplicación se rediseñó el 4 y 5 de octubre: ahora
dibuja los diagramas ella misma, se puede arrastrar una vista y guardarla, y la misma aplicación se
publica como sitio estático. Esa es nuestra respuesta a «compártelo con mi equipo» sin un servicio
hospedado. *(Segundo momento de demostración: Immich en el explorador, de sistema a componentes, y
una flecha abriendo su línea.)*

---

## 13 · Tres estados de verdad

**Etiqueta:** 05 — Qué lo hace confiable

**Titular:** Más modelo de lenguaje exige un límite más preciso, no menos

**Regla 01 — La existencia y las relaciones son solo por extracción.**
Una funcionalidad, un módulo, una tabla o una llamada existe en el modelo solo si un archivo lo
prueba.

**Regla 02 — El significado puede venir del modelo, solo como afirmación citada.**
Cada frase apunta a elementos del modelo; el validador rechaza la afirmación cuyas citas no
resuelven.

**Los tres estados (fila de tres)**
- **● Probado** — Extraído. Abre en su archivo y su línea.
- **◐ Interpretado** — Del modelo, citado, y marcado siempre de forma visible.
- **○ No resuelto** — Evidencia de que algo existe que no pudimos identificar. Nunca se descarta en
  silencio. *En Immich: 11 llamadas.*

**Pie:** Para quien lee código, la cita es un enlace. Para quien no, es una garantía: nada en este
mapa fue inventado.

**Notas.** Si el modelo va a hacer más, el límite tiene que quedar más preciso. Dos reglas. La
existencia es solo por extracción. El significado puede venir del modelo, pero cada frase cita
elementos y el validador rechaza la que no resuelva; esa regla ya está construida y probada, y falta
correrla con un modelo real. Y tres estados que se muestran siempre. El tercero es el que evita
repetir el fallo de machine learning a escala: lo que no se pudo resolver se dibuja como no
resuelto. Además, una memoria de interpretación: se vuelve a preguntar solo por lo que cambió, así
que la prosa no se reescribe en cada corrida.

---

## 14 · Privacidad

**Etiqueta:** 05 — Privacidad con el código como evidencia

**Titular:** Salen nombres. Nunca texto.

**Lo que sale, si se pide una explicación**
Nombres de componentes, rutas de archivos, quién usa a quién, rutas por método y manejador, tablas
y nombres de columnas.

**Lo que no sale nunca**
Ninguna línea de código. Ninguna cadena que el código contenga: ni resúmenes de rutas, ni
docstrings, ni comentarios. Tampoco la procedencia.

**Tres modos**
- **Sin modelo** — por defecto. No se hace ninguna petición y la documentación sale completa.
- **Solo estructura** — nombres, como arriba. Registrado byte por byte, tal como salió al cable.
- **Local** — un modelo en la propia máquina: no sale nada. *Planeado.*

**Pie:** Las cadenas en el código son donde la gente escribe cosas que no pensaba publicar. Los
nombres son lo que cualquiera ve en el árbol del repositorio. Una prueba automática sostiene la
línea.

**Notas.** Con código como evidencia, la frontera había que escribirla con exactitud, porque el
criterio de egreso se mide sobre ella. Decidido el 6 de octubre: salen nombres, nunca texto. Los
resúmenes de rutas y los docstrings ayudarían a la prosa, y aun así se quedan. Y recordar que el
modelo es opcional: sin la bandera no se envía nada y el resultado es completo.

---

## 15 · La prueba que falta

**Diapositiva oscura.** **Etiqueta:** 06 — La prueba que falta

**Titular:** Todo esto descansa en una hipótesis que nadie ha probado

**Hipótesis:** Que un mapa verificado ayuda a una persona a entender software que una IA construyó
— mejor que preguntarle al agente.

**Si es falsa, la corrección no importa.**

**Sprint de evidencia · 20–24 de octubre**

**Piloto de comprensión — tres condiciones**
- **A** · Solo el código
- **B** · Preguntarle a un agente — el competidor real
- **C** · ArchDoc

Se mide en aciertos y en tiempo, sobre una aplicación desconocida hecha con IA. Incluye preguntas
sobre cosas que el participante nunca supo que existían: es donde predecimos ganar.

**Comparación medida** — ArchDoc, Code Wiki y DeepWiki sobre Immich, contra una referencia dibujada
a mano: qué omitió cada uno y qué inventó cada uno.

**Pie:** Una comparación que deje fuera la condición B no prueba nada.

**Notas.** Esto es lo que quiero que el panel escuche con claridad. Todo lo anterior mide
corrección. El objetivo es comprensión, y eso no lo hemos medido. La hipótesis es que un mapa
verificado ayuda más que preguntarle a un agente; si es falsa, la corrección no importa. Habíamos
decidido medir antes de construir; el código avanzó más rápido de lo planeado y el piloto sigue
pendiente. Su resultado todavía puede redirigir lo que queda. La segunda prueba es la comparación
contra Code Wiki y DeepWiki sobre el mismo repositorio; para eso hace falta la referencia dibujada a
mano, que está en curso.

---

## 16 · Plan y cierre

**Etiqueta:** 06 — Plan

**Titular:** La identidad no cambió. Cambió la evidencia que la sostiene.

**Línea de tiempo**

| Cuándo | Qué |
|---|---|
| Hecho · Sep | Versión 1: configuración, el motor completo y la aplicación |
| Hecho · 4–6 Oct | Superficie rediseñada · el código como evidencia |
| **9 Oct** | **Revisión de avance** |
| 13–17 Oct | Semana de planificación: lista de funcionalidades cerrada |
| 20–24 Oct | Sprint de evidencia: piloto, comparación, referencia de Immich |
| 27 Oct – 3 Dic | Lo que falta: diff entre commits, explicaciones en vivo, modelo local, controles de la app |
| 4 Dic | Congelamiento de código |
| 4–10 Dic | Medición de los nueve criterios · informe |
| **11 Dic** | **Entrega** |

**Lo que nunca se recorta:** procedencia en cada elemento · el estado «no resuelto» · el reporte de
cobertura · la medición de aceptación.

**Pie:** Gracias · 16 / 16

**Notas.** Cierro donde empecé. La identidad no cambió: documentación de arquitectura, trazable al
código. Cambió la evidencia que la sostiene, y ese cambio ya está construido. Lo que sigue es
medir: la semana próxima cerramos la lista, la siguiente corremos el piloto y la comparación, y
construimos hasta el 3 de diciembre. Entrega el 11. Si algo se atrasa, ya está acordado qué se
recorta primero y qué no se recorta nunca. Gracias.
