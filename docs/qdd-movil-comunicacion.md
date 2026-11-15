# Comunicación entre qdd y QDD Móvil: revisión del 2026-11-15

Pedido de la persona desde QDD Móvil: «revisa toda la comunicación con qdd-movil-play y crea un test para auto
diagnóstico y reparación usando qdd doctor». Se revisó el código del PC (`cli/pkg/remoto`, `cli/cmd/remoto*`) contra la
app (`lib/protocolo`, `lib/relay`, `lib/push` de qdd-movile) y las funciones del relay (migraciones y Edge Functions de
qdd-movile). Se verificó leyendo los dos lados, no suponiendo. Con el binario de la rama, la sonda contra el relay real
dio ida y vuelta en unos 450 ms con un teléfono mirando.

## Mapa de mensajes

| Mensaje | Dirección | Por dónde | Campos | Quién valida | Si falla |
|---|---|---|---|---|---|
| Presence del PC | PC → app | Realtime presence | `rol: pc`, `version_qdd`, `desde` | app (`PresenciaPc.buscar`) | la app muestra el PC «apagado» |
| Presence del teléfono | app → PC | Realtime presence | `rol: movil`, `dispositivo`, `mirando`, `version_app?`, `capacidades?` (p. ej. `logrado_bajo_pedido`) | PC (`Telefonos`) | sin teléfonos el PC no publica (inv. 12) |
| `sync` | app → PC | broadcast | `version` | PC | el teléfono no recibe la base; reintenta |
| `estado` | PC → app | broadcast | `version`, `qdd`, `maquina`, `en`, `seq`, `fuera_del_plan`, `limites`, `diagnostico` (nuevo), `proyectos[]` | app (`exigirVersion`, `seq`, `proyectos`) | Realtime lo descarta sin aviso si pasa su tope (FND-209) |
| `delta` | PC → app | broadcast | `version`, `seq`, `proyectos{clave: panel\|null}`, `limites` | app | un `seq` que no sigue pide `sync` |
| `flujo` | PC → app | broadcast | `version`, `clave`, `flujo` (≤ 100 KB) | app | el flujo no se dibuja |
| `consola` | PC → app | broadcast | `version`, `proyecto`, `inicio`, `lineas[]`, `pausada` | app | se ve la consola incompleta |
| `accion_estado` | PC → app | broadcast | `version`, `solicitud`, `estado`, `resultado`, `lugar` | app (`_fases`) | la acción queda «sin respuesta» |
| `sincronizar` / `sincronizado` | app → PC / PC → app | broadcast | `version` / avisos por proyecto | PC / app | el teléfono no sabe qué quedó atrasado |
| `pedir_logrado` / `logrado` | app → PC / PC → app | broadcast | `{version, proyecto, solicitud}` / `{version, proyecto, solicitud, entradas, recortado?, error?}` | PC (clave `^[a-z0-9-]{1,80}$`, solicitud ≤ 64) | inválido se ignora; desconocido o fuera del plan, `[]` con `error`; ver «Logrado bajo pedido» |
| `accion` | app → relay → PC | RPC `qdd_remoto_accion` → `realtime.send` | `solicitud`, `proyecto`, `accion_id`, `dispositivo`, `dispositivo_nombre`, `en`, `firma` | relay (sesión, revocado, formato) y PC (firma HMAC, 5 min, id ofrecido) | `rechazada` o descartada al log |
| `responder` | app → relay → PC | RPC `qdd_remoto_responder` | `pregunta`, `respuesta` (texto ≤ 4096), `nombre`, `firma`… | relay y PC (contra la pregunta guardada) | `rechazada`, `vencida` o `ya_aprobado` |
| `ejecutar` | app → relay → PC | RPC `qdd_remoto_ejecutar` | `tipo` (qdd, ia, flujo), `texto`, `nombre`, `firma`… | relay (Pro, largo, caracteres) y PC (lista permitida, sin shell) | `rechazada` |
| `sonda` (nuevo) | PC → PC | broadcast con `self` | `version`, `marca` | PC | la sonda nombra el tramo `eco` |
| Latido | PC → relay | RPC `qdd_remoto_latido` | en curso, consola, mensajes | relay | doctor REM-07 |
| Push | PC → relay → FCM | Edge Function `avisar` | `topico`, `firma`, `tipo`, `proyecto`, `clave` | relay (`validarPedido`) | se loguea; nunca bloquea |
| Directorio | PC → sitio firmado | HTTPS | respuesta firmada | PC | sigue con el relay guardado (REM-11) |

