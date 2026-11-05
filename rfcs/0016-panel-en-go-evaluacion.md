# RFC 0016: ¿El panel en Go? Evaluación por performance

- Estado: adoptado el flujo en el panel actual (2026-11-05, decisión de la persona). Bubble Tea y Wails no se adoptan; su prototipo queda en la rama local `experimental/panel-go`
- Pregunta: pasar el panel de la derecha a Go, con Bubble Tea o con Wails, para mejorar la performance general

## Resumen

El costo del panel no está en quién lo dibuja, sino en lanzar un proceso por lectura aunque nada cambie (MEJ-020).
Con un solo proceso largo, `qdd consola seguir`, y el mismo panel de siempre, el panel gasta **entre 71 % y 82 % menos
CPU** y un cambio llega a la pantalla en **unos 500 ms**, en vez de hasta 2 s más el dibujo. No hace falta reescribirlo.

## Mediciones

Linux, 16 núcleos, Claude Code 2.1.300 en reposo, sin mandarle mensajes. Cinco sesiones a la vez durante 301 s, en un
proyecto y un `QDD_HOME` temporales con una copia del tablero real. El arnés es `scripts/medir-panel.py`.

| Sesión (% de un núcleo) | Corrida con el prototipo | Corrida con el código final | PSS final |
|---|---:|---:|---:|
| Sin panel, con el envoltorio de Node | 2,35 | 2,27 | 185 MB |
| **Panel de hoy** (sondeo, envoltorio) | **6,33** | **5,84** | 232 MB |
| Sin panel, binario directo | 2,24 | 1,04 | 170 MB |
| Panel de hoy, binario directo | 5,71 | 3,02 | 211 MB |
| **Panel con flujo**, binario directo | **3,38** | **1,67** | 235 MB |

Costo del panel, restando la sesión sin panel equivalente:

- Hoy: 3,98 y 3,57 puntos.
- Con flujo: 1,14 y 0,63 puntos, un 71 % y un 82 % menos.
- En memoria el flujo no gana: suma unos 25 MB por su proceso largo.

Las dos corridas no coinciden en lo que cuesta la barra de estado: con binario directo, las sesiones sin panel bajan
de 2,35 a 2,24 en una y de 2,27 a 1,04 en la otra. Aislada, cuesta 30 ms con envoltorio y 6 ms directa, lo que apunta a
la segunda (unos 1,2 puntos cada 2 s), pero no se sabe qué elevó la primera. Hay que medirla aparte antes de cambiarla.
Además, «binario directo» en estas filas también abarata esa barra; el flujo en producción no la cambia.

Los programas aparte, medidos en otra corrida de 180 s, sobre el mismo tablero:

| Programa | CPU | PSS | Procesos | Binario |
|---|---:|---:|---:|---|
| Bubble Tea (`qdd consola tablero`) | 0,28 | 19 MB | 1 | dentro de qdd: +0,6 MB (17,4 → 18,0 MB) |
| Wails, bajo Xvfb sin GPU | 0,19 | 243 MB | 5 (2 de WebKit) | aparte: 8,6 MB, cgo, 144 bibliotecas compartidas |

Latencia: de escribir `estado.yaml` a la línea del flujo, mediana de 58 ms (15 cambios; 50 ms son la pausa que junta las
escrituras); hasta verse en la pantalla de una sesión real, 523 ms en una prueba.

Lo que estas mediciones no dicen:

- Es Linux. En Windows, MEJ-020 midió ~370 ms de CPU por lectura (`cmd`, `node` y `qdd.exe`) y unos 20 % de un núcleo por
  sesión. Se espera más ganancia allá, pero no está medido: el CI solo comprueba que el flujo funcione.
- Es en reposo y con un tablero que no cambia durante la prueba.
- Bubble Tea y Wails no incluyen el mod mínimo que seguiría haciendo falta (ver abajo), y una sesión sin panel es solo
  el piso de ese costo.

### La barra de estado

Tres rondas alternadas de 120 s, con cinco sesiones sin panel, la barra del usuario y el MCP de qdd:

