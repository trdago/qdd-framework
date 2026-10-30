# RFC 0006: Estado por frente en la consola — `qdd estado`

**Status:** Accepted (aprobado por el humano el 2026-10-29; implementado en la rama feat/estado-por-frente)
**Created:** 2026-10-29
**Relacionado:** blueprint `estado-por-frente` (v1.0.0, opción `destino: archivo`), blueprint `consola-claude-code` (v2.0.0), [RFC 0004](0004-blueprints-centrales.md).

## 1. Summary

El blueprint `estado-por-frente` define la tabla de frentes × etapas y sus invariantes, pero hoy la tabla solo vive en la prosa de cada respuesta. Nadie la puede ver de un vistazo mientras trabaja, ni comprobar con una máquina.

Este RFC agrega a la CLI un comando `qdd estado`, escrito en Go puro y agnóstico al sistema operativo. Hace cuatro cosas:

1. lee y escribe el archivo `.qdd/project/estado.yaml` (opción `destino: archivo` del blueprint);
2. comprueba las invariantes con un lector automático: los escenarios de conformidad `EPF-C*` y `EPF-A01`;
3. entrega un resumen de una línea para la barra de estado de Claude Code;
4. abre un panel en vivo (`watch`), pensado para un panel dividido de la terminal, por ejemplo arriba a la derecha.

## 2. Motivation

- El criterio de éxito del blueprint pide que «un lector automático pueda comprobarlo»; hoy no existe ese lector.
- Una sesión cortada o un agente nuevo solo retoma el trabajo si los ⏳ quedan en un archivo (decisión de `destino: archivo`).
- El humano quiere ver los indicadores **en la consola donde trabaja**, como lo hace opencode.

**Por qué no un recuadro superpuesto.** opencode puede dibujar arriba a la derecha porque es dueño de toda la pantalla. Claude Code redibuja la pantalla completa a cada rato: un proceso aparte que pinte encima parpadea o queda tapado. Las superficies confiables son dos:

- la **barra de estado**, que Claude Code ofrece para esto;
- un **panel dividido** de la terminal (tmux, Windows Terminal, WezTerm, iTerm2), donde `qdd estado watch` es dueño de su propio espacio.

## 3. Design

### 3.1 Archivo `.qdd/project/estado.yaml` (formato 1)

```yaml
formato: 1
etapas: [Código, Pruebas, Producción, Validación real]   # las de la ficha del módulo (opción «etapas»)
actualizado: 2026-10-29T14:10:00-03:00
frentes:
  - id: gemini-retirado                  # estable: así se sigue un frente entre reportes
    nombre: Gemini retirado (ADR-008)
    estado: abierto                      # abierto | cerrado | descartado
    cerrado_el: ""                       # obligatorio si no está abierto
    motivo_cierre: ""                    # obligatorio si no está abierto (EPF-C05)
    celdas:
      Código:          {estado: "✅", evidencia: "commit e29393e6"}
      Pruebas:         {estado: "✅", evidencia: "gate 90/100"}
      Producción:      {estado: "✅", evidencia: "run 40828808431"}
      Validación real: {estado: "⏳", bloqueo: "ver un análisis con imagen leída por Textract", de_quien: "uso real"}
```

Cada celda tiene uno de cuatro estados cerrados, cada uno con su campo obligatorio:

| Estado | Campo obligatorio |
|---|---|
| ✅ | `evidencia` (commit, versión, n/n pruebas o id de corrida) |
| ⏳ | `bloqueo` + `de_quien` |
| ❌ | `falla` (se nombra) |
| — | ninguno |

`requiere_aprobacion: "<quién>"` sin `aprobado: true` impide el ✅ (EPF-C06).

Reglas del archivo:

