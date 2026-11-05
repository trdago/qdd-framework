# RFC 0013: Consola remota (QDD Móvil)

- Estado: aceptado (plan del usuario, 2026-11-02)
- Versión: preview desde qdd 2.8.0-rc.19 (las fases rc.1 → rc.4 de abajo salieron juntas); estable con la siguiente versión que cumpla VER-05
- Nombres: app Flutter **QDD Móvil** (`qdd-movil`, hoy en `~/Documents/proyectos/qdd-movile`), subcomando
  **`qdd remoto`** (`pkg/remoto`), blueprint **`consola-remota`** (cuando la implementación esté probada)

## Problema

El panel de QDD (`qdd-tareas@qdd`) solo existe al lado de la conversación, en el PC. Fuera del escritorio no hay forma de:

1. ver qué falta para cumplir el objetivo en foco de cada proyecto;
2. aprobar los pedidos y las fichas en borrador que esperan a la persona;
3. ver qué está haciendo la IA en este momento.

La persona trabaja en varios proyectos y en más de un PC. Necesita una vista móvil de **todos** ellos, en tiempo real.

## Lo que el usuario ya decidió

- **El PC encendido es la fuente de verdad.** Un PC apagado se muestra «apagado desde HH:MM» y nada más: nunca un estado
  guardado o congelado.
- **Supabase solo como relay**: Realtime **Broadcast** (estado, acciones, consola) y **Presence** (qué PC está encendido
  y quién mira una consola). No se guardan en Supabase ni el estado ni las acciones.
- **Un solo canal por PC**, manejado por un único proceso de qdd en Go (`qdd remoto`), que publica todos los proyectos.
- **La app es un cliente más del protocolo**: el panel, la app y cualquier cliente futuro hablan lo mismo.
- **Relay de desarrollo (2026-11-02):** mientras no exista el proyecto Supabase Pro propio, se usa el proyecto Supabase
  del notificador de ofertas, con todo lo nuestro en un **esquema nuevo `qdd_remoto`**. Las políticas sobre
  `realtime.messages` se limitan a los temas `qdd:pc:%`, así que no cambian el acceso de los demás proyectos.
- **Planes, push, secretos, dispositivos, idiomas (complementos del usuario, 2026-11-02):** ver las secciones de abajo.
  El backend vive en `qdd-movil/supabase/` (migraciones, RLS, Edge Functions, límites por plan) y se despliega con la CLI
  de Supabase desde CI; nada se configura a mano en el panel.
- **Materializar también en la app (2026-11-02):** lo que cambia en la app aparece con el mismo efecto que el panel
  (glifos en el acento, bloques que se afinan, fundido; 0,8 s al aparecer, 0,4 s al irse; solo lo mínimo que cambió).

## Invariantes

1. El teléfono solo envía **ids de acción** que el PC ofreció en `qdd consola panel --json`. Nunca texto libre que se
   ejecute. El PC vuelve a calcular la acción antes de ejecutarla (`qdd consola accion <id> --confirmado`). La única
   excepción es **solo para Pro** (decisión del usuario, 2026-11-03): «Ejecutar desde el teléfono», un comando qdd de
   la lista que permite el PC, sin shell, o una instrucción para la IA, que llega como un mensaje de la persona.
2. Toda acción que escribe o ejecuta pasa por la confirmación de la persona en el teléfono (biometría o PIN).
3. Cada acción tiene un **id de solicitud único** y es idempotente: si llega dos veces, se ejecuta una sola vez.
4. Cada acción recibe su respuesta: `recibida` → `ejecutando` → `hecha` | `fallo`. Sin `recibida` en 5 s, el teléfono
   reintenta con el mismo id.
5. Quién aprobó qué, cuándo y desde qué dispositivo queda en `<QDD_HOME>/remoto/log.jsonl`, nunca en Supabase.
6. La consola se transmite solo mientras alguien la mira (Presence de espectadores).
7. La consola es de solo lectura. **Solo con Pro** (decisión del usuario, 2026-11-03) el teléfono puede mandar a un
   proyecto un comando qdd de la lista permitida y una instrucción para la IA de la sesión abierta (ver «Ejecutar desde
   el teléfono»); la instrucción entra a la conversación como un mensaje de la persona, nunca como un comando.
8. Un PC sin conexión nunca aparece «encendido»: lo decide Presence, no el último mensaje.
9. Las credenciales del PC viven en `<QDD_HOME>/remoto/` con permisos `0600`. El PC y la app nunca usan la service key
   (la clave secreta se usa solo para aplicar las migraciones del relay); nada de credenciales en un repositorio.
10. El tráfico remoto no cuenta como uso en `qdd uso` (RFC 0010).
11. Todo el código Go respeta las reglas del repositorio (cero `else`).
12. Sin teléfono conectado, el PC no publica nada más que su Presence.
13. Los deltas se agrupan cada 300 ms y la consola manda como mucho 2 mensajes por segundo, juntando líneas.
14. Cupo por usuario: el relay mide las horas de consola y los mensajes por usuario al mes. Al pasar cualquiera de los
    dos, se pausa solo la consola; el estado y las aprobaciones siguen.