| Ronda | Con envoltorio de Node | Binario directo |
|---|---:|---:|
| 1 | 2,27 | 1,04 |
| 2 | 2,17 | 0,98 |
| 3 | 2,30 | 1,04 |

El envoltorio suma cerca de 1,2 puntos por sesión, más que todo lo que cuesta el panel con flujo. La medición no separa
la barra del MCP: las dos cambian juntas al poner el binario directo en el `PATH`.

## Lo que un programa aparte no puede hacer

El panel de la derecha es una superficie de Claude Code: un programa aparte no puede acoplarse ahí. Queda en otra
ventana o en un lado de tmux. Y estas capacidades solo existen dentro de un mod:

- `$.prompt.submit`: despertar a la IA cuando termina un pedido, las instrucciones del teléfono y el modo goal.
- `$.session.*` y `$.agent.list`: saber qué sesión confirmó un pedido y si la IA está trabajando.
- Los hooks de las preguntas y los permisos que van al teléfono, y el contexto de cada mensaje.

Reemplazar el mod exigiría igual un mod mínimo con estas piezas, además de portar ~2.100 líneas y sus 112 pruebas.

## Opciones

| | Flujo en el mod | Bubble Tea | Wails |
|---|---|---|---|
| Sigue en el panel de la derecha | Sí | No: otra ventana o tmux | No: otra ventana |
| CPU frente a hoy | −71 % a −82 % del costo del panel | Más baja, sin contar el mod mínimo | Más baja, sin contar el mod mínimo |
| Memoria | +25 MB | Reemplaza ~50 MB del panel por 19 MB | +243 MB |
| Distribución | Igual que hoy | Dentro de `qdd`, compila cruzado | Aparte, cgo y WebView por sistema |
| Esfuerzo | Hecho en el prototipo | Portar el panel completo | Portar el panel completo y su empaquetado |
| Sirve fuera de Claude Code | No | Sí | Sí |

Sobre «plugin oficial fácil de activar»: ya lo es. `qdd-tareas` viaja dentro del binario y se instala con
`qdd consola activar`; `qdd doctor --fix` lo repara si queda desalineado.

## Decisión

1. **Se adopta el flujo y el binario directo en el mod.** Un qdd sin `consola seguir` sigue leyendo como antes.
2. **Sigue lo que queda en los hijos**, con foco en Windows: la barra de estado (1.800 procesos por hora por sesión), los
   procesos de `materializar` (MEJ-017), `herramienta listar` cada 10 s (MEJ-020) y la salida en vivo (MEJ-014).
3. **Bubble Tea y Wails no se adoptan.** Bubble Tea podría volver como vista para tmux o SSH si hace falta; Wails gasta
   243 MB para dibujar una lista y rompe el binario único.

## Lo que se agregó

- `qdd consola seguir` (`cli/pkg/consola/seguir.go`, oculto): emite la misma línea JSON que `consola panel --json`, solo
  cuando cambia. Vigila las carpetas con fsnotify y lee cada 10 s por lo que depende de la hora.
- **Latido:** si no hay cambios escribe una línea vacía; si nadie lee, falla y el proceso termina. `os.Getppid` no sirve en
  Windows. Un binario nuevo también lo termina, y el panel lo reabre.
- `qdd consola binario` (oculto): la ruta real del binario, para no pasar por el envoltorio de Node ni por `cmd`.
- El mod abre el flujo solo en sesiones con terminal; si se corta, vuelve a sondear y lo reabre. Una lectura lenta no pisa
  una línea más nueva.
- CI: un trabajo en Windows prueba el flujo con procesos reales (`.github/workflows/quality.yml`).

Probado en una sesión real de Claude Code en Linux: el cambio llega a la pantalla, una actualización reabre el flujo en
4,6 s y matar la sesión no deja procesos. En esa prueba el proceso terminó en 0,2 s, probablemente por la señal de la
terminal: el camino del latido lo cubre la prueba de procesos reales, no esa sesión.
