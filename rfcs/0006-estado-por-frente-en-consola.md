# RFC 0006: Estado por frente en la consola — `qdd estado`

**Status:** Accepted (aprobado por el humano el 2026-10-29; implementado en la rama feat/estado-por-frente)
**Created:** 2026-10-29
**Relacionado:** blueprint `estado-por-frente` (v1.0.0, opción `destino: archivo`), blueprint `consola-claude-code` (aporte v1.1.0), [RFC 0004](0004-blueprints-centrales.md).

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

### 3.4 Barra de estado (aporte `consola-claude-code` v1.1.0)

La invariante actual «La barra solo usa el JSON de la consola: sin red, sin archivos ni llamadas extra» impide leer `estado.yaml`. Se propone (cambio **minor**, porque agrega y no quita) una opción nueva, `segmento_estado`:

- La barra puede mostrar un único segmento extra que entrega `qdd estado statusline`.
- Ese segmento lee solo archivos locales del proyecto. No usa red, tiene un tope de 100 ms y, si falla o tarda, no muestra nada.
- Los demás segmentos siguen usando solo el JSON de la consola.

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
