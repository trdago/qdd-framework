# RFC 0007: Varias fuentes de blueprints en un proyecto

**Status:** Implemented (qdd 2.5.0)
**Created:** 2026-10-30
**Extiende:** el [RFC 0004](0004-blueprints-centrales.md) (registro, sincronización y aportes).

## 1. Summary

Un proyecto puede usar blueprints de **más de un repositorio**. Por ejemplo, los de su organización y los del repositorio público de una persona. El lockfile ya guardaba el origen de cada blueprint; ahora QDD lo usa en todo el ciclo de vida:

- importar y actualizar;
- sincronizar;
- aportar y publicar;
- reparar con `doctor`.

No se agrega configuración nueva.

## 2. Motivation

El RFC 0004 supone una sola fuente por proyecto: el repositorio de su organización. Importar desde otra URL ya funcionaba, pero QDD no sabía administrar ese blueprint después:

- **FND-064.** Un blueprint con el mismo id desde otra fuente reemplazaba al importado sin avisar, porque el lockfile identifica cada entrada solo por su id.
- **FND-065.** `qdd blueprint add <id>`, que es lo que pide `BP-06-BEHIND`, buscaba el id en el repositorio de la organización. Fallaba, o traía el blueprint equivocado.
- **FND-066.** `qdd blueprint sync` solo miraba el repositorio de la organización. Un blueprint de otra fuente nunca recibía aviso de versión nueva.
- Un blueprint **nuevo** solo se podía crear y publicar en el repositorio de la organización.

## 3. Design

### 3.1 Las fuentes son las del lockfile más la registrada

Las fuentes de un proyecto son dos:

- el repositorio registrado, es decir, el de `.qdd/config.yaml` o el de su organización;
- cada `origen` distinto del lockfile.

No hay una lista aparte que mantener ni que pueda desalinearse del lockfile. Importar desde otra fuente **no** cambia el repositorio registrado.

```
qdd blueprint add git@github.com:yo/mis-blueprints.git mi-estandar   # segunda fuente
qdd blueprint add mi-estandar auth          # cada uno desde su fuente
```

### 3.2 Un id pertenece a una sola fuente

`add` rechaza, antes de guardar nada, un id que ya está importado desde otra fuente. El mensaje dice cómo cambiarlo a propósito:

- en la CLI, con `--reemplazar`;
- por MCP, con `reemplazar`.

Prefijar los ids por fuente (`personal/auth`) se descartó. Obligaría a cambiar el formato de los ids en las fichas, en las aplicaciones y en el gate, para un caso que basta con impedir.

### 3.3 Sin origen, cada id va a su fuente

`AddManaged` reparte lo pedido entre las fuentes:

- un id del lockfile se actualiza desde su fuente;
- un id nuevo, `--paquete` y `--todos` van al repositorio registrado.

Hace un solo `Add` por fuente. El reparto también cumple estas reglas:

- **Todo o nada.** Si una fuente falla, el lockfile queda como estaba. El store es inmutable y solo crece, así que basta con restaurar el lockfile.
- **`--ref` vale para una sola fuente.** Un tag es la versión de un repositorio, así que se rechaza si lo pedido viene de varias fuentes.
- **El gate nombra la fuente.** El aviso de `BP-06-BEHIND` dice qué fuente publicó. Su instrucción (`qdd blueprint add <id>`) ahora es correcta para cualquier fuente.

### 3.4 `sync` recorre todas las fuentes

`Sincronizar` devuelve **un resultado por fuente**, con su última versión y sus novedades. Una fuente inaccesible o sin versiones publicadas lleva su `Problema` y no oculta las demás. La CLI la informa y termina con error.

### 3.5 Aportar y publicar en otra fuente

`contribute`, `propose` y `release` aceptan `--origen` (y `origen` por MCP):

- un blueprint **importado** se aporta siempre a su fuente; pasar otra se rechaza;
- uno **nuevo** va a la de `--origen` o, sin ella, al repositorio registrado;
- `release --origen` etiqueta esa fuente.

El versionado sigue siendo el del RFC 0004 §3.4: lo calcula QDD.

### 3.6 `doctor`

`doctor` ya revisaba el acceso y la vigencia por cada origen del lockfile. Rehacer una entrada usa la misma función que `add` (`origenDeEntrada`), así que reconstruye desde su fuente.

## 4. Drawbacks

- El registro central sigue asociando cada patrón de remote a **un** repositorio. Las fuentes extra son del proyecto (su lockfile), no de la organización. Esto es intencional: el estándar de la organización no se mezcla con preferencias personales.
- Cambiar un blueprint de fuente es explícito (`--reemplazar`) y obliga a volver a aplicarlo a sus módulos.
