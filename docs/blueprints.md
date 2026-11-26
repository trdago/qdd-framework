# Blueprints: los estándares de módulo de tu organización

Una organización construye los mismos módulos una y otra vez: usuarios, roles y permisos, notificaciones, auditoría, pagos. Sin un estándar, cada equipo (y cada IA) los resuelve distinto. Un **blueprint** es el esqueleto conceptual de uno de esos módulos, escrito una vez y aplicado en todos los proyectos.

Un blueprint no trae código. Trae decisiones:

| Parte | Qué es | Ejemplo |
|---|---|---|
| **Vocabulario** | Palabras que toda la organización usa igual. | *canal*: medio de entrega (correo, WhatsApp, push). |
| **Preguntas de diseño** | Lo que cada proyecto debe decidir. El blueprint no lo decide por él. | ¿Por qué canales se notifica? |
| **Decisiones** | Lo que implica cada respuesta. | Si es WhatsApp: vía la API oficial. |
| **Invariantes** | Lo que ningún cambio puede romper. | Ninguna notificación incluye secretos. |
| **Escenarios de conformidad** | Casos que toda implementación debe pasar. | El mismo evento procesado dos veces genera una sola notificación. |
| **Referencias** (opcional) | Implementaciones de referencia por stack. | `referencias/go/…` |

QDD es público y solo trae el **mecanismo**: importar, aplicar, vigilar y evolucionar. Los blueprints viven en el repositorio de tu organización (privado o público) y, en tu máquina, en la carpeta central de QDD: el proyecto solo versiona el lockfile. El diseño está en [RFC 0003](../rfcs/0003-blueprints.md), [RFC 0004](../rfcs/0004-blueprints-centrales.md) (carpeta central y ciclo de vida) y [RFC 0007](../rfcs/0007-fuentes-de-blueprints.md) (varias fuentes).

```mermaid
graph LR
    classDef org fill:#6366f1,stroke:#4338ca,color:#fff;
    classDef proj fill:#1e1e1e,stroke:#3b82f6,color:#fff;
    classDef gate fill:#14b8a6,stroke:#0f766e,color:#fff;
    R["Repositorio de la organización<br/>blueprint.yaml + tag v1.0.0"]:::org
    A["qdd blueprint add<br/>carpeta central + lockfile"]:::proj
    P["qdd blueprint apply<br/>el humano responde las preguntas"]:::proj
    F["Ficha del módulo en borrador<br/>+ escenarios de conformidad"]:::proj
    H["qdd modules approve<br/>aprobación humana"]:::proj
    G["qdd audit<br/>BP-01 a BP-08"]:::gate
    C["contribute → propose → release<br/>PR y tag con semver calculado"]:::org
    R --> A --> P --> F --> H --> G
    G -- BP-06: versión nueva --> A
    H -. cambiar el estándar .-> C --> R
```

## 1. Crear el repositorio de la organización

Un solo repositorio guarda todos los blueprints. Cada carpeta de primer nivel con un `blueprint.yaml` es un blueprint:

```
org/qdd-blueprints          (privado si quieres)
├── catalogo.yaml           # opcional: paquetes y nombre de la organización
├── notificaciones/
│   ├── blueprint.yaml
│   └── referencias/…       # opcional
└── auditoria/
    └── blueprint.yaml
```

`catalogo.yaml` agrupa blueprints en paquetes que se importan juntos:

```yaml
organizacion: Mi organización
paquetes:
  base: [notificaciones, auditoria]
```

Si el nombre de una carpeta no coincide con el id del blueprint, decláralo con `blueprints: {<id>: <carpeta>}`.

### Escribir un blueprint

Parte de la plantilla [`.qdd/core/templates/blueprint.yaml`](../.qdd/core/templates/blueprint.yaml), que trae los campos comentados. Son obligatorios `id`, `nombre`, `version` y `ficha`. Lo esencial:

