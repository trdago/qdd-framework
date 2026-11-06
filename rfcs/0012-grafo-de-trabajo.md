# RFC 0012: Grafo de trabajo

- Estado: aceptado (visión y decisiones del usuario, 2026-11-01)
- Versión: qdd 2.10.0 o posterior. Parte de la fase 1 (objetivos y árbol, qdd 2.8.0)

## Problema

1. El tablero de `estado-por-tareas` 2.0.0 ya es un árbol: objetivo → tareas → dependientes. Pero solo conoce dos tipos
   de nodo (objetivo y tarea) y una arista (depende de). Un pipeline de entrenamiento tiene puertas («¿F1 ≥ 0.85?») que
   eligen el camino. Un sistema multiagente tiene agentes que derivan a otros según la consulta. Ninguno de los dos se
   puede dibujar ni validar hoy.
2. Para cambiar el plan, la persona lo escribe en el chat, la IA lo interpreta y vuelve a anotar el tablero. Eso gasta
   tokens y se presta a malentendidos. Mover una tarea, insertar un nivel o cambiar el foco son cambios con estructura:
   pueden viajar como datos, no como prosa.

## Lo que el usuario ya decidió

- **Visión:** un grafo general y estándar que sirve para tareas, entrenamiento, multiagente y decisiones. La persona
  agrega, quita o modifica niveles, y la IA recibe cada edición como un cambio estructurado sobre el trabajo en curso:
  una interfaz eficiente para comunicar ideas como grafo.
