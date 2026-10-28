# RFC 0004: Blueprints centrales — QDD administra su ciclo de vida

**Status:** Implemented (v2.0.0; doctor en v2.1.0)
**Created:** 2026-10-28
**Reemplaza:** la fase 3 del [RFC 0003](0003-blueprints.md) en lo que toca a registro, almacenamiento y evolución.

## 1. Summary

Los blueprints de una organización dejan de copiarse dentro de cada proyecto. QDD los administra en una **carpeta central por usuario**, compartida por todos los proyectos de la máquina. El proyecto solo versiona su lockfile.

QDD también **dicta cómo evolucionan**: qué versión se puede importar, qué versión lleva un cambio, cómo se aporta y cuándo un proyecto quedó atrás. Nada de esto queda a criterio de cada persona.

## 2. Motivation

En v1.10–v1.11 cada `qdd blueprint add` clonaba el repositorio a un temporal y copiaba el blueprint a `.qdd/blueprints/<id>/` (FND-052). Con 10 proyectos de una organización eso significa:

- 10 registros del mismo repositorio;
- 10 copias vendorizadas del mismo blueprint;
- 10 clones completos;
- ningún flujo para aportar: `BP-02` prohibía editar la copia, pero no había cómo llevar el cambio al repositorio de la organización;
- versiones elegidas a mano, ramas y commits sueltos aceptados como origen.

## 3. Design

### 3.1 Carpeta central

```
~/.local/share/qdd/blueprints/          ($QDD_HOME/blueprints en el CI y las pruebas)
  organizaciones.yaml                   qué repositorio usa cada organización
  mirrors/<repo>-<hash>.git             un clon espejo por repositorio (fetch incremental)
  store/<sha256>/                       cada versión importada, inmutable y de solo lectura
  aportes/<repo>-<hash>/                el clon editable para proponer cambios
```

- Permisos 0700: el contenido de un repositorio privado no queda a la vista de otros usuarios.
- No se usa `/tmp`: se borra al reiniciar y suele ser compartido.
- El store se direcciona por contenido. Dos proyectos con la misma versión usan la misma carpeta, y una copia alterada se detecta y se reemplaza sola.
- Clones y escrituras son atómicos (carpeta temporal + rename): dos procesos a la vez nunca dejan algo a medias.

### 3.2 El proyecto solo versiona el lockfile

`.qdd/blueprints.lock.yaml` (formato 2) fija `origen`, `subdir`, `ref` (tag), `commit` y `sha256`.

`Find`, `List`, `apply`, `qdd_project_context` y el gate leen desde el store. Si una versión no está en la máquina (otra persona, el CI), QDD la **reconstruye desde el commit del lockfile** y verifica su huella.

**CI (opción A):** el job necesita acceso de solo lectura al repositorio de la organización (deploy key o token). No hay modo vendor.

### 3.3 Registro por organización

La primera vez que un proyecto importa desde un repositorio de varios blueprints, QDD lo guarda en dos lugares:

- en `.qdd/config.yaml` del proyecto, para el equipo;
- en `organizaciones.yaml`, con el patrón del dueño del remote git del proyecto (`git@github.com:acme/tienda.git` → `github.com/acme/*`).

Desde entonces, **todo proyecto de la misma organización ve sus blueprints sin configurar nada**: `qdd blueprint add auditoria` basta. Un patrón pertenece a una sola organización.

### 3.4 Evolución dictada por QDD

| Regla | Qué impone |
|---|---|
| Solo versiones publicadas | El origen es un repositorio git. `--ref` solo acepta tags `vX.Y.Z`. Sin `--ref`, QDD importa la última versión publicada, nunca lo no etiquetado. |
| La versión la calcula QDD | `qdd blueprint propose` compara con lo publicado: quitar o cambiar invariantes, decisiones, opciones, condiciones, vocabulario o escenarios es **major**; agregar es **minor**; redacción y referencias son **patch**. La `version:` que escribió el autor se reemplaza. |
| Un solo camino para aportar | `contribute` prepara la rama `aporte/<id>` en la carpeta central. `propose` valida, versiona, commitea, publica la rama y abre el PR (con `gh`). |
| Publicar es verificable | `qdd blueprint release` valida todos los blueprints, rechaza los que cambiaron sin subir su versión como corresponde (no pasaron por `propose`) y calcula el tag del repositorio con el mayor cambio. |
| Nadie se queda atrás | `qdd blueprint sync` trae lo publicado. El gate reporta `BP-06-BEHIND` cuando el espejo conoce una versión más nueva que la del lockfile. El CI siempre la conoce, porque clona fresco. |

