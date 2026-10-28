# RFC 0003: Blueprints — arquetipos de módulo de cada organización

**Status:** Implemented (fase 1 y 2). Registro, almacenamiento y evolución: ver [RFC 0004](0004-blueprints-centrales.md).
**Created:** 2026-10-23

## 1. Summary

Un **blueprint** es el esqueleto conceptual de un módulo que una organización reutiliza en todos sus proyectos. Ejemplos: usuarios, roles y permisos; notificaciones; auditoría. Fija seis cosas:

- el **vocabulario**;
- las **preguntas de diseño** que cada proyecto debe responder, por ejemplo "¿login con correo o con RUT?";
- las **decisiones** que implica cada respuesta;
- las **invariantes**, es decir, lo que ningún cambio puede romper;
- los **escenarios de conformidad**, que toda implementación debe pasar;
- opcionalmente, **implementaciones de referencia** por stack.

Los blueprints viven en el repositorio de la organización, privado si quiere. QDD solo trae el mecanismo para importarlos, aplicarlos y vigilarlos.

## 2. Motivation

Las fichas de módulo (`.qdd/project/modules`) le dan a cualquier IA el "corazón" de un módulo, pero cada proyecto las escribe desde cero. Una organización repite los mismos módulos en todos sus productos, y hoy cada equipo resuelve igual de distinto lo que debería ser un estándar.

Un ejemplo real: en una revisión de los login de varios proyectos de una misma empresa aparecieron cinco formas distintas de guardar tokens, roles comparados por nombre en el código, recuperaciones sin expiración y RUT validado solo en el frontend.

Los plugins del core (`.qdd/core/plugins`) no sirven para esto, por dos razones:

- se compilan dentro del binario público de QDD, que tiene licencia MIT;
- traen código, no decisiones.

## 3. Design

### 3.1 Repositorio de la organización

Es un solo repositorio para todos los blueprints. Cada carpeta de primer nivel que tenga un `blueprint.yaml` es un blueprint. `catalogo.yaml` es opcional y declara paquetes, o carpetas cuyo nombre no coincide con el id:

```
org/qdd-blueprints   (privado)
├── catalogo.yaml            # paquetes: {base: [auth, auditoria]}
├── auth/
│   ├── blueprint.yaml
│   └── referencias/…        # implementaciones de referencia (opcional)
└── auditoria/blueprint.yaml
```

La plantilla con todos los campos está en `.qdd/core/templates/blueprint.yaml`. Las reglas de validación son:

- id y valores en minúsculas;
- versión en semver;
- al menos dos opciones por pregunta;
- `si` solo puede depender de preguntas anteriores y de valores que existan;
- los escenarios no se repiten dentro de su grupo ni pisan los de la base;
- las referencias deben quedar dentro de la carpeta del blueprint.

### 3.2 Importar: `qdd blueprint add` / MCP `qdd_blueprint_add`

```bash
qdd blueprint add <repo> --ref v1.2.0 auth   # la primera vez: QDD registra el repositorio
qdd blueprint add auditoria notificaciones   # después, solo los nombres
qdd blueprint add --paquete base
qdd blueprint add --todos --ref v1.3.0       # actualizar todo (y el ref registrado)
qdd blueprint add                            # listar lo disponible
qdd blueprint add <repo>//auth --ref v1.2.0  # una subcarpeta puntual
```

**QDD administra el registro.** Cuando se importa por primera vez desde un repositorio con varios blueprints, guarda la clave `blueprints: {origen, ref}` en `.qdd/config.yaml`. Edita esa clave conservando el resto del archivo, comentarios incluidos.