```yaml
id: notificaciones
nombre: Notificaciones
version: 1.0.0                     # semver
descripcion: Cómo se avisa a las personas de lo que pasa en el producto.

vocabulario:
  - termino: canal
    definicion: Medio de entrega (correo, WhatsApp, push).

preguntas:
  - id: canales
    pregunta: ¿Por qué canales se notifica?
    por_defecto: correo            # sugerencia: nunca se aplica sin preguntar
    opciones:
      - valor: correo
        invariantes: ["Todo correo trae enlace para dejar de recibirlo"]
      - valor: whatsapp
        decisiones: ["WhatsApp vía la API oficial"]
  - id: proveedor
    pregunta: ¿Qué proveedor de WhatsApp?
    si: {canales: whatsapp}        # solo se pregunta si se eligió whatsapp
    opciones:
      - valor: meta
      - valor: otro

ficha:                             # base de la ficha de módulo que se genera
  nombre: Notificaciones
  rutas: [internal/notificaciones] # sugerida; el proyecto la cambia con --ruta
  objetivo_operativo:
    problema: Qué problema resuelve el módulo.
    criterio_de_exito: Cómo se sabe que cumple (medible).
  objetivo_tecnologico:
    arquitectura: Cómo se construye y por qué.
    decisiones: ["Envío asíncrono con reintentos"]
    invariantes: ["Ninguna notificación incluye secretos ni datos sensibles completos"]

conformidad:
  - id: NOT-C01
    titulo: Idempotencia
    cuando: el mismo evento se procesa dos veces
    entonces: la persona recibe una sola notificación
```

Reglas que QDD valida al importar:

- id y valores en minúsculas (números, `-` o `_`);
- `version` en semver;
- al menos dos opciones por pregunta (si no hay alternativa, no es una pregunta: es una invariante);
- `si` solo depende de preguntas anteriores y de valores que existan;
- los ids de escenario no se repiten;
- las `referencias` quedan dentro de la carpeta del blueprint.

En los textos, `{{pregunta}}` se reemplaza por la respuesta del proyecto.

Con ese ejemplo alcanza para importar y aplicar, pero un blueprint es una receta completa. Estos campos también cuentan:

| Campo | Qué dice |
|---|---|
| `alcance` | Qué cubre y qué no |
| `pasos` | El orden para construirlo |
| `parametros`, `constantes` | Los valores con nombre y su unidad |
| `dado` en un escenario | El estado del que parte |
| `requiere_qdd` | El qdd mínimo que lo entiende |
| `reemplaza` | Los ids que este blueprint reemplaza |

```bash
qdd blueprint quality notificaciones   # qué le falta para que otra IA lo construya igual (CAL-01 a CAL-18)
```

### Publicar el primero

El repositorio necesita un primer commit con los blueprints. Después, etiqueta la primera versión:

```bash
git add -A && git commit -m "feat: primeros blueprints" && git push
qdd blueprint release --origen git@github.com:org/qdd-blueprints.git
```

### Versionar

