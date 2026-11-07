# RFC 0019: Un turno único para lo que espera a la persona

- Estado: aceptado (2026-11-06).
- Origen: pedido de la persona: «no puedo estar adivinando cuál debería contestar primero». El orden lo define qdd una sola
  vez, al entrar a la cola, con un número o una fecha, y las dependencias quedan marcadas.

## El problema

El panel y el teléfono ordenan por tipo, cada uno a su manera:

- El panel muestra pedidos, luego ⏳ y, en otra fila, fichas. Las preguntas abiertas no salen.
- El teléfono muestra preguntas, luego fichas y luego pedidos.
- Un ⏳ no guarda cuándo entró, y una ficha tampoco. Solo los pedidos marcan dependencias.
- Una tecla puede reutilizarse, y la de una ficha cambia cuando sale otra.

## La regla

1. Lo que espera a la persona recibe de qdd un **turno**: un número del proyecto que no cambia ni se reutiliza, y la fecha en
   que entró. Se da una sola vez, la primera vez que qdd lo ve.
2. La lista va por número. Lo que otra cosa espera va justo antes de ella.
3. **Siguiente** es lo primero de la lista que la persona puede contestar. Lo calcula qdd.
4. Cada cosa dice qué turnos espera.
5. Panel y teléfono dibujan ese orden y no lo recalculan.

Cada proyecto tiene su numeración. Entre proyectos, el teléfono ordena por la fecha.

## Qué lleva turno

| Tiene turno | Hasta |
|---|---|
| Pregunta de la consola o permiso | Se responde, vence o se cierra. |
| ⏳ de la persona que pregunta algo | Se responde o sale del estado. Reescribir el texto no le cambia el número. |
| Pedido | Queda hecho o se retira. Confirmado, conserva el número y deja de ser «siguiente». |
| Ficha en borrador | Se aprueba. |

No llevan turno los pedidos que qdd ofrece por su cuenta (mergear un PR abierto, actualizar qdd), las herramientas, «✓ Ya lo
vi», ni lo mandado desde el teléfono, porque nada de eso espera una respuesta.

## Dependencias

- Un pedido espera lo que declara con `despues`. Esa regla sigue vigente: no corre hasta que eso termine.
- Lo de una tarea espera lo abierto de las tareas de su `depende_de`.
- En un ⏳, una ficha o una pregunta, la dependencia es un aviso y no impide contestar: una persona nunca queda atrapada por
  un ⏳ escrito de más.

## Contrato

Cada `Accion` de `qdd consola panel --json` suma:

```
{"turno": 12, "turno_desde": "2026-11-06T18:00:00Z", "espera_turno": [11], "siguiente": true}
```

`acciones` trae primero lo que tiene turno, ya en orden. `esperas_persona` suma `pregunta`, el id de su acción, para que el
panel encuentre su lugar.

## Registro

`RutaLocal(root, "turnos")`, con candado y escritura atómica: el próximo número y lo vivo. No se versiona. Si no se puede leer,
se aparta como `.ilegible` y `qdd doctor` lo nombra; la numeración vuelve a empezar. Si el candado está ocupado, esa lectura
sale sin turnos y la siguiente los trae.

## Sin resolver

- La app ordena con sus propios grupos. Hasta que use este orden, los números ayudan solo en el panel.

## Fases

1. PC: turnos, orden, dependencias y `qdd doctor`.
2. Panel: una sola lista en «Para ti», con número, «siguiente» y lo que espera.
3. App (`qdd-movile`): la misma lista, sin sus grupos.

## Fuera de alcance

- Forzar el orden: contestar otra cosa antes sigue permitido, salvo la regla de los pedidos.
- Una numeración global entre proyectos.