Los nombres de eventos, los campos y la firma (`solicitud|proyecto|…|dispositivo|en`) coinciden en los tres lados.

## Desajustes encontrados

| # | Lado | Qué | Estado |
|---|---|---|---|
| 1 | PC | El relay y la app aceptan los push `pregunta` y `permiso` (y sus `_resuelto`), pero el PC nunca los pedía: una pregunta de la IA o un permiso de herramienta no sonaba en el teléfono. | Arreglado (FND-208), con el mismo cuerpo `{topico, firma, tipo, proyecto, clave}` |
| 1b | PC | Contra el contrato `esperas.json` de qdd-movile, un pedido sin estado, en espera de otro o que falló y se puede reintentar no pedía push. | Arreglado (FND-210); qdd prueba los 23 casos del contrato (`cli/pkg/remoto/testdata/esperas.json`) |
| 2 | PC | El estado completo de este PC pesa unos 170 KB tal como sale (JSON compacto), el 66 % de los 256 KB que Realtime acepta en el plan Free; pasado el tope se descarta sin aviso y nada lo medía. `qdd remoto estado --json` imprime con sangría y da unos 250 KB, que no es lo que viaja. | Medido y avisado en doctor REM-21 (FND-209). La salida de cada logro va recortada a 500 caracteres con «…» delante (antes 2000). Reducir cuántos logros o días van lo decide la persona. |
| 3 | PC | Nada probaba de punta a punta que lo que el PC publica llega: el broadcast va sin `ack`. | Sonda nueva (REM-20), automática en el servicio. |
| 4 | PC ↔ relay | La clave de un proyecto (`<carpeta>-<hash de 12>`) no tiene tope de largo, y el relay exige `^[a-z0-9-]{1,80}$` en acciones, respuestas, ejecutar y push: una carpeta de más de 67 caracteres deja ese proyecto sin acciones ni push desde el teléfono. | Pendiente: acortar la clave cambia la ruta de sus archivos locales (colas). Va como pregunta a la persona. |
| 5 | App | La app no muestra el nuevo campo `diagnostico` del estado (lo ignora, como todo campo desconocido). | Pendiente en qdd-movile: mostrar «el canal falló en X / se reparó» en la tarjeta del PC. |
| 6 | App | La presence del teléfono no dice la versión de la app, así que el PC y doctor no pueden saber si el teléfono entiende un mensaje nuevo. | Acordado con qdd-movile: la presence trae `version_app` y `capacidades` (opcionales); el PC ya los lee. |
| 7 | App | La app recibe el broadcast `sonda` (cuando está mirando) y lo ignora porque no se suscribe a ese evento: correcto. Cuenta como un mensaje entregado en la cuota de Realtime. | Sin cambio. |

## Autodiagnóstico y reparación

- **Sonda** (`qdd remoto sonda`, REM-20): entra al canal del PC con su propia conexión. Si hay un teléfono mirando, manda un broadcast con eco y espera que vuelva. Cada falla nombra su tramo: `relay`, `conexion`, `canal`, `realtime` o `eco`.
- **Automática**: con QDD Móvil activado, el servicio la corre al minuto de arrancar y luego cada hora. Con el canal rechazado compara el relay con su directorio firmado; si Realtime no responde o el eco no vuelve, se reinicia. Sin emparejar no corre.
- **Una sola política de reinicio**: con el canal propio del servicio caído, la caída es de REM-18 (el canal reintenta solo y `qdd doctor --fix` reinicia una vez), y la sonda no se suma. Si no, el servicio se reinicia por la sonda como mucho una vez cada 6 h, contando los reinicios que hizo doctor por el canal (todos quedan en `sonda.json`). En una pasada de `qdd doctor --fix`, REM-06, REM-07, REM-17, REM-18 y REM-20 reinician el servicio una sola vez entre todos.
- **Visible**: `qdd doctor` (REM-20 y REM-21, sin red), el campo `diagnostico` del estado para el teléfono (un día) y `qdd incidentes` cuando la falla es del relay y no de la red.

