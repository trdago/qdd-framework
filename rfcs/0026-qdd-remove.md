# RFC 0026: `qdd remove`, quitar QDD de la máquina

**Estado:** propuesto (2026-11-26). El modo goal eligió el alcance (la opción recomendada); falta que la persona lo
confirme. El código va después de la 2.10.0: hoy hay congelamiento y solo entran arreglos.

**Origen:** MEJ-031. La persona lo propuso el 2026-11-06 desde QDD Móvil: «quizás deberíamos tener un qdd remove que
ayude de forma determinista a limpiar el ambiente».

## Resumen

`qdd remove` quitará de la máquina todo lo que QDD instaló y se puede volver a instalar, y dirá pieza por pieza qué
quitó. Los datos de la persona solo los listará. `qdd remove --datos` los borrará después de que la persona los
confirme. Nada dentro de un proyecto cambiará. Hoy el comando no existe.

## Problema

QDD deja fuera de los proyectos al menos 23 piezas. Algunas se quitan con un comando propio (`qdd consola desactivar`,
`qdd remoto desactivar`, `qdd baul quitar`), y otras con ninguno. `npm uninstall -g` deja varias; en Windows, por
ejemplo, quedan los binarios por versión de `%LOCALAPPDATA%\qdd`. Nadie puede saber qué quedó en la máquina ni dejarla
limpia de forma repetible.

Al hacer el inventario apareció además FND-332: `qdd consola desactivar` borraba toda la carpeta `<QDD_HOME>/claude`.
Ahí, además del panel, están las colas de cada proyecto: pedidos, preguntas, goal, turnos y teléfono. Se arregló en la
rc.145.

## Inventario

`H` es `<QDD_HOME>` (por defecto `~/.local/share/qdd`). `C` es la carpeta de Claude Code (`~/.claude`).

**Instalado: se vuelve a instalar sin perder nada.**

| Pieza | Dónde | Hoy se quita con |
|---|---|---|
| Panel de Claude Code | `H/claude/{plugins,.claude-plugin,activo.json}` y su registro en `C/plugins` | `qdd consola desactivar` |
| Barra | `statusLine` de `C/settings.json` (la anterior queda en `activo.json`) | `qdd consola desactivar` |
| OpenCode | `H/opencode/` y una entrada en `tui.json` | `qdd consola desactivar` |
| Extensiones de VS Code y Antigravity | `~/.vscode/extensions/qdd.qdd-panel-*`, `~/.antigravity-ide/extensions/…` | `qdd consola desactivar` |
| Servicio de qdd remoto | unidad de systemd, agente de launchd o tarea de Windows, y `H/remoto/qdd-remoto.xml` | `qdd remoto desactivar` |
| Binarios por versión (Windows) y `qddw.exe` | `%LOCALAPPDATA%\qdd\bin\` | nada |
| Respaldos | `C/settings.json.qdd.bak`, `tui.json.qdd.bak` | nada |
| Caché | `$TMPDIR/qdd-pycache` | nada |
| El paquete de npm | carpeta global de npm | `npm uninstall -g qdd-framework` |

**Datos de la persona: no se recuperan.**

| Dato | Dónde | Hoy se quita con |
|---|---|---|
| Colas por proyecto | `H/claude/<carpeta>/<proyecto>-<hash>.json`: pedidos, preguntas, goal, turnos, teléfono, sesiones, verificaciones y mejoras | nada |
| Baúl | entradas `qdd-baul` del llavero o `qdd-baul-<nombre>` de 1Password, `H/baul/` | `qdd baul quitar <nombre>`, de a una |
| Credenciales del teléfono | `H/remoto/credenciales.json` | `qdd remoto desactivar`. Después hay que emparejar de nuevo |
| Relay y registros de qdd remoto | `H/remoto/{relay.json,log.jsonl,…}` | nada |
| Proyectos de la máquina | `H/proyectos.json` | `qdd doctor` quita solo los que ya no existen |
| Blueprints | `H/blueprints/`; en `aportes/` puede haber cambios sin subir | nada |
| Historiales | `H/{puertas,uso,rendimiento,reportes,incidentes}/` | nada |

**Dentro de los proyectos: no es de `qdd remove`.** Son `.qdd/`, `.mcp.json`, `opencode.json`, los bloques de reglas,
los ganchos de git y los respaldos `.qdd-respaldo-*` de las carpetas padre. Hoy `qdd undo init` quita solo `.qdd/`; lo
demás queda en cada proyecto.

## Diseño

1. **Primero el plan.** `pkg/quitar` arma la lista de piezas antes de tocar nada. De cada una sabe dónde está, quién la
   escribió, si es dato de la persona y cómo se quita.
2. **Cada pieza se quita con lo que ya existe.** Se usan `consola.Desactivar`, `Quitar` de qdd remoto y el borrado del
   baúl. Lo que hoy no tiene cómo quitarse recibe su función en el paquete que lo escribe, no en `pkg/quitar`.
3. **`qdd remove` sin banderas:** quita lo instalado y lista los datos con su tamaño, sin borrarlos. Al final dice el
   comando de npm, porque un programa no puede borrarse mientras corre.
4. **`qdd remove --datos`:** muestra la lista de datos y borra solo lo que la persona confirma, uno por uno o todos
   juntos.
5. **`qdd doctor` dice qué queda** con la misma lista. Hoy no revisa los binarios de Windows, los `.qdd.bak`, los
   blueprints ni el llavero completo.
6. **Lo corre la persona.** Una IA no puede correrlo, igual que no puede encender el modo goal; desde una IA queda como
   pedido del panel.

## Lo que no se puede romper

- Sin `--datos`, no se borra nada que no se pueda volver a instalar.
- Un dato de la persona se borra solo con su confirmación en ese momento.
- Ningún archivo de un proyecto cambia.
- Correrlo dos veces deja lo mismo, y la segunda vez no falla.
- No se toca un servicio de qdd remoto de otro `QDD_HOME` (FND-136).
- No se toca nada que QDD no escribió, aunque se parezca. Por ejemplo, otra unidad de systemd con un nombre parecido.
- Las pruebas usan HOME, QDD_HOME, llavero y Claude Code falsos. Nunca la máquina real.

## Alternativas

- **Que `qdd remove` borre también los datos.** Se descartó: las colas, el baúl y las credenciales del teléfono no se
  recuperan.
- **Solo documentar los comandos que ya existen.** Se descartó: varias piezas no tienen cómo quitarse y nadie sabría qué
  quedó.

## Lo que no entra

- Quitar QDD de los proyectos. Eso es de `qdd undo init`, que hoy quita solo `.qdd/`.
- Borrar cuentas o datos en el relay de QDD Móvil. Eso necesita un acuerdo con qdd-movile.

## Fases

1. El plan y `qdd doctor` dicen qué queda, sin borrar nada.
2. `qdd remove` quita lo instalado.
3. `qdd remove --datos` con confirmación.

Cada fase entra con sus pruebas. La ficha `quitar-qdd` se crea con la fase 1, cuando existan sus rutas.
