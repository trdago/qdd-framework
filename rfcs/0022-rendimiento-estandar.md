# RFC 0022: rendimiento estándar en cada equipo

**Estado:** aceptado (2026-11-07). Lo pidió la persona: «un test de performance que no gaste mucho token … en cada
sistema operativo, cómo funciona, si deja procesos abiertos, la performance de forma estandarizada en cada equipo».

## Problema

El rendimiento de qdd se medía a mano y solo en Linux (`scripts/medir-panel.py`, RFC 0016). Los problemas llegaban de
las máquinas de las personas: 2.280 procesos por hora en Windows (MEJ-020), procesos huérfanos (MEJ-018) y 919 zombies
en ~4 h (MEJ-042, FND-179). Nadie podía comparar un equipo con otro ni una versión con la anterior.

## Decisión

`qdd rendimiento` mide qdd igual en cada equipo:

- **Sin tokens.** No lanza Claude ni una IA. Mide los procesos de qdd, que son los que corren todo el día. Lo que qdd
  suma al contexto de cada sesión (tools/list del MCP) se cuenta en el mismo proceso, sin lanzar nada.
- **La misma caja en cada equipo.** Un proyecto fijo (12 tareas y un flujo de 25 nodos), un repositorio con un remote
  de GitHub ficticio (para que `consola seguir` revise PRs, el camino que dejaba zombies) y `HOME`, `QDD_HOME` y las
  carpetas XDG y de AppData temporales. No toca los proyectos ni el `QDD_HOME` de la persona.
- **Qué mide:**
  - latencia p95 de lo que el panel llama a cada rato: la barra, el panel, `qdd estado` y el dibujo de un flujo;
  - de `qdd consola seguir`, durante 30 s y cambiando el tablero cada 5 s: CPU, memoria máxima y procesos lanzados
    por hora (sin contar el arranque);
  - zombies y procesos que siguen vivos al cerrarla. Esto es una **falla**, no lentitud. La medición los cierra.
- **Presupuesto por sistema operativo.** Cada medida tiene un tope; Windows multiplica los de latencia, porque lanzar
  un proceso cuesta varias veces más. Lo que no se pudo medir queda «sin medir», nunca en cero.
- **Por versión.** El informe queda en `<QDD_HOME>/rendimiento/<versión>.json`, y `qdd doctor` muestra el último de
  la versión que corre.
- **Salida.** 0 si todo cupo, 1 si dejó procesos y 2 si solo pasó un presupuesto.
- **En el CI**, en Linux, Windows y macOS: la falla bloquea y el exceso avisa. El informe de cada sistema queda como
  artefacto.

## Cómo se lee cada sistema

| | Procesos | Zombies | Memoria máxima |
|---|---|---|---|
| Linux | `/proc`, sin lanzar nada | estado `Z` | `Maxrss` al terminar |
| macOS | `ps -A -o pid,ppid,stat,comm` | estado `Z` | `Maxrss` al terminar (en bytes) |
| Windows | Toolhelp32 | no existen | `PeakWorkingSetSize` mientras corre |

Los procesos se muestrean cada 200 ms. Un hijo que vive menos que eso puede no contarse como lanzado; un zombie sí se
ve, porque queda.

## Primeras mediciones

Linux, 16 núcleos, 2026-11-07, 90 s de `consola seguir`:

| | rc.31 (con FND-179) | Esta versión |
|---|---:|---:|
| Barra p95 | 6,7 ms | 6,3 ms |
| Panel p95 | 8,5 ms | 8,3 ms |
| CPU de `consola seguir` | 0,1 % | 0,2 % |
| Memoria máxima | 22,5 MB | 22–48 MB |
| Zombies | **1 (falla)** | 0 |
| Procesos vivos al cerrar | 0 | 0 |
| tools/list del MCP | — | 15.989 de 16.000 caracteres |

La medición detecta el bug de los zombies de rc.31.

## Pendiente

- **Recalibrar los presupuestos** con los informes de Windows y macOS del CI. Los de hoy salen de Linux, con holgura.
- **El panel dentro de Claude Code** (el JS de `qdd-tareas`) no se mide aquí. Se puede medir sin tokens con Claude Code
  y una clave falsa, como la prueba de instalación limpia, sin mandarle mensajes. Queda para una fase 2.
- **El contexto por mensaje** (lo que el hook suma a cada mensaje) no se mide todavía; solo tools/list.
- Las mediciones de MEJ-020 en Windows (~370 ms por lectura, por `cmd` y `node`) no tienen todavía un informe de
  `qdd rendimiento` que las compare.
