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
