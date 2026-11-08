# RFC 0025: toma de requerimientos sobre un flujo, con el cliente, un experto y un asistente

**Estado:** aceptado (2026-11-08), con las decisiones de la persona.

## Problema

Para afinar un proceso con un cliente, hoy hay que mostrarle el panel o una captura. El cliente no tiene Claude ni QDD:
no puede tocar el flujo ni dejar lo que piensa donde nosotros lo vemos. Lo que dice queda en un correo o en una reunión,
y alguien lo pasa a mano. Si además hay un experto que guía la conversación, sus notas técnicas quedan en su cuaderno.

La persona lo pidió así (2026-11-08):

- «para conversaciones con clientes que no tienen claude ni qdd podríamos tener una plantilla de nuestro flujo … para
  mandar en un archivo html a un usuario final … para conversar sobre un proceso, afinar cosas, levantar sus
  requerimientos … con elementos más didácticos … que reaccione en realtime y los capturemos también en realtime en
  nuestro tablero, pero todo se integra cuando nosotros lo aprobamos … el usuario podría decir cualquier cosa y no por
  eso estaría bien».
- «esto podría permitir afinar cualquier cosa pensada en nodos: una página web, su disposición de elementos, su
  navegación, un flujo de datos, un análisis de componentes multi agentes … todo llevado a la idea de nodos dentro de un
  grafo».
- «ten en consideración lo de visualización por niveles N1 N2 N3».
- «imagina una toma de requerimiento asistida con un llm, pero también por un humano experto que te puede ir guiando o
  mejorando tu idea, o anotar las características técnicas de tu idea o ideas de implementación … quizás esto podría
  guardar el audio de la conversación, si es que existe, e ir identificando elementos en realtime, haciendo
  anotaciones y dejándolo como historia de usuario. También considera que los flujos se tienen que versionar».
- «para lo del audio, lo que sea más barato; quizás una capacidad que genera el moderador, que es quien tiene la
  cuenta de qdd, y puede levantarlo con whisper o directo a claude, como lo prefiera» y «en los dos casos la
  transcripción debería quedar completa en el pc de quien tiene la cuenta de qdd como historia-usuaria».
- «también puede ser entre 2 colegas que mejoran un flujo y trabajan en conjunto en cómo debería funcionar un
  concepto» y «también se podría usar para levantar procesos».
- «falta un elemento de agrupación con un nombre que se use como, por ejemplo, cola, ambiente, fase … lo veo como un
  [] que agrupa nodos y sirve para visualizar mejor».

## Para qué sirve

Es la misma herramienta para tres conversaciones:

- **Con un cliente:** afinar un proceso, levantar sus requerimientos y acordar una versión.
- **Entre colegas:** dos personas que mejoran juntas un flujo o definen cómo debería funcionar un concepto. El colega
  recibe una invitación con rol experto, así que propone y anota lo técnico. Si también tiene QDD, puede trabajar en
  su rama (RFC 0021) y las propuestas llegan igual.
- **Para levantar un proceso:** se parte de un flujo casi vacío con la plantilla proceso. Se conversa, con audio si
  se quiere, y el asistente va proponiendo los pasos, decisiones y desvíos que escucha. Al final queda el proceso
  «como es», en una versión con nombre, y sus historias de usuario.

## Decisiones de la persona (2026-11-08)

1. **Entrega: archivo HTML + relay.** qdd genera un solo archivo HTML que se manda por correo o chat y se abre en
   cualquier navegador, sin instalar nada. Lo vivo pasa por el relay de QDD Móvil.
2. **El cliente comenta y propone.** Comenta un nodo, pregunta y propone agregar, cambiar, quitar, mover y conectar. En
   su archivo, sus propuestas se ven como propuestas (borde punteado), nunca como el flujo.
3. **Aprobación una por una.** Cada propuesta llega a «Para ti» como una acción: aprobar o rechazar. Solo lo aprobado
   entra al flujo, y el cliente ve qué se integró y qué no.
4. **Un flujo, tipos por plantilla.** Es el mismo modelo de `pkg/flujo` para todo. La plantilla trae los tipos de nodo,
   su forma, su leyenda y el nombre de cada nivel.