- Desde entonces, un argumento que no parece origen es un id. Un origen contiene `/`, `:` o `\`, o empieza con `.` o `~`.
- Pedir otro `--ref` del mismo repositorio actualiza el ref registrado.
- Importar algo puntual de otro repositorio no reemplaza el registro.
- Por MCP, el humano dice "agrega auth y auditoría" y la IA llama a `qdd_blueprint_add`. Si no hay registro, la tool le exige a la IA preguntarle la URL al humano, nunca inventarla.

- Se clona con `git` y las credenciales del usuario. **La protección es el control de acceso al repositorio**: quien no tiene acceso no puede importar.
- `protocol.ext.allow=never` y `--` evitan la inyección de transportes y de opciones.
- **Todo o nada**: se validan todos los blueprints pedidos antes de copiar cualquiera.
- ~~Cada blueprint se copia a `.qdd/blueprints/<id>/`~~. Desde el [RFC 0004](0004-blueprints-centrales.md), vive en la carpeta central de QDD y el proyecto solo versiona el lockfile.
- `.qdd/blueprints.lock.yaml` fija `origen`, `subdir`, `ref`, `commit` y el `sha256` de cada copia.

### 3.3 Aplicar: `qdd blueprint apply` / MCP `qdd_blueprint_apply`

```bash
qdd blueprint apply auth --ruta internal/auth --prueba "go test ./internal/auth/..."
qdd blueprint apply auth auditoria --respuesta auth.identificador=rut
```

1. Resuelve las preguntas en orden, teniendo en cuenta las condiciones `si`.
   - Con terminal, las pregunta una por una: acepta el número o el valor de la opción, y Enter elige la sugerida.
   - Sin terminal, o por MCP, **lista las que faltan y no escribe nada**: las decisiones de diseño nunca se toman por defecto. La tool MCP le indica a la IA que se las haga al humano.
2. Genera la ficha `.qdd/project/modules/<modulo>.yaml` en **borrador**, con `blueprint: auth@1.2.0`. Lleva las decisiones e invariantes de la base, el vocabulario y lo que implican las opciones elegidas; los `{{pregunta}}` se reemplazan por la respuesta. La aprueba un humano con `qdd modules approve`.
3. Escribe los escenarios de conformidad en `.qdd/project/goldensets/<modulo>/conformidad.yaml`. Es YAML, así que el runner de goldensets JSON lo ignora. El proyecto los implementa como pruebas.
4. Registra las decisiones del proyecto en `.qdd/project/blueprints/<modulo>.yaml`: respuestas, versión y lo que el blueprint generó.
5. Si una ruta del módulo no existe, la crea con un `README.md` que contiene el esqueleto conceptual. Una carpeta existente nunca se toca, y una ruta dentro de `.qdd/` se rechaza (CERT-033).
6. **Re-aplicar** reutiliza las respuestas anteriores y solo pregunta lo nuevo. Reemplaza lo que generó el blueprint y conserva lo que el proyecto agregó: decisiones e invariantes propias, ADRs, findings, comandos de prueba y caso real.

### 3.4 Vigilar: `qdd audit` (categoría BLUEPRINT)

| Regla | Cuándo |
|---|---|
| `BP-01-INVALID` | Un blueprint importado, el lockfile o un registro de aplicación no se puede leer o no es válido. |
| `BP-02-ALTERED` | La copia en `.qdd/blueprints/` no coincide con la huella del lockfile, o no figura en él. Los cambios se hacen en el repositorio de la organización. |
| `BP-03-OUTDATED` | El módulo se generó con una versión distinta de la importada; hay que volver a aplicarlo. |
| `BP-04-MISSING` | El módulo se generó desde un blueprint que ya no está importado. |

Sin blueprints, no reporta nada. Las reglas `HEART-*` siguen vigilando el desfase entre la ficha y el código.

### 3.5 Que la IA entienda el concepto

Una IA conoce los blueprints por tres vías, sin que nadie se lo pida:

1. **Reglas de los adaptadores** (`qdd_sync`: `CLAUDE.md`, `.cursorrules`, …):
   - un pedido como "agrega el blueprint X" se resuelve con `qdd_blueprint_add`;
   - antes de construir un módulo que un blueprint define, la IA llama a `qdd_blueprint_apply` y le hace al humano sus preguntas;
   - las invariantes no se negocian: un cambio va al repositorio de la organización.
2. **`qdd_project_context`**, la primera llamada de toda tarea:
   - suma la regla 6 de la forma de trabajo;
   - agrega la sección *"Estándares de la organización"*, con cada blueprint aplicado o sin aplicar;
   - si la IA va a tocar una ruta sin ficha que coincide con las rutas sugeridas de un blueprint sin aplicar, le pide aplicarlo en vez de proponer una ficha a mano;
   - muestra el `blueprint` de cada ficha.
3. **`qdd_blueprint_list`**, que entrega el vocabulario, las preguntas y las invariantes.

## 4. Drawbacks y límites

- **Confidencialidad:** la copia importada la ve cualquiera con acceso al proyecto. Es la misma situación que cualquier código interno. No hay DRM; la protección es el control de acceso y los contratos.
- **Conformidad:** los escenarios son texto. QDD verifica que existan y estén vigentes, no que el proyecto los haya implementado como pruebas.

## 5. Fase 3 (pendiente)

- Verificación de firma de los tags (cosign o minisign) antes de importar.
- Lista de orígenes permitidos en `.qdd/config.yaml`, para que un proyecto solo acepte blueprints del repositorio de su organización.
- ~~Un registro a nivel de usuario~~: resuelto en el [RFC 0004](0004-blueprints-centrales.md) (registro central por remote git).
- Dependencias entre blueprints, por ejemplo `auditoria` requiere `auth`.
- Trazabilidad entre escenarios y pruebas: que el gate exija una prueba que cite cada id de escenario.
