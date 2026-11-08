# RFC 0024: reportar problemas a QDD sin GitHub

**Estado:** aceptado (2026-11-08), con las decisiones de la persona.

## Problema

Hoy un problema llega a QDD de dos maneras, y las dos dejan gente fuera:

- **Incidentes:** se publican como issues con `gh`, así que hacen falta una cuenta de GitHub y `gh` autenticado.
- **`qdd mejoras reportar`:** escribe en la cola de mejoras, pero solo dentro del mismo PC.

La persona lo pidió así: «necesitamos generar un camino directo a qdd para enviar los problemas; no todos los usuarios
tienen github».

## Decisiones de la persona (2026-11-08, desde QDD Móvil)

1. **Los reportes llegan al relay propio de QDD.** Es el proyecto Supabase de QDD Móvil, con una función y una tabla
   solo de entrada. La cola de mejoras del repositorio de QDD los lee como una fuente más, igual que los issues.
2. **Puede enviar cualquier instalación, con tope.** Cada qdd crea su propia llave, sin cuenta. El relay acepta pocos
   reportes por día y por llave, con un tamaño máximo.
3. **El reporte se limpia y se confirma.** qdd quita secretos, rutas y nombres, muestra el texto exacto y lo envía solo
   con el «sí» de la persona. Una IA puede prepararlo, pero el envío lo confirma la persona.

## Diseño

### En el PC: `qdd reportar`

```
qdd reportar --titulo "…" [--detalle "…" | --detalle-de <archivo>] [--tipo bug|mejora|pregunta]
qdd reportar llave        # muestra la llave pública de esta instalación
```

1. **Limpia** el texto. Pasa por el ocultador de valores del baúl y luego quita credenciales (`clave=valor`, prefijos
   conocidos, cadenas largas), la ruta del proyecto, la carpeta del usuario y su nombre. Es más suave que
   `incidentes.Sanitizar`: un reporte lo escribe una persona, y quitar todas las comillas o nombres de archivo lo
   dejaría sin sentido. Por eso se muestra antes de enviar.
2. **Agrega** la versión de qdd y el sistema (`linux/amd64`). Nada más: ni el nombre del proyecto ni la carpeta.
3. **Confirma** con la persona:
   - **En su terminal:** muestra el texto exacto y pregunta «¿Enviar? [s/N]».
   - **Sin terminal** (la IA, un pedido o el teléfono): deja el reporte en `<QDD_HOME>/reportes/pendientes/<id>.json` y
     un pedido «📮 Enviar reporte: <título>» que solo confirma la persona (`DePersona`, el modo goal no lo lanza, como
     en el baúl). Sale con 75. El pedido corre `qdd reportar enviar <id>`, que comprueba `QDD_PEDIDO` contra la cola.
4. **Firma y envía.** Usa la llave de la instalación, `<QDD_HOME>/reportes/llave` (ed25519, 0600, se crea la primera
   vez). Ningún dato de la persona va en la llave.
5. **Guarda** lo enviado en `<QDD_HOME>/reportes/enviados/` con el id que devuelve el relay, para que la persona sepa qué
   mandó.

### Contrato con el relay (lo implementa qdd-movile)

`POST <relay>/functions/v1/reportar`, con `apikey: <clave anónima del relay>`:

```json
{
  "formato": 1,
  "instalacion": "<llave pública ed25519, base64>",
  "en": "2026-11-08T12:00:00Z",
  "tipo": "bug",
  "titulo": "≤ 120 caracteres",
  "detalle": "≤ 6000 caracteres",
  "version": "2.10.0-rc.41",
  "so": "linux/amd64",
  "firma": "<ed25519 de la llave sobre «formato|instalacion|en|tipo|titulo|detalle|version|so», base64>"
}
```

- **El relay rechaza** con 400 una firma que no valida, un `en` a más de 10 minutos de su hora, un campo de más o un
  tamaño pasado.
- **Topes:** 5 reportes por día y por instalación, y 100 por hora en total. Pasado el tope responde 429 `{"error":"tope"}`.
- **Si lo acepta,** responde 201 `{"id":"REP-…"}`. La misma firma otra vez responde 200 con el mismo id, sin guardarlo de nuevo ni contarlo para el tope.
- **Guarda** en una tabla propia: id, huella de la instalación (sha256 de la llave, no la llave), tipo, título, detalle,
  versión, so y cuándo llegó. No guarda la IP.

`GET <relay>/functions/v1/reportes?desde=<id>` devuelve los reportes **solo a una instalación lectora**: la llave
pública del PC de quien mantiene QDD, que la persona agrega a un secreto del relay (`REPORTES_LECTORES`). El pedido va
firmado con esa llave, sobre «reportes|desde|en», con las cabeceras `x-qdd-instalacion`, `x-qdd-en` y `x-qdd-firma`.
Devuelve hasta 100 por vez, en orden; 403 `{"error":"no-lector"}` si la llave no está en el secreto. Lo implementó
qdd-movile (commit 06c5eb8); la llave lectora sale de `qdd reportar llave`.

### En el repositorio de QDD: una fuente más de la cola

El ciclo de mejoras suma la fuente **relay**. En el PC lector lee `GET reportes` y pasa cada uno a la cola como
reporte con `fuente: relay`, `ref: relay:REP-…` y `autor: QDD <versión> · <so>` (el relay no entrega la huella). Como
cualquiera puede enviar, entra `por_admitir` y no es tarea: la persona lo admite con un pedido 📮 que solo ella confirma
(`qdd mejoras admitir MEJ-NNN`, 2 a la vez), igual que la etiqueta «mejora» de un issue de un desconocido. Solo el
repositorio de QDD lee el relay; sin llave lectora el ciclo avisa cómo serlo.

## Fases

1. **PC:** `qdd reportar` (limpieza, vista previa, confirmación, pedido `DePersona`, llave, firma, envío a un relay
   falso en las pruebas).
2. **Relay (qdd-movile):** `reportar`, `reportes`, la tabla, los topes y `REPORTES_LECTORES`.
3. **Cola:** la fuente relay en `qdd mejoras ciclo`.
4. **Panel y teléfono:** un botón «📮 Reportar a QDD» en el panel y la pantalla en la app.

## Lo que no hace

- No envía nada sin el «sí» de la persona. Los incidentes automáticos siguen yendo a GitHub como hoy, para quien lo tenga.
- No identifica a la persona: la llave es de la instalación y el relay guarda solo su huella. Si quiere respuesta, puede
  escribir un contacto en el detalle, y lo verá en la vista previa antes de enviarlo.
- Un tope por llave no frena a quien cree muchas llaves. El tope global por hora pone el límite. Si hay abuso, el
  relay puede pedir más adelante una prueba de trabajo, sin cambiar el contrato del PC.