- **Un frente no se borra.** Se cierra (`estado: cerrado|descartado` con `motivo_cierre`). Los cerrados se siguen mostrando 7 días (alcance `abiertos` = abiertos + recién cerrados) y después quedan en el archivo, ocultos.
- **Escritura atómica** (temporal + rename) y con validación: `qdd estado set` nunca deja un archivo que no pase `check`.
- **Sin secretos ni datos personales** (EPF-C07): `check` y `set` rechazan textos con forma de credencial (`sk-…`, `ghp_…`, `AKIA…`, `password=`), correos y RUT.

### 3.2 Comandos

| Comando | Qué hace |
|---|---|
| `qdd estado` (= `ver`) | Imprime la tabla en markdown (o `--formato mensajeria`) con el alcance de la ficha. |
| `qdd estado set <frente> <etapa> <✅\|⏳\|❌\|—> [--evidencia …] [--bloqueo … --de-quien …] [--falla …]` | Crea o actualiza una celda (y el frente, con `--nombre`). Valida antes de escribir. Acepta `ok`, `espera`, `falla` y `na` como alias para no escribir emojis. |
| `qdd estado cerrar <frente> --motivo … [--descartado]` | Cierra un frente con su motivo. |
| `qdd estado check [--respuesta archivo.md] [--json]` | Comprueba EPF-C01…C07. Con `--respuesta`, también EPF-A01: la tabla de la respuesta coincide con el archivo. EPF-C08 (texto y tabla) no se puede automatizar y se informa como revisión humana. Sale con código 1 si hay violaciones. |
| `qdd estado statusline [--dir <ruta>]` | Una línea corta para la barra: `◧ 3 frentes · ⏳2 ❌1`. Nunca falla ni escribe en stderr. Sin archivo, no imprime nada. Tope de 50 ms. |
| `qdd estado watch [--intervalo 1s]` | Panel en vivo en pantalla alternativa: se redibuja cuando cambia el archivo (sondeo del mtime, sin dependencias de inotify), se ajusta al ancho y muestra la tabla compacta más los ⏳ (qué espera y de quién) y los ❌. `q` o Ctrl-C salen y restauran la terminal. |

**C05 entre versiones.** `check` compara con la versión del archivo en `git HEAD`. Si un frente que estaba ya no aparece, es violación: hay que cerrarlo, no borrarlo.

**C04 desplegado no es validado.** `Validación real` en ✅ exige una evidencia propia, distinta de la de `Producción`. La regla se lee de la opción `etapas: software` de la ficha; con `modelos`, se aplica «Producción ✅ exige Puerta ✅».

### 3.3 Integración con el gate y el MCP

- `qdd audit` agrega la regla `EPF-01-CHECK`: si el módulo usa `destino: archivo` y el archivo existe, corre `check` (sin `--respuesta`).
- Tool MCP `qdd_estado` con `ver`, `set`, `cerrar` y `check`: los agentes actualizan el archivo por la misma validación que el humano.

### 3.4 Barra de estado (`consola-claude-code` v2.0.0)

La invariante de la v1.0.0, «La barra solo usa el JSON de la consola: sin red, sin archivos ni llamadas extra», impedía leer `estado.yaml`. Se reescribió, y por eso QDD fijó la versión **mayor** (2.0.0, publicada el 2026-10-29). La pregunta nueva es `consola_qdd` (§3.6):

- La barra puede mostrar un único segmento extra que entrega `qdd estado statusline`.
- Ese segmento lee solo archivos locales del proyecto. No usa red, tiene un tope de 100 ms y, si falla o tarda, no muestra nada.
- Los demás segmentos siguen usando solo el JSON de la consola.

### 3.4.1 Panel dentro de Claude Code (mod `qdd-estado`)

Claude Code (v2.1.287+) deja que un plugin de tipo *mod* dibuje un panel: al lado de la conversación si la terminal es ancha y usa la vista fullscreen (`/tui fullscreen`), y encima del prompt si no. Es la vía oficial para tener la tabla en la misma consola, como OpenCode, sin que otro programa escriba sobre la pantalla de Claude Code.