Después de la primera, cada versión es un tag `vX.Y.Z` que no fijas a mano: `qdd blueprint propose` compara con lo publicado y calcula el cambio (ver [§4](#4-cambiar-o-crear-un-blueprint)):

| Cambio | Versión |
|---|---|
| Redacción, ayudas, vocabulario | patch (`1.0.1`) |
| Opciones o preguntas nuevas | minor (`1.1.0`) |
| Invariantes nuevas o distintas | major (`2.0.0`) |

## 2. Importar en un proyecto

La primera vez indicas el repositorio. QDD lo clona con tus credenciales git, valida los blueprints y registra el repositorio en `.qdd/config.yaml` y en el registro central (los demás proyectos de la organización, con el mismo dueño en su remote git, lo encuentran solos):

```bash
qdd blueprint add git@github.com:org/qdd-blueprints.git --ref v1.0.0 notificaciones
```

```yaml
# .qdd/config.yaml (QDD edita solo esta clave y conserva el resto)
blueprints:
    origen: git@github.com:org/qdd-blueprints.git
```

Después basta con los nombres:

```bash
qdd blueprint add auditoria             # uno más, del repositorio registrado
qdd blueprint add --paquete base        # un paquete del catalogo.yaml
qdd blueprint add --todos               # todo
qdd blueprint add                       # lista lo disponible (y sale con 1: no importó nada)
```

- El contenido vive en la **carpeta central** de QDD (`~/.local/share/qdd/blueprints`, o `$QDD_HOME`), nunca en el proyecto. El proyecto solo versiona `.qdd/blueprints.lock.yaml`, con la fuente, la versión, el commit y la huella de cada blueprint. Commitéalo.
- **`--ref`** elige la versión de esa importación; la que queda fija es la del lockfile, por blueprint.
- **Todo o nada:** si uno de los pedidos es inválido, no se importa ninguno, y si una fuente falla el lockfile queda como estaba.
- **Varias fuentes:** puedes sumar tus propios estándares con la URL de otro repositorio (`qdd blueprint add git@github.com:yo/mis-blueprints.git mi-estandar`). El lockfile recuerda la fuente de cada uno y `qdd blueprint add <id>` lo actualiza desde ahí. Un id ya importado desde otra fuente se rechaza, salvo con `--reemplazar`.
- **Repositorios públicos:** se leen por HTTPS sin cuenta ni llave. Si una URL SSH falla (p. ej. en el CI, sin deploy key), QDD reintenta por su URL HTTPS; si tampoco funciona, el error te la dice.
- **La protección es el acceso al repositorio:** quien no tiene acceso no puede importar. QDD bloquea transportes `ext::` e inyección de opciones en git.
- **Otro PC:** `qdd doctor --fix` reconstruye la carpeta central desde el lockfile. Un proyecto con copias del esquema anterior (`.qdd/blueprints/`) se migra igual.

## 3. Aplicar: generar el módulo

```bash
qdd blueprint apply notificaciones --ruta internal/notificaciones \
  --prueba "go test ./internal/notificaciones/..."
```

Con terminal, QDD hace las preguntas una por una (Enter elige la sugerida). Sin terminal, pásalas con `--respuesta`:

```bash
qdd blueprint apply notificaciones --respuesta canales=correo
```

Si falta alguna respuesta, **no escribe nada** y dice cuáles faltan. Las decisiones de diseño nunca se toman por defecto.

Lo que genera:

| Archivo | Contenido |
|---|---|
| `.qdd/project/modules/notificaciones.yaml` | La ficha del módulo en **borrador**, con `blueprint: notificaciones@1.0.0`, las decisiones e invariantes de la base y las de las opciones elegidas. |
| `.qdd/project/goldensets/notificaciones/conformidad.yaml` | Los escenarios de conformidad, para implementarlos como pruebas. |
| `.qdd/project/blueprints/notificaciones.yaml` | Las respuestas del proyecto, la versión y lo que generó el blueprint. |
| `internal/notificaciones/README.md` | Un esqueleto conceptual, solo si la carpeta no existía. Una carpeta existente nunca se toca. |

Volver a aplicarlo (por ejemplo, tras importar una versión nueva) reutiliza las respuestas, pregunta solo lo nuevo y **conserva lo que el proyecto agregó** a la ficha: decisiones e invariantes propias, ADRs, findings, comandos de prueba y caso real.

Por último, un humano revisa y aprueba la ficha. Volver a aplicar el blueprint la devuelve a borrador:

```bash
qdd modules approve notificaciones
```

Desde aquí `qdd audit` bloquea con `BP-08-CONFORMIDAD` hasta que cada escenario tenga su prueba. Lo mide por nombre: busca el
id del escenario en las pruebas de las rutas de la ficha. No comprueba que la prueba verifique lo que dice el escenario.
Si un escenario lo cumple otro, decláralo en la ficha con su motivo:

```yaml
fuera:
  NOT-C01: lo garantiza la cola de mensajes, fuera de este módulo
```

## 4. Cambiar o crear un blueprint

Un blueprint nunca se edita en el proyecto: el cambio va a su repositorio, por PR.

```bash
qdd blueprint contribute notificaciones   # rama aporte/notificaciones en un clon de la carpeta central
# … edita el blueprint en la carpeta que te indica …
qdd blueprint propose notificaciones -m "Agrega el canal push"   # valida, calcula la versión, abre el PR
# tras el merge:
qdd blueprint release                     # etiqueta vX.Y.Z en el repositorio
```

- Uno importado se aporta siempre a su fuente; uno nuevo, a la organización o a la de `--origen`.
- `propose` calcula patch, minor o major según lo que cambió; `release` rechaza un blueprint que cambió sin pasar por `propose`.
- Si SSH no tiene permiso de escritura (por ejemplo, una deploy key de solo lectura), `propose` y `release` reintentan por HTTPS con tus credenciales de git.
- Los demás proyectos lo ven en su próximo `qdd audit` como `BP-06-BEHIND`.
- **Cambiar el id de un blueprint** (o juntar dos en uno): crea el nuevo con `reemplaza: [id-viejo]`. Los proyectos que usan el viejo reciben `BP-07-REPLACED` y migran con `qdd blueprint add <nuevo>` ([RFC 0009](../rfcs/0009-blueprints-que-reemplazan.md)).

## 5. Con una IA

No hace falta usar la CLI. Las reglas que `qdd init` deja en tu IDE (`AGENTS.md`, `.clauderc`, `.cursorrules`…) le enseñan a la IA a usar los blueprints sola:

- Si pides *"agrega el blueprint de notificaciones y auditoría"*, la IA llama a `qdd_blueprint_add`. Si el proyecto no tiene repositorio registrado, te pregunta la URL; nunca la inventa.
- Antes de construir un módulo que un blueprint define, llama a `qdd_blueprint_apply` y **te hace sus preguntas tal cual**. Las respuestas son tuyas, no de la IA.
- `qdd_project_context`, la primera llamada de toda tarea, muestra los estándares de la organización (aplicados o sin aplicar). Si la IA va a tocar una carpeta que coincide con las rutas de un blueprint sin aplicar, le pide aplicarlo en vez de inventar una ficha.
- Las invariantes no se negocian: si una no sirve, la IA prepara el cambio con `qdd_blueprint_contribute` y, con tu confirmación, lo propone con `qdd_blueprint_propose` (QDD fija la versión). Nunca copia ni edita un blueprint dentro del proyecto.

## 6. Vigilar: el gate

`qdd audit` revisa los blueprints en cada corrida. Sin blueprints importados, no reporta nada.

| Regla | Cuándo bloquea | Qué hacer |
|---|---|---|
| `BP-01-INVALID` | Un blueprint, el lockfile o un registro de aplicación no se puede leer o no es válido. | Corrige el blueprint en el repositorio de la organización y vuelve a importarlo. |
| `BP-02-ALTERED` | Lo publicado en el repositorio no coincide con la huella o la versión del lockfile (alguien lo editó a mano). | Vuelve a importarlo desde su fuente; los cambios van por `contribute` → `propose`. |
| `BP-03-OUTDATED` | El módulo se generó con una versión distinta de la importada. | `qdd blueprint apply <id>` otra vez. |
| `BP-04-MISSING` | El módulo se generó desde un blueprint que ya no está importado. | Vuelve a importarlo o quita la referencia de la ficha. |
| `BP-05-LEGACY` | Queda una copia del esquema anterior dentro del proyecto (`.qdd/blueprints/`). | `qdd doctor --fix` la migra a la carpeta central; commitea el resultado. |
| `BP-06-BEHIND` | Su fuente publicó una versión más nueva. | `qdd blueprint add <id>` y vuelve a aplicarlo con `qdd blueprint apply`. |
| `BP-07-REPLACED` | Su fuente publicó otro blueprint que lo reemplaza (`reemplaza: [id]`). | `qdd blueprint add <nuevo>`: saca el viejo del lockfile y pasa tus módulos al nuevo, con su id y sus respuestas. Después `qdd blueprint apply <nuevo> --modulo <módulo>` para responder lo nuevo. |
| `BP-08-CONFORMIDAD` | A un módulo que aplica o implementa el blueprint le falta la prueba de un escenario que debe cumplir. Se mide sin red: con lo importado o con la última versión de la copia local del repositorio. Si el blueprint lo implementa qdd (`estado-por-tareas`, `consola-claude-code`) y el módulo solo lo usa, lo cumple el qdd instalado cuando lo implementa en la misma versión mayor y no más vieja. | Escribe una prueba que lleve el id del escenario (con «-» o «_») en las rutas de su ficha, o déjalo en `fuera` con su motivo si lo hace otro: en la raíz de la ficha si aplica el blueprint, o dentro de `implementa`. `qdd blueprint conformance` muestra qué falta. |
| `BP-08-SIN-MEDIR` | Recomendación, no bloquea: el blueprint no está importado ni en la copia local de esta máquina (p. ej. en el CI). | `qdd blueprint conformance` lo trae y lo mide. |

Las reglas `HEART-*` siguen vigilando que la ficha no quede atrás del código (ver [Reglas del gate](command-reference.md#reglas-del-gate)).

## Los demás comandos

| Comando | Para qué |
|---|---|
| `qdd blueprint list`, `show <id>` | Lo importado, y el detalle de uno |
| `qdd blueprint conformance` | Qué escenarios cubre cada módulo |
| `qdd blueprint quality <id>` | Qué le falta a la receta |
| `qdd blueprint certify <id>` | Certificación ciega: otra IA lo construye solo con el blueprint |
| `qdd blueprint distill <módulo>` | Saca un blueprint nuevo de una ficha aprobada |
| `qdd blueprint sync` | Trae lo publicado por cada fuente |

Las opciones de cada uno están en la [Command Reference](command-reference.md#blueprints).

## Lo que todavía no existe

Verificación de firma de los tags, lista de orígenes permitidos y dependencias entre blueprints
([RFC 0003](../rfcs/0003-blueprints.md#5-fase-3-pendiente)).
