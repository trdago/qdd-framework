# RFC 0011: Herramientas del proyecto en el panel

- Estado: aceptado (decisiones del usuario, 2026-10-31)
- Versión: qdd 2.9.0 (después de 2.8.0)

## Problema

1. Muchos proyectos traen ejecutables para trabajar con ellos: un bot trae uno que simula lo que manda un usuario por
   WebSocket, otro que muestra lo que pasa tras bambalinas, y hay migradores, sembradores de datos o colas. Hoy cada uno
   se usa a mano en otra terminal: hay que acordarse de que existe, de sus flags y de qué variables necesita.
2. La IA puede leer la ayuda de un ejecutable y entender qué pide, pero no hay dónde dejar ese entendimiento para que la
   persona lo use con un clic, siempre igual y sin gastar tokens cada vez.

## Decisiones del usuario

- **Origen:** «+ Agregar herramienta» lista los ejecutables que QDD detecta en el proyecto y deja escribir la ruta de
  cualquier otro.
- **Interfaz:** la arma la IA **una vez**, al agregarla. QDD la valida, la prueba y la guarda como una definición fija.
  Desde ahí el panel la dibuja sin IA, sin tokens y siempre igual.
- **Guardado:** en el proyecto, versionada (`.qdd/herramientas/<id>.yaml`), para que todo el equipo la vea igual. Los
  secretos nunca van ahí: solo el nombre de la variable de entorno que los trae.
- **Paneles:** Claude Code primero. OpenCode después, con lo que su panel permita.

## Diseño

### La definición (`.qdd/herramientas/<id>.yaml`)

```yaml
id: simulador-ws
nombre: Simulador WS
descripcion: Simula lo que manda un usuario al bot por WebSocket.
comando: [./bin/simulador]        # argv base, relativo a la raíz del proyecto; sin shell
entorno: [BOT_URL, BOT_TOKEN]     # nombres de variables que necesita; nunca sus valores
acciones:
  - id: enviar
    nombre: Enviar como usuario
    args: [send, "{{mensaje}}", --chat, "{{chat}}"]
    campos:
      - {id: mensaje, tipo: texto, requerido: true}
      - {id: chat, tipo: opcion, opciones: [soporte, ventas], por_defecto: soporte}
    modo: una-vez                 # corre, espera y muestra su salida
  - id: tras-bambalinas
    nombre: Ver lo que pasa tras bambalinas
    args: [logs, --follow]
    modo: en-vivo                 # muestra la salida mientras corre, hasta ■ detener
  - id: reiniciar
    nombre: Reiniciar el estado del bot
    args: [reset]
    modo: una-vez
    confirmar: true               # pide «¿seguro?» antes de correr
prueba: {args: [--help]}          # la prueba de humo: debe salir con 0
```

- Cada `{{campo}}` es **un argumento entero**: lo que la persona escribe nunca se parte ni pasa por una shell, así que no
  puede inyectar otro comando.
- Un grupo (`[--chat, "{{chat}}"]`) va solo si todos sus campos tienen valor. Un `si-no` va siempre en un grupo y solo
  decide si el grupo va (`[--seco, "{{seco}}"]` pasa `--seco` cuando dice «si»).
- Tipos de campo: `texto`, `numero`, `opcion` (con `opciones`) y `si-no`.
- `confirmar: true` para lo que borra o cambia datos.

### QDD (CLI y MCP)

- `qdd herramienta candidatos`: los ejecutables del proyecto. Mira `bin/` y `scripts/`, `cmd/*` de Go, `target/release`
  de Rust, los scripts de `package.json` y los objetivos del `Makefile`.
- `qdd herramienta ayuda <ruta>`: corre `<ruta> --help` (y `-h`, `help`) con un tiempo límite y sin entrada, para que
  la IA lea qué pide. No puede quitarle la red a un ejecutable: es el que la persona eligió.
- `qdd herramienta guardar <archivo>`: valida la definición y corre su prueba de humo antes de guardarla.
  - La validación revisa que el comando exista y esté dentro del proyecto o en la ruta que la persona eligió, que cada
    `{{campo}}` esté declarado, que no haya valores de secretos y que los modos sean válidos.
- `qdd herramienta listar [--json] | quitar <id> --si`. `listar` dice lo que le falta a cada una en esta máquina.
- `qdd herramienta correr <id> <accion> [campo=valor…] [--si]`: lo que usa el panel; una acción con `confirmar` pide `--si`. Corre sin shell, en la raíz del proyecto,
  con el entorno de la máquina. Si falta una variable de `entorno`, dice cuál y no corre.