5. **El experto usa el mismo archivo, con rol experto.** Recibe su propia invitación. Además de proponer, deja notas
   técnicas e ideas de implementación por nodo. Cada nota es interna o compartida: la persona decide al aprobarla si el
   cliente la ve. Sirve para un experto que no tiene QDD.
6. **El LLM corre en nuestro PC, con Claude.** El archivo del cliente no lleva ninguna llave. Claude recibe en vivo lo
   que se dice y se propone, y sugiere nodos, notas e historias de usuario como propuestas del asistente. Se aprueban
   una por una, igual que las demás.
7. **Audio: lo levanta el moderador, en su PC, con consentimiento.** El moderador es quien tiene la cuenta de QDD y
   lleva la conversación (una llamada o presencial). La sesión con audio es una capacidad que él abre. Elige el
   motor: whisper en local (gratis, nada sale del PC) o directo a Claude con su cuenta. Se graba solo con el acepto de
   todos. En los dos casos, la transcripción completa queda en su PC como la historia-usuaria de la sesión. El audio se
   guarda solo si todos aceptan además guardarlo. Lo que se detecta llega como propuestas del asistente.
8. **Versiones: revisiones y versiones con nombre.** Cada cambio aprobado sube una revisión (r12) con su autor y su
   diff. Cuando la persona quiere, cierra una versión con nombre («v2 acordada con Ana»). Cada propuesta dice sobre qué
   revisión se hizo. Se pueden comparar dos versiones y volver a una.

## Quiénes participan

| Rol        | Dónde                      | Qué hace                                                                |
|------------|----------------------------|-------------------------------------------------------------------------|
| Cliente    | archivo HTML (invitación)  | mira por niveles, comenta, pregunta, propone                            |
| Experto    | archivo HTML (invitación)  | lo del cliente, más notas técnicas e ideas de implementación por nodo   |
| Asistente  | Claude en el PC            | escucha la sesión y propone nodos, notas e historias de usuario         |
| Moderador  | su PC, con cuenta de QDD   | lleva la sesión, abre el audio y elige el motor                         |
| La persona | panel, QDD Móvil           | aprueba o rechaza cada propuesta, cierra versiones                      |

El moderador suele ser la persona; si es otra persona del equipo, la sesión queda en su PC y sus propuestas llegan
igual a «Para ti». Nadie más que la persona cambia el flujo. Lo que escriben el cliente, el experto y el asistente, y lo que se dijo en
el audio, es dato y nunca instrucciones. Una IA que lo lee no hace lo que diga: lo propone para que la persona decida.
Por eso no hay aprobación automática, ni siquiera en modo goal.

## Diseño

### Plantillas

Un flujo tiene `plantilla`. Vacía es `proceso`, así los flujos de hoy no cambian. Una plantilla es un dato, no código:

| Plantilla | Tipos de nodo                            | N1       | N2               | N3        |
|-----------|------------------------------------------|----------|------------------|-----------|
| proceso   | paso                                     | Hitos    | Pasos            | Desvíos   |
| web       | pagina, seccion, componente, enlace      | Páginas  | Secciones        | Elementos |
| datos     | fuente, transformacion, regla, destino   | Fuentes  | Transformaciones | Detalle   |
| agentes   | agente, herramienta, memoria, persona    | Agentes  | Herramientas     | Mensajes  |

Todas traen además `resultado`, el círculo con que un flujo empieza y termina.

Para cada tipo, la plantilla trae:

- su **forma:** `resultado` es círculo y el resto caja, así el panel y el teléfono dibujan cualquier plantilla sin
  conocerla;
- su **ícono**;
- una **frase que lo explica**, que es la leyenda didáctica.

Un tipo que la plantilla no conoce se rechaza al editar, igual que hoy una operación desconocida.

### Grupos con clase

Un grupo junta nodos bajo un nombre y una clase: «[fase] Ingreso», «[cola] Revisión», «[ambiente] Producción». Los
nodos nombran su grupo en `Nodo.Grupo`, como ya lo hacían, y `Flujo.Grupos` le da a ese nombre su clase y su
descripción (`qdd flujo grupo <id> "<nombre>" --clase fase`). La clase es libre; cada plantilla sugiere algunas
(proceso: fase, área, cola; datos: ambiente, capa, cola…).