15. Los límites del plan (PCs, proyectos, horas de consola) los aplica el relay, no la app. Sin Pro vigente la cuenta
    vuelve a Gratis y nunca pierde el estado ni las aprobaciones.
16. **Nada que parezca un secreto sale del PC.** Antes de enviar la consola (y cualquier `resultado` de una acción), el PC
    reemplaza claves de API, tokens, contraseñas, valores de `.env` y cadenas de conexión por «•••». Pruebas con los
    patrones conocidos (`sk-`, `ghp_`, `AKIA`, `eyJ…`, `postgres://…`, `-----BEGIN … PRIVATE KEY-----`, etc.).
17. El push solo dice qué espera y de qué proyecto; nunca el comando ni la salida.
18. Una acción solo vale si la reenvió el relay con la firma de la máquina; un teléfono revocado ve su próxima acción
    rechazada aunque tenga la sesión abierta.
19. Un pedido ya aprobado en otro lugar (el PC u otro teléfono) responde `ya_aprobado` con el lugar y no se ejecuta otra
    vez (protección de la cola, FND-070).

## Diseño

```
[PC: qdd remoto] --WebSocket--> [Supabase Realtime] <--WebSocket-- [QDD Móvil (Flutter)]
  ├─ registro de proyectos (<QDD_HOME>/proyectos.json)
  ├─ un watcher para todos los proyectos
  ├─ arma panel --json por proyecto y calcula deltas
  ├─ ejecuta acciones vía pkg/consola
  └─ transmite la consola si hay espectadores
```

### Registro de proyectos (rc.1)

`<QDD_HOME>/proyectos.json`, permisos `0600`:

```json
{"formato": 1, "proyectos": [{"ruta": "/abs/proyecto", "clave": "proyecto-a45873ff129c", "visto": "RFC3339"}]}
```

- `ruta` es `estado.ClaveLocal` del proyecto: un worktree se registra como su checkout principal.
- `clave` es el nombre de los archivos locales del proyecto (`RutaLocal` sin carpeta ni extensión): es la misma en todo
  QDD y no expone la ruta.
- Se anota en `qdd init` y en cada `qdd consola ciclo` (cada mensaje de la persona), sin escribir si ya está y se vio
  hace menos de una hora: el ciclo corre en cada mensaje y no debe costar más que leer un archivo.
- Escritura atómica (tmp + rename) bajo bloqueo, porque varias sesiones anotan a la vez.
- `qdd doctor` señala las rutas que ya no existen (`REM-01`) y `--fix` las quita.

### `qdd remoto estado [--json]` (rc.1)

Lo que el PC publicaría, sin red. El mismo JSON es el contenido del evento `estado`:

```json
{
  "version": 1,
  "qdd": "v2.9.0-rc.1",
  "maquina": {"nombre": "<hostname>", "so": "linux"},
  "en": "RFC3339",
  "proyectos": [
    {"clave": "qdd-a45873ff129c", "nombre": "qdd", "ruta": "/abs/qdd", "panel": { "...": "qdd consola panel --json" }},
    {"clave": "viejo-0123456789ab", "nombre": "viejo", "ruta": "/abs/viejo", "error": "la carpeta ya no existe"}
  ]
}
```

- `panel` es exactamente `consola.Panel(root, now)`: la app dibuja lo mismo que el panel y no hay un segundo modelo.
- Un proyecto que falla trae `error` y no `panel`; nunca un panel inventado o vacío como si estuviera bien.
- Orden: por `visto`, el más reciente primero.
- Sin `--json` imprime una línea por proyecto: nombre, foco, tareas, esperan, fallan.

### Relay (rc.2)

Esquema `qdd_remoto` (migraciones en `qdd-movil/supabase/migrations/`). Las tablas no se exponen: todo pasa por
funciones `public.qdd_remoto_*` (`security definer`), porque el API del relay compartido solo expone esquemas que ya
existen y el nuestro no se configura a mano.

- `maquinas` (`id`, `user_id`, `nombre`, `topico`, `firma`, `creada`, `latido`, `en_curso`): solo para autorizar
  canales y firmar acciones.
- `ofertas`: el emparejamiento en curso, 5 minutos, un solo uso.
- `dispositivos`, `planes`, `suscripciones`, `uso_mensual`, `config` (interruptor global y presupuesto de la consola
  gratis).
- **El canal es una capacidad (decisión 2026-11-02, autonomía del usuario).** El PC genera un secreto de 128 bits y su
  canal es `qdd:pc:<maquina_id>:<secreto>`. Las políticas de `realtime.messages`, limitadas a `qdd:pc:%`:
  - `authenticated` (el teléfono): se une si la máquina es suya y su dispositivo no está revocado.
  - `anon` (el PC, con la clave publicable): se une si existe una máquina con ese canal.
- El PC no tiene usuario, contraseña ni service key: `<QDD_HOME>/remoto/credenciales.json` (0600) guarda el id, el canal
  y la `firma` (otros 256 bits que solo conocen el PC y el relay; nunca se le entregan a un teléfono). Las funciones del
  PC exigen canal y firma.