- `qdd estado ver --json` entrega el estado ya resuelto (`estado.Datos`): resumen, etapas y abreviaturas, un símbolo por celda, y las listas de ⏳ (con su bloqueo y de quién) y de ❌. Quien dibuja no interpreta reglas ni lee el YAML.
- El mod (`cli/pkg/consola/mod/qdd-estado`, embebido en el binario) llama a esa salida cada 2 s (`$.process.run`, tope de 3 s) y redibuja solo si cambió. Sin `qdd`, con un `qdd` antiguo o sin archivo, el panel dice cómo crearlo. `/estado` abre el panel y `/estado cerrar` lo cierra. `QDD_BIN` apunta a otro binario.
- El panel se abre solo si hay frentes. Si se abre sin que el usuario lo pida, Claude Code solo lo muestra desde 144 columnas.
- Pruebas: `claude plugin validate` y `claude plugin test` (dibujo, sin archivo, redibujo al cambiar).
- Probado en una sesión real con fullscreen en Linux (2026-10-29). Windows: pendiente de una prueba real.

### 3.6 Un solo interruptor: `qdd consola activar`

Decisión del humano (2026-10-29): el usuario solo activa algo; QDD hace el resto.

- `qdd consola activar [--sin-barra]` escribe el mod embebido, con la versión de la CLI en su `plugin.json`, en un marketplace local de la carpeta central (`<QDD_HOME>/claude`). Lo instala con los comandos oficiales de Claude Code: `claude plugin marketplace add <dir> --scope user`, `claude plugin install qdd-estado@qdd --scope user` y `claude plugin update` si la versión instalada es otra. Repetirlo no cambia nada.
- La barra pasa a `qdd consola statusline` (Go, con `refreshInterval: 2`): la misma línea que la referencia del blueprint más el segmento de estado, sin bash ni jq, y por eso también funciona en Windows. Solo reemplaza una barra vacía, la de QDD o el script de referencia v1.0.0 sin cambios; una barra propia no se toca. Antes de cambiar `settings.json` deja una copia en `settings.json.qdd.bak`.
- `qdd consola desactivar` desinstala el panel, quita el marketplace y devuelve la barra anterior. `qdd consola estado` dice si está activo y al día.
- `qdd doctor` suma el grupo «6. Consola de Claude Code» solo si se activó en esa máquina: panel instalado, habilitado y en la versión de la CLI, y barra de QDD si se eligió. `qdd doctor --fix` lo repara.
- `make install` e `npm install` ejecutan `qdd consola actualizar --si-activo`, así que el panel se actualiza junto con la CLI.
- Nada se escribe en el proyecto. El blueprint `consola-claude-code` v2.0.0 (pregunta `consola_qdd`) define si se quiere el panel; la ficha lo registra y la persona lo activa con un comando.

### 3.7 Acciones y jerarquía visual del panel (`consola-claude-code` v3.0.0, qdd 2.3.0)

Decisión del humano (2026-10-29): el usuario vio el prototipo y pidió dejarlo así en el blueprint. El panel pasa de solo
dibujar a ofrecer botones, pero la lógica sigue en Go.

- `qdd consola panel --json` entrega en una sola lectura lo que dibuja el panel: `{formato, estado, motivo, acciones}`.
  `estado` es lo mismo que `qdd estado ver --json` (o `null` con el `motivo`, p. ej. sin `estado.yaml`), y `acciones` es
  la lista que qdd ofrece en ese momento: `aprobar:<ficha>` por cada ficha en `borrador` (hasta 9), `revisar-estado`
  (`qdd estado check`, solo si hay `estado.yaml`) y `auditar` (`qdd audit`). Cada acción trae `id`, `tipo`
  (`aprobar` | `comando`), `etiqueta`, `detalle` (el comando equivalente) y `confirmar` (`true` si escribe).
- `qdd consola accion <id> [--confirmado]` vuelve a calcular las acciones, rechaza un id que no está en la lista y una
  acción con `confirmar` sin `--confirmado`, y la ejecuta con el mismo binario de qdd en la raíz del proyecto. Sale con
  el código de la acción. El panel no arma comandos: solo pasa ids.