- **Fases:**
  - Fase 1 (hecha, PR #50): objetivos y árbol en `estado-por-tareas` 2.0.0, con el modelo interno pensado como grafo.
  - En paralelo: una vista previa HTML editable, con datos de ejemplo de qdd, de un clasificador y de un soporte
    multiagente, para trabajar la idea.
  - Fase 2 (este RFC): un RFC y su propio blueprint, «grafo de trabajo».
- **Ya vale para el árbol** (fase 1):
  - El foco va primero.
  - Lo que no tiene objetivo va al final, bajo «sin objetivo».
  - Con dos dependencias, una tarea cuelga de la primera y dice «también espera a Y».
  - Una dependencia de otro objetivo dice «espera a X (otro objetivo)».
  - Pinchar un nombre pregunta «¿Cómo vamos con…?».

## Propuesta

### El modelo: nodos y aristas con tipo

```yaml
formato: 3                       # solo si usa tipos o aristas que formato 2 no tiene
etapas: [Diseño, Prompt, Pruebas, Producción]
nodos:
  - {id: o1, tipo: objetivo, nombre: Soporte resuelve el 70 % sin humano, foco: true,
     criterio: 70 % de 500 conversaciones reales resueltas sin escalar}
  - {id: orq, tipo: agente, nombre: orquestador, objetivo: o1, celdas: {…}}
  - {id: tipo, tipo: decision, nombre: ¿Tipo de consulta?, objetivo: o1, depende_de: [orq],
     ramas: [{si: factura, hacia: facturas}, {si: técnica, hacia: tecnico}, {si: otra, hacia: humano}],
     elegida: null}
  - {id: facturas, tipo: agente, nombre: agente-facturas, objetivo: o1, celdas: {…}}
```

- **Tipos de nodo:** `objetivo`, `tarea`, `agente` y `decision`. Los tres primeros llevan celdas por etapa, igual que
  una tarea hoy. Una `decision` no lleva celdas: lleva ramas.
- **Aristas:**
  - `objetivo` (a qué sirve);
  - `depende_de` (no termina antes);
  - `ramas` (una decisión elige uno de sus caminos; `elegida` dice cuál, con evidencia).
- **Formato 2 se lee tal cual:** cada frente es un nodo `tarea`, y cada objetivo, un nodo `objetivo`. El archivo pasa
  a `formato: 3` solo cuando usa `agente` o `decision`, igual que el paso de 1 a 2.

### Las reglas (`qdd estado check`)

- Todo lo que vale hoy sigue valiendo: sin ciclos en `depende_de`, y nada termina antes que aquello de lo que depende.
- Una `decision` tiene al menos dos ramas. Cada rama apunta a un nodo que existe en su mismo objetivo.
- Mientras una decisión no tenga `elegida`, sus ramas quedan «espera a la decisión». Al elegir una, las ramas no
  elegidas quedan descartadas, con el motivo de la decisión. Nunca desaparecen en silencio.
- Un nodo no puede ser rama de dos decisiones: sería un camino ambiguo.

### Las ediciones estructuradas

Cada cambio que la persona hace en el panel es una edición con nombre, como en la vista previa:

| Edición | Qué es | Ejemplo de lo que recibe la IA |
| --- | --- | --- |
| `+` agrega | un nodo nuevo bajo otro | `+ tarea «destilar» en o2, depende de fine-tune` |
| `−` quita | un nodo, con motivo | `− tarea «crm»: ya no se usa el CRM` |
| `~` modifica | nombre, criterio o tipo | `~ o1 criterio: «F1 ≥ 0.85» → «F1 ≥ 0.90»` |
| `⇡` inserta un nivel | un nodo entre otro y sus hijos | `⇡ decisión «¿idioma?» entre orq y tipo` |
| `↷` mueve | cambia de padre o de objetivo | `↷ «limpiar» de sin objetivo → o1` |
| `★` foco | cambia el objetivo foco | `★ o2` |
| `?` pregunta | cómo vamos | `? fine-tune` |

- qdd valida cada edición con las reglas de arriba **antes** de que llegue a la IA. Una edición que deja un ciclo o una
  rama colgando se rechaza en el panel, con el porqué, y la IA no la ve.
- Las ediciones viajan en el mismo canal de la sincronía panel–chat (`qdd consola ciclo --contexto`). Llegan solo
  cuando hay algo nuevo, en una línea compacta por edición (≤ 160 caracteres), y la IA responde qué cambia en el plan.
- Las ediciones quedan en un registro (`.qdd/project/estado-ediciones.jsonl`, local, no versionado) para que `qdd
  doctor` pueda decir qué se aplicó y repararlo si el archivo quedó a medias.

### El blueprint «grafo-de-trabajo»

- Su receta es agnóstica del lenguaje y completa: el modelo, las reglas, las ediciones y sus escenarios.
- `estado-por-tareas` y `consola-claude-code` lo usan:
  - `estado-por-tareas` aporta las etapas, las celdas y la actividad;
  - `consola-claude-code` dibuja el árbol y emite las ediciones.

### Presupuesto

- **Tokens:** una edición cuesta lo que su línea, no un párrafo. Nada se repite si no cambió, igual que en la sincronía.
- **Rendimiento:** validar un grafo de 200 nodos tarda < 20 ms (las reglas son lineales en nodos + aristas).
- **UX:** cada edición se puede deshacer en el panel hasta que la IA la toma.

## Decisiones de diseño del usuario (2026-11-01)

1. **Una edición del panel se aplica de inmediato.** qdd la valida y la escribe en el archivo. La IA solo recibe la línea
   del cambio y responde qué cambia en el plan.
2. **Los tipos de nodo son fijos:** objetivo, tarea, agente y decisión. Las reglas son deterministas y el dibujo es el
   mismo en todos los proyectos.
3. **`estado-por-tareas` usa `grafo-de-trabajo`:** son dos blueprints, uno sobre otro.
   - `grafo-de-trabajo` trae el modelo, las reglas y las ediciones.
   - `estado-por-tareas` trae las etapas, las celdas y la actividad.
4. **Se edita solo en el panel de Claude Code**, con teclas. No hay una vista HTML aparte.

## Plan

1. La receta `grafo-de-trabajo` (contribute → certify → propose) y la nueva versión de `estado-por-tareas`.
2. qdd: el modelo en `pkg/estado`, las reglas en `check`, las ediciones en `qdd estado editar` y en la tool MCP.
3. El panel: dibujar `agente` y `decision`, y las teclas de edición. Cada fase con su prueba en el gate.

## Ampliación propuesta: flujos (2026-11-05)

Fases 1 a 3 implementadas (ver «Lo que se hizo» al final). Origen, pedido de la persona: «una forma de mostrar grafos en el panel de la derecha […] para
entender flujos completos y explicarle al llm como lo queremos o algún ajuste que no está mirando». La persona eligió que
esto sea parte de este RFC, no una vista aparte, y que los flujos se versionen en el proyecto.

### El problema que agrega

Este RFC cubre el plan de trabajo. Explicar el camino de una solicitud, un pipeline o un proceso tiene el mismo problema:
la persona describe en prosa algo que tiene forma, y la IA lo interpreta. Y cuando la IA explica cómo funciona algo, pasa lo
mismo al revés.

### Un solo grafo

- Es el mismo modelo de nodos y aristas, las mismas ediciones y el mismo canal hacia la IA. Una edición en un flujo se escribe
  y se valida igual que en el plan.
- **Dos tipos de nodo nuevos**, que amplían la decisión 2: `paso`, que se dibuja como caja, y `resultado`, que se dibuja como
  círculo (un inicio, un fin o un resultado). Siguen siendo una lista fija. Llevan nombre y descripción, sin celdas.
- **Un flujo puede nombrar una tarea** del tablero: una caja que muestra su estado. Es lo que se gana al unirlos.
- **Los archivos son distintos:** cada flujo es `.qdd/project/flujos/<id>.yaml`, versionado, porque `estado.yaml` no siempre
  lo está (en este repositorio no). Así se comparte con el equipo y se revisa en un PR.

### La vista

Un flujo hacia abajo, con scroll, en el panel de la derecha:

```text
      ( Llega la solicitud )
                │
      ┌─────────▼─────────┐
    ▶ │ Valida la firma   │
      │ rechaza si vence  │
      └─────────┬─────────┘
          ┌─────┴─────┐
       ( ok )     ( rechazo )

 + agregar  m mover  e editar  x quitar  ? detalle
```

Dos salidas o más van lado a lado si caben en el ancho, y una debajo de otra, con sangría, si no. Un ciclo se dibuja como
«↺ vuelve a X».

### Las ediciones

Se usan las de la tabla de arriba. Sobre un flujo, además:

- `x` une lo de antes con lo de después del nodo que se quita. El flujo no pide motivo, a diferencia de una tarea.
- La persona puede cambiar un nodo de lugar con `m`, ↑↓ y Enter, y deshacer con `u`.

### Pedir más detalle

`?` sobre un nodo le pide a la IA que lo explique. Le llega un mensaje que empieza con 🔎 y nombra el flujo y el nodo. Responde
con `qdd flujo detalle <flujo> <nodo>`: agrega la descripción, o nodos nuevos debajo si el detalle es un subflujo. Gasta tokens
solo porque la persona lo pidió.

### El efecto materializar

Lo nuevo o cambiado aparece con el efecto del panel: qdd calcula los cuadros (`qdd consola materializar`) y el mod los
reproduce. Solo se anima lo que cambió, y si el efecto está desactivado se muestra el texto final.

### Cómo llega

- `qdd flujo crear | ver | detalle` es la entrada de la IA. No lleva una tool MCP propia: `tools/list` está junto a su tope de
  16.000 caracteres.
- El panel recibe los cambios por `qdd consola seguir` (RFC 0016), que vigila esa carpeta, sin procesos por lectura.

### Lo que no se sabe todavía

La primera fase tiene que medirlo:

- **Mouse:** los botones del panel responden al clic, pero no sé si se puede arrastrar. Por eso «mover» va con teclas.
- **Ancho:** cuántas ramas lado a lado caben en 40 a 60 columnas.
- **Rendimiento:** cuánto tarda un flujo de 50 nodos en dibujarse y materializarse, sobre todo en Windows.
- **Teclas:** si elegir un nodo con ↑↓ convive con el scroll del panel.

### Fases

1. **Ver:** la IA crea un flujo y el panel lo dibuja con scroll y materializar. Ya sirve para que la IA explique algo.
2. **Editar:** agregar, mover, editar, quitar y deshacer, con las líneas que recibe la IA.
3. **Detalle:** `?` sobre un nodo y su respuesta.
4. **El plan en el mismo dibujo:** el tablero usa esta vista y las ediciones del resto de este RFC.

### Lo que se hizo (2026-11-06)

Pedido de la persona, sprint 6: «editar, mirar, modificar, agregar, comentar, con el fin de permitir interacción más
detallada con el llm y es una interface de trabajo y aclaración entre llm qdd y el humano».

- **Comentarios, que este RFC no traía.** Cada nodo tiene un hilo, en el YAML versionado. La IA responde con
  `qdd flujo comentar` y la respuesta se ve bajo el nodo. No viajan al teléfono, que todavía no los muestra.
- **Cuándo se entera la IA.** Una edición espera a «Enviar a la IA» o al próximo mensaje del chat: un turno por cambio
  gastaría de más. Un comentario y un detalle la llaman en el acto, porque piden una respuesta. Cada línea llega una vez.
- **Deshacer** vale mientras la IA no la recibe y nadie cambió el flujo después. No pisa lo que hizo la IA.
- **Registro local:** las ediciones sin enviar viven en `.qdd/working/flujos/<id>.jsonl`, sin versionar.
- **Elegir y mover con clics, no con flechas.** El dibujo no sabe qué nodo hay en cada columna, así que bajo el dibujo hay una
  lista de nodos. Mover es un clic en el nodo destino.
- **`d`, no `?`:** una tecla del panel solo puede ser un dígito o una letra minúscula.
- **Probado con clics en un panel real, en Linux.** Un campo no toma las teclas por sí solo (FND-166), y un clic en un botón
  tampoco se las da. No se probó arrastrar ni elegir con flechas, porque no se usan. No se midió en Windows.

Falta: la fase 4, y que `qdd doctor` revise este registro.