- Revocar la máquina es borrar la fila: el canal deja de admitir a nadie.

| Función | Quién | Para qué |
|---|---|---|
| `qdd_remoto_ofrecer(p_codigo_hash, p_nombre, p_topico, p_firma)` | PC | Abre una oferta de emparejamiento (5 min). |
| `qdd_remoto_emparejada(p_codigo_hash)` | PC | El id de la máquina cuando el teléfono aceptó; cierra la oferta. |
| `qdd_remoto_aceptar(p_codigo)` | teléfono | Crea la máquina del usuario (respeta el límite de PCs del plan). |
| `qdd_remoto_mis_maquinas()` · `qdd_remoto_olvidar(p_maquina)` | teléfono | Lista y «Quitar este PC». |
| `qdd_remoto_registrar_dispositivo(p_nombre, p_plataforma, p_push_token)` | teléfono | Registra el teléfono (atado a su sesión). |
| `qdd_remoto_mis_dispositivos()` · `qdd_remoto_revocar_dispositivo(p_dispositivo)` | teléfono | Lista y revoca. |
| `qdd_remoto_accion(p_maquina, p_solicitud, p_proyecto, p_accion_id)` | teléfono | Sella, verifica y reenvía la acción firmada. |
| `qdd_remoto_responder(p_maquina, p_solicitud, p_proyecto, p_pregunta, p_respuesta)` | teléfono | Sella, verifica y reenvía la respuesta firmada a una pregunta de la consola. |
| `qdd_remoto_ejecutar(p_maquina, p_solicitud, p_proyecto, p_tipo, p_texto)` | teléfono | Solo Pro: sella, verifica y reenvía firmado lo que se manda a ejecutar (ver «Ejecutar desde el teléfono»). |
| `qdd_remoto_mi_plan()` | teléfono | Plan, límites y horas de consola usadas. |
| `qdd_remoto_dispositivos(p_topico, p_firma)` · `qdd_remoto_revocar(p_topico, p_firma, p_dispositivo)` | PC | `qdd remoto dispositivos` y `revocar`. |
| `qdd_remoto_latido(p_topico, p_firma, p_en_curso, p_consola_min, p_mensajes)` | PC | Cada minuto: latido, consumo; responde el plan y si la consola sigue permitida. |
| `qdd_remoto_desactivar(p_topico, p_firma)` | PC | `qdd remoto desactivar`. |