- **En N1** el grupo es una caja plegada con su clase y lo que esconde.
- **En N2 y N3** es un marco que sigue a sus nodos fila por fila, así no encierra a los que no son suyos, con su
  etiqueta arriba a la izquierda.
- Un nodo está en un solo grupo. Grupos dentro de grupos (un ambiente con sus fases) quedan para cuando haga falta.

### Niveles N1, N2, N3

Ya existen: `Nodo.Nivel`, `flujo.Niveles` y el plegado (#186). El nivel 1 es hito, el 2 es paso y el 3 es desvío. Se
calculan desde el camino principal y se corrigen por nodo. En el archivo funcionan así:

- **Abre en N1:** solo hitos, y los grupos como cajas. Es el panorama, y se lee aunque el flujo sea grande.
- **Tres botones fijos, N1 · N2 · N3,** con el nombre de la plantilla («N1 Páginas»). Cada nivel incluye los
  anteriores. Tocar una caja de grupo abre solo ese grupo en N2.
- **Una propuesta lleva su nivel.** Lo agregado estando en N1 nace como N1. Si nadie lo dice, se calcula.
- **qdd pliega cada nivel al generar el archivo,** con la misma regla que el panel. El navegador no recalcula nada.
- **Abre en N1 si el flujo tiene grupos o 15 nodos o más,** como el panel; si no, en N3.
- **Un enlace `#n2/<nodo>`** abre ese nivel con ese paso elegido, para decir «mira este paso». Abierto así, no
  muestra el recorrido.
- **Una vuelta atrás** sube por un carril a la izquierda, en ámbar, sin cruzar nodos.

### Versiones

- **`Flujo.Revision`:** sube con cada cambio guardado, venga de la IA, de la persona o de una propuesta aprobada.
- **`.qdd/project/flujos/<id>.historial.jsonl`:** es versionado y solo crece. Una línea por revisión:
  `{rev, en, autor, origen, resumen, operacion}`. El origen es `ia`, `persona`, `cliente:<nombre>`,
  `experto:<nombre>`, `asistente` o `transcripcion@mm:ss`, más la invitación. Da el «quién cambió qué» que git no da.
- **`qdd flujo version <id> "<nombre>"`:** guarda la foto entera en `.qdd/project/flujos/versiones/<id>/<n>.yaml` y la
  anota en `Flujo.Versiones` (`{n, nombre, revision, en}`).
- **`qdd flujo comparar <id> v1 v2|actual`:** muestra qué nodos se agregaron, cambiaron o quitaron, por nivel.
- **`qdd flujo volver <id> v1`:** escribe esa foto como una revisión nueva. La historia nunca se reescribe.
- **Cada propuesta lleva `base` (la revisión sobre la que se hizo).** Si el nodo cambió después, la acción de «Para
  ti» lo dice («cambió desde que Ana lo propuso») y deja ver las dos cosas antes de aprobar.
- **El archivo del cliente muestra la versión:** «v2 · r14». La lista de versiones deja ver cómo era cada una.

### Historias de usuario

Un flujo guarda sus historias junto a los nodos, en `Flujo.Historias`:

```yaml
historias:
  - id: h3
    como: vendedor
    quiero: validar el RUT antes de crear el cliente
    para: no duplicar clientes
    criterios: ["un RUT inválido no deja seguir", "uno repetido muestra el cliente que ya existe"]
    nodos: [validar-rut]
    origen: transcripcion@12:40   # o cliente:Ana, experto:Luis, asistente
    estado: aprobada              # solo las aprobadas se guardan; las propuestas viven en la cola
```

- **El asistente las propone** desde lo que se dice y se propone. El cliente y el experto también pueden escribirlas.
- **Aprobadas, se ven en el nodo** del panel y del archivo.
- **`qdd flujo historia tarea <id> <h>` crea la tarea del tablero.** Lleva la historia y sus criterios, y lo que sigue
  es el ciclo de siempre.
- **Las notas técnicas del experto son ítems** del nodo (`nota`, `caracteristica`, `funcion`, que ya existen), con
  `visible: interna|compartida`. La foto del cliente nunca lleva una interna.

### Sesión, audio y asistente

```
qdd flujo sesion <id> [--audio whisper|claude] [--sin-guardar-audio]   # la abre el moderador; «Terminar» la cierra
```

1. **Consentimiento primero.** Con `--audio`, el archivo de cada participante muestra «esta conversación se está
   transcribiendo» y pide «Acepto». En una sesión presencial, la persona lo confirma en el panel por quienes están.
   Sin el acepto de todos no se graba. El audio se guarda solo si todos aceptaron además guardarlo; si no, se borra
   cada tramo apenas se transcribe.
2. **Graba en el PC** con lo que el sistema tenga (PipeWire o PulseAudio en Linux, WASAPI en Windows, CoreAudio en
   Mac), en tramos de unos 15 s.
3. **Transcribe con el motor que eligió el moderador:**
   - **`whisper`** (el más barato y el de antes): whisper.cpp en local, con el modelo que instaló. Nada sale del PC.
     `qdd doctor` dice si falta y cómo instalarlo.
   - **`claude`:** cada tramo va a Claude con la cuenta del moderador, si su cuenta lo permite. qdd lo comprueba al
     abrir la sesión y, si no, lo dice y ofrece whisper. El audio sale del PC solo hacia Claude, y el aviso de
     consentimiento lo nombra.
4. **La transcripción completa queda en el PC del moderador** como la historia-usuaria de la sesión, con los dos
   motores. Va en `<QDD_HOME>/sesiones/<proyecto>/<flujo>/<fecha>/`:
   - `historia-usuaria.md`: la transcripción entera, con minuto y quién habla si se distingue, más al final las
     historias, notas y nodos que salieron de ella, con su estado (aprobada, rechazada, pendiente);
   - `audio/`: solo si todos aceptaron guardarlo.

   No va a git, porque es la voz de un cliente. Tampoco vence sola: la borra el moderador con `qdd flujo sesion borrar`.
   Al flujo versionado llegan solo las historias y notas aprobadas, con su minuto y la sesión como origen.
   `qdd flujo sesion ver <id>` lista las sesiones y abre la historia-usuaria.
5. **El asistente recibe los tramos nuevos en tandas.** Es la sesión de Claude que ya está abierta: llegan como
   «lo que falta enviar» del flujo, en cada pausa de la conversación y como mucho cada 60 s, para no gastar un turno
   por frase. Con `qdd flujo proponer --de asistente` deja propuestas de nodos, notas e historias, cada una con el
   minuto que la respalda.
6. **La persona aprueba en vivo o después.** Las propuestas no vencen al cerrar la sesión.

### En el PC: `qdd flujo compartir`

```
qdd flujo compartir <id> --para "<nombre>" [--rol cliente|experto] [--dias 14] [--tope 50] [--salida <archivo.html>]
qdd flujo compartir --revocar <invitación>
qdd flujo invitaciones            # las de este proyecto: flujo, rol, para quién, vencimiento y si siguen vivas
qdd flujo propuestas [ciclo|aprobar <PRO-n>|rechazar <PRO-n> --motivo …]
```

1. **Arma la foto.** Lleva plantilla, versión y revisión, y por nodo nombre, descripción, tipo, siguen, grupo y nivel,
   más los niveles plegados, las historias aprobadas y las notas compartidas. Nunca lleva tareas, pruebas, evidencias,
   comentarios internos, notas internas, rutas ni ids del tablero. El texto pasa por `reportes.Limpiar`.
2. **Crea la invitación** en el relay: un token al azar, con su rol, su vencimiento (14 días si no se dice) y un tope
   de propuestas por día. El token solo deja leer la foto vigente de ese flujo y escribir propuestas para él.
3. **Escribe el HTML.** Es un solo archivo, sin scripts ni estilos de afuera (CSP estricta), con la foto, la plantilla,
   la URL del relay y el token. Por defecto queda en `./tmp/`, y qdd dice la ruta para adjuntarlo.
4. **Revocar** invalida el token al momento. El archivo sigue abriendo, en solo lectura, y dice que la invitación
   terminó.

### En el navegador del cliente y del experto

- **Con la marca de QDD y todo adentro** (pedido de la persona): isotipo «▶_», tokens Gruvbox oscuros, JetBrains Mono
  embebida, esquinas de 2 px sin sombras y un ícono SVG por tipo de nodo. No carga nada de afuera.
- **Didáctico:** la leyenda de la plantilla a la vista, un recorrido corto la primera vez y nombres en vez de ids. Cada
  acción dice qué va a pasar: «tu propuesta le llega al equipo; la verás integrada cuando la aprueben».
- **Las mismas operaciones que QDD Móvil** (`flujo.Operacion`), sin las internas (marcar pruebas, deshacer, enviar),
  más `nota` (solo el experto) e `historia`. El esquema del PC sigue siendo quien decide qué vale.
- **En vivo:** el archivo escucha el canal de su invitación. Cuando la persona aprueba o rechaza, la propuesta cambia
  de estado en pantalla; cuando el flujo sube de revisión, llega la foto nueva. En una sesión, también ve las
  propuestas del asistente (sin la transcripción).
- **Sin conexión:** lee igual. Lo que propone queda en el navegador y se envía al volver, con aviso.
- **Su nombre** se pide una vez y va en cada propuesta.

### Contrato con el relay (lo implementa qdd-movile)

Como en RFC 0024, la tecnología del relay es de qdd-movile. Lo que el PC necesita:

- **Invitación:** `POST /functions/v1/flujo-invitar`, firmado por la instalación con la misma llave ed25519 de
  `qdd reportar`, con `{flujo, rol, foto, vence, tope}`. Devuelve `{invitacion, token}`. `DELETE` la revoca.
- **Foto nueva:** `PUT`, firmado, con la foto de la revisión aprobada. El relay la entrega a quien tenga un token vivo.
- **Propuesta:** `POST /functions/v1/flujo-proponer` con el token y `{autor, base, operacion}`. El relay revisa que el
  token esté vivo, que el rol permita la operación (`nota` solo experto), el tope, el tamaño (`flujo.MaxOperacion`) y
  que sea JSON. Nada más.
- **Lectura del PC:** `GET` firmado de las propuestas desde la última vista, paginado como `reportes.Leer`, y un canal
  realtime por instalación para que lleguen a «Para ti» sin esperar el sondeo.
- **Resolución:** `POST`, firmado, con `{propuesta, estado: aprobada|rechazada, motivo?}`. El relay la empuja al canal
  de la invitación.
- **Consentimiento de sesión:** `POST` con el token y `{sesion, acepta_transcribir, acepta_guardar_audio}`. El PC lo
  lee antes de grabar.
- **Los datos** quedan en el esquema `qdd_remoto` del proyecto propio de QDD Móvil, con cambios solo aditivos. Una
  invitación vencida borra sus propuestas a los 30 días. Por el relay nunca pasan ni el audio ni la transcripción.

### En el tablero: una por una

- **Cada propuesta es una acción de «Para ti»,** no una pregunta. Dice quién propone, el nodo y el cambio en palabras:
  «📝 Ana propone: agregar “Validar RUT” después de “Ingreso”» o «🎙 asistente (12:40): historia “validar el RUT…”».
  Botones: aprobar, rechazar (con motivo opcional) y ver en el flujo. Una nota del experto se aprueba como interna o
  como compartida.
- **Aprobar** aplica la operación con la edición de siempre (`edicion.go`), validada contra el esquema y la plantilla.
  Sube la revisión, queda en el historial con su origen, y qdd sube la foto nueva.
- **Rechazar** no toca el flujo. El que propuso ve «no se integró», con el motivo si lo hay.
- **Un comentario o una pregunta** también llega a «Para ti». La respuesta es una propuesta de respuesta que la persona
  aprueba, y no cambia el flujo.
- **Una propuesta cuyo nodo ya no existe** se muestra como tal y solo deja rechazarla.

### doctor

Un grupo «Flujos compartidos»:

- **FLC-01:** una invitación viva cuyo flujo ya no existe.
- **FLC-02:** propuestas sin resolver con más de 7 días.
- **FLC-03:** la foto del relay es más vieja que la revisión del flujo. Se arregla subiéndola de nuevo.
- **FLC-04:** el historial no cuadra con la revisión del flujo (por ejemplo, alguien editó el YAML a mano). Se repara
  agregando una línea «edición sin registro».
- **FLC-05:** una sesión con audio pide whisper y no está, o pide Claude y la cuenta no lo permite. Dice cómo
  arreglarlo y no se repara solo.
- **FLC-06:** una historia-usuaria cuya sesión quedó a medias (el PC se apagó). Se repara cerrándola con lo que alcanzó
  a transcribir.

`--fix` repara todo eso sin tocar ninguna propuesta.

## Fases

1. **Versiones, plantillas y archivo de solo lectura.** `Revision`, el historial, `version`, `comparar` y `volver`.
   Las cuatro plantillas con forma, ícono y leyenda. La foto limpia y el HTML con N1 · N2 · N3, leyenda y recorrido. Ya
   sirve para mostrar, sin relay.
2. **Relay (qdd-movile):** invitaciones por rol, propuestas, foto nueva, resolución, consentimiento y canal realtime.
3. **PC: propuestas.** `compartir` con invitación, la lectura de propuestas, las acciones de «Para ti» una por una,
   las notas del experto con su visibilidad, las historias de usuario y `historia tarea`.
4. **Asistente.** Las tandas de lo que llega en vivo a la sesión de Claude, y `qdd flujo proponer --de asistente`.
5. **Audio.** `qdd flujo sesion` abierta por el moderador, con consentimiento, grabación local, motor whisper o claude,
   y la historia-usuaria completa en su PC.
6. **doctor** (FLC-01 a FLC-06) y la vista de invitaciones y sesiones en el panel.

### Cómo quedó la fase 3 (2026-11-08)

Probada de punta a punta contra el relay real: invitación de experto, nota técnica propuesta desde el navegador,
pedido 📝 en el panel, aprobación a nombre del invitado (revisión r4 con autor `experto:<nombre>`), «Integrada» y la
foto r4 en el archivo sin recargarlo, y la invitación revocada deja el archivo en solo lectura.

- **El PC decide.** `pkg/invitaciones` revisa cada propuesta con `flujo.LeerOperacion`, el rol que anotó al invitar
  (no el que dice el relay) y el autor. Lo que no cumple se rechaza solo, con el motivo para el invitado.
- **Pedidos de la persona, tres a la vez.** Cada propuesta válida es un pedido `DePersona` (`qdd flujo propuestas
  aprobar PRO-n`), así que el modo goal nunca la aprueba. Descartar el pedido la rechaza; uno retirado vuelve a
  pedirse. Las demás esperan su turno para no llenar el panel.
- **Sondeo en vez de canal.** El panel llama `qdd flujo propuestas ciclo --si-toca` cada 15 s. qdd sale al relay solo
  si hay invitaciones vivas o algo que avisar. El archivo también sondea cada 15 s. El canal realtime queda para
  después, si el sondeo no alcanza.
- **Sin red no se pierde nada.** Aprobar no depende del relay. La resolución y la foto nueva se le avisan en el
  siguiente ciclo, y lo que el invitado propone sin conexión espera en su navegador.
- **El token** va solo dentro del archivo: qdd no lo imprime ni lo guarda, y el PC anota la invitación sin él.
- **Falta de esta fase:** la visibilidad interna o compartida de la nota del experto (por ahora toda nota es interna
  y no viaja en la foto), las historias de usuario con `historia tarea`, y que el relay devuelva la operación en
  `flujo-proponer GET`, para que un navegador nuevo muestre qué se propuso y no solo «Propuesta PRO-n».

## Lo que no hace

- No le da al cliente ni al experto una cuenta, ni acceso al tablero, a las tareas o a otros flujos.
- No integra nada sin la aprobación de la persona, una propuesta por vez, ni en modo goal.
- No pone una llave de LLM en el archivo. El audio y la transcripción nunca pasan por el relay; el audio sale del PC
  solo hacia Claude, y solo si el moderador eligió ese motor.
- No graba sin el acepto de todos, ni guarda audio sin un acepto aparte.
- No reescribe la historia: volver a una versión es una revisión nueva.
- No crea un modelo por tipo de cosa: una página web, un flujo de datos o un sistema de agentes son el mismo grafo con
  otra plantilla.