### El campo `diagnostico` del estado

`{ok, tramo, motivo, reparado?, en}`. Va solo si la última prueba falló o reparó algo en el último día. `en` va en
RFC3339 UTC. `motivo` es un código de una lista cerrada para que la app lo traduzca; el texto en español queda para
doctor y la CLI. Un código no se renombra: si hace falta otro, se agrega.

| `tramo` | `motivo` | Qué pasó |
|---|---|---|
| `relay` | `sin_relay` | El PC no tiene relay guardado o no está emparejado. |
| `conexion` | `sin_red` | No se llega al relay. |
| `canal` | `canal_rechazado` | El relay no deja entrar al canal del PC. |
| `realtime` | `realtime_sin_respuesta` | El relay acepta la conexión pero Realtime no responde. |
| `eco` | `eco_perdido` | Lo que el PC publica no vuelve por el relay. |
| `avisar` | `avisar_ausente` | La Edge Function de los push respondió 404: no está desplegada. |
| `avisar` | `avisar_atrasado` | Respondió 400 «tipo inválido»: es de antes de los push de preguntas y permisos. |
| `avisar` | `avisar_inesperado` | Respondió otra cosa que el 401 de un relay al día. |

La prueba de `avisar` (sugerencia de qdd-movile) es un POST por sonda, una vez por hora, con un tópico y una firma que
pasan la validación de formato pero no son de ninguna máquina, `tipo: pregunta` y `proyecto`/`clave` `sonda`. Nunca
lleva las credenciales de la máquina. Un relay al día responde 401 «credenciales de la máquina inválidas». El PC no
repara este tramo: lo despliega quien lleva el relay. Medido contra el relay real el 2026-11-15: al día.

`reparado` es `directorio` (se comparó el relay con su directorio firmado) o `reiniciar` (el servicio se reinició).

## Lo que revisa doctor del resto del canal

Lo agregó en paralelo el PR #263 (REM-15 a REM-19) y completa el mapa: la sonda prueba el canal de punta a punta, y estos ítems dicen por qué no arrancaría o no se uniría. Todos se diagnostican sin red; solo `--fix` consulta el directorio.

| Código | Qué mira | Qué hace `--fix` |
|---|---|---|
| REM-15 | El relay respondió `28000`: el teléfono quitó este PC (`quitado.json`). | Nada: pide `qdd remoto desactivar` y `qdd login`. |
| REM-16 | `credenciales.json` y `relay.json` se leen. | Aparta el roto; pide el relay al directorio firmado. |
| REM-17 | El servicio lanza un qdd que existe (otra versión de node lo borra). | Lo reescribe con el qdd de ahora. |
| REM-18 | El canal en tiempo real está unido (`conexion.json`), aunque el latido responda. | Reinicia el servicio si lleva más de 3 min caído. |
| REM-19 | Acciones del teléfono descartadas por la hora: desfase entre el PC y el teléfono. | Nada: la hora la corrige la persona. |
| REM-20 | La última sonda del servicio (`sonda.json`). | Sin sondear, aplica la reparación de su tramo (directorio o reinicio); el servicio la repite al arrancar. |
| REM-21 | El peso del último estado completo frente al tope de Realtime (FND-209). | Nada: reducirlo es una decisión. |
| REM-22 | El último aviso que el relay aceptó y no llegó a ningún teléfono (`push.json`, FND-217). | Nada: solo avisa; nunca reintenta ni bloquea. |

### Avisos que no llegan (REM-22, MEJ-027, 2026-11-15)

