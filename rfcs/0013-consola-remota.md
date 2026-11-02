# RFC 0013: Consola remota (QDD Móvil)

- Estado: aceptado (plan del usuario, 2026-11-02)
- Versión: qdd 2.9.0, canal preview (rc.1 → rc.4)
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
   ejecute. El PC vuelve a calcular la acción antes de ejecutarla (`qdd consola accion <id> --confirmado`).
2. Toda acción que escribe o ejecuta pasa por la confirmación de la persona en el teléfono (biometría o PIN).
3. Cada acción tiene un **id de solicitud único** y es idempotente: si llega dos veces, se ejecuta una sola vez.
4. Cada acción recibe su respuesta: `recibida` → `ejecutando` → `hecha` | `fallo`. Sin `recibida` en 5 s, el teléfono
   reintenta con el mismo id.
5. Quién aprobó qué, cuándo y desde qué dispositivo queda en `<QDD_HOME>/remoto/log.jsonl`, nunca en Supabase.
6. La consola se transmite solo mientras alguien la mira (Presence de espectadores).
7. La consola es de solo lectura.
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
| `qdd_remoto_mi_plan()` | teléfono | Plan, límites y horas de consola usadas. |
| `qdd_remoto_dispositivos(p_topico, p_firma)` · `qdd_remoto_revocar(p_topico, p_firma, p_dispositivo)` | PC | `qdd remoto dispositivos` y `revocar`. |
| `qdd_remoto_latido(p_topico, p_firma, p_en_curso, p_consola_min, p_mensajes)` | PC | Cada minuto: latido, consumo; responde el plan y si la consola sigue permitida. |
| `qdd_remoto_desactivar(p_topico, p_firma)` | PC | `qdd remoto desactivar`. |

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
| `delta` | PC → teléfono | `{seq, proyectos: {<clave>: panel | null}}`: los proyectos cuyo panel cambió (null: salió). Si falta un `seq`, el teléfono pide `sync`. |
| `accion` | relay → PC | El teléfono llama a `qdd_remoto_accion(maquina, solicitud, proyecto, accion_id)`; el relay sella `dispositivo`, verifica que no esté revocado y lo envía con `realtime.send` y `firma = hmac_sha256(firma_maquina, solicitud\|proyecto\|accion_id\|dispositivo\|en)`. El PC descarta lo que no traiga una firma válida. |
| `accion_estado` | PC → teléfono | `{solicitud, estado: recibida/ejecutando/hecha/fallo/ya_aprobado/rechazada, resultado, lugar}`. |
| `consola` | PC → teléfono | Fragmentos de la conversación o de un pedido, solo con espectadores. |

Presence: el PC publica `{rol: "pc", version_qdd, desde}`; cada teléfono `{rol: "movil", dispositivo, mirando}`. Cada
mensaje lleva `version`; un cliente que no la entiende lo dice y no adivina.

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

Escribir en la consola, guardar estado en la nube, cifrado de extremo a extremo (siguiente versión).

## Modelo de negocio

Suscripción de USD 1 al mes por usuario (neto ≈ USD 0,71 tras IVA y tienda); ≈ 47 usuarios cubren los costos fijos
(Supabase Pro, Apple, Google). `qdd remoto` sigue siendo MIT; se cobra el relay y la app. Requisitos de tienda: compra
integrada, PC de demostración para la revisión, borrar la cuenta desde la app, política de privacidad, D-U-N-S para Apple.

## Preguntas abiertas

1. ¿«¿Cómo vamos?» desde el teléfono debe llegar a la conversación de Claude Code, o basta con ver la tabla?
2. **Usuarios compartidos.** En el relay de desarrollo, el Auth es el del notificador: una cuenta de QDD Móvil es una
   cuenta de ese proyecto. Vale para desarrollar; para las tiendas hace falta el proyecto propio.
