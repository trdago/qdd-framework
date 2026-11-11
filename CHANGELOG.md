# Changelog

Todos los cambios notables de este proyecto serán documentados en este archivo.

El formato está basado en [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/), y este proyecto adhiere a [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Corregido

- **En Windows sin administrador, la tarea de qdd remoto ya queda registrada (FND-195, MEJ-052, MEJ-060):** `qdd login` y `qdd remoto activar` escribían una tarea de inicio que valía para cualquier usuario del equipo, y Windows respondía «Acceso denegado» al registrarla, así que QDD Móvil dejaba de ver el PC en cuanto el servicio se cortaba. Ahora la tarea es solo de tu usuario. Una tarea de antes se corrige sola al actualizar qdd, y `qdd doctor` la marca (REM-12) y `--fix` la registra.

## [2.10.0-rc.60] - 2026-11-10

### Corregido

- **«Actualizar qdd» espera a que npm entregue la versión (FND-194, MEJ-049):** el panel la ofrece en cuanto npm la nombra, pero npm install podía no verla todavía y fallaba con ETARGET. Ahora el pedido le pregunta a npm primero; si no la ve, dice «todavía no» y se reintenta solo.

## [2.10.0-rc.59] - 2026-11-10

### Corregido

- **En Windows, un pedido ya no muere al cerrar Claude Code (FND-193, MEJ-048):** el pedido corría aparte, pero seguía dentro del Job Object de quien lo lanzó, y un job que mata a sus procesos al cerrarse se lo llevaba a medio camino («exit -1 · terminó sin anotar el resultado»). Ahora pide salir del job; si el job no lo permite, se lanza como antes.

## [2.10.0-rc.58] - 2026-11-09

### Corregido

- **El aviso de un ⏳ ya no llega viejo ni repetido (FND-192, MEJ-046):** pulsar una etapa ⏳ mientras la IA trabajaba dejaba el aviso en la cola del chat, y llegaba después de que la IA cerró la etapa, una vez por pulsación. Ahora el panel guarda uno por etapa, dice «se envía cuando la IA termine» y al terminar el turno se lo vuelve a pedir a qdd: si la etapa ya no espera, no se envía.

## [2.10.0-rc.57] - 2026-11-09

### Added
- El baúl de accesos se administra desde Config en el panel (pedido de la persona, 2026-11-09). Ahí ves cada token con su tipo, lugar, dueño, vencimiento y último uso, nunca su valor; los rechazados aparecen en rojo. «+ Agregar token» y «Modificar» guardan lo que copiaste: presionas «📋 Pegar del portapapeles» y qdd lo lee, vacía el portapapeles, lo prueba con su API y lo guarda (`qdd baul guardar … --portapapeles`). «Quitar» pregunta antes y borra el valor y sus permisos (`qdd baul quitar`). «Verificar» lo prueba de nuevo.

## [2.10.0-rc.56] - 2026-11-09

### Agregado

- **`qdd doctor` revisa que la sesión late (FND-191):** desde una sesión de Claude Code, el grupo de la consola dice si el panel la anota como viva. Si no, explica para qué importa (el baúl y el teléfono) y qué hacer, y si la sesión late en una carpeta de arriba con `.qdd`, la nombra. `--fix` no reinicia la sesión ni el panel: lo dice.

## [2.10.0-rc.55] - 2026-11-09

### Corregido

- **El baúl pide permiso sin QDD Móvil (FND-191, MEJ-056):** sin teléfono emparejado, la sesión no se anotaba como viva y `qdd baul usar` decía «la sesión no late en este proyecto» sin dejar el pedido 🔑 (visto en Windows). Ahora la sesión se anota siempre; entregar instrucciones sigue siendo solo con QDD Móvil.
- **BP-08 nombra el `fuera` mal puesto (MEJ-022, #143):** si un módulo que aplica un blueprint lo declaró fuera en `implementa`, el gate dice que esa entrada no vale para lo aplicado y que el `fuera` va en la raíz de la ficha. Desde la rc.26 (FND-121), el registro de aplicación ya toma el `fuera` de la raíz.

## [2.10.0-rc.54] - 2026-11-09

### Fixed
- El panel sin trabajo ya no gasta CPU (FND-190). Con 8 paneles abiertos se iba cerca de medio núcleo constante, que se notaba como caídas de FPS en un juego. El candado del latido de cada panel despertaba a todos los demás paneles de la máquina, y cada relectura parseaba `estado.yaml` unas cinco veces. Ahora un panel solo despierta por los archivos de su proyecto, y el estado se parsea una vez mientras no cambie: la relectura bajó de 31 ms a 3,5 ms.

## [2.10.0-rc.53] - 2026-11-09

### Added
- Logrado para QDD Móvil (rediseño elegido por la persona, 2026-11-09): `qdd consola panel --con logrado` trae lo que salió bien en los últimos 14 días. Eso son los pedidos hechos con su salida, las tareas cerradas con su motivo, los objetivos cumplidos con su evidencia y lo que mandaste desde el teléfono con cómo terminó. El relay lo publica pasado por el ocultador. Los pedidos se empiezan a juntar desde esta versión, porque antes salían de la cola a los 30 minutos.

## [2.10.0-rc.52] - 2026-11-08

### Changed
- Flujos: cada nodo se dibuja con solo su título, para leer el flujo de un vistazo (pedido de la persona, 2026-11-08). `qdd flujo ver --detalle`, o `h  ☰ Ver detalle` en el panel, pone la descripción debajo del título, separada por una línea; la tarjeta del nodo elegido la muestra siempre.
- Flujos con peso visual (pedido de la persona, 2026-11-08: «le falta peso visual»). En el panel, el título de cada nodo va en negrita; los bordes y las flechas van en el gris de la marca; la descripción va en el gris claro; el estado de su tarea va en verde, ámbar o rojo; y el nodo elegido va entero en el azul de acento. Las columnas no cambian. qdd marca el rol de cada trozo del dibujo (`rol` en los segmentos de `qdd flujo ver --json`), y un qdd anterior dibuja como antes. En la página HTML, los títulos van en peso medio y el elegido en negrita.

## [2.10.0-rc.51] - 2026-11-08

### Added
- Panel: la pestaña Flujos abre a pantalla completa cuando la terminal está en pantalla completa (pedido de la persona, 2026-11-08). ⤡ Al lado (z) / ⤢ Pantalla completa la alternan y ← Tareas (t) vuelve al tablero con el reparto de siempre.

## [2.10.0-rc.50] - 2026-11-08

### Fixed
- El panel ya no queda en negro sin repositorio de blueprints (MEJ-045, MEJ-050, FND-189). Lo que qdd le entrega al panel nunca trae una lista null (`pkg/listas`), una prueba de contrato lo revisa en un proyecto vacío y en uno con datos, y cada vista se dibuja protegida: si una falla se ve «✗ La vista X falló», se puede volver al tablero y la falla queda en `qdd incidentes panel`.

## [2.10.0-rc.49] - 2026-11-08

### Added
- Compartir un flujo con un enlace (RFC 0025, fase 3b; propuesta tuya): «↗ Compartir» (c) en la esquina de la pestaña Flujos, o `qdd flujo compartir <flujo> --enlace`. Con un token de Cloudflare en el baúl, qdd instala un Worker en tu cuenta, sube la página y copia el enlace; quien lo recibe solo lo abre y propone. La primera vez en cada consola, el baúl pide tu permiso en «Para ti». El token de la invitación va solo en el `#` del enlace, que el navegador no le manda a Cloudflare. Revocar la invitación o `--quitar-enlace` hace que el enlace deje de abrir. Sin Cloudflare en el baúl, queda el archivo de siempre.

## [2.10.0-rc.48] - 2026-11-08

### Fixed
- FND-188: un pedido 📮 que confirmaste ya no falla porque un qdd anterior le borró la marca «de la persona». La cola conserva los campos que una versión no conoce, y el siguiente ciclo de mejoras le devuelve la marca a un pedido que la perdió.

## [2.10.0-rc.47] - 2026-11-08

### Added
- Flujos versionados y con plantilla (RFC 0025, fase 1). Cada cambio sube una revisión con su autor en `<flujo>.historial.jsonl` (`qdd flujo historial`). `qdd flujo version` guarda una versión con nombre, `comparar` dice qué nodos cambiaron entre dos y `volver` deja el flujo como en una, en una revisión nueva. La plantilla (`proceso`, `web`, `datos`, `agentes`; `qdd flujo plantillas`) trae los tipos de nodo y el nombre de los niveles N1 · N2 · N3. Doctor avisa de un flujo editado a mano cuyo historial no llega a su revisión (FLU-02), y `--fix` lo anota.
- Grupos con clase: `qdd flujo grupo <flujo> "<nombre>" --clase fase|cola|ambiente…` le da clase y descripción a un grupo de nodos («[fase] Ingreso»).
- `qdd flujo compartir <flujo> [--para <nombre>]` escribe un HTML de un solo archivo para conversar el flujo con alguien sin QDD. Trae N1 · N2 · N3, grupos como marcos, la leyenda de la plantilla, un recorrido de la primera vez, el detalle al tocar un paso y enlaces `#n2/<nodo>`. Va limpio de rutas y secretos, sin nada de afuera y con CSP por hash. Sin `--rol` es de solo lectura.
- El archivo de `qdd flujo compartir` lleva la marca de QDD y todo adentro: isotipo «▶_», paleta Gruvbox oscura, JetBrains Mono embebida, esquinas de 2 px sin sombras y un ícono SVG por tipo de nodo. Los íconos de las plantillas ya no usan emojis.
- Invitados que proponen (RFC 0025, fase 3): `qdd flujo compartir <flujo> --rol cliente|experto [--dias N] [--tope N]` crea una invitación en el relay de QDD y el archivo propone. En el detalle de cada paso hay acciones directas (comentar, agregar después, renombrar, describir, conectar, mover, quitar y, para el experto, nota técnica). La lista «Propuestas» muestra el estado de cada una en vivo, y la foto nueva llega sola cuando apruebas algo. Sin conexión, lo propuesto espera en el navegador. Cada propuesta llega a tu panel como un pedido 📝 que solo tú confirmas (`qdd flujo propuestas aprobar PRO-n`); descartarlo la rechaza. Lo que no cumple el esquema o el rol se rechaza solo, con su motivo. Abierto en otro navegador, el archivo igual dice qué se propuso en cada una. También `qdd flujo propuestas`, `qdd flujo invitaciones` y `qdd flujo compartir --revocar <invitación>`.
- Historias de usuario en el flujo (RFC 0025): `qdd flujo historia <flujo> --como … --quiero … --para … --criterio … --nodo …` las escribe con id `h1`, `h2`…, `qdd flujo historias` las lista y `qdd flujo historia tarea <flujo> <h>` crea la tarea del tablero que la implementa e imprime sus criterios. El cliente y el experto también las proponen desde el archivo («Historia de usuario»), y entran a su nombre cuando apruebas. Se ven en su nodo, en el panel y en el archivo.
- Notas internas o compartidas: una nota es interna hasta que la compartes (`qdd flujo item compartir <flujo> <nodo> <item> [--interna]`, o el botón «interna / la ve el cliente» del nodo en el panel). El experto puede pedir compartirla al proponerla, y la ves así antes de aprobar. El archivo del cliente lleva solo las compartidas, nunca una interna ni el origen de una historia.

### Changed
- Panel, pestaña Flujos: cada nodo se toca en el dibujo mismo y su tarjeta (comentar, preguntar, editar, mover…) se abre justo debajo de él, sin bajar a la lista. En el panorama, tocar la caja de un grupo lo abre. La lista de nodos sigue debajo para el teclado y los que el nivel esconde. `qdd flujo ver --json` entrega cada línea partida por nodo (`segmentos`).

## [2.10.0-rc.46] - 2026-11-08

La rc.45 no llegó a npm: su tag salió sin subir la versión del paquete. Esta trae lo mismo.

### Fixed
- FND-187: un pedido que terminó bien ya no se le avisa a la IA como «se cortó». El panel envía el resultado que qdd tiene al reclamar el aviso, no el de su lectura anterior; si el resultado real llega después de un corte, se vuelve a avisar; y un pedido no se da por cortado en sus primeros 5 s.

## [2.10.0-rc.44] - 2026-11-08

### Added
- Esperas que se responden solas: `qdd estado set … espera --verifica npm:<paquete>@<versión>|pr:<n>|tag:<tag>|pedido:<n>|archivo:<ruta>` (también `verifica` en `qdd_estado`). `qdd estado verificar` las revisa (el panel cada minuto con `--si-toca`, cada 10 min por espera), y cuando se cumple la celda pasa sola a ✅ con su evidencia. La espera muestra «⟳ se responde sola» y no se le pregunta a la persona. Doctor avisa de las que fallan (VRF-01) o no se entienden (VRF-02).

## [2.10.0-rc.43] - 2026-11-08

### Changed
- Panel: en «▶ Para ti» van primero las acciones (pedidos y fichas); las preguntas (⏳ y lo abierto en la terminal) quedan plegadas en un botón «▸ ? N» que se despliega si quieres, cada una con su número de turno.

## [2.10.0-rc.42] - 2026-11-08

### Added
- La cola de mejoras lee los reportes del relay (RFC 0024, fase 3): en el repositorio de QDD, `qdd mejoras ciclo` trae lo que llegó con `qdd reportar` y lo deja `por_admitir`. Como cualquiera puede enviar, no es tarea hasta que la persona lo admite con el pedido 📮 del panel (`qdd mejoras admitir MEJ-NNN`), igual que la etiqueta «mejora» de un issue.

## [2.10.0-rc.41] - 2026-11-08

### Added
- `qdd reportar` (RFC 0024): un camino directo a QDD para quien no tiene GitHub. Limpia el texto (credenciales, valores del baúl, la ruta del proyecto, tu carpeta y tu nombre de usuario), lo muestra tal cual viaja y lo envía al relay de QDD solo con tu «sí»: en tu terminal pregunta; desde una IA deja un pedido 📮 que solo la persona confirma. Firma con la llave de la instalación (`qdd reportar llave`); `qdd reportar enviados` lista lo enviado.

### Fixed
- FND-186: la limpieza de un reporte reemplaza la ruta del proyecto antes que las cadenas largas, para que un tramo largo no deje pasar el resto de la ruta.

## [2.10.0-rc.40] - 2026-11-08

### Corregido

- **Un lockfile de blueprints escrito en Windows ya no rompe `qdd audit` en Linux (FND-185, MEJ-038).** La huella de
  cada blueprint dependía del fin de línea: con CRLF, el gate acusaba edición a mano sin que la hubiera. Ahora se
  calcula con los archivos de texto en LF, como los guarda git. Las huellas ya escritas en Linux no cambian. Importar
  desde una carpeta local avisa que esa ruta no existe en otro PC.

## [2.10.0-rc.39] - 2026-11-08

### Corregido

- **Un comentario que qdd no puede guardar ya no se pierde (FND-184, MEJ-037).** Si qdd rechaza un comentario, una
  pregunta, un ítem o el nombre de un nodo nuevo en la pestaña Flujos, el texto sigue en el campo junto al aviso, para
  reintentar.

## [2.10.0-rc.38] - 2026-11-08

### Agregado

- **Flujos grandes por niveles: panorama, camino y todo (#186, partes B y C).**
  - **Nivel 1, panorama:** una caja por grupo, con sus hitos y cuántos pasos y desvíos esconde. En la pestaña Flujos,
    un flujo de 15 nodos o más abre aquí, y cada grupo se abre o se pliega con un clic.
  - **Nivel 2, camino:** el camino principal, con los desvíos contados.
  - **Nivel 3, todo:** el flujo entero.
  - Las teclas 1, 2 y 3 cambian el nivel.
  - Con cuatro grupos, `analisis-oportunidades` ocupa 21 líneas en el panorama, frente a 95 completo.
  - La tarjeta del nodo muestra su nivel (el calculado o el que pusiste) y su grupo, y Editar los cambia. En la CLI es
    `qdd flujo modificar --grupo G --nivel N`, y `qdd flujo ver` acepta `--nivel` y `--abrir`.

## [2.10.0-rc.37] - 2026-11-08

### Agregado

- **`qdd baul`: los tokens se guardan una vez y la IA los usa sin verlos (RFC 0023, fase 1).**
  - `qdd baul guardar <nombre> --tipo cloudflare|github|supabase|resend|aws-sso|otro` pide el valor oculto, solo en tu
    terminal. Lo guarda en el llavero del sistema, o en 1Password con `--en 1password`. De AWS guarda solo el perfil de
    IAM Identity Center.
  - Al guardar, y con `qdd baul verificar`, QDD prueba el token contra su API y anota dueño, alcance y vencimiento.
  - `qdd baul usar <nombre> --para "<fin>" -- <comando>` corre el comando con el token solo en su entorno y tapa los
    valores del baúl en la salida.
  - El permiso es por sesión, proyecto, token y fin, y dura lo que la consola. Sin permiso, `usar` deja un pedido «🔑»
    en el panel que solo confirma la persona (el modo goal no lo lanza) y sale con 75.
  - Cada uso queda en un registro encadenado (`qdd baul registro`), y `qdd doctor` avisa si se rompe.
  - Doctor avisa también de tokens vencidos o sin uso y de `.env` con secretos, mostrando solo los nombres de las
    claves. No existe `qdd baul ver`.

## [2.10.0-rc.36] - 2026-11-08

### Corregido

- **En Windows, la instalación ya no falla con EBUSY (#175, FND-183).** Un antivirus puede retener unos segundos el
  `qdd.exe` recién descargado. Ahora la instalación espera hasta unos 30 s y, si sigue ocupado, lo copia. Lo temporal
  que no se puede borrar ya no tapa el mensaje, y no queda una carpeta de versión vacía.

## [2.10.0-rc.35] - 2026-11-08

### Corregido

- **Los flujos ya no fallan con «not a directory» (FND-182, MEJ-036, MEJ-037).** Cada comando dejaba su nombre en
  `.qdd/working` como archivo, la misma ruta donde los flujos guardan su carpeta; si el comando se cortaba, el archivo
  quedaba y `qdd flujo crear` o un comentario fallaban. El nombre va ahora en `.qdd/working/comando`, y qdd quita solo
  el archivo viejo.

## [2.10.0-rc.34] - 2026-11-07

### Agregado

- **`qdd rendimiento`: la misma medición en cada equipo, sin tokens (RFC 0022).** Corre qdd en una caja temporal, igual
  en cada equipo y sin tocar tus proyectos, y mide la latencia de la barra, el panel, `qdd estado` y un flujo; la CPU,
  la memoria y los procesos por hora de `qdd consola seguir`; los zombies y los procesos que quedan vivos al cerrarla
  (los cierra), y lo que el MCP de qdd suma al contexto de cada sesión. Cada medida tiene un presupuesto por sistema
  operativo. Sale con 1 si dejó procesos y con 2 si solo pasó un presupuesto. `qdd doctor` muestra lo último medido.
  El CI la corre en Linux, Windows y macOS en cada PR. Con rc.31 marca la falla de los zombies (FND-179).

## [2.10.0-rc.33] - 2026-11-07

### Cambiado

- **Todos los flujos en una sola pestaña, Flujos.** Con muchos flujos (20 de prueba en un proyecto), una pestaña por
  flujo era inmanejable, y cerrarlas también. Ahora arriba va un mapa de todos: cada flujo en una fila con sus nodos,
  sus pruebas (✓ pasan, ✗ fallan, ☐ faltan) y lo que tiene por enviar; las teclas 1…9 abren los primeros. Elegido uno,
  se dibuja con **☰ Mapa** (`w`), **‹** (`p`), «2 de 20» y **›** (`n`). Con un solo flujo se dibuja directo. Las
  pestañas de cada flujo que dejó la versión anterior se cierran solas.
- **Un flujo grande se lee sin bajar cientos de líneas.** Cada rechazo, vuelta atrás o atajo partía el ancho, y el
  camino principal nunca lo recuperaba: las cajas quedaban de 9 columnas, con las palabras cortadas. Ahora un desvío
  cuelga al costado del nodo en una línea (`├─ ( REJECTED_REGION )`, `├─ ↷ salta a Cola`) y el camino principal conserva
  todo el ancho. Un flujo de 23 nodos bajó de 315 a 94 líneas a 100 columnas (#186).

### Corregido

- **El dibujo dejaba fuera un nodo cuando las ramas se cruzaban (FND-180).** En 24 000 flujos al azar faltaba uno en 8;
  ahora en ninguno.

## [2.10.0-rc.32] - 2026-11-07

### Corregido

- **Procesos zombie de la consola (FND-179, MEJ-042):** cada consola abierta dejaba un proceso `qdd <defunct>` por
  minuto, más o menos (744 en un PC con 7 consolas). Los pedidos, la revisión de PRs, el aviso de versión nueva y la
  publicación de incidentes se lanzaban y se soltaban sin recogerlos. Ahora cada uno se recoge al terminar. Los que ya
  existen se van al cerrar y volver a abrir cada consola con esta versión.

## [2.10.0-rc.31] - 2026-11-07

### Agregado

- **Ítems en cada nodo de un flujo:** pruebas, notas, características y funciones. Al probar con qdd-human, las pruebas de
  cada paso se ven en el nodo como ☐ pendiente, ✓ pasa o ✗ falla, con su evidencia, y desde el mismo nodo agregas algo
  específico que probar. qdd-human las lee del flujo y las marca con la carpeta del run. En el panel: **Ítem** (`b`) en
  cada nodo, ✓ ✗ para marcar y el resumen en la fila del nodo. El teléfono manda las mismas operaciones. Nuevo
  `qdd flujo item agregar | marcar | quitar`.

## [2.10.0-rc.30] - 2026-11-07

### Agregado

- **Preguntarle a un nodo o al flujo completo:** al mirar un flujo, cada nodo tiene **Preguntar** (`g`), y sobre los nodos
  está **Preguntar sobre el flujo** (`i`). La IA recibe la pregunta en el acto, con el contexto del nodo (su descripción,
  de dónde viene y qué sigue), y responde ahí mismo, orientada a ese nodo. Sirve para entender un sistema nuevo palabra
  por palabra. El teléfono manda la misma operación (`preguntar`). Nuevos `qdd flujo preguntar` y `qdd flujo responder`.

## [2.10.0-rc.29] - 2026-11-07

### Agregado

- **Botones en cada ⏳ tuyo: «Ya lo hice», «Todavía no» y «Ya no va»,** en el panel del PC (bajo cada pregunta de «Para
  ti») y en QDD Móvil. La respuesta queda en el estado aunque no haya una sesión de Claude Code abierta: «Ya lo hice» pasa
  el ⏳ a la IA para que lo verifique, «Ya no va» deja la etapa en — y «Todavía no» lo vuelve a preguntar al día. Si hay
  una sesión, la IA se entera al tiro. En el PC, `qdd consola pregunta responder <id> <opción>`.

### Cambiado

- **QDD Móvil muestra los ⏳ aunque el proyecto no tenga una sesión abierta** (`sin_sesion: true`). Una respuesta escrita
  con «Otra» también queda en el estado, para la IA.

## [2.10.0-rc.28] - 2026-11-07

### Agregado

- **Botón 📷 Foto en el panel:** con una imagen copiada aparece 📷 (tecla `t`). Al pulsarlo trae la foto con `qdd foto` y deja
  «mira <ruta>» en el chat, sin enviarlo, así la IA no gasta tokens en descubrir la herramienta. En Windows y macOS, donde
  no hay una consulta barata del portapapeles, el botón está siempre. `qdd foto` suma `--json` y `--hay`.

## [2.10.0-rc.27] - 2026-11-07

### Agregado

- **Sincronizar desde QDD Móvil:** el botón manda `sincronizar` y el PC relee de cero el estado y los flujos de todos sus
  proyectos y los vuelve a publicar. Después responde `sincronizado` con lo que el tablero de cada proyecto con una consola
  abierta ya no dice bien: ⏳ cuyo pedido ya terminó, tareas listas sin cerrar, cerradas o descartadas con ⏳ y flujos
  inválidos. No corrige nada. En el PC, `qdd sincronizar [--json]` da lo mismo.
- **El teléfono ve qué proyecto tiene las reglas de las IAs atrasadas:** cada proyecto publicado trae `reglas_atrasadas`.

## [2.10.0-rc.26] - 2026-11-07

### Agregado

- **Reglas de las IAs al día en todos los proyectos:** `qdd doctor --todos` corre doctor (con `--fix` si lo pides) en cada
  proyecto registrado en la máquina y resume una línea por proyecto. La vista Proyectos del panel marca con ⚑ el proyecto
  cuyas reglas son de un qdd anterior (`reglas_atrasadas` en `qdd consola proyectos`). Nada se renueva solo: `--fix` deja
  cambios en git en cada repositorio.

## [2.10.0-rc.25] - 2026-11-07

### Corregido

- **El teléfono ya no vuelve a preguntar un ⏳ cuyo pedido terminó (FND-176):** «¿Confirmas [a]?» se ocultaba solo mientras
  [a] estaba abierto. Ahora un ⏳ que nombra un pedido de su misma tarea que terminó bien en las últimas 24 h no se publica,
  y `qdd estado check` le pide a la IA actualizar la fila.
- **`qdd estado check` nombra también las tareas descartadas que conservan un ⏳ (FND-177).**
- **El panel ya no retira un pedido que la persona confirmó (FND-178):** si volvía a pendiente (dejó de esperar o salió con
  75), el siguiente cambio de su tarea lo sacaba de «Para ti».

## [2.10.0-rc.24] - 2026-11-07

### Corregido

- **`qdd audit --verificar-rutas` ya no confunde lo instalado con rutas incompletas (FND-175):** la copia aislada no traía
  `node_modules` y una suite que lo necesita fallaba con un falso «incompletas». Ahora la copia enlaza las carpetas que git
  ignora. Los archivos ignorados, como un `.env`, no entran.

## [2.10.0-rc.23] - 2026-11-07

### Agregado

- **Docker en desarrollo, solo para la base de datos.** La app corre directo en el PC. Si un compose o un `.devcontainer` la
  pone en Docker, `qdd audit` avisa que será más lento y por qué (`DEV-01-APP-EN-DOCKER`), sin bloquear. La forma de trabajo
  le pide a la IA avisarlo antes de proponerlo. La explicación, con sus fuentes, está en `docs/docker-en-desarrollo.md`.

## [2.10.0-rc.22] - 2026-11-07

### Agregado

- **Una suite no se repite si no cambió nada de lo suyo (RFC 0020).** En `test_commands`, una suite puede declarar
  `rutas: [carpetas]`: si ya pasó con esos mismos archivos en las últimas 24 h, `qdd audit` la salta y lo dice. Sirve para
  cualquier lenguaje: un cambio de estilos no repite las pruebas de reglas de negocio. El CI, `qdd certify` y
  `qdd audit --completo` corren siempre todo. Cada suite muestra además cuánto tardó.
- **`qdd audit` no espera a las pruebas si una regla ya bloquea.** Las reglas van primero. En este repo, con una ficha rota y
  un cambio en el panel, pasó de 37 s a 1,6 s. Sigue saliendo con 1 y dice que las pruebas quedaron pendientes
  (`QA-04-PRUEBAS-PENDIENTES`). El CI, `--completo` y `qdd certify` corren siempre las pruebas.
- **`qdd audit --verificar-rutas`:** corre cada suite que declara `rutas` en una copia con solo esos archivos y dice si le
  falta algo. El CI lo hace en cada PR, para que unas `rutas` incompletas no dejen pasar un cambio sin probar.
- **Actualizar una herramienta repite su suite:** la versión de cada programa de `requiere` entra en la huella. Si no se
  puede saber la versión, la suite corre.

### Corregido

- **Las pruebas ya no lanzan un agente real (FND-172):** cuatro pruebas de Go corrían `claude -p` o `agentapi` en cada
  `go test` con un agente instalado: gastaban tokens y sumaban 127 s. `go test ./...` sin caché pasó de 101 s a 33 s, sin
  saltarse ninguna prueba.
- **La caché de Go ya no aprueba pruebas con archivos rotos (FND-173):** Go solo vigila los archivos de `cli/`, así que las
  pruebas que leen docs o workflows pasaban desde la caché aunque esos archivos cambiaran. El gate las corre sin caché.
- **La suite del panel ya corre en el CI y al publicar (FND-174):** quedaba «sin medir» porque el CI no tenía `claude`, así
  que sus 221 pruebas solo corrían en el PC de quien auditaba.

## [2.10.0-rc.21] - 2026-11-06

### Corregido

- **Un relay antiguo guardado ya no da un código sin aviso (FND-171).** `qdd login` y `qdd remoto emparejar` avisan si no
  pudieron confirmar el relay. `qdd doctor` lo revisa sin red, también sin emparejar y con el servicio apagado, y
  `qdd doctor --fix` lo compara con el directorio: lo reemplaza si no coincide o muestra el motivo si no puede consultar.
- **El directorio del relay se sirve desde varios sitios (FND-171).** En una red que bloquea uno, qdd pregunta al siguiente;
  si fallan todos, dice el motivo de cada uno.

### Cambiado

- **Un relay propio protege solo a un PC emparejado (FND-171).** Sin emparejar, sin banderas, manda el directorio firmado y
  se avisa; `qdd login --relay <url> --clave <publicable>` lo vuelve a dar.

## [2.10.0-rc.20] - 2026-11-06

### Agregado

- **`qdd consola proyectos --json` (RFC 0018).** Entrega una fila por proyecto de este PC: puntaje, hace cuánto, findings
  abiertos, ⏳ y ❌, con su historial. Lo que no tiene fuente sale como `null` («sin medir»). Solo lee archivos locales.
- **Vista «Proyectos» en el panel (RFC 0018).** La tecla `y` compara los proyectos de este PC lado a lado, y cada fila abre
  su historial de puntajes y findings con su número o un clic. Se lee al abrirla y con `l ↻`: cerrada no gasta procesos, y
  nada le escribe a la IA.

### Obsoleto

- **`qdd dashboard` queda obsoleto (RFC 0018).** Avisa al arrancar y sigue funcionando igual, pero solo recibe arreglos.
  El panel que sigue es `qdd-tareas`, con su vista Proyectos; se instala con `qdd consola activar`. Quitarlo es una decisión aparte.

## [2.10.0-rc.19] - 2026-11-06

### Corregido

- **`qdd doctor` ya no avisa «sin respuesta del relay» cuando el PC late bien (FND-169).** La hora salía de `plan.json`,
  que solo se escribe si el plan cambia: sin teléfonos conectados, REM-07 se ponía rojo a los 3 minutos. Cada latido
  respondido anota ahora su hora en `latido.json`. Si el relay deja de responder, REM-07 sigue avisando.

## [2.10.0-rc.18] - 2026-11-06

### Cambiado

- **El relay de QDD Móvil ya no está en el código: sale de un directorio firmado.** Si el relay cambia, cambia la URL y no qdd. El PC lo
  consulta al arrancar, en `qdd login` y `qdd remoto activar`, y si la conexión sigue caída; mientras funciona, usa el que recuerda. Solo acepta una
  respuesta firmada con la clave que trae qdd. Si la consulta falla sigue con el recordado y `qdd doctor` lo dice (REM-11). Un relay dado con
  `--relay` y `--clave` es tuyo y no se consulta. Un PC que nunca consultó y no alcanza el directorio no puede emparejarse hasta que responda o
  le des un relay a mano: qdd ya no trae uno de respaldo.

## [2.10.0-rc.17] - 2026-11-06

### Corregido

- **Una instalación nueva se empareja en el relay nuevo de QDD Móvil (FND-168):** el valor por defecto seguía apuntando al relay anterior.
  Un PC ya emparejado no cambia por esto: usa el que tiene guardado.
- **El PC pide Presence de forma explícita al unirse al canal (FND-167):** es preventivo: un cliente de prueba que no lo pedía no recibió
  quién entraba al canal, aunque un PC con rc.16 sí se vio en el relay nuevo.

### Agregado

- **Un turno único para lo que te espera (RFC 0019):** cada pregunta, ⏳, pedido y ficha recibe de qdd un número que no cambia. «Para ti»
  es una sola lista en ese orden, con `▶` en lo que sigue y `espera a #N` en lo que depende de otra cosa. QDD Móvil recibe los mismos
  números; la app todavía tiene que usarlos.

## [2.10.0-rc.16] - 2026-11-06

### Agregado

- **Editar y comentar un flujo desde QDD Móvil (lado PC, solo Pro):** el teléfono puede mandar una edición de un flujo, con el
  tipo `flujo` y un esquema cerrado. Se aplica como una edición de la persona en el panel; comentar y pedir detalle llaman a la IA.
  Los comentarios salen del PC por el ocultador. Necesita el relay y la app nuevos.

## [2.10.0-rc.15] - 2026-11-06

### Agregado

- **Editar y comentar un flujo en el panel:** en la pestaña de un flujo, un clic elige un nodo. Se puede comentar, pedir detalle, editar, agregar, mover, poner flechas y quitar. qdd valida cada cambio. La IA recibe un comentario o un detalle en el acto, y las ediciones al pulsar «Enviar a la IA» o con tu próximo mensaje. Responde bajo el mismo nodo con `qdd flujo comentar`. Lo que aún no recibe se puede deshacer. `Flujos` también deja crear uno nuevo desde el panel.
- **`qdd doctor` revisa QDD Móvil:** dice si este PC está emparejado, si el servicio corre, si llega al relay, si el proyecto se
  publica y si el último envío salió. `--fix` arranca el servicio y anota el proyecto; emparejar sigue siendo con `qdd login`. No
  usa la red ni comprueba lo que dibuja la app.
- **`qdd doctor` revisa también las colas del teléfono y los flujos:** nombra una cola apartada por ilegible y los flujos que no se
  pueden usar. No los toca, porque corregirlos lo decide la persona.

### Corregido

- **Un campo del panel recién abierto no recibía lo que se escribía (FND-166):** se iba al chat, porque el campo necesitaba un segundo clic que nada avisaba. En los flujos el campo toma las teclas solo; si Claude Code no las suelta, el panel lo dice.
- **Una pregunta abierta en la consola ya no se va del teléfono a los 30 min (FND-165):** seguía abierta en la terminal, con el aviso
  de que se podía responder desde QDD Móvil, pero allí ya no estaba. Ahora sigue en el teléfono mientras la sesión la espere.

### Cambiado

- **Las preguntas ahora están en «▶ Para ti» (issue #149):** se quita la sección «⏳ Te esperan», que duplicaba esa lista y a la
  persona no le dejaba contestar. Al tocar una pregunta, llega al chat y se responde ahí.

## [2.10.0-rc.14] - 2026-11-06

### Seguridad

- **Trae el arreglo de `2.9.11` (FND-164):** `qdd dashboard` ya solo atiende a este PC.

### Agregado

- **Flujos para QDD Móvil:** el panel que se publica lleva el resumen de cada flujo, y los nodos viajan en un evento `flujo` aparte, después de cada estado completo y cuando cambia su huella. Un cambio de tarea no reenvía el flujo. Un flujo pesa como mucho 100 KB.

### Cambiado

- **Declarar la tarea en curso ya es una regla (MEJ-021):** el bloque de `AGENTS.md` (revisión 4) y la forma de trabajo piden `activar` al empezar y `terminar` al acabar o quedar esperando. `qdd_estado set` lo recuerda si un ✅ llega sin tarea declarada en la sesión, pero no la enciende. `qdd doctor --fix` renueva el bloque en cada proyecto.

## [2.10.0-rc.13] - 2026-11-06

### Corregido

- **Con QDD Móvil, cada sesión ya no lanza un proceso cada 5 s (MEJ-020, #108):** el proceso `qdd consola seguir`, que ya
  estaba abierto, hace el latido y entrega lo mandado desde el teléfono. Con una versión anterior de qdd, el panel sigue
  latiendo como antes.
- **Una salida en vivo larga ahora muestra lo último (FND-163):** Claude Code dibuja solo los primeros 100.000 caracteres de un
  texto, así que una herramienta ruidosa o una barra de progreso mostraban lo viejo. El panel guarda lo último, hasta 64 KB, y
  redibuja como mucho cada 100 ms.
- **El panel ya no acumula lo que dejó de existir (MEJ-014, #114):** olvida las tareas y los pedidos que ya no están, no suma
  lecturas si la anterior sigue en curso, y una respuesta vieja no pisa a una más nueva.

## [2.10.0-rc.12] - 2026-11-06

### Agregado

- **El panel muestra qué te preguntan (#149):** la sección «⏳ Te esperan» trae el texto de los ⏳ que esperan a la
  persona, recortado. Al tocar uno se ve entero y, si es una pregunta, se responde ahí mismo. La respuesta le llega a la
  IA de esa sesión, y el teléfono ya no vuelve a preguntarla.

### Cambiado

- **Mensajes, sesiones y herramientas ya no esperan a qdd (MEJ-015, #113):** un mensaje y el inicio de sesión esperan a lo más
  400 ms; si qdd tarda más, lo que tenía que decirle a la IA viaja con el mensaje siguiente en vez de perderse. El final de una
  herramienta no espera nada, y una ráfaga de herramientas de qdd lanza una lectura en curso y una más, no una por cada una.

### Corregido

- **`qdd audit` nombra la prueba que falló aunque la suite sea larga:** guardaba solo las últimas 60 líneas, así que podía
  decir «1 fail» sin decir cuál. Ahora también muestra las líneas anteriores que dicen «fail», hasta 40.
- **`qdd remoto` ya no gasta CPU por cada latido de las sesiones (FND-161):** cada sesión reescribía cada 5 s su cola del
  teléfono aunque siguiera vacía, y el servicio volvía a leer todos los proyectos. Con 7 proyectos y sin teléfono conectado
  usaba del 13 al 21 % de un núcleo. Las colas ya no se escriben si no cambiaron.

## [2.10.0-rc.11] - 2026-11-05

### Agregado

- **Flujos en el panel (RFC 0012, fase 1):** `qdd flujo crear` guarda un flujo de cajas y círculos en
  `.qdd/project/flujos/`, versionado. El panel lo abre en su propia pestaña, hacia abajo y con scroll, y solo materializa lo
  que cambió. Por ahora la persona lo ve; editarlo llega en la fase 2.

### Cambiado

- **Menos mensajes de Realtime:** `hace`, el pulso de una tarea en curso y los reintentos de un pedido ya no mandan el panel
  entero al teléfono por sí solos. Viajan con el siguiente cambio real, así que en la app pueden estar atrasados hasta
  entonces.
- **«✓» y «Quitar» reemplazan a la ✕ en cada tarea del panel.** Sirven para las tareas atrapadas por algo que no se puede
  validar en producción, o que ya no quieres ver. Los dos piden «s  Sí».
  - **✓ (ya lo vi):** deja cada etapa en ⏳ en ✅ con la nota «la vio la persona el DD/MM, sin medición» y cierra la tarea si
    no queda nada por hacer. Una etapa con falla o sin empezar, o una aprobación pendiente, sigue como está.
  - **Quitar:** saca la tarea sin pedir motivo; queda en el historial con «la quitó la persona desde el panel».

### Corregido

- **El panel lanza menos procesos con el flujo abierto (MEJ-020):** las herramientas y si toca el ciclo de mejoras llegan por
  `qdd consola seguir`, sin `herramienta listar` cada 10 s ni `mejoras ciclo` cada minuto. El latido del teléfono sigue
  siendo un proceso.
- **El CI también corre en los PRs hacia una rama `feat/**`.** Un PR apilado no tenía CI, y su merge desde el panel quedaba
  esperando para siempre.

## [2.10.0-rc.10] - 2026-11-05

### Agregado

- **El aviso del teléfono se quita cuando ya no hace falta (FND-028 de qdd-movile):** cuando un proyecto ya no tiene pedidos
  o fichas esperando, `qdd remoto` pide `pedido_resuelto` o `ficha_resuelto`. Necesita la app con el receptor (qdd-movile
  6551f3f).

### Corregido

- **Un proyecto que pierde su panel un momento ya no repite el push al volver:** lo que esperaba sigue contado como visto.
- **Un ⏳ de «usuario» también es de la persona (FND-159):** `esperan_persona` daba 0 en un tablero con 4 de ellos. Ahora
  cuentan, y si preguntan algo también llegan al teléfono como pregunta y al chat como espera.

## [2.10.0-rc.9] - 2026-11-05

### Seguridad

- **El tablero y los pedidos ya no salen al teléfono sin pasar por el ocultador (FND-157):** solo se tapaban las preguntas,
  lo mandado desde el teléfono y las elecciones del modo goal. El teléfono actúa por id, así que tapar un comando no cambia
  lo que se ejecuta.

### Corregido

- **El teléfono muestra lo que sigue a la pregunta de un ⏳ (FND-155):** el texto va desde la primera pregunta hasta el final
  del bloqueo. Lo anterior a la pregunta sigue sin publicarse.
- **Un seguro contra lanzar dos veces lo mismo (FND-154):** lo que terminó bien en las últimas 24 h no se vuelve a pedir,
  aunque cambie el texto (cerrar el mismo issue, mergear el mismo PR). Repetirlo de verdad se dice con `--de-nuevo`.

### Agregado

- **`esperan_persona` en el panel (FND-156):** cuenta los ⏳ de la persona de todas las tareas pendientes, incluidas las
  ocultas, para que el teléfono no muestre menos de los que hay.
- **La sesión se entera cuando se retira su pedido (RFC 0017 §1, MEJ-010):** antes el pedido solo cambiaba de estado en la
  cola. Ahora `qdd consola ciclo --contexto --sesion` se lo dice una vez a la sesión que lo pidió, con el pedido y la tarea.
  Un pedido sin sesión sigue visible en la cola y no avisa a nadie.

## [2.10.0-rc.8] - 2026-11-05

### Corregido

- **Un pedido ya no se retira por el trabajo de otra sesión (FND-153, MEJ-010):** sin tarea explícita, solo hereda la tarea
  en curso de su propia sesión.
- **Una tarea cerrada ya no aparece en las esperas (FND-151, MEJ-010):** `esperas` y `fallas` solo traen tareas abiertas.
- **`qdd remoto dispositivos` y `revocar` ya no piden la URL y la clave (issue #102):** en un PC sin relay mandan a
  `qdd login`. La ayuda de `qdd remoto` también dice que esas banderas son solo para quien opera su propio relay.

### Agregado

- **`qdd estado check` avisa de las tareas desfasadas (MEJ-010):** las abiertas con todo listo y las cerradas que conservan
  un ⏳. No bloquea el gate.

## [2.10.0-rc.7] - 2026-11-05

Trae también lo de 2.9.10 (MEJ-002).

### Corregido

- **En un proyecto nuevo el panel da un comando que funciona (FND-147):** `qdd estado init` falla sin `--etapas`, así que
  el aviso ahora lo incluye y explica que las etapas son las columnas del tablero.

## [2.10.0-rc.6] - 2026-11-05

### Corregido

- **El panel ve los comandos qdd corridos por PowerShell o con una ruta entre comillas (FND-146, MEJ-016):** se relee tras
  ellos, y la negativa a que una IA active el modo goal también los cubre.

## [2.10.0-rc.5] - 2026-11-05

Trae también lo de 2.9.9 (FND-143).

### Corregido

- **La barra de estado ya no queda viva si nadie cierra su stdin (FND-145, MEJ-018):** corta a los 3 s y sale sin imprimir
  nada ni fallar.
- **Los efectos del panel ya no dejan un hueco en blanco largo (FND-144, issue #111):** si los cuadros tardan más de 250 ms,
  el proceso se corta y se dibuja el texto sin efecto. Antes, en una máquina lenta, el hueco podía durar hasta 3 s. Abrir el
  panel ya pedía todos los cuadros en un solo proceso desde 2.10.0-rc.1.

## [2.10.0-rc.4] - 2026-11-05

Trae también lo de 2.9.8 (FND-143).

### Cambiado

- **La barra de estado ya no arranca Node cada 2 s:** llama al binario de qdd por su ruta y, si esa ruta ya no existe, a
  `qdd` como antes. Cada llamada gasta unas 4 veces menos CPU (en Linux, de 50-65 ms a 12-16 ms); en Windows no está
  medido. `settings.json` guarda la ruta, y `qdd consola activar` o `qdd doctor --fix` la renuevan al actualizar. Un
  binario compilado a mano no la escribe, y una barra propia no se toca.

### Corregido

- **El botón «⬆ Actualizar qdd» ya no deja a qdd sin binario con npm 12 (FND-143):** instala con
  `--allow-scripts=qdd-framework`, igual que el aviso de Windows.

## [2.10.0-rc.3] - 2026-11-05

### Corregido

- **En Windows, npm ya puede actualizar qdd con sesiones abiertas (FND-142, issue #107):** `qdd.exe` vive fuera del paquete,
  en `%LOCALAPPDATA%\qdd\bin\<versión>`, así que lo que corre no se interpone. Lo que corre se cierra solo al ver la
  versión nueva, y el panel lo reabre. Windows tiene ahora el mismo botón «⬆ Actualizar qdd» que Linux y macOS. `qdd doctor`
  revisa que la tarea de `qdd remoto` lance la versión instalada (REM-04). La primera actualización desde una versión
  anterior todavía pide cerrar las sesiones.

## [2.10.0-rc.2] - 2026-11-05

Trae también lo de 2.9.7 (FND-141).

### Agregado

- **El panel avisa de una versión nueva de qdd y la instala con una tecla:** una vez por hora pregunta a npm, aparte y sin
  demorar al panel, y deja el pedido opcional «⬆ Actualizar qdd a X» del canal instalado. Corre solo si confirmas con
  «s  Sí»; el modo goal no lo lanza. Un binario que no vino de npm no pregunta nada. En Windows, el panel avisa con lo que
  hay que hacer y no ofrece botón, porque npm no puede reemplazar `qdd.exe` mientras algo lo usa (issue #107).

## [2.10.0-rc.1] - 2026-11-05

### Cambiado

- **El panel ya no lanza un proceso cada 2 s (RFC 0016, MEJ-020):** recibe los cambios de un solo proceso, `qdd consola
  seguir`. En Linux, en reposo, su costo de CPU baja entre 71 % y 82 %, y un cambio llega a la pantalla en unos 500 ms. Sin
  medir todavía en Windows. A cambio suma unos 25 MB de memoria por sesión.
- **El panel llama directo al binario:** deja de pasar por el envoltorio de Node y, en Windows, por `cmd`.
- **Las sesiones sin terminal (`claude -p`) ya no leen el panel cada 2 s.**
- **Abrir el panel pide los cuadros de los efectos en un solo proceso (MEJ-017):** antes lanzaba uno por cada celda que
  cambiaba. Un qdd anterior no entiende el lote y el panel se dibuja sin efecto.
- **El panel se ve como QDD Móvil:** usa los colores de la marca en cualquier terminal, y el ámbar relleno marca «▶ N para
  ti». Cada tarea ocupa una fila y el estado de cada objetivo queda a la derecha. La versión pasa a la cabecera y el pie se
  acorta. En un tema claro de terminal, el ámbar y el verde pierden contraste.
- **El botón Modo goal pasa a la tecla `z`.**

### Corregido

- **Los botones de la cabecera del panel ya no se pisan (FND-140):** van en su propia fila, sin emojis anchos.
- **`k` ya no es a la vez Config y Modo goal (FND-139).**

## [2.9.11] - 2026-11-06

### Seguridad

- **`qdd dashboard` ya solo atiende a este PC (FND-164):** escuchaba en todas las interfaces y sus endpoints de escritura no
  pedían autenticación. Mientras estaba abierto, otro equipo de la misma red o una página abierta en el navegador podían
  leer el estado, cambiar políticas, escribir archivos en el proyecto y enviar instrucciones al LLM configurado. Ahora
  escucha en `127.0.0.1` y rechaza las peticiones que no vienen de su propia página. Si lo usabas desde otro equipo, ya
  no se puede: usa un túnel SSH. Actualiza si alguna vez lo abriste en una red compartida, y revisa si hay cambios que no
  reconoces en esos proyectos.

## [2.9.10] - 2026-11-05

### Corregido

- **El gate ya no deja pasar un finding sin estado (MEJ-002):** un finding sin `status` no se contaba como abierto ni como
  resuelto, y nadie se enteraba. Ahora bloquea, y QDD no lo infiere: escribe `open`, `resolved` o `wontfix`. Desde 2026-11-06,
  un finding resuelto también tiene que decir en `test_result` qué lo prueba; los anteriores no se revisan.

## [2.9.9] - 2026-11-05

### Corregido

- **Con npm 12, qdd ya no queda sin binario si instalas sin la bandera (FND-143):** la primera vez que usas `qdd`, baja su
  binario desde el release de GitHub de esa versión. Tarda unos 30 s y queda fuera del bloqueo de scripts de npm.

## [2.9.8] - 2026-11-05

### Corregido

- **Con npm 12, instalar o actualizar qdd lo dejaba sin binario (FND-143):** npm 12 bloquea por defecto los scripts de
  instalación, y terminaba bien aunque qdd quedara sin funcionar. El comando correcto es ahora `npm install -g qdd-framework
  --allow-scripts=qdd-framework`, y es el que indican la documentación y los mensajes de qdd. Si ya te pasó, ese comando lo
  arregla.

## [2.9.7] - 2026-11-05

### Corregido

- **La respuesta a un ⏳ desde el teléfono ya no dice «entregada» antes de tiempo (FND-141):** queda en curso hasta que
  una sesión de Claude Code la acepta. Si no llega, el teléfono ve la falla o el vencimiento y el ⏳ se puede responder de
  nuevo. Antes quedaba como respondida y no se podía repetir en 24 h. Límite conocido: confirmar significa que Claude Code
  la aceptó, no que la IA ya la leyó.

## [2.9.6] - 2026-11-05

Corrección de 2.9.5, sin esperar uso medido por ser un arreglo. Trae lo mismo que 2.9.6-rc.1.

## [2.9.6-rc.1] - 2026-11-05

### Corregido

- **El teléfono ya no dice «entregada» si la instrucción no llegó (FND-138):** queda entregada solo cuando Claude Code la
  acepta. Si la rechaza, pasa a falla con su motivo; sin confirmación en 15 min, pasa a falla diciendo que puede no haber
  llegado. Nunca se entrega dos veces, así que en ese caso hay que volver a mandarla. Hace falta reiniciar Claude Code
  para cargar el panel nuevo. Límite conocido: confirmar significa que Claude Code la aceptó, no que la IA ya la leyó.

## [2.9.5] - 2026-11-05

Corrección de 2.9.4, sin esperar uso medido por ser un arreglo. Trae lo mismo que 2.9.5-rc.1: `npm install` ya no termina
bien sin dejar el binario (FND-137).

## [2.9.5-rc.1] - 2026-11-05

### Corregido

- **`npm install` ya no termina bien sin dejar el binario (FND-137, issue #106):** en Windows usa el `tar` del sistema, aunque
  Git Bash ponga otro primero en el `PATH`. Si la descarga o la extracción fallan, o el binario no responde, la instalación
  termina con error y dice el motivo; npm conserva la versión anterior. Sin probar todavía en un Windows real.

## [2.9.4] - 2026-11-04

Corrección de 2.9.3, sin esperar uso medido por ser un arreglo. Trae lo mismo que 2.9.4-rc.1.

## [2.9.4-rc.1] - 2026-11-04

### Corregido

- **`qdd remoto desactivar` ya no borra el servicio de otro `QDD_HOME` (FND-136):** saca del relay la máquina de ese
  `QDD_HOME`, pero deja el servicio que publica otro y lo dice.
- **`qdd remoto emparejar` manda a `qdd login`:** en un PC sin relay ya no pide banderas, y avisa que sin servicio el PC
  no publica.

### Agregado

- **Los pedidos del panel dicen su tarea** (campo `tarea`, opcional): QDD Móvil puede dejar de ofrecer seguir con una tarea
  que ya tiene un pedido abierto.

## [2.9.3] - 2026-11-04

Corrección de 2.9.2, sin esperar uso medido por ser un arreglo. Trae lo mismo que 2.9.3-rc.1: un pedido ya no falla porque
se está reinstalando qdd (FND-135).

## [2.9.3-rc.1] - 2026-11-04

### Corregido

- **Un pedido ya no falla porque se está reinstalando qdd (FND-135):** si falta el binario, `qdd` dice que se está
  actualizando y sale con 75, y el panel lo muestra como «todavía no» en vez de marcarlo como fallo. Límite conocido: si
  en ese instante falta el propio `qdd.js`, sigue siendo un fallo.

## [2.9.2] - 2026-11-04

Corrección de 2.9.1, sin esperar uso medido por ser un arreglo. Trae lo mismo que 2.9.2-rc.1: CERT-021 ya no bloquea leer
la query string de una URL (FND-134).

## [2.9.2-rc.1] - 2026-11-04

### Corregido

- **CERT-021 ya no bloquea leer la query string (FND-134):** `r.URL.Query()` y `u.Query()` salían como una consulta a la
  base sin contexto. Una consulta sigue marcada si lleva su SQL. Límite conocido: `stmt.Query()` sin parámetros tampoco
  se marca.

## [2.9.1] - 2026-11-04

Corrección de 2.9.0, sin esperar uso medido por ser un arreglo. Trae lo mismo que 2.9.1-rc.1: `qdd login` funciona en
un PC nuevo, sin banderas (FND-133).

## [2.9.1-rc.1] - 2026-11-04

### Corregido

- **`qdd login` funciona en un PC nuevo, sin banderas (FND-133, issue #102):** qdd ahora trae el relay de QDD Móvil,
  solo con su clave publicable. `--relay` y `--clave` quedan para quien opera su propio relay.

## [2.9.0] - 2026-11-04

Versión estable (`npm i -g qdd-framework`). Reúne las 6 previews de 2.9.0, con 15,0 h de uso real medido (VER-05,
`.qdd/project/evidence/uso/v2.9.0.yaml`). El detalle de cada cambio está en su preview, abajo. Lo principal:

- **Modo goal por proyecto** (`qdd goal`, RFC 0015): lo activas tú, desde el panel o el teléfono. Todavía no se ha
  probado en una sesión real ni en el teléfono.
- **`qdd login`** empareja este PC con QDD Móvil (issue #102). Todavía pide `--relay` y `--clave`.
- **QDD Móvil**: límites de Claude en el teléfono, sin saltos entre sesiones (FND-130), y no pregunta lo que nadie puede
  resolver (FND-124).
- **Reglas para las IAs**: lo que necesita de la persona no se queda en el chat, y `qdd doctor` renueva el bloque al
  actualizar (FND-126).
- **Pedidos**: uno armado no corre dos veces (FND-125), y uno sobrevive a detener el servicio que lo lanzó (FND-128,
  FND-129).

## [2.9.0-rc.6] - 2026-11-04 (preview)

Sexta **preview** de 2.9.0: `npm i -g qdd-framework@preview`.

### Añadido

- **`qdd login` empareja este PC con QDD Móvil** (issue #102). Es `qdd remoto activar` con otro nombre. Todavía pide
  `--relay` y `--clave`, porque esta versión no trae un relay por defecto; antes `qdd login` se tomaba como una intención
  en texto libre.

## [2.9.0-rc.5] - 2026-11-04 (preview)

Quinta **preview** de 2.9.0: `npm i -g qdd-framework@preview`. El arreglo también sale en `latest` como 2.8.6.

### Arreglado

- **El panel ya no pide dos veces mergear el mismo PR** (FND-132, MEJ-001). Un pedido que nombra el PR reemplaza al
  automático «Mergear #N» aunque este haya llegado primero. Si ya lo confirmaste, sigue su curso; si descartas el otro
  pedido y el PR sigue abierto, el automático vuelve.
- **Una publicación ya no queda a medias por una carrera al crear el release** (FND-131). Los binarios se suben de a uno.

## [2.9.0-rc.4] - 2026-11-04 (preview)

Cuarta **preview** de 2.9.0: `npm i -g qdd-framework@preview`. El arreglo de los pedidos sale también en `latest` como 2.8.4.

### Arreglado

- **Un pedido ya no muere con la unidad de systemd que lo lanzó** (FND-129, MEJ-003). En Linux corre en su propio scope, así
  que sigue aunque se detenga el servicio o la sesión desde la que se lanzó. Sin systemd de usuario se lanza como antes.
- **Los límites de Claude ya no saltan en el teléfono** (FND-130). Cada sesión abierta escribía lo último que sabía, y una
  inactiva hacía retroceder el dato. Ahora, en la misma ventana, lo usado solo sube.

## [2.9.0-rc.3] - 2026-11-04 (preview)

Tercera **preview** de 2.9.0: `npm i -g qdd-framework@preview`.

### ✨ Añadido

- **Modo goal por proyecto** (`qdd goal`, RFC 0015, decisión de la persona). Lo activas tú, desde el panel o el teléfono.
  Mientras está activo, qdd elige la opción recomendada de cada pregunta, corre los pedidos de la IA y aprueba las fichas
  sin esperarte, y cada elección queda registrada. Con 2 min de silencio le recuerda a la IA lo que queda; tras 3
  recordatorios sin avance real se detiene y avisa. Sin una sesión abierta o sin límite de uso se pausa. Las preguntas de
  diseño de un blueprint, las que no traen opción recomendada, los permisos de Claude Code y los PRs que qdd ofrece solo
  siempre te esperan.
- **QDD Móvil**: `goal` opcional en el panel, `goal activar` y `goal desactivar` desde el teléfono, y dos avisos nuevos:
  `goal_pausado` y `goal_terminado`.

### Arreglado

- **Actualizar qdd desde un pedido del teléfono ya no se corta a la mitad** (FND-128). `qdd remoto` se reinicia al ver un
  qdd nuevo, y el sistema mataba también lo que había lanzado: npm revertía la instalación y el pedido fallaba con código
  -1. `qdd doctor --fix` le agrega la línea que falta al servicio ya instalado (REM-03).
- **El botón «Ciclo de mejoras» se ve en el panel**, con la tecla `m` (FND-127).

## [2.9.0-rc.2] - 2026-11-04 (preview)

Segunda **preview** de 2.9.0: `npm i -g qdd-framework@preview`.

### ✨ Añadido

- **Límites de Claude en QDD Móvil** (pedido de qdd-movile). `qdd remoto` publica cuánto se usó de la ventana de 5 h y de la
  semana, y cuándo se reinician, sin gastar tokens. Claude Code los entrega a la barra, que deja el último valor en
  `<QDD_HOME>/claude/limites.json`. El campo `limites` es opcional en `estado` y `delta`, y no cambia la versión del
  protocolo.

### Arreglado

- Al actualizar qdd, los proyectos seguían con las reglas viejas para las IAs: el bloque de `AGENTS.md` y de los otros
  archivos solo se escribía si faltaba el archivo. Ahora `qdd doctor` avisa del bloque viejo y `qdd doctor --fix` lo
  renueva, conservando lo demás. Un qdd más viejo no pisa el bloque de uno más nuevo (FND-126).

## [2.9.0-rc.1] - 2026-11-03 (preview)

Primera **preview** de 2.9.0: `npm i -g qdd-framework@preview`. Sus dos arreglos salen también en `latest` como 2.8.3,
en la rama `hotfix/2.8.3`.

### Arreglado

- **QDD Móvil no pregunta lo que nadie puede resolver** (FND-124). Sin una sesión de Claude Code abierta en el proyecto,
  un ⏳ no llega al teléfono y su respuesta se rechaza. Antes la respuesta vencía a los 30 min mientras la pregunta seguía
  como «respondida», y se perdía.
- **Un pedido armado ya no puede ejecutarse dos veces** (FND-125, MEJ-008). Si dos procesos quedaban esperando el mismo
  pedido, los dos lo corrían al mismo tiempo: en sfrc lanzó dos despliegues en el mismo segundo. Ahora solo lo corre su
  dueño, y el otro termina sin correrlo.

### ✨ Añadido

- **Lo que la IA necesita de la persona no se queda en el chat** (pedido de sfrc). El bloque de `AGENTS.md` trae la
  regla: una decisión va con la herramienta de preguntar (AskUserQuestion en Claude Code) y una opción recomendada; lo
  que no es una elección, como ⏳ escrito con `¿…?`; un comando, con `qdd_consola_pedir`.
- `qdd estado check` avisa, sin bloquear, de los ⏳ de la persona que no le llegan como pregunta: los que no preguntan
  nada y que ningún pedido abierto cubre. El código de salida no cambia. Con `--json`, salen en `esperas_sin_pregunta`.

## [2.8.2] - 2026-11-03

Corrección (`npm i -g qdd-framework@latest`). Sale a la vez como 2.8.2-rc.1 en `preview`.

### Cambiado

- **QDD Móvil pregunta solo lo que es pregunta** (MEJ-004, FND-123). Un ⏳ de la persona llega al teléfono solo si su
  bloqueo pregunta algo (`¿…?`), y el teléfono ve solo esa pregunta, no el historial de la tarea. Si nombra con `[tecla]`
  un pedido abierto del panel, no se publica, porque el pedido ya lo cubre.

## [2.8.2-rc.1] - 2026-11-03 (preview)

Preview (`npm i -g qdd-framework@preview`) con el mismo cambio de 2.8.2 (FND-123).

## [2.8.1] - 2026-11-03

Corrección (`npm i -g qdd-framework@latest`). El mismo arreglo sale a la vez como 2.8.1-rc.1 en `preview`: el bug estaba
en todas las versiones publicadas, de los dos canales.

### Arreglado

- **Windows: sin ventanas de consola** (FND-122, issue #83). Los procesos que qdd lanza aparte (`qdd consola prs`, los
  pedidos del panel y `qdd incidentes publicar`) corren con una consola oculta. Antes cada `gh` o `git` que llamaban abría
  una ventana negra visible, cada minuto y por proyecto. `qdd consola prs` tampoco llama a `gh` si el remote no es de
  GitHub.

## [2.8.1-rc.1] - 2026-11-03 (preview)

Preview (`npm i -g qdd-framework@preview`) con el mismo arreglo de 2.8.1 (FND-122, issue #83).

## [2.8.0] - 2026-11-03

Versión estable (`npm i -g qdd-framework`). Reúne las 27 previews de 2.8.0, con 60,3 h de uso real medido (VER-05,
`.qdd/project/evidence/uso/v2.8.0.yaml`). El detalle de cada cambio está en su preview, abajo. Lo principal:

- **Uso medido y canal preview** (RFC 0010): cada versión nueva sale primero como `@preview` y pasa a `latest` con sus
  horas de uso real.
- **Panel de Claude Code**: el tablero por tareas, con objetivos, árbol por dependencias y ciclos con nombre
  (`estado-por-tareas` 2.2.0). También los pedidos para la persona, que se ordenan, se arman y avisan al chat; los
  blueprints a un clic; las herramientas del proyecto; el atajo desde el chat; y las animaciones solo de lo que cambió.
- **Ids de QDD** para tareas, objetivos y ciclos: qdd los da o los valida (FND-109).
- **Blueprints**: un solo `estado-por-tareas`, que reemplaza a `estado-por-frente` (RFC 0009), además del índice, la
  calidad de la receta, la conformidad en el gate (BP-08, con `fuera` también en lo aplicado: FND-121), los aportes en
  paralelo y la lectura por HTTPS.
- **QDD Móvil** (RFC 0013): `qdd remoto` publica el panel de cada proyecto en el teléfono, que aprueba pedidos, responde
  las preguntas de la consola y los ⏳ de la persona, y ejecuta comandos qdd e instrucciones para la IA (Pro). El servicio
  vuelve a arrancar solo cuando se actualiza qdd (FND-120).
- **Cola de mejoras** (`qdd mejoras`): los reportes de las sesiones y los issues de GitHub se vuelven tareas del
  objetivo `mejora-continua`, 2 abiertas como mucho.

## [2.8.0-rc.27] - 2026-11-03 (preview)

Vigesimoséptima **preview** de 2.8.0: `npm i -g qdd-framework@preview`. La cola de mejoras (`qdd mejoras`, objetivo
mejora-continua) y lo que el estado espera de la persona como pregunta en QDD Móvil.

### ✨ Añadido

- **Cola de mejoras** (`qdd mejoras`). Lo que reportan las sesiones de Claude Code (`qdd mejoras reportar --proyecto qdd`)
  y los issues de GitHub de quien tiene acceso al repositorio, o con la etiqueta `mejora`, queda como `MEJ-NNN` en
  `.qdd/project/cola_de_mejoras/`. El ciclo es determinista y mantiene el objetivo `mejora-continua` del tablero con
  una tarea por mejora, 2 abiertas como mucho: primero las de seguridad o bugs, después en orden de llegada. Solo lo que
  ya es tarea pasa a la IA, que recibe un aviso 🔁. El panel corre el ciclo solo cada 10 min mientras hay una sesión
  abierta, y el botón «Ciclo de mejoras» lo fuerza. Cerrar el issue de una mejora hecha queda como pedido en el panel.
  `qdd doctor` revisa la cola (MEJ-01 a MEJ-03).

- Lo que el estado espera de la persona llega a QDD Móvil como pregunta: cada ⏳ «de la persona» de una tarea abierta
  se responde desde el teléfono con texto libre, y la respuesta le llega a la IA del proyecto como un mensaje tuyo. Sale
  sola cuando la celda cambia (pedido de sfrc, RFC 0013).

## [2.8.0-rc.26] - 2026-11-03 (preview)

Vigesimosexta **preview** de 2.8.0: `npm i -g qdd-framework@preview`. Dos arreglos: `qdd remoto` vuelve a arrancar solo
tras actualizar qdd, y un módulo que aplica un blueprint puede declarar escenarios fuera.

### 🐛 Corregido

- Al actualizar qdd, `qdd remoto` sigue corriendo la versión vieja hasta que alguien reinicia el servicio: lo que
  el teléfono pedía a la versión nueva quedaba cargando. Ahora el servicio nota que su qdd cambió en el disco y vuelve
  a arrancar solo con la nueva (FND-120).
- La ficha de un módulo que aplica un blueprint ya puede declarar, con su motivo, los escenarios que su código no hace:
  `fuera: {ID: motivo}` en la raíz, junto a `blueprint`. Antes BP-08 pedía `implementa: fuera`, que para un módulo
  aplicado no servía. La plantilla de ficha muestra las dos formas (FND-121).

## [2.8.0-rc.25] - 2026-11-03 (preview)

Vigesimoquinta **preview** de 2.8.0: `npm i -g qdd-framework@preview`. QDD Móvil, sprint 2: ejecutar desde el teléfono.
Trae también lo de rc.24, que no llegó a npm: responder desde el teléfono lo que pregunta la consola.

### 🐛 Corregido

- Las pruebas de `pkg/remoto` que lanzan un pedido esperan a que termine: si no, el pedido escribía en la carpeta
  temporal mientras se borraba y el gate del CI fallaba a veces (FND-119).

### ✨ Ejecutar desde el teléfono (QDD Móvil, solo Pro)

- Con Pro, la persona manda a un proyecto, desde QDD Móvil y con biometría, un **comando qdd** o una **instrucción para
  la IA**. Siempre a través de qdd: nunca un comando de shell.
- `qdd remoto` atiende el evento firmado `ejecutar` del relay (`public.qdd_remoto_ejecutar`), una sola vez por
  solicitud. Exige Pro con el plan del último latido y vuelve a validar el texto.
- Un comando qdd se parte sin shell y solo corre si está en la lista permitida: `audit`, `doctor` (sin `--fix` ni
  `-f`), `estado`, `estado check`, `modules list`, `blueprint list` y `blueprint quality [id]`, con banderas de solo
  lectura. Corre con el qdd de este PC en la carpeta del proyecto, a lo más 10 min y uno por proyecto a la vez.
- El teléfono recibe `accion_estado`: `recibida` → `ejecutando` → `hecha` o `fallo`, con el final de la salida sin
  secretos (hasta 4000 caracteres). Lo que no se permite vuelve `rechazada` con el motivo.
- Una instrucción llega a la sesión abierta de Claude Code del proyecto como un mensaje de la persona, «📱 La persona te
  escribe desde «<teléfono>» (QDD Móvil)…»: nunca como un comando `!` o `/`, y a una sola sesión. Sin sesión abierta
  queda en cola y vence a los 30 min. Desde el teléfono también se puede retirar mientras espera.
- `qdd consola telefono latir | fallo | lista`: el mod `qdd-tareas` late cada 5 s en las sesiones interactivas y
  recibe lo que le toca.
- `qdd consola panel --json` trae `ejecutar: {qdd, ia, sugeridos}` con QDD Móvil activado y Pro. Con `--con telefono`
  muestra lo mandado como pedidos `origen:"telefono"`, con `dispositivo` y `solicitud`.
- Diseño en la RFC 0013, «Ejecutar desde el teléfono»; las invariantes 1 y 7 ahora permiten esto, solo con Pro.

### 🐛 Corregido

- «Auditar» y las demás acciones aprobadas desde el teléfono devolvían el principio de la salida, y el veredicto (que
  va al final) no llegaba. Ahora llega el final, sin secretos (FND-118).

## [2.8.0-rc.24] - 2026-11-03 (sin publicar)

El tag existe, pero no llegó a npm: el gate del workflow falló por una prueba inestable (FND-119). Todo lo de esta
versión sale en rc.25. Vigesimocuarta preview de 2.8.0. QDD Móvil, sprint 1: el teléfono responde lo que
pregunta la consola.

### ✨ Responder desde el teléfono lo que pregunta la consola (QDD Móvil)

- Cuando Claude Code espera a la persona, la pregunta aparece en QDD Móvil y se responde desde el teléfono o desde la
  terminal: gana la primera, y el otro lado ve «ya respondida en …».
  - Preguntas de la IA (`AskUserQuestion`): una opción, varias, o «Otra» con texto libre.
  - Permisos de herramientas: desde el teléfono solo **permitir una vez** o **negar**; nunca «permitir siempre» ni
    cambios de reglas.
- El mod `qdd-tareas` las registra desde `PermissionRequest`, que corre junto al diálogo de la terminal. Solo lo hace en
  sesiones interactivas, en el hilo principal, y cuando el PC tiene QDD Móvil activado.
- `qdd consola pregunta registrar | esperar | cerrar | vencer | lista` maneja la cola local.
- `qdd consola panel --json --con pregunta` muestra las preguntas como acciones `tipo:"pregunta"`.
- `qdd remoto` las publica pasadas por el ocultador de secretos y atiende el evento firmado `responder` del relay
  (`public.qdd_remoto_responder`). Contesta con `accion_estado` (`hecha`, `ya_aprobado`, `vencida` o `rechazada`) una
  sola vez por solicitud.
- Una pregunta vence si deja de recibir latidos por 60 s, a los 30 min o cuando termina la sesión.
- Diseño completo en la RFC 0013, «Preguntas desde el teléfono».

### 🐛 Corregido

- Las pruebas corridas por un hook de git (el pre-push desde un worktree exporta `GIT_DIR`) ya no escriben commits ni
  tags en el repositorio real. El nuevo `pkg/gitenv` limpia ese entorno en cada paquete cuyas pruebas corren git, y
  los clones de blueprints corren git con el entorno limpio aunque qdd corra dentro de un hook (FND-117).

## [2.8.0-rc.23] - 2026-11-03 (preview)

Vigesimotercera **preview** de 2.8.0: `npm i -g qdd-framework@preview`. Arreglos de blueprints que encontraron las
sesiones que aportan blueprints.

### 🐛 Certificación y calidad de blueprints, más claras (FND-115, FND-116)

- La variante adaptada de `qdd blueprint certify` ya no inventa «otro <valor>» en un parámetro de texto sin otro
  ejemplo: conserva su valor por defecto.
- `qdd blueprint quality <carpeta>` con un `blueprint.yaml` inválido dice el error de YAML, en vez de «no está
  importado ni publicado».

### 🐛 El PR de `qdd blueprint propose` ya no falla con un mensaje largo (FND-114)

- El título es la primera línea del mensaje, recortada a 120 caracteres en total; el mensaje completo va en el cuerpo
  del PR y del commit. Antes, un mensaje largo pasaba los 256 caracteres de GitHub y el PR había que abrirlo a mano.

## [2.8.0-rc.22] - 2026-11-02 (preview)

Vigesimosegunda **preview** de 2.8.0: `npm i -g qdd-framework@preview`.

### ✨ Los PRs abiertos, a un clic en el panel (siempre opcionales)

- El panel ofrece solo un pedido «🔀 Mergear #N» por cada PR abierto del repositorio (no borradores, hasta 3). qdd
  los pregunta con `gh` a lo más cada minuto, aparte, sin demorar al panel.
- Confirmado, `qdd consola mergear-pr N` mergea solo con el CI verde; con el CI corriendo queda armado y mergea cuando
  pasa. Con el CI en rojo o el PR cerrado, no mergea.
- Son opcionales: no abren el panel, no le piden nada a la IA, ceden su lugar a un pedido de la IA, uno descartado no
  vuelve y uno de un PR que se cerró afuera sale solo. Un PR que un pedido de la IA ya nombra («#69») no se repite.

## [2.8.0-rc.21] - 2026-11-02 (preview)

Vigesimoprimera **preview** de 2.8.0: `npm i -g qdd-framework@preview`.

### 🐛 Dos aportes de blueprint en paralelo ya no se mezclan (FND-113)

- Cada aporte tiene su propio espacio en la carpeta central (un worktree `<clon>@<id>` en la rama `aporte/<id>`).
  Antes, pedir un aporte con otro a medio hacer respondía «Aporte listo: rama aporte/<el otro>» y dejaba los dos
  trabajos en el mismo clon, y propose los habría subido juntos. Un aporte abierto antes sigue donde estaba.

## [2.8.0-rc.20] - 2026-11-02 (preview)

Vigésima **preview** de 2.8.0: `npm i -g qdd-framework@preview`. Este qdd implementa `estado-por-tareas` 2.2.0, así
que un proyecto que lo usa ya puede subir a esa versión.

### 🐛 Un blueprint nuevo ya no traba el gate (FND-112)

- Si la organización publica una versión de un blueprint que implementa qdd antes de que exista un qdd que la
  implemente, `BP-06-BEHIND` ya no bloquea: el gate lo avisa como recomendación `BP-06-ESPERA-QDD` («quédate en la
  versión actual hasta instalar un qdd que implemente la nueva»). Antes, quedarse daba BP-06 y subir daba BP-08: no
  había salida. Con un qdd que ya la implementa, quedarse atrás vuelve a bloquear.

### ✨ estado-por-tareas 2.2.0

- Los escenarios de ids EPT-I01…I04 (nombrar no crea, crear da el id, una referencia equivocada sugiere la que era,
  sin duplicados) se prueban con los datos del blueprint.

### 🐛 Ids de QDD: lo que vio la certificación ciega de estado-por-tareas 2.2.0 (FND-109)

- Un id largo quita palabras del final en vez de cortar una por la mitad, y el sufijo `-2` acorta la base para seguir en 40 caracteres. Antes, un segundo id largo pasaba de 40 y qdd mismo lo rechazaba.
- Una referencia a 2 letras de un id de largo impar (como `barra`) también se sugiere.
- Un ciclo nuevo lleva un id de QDD; los que ya existían siguen valiendo.

## [2.8.0-rc.19] - 2026-11-02 (sin publicar)

El tag existe, pero no llegó a npm: el gate del workflow se trabó con FND-112. Todo lo de esta versión sale en
rc.20. Decimonovena preview de 2.8.0. El lado PC de QDD Móvil: ver y aprobar lo de todos
tus proyectos desde el teléfono.

### 📱 Consola remota: QDD Móvil (RFC 0013)

- **Registro de proyectos de la máquina:** `<QDD_HOME>/proyectos.json` (0600) guarda los proyectos de este PC. Lo anotan `qdd init` y cada mensaje con el panel; un worktree cuenta como su checkout principal. `qdd doctor` señala las rutas que ya no existen (REM-01) y un registro ilegible (REM-02); `--fix` las quita o lo aparta.
- **`qdd remoto estado [--json]`:** lo que este PC publicaría en QDD Móvil, sin red: el panel de cada proyecto (el mismo de `qdd consola panel --json`), o el error si la carpeta ya no está.
- **`qdd remoto activar` / `emparejar` / `desactivar`:** guarda el relay (solo la clave publicable; una secreta se rechaza), empareja el PC con un QR en la terminal y deja un servicio del usuario (systemd, launchd o una tarea de Windows) que lo publica. Con teléfonos conectados, el PC manda el estado y luego solo lo que cambió, agrupado cada 300 ms; sin teléfonos, solo dice que está encendido. Un latido por minuto informa las tareas en curso y respeta el límite de proyectos del plan. Un pedido o una ficha nueva avisa por push, solo con el tipo y el proyecto.
- **Aprobar desde el teléfono:** una acción llega firmada por el relay; el PC la verifica, vuelve a calcular el panel y la ejecuta como el panel, una sola vez aunque llegue repetida. Responde recibida → ejecutando → hecha o fallo; un pedido ya aprobado en el PC u otro teléfono responde «ya aprobado» con el lugar. Todo queda en `<QDD_HOME>/remoto/log.jsonl`, nunca en el relay. Al terminar un pedido aprobado desde el teléfono, llega un push con cómo terminó.
- **Dos teléfonos a la vez:** el segundo que abre una consola recibe lo último y, si está pausada, el motivo (FND-110). Un teléfono que vuelve se entera de cómo terminó lo que aprobó mientras no estaba (FND-111).
- **Rechazar desde el teléfono:** un pedido que nadie aprobó sale de la cola sin ejecutarse; uno ya aprobado dice dónde.
- **`qdd remoto dispositivos` y `qdd remoto revocar <id>`:** los teléfonos de tu cuenta; revocar uno lo saca al instante, en el relay y en este PC.
- **La consola en el teléfono:** mientras miras un proyecto, ves la conversación con la IA (los turnos y el nombre de cada herramienta, sin sus resultados) y los pedidos que terminan, con como mucho dos mensajes por segundo. Al dejar de mirar se corta. Si el plan no la permite, el teléfono dice por qué y el resto sigue.
- **Sin secretos:** todo lo que sale del PC pasa por un ocultador de claves, tokens, JWT, cadenas de conexión, llaves privadas y los valores de los `.env` del proyecto.

## [2.8.0-rc.18] - 2026-11-02 (preview)

Decimoctava **preview** de 2.8.0: `npm i -g qdd-framework@preview`. Los ids de las tareas los da o valida qdd.

### 🏷️ Ids de QDD: nombrar una tarea ya no la crea (FND-109)

- **Antes:** `qdd estado set <id> …` con un id que no existía creaba una tarea nueva, y le ponía el id como nombre. En notificador_de_ofertas la IA escribió `segmento-barra` y `qdd-estado` para tareas que ya existían (`barra-estado`, `estado-por-frente`), y hubo que descartarlas a mano.
- **Ahora, crear es explícito:** `qdd estado nueva "Quitar claves de Gemini"` le da el id `quitar-claves-gemini` y lo imprime. La IA hace lo mismo con la acción `nueva` de `qdd_estado`. `set` y `tarea` siguen creando con `--nombre`, pero solo con un id de QDD (minúsculas, números y guiones).
- **Un id equivocado se rechaza con sugerencia:** «no hay una tarea "qdd-estado": ¿quisiste decir estado-por-frente («qdd estado (RFC 0006)»)?». Vale para `set`, `cerrar`, `activar`, `tarea`, `--objetivo`, `--depende-de`, `--ciclo` y el `--tarea` de un pedido. No se escribe nada.
- **Sin duplicados:** una tarea cuyo nombre casi repite el de una abierta se rechaza nombrando la otra; `--aunque-exista` la crea igual.
- **Doctor:** señala las tareas abiertas que se llaman como su id o casi igual que otra, y dice cómo resolverlo. No decide cuál es la buena.
- **Compatibilidad:** los ids de un estado anterior siguen valiendo; el formato solo se exige al crear.

## [2.8.0-rc.17] - 2026-11-02 (preview)

Decimoséptima **preview** de 2.8.0: `npm i -g qdd-framework@preview`. El chat se entera de los pedidos que confirmas en el panel.

### 🔔 El chat se entera de lo que confirmas en el panel

- **Antes:** confirmabas un pedido en el panel y la conversación no se enteraba. La IA no sabía que ya corría ni cuándo terminaba, y había que contárselo.
- **Ahora, al confirmar:** arriba del prompt aparece «▶ [a] Publicar rc.16 · en marcha» (o «◷ … armado» si espera) mientras corre. La IA recibe una nota de que el pedido corre aparte, para que no lo repita.
- **Al terminar:** el resultado llega a la conversación como un mensaje del panel («✓ … terminó bien» o «✗ … falló», con el final de la salida). La IA lo lee en el acto y sigue sola. Además sale una notificación de escritorio.
- **Una sola vez:** solo avisa la sesión que lo confirmó, aunque tengas varias abiertas, y un reload del mod no lo repite. Lo decide qdd con `qdd consola avisado`.
- **Compatibilidad:** con un qdd anterior el pedido corre igual, sin aviso. El panel de OpenCode todavía no avisa.

## [2.8.0-rc.16] - 2026-11-02 (preview)

Decimosexta **preview** de 2.8.0: `npm i -g qdd-framework@preview`. Trae los ciclos con nombre (blueprint `estado-por-tareas` 2.1.0).

### 🔁 Ciclos con nombre: cada objetivo puede tener sus propias etapas

- **Qué cambia:** las etapas del estado eran una sola lista para todo el proyecto. Ahora esas etapas son el ciclo «proyecto» y puedes declarar otros ciclos, cada uno con sus etapas. Un objetivo sigue un ciclo, y sus tareas toman esas etapas sin cambiar las del proyecto.
  - `qdd estado ciclo agentico Concepto Artefacto Medición Puerta` declara el ciclo «agentico».
  - `qdd estado objetivo tutor --ciclo agentico` hace que el objetivo «tutor» lo siga.
  - La IA hace lo mismo con la acción `ciclo` de la herramienta `qdd_estado`.
- **Por qué:** un blueprint que trae su propio ciclo, como `ciclo-de-vida-agentico`, ya no choca con las etapas del proyecto.
- **No se pierde avance:** si una tarea cambia de ciclo, conserva las etapas que comparten los dos ciclos y agrega las nuevas en «·». Si eso borrara un ✅, ⏳ o ❌, no se escribe y el mensaje dice qué etapa es.
- **Cómo se ve:**
  - El reporte trae una tabla por ciclo, la del proyecto primero, y `qdd estado check` las revisa todas.
  - En los paneles de Claude Code y OpenCode, cada objetivo con ciclo propio muestra sus abreviaturas bajo su cabecera, y la leyenda nombra cada ciclo.
- **Compatibilidad:** un estado sin ciclos se guarda igual que antes. Con ciclos se guarda en formato 3, y un qdd anterior lo rechaza en vez de borrarlos al escribir.

## [2.8.0-rc.15] - 2026-11-01 (preview)

Decimoquinta **preview** de 2.8.0: `npm i -g qdd-framework@preview`. Permite aplicar el blueprint ciclo-de-vida-agentico con su registro en `.qdd/project/tutor/`.

### 🐛 `qdd blueprint apply` aceptaba solo `estado.yaml` dentro de `.qdd` (FND-108)

- **Qué pasaba:** `qdd blueprint apply ciclo-de-vida-agentico` rechazaba las rutas `.qdd/project/tutor/…`, aunque la receta guarda ahí su registro, y no generaba la ficha.
- **Arreglo:** `.qdd/project/tutor/` es un área de datos de un módulo: puede ser ruta de una ficha y no tiene que existir todavía, porque el esqueleto la crea. El resto de `.qdd` sigue fuera.

## [2.8.0-rc.14] - 2026-11-01 (preview)

Decimocuarta **preview** de 2.8.0: `npm i -g qdd-framework@preview`. Trae el atajo desde el chat y los pedidos ◷ armados.

### ◷ Confirmas una vez y el pedido corre solo cuando está listo

- **Tu intención queda tomada:** si un pedido confirmado responde «todavía no» (código 75: falta un tag, npm aún no publica…), queda **◷ armado**. qdd lo reintenta solo cada 30 s, sin la IA y sin que lo vuelvas a pulsar, hasta que sale bien.
  - Una falla de verdad no se reintenta.
- **Un pedido que espera a otro («espera a [a]») ya se puede confirmar:** corre solo en cuanto [a] termina.
- **✕ Dejar de esperar** lo devuelve a pendiente (`qdd consola desarmar <tecla>`). Pasadas 2 h, o si se cierra el proceso que esperaba, también vuelve a pendiente, y el panel dice por qué.
- **Se lee de un vistazo:** «◷» y el nombre van en amarillo y negrita, «Armado · corre solo cuando…» en amarillo, y lo que dijo el comando, atenuado debajo. Un «todavía no» sin armar ya no se pierde en gris.

### ⌨ Aprobar sin mouse desde el chat

- **Funciona en cualquier terminal y sistema, sin configurar nada:**
  1. Con el chat vacío, escribe la tecla de lo que te espera: un pedido (`a`, `b`, `d`…) o una ficha (`1`…`9`). Mientras la escribes, el chat te muestra qué abre Enter.
  2. **Enter:** no le envía nada a la IA. Abre la confirmación en el panel, con el foco en «s  Sí».
  3. **Enter** otra vez aprueba, y el teclado vuelve solo al chat.
- Solo se toma un mensaje que sea exactamente esa tecla. Si quieres enviarle a la IA solo una letra, por ejemplo «a», escribe «a.».
- **En un solo paso, si tu terminal o tu sistema lo permite:** mapea AltGr+letra (u otra tecla) al carácter privado U+E000 + la letra (AltGr+a → U+E061). El panel lo recibe sin escribirlo en el chat y abre la confirmación al instante.

## [2.8.0-rc.13] - 2026-11-01 (preview)

Decimotercera **preview** de 2.8.0: `npm i -g qdd-framework@preview`. Arregla el panel de rc.11 y rc.12.

### 🐛 El panel se instalaba sin `materializar.js` (FND-107)

- **Qué pasaba:** con rc.11 y rc.12, Claude Code no cargaba el panel de tareas («hooks module did not load: …/materializar.js: no such file») o lo dibujaba sin el efecto, porque el binario no llevaba ese archivo.
- **Arreglo:** el binario ahora lleva todos los archivos del mod, y una prueba impide que vuelva a faltar uno.
- **El doctor lo detecta:** `qdd doctor` compara el panel instalado en Claude Code con el del binario y dice qué falta. `qdd doctor --fix` y `qdd consola activar` lo reinstalan aunque la versión sea la misma.

## [2.8.0-rc.12] - 2026-11-01 (preview)

Duodécima **preview** de 2.8.0: `npm i -g qdd-framework@preview`. Trae los blueprints a un clic en el panel.

### 📦 Los blueprints a un clic en el panel

- **Tecla `p` (📦 Blueprints) en el panel de tareas:** lista los blueprints instalados y los disponibles en las fuentes del proyecto, con sus cuentas: instalados, cuántos se actualizan sin la IA, cuántos cambian la receta y cuántos están disponibles. Para cada uno muestra la versión instalada → la publicada y, si la receta cambió, por qué.
- **Sin gastar tokens:** todo lo hace qdd en Go (`qdd consola blueprints`) y sin red, con la copia central de cada fuente.
  - **r ↻ Actualizar catálogo** trae las versiones nuevas (`qdd blueprint sync`).
  - **⬆ Actualizar** toma una versión nueva y **u ⬆ Actualizar los N sin la IA** toma de una vez todas las que no cambian la receta.
  - **＋ Instalar** agrega uno disponible.
  - Las acciones que escriben el lockfile piden «s  Sí». Después de cada una, el panel dice cuántas versiones nuevas trajo o cuántos blueprints se actualizaron o instalaron.
- **Solo ✦ Aplicar con la IA gasta tokens:** construye o adapta el módulo. Le envía a la IA un mensaje corto que le pide usar `qdd_blueprint_apply` y hacerte las preguntas de diseño.
- Un blueprint instalado desde la URL https se reconoce aunque la organización esté registrada con la URL ssh: es el mismo repositorio.

## [2.8.0-rc.11] - 2026-11-01 (preview)

Undécima **preview** de 2.8.0: `npm i -g qdd-framework@preview`. Trae lo de abajo, el panel que relee con cualquier qdd (FND-106) y estado-por-tareas desde v10.0.0.

### ✨ Lo que cambia en el panel se materializa

- **Como las ventanas de hypr-materialize:** caracteres que parpadean en el acento, bloques que se afinan (█ ▓ ▒ ░) y el texto nítido, en 0,8 s. Lo que se va hace lo mismo al revés, en 0,4 s.
  - Solo se anima lo mínimo que cambió: una celda ⏳ → ✅, el nombre de una tarea, la cabecera de un objetivo o la tarjeta de un pedido. El panel entero se materializa solo la primera vez que se dibuja.
  - Nada cambia de ancho ni se mueve de lugar.
- **Funciona igual en cualquier sistema:** solo usa caracteres y colores del terminal. Si existe `~/.config/hypr/materialize.conf`, toma de ahí el acento y las duraciones (solo los lee). Si no, usa el acento del tema y 0,8 s / 0,4 s.
- **Cómo se calcula:** qdd calcula los cuadros en Go (`qdd consola materializar`, pkg/materializar, determinista), una vez por cambio. El panel solo los reproduce con su propio reloj.
- **Encendido por defecto:** `qdd consola efectos off` o `QDD_EFECTOS=off` lo apaga. `qdd doctor` dice cómo está y `--fix` repara el interruptor si quedó ilegible.

### 🐛 Corregido

- **El panel relee en el acto con cualquier qdd** (FND-106): un comando como `…/qdd-rc10 consola pedir`, `./qdd` o `~/.local/bin/qdd` ahora hace que el pedido aparezca en «▶ Para ti» al terminar, sin esperar la lectura de cada 2 s. Una carpeta o un archivo llamado qdd en los argumentos no cuenta.

## [2.8.0-rc.10] - 2026-11-01 (preview)

Décima **preview** de 2.8.0: `npm i -g qdd-framework@preview`.

### 🔁 Sincronía panel–chat

- Lo que el panel espera de la persona puede resolverse en el chat: «ya lo mergeé», «eso no va», «hazlo distinto». Puede ser un pedido, una ficha en borrador o un ⏳ suyo.
  - **Con cada mensaje:** `qdd consola ciclo --contexto` le dice a la IA qué espera el panel, en una línea y solo si cambió desde el último mensaje. Sin cambios no gasta tokens.
  - **La IA lo refleja en el acto:** con `qdd_consola_pedir` y `resuelto` (`hecho` si lo hizo la persona, lo que libera a los pedidos que esperaban) o con `qdd_estado set`. Una ficha solo se aprueba si la persona lo dice.
  - **El panel se relee en el acto** después de cada tool de qdd o comando `qdd`, sin esperar a la próxima lectura.
- `qdd consola descartar` acepta la tecla que ve la persona, y `--hecho` marca el pedido como hecho por ella.

### 🎯 Objetivos, dependencias y árbol en el tablero

- **Objetivos** (`qdd estado objetivo`): cada objetivo dice para qué se hacen sus tareas y cuándo se cumple (su criterio de éxito).
  - A lo más uno abierto es el **foco** y se muestra primero.
  - Se cumple solo con evidencia de su criterio; con todas sus tareas terminadas dice «tareas listas, falta verificar el criterio».
  - Se cierra cumplido o descartado, con su motivo, y nunca desaparece en silencio.
  - Una tarea puede no tener objetivo: se ve al final, bajo «sin objetivo».
- **Dependencias** (`qdd estado tarea --depende-de`): una tarea no termina antes que las tareas de las que depende. Su última etapa no pasa a ✅ y la tabla dice «(espera a X)». No se admiten ciclos.
- **El tablero es un árbol:** objetivo → tareas → dependientes, con las etapas en columnas a la derecha.
  - qdd entrega el árbol resuelto en el JSON del panel (prefijos, notas «espera a X», el estado de cada objetivo).
  - La tabla de la respuesta lleva una fila 🎯 por objetivo antes de sus tareas.
- **Pinchar el nombre** de una tarea o de un objetivo en el panel le pregunta a la IA «¿Cómo vamos con…?»: avance, bloqueos y siguiente paso. `l ↻` vuelve a leer el tablero.
- **La barra muestra la carpeta del proyecto** (solo su nombre, «📂 qdd») debajo del modelo, a la izquierda.
- El archivo pasa a `formato: 2` solo si usa objetivos o dependencias. Un qdd anterior lo rechaza en vez de perderlos al escribir; un tablero plano sigue en formato 1.
- El repositorio de qdd aplica `estado-por-tareas` 2.0.0, con objetivos, `Producción` como la etapa que requiere aprobación y vida de 120 s. Este qdd declara que implementa la 2.0.0. Un ❌ sin nombrar su falla se informa como EPF-C09, su escenario propio.

### 🛰️ Las fallas de QDD se publican solas como issue (incidentes)

- **`qdd incidentes`**: un pánico de la CLI o de una tool MCP y cada autorreparación de `qdd doctor --fix` sobre datos que QDD escribió quedan en un registro local y se publican solos, en segundo plano, como issue del repositorio de QDD.
  - Formato estándar `qdd-incidente/1`: qué pasó, dónde en el código de QDD, cómo reproducirlo y un bloque YAML para que una IA lo reproduzca y lo repare con su prueba de regresión.
  - Sin duplicar: la misma firma (tipo, origen, mensaje sin números y las primeras funciones de la traza) suma una ocurrencia y comenta el issue que ya existe, también si lo abrió otra máquina. Tope de 5 publicaciones por día.
  - Solo datos de QDD: versión, sistema, el comando con sus valores como `<marcador>` y la traza dentro del código de QDD. Rutas, nombres de archivo, valores entre comillas, correos, URLs y credenciales se reemplazan; el nombre de la carpeta del proyecto y el del usuario también.
  - `qdd incidentes ver <firma>` muestra el issue exactamente como sale; `QDD_INCIDENTES=off` los apaga; `qdd doctor` revisa el registro y aparta un archivo ilegible.

### 🧰 Las herramientas del proyecto en el panel (RFC 0011)

- **`qdd herramienta`**: un ejecutable del proyecto (un simulador, un visor de logs, un migrador) queda como pestaña del panel, con sus acciones como botones.
  - La IA arma una vez su definición con `candidatos`, `ayuda` y `guardar`. QDD la valida, corre su prueba de humo y la versiona en `.qdd/herramientas/<id>.yaml`.
  - Desde ahí se corre con un clic, sin IA y sin tokens: `correr` la ejecuta sin shell y cada campo llega como un solo argumento.
  - Los secretos nunca van en la definición: solo el nombre de su variable de entorno. Si falta, dice cuál y no corre.
- **En el panel de Claude Code**:
  - **⚙ Config**, a la derecha de la primera fila, lista las herramientas para abrirlas o quitarlas. «+ Agregar herramienta» ofrece los ejecutables que detecta o una ruta escrita a mano, y guía los 5 pasos.
  - Cada herramienta es su propia pestaña: sus acciones son botones, cada una con sus campos, y ▶ Correr muestra la salida al terminar (una-vez) o mientras corre, hasta ■ Detener (en-vivo).
- **El gate valida las definiciones** (`HER-01-DEFINICION`) y **`qdd doctor`** dice qué le falta a cada una en esta máquina, con el comando para arreglarlo.

### 🧪 El panel se prueba en el gate y se mide contra su receta

- **Las pruebas del panel corren en `qdd audit`**: las de OpenCode (`node --test`) y las del mod de Claude Code (`claude plugin test`).
  - Un `test_commands` puede ser `{comando, requiere: [programas]}`. Sin el programa en esta máquina (p. ej. la CLI de Claude en el CI), la suite queda **sin medir** (`QA-03-SIN-MEDIR`): no bloquea y nunca cuenta como aprobada.
- **`qdd blueprint conformance`**: cada escenario que un módulo debe cumplir tiene una prueba que lleva su id (con «-» o «_»), en los archivos de prueba de su ficha.
  - Un módulo que aplicó un blueprint se mide con sus respuestas.
  - Uno que lo implementa como origen (`implementa:` en su ficha) cubre todas las opciones que el blueprint ofrece.
  - El blueprint sale de lo importado o, si el proyecto no lo importa, de lo publicado.
  - Una ficha puede dejar un escenario `fuera`, con su motivo (lo hace otro, no este código): no se exige, pero siempre se ve.
  - En QDD el panel y el estado quedan conformes: estado-por-tareas 23/23 y consola-claude-code 25/25 (CCC-C05 fuera: la compactación la hace Claude Code).
- **`qdd audit` exige la conformidad** (`BP-08-CONFORMIDAD`): un módulo que aplica o implementa un blueprint no pasa el gate si a un escenario le falta su prueba.
  - Se mide sin red: el blueprint sale de lo importado o de la última versión de la copia local del repositorio de la organización.
  - Sin ninguno de los dos (p. ej. en el CI) queda **sin medir** (`BP-08-SIN-MEDIR`): no bloquea y nunca cuenta como cumplida.
  - Un proyecto que solo usa un blueprint que implementa qdd (`estado-por-tareas`, `consola-claude-code`) no necesita sus pruebas: lo cumple el qdd instalado si lo implementa en la misma versión mayor y no más vieja; si no, el gate dice si actualizar qdd o el blueprint (FND-102).
  - Una prueba de Go que lleva el id pegado al prefijo (`TestCVA_C01_…`) cuenta para su escenario (FND-103).
- **`qdd estado check --respuesta` revisa EPF-C08**: lo que el texto afirma con «<tarea> · <etapa>: <estado>» es lo que dice la tabla, y una afirmación sobre una tarea o etapa que no está en ella la contradice.
- **Arreglos:**
  - pulsos simultáneos ya no borran la activación de una tarea (FND-097): la actividad, la cola de pedidos y el registro de uso se escriben uno a la vez entre procesos (`pkg/candado`);
  - la certificación ciega ya no toma «P-020» como prueba de P-02 y acepta `test_P_01` como prueba de P-01 (FND-098);
  - guardar o aprobar una ficha con un qdd más viejo ya no borra los campos que no conoce (FND-099);
  - las pruebas llevan el id de su escenario actual: una con el id de otro hacía ver cubierto lo que no lo estaba (FND-100);
  - en adopción gradual, una recomendación como `QA-03-SIN-MEDIR` ya no vuelve a bloquear el gate (FND-101);
  - sin llave SSH (p. ej. en el CI), un repositorio de blueprints público se lee por HTTPS: `add`, `sync`, el gate y `doctor` reintentan por la URL HTTPS equivalente, como ya lo hacían `propose` y `release` (FND-104);
  - `propose` encuentra el aporte aunque la organización esté escrita por SSH en `config.yaml` y por HTTPS en el lockfile: es el mismo aporte (FND-105).

## [2.8.0-rc.9] - 2026-10-31 (preview)

Novena **preview** de 2.8.0: `npm i -g qdd-framework@preview`.

### 🧪 QDD genera blueprints reproducibles y los certifica a ciegas

- **La receta separa lo técnico de lo de negocio.** Un blueprint declara:
  - `alcance` (qué incluye y qué queda fuera);
  - **parámetros** de negocio `{{id}}` (nombres, textos, idioma, umbrales, cantidades), que la persona decide al aplicar, igual que las preguntas, con un valor sugerido;
  - **constantes** técnicas `{{id}}`, fijas y con su porqué;
  - **pasos** en orden (acción → resultado, con el escenario que verifica cada uno), que quedan en la ficha como «Paso N · …»;
  - un parámetro con `si` (como una opción) solo se pregunta con esas respuestas: sin la tarea en curso, nadie pregunta cuánto vive;
  - una opción puede traer su propio `alcance`: la receta de cada variante lleva solo los límites de sus respuestas;
  - cada respuesta queda con lo que significa («etapas → software: Código · Pruebas · …»), no solo su valor (FND-095);
  - `requiere_qdd`: un qdd anterior no aplica a medias una receta que no entiende.
- **`qdd blueprint distill <modulo> --id <blueprint>`** (o `qdd_blueprint_contribute {modulo}`): QDD destila el borrador desde la ficha aprobada (nunca copia las rutas ni el caso real del proyecto) y entrevista a la persona hasta delimitarlo. Pregunta qué incluye y qué no, si cada valor escrito en una regla es de negocio o técnico, y qué hace cada detalle de la implementación. Los ejemplos entre «» o tras «p. ej.» no cuentan.
- **`qdd blueprint certify`** (o `qdd_blueprint_contribute {certificar}`):
  - Prepara, fuera del proyecto, una carpeta por variante (valores sugeridos y otros valores de negocio) con solo `RECETA.md`.
  - Un agente nuevo, sin el original, la construye en el lenguaje que elija.
  - `--verificar` corre sus pruebas y exige una por escenario. Así lo técnico se replica al 100 % y lo de negocio se adapta.
  - Quien construye deja en `construccion.yaml` sus `supuestos`: lo que la receta no decía. La certificación los guarda, el comando los convierte en preguntas para la persona (¿se fija en la receta o queda libre?) y el PR los lista.
  - Queda `certificacion.yaml`, atada a la huella de la receta, y `propose` la exige para una receta con pasos.
- **Calidad nueva**:
  - **defectos:** sin límites (CAL-10), sin pasos (CAL-11), un paso sin escenario (CAL-12), un escenario al que ningún paso lleva (CAL-13), un valor fijo sin clasificar (CAL-14), un detalle de implementación (CAL-15), una receta nueva sin `requiere_qdd` (CAL-17), un `{{id}}` sin valor con alguna combinación de respuestas (CAL-18);
  - **mejora:** un parámetro o una constante sin uso (CAL-16).
- **Arreglos de la medición** al migrar estado-por-tareas:
  - un valor repetido se pregunta una vez;
  - el vocabulario cuenta como regla, y un valor ya declarado escrito a mano pide «escribe {{id}}»;
  - cada límite del alcance se mide solo (FND-094).
- `qdd_blueprint_list` cuenta ▶pasos y ⚙parámetros y da las secciones `alcance`, `pasos` y `parametros`. La leyenda de los símbolos pasó de la descripción de la tool a su salida, y tools/list sigue bajo 16.000 caracteres.

## [2.8.0-rc.8] - 2026-10-31 (preview)

Octava **preview** de 2.8.0: `npm i -g qdd-framework@preview`.

### 📐 QDD mide la calidad de cada blueprint como receta

- **`qdd blueprint quality`** (y `qdd_blueprint_list {seccion: calidad}`): un blueprint es como el plano de un arma en el juego Rust, con él se construye la pieza entera en cualquier lenguaje. QDD lo mide de forma determinista en una línea: `consola-claude-code@7.1.0 · calidad 12/100 · ⚠7 ✂6 · receta ≈3.4k (−293)–4.0k tokens · índice ≈355`.
- **⚠ Defectos** (lo que falta o está mal puesto para construirla tal cual): un escenario de la base que solo vale con una opción, una opción que usa una versión mayor que la que pide, una opción que no dice qué implica, código en la receta, invariantes de una opción sin un escenario que las verifique, una opción que depende de otra respuesta en una pregunta que se hace sin ella (admitiría una combinación imposible).
- **✂ Mejoras sin perder calidad**: una regla de más de 280 caracteres, una opción que repite la base, un término que nada usa. Ningún arreglo quita una regla, opción ni escenario.
- **Tokens:** cuánto cuesta el índice y la receta que recibe un proyecto (con las respuestas por defecto y con las que más traen), y cuánto ahorra cada arreglo.
- **La receta nunca empeora:** `qdd blueprint propose` rechaza un aporte que agrega defectos (los que ya tenía no bloquean) y un blueprint nuevo nace sin defectos. El PR dice la calidad antes → después.
- Sin blueprints importados, mide lo publicado en la fuente de la organización; también mide la carpeta de un aporte en curso.

### 🐛 Corregido

- El índice contaba dos veces las invariantes que comparten dos opciones (FND-093): estado-por-tareas decía 🔒25 y tiene 18.

## [2.8.0-rc.7] - 2026-10-31 (sin publicar)

No se etiquetó por separado: se publica dentro de la rc.8.

### ⚡ Blueprints con índice: leerlos cuesta casi nada

- **Una línea por blueprint:** `qdd_blueprint_list` (y `qdd blueprint list`) muestra `consola-claude-code@7.1.0 · Consola de Claude Code · ❓6 🔒14 ✓26 · en estado`. En un proyecto con dos blueprints, la lista pasó de ~2.700 a ~200 tokens.
- **Detalle a pedido:** con `blueprint`, el índice en JSON (preguntas con sus opciones, escenarios y la línea de cada sección en blueprint.yaml); con `seccion`, solo `preguntas`, `invariantes`, `decisiones`, `escenarios`, `vocabulario` o `todo`. En la terminal: `qdd blueprint list --json`, `qdd blueprint show <id> --json` y `--seccion`.
- **Aportar sin leer el archivo entero:** `qdd blueprint contribute` dice en qué línea está cada sección.
- El índice se calcula del contenido verificado por su sha256: no hay un archivo extra que se pueda desactualizar.

### 🧭 La IA sabe por qué cero `else`

- No es una moda: cada vez menos personas leen el código línea a línea (lo escriben y revisan IAs), así que el que sí se lee tiene que entenderse de un vistazo. Con guardas, cada caso especial se resuelve arriba y sale, y el camino principal queda recto.
- La razón aparece donde la IA lee la regla: en la forma de trabajo de `qdd_project_context`, en el mensaje del gate cuando encuentra un `else`, en las instrucciones que QDD deja en AGENTS.md / CLAUDE.md y en el núcleo que se instala en cada proyecto (principios, CERT-040, CERT-005).

### ✂️ Tools MCP más livianas en cada sesión

- Las descripciones de las 35 tools quedan en estilo chat, sin quitar ninguna regla (quién responde las preguntas, quién confirma, qué no ejecuta la IA). `tools/list` pasó de 23.815 a ~15.800 caracteres (de ~6.400 a ~4.000 tokens por sesión), con una prueba que no lo deja crecer.
- Las anotaciones dicen la verdad: `readOnlyHint` en las tools que solo leen (contexto, blueprints, findings, puntaje…), y ya no se marca como «destructiva» cada tool.

## [2.8.0-rc.6] - 2026-10-31 (preview)

Sexta **preview** de 2.8.0: `npm i -g qdd-framework@preview`.

### ⏳ Pulsa una etapa y la IA la atiende

- En la tabla del panel, cada etapa ⏳ o ❌ es un botón. Al pulsarla, la IA recibe un mensaje armado por qdd: qué etapa, qué espera (o qué falla) y de quién, y que intente completarla.
- Si falta algo que tienes que hacer tú, la IA lo deja en «▶ Para ti» ligado a esa tarea; si falta una decisión, te pregunta; si todavía no se puede (p. ej. faltan horas de uso), te dice qué falta y cuándo.
- qdd vuelve a calcular la etapa al pulsarla: si ya no espera, no se envía nada.

### ✂️ Textos cortos y sin botones «undefined» (FND-092)

- **Sin botones «undefined»:** «atender» es un tipo opcional; qdd lo entrega solo al panel que lo pide (`qdd consola panel --con atender`). En «Acciones» el panel muestra solo lo que sabe dibujar.
- **Mensajes estilo chat:** lo que recibe la IA al pulsar una etapa es una línea de 280 caracteres como mucho («⏳ Publicación · Barra… (tarea barra): espera … → ciérrala · ¿falta algo mío? …»), y el motivo de un pedido, 60 como mucho; uno más largo se rechaza pidiendo un resumen.

## [2.8.0-rc.5] - 2026-10-31 (preview)

Quinta **preview** de 2.8.0: `npm i -g qdd-framework@preview`.

### 🎨 Barra de abajo ordenada (blueprint consola-claude-code v7)

- **A la izquierda, solo el modelo y el modo de la IA** («Opus 5.5 · auto»). El modo es el que qdd anota con cada mensaje; Claude Code no se lo entrega a la barra.
- **A la derecha, en lista:** «contexto 38%», «5h 12% ↻ 02:00» y «semana 40% ↻ mar 12:00», uno por línea y pegados al borde. El ancho sale de la terminal; si no se sabe, van sin relleno, y si el contexto no cabe junto al modelo, baja a la lista.
- **Fuera de la barra:** la carpeta del proyecto y «◧ N tareas · ⏳ ❌»; los dos están en el panel. La línea de recursos (opcional) va al final, a la derecha.
- **La lista queda de verdad a la derecha (FND-091):** Claude Code borra los espacios al inicio de cada línea y corta la que llega al borde; el relleno empieza con un blanco que no borra y el margen es de 6 columnas.

### ▶ Un pedido sale del panel si sigues trabajando en su tarea

- Cada pedido extiende una tarea del estado (`--tarea`, o la tarea en curso). Si después de tu mensaje esa tarea sigue cambiando (preferiste seguir trabajando en vez de publicar), el pedido sale de «▶ Para ti» al instante.
- Cuando la tarea vuelve a estar lista, la IA lo pide de nuevo y reaparece con la misma tecla. `qdd consola pedidos` lo muestra como «retirado» y dice qué hacer.

## [2.8.0-rc.4] - 2026-10-31 (preview)

Cuarta **preview** de 2.8.0: `npm i -g qdd-framework@preview`.

### 🎨 Panel de tareas ordenado (FND-090)

- **Solo el nombre de la carpeta.** Arriba dice «📂 qdd  o: Abrir carpeta», sin la ruta completa (blueprint consola-claude-code) ni la tecla repetida. La carpeta de un pedido se nombra desde la del proyecto (`qdd/npm`).
- **Lo tuyo primero.** Debajo de la tabla, separadas por una línea fina: «▶ Para ti», ❌ Fallas y, al final, las Acciones en una sola fila de botones.
- **Sin ceros en rojo.** El resumen omite los conteos en cero; sin esperas ni fallas dice «✓ nada espera ni falla».
- **Cada respuesta junto a su botón.** La confirmación de la ✕ de una tarea aparece bajo la tabla, y el resultado de un pedido en «Para ti».
- **Tarjetas más cortas.** Un pedido hecho ocupa una línea; uno en espera o en marcha dice cómo va en una línea atenuada. La versión de qdd pasa al pie.

## [2.8.0-rc.3] - 2026-10-31 (preview)

Tercera **preview** de 2.8.0: `npm i -g qdd-framework@preview`.

### 🐛 Los pedidos del panel ya no se cortan ni lo bloquean (FND-089)

- **Un pedido corre aparte del panel.** El panel lo lanza y queda libre al instante: recargarlo (`/reload-plugins`) o cerrar la terminal ya no lo corta, y los demás botones siguen respondiendo mientras corre.
- **Si igual muere** (el PC se apaga, alguien lo mata), queda en rojo al instante con «↻ Reintentar», en vez de quedarse «ejecutando…» 15 minutos.
- **El comando se ve en una línea** («… (+N líneas)»). Entero, solo al confirmarlo.
- **«Para ti» cuenta lo que se está ejecutando:** ya no dice «nada por hacer» arriba de un pedido en marcha.
- **La IA deja comandos más seguros:** la ayuda de `qdd consola pedir` y de `qdd_consola_pedir` pide comandos que se puedan repetir y que no esperen adentro (si falta algo, salen con `exit 75`).

## [2.8.0-rc.2] - 2026-10-31 (preview)

Segunda **preview** de 2.8.0: `npm i -g qdd-framework@preview`. Las horas de uso de rc.1 y rc.2 se suman para VER-05.

### 🖼️ Imágenes a la consola

- **`qdd foto`** trae la imagen del portapapeles con la herramienta del sistema: wl-paste o xclip en Linux, PowerShell en Windows y osascript en macOS. Si el portapapeles no tiene imagen, usa la última captura de pantalla. Deja la imagen en `tmp/fotos` (git no la ve) e imprime la ruta para decirle a la IA «mira <ruta>».
- **`qdd_foto`** (MCP) entrega la imagen como imagen: la IA la ve directamente. Sirve cuando pegas una captura y llega como bytes de PNG.
- **`qdd doctor`** dice con qué se lee el portapapeles y qué instalar si falta.

### 🐛 Tablero

- **El panel avisa si el tablero quedó viejo** (FND-088). El título dice «Tareas · hace 9 h» en vez de la hora sola, y si hay tareas pendientes sin tocar en 2 h avisa en amarillo «⚠ Sin actualizar…». Los ⏳ y ❌ del resumen se cuentan como etapas: «⏳ 10 etapas esperan».
- **Pruebas y PR ya no se leen igual** (FND-087): las columnas eran «Pr» y «PR» y ahora son «Pe» y «PR».

## [2.8.0-rc.1] - 2026-10-30 (preview)

Primera **preview**: `npm i -g qdd-framework@preview`. Pasa a `latest` como 2.8.0 con 5 h de uso real medido (VER-05).

### 📏 Uso real medido y canal preview (RFC 0010)

- **`qdd uso`** mide cuánto se usa de verdad cada versión de qdd en tus proyectos QDD: comandos, herramientas MCP, pulsos del agente y mensajes con el panel. Dos eventos a menos de 5 min son uso continuo; uno aislado no suma.
- **`qdd uso evidencia <versión>`** deja esas horas en `.qdd/project/evidence/uso/` para commitearlas con la release.
- **`VER-05-UNPROVEN-RELEASE`** (`uso_minimo_horas` en `.qdd/policies.yaml`; qdd usa 5): una minor o una major pasa a `latest` solo con sus horas medidas. Una preview y un fix no esperan.
- **Canal preview:** un tag `vX.Y.Z-rc.N` se publica en npm con `--tag preview` y como pre-release en GitHub.

### 🧭 El panel se ordena solo y se limpia solo

- **Cada mensaje es un ciclo:** lo que ya se vio (un pedido hecho, el resultado de una acción) sale del tablero en el mensaje siguiente.
- **Autorización según el modo de la IA:** el panel anota el modo de permisos de la IA principal. `qdd_project_context` le dice cuándo pedir permiso: en auto, solo antes de cambiar algo ya definido (una ficha aprobada, un blueprint, una versión o un tag, o algo que no se puede deshacer).
- **Pedidos en orden** (FND-080): `--despues`/`--antes` en `qdd consola pedir` (y `despues`/`antes` en `qdd_consola_pedir`). El que espera va justo después y no tiene botón hasta que lo otro esté hecho.
- **«Todavía no» no es una falla** (FND-082): un pedido que sale con 75 vuelve a pendiente, en gris, con su aviso.
- **Una cola por repositorio** (FND-081): un worktree usa la del checkout principal, y `qdd doctor --fix` junta lo que quedó fuera de lugar.
- **Sin sección «Esperando»:** el ⏳ ya está en la tabla y en el resumen.

### 🐛 Estado por tarea

- **«·» es sin empezar** (FND-085). Una tarea nueva con solo «Código ✅» ya no queda terminada ni sale del panel, y «—» es solo «no aplica».
- **La barra y el panel cuentan lo mismo** (FND-086): las tareas pendientes y cada ⏳ y ❌ que tienen.
- **Los textos dicen «tarea»** en la ayuda, la tabla (`| Tarea |`), el panel y los errores.
- **Blueprints** (FND-083, FND-084): `propose` lleva el catálogo y el README, y la ficha del estado puede apuntar a `.qdd/project/estado.yaml`.

### 🔁 Un blueprint puede reemplazar a otro (RFC 0009)

- **`reemplaza: [id]` en `blueprint.yaml`.** Sirve para cambiar el id de un estándar publicado o juntar dos en uno, como `estado-por-tareas`, que reemplaza a `estado-por-frente`.
- **El gate lo avisa.** Si la fuente publicó el que reemplaza, `qdd audit` reporta `BP-07-REPLACED` con los comandos exactos para migrar. En ese caso ya no reporta `BP-06` para el viejo, y `qdd doctor` lo informa.
- **`qdd blueprint add <nuevo>` migra.** Saca el viejo del lockfile y pasa tus módulos al nuevo, con su mismo id y sus respuestas. Después `qdd blueprint apply <nuevo> --modulo <módulo>` te pregunta solo lo nuevo. Un reemplazo de otra fuente no toca tu blueprint.
- **`qdd estado` encuentra su ficha por el blueprint** que la generó (`estado-por-tareas` o, como legado, `estado-por-frente`), no por un id fijo.

### 🔢 El gate revisa el versionamiento semántico (CERT-024, FND-078)

- **`qdd audit` verifica que el número de versión diga la verdad.** Si desde el último tag hay un commit `feat`, la versión nueva tiene que subir el MINOR; con `!` o `BREAKING CHANGE:`, el MAJOR (en `0.y.z`, el MINOR). Lo reporta como `VER-03-UNDERSIZED-BUMP` y dice qué versión corresponde.
- También: una versión que no es semver (`VER-01`), una que retrocede respecto de un tag publicado (`VER-02`) y una sin entrada en un CHANGELOG que documenta versiones (`VER-04`). Lee `package.json`, `pyproject.toml`, `Cargo.toml` o `VERSION`.
- Activa por defecto; se apaga con `versioning: false` en `.qdd/policies.yaml`. Los pipelines de QDD usan `fetch-depth: 0` para tener los tags.
- Caso real: QDD v1.9.23 salió como patch con dos `feat`.

### 📚 Documentación al día (FND-072)

- **Guía nueva [`docs/blueprints.md`](docs/blueprints.md):** cómo crear el repositorio de blueprints de la organización (estructura, `catalogo.yaml`, campos, validaciones, versionado), importarlo, aplicarlo, usarlo con una IA y qué vigila el gate.
- **Command Reference completa:**
  - comandos que faltaban: `qdd modules`, `modules approve`, `modules context`, `blueprint show`, `certify human`, `undo` y `mcp-server`;
  - todas las tools MCP, agrupadas, incluidas `qdd_project_context`, `qdd_module_save` y `qdd_certify_human`;
  - secciones nuevas: fichas de módulo, reglas del gate (`QA-*`, `CI-*`, `HEART-*`, `BP-*`) y adaptadores de IA.
- **Getting Started** con el recorrido actual: init, audit, blueprints, fichas, programar con IA y certificar.
- **Architecture:** fichas de módulo y blueprints. Ya no cita `qdd sync` ni `qdd score`, que no son comandos de la CLI (son las tools `qdd_sync` y `qdd_score`).
- **README:** blueprints, fichas, gate veraz y OpenCode; índice de la documentación; badge de build apuntando a `quality.yml`.
- **README de npm** al día; deja de listar `/qdd validate` y `/qdd docs`, que no están implementados.
- **Al día con v2.x:** la guía de blueprints describe la carpeta central (el proyecto solo versiona el lockfile), varias fuentes, repositorios públicos por HTTPS, el ciclo `contribute` → `propose` → `release` y las reglas `BP-05` y `BP-06`. La referencia documenta `qdd estado`, `qdd consola` y las tools `qdd_estado` y `qdd_consola_pedir`, y el README presenta el panel de Claude Code.
- **La documentación ya no puede quedar atrás:** `cli/cmd/docs_reference_test.go` falla si la referencia omite un comando o una tool registrada, si nombra una tool que el servidor no registra o si un enlace relativo está roto.

### 🐛 Blueprints: mensajes más claros (FND-073, FND-074)

- **`qdd blueprint apply` sin terminal** marca cada pregunta condicional con `(solo si …)`, igual que la tool MCP. Antes parecía obligatoria aunque no aplicara.
- **`qdd blueprint add` sin nombres** ya no dice "el repositorio tiene varios blueprints" cuando hay uno solo, ni sugiere volver a indicar el repositorio registrado.

### 🧹 Lo temporal va en `./tmp` (FND-075 a FND-077)

- **`qdd init` crea `./tmp`** para lo temporal y casuístico: scripts de un solo uso, pruebas manuales, descargas, logs y salidas.
  - Trae su propio `.gitignore`, que la ignora entera, así que no se toca el `.gitignore` del proyecto.
  - Si el proyecto ya versiona archivos en `tmp/`, QDD no se la apropia y avisa.
- **Las IA lo saben:** las reglas sincronizadas (`CLAUDE.md`, `AGENTS.md`, `.cursorrules`…) y el método de trabajo de `qdd_project_context` indican usar `./tmp` y no dejar nada temporal en la raíz ni en las carpetas de código.
- **QDD ya no copia caché de Python a los proyectos:**
  - Se quitaron del repositorio los `__pycache__/*.pyc` que estaban versionados.
  - El empaquetado los descarta.
  - `qdd init` borra los que dejaron versiones anteriores dentro de `.qdd/core`.
- **`qdd audit` ya no ensucia el proyecto:**
  - Las pruebas corren con la caché de Python en `./tmp/pycache`.
  - pytest no crea `.pytest_cache/`.
  - Antes, cada auditoría de un proyecto Python dejaba `__pycache__/` en las carpetas de código.
- **El recorrido de primera vez lo verifica** (paso 21): `./tmp` existe, git no la ve y no aparece ningún `.pyc`, ni siquiera después de correr las pruebas.

## [v2.7.0] - 2026-10-30

### ● Tarea en curso y pulsos de vida (RFC 0006 §3.13, blueprint nuevo `estado-por-tareas` v1.0.0)

- **«● running» en la fila de la tarea que se trabaja.** El agente la declara con `qdd estado activa <tarea>` o con `qdd_estado accion=activar`. El `●` alterna entre tenue y brillante cada 1 s, y «running» va atenuado. Una tarea en curso siempre se ve, aunque no esté entre las 4 más nuevas.
- **Pulsos de vida.** Cada herramienta del agente y cada subagente, en Claude Code y en OpenCode, hacen destellar el `●` en el color de acento. Los comandos externos (scripts, tests, deploys, workers) laten con `qdd estado pulso <tarea> --fuente comando`. El panel envía como mucho un pulso cada 5 s y nunca demora la herramienta.
- **Se apaga sola.** Una tarea que pasa 120 s sin pulsos deja de estar en curso. También se apaga con `qdd estado activa <tarea> --fin`, al cerrarla o al descartarla.
- **No es estado.** La actividad vive fuera del repositorio, en `<QDD_HOME>/claude/actividad/`, junto a la cola de pedidos y con la misma clave por proyecto. No se commitea. `estado.yaml` no cambia. `qdd consola panel --json` agrega `actividad` en cada tarea en curso y `actual` a nivel del estado.

### 🪶 Panel liviano y solo lo pendiente (RFC 0006 §3.10, `consola-claude-code` v6.0.0)

- **Tareas.** En el panel, los frentes se llaman tareas. Se ven solo las 4 pendientes más nuevas, y las más antiguas se resumen en «+5 más antiguas».
- **✕ para descartar.** Cada tarea tiene una ✕ al final de su fila. Pide «s  Sí» y la pasa a «descartado» con su motivo: sale del panel y queda en el archivo.
- **Panel liviano.** La cabecera pasa a ser una línea con la hora y la versión de qdd, y el resumen va en otra línea con colores: `4 de 9 tareas · ⏳ 3 esperan · ❌ 1 fallan`. Si hay comandos para ti que se pueden ejecutar, el resumen suma `▶ N para ti`. Ya no hay recuadro ni números grandes. Las listas de ⏳ y ❌ ocupan dos líneas por elemento.
- **Un frente terminado sale del panel.** Si todas sus etapas están en ✅ o —, o si está cerrado o descartado, deja de verse en el acto. Sigue en `estado.yaml` y en `qdd estado ver --todos`.
- **Aprobar sin commits.** En un repositorio sin commits, el aviso ya no queda como «(revisada en )».

### 📂 Abrir la carpeta del proyecto desde el panel

- **Arriba a la derecha** el panel muestra la carpeta del proyecto (`📂 ~/…/proyecto`) y el botón `o  Abrir carpeta`, que la abre con el gestor de archivos que el sistema tenga configurado: `xdg-open` en Linux y BSD, `open` en macOS y `explorer` en Windows.
- **Sin confirmación**, porque no escribe nada. Como las demás acciones, la decide qdd: `qdd consola panel --json` trae `carpeta` y la acción `abrir-carpeta` (solo si existe el abridor), y `qdd consola accion abrir-carpeta` la lanza en una sesión propia. No es un enlace: los paneles de Claude Code solo aceptan enlaces `https://`. Si el abridor falla en los primeros 3 s, el panel lo dice.

### 🧭 El panel en OpenCode, solo para mirar (RFC 0006 §3.11)

- **Misma sincronización que en Claude Code.** Si OpenCode está en la máquina, `qdd consola activar` deja el panel en su barra lateral derecha. El panel ejecuta `qdd consola panel --json` cada 2 s, con un tope de 3 s, y se redibuja solo si algo cambió. Muestra las tareas, el resumen, ⏳/❌ y cuántas fichas esperan aprobación. No tiene botones, ✕ ni diálogos.
- **Siempre en la versión de qdd.** El plugin viaja dentro del binario. `actualizar` y `doctor --fix` lo reescriben, `doctor` avisa si es de otra versión o si falta su entrada en `tui.json`, y `desactivar` lo quita.
- **`tui.json` con cuidado.** Antes del primer cambio queda `tui.json.qdd.bak`. Solo se agrega una entrada, y un archivo que no es JSON válido no se toca: se avisa.

### 🏷️ Son tareas, no frentes (RFC 0006 §3.12)

- El panel pasa a llamarse `qdd-tareas@qdd` y se abre con `/tareas`. Al activar o actualizar se desinstala el `qdd-estado@qdd` anterior.
- La barra dice `◧ 9 tareas · ⏳7 ❌2`. No cambian el formato de `estado.yaml` ni los comandos `qdd estado …`.

## [v2.6.2] - 2026-10-30

### ✓ Un comando aceptado no se vuelve a ofrecer

- **Cada pedido muestra su estado.** En «▶ Para ti», un comando pendiente tiene su botón; apenas lo confirmas con «s  Sí» pasa a **⟳ ejecutando** y, al terminar, a **✓ hecho**, los dos en gris y sin botón: no se puede aceptar de nuevo. Si falla queda en rojo, con las últimas líneas del error y **↻ Reintentar** en su misma tecla (FND-070).
- **qdd también lo impide.** `qdd consola accion pedido:<id>` rechaza un pedido que se está ejecutando o que ya se ejecutó, y la cola se cambia con un bloqueo: dos clics seguidos corren el comando una sola vez.
- **Las teclas no se mueven.** Cada pedido guarda su tecla (a, b, d…) desde que se pide; antes, al salir uno de la cola, los demás cambiaban de letra (FND-071).
- **Panel más claro.** Cada pedido ocupa tres líneas iguales: qué, el comando exacto en monoespaciado y dónde/cuándo. El título de «▶ Para ti» resume cuántos hay por hacer y cuántos hechos; los hechos van al final y se van solos a la media hora. La confirmación dice qué vas a confirmar antes de la pregunta.
- Un pedido que lleva más de 15 minutos ejecutándose (por ejemplo, si se cerró Claude Code) se da por cortado y queda para reintentarlo. La cola de qdd 2.6.0 se sigue leyendo.

## [v2.6.1] - 2026-10-30

### 🧭 Nuevo orden del panel

- Las secciones van de arriba hacia abajo así: estado por frente (cabecera y tabla), **Acciones**, **▶ Para ti**, ⏳ Esperando y ❌ Fallas. Los botones quedan juntos bajo la tabla, y la confirmación aparece en la sección del botón que pulsaste.
- **Acciones** son los botones fijos de QDD según el estado del proyecto (aprobar fichas, revisar el estado, auditar). **Para ti** son los comandos que la IA te dejó para ejecutar con un clic.

## [v2.6.0] - 2026-10-30

### ▶ Comandos para la persona, con un clic desde el panel (RFC 0006 §3.9, `consola-claude-code` v5.0.0)

- **La IA deja el comando; tú lo ejecutas con un clic.** Cuando la IA no puede o no debe ejecutar un comando (no tiene permiso, se lo bloquearon, o publica algo, como el push del tag de una versión), ya no te pide que lo escribas con `!`: lo deja en tu panel con la tool MCP `qdd_consola_pedir` o con `qdd consola pedir -m <motivo> <comando…>`. QDD no lo ejecuta.
- **«▶ Para ti» en el panel.** Cada comando aparece primero, con su motivo, el comando exacto y la carpeta, y una tecla (a, b, d…). Con «s  Sí» corre en su carpeta y el panel muestra la salida; si termina bien sale de la cola y si falla queda para reintentarlo. «x  Descartar» lo quita sin ejecutarlo. El panel se abre solo cuando llega un pedido nuevo.
- **Cualquier comando, siempre confirmado.** La cola es de tu máquina (`<QDD_HOME>/claude/pedidos`, no se versiona), con tope de 9 y sin duplicados. `qdd consola pedidos` la lista y `qdd consola descartar <id>` quita uno.
- Las instrucciones para la IA (AGENTS.md, CLAUDE.md…) le dicen que use `qdd_consola_pedir` y que nunca ejecute ella misma un pedido.

## [v2.5.1] - 2026-10-30

### 🌐 Repositorios de blueprints públicos, sin llave SSH

- **Sugiere HTTPS.** Si no hay acceso a una URL SSH (`git@github.com:…`), el error de `add`, `sync`, `contribute`, `release` y `doctor` dice cuál es su URL HTTPS: un repositorio público se importa por ahí sin cuenta ni llave (FND-068).
- **Publica por HTTPS si SSH no puede.** `propose` y `release` reintentan el push por la URL HTTPS del mismo repositorio cuando SSH no tiene permiso (por ejemplo, una deploy key de solo lectura), con las credenciales de git como las de `gh`. La salida lo dice y el PR se abre igual (FND-063).
- **Repositorios bare en disco.** `qdd blueprint add /ruta/blueprints.git` acepta un repositorio bare, por ejemplo uno compartido en un disco de red (FND-067).
- La ayuda de `release --origen` dice que es la fuente que se etiqueta (FND-069), y `LICENSE` trae el texto MIT completo.

## [v2.5.0] - 2026-10-30

### 🧩 Varias fuentes de blueprints en un proyecto (RFC 0007)

- **Tus estándares junto a los de tu organización.** Importa desde otro repositorio con su URL (`qdd blueprint add git@github.com:yo/mis-blueprints.git mi-estandar`). El lockfile recuerda la fuente de cada blueprint, y el repositorio de la organización no cambia.
- **Cada blueprint se actualiza desde su fuente.** `qdd blueprint add <id>` sin origen (lo que pide `BP-06-BEHIND`) usa la fuente del lockfile; los ids nuevos, `--paquete` y `--todos` van a la organización. Si una fuente falla, el lockfile queda como estaba (FND-065).
- **Un id no se pisa.** Un blueprint con un id ya importado desde otra fuente se rechaza; para cambiarlo a propósito, `--reemplazar` (FND-064).
- **`sync` revisa todas las fuentes.** Da un resultado por fuente, y una inaccesible no oculta las demás (FND-066).
- **Crear y publicar en tu repositorio.** `contribute`, `propose` y `release` aceptan `--origen` para un blueprint nuevo. Uno importado se aporta siempre a su fuente.

## [v2.4.0] - 2026-10-29

### 📊 Línea de recursos en la barra (RFC 0006 §3.8, `consola-claude-code` v4.0.0)

- **`qdd consola activar --recursos`** suma a la barra una segunda línea con el uso del equipo, en etiquetas de una letra: `D 70% · G0 7% · G1 — · R 38% · C 12%`. D es el disco de la carpeta del proyecto, G0…Gn el % de uso de cada GPU, R la RAM usada y C la CPU usada. `--sin-recursos` la quita, y `actualizar` y `doctor --fix` conservan la elección.
- **Liviana.** Solo lee contadores locales. `nvidia-smi` se consulta como mucho cada 5 s (unos 20 ms) y con caché la barra tarda alrededor de 6 ms. Toda la medición tiene un tope de 400 ms.
- **Nunca inventa.** Una GPU sin contador legible, como la integrada de Intel sin root, va con «—». Lo que el sistema no expone se omite: la primera lectura de la CPU no tiene con qué comparar, y en macOS solo se mide el disco.

## [v2.3.0] - 2026-10-29

### 🖱️ El panel de estado con acciones y jerarquía visual (RFC 0006 §3.7, `consola-claude-code` v3.0.0)

- **Acciones desde el panel.** Cada ficha en borrador tiene su botón (teclas `1…9`) para aprobarla. `c` revisa el estado (`qdd estado check`, solo lectura) y `u` audita (`qdd audit`). Lo que escribe abre un recuadro «¿…?» con `s  Sí` y `n  No` dentro del panel, y el resultado queda a la vista en verde o rojo.
- **La lógica sigue en Go.** `qdd consola panel` entrega en un JSON el estado por frente y las acciones de ahora. `qdd consola accion <id> [--confirmado]` las vuelve a calcular antes de ejecutar: rechaza un id que no ofrece y una acción que escribe sin `--confirmado`. El panel solo pasa ids, nunca comandos.
- **Cuatro niveles visuales.** La cabecera, con borde, muestra los frentes, los ⏳ y los ❌ en números grandes de 3 filas. Debajo van los títulos en negrita, el texto y el detalle atenuado. Cada frente toma el color de su peor etapa: ❌ rojo, ⏳ amarillo, ✅ verde.
- **Se abre también sin frentes** si hay fichas por aprobar. Sin `estado.yaml`, dice cómo crearlo y sigue ofreciendo las fichas.

## [v2.2.0] - 2026-10-29

### ◧ `qdd estado`: el estado por frente en la consola (RFC 0006)

El blueprint `estado-por-frente` con destino «archivo» ya tiene herramienta. La tabla de frentes × etapas vive en `.qdd/project/estado.yaml`; una sesión cortada o un agente nuevo retoma desde los ⏳ sin rehacer lo hecho. Es un binario Go, igual en Linux, macOS y Windows.

| Comando | Qué hace |
|---|---|
| `qdd estado` / `qdd estado ver` | Imprime la tabla en markdown o mensajería, según la ficha. |
| `qdd estado init` | Crea el archivo con las etapas de la ficha (o `--etapas`). |
| `qdd estado set <frente> <etapa> <ok\|espera\|falla\|na>` | Actualiza una celda. Valida antes de escribir: un ✅ sin evidencia no se guarda. |
| `qdd estado cerrar <frente> --motivo …` | Cierra o descarta un frente. Nunca se borra. |
| `qdd estado check [--respuesta r.md] [--json]` | Lector automático EPF-C01…C07 y EPF-A01. Compara contra git HEAD para detectar frentes que desaparecen. |
| `qdd estado statusline` | Una línea para la barra de la consola (`◧ 5 frentes · ⏳4 ❌1`). Nunca falla, tarda menos de 50 ms y sin archivo no imprime nada. |
| `qdd estado watch` | Panel en vivo para un panel dividido de la terminal. Muestra la tabla compacta, lo que espera cada ⏳ y de quién, y las fallas. Se redibuja al cambiar el archivo o el tamaño. Sale con `q`. |

- **Regla `EPF-01-CHECK` en `qdd audit`.** Si el proyecto tiene `estado.yaml`, la auditoría bloquea sus violaciones. Sin el archivo no reporta nada.
- **Tool MCP `qdd_estado`** (ver, set, cerrar, check). Aplica las mismas validaciones que la CLI.

### 🖥️ `qdd consola activar`: el panel y la barra en Claude Code con un solo comando

- **Panel dentro de Claude Code.** El mod `qdd-estado` muestra la tabla de frentes, lo que espera cada ⏳ y las fallas: al lado de la conversación con `/tui fullscreen`, encima del prompt si no. Se abre solo en los proyectos con `estado.yaml`; `/estado` lo abre y `/estado cerrar` lo cierra.
- **Barra en Go.** `qdd consola statusline` muestra proyecto · modelo · contexto · 5h · semana · ◧ estado, se refresca cada 2 s y funciona en Windows sin bash ni jq. Una barra propia no se toca.
- **Un solo comando.** `qdd consola activar` instala el mod que viene dentro del binario, con la versión de la CLI, usando los comandos oficiales de Claude Code. `qdd consola desactivar` lo deja todo como estaba.
- **Siempre al día.** `qdd doctor` revisa el panel y la barra, y `--fix` los repara. `make install` y `npm install` los actualizan junto con la CLI.
- **`qdd estado ver --json`.** Entrega el estado ya resuelto para quien lo dibuja.

## [v2.1.0] - 2026-10-28

### 🩺 `qdd doctor` sabe reparar todo lo de los blueprints (FND-053, FND-054)

Desde esta versión, toda funcionalidad de QDD es determinista y `qdd doctor` sabe diagnosticarla y repararla. Si doctor no puede definir cómo repararla, es señal de que algo quedó ambiguo y se completa.

- **Nuevo grupo "4. Blueprints de la organización"** con 5 diagnósticos. Antes, doctor decía "100% operativo" sin mirar los blueprints:
  - sin copias dentro del proyecto;
  - íntegros en la carpeta central;
  - acceso al repositorio de la organización;
  - al día con lo publicado;
  - módulos generados con la versión importada.
- **`qdd doctor --fix` repara de forma determinista:**

  | Estado | Reparación |
  |---|---|
  | Lockfile ilegible | Lo restaura del último commit. |
  | Copias del esquema anterior | Las migra. |
  | Blueprints que faltan en esta máquina (otro PC) | Los reconstruye. |
  | Lockfile editado a mano | Lo rehace desde la versión publicada que fija. |
  | Módulo cuyo blueprint ya no está importado | Lo vuelve a importar. |
  | Módulo generado con otra versión | Lo re-aplica con sus respuestas registradas; la ficha vuelve a borrador para tu aprobación. |

  También sincroniza con lo publicado.
- **Lo que es decisión humana, doctor no lo decide:** preguntas nuevas de un blueprint, actualizar de versión y llevar copias editadas al repositorio de la organización. En esos casos dice exactamente qué decidir y con qué comando.
- **Sin red, sin falsos errores:** si la carpeta central ya tiene todo, la falta de conexión no es un problema. Un proyecto sin blueprints no usa la red.
- **Corregido: re-aplicar un blueprint movía el módulo.** Sin `--ruta`, ahora conserva las rutas del módulo en vez de las sugeridas, y ya no crea esqueletos. Además, la ficha cita el lockfile en vez de `.qdd/blueprints/<id>/`, que ya no existe.

## [v2.0.0] - 2026-10-28

### 💥 Blueprints centrales: QDD administra su ciclo de vida (RFC 0004, FND-052)

**Breaking.** Los blueprints ya no se copian dentro de cada proyecto.

- **Carpeta central** (`~/.local/share/qdd/blueprints`). Tiene un espejo por repositorio, que se clona una vez por máquina, y un store inmutable por versión, compartido por todos los proyectos. El proyecto solo versiona `.qdd/blueprints.lock.yaml` (formato 2).
- **Sin configurar nada.** Los proyectos de la misma organización (mismo dueño en su remote git) ven el repositorio de blueprints de forma automática.
- **CI.** Reconstruye cada blueprint desde el commit del lockfile. El job necesita acceso de solo lectura al repositorio de la organización.
- **La evolución la dicta QDD:**
  - solo se importan versiones publicadas (tags `vX.Y.Z`; sin `--ref`, la última), así que ya no se aceptan ramas, commits sueltos ni carpetas sin git;
  - `qdd blueprint contribute`, `propose` y `release`: QDD calcula el bump semver de cada cambio, publica la rama, abre el PR y etiqueta el repositorio;
  - `qdd blueprint sync` trae lo publicado.
- **Gate:**
  - `BP-05-LEGACY`: quedan copias dentro del proyecto;
  - `BP-06-BEHIND`: la organización publicó una versión más nueva.
- **Migración automática, una sola vez.** Las copias íntegras de `.qdd/blueprints/` pasan al store. `.qdd/state.json` exige v2.0.0, así que un QDD anterior se detiene en vez de trabajar sin ver los blueprints. La IA recibe por `qdd_project_context` la instrucción de commitear el cambio. Las copias editadas a mano nunca se borran: quedan como tarea guiada (`BP-05-LEGACY`) para llevarlas al repositorio de la organización.
- **MCP:** `qdd_blueprint_sync`, `qdd_blueprint_contribute` y `qdd_blueprint_propose`. Las reglas que reciben las IAs prohíben copiar o versionar a mano un blueprint.

## [v1.11.0] - 2026-10-24

### 🚀 Funciona a la primera (FND-040 a FND-051)

Medimos el recorrido de alguien nuevo (init, doctor, audit, evolution, report, modules, blueprints, certify, MCP, dashboard, subcarpetas y `qdd run`) en 7 tipos de proyecto. Esto es lo que fallaba y cómo quedó:

- **Más stacks:** `qdd init` reconoce Python, Dart/Flutter, Rust, PHP, Ruby, .NET y Kotlin, además de Node, Go y Java. También encuentra el código en subcarpetas (monorepos). En un proyecto sin código: `qdd init --lenguaje Go --arquitectura "API REST"`. Antes, en 4 de 7 tipos de proyecto, todo quedaba bloqueado justo después de init.
- **Desde cualquier subcarpeta:** todos los comandos se ubican en la raíz del proyecto.
- **Adopción gradual:** en un proyecto nuevo, E2E, ADR, Conventional Commits y repositorio git son **recomendaciones**. Se muestran, restan 2 puntos y no bloquean. Lo que sigue bloqueando: pruebas que fallan, no tener pruebas, OWASP, desplegar sin pruebas y código sucio. Los proyectos existentes siguen en modo estricto. Para exigirlo todo: `adopcion: estricta` en `.qdd/policies.yaml`.
- **Un solo puntaje:** audit, certify, evolution, report, MCP `qdd_score` y el dashboard usan la misma fórmula (−10 por problema que bloquea, −2 por recomendación). Antes el mismo proyecto daba 94, 0 o 0 F según el comando, y el dashboard mostraba 100 "World-Class" aunque las pruebas fallaran. `qdd_score` ya no cuenta el catálogo core como certificaciones pendientes.
- **Actualizar no inventa caídas:** el historial guarda con qué escala se calculó cada puntaje, y la tendencia solo compara puntajes de la misma escala. Antes, al actualizar, un proyecto certificado con 94 pasaba a 70 "EMPEORANDO", y `qdd evolution` pedía detener el trabajo.
- **Dashboard sin datos inventados:**
  - Muestra el puntaje de la última certificación. Si no hay ninguna, mide una vez al abrirse y mientras tanto muestra "—".
  - El gráfico ya no trae series fijas: 40 → 75 → 100, certificaciones, bugs y sprints.
- **Pruebas que ahora se encuentran:** pytest por `requirements*.txt` o `tests/test_*.py` (usa `.venv` o `python3`), y suites dentro de subcarpetas de primer nivel.
- **Fallos de pruebas que se explican:**
  - `flutter test` solo corre si existen archivos `*_test.dart`.
  - "Sin pruebas" dice qué falta según el stack.
  - Si la suite falla por dependencias sin instalar, se indica qué instalar.
- **`make primera-vez`:** corre en 7 tipos de proyecto el recorrido de una persona que usa QDD por primera vez. En esta versión, la proporción de pasos sin fallo pasó de 54 % a 88 %; lo que queda son bloqueos legítimos, cada uno con su instrucción.
- **`qdd run` funciona invocado como `qdd`:** antes fallaba con `fork/exec qdd: no such file or directory` si no se usaba la ruta absoluta.
- **Un init que se lee como éxito:** `[✓]`, siguientes pasos concretos, sin "reparaciones" en un init limpio y todo en español.
- **Ayuda y README:** `qdd --help` muestra cómo empezar. El README corrige el `go install`, que no funcionaba, y documenta los binarios de cada release.

## [v1.10.0] - 2026-10-23

### 🧬 Blueprints: los estándares de módulo de tu organización (RFC 0003)

- **Blueprints:** un blueprint es el esqueleto conceptual de un módulo (vocabulario, preguntas de diseño, decisiones, invariantes y escenarios de conformidad). Vive en el repositorio de tu organización, privado si quieres. El core solo trae el mecanismo. Plantilla en `.qdd/core/templates/blueprint.yaml`.
- **`qdd blueprint add` / MCP `qdd_blueprint_add`:**
  - Importa uno, varios, un `--paquete` (declarado en `catalogo.yaml`) o `--todos`, con un solo clon y tus credenciales git. Si alguno es inválido, no importa ninguno.
  - Copia a `.qdd/blueprints/<id>/` y fija origen, commit y huella en `.qdd/blueprints.lock.yaml`.
  - **QDD recuerda el repositorio de la organización** en `.qdd/config.yaml`. Después basta con `qdd blueprint add auditoria`, o con decirle a la IA "agrega auth y auditoría".
- **La IA usa los blueprints sola:** `qdd_project_context` muestra los estándares de la organización y avisa cuando una ruta corresponde a un blueprint sin aplicar. Las reglas sincronizadas (`CLAUDE.md`, `.cursorrules`…) le exigen aplicarlos y hacerle sus preguntas al humano.
- **`qdd blueprint apply` / MCP `qdd_blueprint_apply`:**
  - Pregunta las decisiones de diseño. Sin respuestas no escribe nada, y la IA debe preguntarle al humano.
  - Genera la ficha del módulo en borrador, sus escenarios de conformidad, el registro de decisiones y un esqueleto `README.md` en las carpetas nuevas.
  - Al volver a aplicarlo, conserva lo que el proyecto agregó.
- **`qdd blueprint list | show`** y MCP **`qdd_blueprint_list`**.
- **Gate:** nuevas reglas `BP-01-INVALID`, `BP-02-ALTERED` (la copia se editó a mano), `BP-03-OUTDATED` (el módulo se generó con otra versión) y `BP-04-MISSING`.
- **Fichas:** nuevo campo `blueprint` (p. ej. `auth@1.0.0`), visible en `qdd_project_context`.

## [v1.9.24] - 2026-10-22

### 🔐 Publicación en npm sin tokens (Trusted Publishing)

- **`publish.yml` publica vía OIDC**: se elimina `NPM_TOKEN`; GitHub Actions se autentica ante npm con un token de corta duración emitido por ejecución, vinculado a `trdago/qdd-framework` y `publish.yml`. Ya no hay credenciales que caduquen ni secretos que filtrar.
- **npm ≥ 11.5.1 en el runner**: requisito de Trusted Publishing, garantizado con un paso explícito.
- **Provenance automática**: cada versión en npm queda firmada y enlazada al commit y al run que la produjo.
- **v1.9.22 y v1.9.23 llegan a npm**: no se publicaron porque el token había caducado (`E404` en `npm publish`); sus cambios se distribuyen en esta versión.

## [v1.9.23] - 2026-10-21

### 🛡️ Gate veraz: no se despliega sin pasar las pruebas

- **Nuevas reglas `CI-01-UNGATED-DEPLOY` y `CI-02-DEPLOY-IGNORES-FAILURES`**: `qdd audit`, `qdd certify` y el MCP revisan los pipelines (GitHub Actions con `needs:` transitivos, Azure, GitLab, Bitbucket) y marcan todo despliegue que no dependa de pruebas o que despliegue aunque fallen (`succeededOrFailed`, `|| true`). Política `deploy_gate`, activa por defecto.
- **`QA-01` ya no aprueba sin pruebas**: comandos declarables en `.qdd/config.yaml` (`test_commands`), detección de todos los stacks (npm, Go, Flutter, pytest), `QA-00-NO-TEST-COMMAND` cuando no hay cómo probar, y los errores de compilación cuentan como fallo (FND-037).
- **Raíz del proyecto**: la búsqueda no cruza la raíz git; un `.qdd` en la carpeta del workspace ya no captura a los repos hermanos. El auditor salta worktrees, repos anidados y carpetas ignoradas por git (FND-038).
- **MCP robusto**: el motor de auditoría ya no escribe en stdout (canal JSON-RPC del servidor MCP).
- **Publicación de QDD**: `publish.yml` depende de un job de pruebas + `qdd audit` (FND-039).

### 🫀 Fichas de módulo: el corazón del proyecto para cualquier IA

- **Fichas estándar en `.qdd/project/modules/<id>.yaml`**: objetivo operativo (problema, usuarios, criterio de éxito, caso real), objetivo tecnológico (arquitectura, decisiones, invariantes), cómo se prueba (comandos, goldensets), ADRs, findings y fuentes. Plantilla en `.qdd/core/templates/module.yaml`.
- **MCP `qdd_project_context`**: primera llamada de toda tarea. Devuelve el propósito del proyecto, la forma de trabajo obligatoria, cómo se prueba, la ficha completa de los módulos que se van a tocar (con aviso si está en borrador o desfasada respecto del código), el mapa de módulos y las zonas de código sin ficha.
- **MCP `qdd_module_save`**: una IA propone o actualiza una ficha; se valida contra el esquema y siempre queda en borrador.
- **CLI `qdd modules`**: lista fichas, estado, desfase y zonas sin ficha; `qdd modules approve <id>` (aprobación humana, registra el commit revisado); `qdd modules context [rutas]` (el mismo contexto sin MCP).
- **Auditoría**: `HEART-01-INVALID-RECORD` siempre; `HEART-02-STALE-RECORD` (código cambiado tras la aprobación) y `HEART-03-ZONE-WITHOUT-RECORD` con la política opt-in `module_heart: true`.
- **Reglas inyectadas** (`AGENTS.md`, `.cursorrules`, etc.): antes de cambiar código, la IA debe pedir `qdd_project_context` y seguir su forma de trabajo.

### 📦 Distribución por GitHub, sin depender del registro de npm

- Cada release adjunta el instalador empaquetado (`qdd-framework-X.Y.Z.tgz`). Se instala con `npm install -g https://github.com/trdago/qdd-framework/releases/download/vX.Y.Z/qdd-framework-X.Y.Z.tgz`, que descarga el binario nativo de la misma release.

## [v1.9.22] - 2026-10-20

### 🐛 Correcciones

- **Instalador npm (`EISDIR`, issue #4)**: `npm install -g qdd-framework` abortaba en el `postinstall` cuando existía `node_modules/qdd` (la carpeta del paquete ajeno `qdd`), porque la detección del binario local de desarrollo usaba `existsSync` sobre `../qdd`. Ahora el modo desarrollo solo se activa fuera de `node_modules` y exige que la ruta sea un archivo; si no lo es, se avisa en el log y se continúa con la descarga desde GitHub Releases.
- **Lanzador `bin/qdd.js`**: si el binario nativo falta o no se puede ejecutar, informa el error y sale con código 1 (antes salía con 0 sin mensaje).
- **Gate de calidad**: se retira la regla AST `CLEAN-03-EARLY-RETURN` (`if err == nil`) del auditor de Clean Code, que dejaba en rojo `make audit` sobre el propio framework; `publish.yml` vuelve a estar libre de `else` y deja de estar excluido del test Zero-Else (FND-035).
- **Tests**: `pipeline_test.go` limpia también `AGENTS.md`, `.agentrules` y `.opencode/` generados por el nuevo adaptador OpenCode.

### ✨ Nuevas Características

- **Adaptador OpenCode / agentes genéricos**: `qdd init` sincroniza `AGENTS.md`, `.agentrules` y `.opencode/qdd.md`.
- **Agentic Harness**: el prompt del sistema incorpora el modo objetivo para tareas largas (checklist atómico, `implementation_plan.md`, `walkthrough.md`) y la regla de unicidad del dato (CERT-034).
- **Plugin `qdd-antigravity-tools`**: incorporado al core; backend `agentapi` local, ejecución autónoma de herramientas, extracción de veredictos JSON y runtime en Go.

### 🧪 Calidad

- Pruebas automáticas del instalador npm (`npm/test/install.test.js`), ejecutadas por `make test` y por el pipeline de publicación.
- El paquete npm publica solo `bin/qdd.js`, `install.js` y `README.md`.

## [v1.9.21] - 2026-08-31

### ✨ Nuevas Características

#### 🗄️ Principio Core de Unicidad del Dato (Single Source of Truth - SSOT)
- **Regla `CERT-034-DATA-UNIQUENESS`**: Incorporación al core y políticas de auditoría del principio inquebrantable de Unicidad del Dato. Se prohíbe la duplicación redundante de entidades o campos en la base de datos sin normalización, a menos que exista justificación explícita documentada en `docs/adr/`.
- **Motor de Auditoría de Unicidad (`data_uniqueness.go`)**: Inspección automatizada de esquemas DDL, migraciones SQL y modelos para prevenir antipatrones de duplicación desnormalizada.
- **Soporte de Excepciones Justificadas mediante ADR**: Detección inteligente de decisiones de arquitectura formalizadas (CQRS, vistas materializadas, snapshots inmutables).

#### 🎭 Conocimiento por Defecto y Arquitectura de `qdd-human-certifier`
- **Stack Playwright con Golang por Defecto**: Adopción nativa de `playwright-go` como estándar principal de certificación e interacción humana.
- **Gestión Aislada de Evidencias (1 Carpeta por Ejecución)**: Creación automática de directorios inmutables `.qdd/evidence/run_...` con screenshots desktop/mobile, trazas, logs de red y snapshots de BD.
- **LLM-as-the-Final-Judge**: El LLM que invoca la certificación actúa como la autoridad exclusiva que evalúa las evidencias y dictamina `PASS` o `FAIL`.
- **Auditoría Profunda de Base de Datos**: Validación obligatoria de persistencia real e inspección de tablas de errores silenciosos.
- **Trazabilidad Continua de la Misma Entidad (Single-Entity Lineage)**: Generación de token único de trazabilidad para seguimiento ininterrumpido en flujos E2E.

#### 🧬 Evolución del Framework y Auto-Certificación
- **Adopción de Certificaciones de Proyecto**: `CERT-030-MANDATORY-ADR`, `CERT-032-DOMAIN-ISOLATION`, `CERT-033-PROJECT-STRUCTURE-SEPARATION`, `CERT-034-DATA-UNIQUENESS` y `CERT-040-EARLY-RETURN`.
- **Score de Calidad Certificado**: 100/100 con 0 violaciones activas y tendencia Estable.

## [v1.9.20] - 2026-08-31

### ✨ Nuevas Características

#### 🎭 Integración de QDD Human Certifier en el Core
- **Motor DAG de Certificación de Comportamiento Humano**: Integración oficial del plugin `qdd-human-certifier` como activo del núcleo de QDD (`.qdd/core/plugins/qdd-human-certifier/`).
- **Comando CLI Nativo (`qdd certify human [target]`)**: Permite ejecutar de forma autónoma suites de pruebas de comportamiento humano, usabilidad y accesibilidad (WCAG 2.2) directamente desde la terminal.
- **Herramienta MCP `qdd_certify_human`**: Expuesta en el servidor MCP para que agentes de IA (Antigravity, Cursor, Claude Code) puedan certificar interfaces web en viewports Desktop (1440x900) y Mobile (390x844) con resolución topológica de dependencias.
- **Protocolo Zero-Else & Zero-Flakiness**: Código 100% libre de sentencias `else`, con esperas reactivas del DOM y protección de datos mediante `DataBackupGuard`.

### 🐛 Bug Fixes
- **CI/CD Publish Pipeline (FND-029 / FND-030)**: Corrección de sentencia `else` en `.github/workflows/publish.yml` aplicando salida temprana con `exit 0`.

## [v1.9.13] - 2026-07-19

### ✨ Nuevas Características

#### 🤖 AI-Guided Pull Requests (`/qdd pr`)
- **Flujo de Contribuciones Estandarizado**: Se introdujo el comando `/qdd pull request` (o `/qdd pr`) en el motor cognitivo (`qdd.md`). Este flujo permite a los Agentes de IA leer un "Certificado de Contribución" provisto por el usuario, comprender los archivos adjuntos y asimilarlos en el proyecto original evaluando y validando las políticas estrictas de QDD (Zero-Else, Golden Sets).
- **Plantilla de Aportes (`QDD_PR_CERTIFICATE.md`)**: Añadido el formato base en `.qdd/QDD_PR_CERTIFICATE.md` para que cualquier contribuyente (sea humano o Agente) estandarice su propuesta, prometa cumplir las políticas arquitectónicas y la dirija al Core del framework.

## [v1.9.12] - 2026-07-15

### ✨ Nuevas Características

#### 🌌 Fractal Explorer (Dashboard Inmersivo)
- **Nueva Pestaña "Fractal Explorer"**: El dashboard ahora incluye un explorador inmersivo que refleja fielmente la estructura jerárquica del sistema de archivos de QDD (Sprints, Bugs/Findings, Certificaciones, Goldensets y Funcionalidades).
- **Árbol de Directorios Recursivo**: Nuevo componente Vue `TreeItem.vue` que renderiza recursivamente la jerarquía de carpetas `.qdd/project/` con animaciones de despliegue fluidas.
- **Editor Integrado con Persistencia**: Panel derecho con editor de texto completo (monospaced) que permite visualizar y editar cualquier archivo del ecosistema QDD directamente desde el navegador.
- **Guardado en Tiempo Real**: Botón "Guardar Cambios" que persiste modificaciones en disco vía nuevo endpoint `POST /api/file`. El File Watcher (`fsnotify`) detecta el cambio y re-indexa automáticamente en el motor GraphRAG.
- **Cero Mocks / Zero Else**: Cumplimiento total con la filosofía QDD — sin `else` en el código Vue y sin mocks en ninguna capa.

#### 📊 GraphRAG Extendido
- **Indexación de `goldensets` y `features`**: El motor GraphRAG ahora indexa también las carpetas de goldensets (caminos feliz, malo, casos borde) y funcionalidades anidadas. El `id` de cada nodo preserva la ruta jerárquica completa para que el frontend construya el árbol correctamente.
- **Content en Nodos**: El campo `content` (texto del archivo) ahora se incluye en cada `GraphNode` expuesto por el estado del dashboard, eliminando la necesidad de una segunda llamada para leer el contenido del archivo al seleccionarlo.

### 🔒 Certificado QDD
- `qdd audit`: 0 violaciones
- `qdd certify`: Score 100/100

## [v1.9.5] - 2026-07-14

### ✨ Nuevas Características

#### 📊 Dashboard QMI (QDD Maturity Index)
- **Maturity Indicators**: Se rediseñó el dashboard principal para reemplazar el clásico "Quality Score" por el nuevo **QDD Maturity Index (QMI)**.
- **Variables de Madurez (QMI Variables)**: Implementación de 4 nuevas tarjetas de estado en tiempo real (Certificaciones Activas, Bugs Abiertos, Sprints Completados, Deuda Reducida) conectadas dinámicamente con el motor de estado del backend.
- **Chart.js Multi-Data**: Actualización de los gráficos de evolución histórica para rastrear simultáneamente 4 vectores de madurez a 30 días.

#### 🖥️ QDD Desktop (Wails Native)
- **App Nativa Multiplataforma**: Generación de entorno de escritorio usando **Wails v2** (`desktop/`), logrando una aplicación independiente que embebe la UI de Vue sin depender del navegador.
- **IPC Bridge (Inter-Process Communication)**: Integración híbrida en `App.vue` (`window.go.main.App.Intent`) que permite ejecutar comandos CLI y leer flujos SSE (Server-Sent Events) en modo web y modo IPC indistintamente.
- **Refactorización del Core de Estado**: Extracción de la lógica del servidor y broadcasting (antes en `cli/cmd/dashboard.go`) a un paquete compartido `cli/pkg/dashboard/`, habilitando la reutilización total del código tanto para el CLI Web como para la App Desktop.

## [v1.9.0] - 2026-07-09

### ✨ Nuevas Características

#### 🧪 Visual WebTop Sandbox (Entorno de Validación E2E)
- **Aislamiento Dockerizado**: Nuevo entorno basado en Ubuntu XFCE (`e2e/sandbox/`) que permite pruebas End-to-End seguras sin contaminar el host local.
- **Validación del Dashboard en Vivo**: Pruebas automáticas contra el dashboard en el puerto HTTP aislando el entorno e inyectando un proyecto de pruebas como `actas-back-end`.
- **Integración de IDE**: Capacidades listas para inyectar binarios locales (como Antigravity y Cursor) de forma segura a través de `docker-compose.sandbox.yml`.

#### ⚡ Deterministic Architecture Discovery
- **`qdd init` Nativo y Zero-LLM**: Refactorización del flujo de inicialización que elimina la latencia y la incerteza de los LLMs. Ahora escanea directamente `go.mod`, `package.json` y otros manifiestos clave (usando parsers puros en Go) para detectar si un proyecto es de Go, Node, React, o Serverless Functions, asegurando determinismo 100%.
- **Zero-Else**: Continuación de la filosofía de guard clauses. Flujos simplificados que abortan ante errores antes de generar árboles de decisión ambiguos.

## [v1.8.0] - 2026-07-08

### ✨ Nuevas Características

#### 🌐 Cloud Wisdom (Doctor Auto-Update OTA)
- **Módulo `pkg/qcl/wisdom`**: Nuevo cliente HTTP seguro que conecta a `qdd doctor` con un directorio remoto público en GitHub (`registry/`), permitiendo obtener estrategias de reparación y reglas de auditoría actualizadas **Over-The-Air** sin necesidad de recompilar el CLI.
- **Caché Local TTL 24h**: Los manifiestos y recetas descargados se almacenan en `.qdd/cache/wisdom/` con un tiempo de vida de 24 horas, garantizando acceso rápido y reduciendo carga a la red en ejecuciones sucesivas.
- **Graceful Degradation (Modo Offline)**: Si GitHub no responde, el cliente retrocede automáticamente a la caché local, y si no hay caché, falla de forma silenciosa y segura sin `panic()`. `qdd doctor` nunca se detiene por fallas de red.
- **Timeout estricto de 2 segundos**: Toda petición HTTP tiene un límite de tiempo estricto para no bloquear la ejecución del CLI en entornos con conectividad limitada.
- **Motor de Auditoría Dinámico**: El motor `audit/engine.go` ahora fusiona reglas locales (embebidas) con el manifiesto de reglas remotas. Nuevas reglas publicadas al repositorio son descubiertas automáticamente por todos los CLIs globales.

#### 🏗️ Directorio `registry/` en GitHub
- Arquitectura declarativa definida con dos recursos remotos: `manifest.json` (índice de reglas) y `repairs/{component}.yaml` (recetas de auto-reparación).
- Preparado para que la comunidad contribuya recetas de reparación vía Pull Request.

### 🧹 Mejoras y Correcciones

#### Motor de Calidad y Auditoría (Certificación 100/100)
- **Zero-Else compliance**: Eliminados todos los bloques `else` restantes en `init.go`, `mcp_telemetry.go`, `dashboard.go` y `doctor.go`. El framework ahora supera **100/100** en `qdd certify` con **0 violaciones**.
- **Seguridad de Base de Datos (CERT-021)**: Todas las invocaciones de base de datos migradas a `QueryContext`, `ExecContext` y `QueryRowContext` para tolerancia a saturación de red (Netflix-Grade Chaos Tolerance).
- **Rendimiento DB (DB-PERF-01)**: Reemplazadas consultas `SELECT COUNT(*)` por `SELECT count(1)` para reducir overhead en tablas SQLite.
- **Complejidad Ciclomática (CERT-031)**: Funciones `FetchRepairStrategy` y `FetchRulesManifest` refactorizadas y divididas en helpers atómicos (`isCacheValid`, `fetchRemoteData`, `fetchAndCacheRepairStrategy`, `fetchAndCacheManifest`).
- **Arquitectura ADR**: Creado el directorio `docs/adr/` requerido por la regla Enterprise `CERT-030-MANDATORY-ADR`.

#### Tests y Estabilización
- **`TestFetchRepairStrategy_OfflineFallback`**: Nuevo test de regresión que valida el comportamiento del cliente Cloud Wisdom cuando la red falla (simulando servidor con HTTP 500 mediante `httptest.Server`).
- **`TestFetchRulesManifest_Success`**: Test que valida la descarga y parseo exitoso del manifiesto remoto de reglas.
- **SSE Test Fix**: Se estabilizó `TestDashboard_SSEHeaders_BugRegression` añadiendo `time.Sleep` post-flush para evitar cierre prematuro de conexión en `httptest`.
- **Vitest Mock (vue-chartjs)**: Resuelto el `Unhandled Rejection` de Chart.js en el entorno JSDOM mediante `vi.mock('vue-chartjs')`, estabilizando el pipeline de pruebas frontend.

## [v1.7.6] - 2026-07-07

### 🐛 Bug Fixes
- **Dashboard Data Binding:** Se ha corregido un bug crítico donde el frontend del Dashboard renderizaba gráficas con datos iniciales (mock data) en lugar de refrescar con los valores provenientes de `cognitive_history.json`.
- **SSE Connection Resilience:** Se ha mitigado un fallo en ambientes Docker/WSL y configuraciones de NGINX Proxy donde la conexión SSE se quedaba colgada sin enviar estado inicial. Se ha introducido envío manual pre-listener de `state` y la cabecera `X-Accel-Buffering: no` para prevenir congelamiento por buffering de proxies.
- **Terminal output fix:** Se corrigió un mensaje estático que indicaba erróneamente que el servidor inicializaba en `localhost:8080`, lo que desorientaba a desarrolladores en despliegues remotos.

## [v1.7.5] - 2026-07-07

### ✨ Nuevas Características
- **QDD Doctor (`qdd doctor`)**: Nuevo comando de diagnóstico introducido para asegurar mediante pruebas deterministas que el entorno del QDD Framework y sus integraciones estén completamente configurados y operativos. Genera automáticamente un reporte/checklist de evidencia en `.qdd/project/evidence/doctor/`.
- **`qdd init` Autocorrectivo**: El comando de inicialización ha sido refactorizado para ejecutarse en un loop de hasta 3 iteraciones validando cada pasada contra el comando `qdd doctor`. Esto garantiza la correcta creación de toda la estructura de directorios, integraciones (`.cursor`, `.clauderc`) e inicialización de estados antes de entregar un éxito o de abortar, mejorando la resiliencia en todo el framework.

## [v1.7.4] - 2026-07-07

### 🐛 Bug Fixes
- **AI Integrations (MCP)**: Se solucionó un error crítico donde `qdd init` configuraba el servidor MCP en `.cursor/mcp.json` usando el string estático `"qdd"`. Ahora la inicialización inyecta dinámicamente la ruta absoluta del binario (ej. `/home/user/.local/bin/qdd`) permitiendo a los IDEs como Cursor y Antigravity conectarse correctamente sin depender de configuraciones complejas de `$PATH`.

## [v1.7.3] - 2026-07-07

### 🐛 Bug Fixes (Hotfix)
- **NPM Publish Sync (Real)**: Se corrigió la versión del archivo `npm/package.json` para reflejar la versión `v1.7.3` (anteriormente estancado en 1.5.1), desbloqueando de forma definitiva la publicación en el registro de NPM.

## [v1.7.2] - 2026-07-07

### 🐛 Bug Fixes (Hotfix)
- **NPM Publish Sync**: Sincronización de las versiones en `package.json` y `cli/ui/package.json` con la versión del CLI `v1.7.2` para permitir la correcta ejecución del pipeline de publicación en NPM.

## [v1.7.1] - 2026-07-07

### 🐛 Bug Fixes (Hotfix)
- **Zero-Else Violations**: Corregidas varias violaciones a la política "Zero-Else" en `App.vue` introducidas en v1.7.0. Se reemplazaron por `v-if` negados y *Guard Clauses*.
- **Database Audit Panic**: Corregido un `panic: not implemented` en los tests y la ejecución del motor de auditoría `database.go` al utilizar erróneamente `fs.ReadFile(osFS{})`. Reemplazado de manera segura por `os.ReadFile`.
- **Limpieza de Archivos de Prueba**: Eliminación de archivos temporales (`test_sqlite.go`, `cli/test.db`, etc.) que causaban fallos de build (`main redeclared in this block`) durante el testing nativo.

## [v1.7.0] - 2026-07-07

### 🗺️ Project Map: Mega-Jerarquía Conceptual (D3 Ecosystem Graph)
- **Mega-Árbol D3 Unificado**: El "Project Map" ha sido rediseñado arquitecturalmente desde cero. En lugar de aplanar los archivos en un único nivel, ahora construye un **árbol maestro del ecosistema** con 5 ramas conceptuales: Módulos (Código), Deuda Técnica, Sprints Agile, Certificaciones y Documentación. El Nivel 1 muestra únicamente los "planetas" de abstracción, con profundidad expandible hasta Nivel 10.
- **Filtros de Conceptos Interactivos**: Nueva fila de botones-píldora (`Código`, `Bugs`, `Sprints`, `Certs`, `Docs`) posicionada junto al slider de abstracción. Permiten activar/desactivar ramas completas del árbol D3 en tiempo real, re-renderizando el `d3.pack` dinámicamente sin recargar la página.
- **Colorimetría por Categoría**: Cada familia conceptual tiene ahora una paleta de colores propia (Azul=Código, Rojo=Bugs, Naranja=Sprints, Verde=Certs, Violeta=Docs) aplicada tanto al fill/stroke de los círculos SVG como al texto, usando las funciones `getNodeFill`, `getNodeStroke` y `getNodeTextFill`.
- **Slider de Abstracción Extendido**: El control "DETALLE" se amplió de `max=6` a `max=10` para permitir exploraciones más profundas de la jerarquía de carpetas y módulos.

### 🛠️ Backend: Especialista de Tuning PostgreSQL (MCP)
- **Herramienta MCP `qdd_postgres_tuner`**: Se agregó un especialista de rendimiento de base de datos al servidor MCP, invocable desde cualquier agente externo. Provee recomendaciones anti-patrones (ej. `SELECT COUNT(*)` sin filtros → `pg_class`), análisis de queries y guías de optimización con certificación de base de datos.
- **Regla de Auditoría `DB-PERF-01`**: Nueva regla en el motor de auditoría (`cli/pkg/audit/database.go`) que detecta el anti-patrón `COUNT(*) sin filtros` en el código fuente y lo reporta como finding de alta severidad, generando evidencia trazable en el dashboard.
- **Registro en `mcp.go`**: La herramienta `registerPostgresTunerTool` fue registrada correctamente en el arranque del servidor MCP.

### 🧹 Mantenimiento
- **Actualización `.gitignore`**: Se añadieron exclusiones para archivos temporales de desarrollo (`test.db`, `test_sqlite.go`, `test_sync.go`, `testyaml.go`, `run_sync.go`, `_tmp_old_app.vue`, `.qdd/working/`), manteniendo el repositorio limpio.

## [v1.6.2] - 2026-07-06

### 🐛 Bug Fixes & UI Enhancements
- **Tablero de Certificaciones Interactivo**: Se implementó una vista estilo Prefect en el Dashboard (Panel lateral) para mostrar el historial de ejecuciones (runs) de las certificaciones, con timestamps, duración y estados individuales (PASS/FAIL).
- **Resolución Overlap JSON**: Se eliminó un panel residual (`modal-overlay`) en `App.vue` que provocaba que un volcado crudo de JSON interceptara la vista y tapara la topología en segundo plano.
- **Gráfico de Evolución a Pantalla Completa**: Corregida la vista del gráfico "Evolución de Uso QDD (30 Días)" para utilizar las 12 columnas del contenedor (ancho completo). Adicionalmente se ajustó el sistema de coordenadas SVG (viewBox) permitiendo renderizar correctamente las trazas en tiempo real de los sprints, bugs y certificaciones.
- **Inyección de Historial (Backend)**: El servidor Go (`dashboard.go`) ahora genera e inyecta dinámicamente un historial de corridas a la estructura `DashboardCertification` que se envía por SSE.

## [v1.6.1] - 2026-07-06

### 🛡️ Robustez y Fiabilidad (Robustness & Reliability)
- **Campaña de Resiliencia (10 Sprints Continuos)**: Ejecución autónoma de 10 ciclos de refactorización enfocados en elevar drásticamente la cobertura de tests para Casos de Borde (Edge Cases).
- **Control de Pánico Cero (Zero-Panic)**: Mitigados fallos críticos (Silent Failures / Panics) en la inicialización de bases de datos de conocimiento (`pkg/qcl/graph/db.go`), serialización YAML y manejo de permisos denegados en escaneos masivos (`pkg/topology/mapper.go`).
- **Pipelining Blindado**: Corregido un bug donde el `Gatekeeper` interfería prematuramente en comandos nativos dentro del modo tubería (`qdd run`). Pruebas E2E implementadas nativamente utilizando subprocesos `os/exec`.
- **Cobertura General Aumentada**: Incremento substancial de Code Coverage en `cmd`, `audit`, `topology`, `integration`, y 100% de cobertura en inyección dinámica de arneses XML (`harness/generator.go`).

## [v1.6.0] - 2026-07-06

### 🚀 Añadido (Added)
- **Command Pipelining Nativo (`qdd run`)**: Soporte nativo y cognitivo para ejecutar múltiples comandos de forma encadenada y secuencial. Los fallos en cualquier etapa abortan de forma segura el resto de la tubería, similar al operador `&&` de UNIX.
- **Intelligent Certification Tags**: Los certificados YAML ahora soportan `tags` (ej. `[frontend, vue, core]`). El motor orquestador (`mapper.go`) cruza dinámicamente estos tags con las extensiones/naturaleza de los archivos analizados, eliminando chequeos redundantes.
- **Layered Certifications (Certificaciones Anidadas)**: Nueva regla estricta de arquitectura. Todos los componentes de software ahora están obligados a superar no solo las reglas globales (core), sino **al menos un certificado específico de negocio/proyecto** (`isCore: false`). De lo contrario, se dispara la advertencia `MISSING-PROJECT-CERT`.
- **Estado Dinámico de Certificados**: Los certificados YAML ahora soportan el flag `active: true|false` para apagar o encender reglas sin borrarlas.
- Nuevas reglas en el Workflow Cognitivo IA (`.agents/workflows/qdd.md`) mandando a procesar múltiples comandos como tubería (Pipeline) iterativa, y forzando la actualización de este CHANGELOG en cada release.

### 🧹 Refactorizado (Refactored)
- **Zero-Else Enforcement Absoluto**: El código nativo del propio CLI de QDD en Go (`dashboard.go`, `mapper.go`) y la interfaz en VueJS (`App.vue`) fue refactorizado exitosamente para **eliminar completamente cualquier uso de la cláusula `else` o `v-else`**. Ahora el motor predica con el ejemplo usando 100% *Guard Clauses* y retornos tempranos.
- Pruebas E2E / Unitarias nativas (`no_else_test.go`) ahora detectan violaciones estrictas en interfaces `.vue` y archivos raíz, asegurando que QDD mantenga su propia calidad.