- El mod dibuja cuatro niveles: cabecera con borde y los tres conteos en números grandes de 3 filas (█ ▀ ▄), títulos en
  negrita, texto y detalle atenuado. Cada frente toma el color de su peor etapa (❌ rojo, ⏳ amarillo, ✅ verde). El tamaño
  de letra (`size`) se declara, pero la terminal lo ignora; en las apps que lo respetan se ve de verdad.
- Teclas: `1…9` aprueban la ficha de esa fila, `c` revisa el estado y `u` audita. Lo que escribe abre un recuadro
  «¿…?» con `s  Sí` y `n  No` dentro del mismo panel. El resultado (últimas líneas de la salida) queda en un recuadro
  verde o rojo, y el panel se vuelve a leer.

### 3.8 Línea de recursos en la barra (`consola-claude-code` v4.0.0, pregunta `recursos`; qdd 2.4.0)

Decisión del humano (2026-10-29): la barra muestra el uso del equipo con etiquetas de una letra para ahorrar espacio.
Va en la barra, no en el panel. D es el disco de la carpeta del proyecto y G0…Gn el % de uso de cada GPU.

- `qdd consola activar --recursos` lo guarda en `activo.json` (`--sin-recursos` lo quita). `actualizar` y
  `doctor --fix` conservan la elección. La barra suma una segunda línea: `D 70% · G0 7% · G1 — · R 38% · C 12%`.
- **D**: usado / (usado + disponible) del disco de `workspace.project_dir`, como `df` (statfs; en Windows,
  `GetDiskFreeSpaceEx`). **R**: (total − disponible) / total (`/proc/meminfo`; `GlobalMemoryStatusEx`). **C**: uso
  entre dos lecturas de la barra (`/proc/stat`; `GetSystemTimes`). La muestra anterior se guarda en
  `<QDD_HOME>/claude/recursos.json`; en la primera lectura, o si la anterior tiene más de 1 min, C no aparece.
- **G**: primero las NVIDIA (`nvidia-smi --query-gpu=utilization.gpu`, como mucho cada 5 s y con tope de 300 ms; entre
  consultas se usa la caché), después las demás de `/sys/class/drm` en orden de tarjeta (`gpu_busy_percent`, que exponen
  p. ej. las AMD). Una GPU que existe sin contador legible, como la integrada de Intel sin root, va con `—`.
- macOS mide solo D (RAM, CPU y GPU necesitan APIs de Mach que Go no expone sin cgo); otros sistemas no muestran la línea.
  Toda la medición tiene un tope de 400 ms: si se pasa, la barra sale sin la segunda línea.

### 3.9 Comandos que la IA deja para la persona (`consola-claude-code` v4.1.0; qdd 2.6.0)

Decisión del humano (2026-10-30): cuando la IA no puede o no debe ejecutar un comando (no tiene permiso, se lo
bloquearon, o el comando publica algo, como el push del tag de una versión), no le pide a la persona que lo escriba con
`!`: lo deja en su panel, que lo ejecuta con un clic. Vale **cualquier comando**, siempre con la confirmación «s  Sí».

- La IA lo deja con la tool MCP `qdd_consola_pedir` (`comando`, `motivo`, `carpeta`) o con
  `qdd consola pedir -m <motivo> [--carpeta <dir>] <comando…>`. QDD **no lo ejecuta**: solo lo guarda.
- La cola es de esta máquina y no se versiona: `<QDD_HOME>/claude/pedidos/<carpeta>-<hash>.json`, una por proyecto
  (privada, escritura atómica). El mismo comando en la misma carpeta no se duplica; el tope es 9.
- `qdd consola panel --json` entrega cada pedido como una acción `pedido:<id>` con su motivo (`etiqueta`), el comando
  exacto (`detalle`), la `carpeta` y la `tecla` (a, b, d…: no usan las de otros botones). El panel los muestra en
  «▶ Para ti» y se abre solo cuando llega uno nuevo. Orden de las secciones (qdd 2.6.1, decisión del humano): estado
  por frente (cabecera y tabla), Acciones, «▶ Para ti», ⏳ Esperando y ❌ Fallas. La confirmación aparece en la
  sección del botón que se pulsó.
