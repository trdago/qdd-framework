# RFC 0018: Comparar proyectos en el panel y dejar obsoleto `qdd dashboard`

- Estado: propuesto (2026-11-06).
- Origen: tarea `dash`. La persona aceptó «comparar proyectos lado a lado e historial de auditorías y findings», y luego
  decidió que el panel que sigue es `qdd-tareas` y que `qdd dashboard` se descontinúa, primero marcándolo como obsoleto.

## Resumen

`qdd-tareas` suma una vista «Proyectos»: una fila por proyecto, con las mismas columnas, y el historial del que se elija.
Solo lee archivos locales, y solo cuando la persona abre la vista. No hay servidor ni puerto, así que desaparece el riesgo
que dejó el FND-164.

`qdd dashboard` queda obsoleto: avisa al arrancar, no recibe funciones nuevas y solo se le corrigen fallas. Quitarlo es una
decisión aparte, que se toma cuando el panel cubra lo que hace falta.

## Por qué en el panel

- `qdd dashboard` lee un solo proyecto, el de la carpeta donde se abre. Un panorama central habría exigido hacerlo crecer.
- Solo un mod puede despertar a la IA y recibir las preguntas del teléfono (RFC 0016), así que el panel se mantiene de
  todos modos. Una web sería una segunda superficie por mantener.
- El dashboard atendió a la red sin autenticación hasta la 2.9.10 (FND-164). Cada acción nueva en esa web sería otra
  superficie que proteger.

## La vista «Proyectos»

| Columna | Fuente | Sin fuente |
|---|---|---|
| Proyecto | nombre de la carpeta, del registro `proyectos.json` | `no existe` si la carpeta ya no está |
| Puntaje | último de `metrics/certificate_history.json`, en la escala actual | `sin medir` |
| Hace | fecha de ese mismo registro | `sin medir` |
| Findings abiertos | `pkg/findings`, el lector único | `sin medir` |
| ⏳ y ❌ | `estado.yaml` del proyecto | `sin medir` |

Al elegir una fila, la vista muestra el historial de ese proyecto:

- los puntajes por día, con el último de cada día, como ya hace el dashboard;
- los findings por fecha, separados en abiertos y cerrados.

Medido en este PC, con los 7 proyectos del registro: 5 tienen `estado.yaml`, 6 tienen findings y solo 3 tienen historial
de puntaje. Por eso `sin medir` será frecuente, y es lo correcto: nunca se muestra un cero ni un valor inventado.

Dos puntajes en escalas distintas no se comparan. Un registro de otra escala cuenta como `sin medir` en la tabla y, en el
historial, se marca aparte.

### Contrato

- Un comando nuevo, `qdd consola proyectos --json`, entrega las filas ya calculadas. El panel las dibuja y no decide nada,
  como con `qdd consola panel`.
- Se lee al abrir la vista y con `l ↻`. No usa `consola seguir`, un vigilante de archivos ni un temporizador: con la vista
  cerrada no gasta CPU.
- No escribe en ningún proyecto, no corre auditorías y no usa la red.
- Del proyecto solo muestra el nombre de la carpeta, nunca la ruta completa.
- La primera versión es solo lectura: sin botones que actúen sobre otro proyecto.

### Presupuesto

| Medida | Tope |
|---|---|
| Abrir la vista con 7 proyectos, en el percentil 95 | 200 ms |
| CPU con la vista cerrada | 0 procesos nuevos |
| Tokens enviados a la IA | 0 |
| Ancho de 40 columnas | proyecto, puntaje y ⏳ y ❌; el resto, al elegir la fila |

Leer los archivos de historial tomó 1,9 ms con los 7 proyectos, sin interpretar los findings. Los 200 ms todavía no están
medidos con el comando real.

## La obsolescencia de `qdd dashboard`

1. **Ahora:** `qdd dashboard` avisa al arrancar que está obsoleto y que `qdd-tareas` es el panel que sigue. La
   documentación dice lo mismo. Sigue funcionando igual, y se le corrigen fallas y problemas de seguridad.
2. **Con la vista publicada:** el aviso nombra la vista «Proyectos».
3. **Para quitarlo:** la persona lo decide. Quitar un comando es una versión mayor y exige resolver las preguntas abiertas.

Hasta entonces siguen vigentes los invariantes de su ficha, incluido el que exige autenticación antes de abrirlo a la red.

### Lo que hace el dashboard y no hace el panel

| Función | Destino |
|---|---|
| Puntaje, findings y su tendencia | vista «Proyectos» |
| Certificaciones, sprints y módulos | por decidir: `qdd report` ya los cubre en parte |
| Editar políticas (`/api/policies`) | sin reemplazo definido |
| Mandar intenciones y ejecutar agentes locales (`/api/intent`, `/api/agent/execute`) | sin reemplazo definido |
| Escribir archivos del proyecto (`/api/file`) | sin reemplazo definido |

## Preguntas abiertas

1. **`desktop/`:** la aplicación de escritorio importa `pkg/dashboard`. Quitar el dashboard la rompe, y no estaba entre las
   opciones que se le mostraron a la persona. Hay que decidir si también se descontinúa.
2. **Funciones sin reemplazo:** ¿alguien usa las de la tabla? El dashboard no registra su uso, así que hoy no se puede
   saber. Se responde antes de quitarlo.
3. **El propósito del proyecto:** el texto que entrega `qdd_project_context` promete un dashboard en tiempo real y ejecutar
   intenciones desde él. Se cambia cuando se quite.

## Fases

1. Aviso de obsoleto en `qdd dashboard` y documentación, con su prueba.
2. `qdd consola proyectos --json`: pruebas con proyectos temporales para cada `sin medir`, escalas distintas y una carpeta
   que ya no existe.
3. La vista en el mod: pruebas a 40 y 170 columnas y una corrida en el panel real, con la medición del presupuesto.
4. Aporte aditivo al blueprint `consola-claude-code`, con la confirmación de la persona.

## Fuera de alcance

- Mostrar esta vista en QDD Móvil.
- Acciones sobre otro proyecto desde el panel.
- Abrir nada a la red.