- **Sin tool MCP** (cambio al implementar, 2026-10-31): `tools/list` estaba en 15.997 de 16.000 caracteres. La IA usa
  la CLI (`qdd herramienta ayuda` y `guardar`), que el panel le nombra al pedirle que la configure. Así no cuesta
  tokens fijos en cada sesión, solo cuando se agrega una.
- `qdd doctor` diagnostica cada definición: no se lee o no es válida, el ejecutable no existe, falta una variable o la
  prueba de humo falla. `--fix` no las repara: un ejecutable ausente puede no estar compilado o ya no existir, y eso lo
  decide la persona. El detalle dice el comando (compilarlo, o `qdd herramienta quitar <id> --si`).
- `qdd audit` valida `.qdd/herramientas/*.yaml` (`HER-01-DEFINICION`): una definición inválida o con un secreto no pasa
  el gate. No corre nada ni exige el ejecutable, porque el CI puede no compilarlo.

### El panel (Claude Code)

- **⚙ Config** va a la derecha de la primera fila del panel, bajo la X; la barra de la X la dibuja el motor.
- **Config** lista las herramientas, cada una con **Quitar** (con confirmación), y **+ Agregar herramienta**. Agregar
  guía en pasos que se ven en el panel:
  1. Elegir un candidato de la lista o escribir una ruta.
  2. El panel le pide a la IA, en la sesión, que la configure con `qdd herramienta ayuda` y `guardar`. La IA lee su ayuda y propone las
     acciones con sus campos.
  3. La persona ve las acciones propuestas y las acepta o pide cambios.
  4. QDD la valida y corre la prueba de humo (✅ o qué falló).
  5. Queda guardada y aparece como pestaña.
- **Cada herramienta es su propio panel** (`qdd-herr-<id>`). Con más de un panel abierto, el motor dibuja las pestañas
  con su nombre. La pestaña muestra sus acciones como botones. Cada acción abre sus campos (Input y Select) y **▶ Correr**.
  La salida va debajo: completa en `una-vez`, en vivo en `en-vivo`, con **■ Detener**.
- Correr es siempre un clic de la persona; la IA nunca corre una herramienta sola desde el panel.

### Límites (del motor)

- El motor le da a un proceso su entrada estándar una sola vez y la cierra: no hay conversación continua por stdin. Una
  herramienta interactiva se usa como acciones separadas: enviar un mensaje es una ejecución, y ver las respuestas es otra
  acción en vivo. Si un ejecutable solo funciona en modo interactivo, la IA lo dice al agregarlo y no se guarda.
- La salida se muestra hasta las últimas 2.000 líneas y no se guarda en disco.

## Escenarios (cada uno, una prueba)

- **HER-C01** Candidatos del proyecto: cuando se pide agregar → se listan los ejecutables detectados y se puede escribir
  una ruta.
- **HER-C02** Sin definición válida no hay pestaña: cuando la definición cita un campo no declarado, trae un secreto o su
  prueba de humo falla → no se guarda y se dice por qué.
- **HER-C03** Un campo es un argumento: cuando la persona escribe `hola; rm -rf /` → el ejecutable recibe exactamente ese
  texto como un argumento y no corre nada más.
- **HER-C04** Falta una variable: cuando falta una variable de `entorno` → no corre y dice cuál falta, sin mostrar
  ningún valor.
- **HER-C05** Una vez y en vivo: una acción `una-vez` muestra su salida al terminar; una `en-vivo` la muestra mientras
  corre y ■ la detiene.
- **HER-C06** Confirmar lo peligroso: una acción con `confirmar` no corre sin el «¿seguro?» de la persona.
- **HER-C07** Pestañas: cada herramienta guardada se abre como su propio panel, con su nombre como pestaña.
- **HER-C08** Quitar: quitar una herramienta pide confirmación, borra su definición y cierra su pestaña.
- **HER-C09** Sin IA al usarla: abrir una herramienta y correr una acción no le pide nada a la IA.

## Cómo se prueba

- `go test ./...` (pkg/herramientas: candidatos, validación, argumentos, entorno; cmd: CLI y tool MCP).
- `claude plugin test` del mod: Config, agregar en pasos, pestañas, correr, detener y quitar.
- `qdd blueprint conformance`, cuando la consola lo incluya en su blueprint.
- Sin `qdd_herramienta` en la MCP, `TestMCP_DescripcionesDentroDelPresupuesto` sigue igual.