- La confirmación muestra el motivo, «Ejecutará: <comando>» y «En: <carpeta>». Con «s  Sí» el panel pide
  `qdd consola accion pedido:<id> --confirmado`, que corre el comando con la shell del sistema (`sh -c`; en Windows,
  `cmd /C`) en su carpeta. «x  Descartar» lo quita sin ejecutarlo (`qdd consola descartar <id>`). `qdd consola
  pedidos` lista la cola.
- Estados (qdd 2.6.2): un pedido pasa por `pendiente` → `ejecutando` → `hecho` (salió con 0) o `fallo`, y la acción
  los entrega en `estado`, con `inicio`, `fin` y `resultado` (últimas líneas de la salida). Solo uno pendiente o con
  falla se ejecuta: el paso a `ejecutando` se hace con la cola bloqueada (un archivo `.lock`), así un pedido aceptado
  no corre dos veces (FND-070). Cada pedido guarda su tecla desde que se pide y la conserva mientras está abierto; la
  de uno hecho queda libre (FND-071). En el panel, cada pedido ocupa tres líneas (qué, `$ comando`, dónde y cuándo):
  pendiente con su botón; ejecutando (⟳) y hecho (✓) en gris y sin botón; con falla (✗) en rojo, con el error y
  «↻ Reintentar» en su misma tecla. El panel marca el pedido como ejecutando apenas se confirma, sin esperar a qdd.
  Un hecho se sigue viendo 30 minutos; uno que lleva más de 15 minutos ejecutándose se da por cortado (fallo). El
  tope de 9 y los duplicados cuentan solo los abiertos. Una cola de qdd 2.6.0 (sin estado ni tecla) se lee como
  pendiente.
- Las instrucciones para la IA (bloque de AGENTS.md, CLAUDE.md…) le dicen que use `qdd_consola_pedir` y que nunca
  ejecute ella misma `qdd consola accion` sobre un pedido. QDD no puede impedirlo técnicamente a un agente con shell
  propia: lo que lo protege es que la confirmación está en la interfaz de la persona y los permisos del agente.
### 3.10 Panel liviano, solo lo pendiente y la carpeta del proyecto (`consola-claude-code` v6.0.0; qdd 2.7.0)

Decisiones del humano (2026-10-30):
1. El panel se ve **liviano**: el usuario comparó el panel cargado (cabecera con borde y números grandes) con el de un
   proyecto sin frentes y prefirió este. Reemplaza el primer nivel de §3.7.
2. **Un frente terminado sale del panel.** En el panel los frentes se llaman **tareas**, se ven solo las **4 más
   nuevas** y cada una tiene una **✕** para descartarla y sacarla «del mapa mental» si no se va a hacer nada.
3. Hay un botón para **abrir la carpeta del proyecto** con el gestor de archivos que el sistema tenga configurado.

**Panel liviano.** La cabecera es una línea `Tareas · HH:MM` en cian y negrita, con la versión de qdd
atenuada a la derecha (`version` en `qdd consola panel --json`), y debajo el resumen en una línea con el color de cada
conteo: `4 de 9 tareas · ⏳ 3 esperan · ❌ 1 fallan`, más `▶ N para ti` si hay comandos de §3.9 que se pueden
  ejecutar (pendientes o con falla; lo mismo en OpenCode). Ya no hay recuadro ni números de 3 filas. Las listas de ⏳ y ❌ van en
dos líneas por elemento: `• frente › etapa` y, atenuado, `→ qué falta · de quién`. La tabla, los botones, la
confirmación y el recuadro del resultado no cambian.

