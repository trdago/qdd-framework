# RFC 0017: Un tablero que no se queda atrás

- Estado: en curso. El punto 1 está publicado en la rc.9 (2026-11-05); el punto 2 espera la decisión de la persona sobre el
  aporte al blueprint `estado-por-tareas`, y el 3 depende del 2.
- Origen: MEJ-010, reporte de sfrc. La persona dueña de sfrc pidió: «las tareas y objetivos sí mutan en el tiempo, él tiene
  que darse cuenta y mantenerlos actualizado».

## Resumen

QDD puede detectar por sí solo lo que se deduce de la tabla y de git. No puede saber si un nombre o un criterio siguen siendo
ciertos: eso lo juzga la IA que trabaja en el proyecto, y QDD solo puede avisarle cuándo revisarlo. Propongo tres cambios,
en este orden: avisar cuando se retira un pedido, detectar tareas con commits nuevos y pedir revisión cuando cambia una ficha.
No propongo un cruce automático de requisitos contra tareas, porque no hay de dónde sacarlo sin adivinar.

## Lo que ya está hecho

| Parte del reporte | Estado |
|---|---|
| Tarea con todo listo que sigue abierta; tarea cerrada con ⏳ | `qdd estado check` avisa sin bloquear (FND-151, rc.8) |
| Un ⏳ de una tarea cerrada contaba en `esperas` | Corregido (FND-151, rc.8) |
| Un pedido retirado por el trabajo de otra sesión | Corregido al heredar la tarea (FND-153, rc.8) |
| La sesión no se enteraba de que se retiró su pedido | `qdd consola ciclo --contexto --sesion` se lo dice una vez (punto 1, rc.9) |

## Lo que falta

### 1. Avisar cuando se retira un pedido (hecho)

Un pedido retirado solo cambiaba de estado en la cola, y la sesión que lo pidió no se enteraba si nadie revisaba.

El pedido anota la sesión que lo pidió, con el mismo id de FND-153. `qdd consola ciclo --contexto --sesion` le dice a esa
sesión, una vez, qué pedido se retiró y por qué tarea. Si el pedido no tiene sesión, no se avisa a nadie en particular y
sigue visible en la cola.

### 2. Tareas con commits nuevos

Es la idea 1 del reporte, y hoy no se puede hacer por dos motivos medidos en el código:

- Una celda no guarda cuándo se escribió, y `estado.yaml` no siempre está en git: en qdd no se versiona; en sfrc,
  notificador_de_ofertas y llmlocal sí.
- Una tarea no dice qué rutas toca.

**Propuesta, con campos opcionales:**

- `qdd estado set` anota `tocada_el` en la tarea.
- Una tarea puede declarar `modulo: <id de ficha>`, y sus rutas son las de esa ficha. Sin módulo, QDD no opina.
- `qdd estado check` avisa, sin bloquear: «tarea X: N commits en sus rutas desde su última celda».

**Costo:** corre `git log` por tarea con módulo, solo en `check` y nunca en el panel ni en la barra. Si pasa de 200 ms en
total, se limita a las tareas abiertas del objetivo en foco. Medido en Linux, con un `git log` por módulo y una ventana de
7 días (2026-11-05): 40 ms en sfrc (18 módulos, 43 commits) y 57 ms en qdd (7 módulos de una copia atrasada, 625 commits).
Falta medirlo en Windows.

**Falsos positivos:** un commit que no cambia lo que falta también avisa. Por eso no bloquea.

### 3. Pedir revisión cuando cambia una ficha

Es la idea 3 del reporte. QDD no reescribe nombres ni criterios: no puede saber si «autoregistro» sigue siendo cierto.

**Propuesta:** cuando las `decisiones` o el `objetivo_operativo` de una ficha cambian después de la última `tocada_el` de una
tarea con ese `modulo`, `qdd_project_context` le pide a la IA revisar el nombre y el criterio de esa tarea, y formularlos
según lo que falta. Depende del punto 2.

No cubre lo que cambia sin tocar una ficha, como una decisión que solo quedó en el chat.

### 4. Requisitos sin tarea: no se automatiza

Es la idea 4 del reporte. En sfrc faltaban tareas para indicadores, fichas técnicas y borrado por conservación. Una ficha
tiene invariantes y un criterio de éxito, pero no una lista de requisitos que se pueda contar, y decidir que una tarea cubre
un requisito es juzgar un texto. Un chequeo así daría una respuesta inventada, contra el principio de que cada función de QDD
sea determinista.

**Alternativa:** una línea en la forma de trabajo de `qdd_project_context`, para que la IA compare la especificación con las
tareas cuando la lea. No promete detectar nada.

## Decisión que falta

¿Vale la pena el punto 2? Agrega dos campos al blueprint `estado-por-tareas`, y eso va por su propio aporte, con la
certificación ciega. Recomiendo hacer primero el punto 1, que no toca el blueprint, y medir cuántos avisos reales da el
punto 2 en una copia de sfrc antes de proponer el aporte.
