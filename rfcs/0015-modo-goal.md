# RFC 0015: Modo goal por proyecto

- Estado: aceptado (2026-11-04), decisión de la persona
- Versión: preview en 2.9.0-rc.3
- Módulo: `pkg/consola/goal.go`, `cli/cmd/goal.go`, mod `qdd-tareas`

## Problema

Un trabajo largo se detiene cada vez que la IA hace una pregunta, deja un pedido o una ficha, o simplemente se queda
callada. La persona quiere que un proyecto siga hasta terminar mientras no está mirando, y que sepa cuándo se detuvo.

## Decisión

La persona enciende el modo en un proyecto. Mientras está activo, qdd decide de forma determinista, sin llamar a un
modelo:

1. Elige la opción recomendada de cada pregunta.
2. Corre los pedidos de la IA y aprueba las fichas.
3. Con 2 min de silencio en la consola, si quedan tareas abiertas, le recuerda a la IA que siga.
4. Tras 3 recordatorios sin avance real, se detiene y avisa.
5. Sin una sesión abierta, se pausa y avisa. Nunca abre una sesión.

La persona aceptó que aprobar pedidos y fichas entra en el modo: con él activo pueden correr merges, publicaciones y
despliegues que la IA deje como pedido, sin el «s  Sí» de cada uno.

## Límites

- **Lo activa solo la persona**: el botón del panel con confirmación o el teléfono con biometría.
- **Nunca se eligen solos**: las preguntas de diseño de un blueprint, las que no traen opción recomendada y los permisos
  de Claude Code. Dejan el modo en pausa (`espera_persona`) hasta que la persona responda.
- **No se corren solos**: los PRs que qdd ofrece por su cuenta, porque también son de otras sesiones. Un pedido que falló
  vuelve a la IA con su resultado y no se reintenta.
- **Cada elección queda registrada**: las últimas 20, en el panel, en `qdd goal estado` y en el teléfono.

## Cómo reconoce cada cosa

- **Opción recomendada**: la etiqueta termina en `(Recommended)`, `(Recomendado)` o `(Recomendada)`.
- **Pregunta de diseño**: su texto, normalizado, coincide con una pregunta de un blueprint importado en el proyecto.
- **Avance real**: cambió la huella del proyecto, hecha del tablero sin su fecha, el último commit y los cambios de git,
  sin lo que qdd escribe solo (métricas, evidencia, cola de mejoras, `.qdd/working` y `estado.yaml`, que ya cuenta por
  contenido).
- **Silencio**: el mod cuenta desde la última herramienta, turno o tecla. Un turno, una herramienta o un subagente en
  curso son actividad. Si otra terminal del proyecto trabajó en los últimos 2 min, no se le recuerda a la que mira.
- **Sin límite**: la barra leyó en los últimos 10 min que una ventana de uso está al 100 % y aún no se reinicia. No elige
  nada hasta que se reinicie, y sigue solo.

## Dónde vive

El estado va en `<QDD_HOME>/claude/goal/<proyecto>.json`, con el archivo bloqueado: varias terminales no duplican un recordatorio ni una
elección. Doctor revisa que se pueda leer (GOAL-01); `--fix` lo aparta y el modo queda apagado, porque encenderlo es
decisión de la persona. No hay tool MCP: la IA no tiene nada que hacer con el modo.

## Lo que no garantiza

El rechazo a una IA es un resguardo, no una frontera de seguridad. `qdd goal activar` falla con `CLAUDECODE` en el
entorno y el mod niega el comando en Bash, pero una IA en la misma máquina con permisos amplios podría limpiar esa
variable con otro comando. La regla para las IAs lo prohíbe.

Reconocer una pregunta de diseño depende de su texto exacto: si la IA la reformula, ya no coincide. Por eso el modo solo
elige lo que la IA marcó como recomendado, y la regla le pide no responder estas preguntas por su cuenta.

## Teléfono

Contrato en RFC 0013 §«Modo goal en el teléfono».