**Solo lo pendiente, 4 tareas.** El panel usa `estado.DatosPendientes(e, now, 4)`: de los frentes `abierto` con alguna
etapa que no esté en ✅ ni en —, los 4 más nuevos (los últimos del archivo, que es donde `qdd estado set` agrega uno
nuevo). `ocultos` cuenta los pendientes más antiguos, y el panel los resume en «+5 más antiguas · qdd estado ver». Los
⏳ y ❌ que se listan son solo los de esas 4 tareas. Un frente terminado (todo ✅ o —), cerrado o descartado sale en el
acto, sin los 7 días de `RecienCerrado`. Sigue en `estado.yaml` y en `qdd estado ver --todos`; `qdd estado ver`, la barra
y el archivo (`frentes:`) no cambian de nombre.

**La ✕ de cada tarea.** `qdd consola panel --json` ofrece `descartar:<id>` (`tipo: descartar`, `confirmar: true`) por
cada tarea visible. La ✕ va al final de su fila y se pulsa con el mouse. Abre el recuadro «¿Descartar «…»?» con
`s  Sí` / `n  No`, y con «Sí» hace lo mismo que `qdd estado cerrar <id> --descartado --motivo "descartada desde el panel
de Claude Code (✕)"`. La tarea nunca desaparece en silencio: queda en el archivo con estado y motivo, y su lugar lo toma
la siguiente más nueva. Si la fila no cumple las invariantes de `qdd estado`, no se escribe y el panel muestra el motivo.

**Aviso de aprobación sin commit.** En un repositorio sin commits, aprobar una ficha ya no dice «(revisada en )»:
dice «(sin commit de referencia: el repositorio aún no tiene commits)» (`modules.TextoAprobada`).

**Abrir la carpeta.**

- `qdd consola panel --json` suma `carpeta` (la raíz del proyecto) y la acción `abrir-carpeta` (`tipo: abrir`,
  `confirmar: false`). Se ofrece solo en un proyecto QDD y si existe el abridor del sistema: `xdg-open` en Linux y BSD
  (respeta la aplicación predeterminada para carpetas de XDG), `open` en macOS y `explorer` en Windows. El `detalle` es
  el comando equivalente (`xdg-open /ruta`).
- `qdd consola accion abrir-carpeta` lanza el abridor en su propia sesión (sobrevive al fin de qdd y al tope del panel).
  Si termina con error dentro de 3 s, lo informa («¿hay una aplicación predeterminada para carpetas?») y sale con 1; si
  sigue vivo, el gestor ya está abriendo y sale con 0.
- El mod dibuja arriba, a la derecha, `📂 ~/…/carpeta` (acortada a `~` y, si no cabe, al final) y el botón
  `o  Abrir carpeta`, que es lo clicable (la tecla es `o`, de *open*: `a, b, d…` son de los comandos de §3.9). No se usa un enlace: el elemento `Link` de los mods solo acepta `https:` (o
  `http://localhost`), no `file://`, y los mods no tienen una API para abrir archivos. «Abrir carpeta» no se repite en la
  lista de acciones de abajo. Sin abridor en el sistema, se ve la carpeta sin el botón. Como las demás acciones, el panel
  no arma comandos: solo pide `abrir-carpeta`.
- Probado en Omarchy (Hyprland, `inode/directory` → `files.desktop`): abre Thunar en la raíz y `qdd` sale en 0,1 s.

### 3.11 El panel en OpenCode: solo actualizar y sincronizar (qdd 2.7.0)

Decisión del humano (2026-10-30): llevar a OpenCode **solo la forma de actualizar y sincronizar el panel**, es decir,
cómo se mantiene al día con el proyecto y con la versión de qdd, y dónde vive. En OpenCode el panel es **solo para
mirar**: sin botones, ✕ ni diálogos, que siguen siendo de Claude Code.

- **Dónde.** En la barra lateral derecha de OpenCode, con el espacio `sidebar_content` (orden 50, encima del contexto,
  MCP, LSP, todo y archivos de OpenCode). OpenCode la muestra en la vista de una sesión y la oculta si la terminal es
  angosta; se alterna con su propio atajo.