### 3.5 Reglas del gate

| Regla | Cuándo |
|---|---|
| `BP-01-INVALID` | Un blueprint, el lockfile o un registro de aplicación no se puede leer o no es válido, o no se pudo obtener del repositorio (credenciales en el CI). |
| `BP-02-ALTERED` | Lo publicado no coincide con la huella o la versión del lockfile: el lockfile se editó a mano. |
| `BP-03-OUTDATED` | El módulo se generó con otra versión del blueprint; hay que re-aplicarlo. |
| `BP-04-MISSING` | El módulo se generó desde un blueprint que ya no está importado. |
| `BP-05-LEGACY` | Queda una copia dentro del proyecto (esquema anterior o copiada a mano). |
| `BP-06-BEHIND` | La organización publicó una versión más nueva. |

### 3.6 Migración automática, una sola vez

La primera vez que esta versión corre en un proyecto con `.qdd/blueprints/`:

- en cualquier comando `qdd` (aviso por stderr);
- o en la primera llamada MCP a `qdd_project_context` (aviso para la IA).

QDD hace lo siguiente:

1. Cada copia íntegra (su huella coincide con el lockfile) se guarda en el store y se borra del proyecto. No necesita red ni que el origen exista.
2. El lockfile pasa al formato 2, y el repositorio del `config.yaml` se lleva al registro central.
3. Una copia **editada a mano** o **fuera del lockfile** no se borra nunca. Queda como `BP-05-LEGACY`, con los pasos para que la IA la termine: `contribute`, portar los cambios, `propose` con confirmación del humano y borrar la copia.
4. Se fija `version` en `.qdd/state.json` (nunca hacia abajo). Un QDD anterior en otra máquina o en el CI se detiene con "El proyecto requiere QDD v2.0.0", en vez de trabajar sin ver los blueprints o volver a copiarlos dentro del proyecto.
5. La IA recibe la instrucción de commitear la eliminación de `.qdd/blueprints/`, el lockfile y `state.json`.

Sin `.qdd/blueprints/`, la migración no hace nada. Por eso ocurre una sola vez.

### 3.7 `qdd doctor` (v2.1.0)

**Principio:** toda funcionalidad de QDD es determinista y `qdd doctor` sabe diagnosticarla y repararla. Si doctor no puede definir cómo debería comportarse para repararla, la funcionalidad quedó ambigua y se completa.

Aplicar ese principio a los blueprints encontró dos bugs (FND-054): re-aplicar movía el módulo a las rutas sugeridas, y la ficha citaba la copia legada.

El grupo **"4. Blueprints de la organización"** tiene cinco diagnósticos. Sin red, el acceso no falla si la carpeta central ya tiene todo; un proyecto sin blueprints no usa la red, porque `qdd init` corre `doctor --fix`.

| Diagnóstico | `qdd doctor --fix` |
|---|---|
| Sin copias dentro del proyecto | Migra. Las copias editadas o fuera del lockfile no las borra: indica `contribute` y `propose`. |
| Íntegros en la carpeta central | Reconstruye desde el commit del lockfile. Si el lockfile está ilegible, lo restaura del último commit; si fue editado a mano, lo rehace desde la versión publicada que fija. |
| Acceso al repositorio | Explica qué credencial falta (llave SSH, deploy key o token). |
| Al día con la organización | Sincroniza el espejo. **No actualiza**: actualizar cambia invariantes que el código debe acatar, así que dice el comando exacto. |
| Módulos con la versión importada | Si el blueprint ya no está importado, lo importa del registro. Re-aplica con las respuestas registradas y la ficha vuelve a borrador para el humano. Si hay preguntas nuevas, dice cuáles y con qué comando responderlas. |

## 4. Drawbacks

- **Breaking (v2.0.0):** dejan de aceptarse carpetas sin git, ramas y commits como origen o `--ref`. El CI necesita acceso de lectura al repositorio de la organización.
- Un lockfile cuyo origen es una carpeta local solo se reconstruye en esa máquina. La organización debe publicar su repositorio en un remoto.
- `propose` abre el PR con la CLI de GitHub (`gh`). Sin ella, publica la rama e indica dónde abrir el PR.