- **Login (issue #102):** `qdd login` es `qdd remoto activar` sin banderas: usa el relay guardado o, la primera vez, el que trae qdd. El PC muestra el QR y necesita hablar con el relay antes del escaneo, así que la app no puede entregárselo en el QR. Desde 2.9.1 qdd trae ese relay, solo con su clave publicable.
- **Emparejar:** `qdd remoto emparejar` genera un código de un solo uso, llama a `qdd_remoto_ofrecer` y muestra un QR con
  `{"v":1,"codigo","topico","nombre"}`. El teléfono llama a `qdd_remoto_aceptar(codigo)`. El PC consulta
  `qdd_remoto_emparejada` hasta ver el id. Un código usado o vencido no sirve.
- El secreto viaja en el nombre del canal: Supabase lo ve, como ve el contenido (el cifrado de extremo a extremo queda
  para la siguiente versión).

### Protocolo (canal `qdd:pc:<maquina_id>`)

| Evento | Dirección | Contenido |
|---|---|---|
| `sync` | teléfono → PC | Pide el estado completo. |
| `estado` | PC → teléfono | El JSON de `qdd remoto estado --json`, con `seq`. |
| `delta` | PC → teléfono | `{seq, proyectos: {<clave>: panel | null}, limites?}`: los proyectos cuyo panel cambió (null: salió) y los límites de Claude si cambiaron. Si falta un `seq`, el teléfono pide `sync`. |
| `accion` | relay → PC | El teléfono llama a `qdd_remoto_accion(maquina, solicitud, proyecto, accion_id)`; el relay sella `dispositivo`, verifica que no esté revocado y lo envía con `realtime.send` y `firma = hmac_sha256(firma_maquina, solicitud\|proyecto\|accion_id\|dispositivo\|en)`. El PC descarta lo que no traiga una firma válida. |
| `responder` | relay → PC | El teléfono responde una pregunta de la consola con `qdd_remoto_responder`; el relay lo firma como `accion`, con la respuesta entre la pregunta y el dispositivo (ver «Preguntas desde el teléfono»). |
| `ejecutar` | relay → PC | Solo Pro: el teléfono llama a `qdd_remoto_ejecutar`; el relay lo firma como `responder`, con el tipo y el texto (ver «Ejecutar desde el teléfono»). |
| `accion_estado` | PC → teléfono | `{solicitud, estado: recibida/ejecutando/hecha/fallo/ya_aprobado/rechazada/vencida, resultado, lugar}`. |
| `consola` | PC → teléfono | Fragmentos de la conversación o de un pedido, solo con espectadores. |

Presence: el PC publica `{rol: "pc", version_qdd, desde}`; cada teléfono `{rol: "movil", dispositivo, mirando}`. Cada
mensaje lleva `version`; un cliente que no la entiende lo dice y no adivina.


**Límites de Claude de la máquina** (pedido de qdd-movile, 2026-11-04). Campo opcional `limites` de `estado` y `delta`, sin
subir `version`. Sirve para ver cuánto queda sin gastar tokens:

```json
"limites": {
  "cinco_horas": {"usado": 12.2, "reinicia": "2026-11-04T18:30:00-03:00"},
  "semana": {"usado": 40, "reinicia": "2026-11-09T09:00:00-03:00"},
  "actualizado": "2026-11-04T14:02:11-03:00"
}
```

- `usado` es el `used_percentage` que Claude Code entrega a la barra, tal cual.
- Un tramo o un `reinicia` que no vino se omite, nunca un 0 inventado. Sin ningún tramo, no viaja `limites`.
- `actualizado` es cuándo la barra leyó el dato. La barra solo corre con una sesión abierta, así que el dato puede quedar
  viejo, incluso con `reinicia` ya pasado: el teléfono dice de qué hora es.
- La barra lo deja en `<QDD_HOME>/claude/limites.json`: al cambiar, o cada minuto si no cambia.
- Un `delta` lo lleva solo si cambió `usado` o `reinicia`, no por `actualizado`. Sin el campo, el teléfono conserva el último
  valor.
- Es de la máquina, no de cada proyecto: no lleva rutas ni el modelo.

### App (`qdd-movil`, Flutter)

- `supabase_flutter`, `flutter_riverpod`, `local_auth`, `mobile_scanner`, `go_router`.
- Pantallas: Mis proyectos, Proyecto, Para ti, Consola (solo lectura), Emparejar PC, Inicio de sesión.
- Estilo: fondo `#0F1113`, superficie `#181B1F`, borde `#2A2F35`, texto `#ECEAE6`, secundario `#9AA3AD`; ámbar `#F2B544`
  solo para lo que espera a la persona, `#5CC8B5` para «● running», `#7BD88F` ✓, `#FF8A7A` ✕. Inter y JetBrains Mono.
- Materializar: un widget envuelve cada unidad mínima (tarjeta de proyecto, fila de tarea, tarjeta de pedido) y anima
  cuando cambia su huella (el JSON que dibuja), con los tiempos y el acento de `efectos` del panel.
- La app lee el mismo JSON que `qdd remoto estado --json`. Hasta rc.2 usa datos de ejemplo con ese formato.

### Dispositivos

- El teléfono se registra al iniciar sesión (`qdd_remoto_registrar_dispositivo`), atado a su sesión de Supabase
  (`session_id` del JWT).
- `qdd remoto dispositivos` lista los teléfonos del dueño; `qdd remoto revocar <id>` saca uno al instante: el relay
  rechaza su próxima acción y el PC lo agrega a `<QDD_HOME>/remoto/revocados.json`.
- En la app: lista de PCs con «Quitar este PC» y de teléfonos con «Revocar».

### Push (primera versión)

- Avisos: pedido o ficha nueva en «Para ti», un pedido aprobado que terminó (bien o con falla), y un PC que se apaga con
  tareas en curso.
- FCM (Android) y APNs (iOS). El PC llama a la Edge Function `avisar` con un tipo cerrado (`pedido`, `ficha`,
  `termino_bien`, `termino_falla`) y la clave del proyecto: no acepta texto libre, así que nunca lleva un comando ni una
  salida. No pasa por Realtime, no consume mensajes.
- «PC apagado con tareas en curso» no lo puede avisar el PC: el PC da un latido por RPC cada minuto con cuántas tareas
  tiene en curso, y un trabajo de `pg_cron` avisa cuando el latido se corta con tareas en curso.

### Preguntas desde el teléfono (2026-11-03)

Cuando la consola de Claude Code espera a la persona, la pregunta aparece en la app y se puede responder desde ahí o
desde la terminal: gana la primera respuesta.

**Tipos.**

- **(a) Preguntas de la IA:** `AskUserQuestion`, con opción única, varias o «Otra» (texto libre, hasta 500 caracteres).
- **(b) Permisos de herramientas:** desde el teléfono solo hay **permitir una vez** o **negar**. Nunca «permitir siempre» ni
  cambios de reglas: la decisión no lleva `updatedPermissions`.
- **(c) Preguntas interactivas de los comandos de qdd:** hoy ningún comando de qdd pregunta sin una terminal. Las
  preguntas de diseño de `qdd_blueprint_apply` llegan a la IA, que las hace con `AskUserQuestion`, así que entran como
  (a). Para un comando futuro que pregunte por su cuenta, el diseño ya existe: `qdd consola pregunta registrar` con
  `origen:"qdd"`, `etiqueta` y `preguntas` explícitas, y después `qdd consola pregunta esperar <id>` hasta tener la
  respuesta. Ningún comando lo usa todavía.
- **(d) Lo que el estado espera de la persona** (pedido de sfrc, 2026-11-03): cada ⏳ cuyo «de quién» es la persona,
  en una tarea abierta, es una pregunta `origen:"estado"` de respuesta libre: una sola pregunta `q1`, con la tarea y la
  etapa como `encabezado`, lo que espera como `texto`, `otra:true` y sin opciones. Se responde con
  `{"q1":{"otra":"…"}}`. No se guarda una copia: sale del estado cada vez que se lee la cola, con un id
  `espera-<hash de tarea, etapa y bloqueo>` que es el mismo en cada lectura, y desaparece cuando la celda cambia. La
  respuesta del teléfono le llega a la IA del proyecto como una instrucción del teléfono (la misma cola que «Ejecutar
  desde el teléfono»), que dice a qué ⏳ responde. La cola la recuerda 24 h como respondida para no volver a
  preguntarla; si la IA no cambió la celda en ese tiempo, se pregunta de nuevo. Una respuesta a una espera que la celda
  ya no tiene da `vencida`.
  Solo es pregunta el ⏳ que pregunta algo (MEJ-004, qdd-movile, 2026-11-03: de 11 publicados, casi todos eran notas de
  avance): su bloqueo tiene al menos una frase entre `¿` y `?`, y `texto` es solo esas frases, sin el historial de la
  tarea. Tampoco lo es si nombra con `[tecla]` un pedido abierto de la cola, que ya lo cubre en el panel.
  Sin una sesión de Claude Code abierta en el proyecto (`ejecutar.ia=false`) no se publica y su respuesta da `vencida`,
  porque nadie podría resolverlo y la respuesta se perdería (FND-124). Un ⏳ nunca genera push: solo los pedidos y las
  fichas nuevas.

**Cómo se engancha.** El mod `qdd-tareas` escucha `PermissionRequest`, que Claude Code corre **al mismo tiempo** que
muestra el diálogo en la terminal. El primero que responde se queda con la decisión, y la respuesta del hook cierra el
diálogo.

1. El hook deja pasar primero los hooks de la persona: si alguno ya decidió, no pregunta al teléfono.
2. Si la sesión es interactiva, es del hilo principal (no de un subagente) y el PC tiene QDD Móvil activado, registra la
   pregunta con `qdd consola pregunta registrar` y avisa en la terminal «📱 También se puede responder desde QDD Móvil».
3. Espera con `qdd consola pregunta esperar <id> --segundos 10` en un bucle. Cada espera es un latido.
4. Con la respuesta del teléfono:
   - para (a), devuelve `allow` con `updatedInput.answers`;
   - para (b), devuelve `allow` (sin reglas) o `deny` con «La persona lo negó desde <teléfono> (QDD Móvil)».
5. Si la persona responde en la terminal, la herramienta sigue adelante. El mod ve terminar la llamada y cierra la
   pregunta con `qdd consola pregunta cerrar <id>` («respondida en el PC»).

**Cola local.** Las preguntas viven en `<QDD_HOME>/claude/preguntas/<proyecto>.json`, con candado y escritura atómica.
`qdd consola panel --json --con pregunta` las muestra primero, como acciones `tipo:"pregunta"`:

```
{ "id":"pregunta:<id>", "tipo":"pregunta", "origen":"ia|permiso|qdd|estado", "sesion", "etiqueta"(≤60),
  "estado":"pendiente|respondida|vencida", "respondida_en":"pc|telefono", "confirmar":true,
  "preguntas":[{"id":"q1","encabezado","texto","multiple","otra","opciones":[{"valor","etiqueta","descripcion"}]}],
  "permiso":{"herramienta","detalle"} }
```

- `permiso` va solo con `origen:"permiso"`, y entonces las opciones son exactamente `permitir` y `negar`.
- `qdd consola accion` no acepta una pregunta: se responde, no se ejecuta.
- `qdd remoto` las publica pasadas por el ocultador de secretos (etiqueta, detalle, textos, opciones y el detalle del
  permiso).

**Relay.** El teléfono llama a
`qdd_remoto_responder(p_maquina, p_solicitud, p_proyecto, p_pregunta, p_respuesta jsonb)`.

- `p_respuesta` es `{"<qid>": {"opciones":[…]} | {"otra":"texto"}}`, de 4096 bytes como máximo.
- `p_pregunta` puede ser el id solo o `pregunta:<id>`.
- El relay hace las mismas comprobaciones que `qdd_remoto_accion` (máquina del usuario, teléfono no revocado) y envía el
  evento `responder` con `{version, solicitud, proyecto, pregunta, respuesta, dispositivo, nombre, en, firma}`:
  - `respuesta` es el texto de `p_respuesta::text`, para que la firma se calcule sobre los mismos bytes;
  - `firma = hmac_sha256(firma_maquina, solicitud|proyecto|pregunta|respuesta|dispositivo|en)`, con
    `en = YYYY-MM-DDTHH:MM:SSZ`.
- Como las aprobaciones, no mira el plan ni el interruptor de la consola (inv. 14).

**Respuesta del PC.** El PC verifica la firma y la vigencia (±5 min) y contesta con `accion_estado`
`{solicitud, estado, resultado, lugar}`, una sola vez por `solicitud`:

| Estado | Cuándo | `resultado` / `lugar` |
|---|---|---|
| `hecha` | La respuesta quedó entregada a la consola. | «respuesta entregada a la consola del PC» |
| `ya_aprobado` | Ya la respondió la terminal u otro teléfono. | `lugar` = «PC» o el nombre del teléfono |
| `vencida` | La pregunta venció o ya no está en el PC. | El motivo |
| `rechazada` | Respuesta inválida, teléfono revocado, proyecto no publicado. | El motivo |

- La respuesta a un ⏳ del estado (id `espera-…`) no se cierra al recibirla (FND-141): pasa por `ejecutando` y termina
  `hecha` cuando una sesión de Claude Code la acepta, o `fallo`, `vencida` o `rechazada` si no llegó. En esos tres casos el
  ⏳ vuelve a preguntarse con el mismo id y una solicitud nueva.
- Si la terminal gana después de que el PC dijo `hecha` (la respuesta del teléfono llegó pero el motor ya había
  aceptado la de la terminal), el PC corrige esa solicitud con `ya_aprobado` y `lugar:"PC"`.
- Una respuesta sin firma válida se anota en el log y no se contesta.

**Vencimiento.** Una pregunta vence cuando:

- deja de recibir latidos de `esperar` por 60 s (el hook terminó por su tiempo límite o Claude Code lo canceló);
- pasan 30 min desde que se creó;
- termina la sesión de Claude Code (`qdd consola pregunta vencer --sesion`).

Una pregunta cerrada sigue visible 5 min como «ya respondida en …» o «venció». La cola guarda como máximo 20.

**Límites de la API de Claude Code.**

- Las sesiones no interactivas y los subagentes quedan fuera: hay subagentes que esperan a los hooks antes de mostrar el
  diálogo, y la pregunta bloquearía la terminal.
- Claude Code no avisa al hook cuando la persona responde el diálogo; el mod lo deduce cuando termina la llamada a la
  herramienta. Así, un permiso aceptado en la terminal para una herramienta larga (un `Bash` de minutos) deja la
  pregunta «pendiente» hasta que la herramienta termina. Si el teléfono responde en ese rato, recibe `hecha` aunque el
  motor ya no la use, y el PC lo corrige con `ya_aprobado` PC cuando la herramienta termina.
- La biometría para responder es cosa de la app.

**Pendiente.** No hay un tipo `pregunta` en la Edge Function `avisar`, porque sus tipos son cerrados. Hasta que se
agregue, la pregunta llega por Realtime con la app abierta y no hay aviso push.

### Ejecutar desde el teléfono (2026-11-03, solo Pro)

Con Pro, la persona manda desde la app (Proyecto → «Ejecutar», con biometría o PIN) un **comando qdd** o una
**instrucción para la IA** de un proyecto. Siempre a través de qdd: nunca un comando de shell.

**Relay.** `qdd_remoto_ejecutar(p_maquina, p_solicitud, p_proyecto, p_tipo, p_texto)` hace las comprobaciones de
`qdd_remoto_responder` y además exige Pro vigente (`53400` «requiere Pro»). Reenvía el evento `ejecutar` con
`{version: 1, solicitud, proyecto, tipo, texto, dispositivo, nombre, en, firma}`:

- `firma = hmac_sha256_hex(concat_ws('|', solicitud, proyecto, tipo, texto, dispositivo, en), firma_maquina)`, el mismo
  esquema de `responder`. `nombre` es solo para mostrar y no va firmado.
- `tipo` es `qdd` o `ia`. `qdd`: una línea de 1 a 1000 caracteres, sin la palabra `qdd` del principio y sin caracteres
  de control. `ia`: de 1 a 4000 caracteres; admite saltos de línea y tabulaciones, ningún otro control.
- El relay no guarda ni el comando ni su salida.

**PC.** El PC nunca confía en el relay:

1. Como una acción, descarta sin respuesta lo que no trae la firma de la máquina o es viejo (±5 min; queda en el log
   como `descartada`, inv. 18). No lo recuerda, así que no le quita la solicitud a la de verdad.
2. Aplica cada `solicitud` una sola vez (`solicitudes.json`); una repetida responde su último estado.
3. Rechaza (`rechazada` con el motivo) lo que llega de un teléfono revocado, de un proyecto que no publica o sin Pro.
   «Pro» es el plan del último latido (`qdd_remoto_latido`); hasta el primero se supone Gratis. El PC guarda ese plan en
   `<QDD_HOME>/remoto/plan.json` (0600) para `qdd consola panel` y `qdd remoto estado`, que corren en otro proceso.
4. Vuelve a validar el texto como el relay. Un comando qdd se parte sin shell: por espacios, con comillas simples o
   dobles. Se rechaza entero si trae ``; | & $ ` > < ( )``, un salto de línea o un carácter de control.
5. Compara el comando con la lista permitida. Solo comandos que leen o revisan, y solo banderas de solo lectura:

   | Comando | Acepta |
   |---|---|
   | `audit` | nada más |
   | `doctor` | nada más: `--fix` y `-f` reparan archivos |
   | `estado` | `--json`, `--alcance=abiertos` o `=todos`, `--formato=markdown` o `=mensajeria` |
   | `estado check` | `--json` |
   | `modules list` | nada más |
   | `blueprint list` | `--json` |
   | `blueprint quality [id]` | un id de blueprint (`^[a-z0-9][a-z0-9._-]{0,79}$`, sin rutas) y `--json` |

   `status` y `sprint status` no son comandos de la CLI: caen al manejador de intenciones, que solo dice que está
   deprecado. Por eso no están en la lista. Las banderas con valor van con `=`, y cualquier otra palabra se rechaza
   (`--dir`, `--respuesta`, `set`, `approve`…).
6. Corre el mismo binario de qdd (`os.Executable`) con `exec`, en la raíz del proyecto, sin entrada y con el entorno
   limpio de las variables que eligen repositorio (FND-117). Lo corta a los 10 min. Corre uno por proyecto a la vez:
   dos `audit` juntos se pisarían lo que escriben en `.qdd`.
7. Una instrucción queda en la cola del teléfono hasta que la toma una sesión interactiva de Claude Code del proyecto
   (ver abajo). Si ninguna la toma en 30 min, vence.

**Respuesta.** `accion_estado` `{solicitud, accion_id: "ejecutar", estado, resultado}`:

| Estado | Comando qdd | Instrucción |
|---|---|---|
| `recibida` | Llegó. | Llegó; sin sesión abierta, `resultado` dice que queda en cola. |
| `ejecutando` | Corre. | Hay una sesión abierta: va en camino. |
| `hecha` | Salió con 0. | La tomó la sesión. |
| `fallo` | Salió con otro código, se cortó o qdd remoto se cerró mientras corría. | La sesión no pudo dársela a la IA. |
| `vencida` | — | Ninguna sesión la tomó en 30 min. |
| `rechazada` | Teléfono revocado, proyecto desconocido, sin Pro, texto mal formado, comando fuera de la lista u otro comando del teléfono corriendo. | Lo mismo, o el teléfono la retiró. |

- La salida de un comando pasa entera por el ocultador de secretos (inv. 16). Al teléfono va su final: como mucho 4000
  caracteres, con «…» delante si se cortó.
- Se usa `fallo` y no `fallida`: es la fase que la app conoce.

**Cola del teléfono.** Lo mandado vive en `<QDD_HOME>/claude/telefono/<proyecto>.json` (`RutaLocal`, como los pedidos),
con candado y escritura atómica. Guarda las 10 más nuevas, y lo cerrado sale a los 30 min.

`qdd consola panel --json --con telefono` lo muestra como pedidos:

```
{"id": "telefono:<solicitud>", "tipo": "pedido", "origen": "telefono", "dispositivo": "<nombre del teléfono>",
 "solicitud", "etiqueta", "detalle", "estado", "resultado"}
```

- Estos pedidos no tienen tecla y no son acciones de `qdd consola accion`.
- Un comando sale `ejecutando`, `hecho` o `fallo`.
- Una instrucción sale `armado` mientras espera una sesión, `hecho` al entregarse y `fallo` si venció, se retiró o no
  se pudo entregar.
- El «Rechazar» de la app (`rechazar:telefono:<solicitud>`) retira una instrucción que todavía espera.
- El panel del PC no los recibe (no pide `--con telefono`). `qdd remoto` sí, pasados por el ocultador.

**`ejecutar` del panel.** Con QDD Móvil activado y Pro, el panel trae `ejecutar: {qdd: true, ia, sugeridos}`. Sin
eso, el campo no va:

- `ia` es `true` si una sesión de Claude Code del proyecto latió en los últimos 30 s.
- `sugeridos` es la lista permitida, sin la palabra `qdd`.

**Entrega a la IA.** En cada sesión interactiva, el mod `qdd-tareas` corre `qdd consola telefono latir --sesion <id>`
cada 5 s. Así el proyecto sabe que hay una sesión abierta, y el latido devuelve las instrucciones que le tocan:

- Cada instrucción va a una sola sesión, una sola vez, aunque haya varias abiertas: qdd lo decide con la cola bloqueada.
- El mod la envía con `prompt.submit` como un mensaje de la persona: «📱 La persona te escribe desde «<teléfono>» (QDD
  Móvil). Atiéndelo como un mensaje suyo:» y el texto tal cual. También avisa en el escritorio.
- El mensaje empieza con 📱, así que Claude Code nunca lo lee como un comando de shell (`!`) ni uno suyo (`/`). El mod
  no envía nada que no empiece así.
- Si Claude Code no acepta el mensaje, `qdd consola telefono fallo` la marca sin entregar.
- Las sesiones sin terminal y los subagentes no laten ni reciben nada.
- Sin QDD Móvil activado, el mod late cada minuto. Con un qdd anterior, deja de latir.
- Al cerrarse la sesión, `latir --fin` la quita en el acto.

### Planes y controles de costo

| | Gratis | Pro (USD 1 al mes) |
|---|---|---|
| PCs | 1 | Hasta 3 |
| Proyectos | 3 | Sin límite |
| Estado y aprobaciones | Sí | Sí |
| Consola en vivo | 3 h al mes | 10 h al mes (a validar en la beta) |

- La persona ve horas de consola, no mensajes; cada plan tiene además un tope interno de mensajes.
- El PC pide permiso de consola por RPC cada minuto mientras transmite e informa los minutos y mensajes usados; el relay
  responde si sigue permitida (plan, tope, presupuesto global de la consola gratis, interruptor global).
- Spend Cap de Supabase activo durante la beta; aviso al 80 % de los mensajes incluidos.
- **Gratis con más de 1 PC o 3 proyectos (decisión 2026-11-02, autonomía):** el PC publica los proyectos más recientes
  (por `visto`) hasta el límite, y la app dice cuántos quedaron fuera. Emparejar un segundo PC en Gratis se rechaza con el
  motivo y la oferta de Pro.
- Suscripción con RevenueCat (StoreKit y Google Play Billing); su webhook (Edge Function `revenuecat`) marca quién tiene
  Pro vigente.

### Idiomas

La app en español e inglés desde el principio, con las fichas de las tiendas en los dos.

## Fases (canal preview)

| Fase | Entrega | Hecha cuando |
|---|---|---|
| rc.1 | Registro y `qdd remoto estado --json`, sin red | Lista los proyectos del PC con su panel; doctor limpia las rutas muertas. |
| rc.2 | `activar`, `emparejar`, Presence, `sync`, `estado`, `delta` | La app (solo lectura) muestra en vivo 2 PCs y uno apagado aparece «apagado». |
| rc.3 | Acciones firmadas, log local, dispositivos y push | Aprobar desde el teléfono ejecuta una sola vez aunque llegue repetido, queda en el log; un teléfono revocado no puede. |
| rc.4 | Consola de solo lectura, sin secretos, con cupo | Se transmite solo con espectadores, sin secretos y se pausa al pasar el cupo. |
| 2.9.0 | Estable | ≥ 5 h de uso medido (VER-05). |

## Cómo se prueba

- Go: registro (anotar, sin duplicar, worktree = principal, rutas muertas), `remoto estado` (proyecto con y sin estado,
  carpeta borrada), deltas (secuencia y huecos), idempotencia de acciones y log.
- Integración: relay con un PC simulado y un cliente de prueba (`sync`, `delta`, `accion` repetida, PC que se cae).
- Flutter: pruebas de widgets de las pantallas con estados de ejemplo (PC apagado, sin pendientes, acción que falla).
- Seguridad: un usuario no se une al canal de un PC ajeno; un `accion_id` no ofrecido se rechaza; un código de
  emparejamiento usado o vencido no sirve.

## Fuera de alcance

Escribir en la consola (salvo, solo con Pro, lo de «Ejecutar desde el teléfono»: comandos qdd de la lista permitida e
instrucciones a la IA), cualquier comando que no pase por qdd, guardar estado en la nube, cifrado de extremo a extremo
(siguiente versión).

## Modelo de negocio

Suscripción de USD 1 al mes por usuario (neto ≈ USD 0,71 tras IVA y tienda); ≈ 47 usuarios cubren los costos fijos
(Supabase Pro, Apple, Google). `qdd remoto` sigue siendo MIT; se cobra el relay y la app. Requisitos de tienda: compra
integrada, PC de demostración para la revisión, borrar la cuenta desde la app, política de privacidad, D-U-N-S para Apple.

## Preguntas abiertas

1. ¿«¿Cómo vamos?» desde el teléfono debe llegar a la conversación de Claude Code, o basta con ver la tabla?
2. **Usuarios compartidos.** En el relay de desarrollo, el Auth es el del notificador: una cuenta de QDD Móvil es una
   cuenta de ese proyecto. Vale para desarrollar; para las tiendas hace falta el proyecto propio.

## Modo goal en el teléfono (RFC 0015)

El panel de cada proyecto trae `goal`, opcional: sin él, ese qdd no ofrece el modo y la app no muestra nada. No cambia la
versión del protocolo.

```json
"goal": {
  "activo": true,
  "estado": "trabajando",
  "motivo": "sin_consola",
  "desde": "2026-11-04T10:00:00Z",
  "recordatorios": 1,
  "elegidas": [{"en": "2026-11-04T10:02:00Z", "tipo": "pregunta", "etiqueta": "…", "eleccion": "…"}]
}
```

- `estado` es `trabajando`, `pausado` o `terminado`. `motivo` viene solo con `pausado`: `sin_consola`, `sin_avance`,
  `sin_limite` o `espera_persona`.
- `activo: false` dice que este qdd lo ofrece y está apagado.
- `elegidas` son las últimas 20, de la más nueva a la más vieja, ya sin secretos (inv. 16).
- `qdd_remoto_ejecutar` con tipo `qdd` acepta `goal activar` y `goal desactivar`, sin banderas. No están en `Sugeridos`: la
  app los ofrece con biometría.
- Dos avisos nuevos para `avisar`, con solo tipo y proyecto (inv. 17): `goal_pausado` y `goal_terminado`. Se piden cuando
  el modo llega a ese estado; la primera vez que `qdd remoto` ve un proyecto es la base. Un relay que no los acepta solo
  deja el aviso sin enviar.