- **Cómo se sincroniza.** Usa el mismo contrato que Claude Code: cada 2 s ejecuta `qdd consola panel --json` en la carpeta
  del proyecto de la sesión (`api.state.path.directory`), con un tope de 3 s, y redibuja solo si el JSON cambió. Dibuja
  lo mismo, en versión de lectura: `Tareas · HH:MM` con la versión, el resumen en una línea, las 4 tareas más nuevas, las
  listas ⏳/❌ y cuántas fichas esperan aprobación. Sin `estado.yaml`, sin qdd o con un qdd antiguo, dice el motivo y
  OpenCode sigue igual. `QDD_BIN` apunta a otro binario.
- **Cómo se instala y se mantiene al día.** El plugin viaja dentro del binario de qdd, como el de Claude Code, así que su
  versión es siempre la de la CLI. `qdd consola activar` lo deja en `<QDD_HOME>/opencode/qdd-tareas.tsx` (id `qdd.tareas`), con la versión
  escrita en el archivo, si en la máquina hay OpenCode (`opencode` en el PATH o su carpeta de configuración). Lo registra
  en `tui.json`, en la clave `"plugin"`, de `$XDG_CONFIG_HOME/opencode` (o `~/.config/opencode`). Antes de tocar
  `tui.json` por primera vez guarda `tui.json.qdd.bak`; solo agrega su entrada, y si el archivo no es JSON válido no lo
  toca y lo dice. `activo.json` recuerda `opencode: true`.
- `qdd consola actualizar` y `qdd doctor --fix` reescriben el plugin con la versión de la CLI. `qdd doctor` avisa si
  falta la entrada en `tui.json` o si el archivo es de otra versión. `qdd consola desactivar` quita la entrada y el
  archivo. Los cambios se ven en una sesión nueva de OpenCode.

### 3.12 Son tareas, no frentes: el nombre del panel (qdd 2.7.0)

Decisión del humano (2026-10-30): «no son frentes, son tareas que estamos trabajando». Cambia el nombre en todo lo que
se ve, no el formato del archivo:

- El mod de Claude Code pasa de `qdd-estado@qdd` a **`qdd-tareas@qdd`** (carpeta `cli/pkg/consola/mod/qdd-tareas`) y el
  comando de `/estado` a **`/tareas`** (`/tareas cerrar` lo cierra). `qdd consola activar`, `actualizar` y
  `doctor --fix` desinstalan el `qdd-estado@qdd` que haya y lo dicen; `desactivar` quita los dos.
- La barra dice `◧ 9 tareas · ⏳7 ❌2` (antes «frentes»), y lo mismo `qdd estado statusline`.
- En OpenCode el plugin es `qdd-tareas.tsx`, con id `qdd.tareas`.
- No cambian `.qdd/project/estado.yaml`, sus claves (`frentes:`), el JSON de `qdd consola panel` ni los comandos
  `qdd estado …`: son el formato del blueprint `estado-por-frente`.

### 3.13 Tarea en curso: «running» y pulsos de vida (blueprint `estado-por-tareas` v1.0.0; qdd 2.7.0)

Decisión del humano (2026-10-30; aprobada con «resuelve de forma autónoma todo», como blueprint aparte `estado-por-tareas`): la fila de una tarea dice cuándo se está trabajando y late cuando algo de esa tarea
se ejecuta, de forma sutil y elegante. **La declara el agente**; laten las herramientas del agente, los subagentes y los
comandos externos; se ve como **punto + running**.

- **Dónde vive.** En `.qdd/working/actividad.json`, que es local y está en `.gitignore`. Es actividad, no estado:
  `estado.yaml` no cambia y nada se commitea. Por cada tarea guarda `desde`, `pulso` (la última señal de vida),
  `fuente` (`agente` | `subagente` | `comando`) y `quien` (opcional). Se escribe de forma atómica; si el archivo falta o
  no se puede leer, no hay tareas en curso.
