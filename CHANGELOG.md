# Changelog

Todos los cambios notables de este proyecto serán documentados en este archivo.

El formato está basado en [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/), y este proyecto adhiere a [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