Contrato con qdd-movile (relay a1fbc9a): `avisar` responde 200 con `{enviados, destinos, motivo?}`. `destinos` son
los teléfonos de la máquina con push no revocados. `motivo` va solo con `enviados` 0: `sin_destinos` si `destinos`
es 0 (gana aunque falte FCM), `sin_fcm` si falta `FCM_CUENTA_SERVICIO` y `fcm_rechazo` si FCM no entregó ninguno.
Con `enviados` > 0 no va motivo aunque alguno falle.

- **PC**: lee hasta 4 KB de la respuesta; una más larga o sin `enviados` (relay anterior) se trata como antes.
- **Con `enviados` 0**: el servicio anota en `<QDD_HOME>/remoto/push.json` (0600) la hora, el tipo, el motivo y los
  destinos, sin el proyecto ni nada secreto. El registro del servicio dice lo mismo una vez por hora para cada tipo y
  proyecto. Un aviso posterior que sí llega borra el archivo y REM-22 vuelve a verde.
- **doctor** (sin red, solo aviso, `--fix` no hace nada) muestra la hora y la causa:

| Respuesta | Texto |
|---|---|
| `sin_destinos` o `destinos` 0 | ningún teléfono tiene los avisos activados (sin token de push) |
| `sin_fcm` | falta la configuración de FCM en el relay |
| `fcm_rechazo` | FCM no entregó: tokens vencidos o FCM caído |
| `destinos` > 0 sin motivo (relay intermedio, o un motivo nuevo) | FCM no entregó o falta su configuración en el relay |
| sin `destinos` | el último aviso no llegó a ningún teléfono |

- **Costo**: sin teléfono, ningún broadcast (solo el join). Con teléfono, 2 broadcasts (envío y eco) más la entrega a cada teléfono, cada hora: unos 72 mensajes al día con un teléfono conectado todo el día.

## Logrado bajo pedido (2026-11-15)

Acordado con qdd-movile. La opción la eligió el modo goal de la persona, con la recomendada. Es opcional por
capacidad, como goal: la versión del protocolo no sube.

- **Presence del teléfono**: puede traer `version_app` y `capacidades: ["logrado_bajo_pedido"]`.
- **Estado**: el panel de cada proyecto lleva siempre `logrado_bajo_pedido: true`, que anuncia que el PC sabe
  responder. Deja de llevar `logrado` solo cuando todos los teléfonos del canal declaran la capacidad. Si uno no la
  declara, lo lleva, con la salida recortada a 500. Cuando se suma un teléfono que no la declara (o cambia quién la
  declara), va un `estado` completo sin esperar al próximo cambio.
- **Pedido**: `pedir_logrado {version, proyecto, solicitud}` por el canal privado, como `sincronizar`. Se ignora si
  `proyecto` no cumple `^[a-z0-9-]{1,80}$` o si `solicitud` no tiene entre 1 y 64 caracteres `[A-Za-z0-9-]`.
- **Respuesta**: `logrado {version, proyecto, solicitud, entradas, recortado?, error?}`.
  - Las entradas siguen el contrato de siempre: como mucho 60, de los últimos 14 días, la más nueva primero, tipos
    pedido, tarea, objetivo o mensaje, salida de hasta 2000 caracteres con «…» y todo pasado por el ocultador.
  - Un proyecto desconocido da `[]` con `error: "proyecto"`; uno fuera del plan, `[]` con `error: "plan"`.
  - Si pasa 64 KB, quita lo más viejo hasta caber y pone `recortado: true`.
  - Se calcula como mucho una respuesta cada 5 s por proyecto; un pedido dentro de esa ventana recibe la misma, con
    su propia `solicitud`.
  - Solo responde con un teléfono mirando (inv. 12).
- **Peso medido**: con datos de prueba (3 proyectos con 60 logros cada uno), el estado pasa de 112.265 a 4.649 bytes.
  En este PC, con 11 proyectos, el estado compacto que sale por el canal pasa de unos 170 KB a unos 122 KB.