- **Quién la declara.** El agente, al empezar a trabajar una tarea: `qdd estado activa <id>` o, por MCP,
  `qdd_estado accion=activar frente=<id>` (`accion=terminar` la apaga, `accion=pulso` late). La tarea tiene que existir y estar abierta. Puede haber más de una en curso
  (dos sesiones en tareas distintas). `qdd estado activa <id> --fin` la apaga, y también se apaga al cerrarla o
  descartarla (también con la ✕).
- **Pulsos.** `qdd estado pulso [<id>] [--fuente agente|subagente|comando]` actualiza `pulso`. Sin `<id>`, late la
  última tarea que se activó en el proyecto: así el mod no necesita saber cuál es. La envían:
  - el mod de Claude Code en cada `tool.call` (`--fuente agente`) y en cada `agent.spawn` (`--fuente subagente`), con
    un máximo de 1 escritura cada 5 s. En el panel de esa sesión, el destello es inmediato;
  - el plugin de OpenCode, cuando su agente empieza a ejecutar una herramienta (evento `message.part.updated` con
    `state.status = running`; `--fuente subagente` para la herramienta `task`, `agente` para las demás; mismo tope). Es su
    única escritura y no cambia que sea solo para mirar: no tiene botones ni diálogos;
  - cualquier script, test, deploy o worker: `qdd estado pulso <id> --fuente comando`.
- **Cuándo deja de estar en curso.** Si pasan 120 s sin pulso, la tarea deja de verse «running». Su entrada se limpia
  en la próxima escritura. Por eso un agente que se cae no deja una tarea encendida para siempre.
- **Contrato.** `qdd consola panel --json` agrega a cada tarea `actividad: {running, pulso, fuente}`; una tarea sin
  actividad no lleva el campo. Una tarea en curso **siempre se ve**, aunque no esté entre las 4 más nuevas. Los paneles
  siguen sin interpretar reglas: solo dibujan.
- **Cómo se ve.** `● running` en la fila de la tarea (en Claude Code, después de las etapas y antes de la ✕; en OpenCode, entre las etapas y el nombre). El `●` alterna entre tenue y brillante cada 1 s, con
  «running» en gris. Cuando cambia `pulso`, el `●` destella con el color de acento durante 2 tics de 500 ms (entre 0,5 y 1 s). El
  parpadeo es local (`$.clock.every(500)` e `invalidate` en Claude Code, `setInterval` en OpenCode), sin volver a ejecutar qdd, y
  solo corre si hay alguna tarea en curso. La lectura de qdd sigue siendo cada 2 s.
- **Pruebas** (escenarios EPT-C01…C12 del blueprint `estado-por-tareas`). En Go: activar, pulsar con y sin id, vencer a los 120 s, apagar al cerrar o descartar, la tarea en curso
  fuera del tope de 4 y una tarea que no existe (error). En el mod: el `●` alterna, el pulso destella, `tool.call`
  respeta el tope de 5 s y `next(e)` nunca se demora. En OpenCode: la lógica pura de la vista con `node --test`.

### 3.5 Agnóstico al sistema operativo

Todo en Go puro, sin cgo. La terminal se maneja con secuencias ANSI y `golang.org/x/term` (tamaño y modo crudo), que funcionan en Windows 10+, macOS y Linux. No hay dependencia de tmux; el panel dividido lo arma cada terminal.

## 4. Pruebas

- Golden set del blueprint (`conformidad.yaml`): un caso por escenario EPF-C01…C08 y EPF-A01, con su archivo de entrada y su resultado esperado.
- `statusline`: sin archivo → salida vacía y código 0; archivo dañado → salida vacía y código 0.
- `watch`: el render es una función pura del archivo y del ancho (probada sin terminal real).
- `set`: escritura atómica; un `set` inválido no toca el archivo.

## 5. Fuera de alcance

- El tablero compartido (`destino: tablero`).
- Agregar estados de varios proyectos (el formato lo permite; el comando, para otro RFC).
