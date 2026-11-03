# RFC 0014: El panel de QDD en Antigravity (y VS Code)

- Estado: propuesto (2026-11-03), objetivo «El panel de QDD en Antigravity» del estado de qdd
- Versión: preview en la 2.8.0-rc que salga con la fase A; estable con la siguiente versión que cumpla VER-05
- Nombres: extensión **`qdd.qdd-panel`** (una vista en la barra de actividad), módulo **`pkg/consola/mod/vscode`**

## Problema

Quien trabaja en Antigravity no ve el panel de QDD: el tablero de tareas, los pedidos que esperan su «s  Sí» y las
fichas por aprobar viven en Claude Code (`qdd-tareas@qdd`), en OpenCode (solo lectura) y en el teléfono (RFC 0013).
En Antigravity tiene que ir a otra ventana o preguntarle a la IA.

**Criterio del objetivo:** en Antigravity la persona ve el tablero, confirma pedidos con «s  Sí» y aprueba fichas
igual que en Claude Code, sin lógica repetida: todo sale de `qdd consola panel --json`.

## Decisiones

1. **Una extensión de VS Code.** Antigravity es un fork de VS Code y carga sus extensiones; la misma extensión sirve
   en VS Code y Cursor. Es una `WebviewView` en la barra de actividad («QDD»).
2. **La extensión dibuja; qdd decide (RFC 0006 §3.7).** Lee `qdd consola panel --json` cada 2 s solo mientras la
   vista está visible, y redibuja si cambió. Para actuar llama `qdd consola accion <id>` y, después de «s  Sí»,
   `qdd consola accion <id> --confirmado --sesion antigravity`: qdd vuelve a calcular la acción, la autoriza y la lanza
   aparte (FND-089), igual que en Claude Code. La extensión no conoce reglas de pedidos, fichas ni PRs.
3. **Mismo contrato que los otros paneles.** Lo que no sepa dibujar lo ignora (campos nuevos del JSON); un qdd más
   viejo que no trae un campo no la rompe. Los PRs opcionales (rc.22) se ven igual: no abren la vista.
4. **Sin shell.** La extensión ejecuta el binario `qdd` con `execFile` y argumentos fijos; el webview solo manda ids
   por `postMessage`. CSP estricta: sin contenido remoto, sin `eval`, estilos y script del propio paquete.
5. **Viaja dentro del binario**, como el plugin de OpenCode: `qdd consola activar` la instala si encuentra
   Antigravity o VS Code, `qdd consola actualizar` la deja en la versión de la CLI y `qdd doctor` la diagnostica y la
   repara (principio: todo diagnosticable y reparable).
6. **Doctor nunca ejecuta el IDE.** `antigravity --version` abre la aplicación (visto el 2026-11-03): la instalación y el
   diagnóstico leen y escriben la carpeta de extensiones (`~/.antigravity-ide/extensions`, `~/.vscode/extensions`) y su
   `extensions.json`, con escritura atómica y sin tocar las demás entradas.
7. **El proyecto es la carpeta del workspace** que tenga `.qdd/`; con varias carpetas, la primera que lo tenga (y un
   selector si hay más de una).
8. **Sin relay.** Es local: lee el qdd de esta máquina. El cliente del relay (RFC 0013) es para otro PC u otro lugar.

## Lo que se ve

- Cabecera: «Tareas · HH:MM», la versión de qdd atenuada y el resumen en una línea (como la barra).
- **▶ Para ti** primero: pedidos con su tecla, motivo, comando y carpeta; «s  Sí» / «n  No» en dos pasos; los armados
  con «◷ espera …», los hechos en gris con su salida corta; fichas por aprobar con su botón.
- La tabla de tareas con un símbolo por etapa de su ciclo y la tarea en curso «● running».
- Atajos: las mismas teclas del panel de Claude Code cuando la vista tiene el foco.

## Pruebas (pruebas primero)

- `panel.js`: lógica pura (JSON → modelo de vista) con `node --test`, como el panel de OpenCode; las mismas entradas
  de ejemplo que las pruebas del mod de Claude Code.
- `extension.js`: arranque, ciclo de lectura que se detiene con la vista oculta y confirmación en dos pasos, con un
  `vscode` simulado.
- Go: instalar, actualizar y diagnosticar en una carpeta de extensiones temporal (entrada propia en
  `extensions.json`, las ajenas intactas, reparación de una instalación vieja o rota).
- Conformidad: la ficha declara `implementa: consola-claude-code` y deja fuera, con su motivo, lo que es propio de
  Claude Code (la barra de estado, el aviso al chat).

## Fases

| Fase | Entrega | Hecho cuando |
|---|---|---|
| A | Vista de solo lectura: tablero y pedidos | en Antigravity se ve lo mismo que en el panel de Claude Code |
| B | Acciones: «s  Sí» en pedidos y aprobar fichas, con su resultado | un pedido confirmado en Antigravity corre una sola vez y queda en el log |
| C | `qdd consola activar/actualizar` y `qdd doctor` la instalan y reparan | un PC limpio queda con la extensión tras `qdd consola activar` |
| D | Materializar lo que cambia (efectos de RFC 0006) | la vista anima solo lo que cambió, con `qdd consola efectos` |

## Preguntas abiertas

1. ¿Publicarla también en Open VSX para instalarla a mano, o solo vía `qdd consola activar`? (por ahora, solo qdd)
2. ¿La vista abre sola cuando llega un pedido de la IA, como en Claude Code? (propuesta: sí, salvo los PRs opcionales)
