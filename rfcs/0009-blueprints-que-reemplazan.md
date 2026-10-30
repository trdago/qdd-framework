# RFC 0009: Un blueprint que reemplaza a otro

**Status:** Accepted (qdd 2.8.0)
**Created:** 2026-10-30
**Extiende:** el [RFC 0004](0004-blueprints-centrales.md) (ciclo de vida) y el [RFC 0007](0007-fuentes-de-blueprints.md) (fuentes).

## 1. Summary

Un blueprint puede declarar que **reemplaza** a otros: `reemplaza: [estado-por-frente]`. Cuando la fuente publica el que reemplaza, el gate avisa a los proyectos que todavía usan el reemplazado (`BP-07-REPLACED`), y `qdd blueprint add <nuevo>` los migra. La migración saca el viejo del lockfile, pasa el registro de aplicación al nuevo blueprint y conserva el módulo, sus respuestas y lo que el proyecto agregó a la ficha. Las decisiones nuevas las sigue tomando la persona con `qdd blueprint apply`.

## 2. Motivation

El estándar de avance nació como `estado-por-frente` (la tabla) y después se le sumó `estado-por-tareas` (la tarea en curso y sus pulsos), que dependía del primero. Eran dos blueprints, dos palabras («frente» y «tarea») y un panel que ya decía «Tareas». La persona decidió que fueran **uno solo, `estado-por-tareas`** (2026-10-30).

Sin este RFC, cambiar el id de un blueprint publicado deja huérfanos a los proyectos que lo usan, como `llmlocal` y `notificador_de_ofertas`:

- su lockfile sigue en `estado-por-frente@1.0.0` y ninguna regla dice que existe un sucesor;
- `qdd blueprint add estado-por-tareas` lo importa **al lado** del viejo, y quedan dos estándares para el mismo módulo;
- la ficha generada se llama `estado-por-frente`, y `qdd estado` la buscaba por ese id fijo.

## 3. Design

### 3.1 El campo `reemplaza`

```yaml
id: estado-por-tareas
reemplaza: [estado-por-frente]
```

- Es una lista de ids válidos. No puede incluir su propio id.
- Solo vale **dentro de la misma fuente**: un blueprint de otro repositorio no puede reemplazar a uno ajeno.
- Las versiones de qdd anteriores a 2.8.0 ignoran el campo, porque el YAML se lee sin modo estricto.

### 3.2 `BP-07-REPLACED`

Para cada entrada del lockfile, el gate busca en el espejo central de su fuente, en el último tag publicado, un blueprint que la reemplace. Si lo encuentra:

> `estado-por-frente` fue reemplazado por `estado-por-tareas@2.0.0` en <fuente> (v6.0.0). Migra: `qdd blueprint add estado-por-tareas` (y luego `qdd blueprint apply estado-por-tareas --modulo estado-por-frente`).

Igual que `BP-06-BEHIND`, se basa en el espejo, sin red. El espejo se actualiza al importar y con `qdd blueprint sync`. Tiene la misma severidad que `BP-06`. Cuando una entrada está reemplazada, el gate no reporta además `BP-06` para ella.

### 3.3 La migración la hace `add`

`qdd blueprint add <nuevo>` importa el blueprint como siempre. Si el nuevo reemplaza ids que el lockfile tiene **desde la misma fuente**, además:

1. quita esas entradas del lockfile;
2. pasa cada registro de aplicación del viejo (`.qdd/project/blueprints/<modulo>.yaml`) al nuevo id;
   - **el módulo conserva su id**, así no se rompen las fichas aprobadas, HEART ni los enlaces;
   - las respuestas se conservan: el nuevo blueprint usa los mismos ids de pregunta para lo que heredó;
   - la versión del registro queda en la del blueprint viejo, para que `BP-03-OUTDATED` pida volver a aplicarlo;
3. lo dice en la salida.

No se aplica nada por la persona. Las preguntas nuevas (por ejemplo, `en_curso`) las pregunta `qdd blueprint apply`, y la tool MCP las devuelve a la IA para que se las haga a la persona.

Todo o nada: si falla la importación, el lockfile y los registros quedan como estaban.

### 3.4 Una ficha se encuentra por su blueprint

`qdd estado` y la consola buscaban la ficha por el id fijo `estado-por-frente`. Ahora la buscan por el blueprint que la generó:

- primero `estado-por-tareas`;
- después `estado-por-frente`, como legado.

Si no encuentran ninguna, prueban por id. Un proyecto nuevo tiene la ficha `estado-por-tareas` y uno migrado conserva `estado-por-frente`; los dos funcionan.

### 3.5 `doctor`

En el grupo 4, el chequeo «Blueprints al día con la organización» también informa los reemplazos, con el mismo comando de migración. `doctor --fix` **no** migra, igual que con `BP-06` no actualiza: migrar es una decisión de la persona y abre preguntas de diseño.

## 4. Pruebas

- Validación: `reemplaza` con un id inválido o con el propio id es inválido.
- `BP-07-REPLACED` aparece cuando el último tag de la fuente trae el reemplazo, y entonces no aparece `BP-06`.
- `add <nuevo>`: quita la entrada vieja, pasa el registro al nuevo id con el módulo y las respuestas, y `BP-03` pide volver a aplicar.
- `add <nuevo>` desde otra fuente no toca la entrada vieja.
- `qdd estado` encuentra una ficha generada por `estado-por-tareas` o por `estado-por-frente`.

## 5. Fuera de alcance

- Fusionar dos módulos aplicados en uno. Cada registro migra por separado.
- Reemplazos entre fuentes distintas.
