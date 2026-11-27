# RFC 0027: cifrar en el teléfono lo que va al PC

**Estado:** propuesto (2026-11-27). El modo goal eligió la opción (cifrar en el teléfono, la recomendada); falta que la
persona la confirme. qdd-movile revisa este diseño antes de tocar código, y ninguno de los dos lados programa el cifrado
hasta esa confirmación.

**Origen:** FND-321 y FND-059 de qdd-movile. En el RFC 0013, la persona decidió que el relay no guarde lo que manda desde
el teléfono. No se cumple: las funciones del relay llaman a `realtime.send`, y cada `accion`, `responder` y `ejecutar`
queda entero en `realtime.messages` de 72 horas a 4 días. qdd-movile contó del 21 al 24-11 62 textos o comandos, 31
acciones y 11 respuestas.

## Resumen

El PC tendrá un par de claves propio. El teléfono cifrará hacia ese PC el contenido de lo que manda: el texto de un
`ejecutar`, la respuesta de un `responder`, la acción elegida y los datos de cada parte de una foto. Supabase seguirá
viendo pasar los mensajes y guardando los de `realtime.send`, pero ya no podrá leer su contenido. En claro queda solo
lo que el relay necesita para validar y firmar.

## Problema

- **Lo guardado se puede leer:** el relay guarda el contenido entero, y quien administra el proyecto de Supabase puede
  leerlo durante esos días, igual que un error de permisos como el de las 9 tablas de noviembre.
- **No hay cifrado de punta a punta:** el secreto del tópico y la firma de la máquina también los conoce el relay, porque
  él firma cada mensaje.
- **El broadcast directo no basta para todo:** la rc.149 manda las fotos sin pasar por `realtime.send`, y eso evita que
  se guarden. Pero una `accion`, un `responder` o un `ejecutar` necesitan que el relay revise la sesión, el plan y el
  teléfono revocado antes de firmar, y eso solo pasa por sus funciones.

## Diseño

**Claves del PC**
- **Identidad:** al emparejar, el PC genera una clave de firma Ed25519. Su parte pública va en el QR, que es el único
  canal que no pasa por Supabase. No cambia mientras el PC no se borre.
- **Cifrado:** el PC genera además un par X25519 y firma su pública con la de identidad. La pública firmada viaja en la
  oferta de emparejamiento y en la cabecera del estado (`clave: {x25519, firma, desde}`). Así la app la reconfirma al
  reconectar sin volver a escanear (punto de qdd-movile).
- **Rotación:** al volver a emparejar, el PC rota la de cifrado. Como la identidad no cambia, la app acepta la nueva sin
  escanear.
- **Dónde se guardan:** las privadas quedan donde hoy está la firma de la máquina, con los mismos permisos. Nunca van en
  un registro, en el estado ni al relay.

**Cifrado de cada mensaje**
- **Construcción:** ECIES de una sola vía (X25519 efímero, HKDF-SHA256 y AES-256-GCM). Es la de HPKE en modo base.
  - En Go sale de la biblioteca estándar (`crypto/ecdh`, `crypto/hkdf` y `crypto/cipher`), sin dependencias nuevas.
  - qdd-movile elige la biblioteca de Dart. El fixture compartido fija los bytes exactos.
- **Qué va cifrado:**
  - de `ejecutar`, `texto`;
  - de `responder`, `respuesta`;
  - de `accion`, `accion_id`;
  - de cada parte de `foto2`, `datos`.
  El resto queda en claro: solicitud, proyecto, tipo, pregunta, parte, de, dispositivo y en.
- **Fotos:** sealed box por cada parte de 48 KB. El id sigue siendo el sha256 de la foto en claro, así que el PC
  verifica la huella después de abrir las partes, y la firma del `ejecutar` sigue cubriendo la foto (punto de
  qdd-movile).
- **Firma del relay:** firma el texto cifrado como hoy firma el texto en claro, porque para él son bytes. El largo que
  revisa es el del cifrado. Los caracteres los vuelve a revisar el PC al abrir, con `ValidarTexto`, como ya hace.
- **Error de clave:** si el PC no puede abrir un mensaje porque la app usó una pública vieja, responde con el motivo
  nuevo `clave` (en `accion_estado` y en `foto_ack`). La app reconfirma la pública desde el estado y ofrece reenviar.
  La persona ve «Tu PC cambió su clave; vuelve a mandarlo» (punto de qdd-movile).
- **Capacidad:** el PC declara `cifrado1` en `capacidades`. Mientras un PC no la declare, la app no le manda contenido
  cifrado. Cuando la app nueva esté en Play, el PC deja de aceptar contenido en claro: un mensaje en claro termina
  rechazado con su motivo.

## Lo que no se puede romper

- La privada del PC nunca sale de la máquina ni aparece en registros, pedidos, el baúl ni el teléfono.
- Un mensaje que el PC no puede abrir o verificar nunca se ejecuta. Termina rechazado y dice por qué.
- Las validaciones del relay (sesión, Pro, revocado, largo) y las del PC (firma, vigencia de 5 min, lista permitida,
  sin shell) siguen todas.

## Alternativas

- **Aceptarlo y decirlo en la política:** es lo más rápido, pero contradice lo que decidió la persona en el RFC 0013.
- **Todo por canal directo, como las fotos:** no se guarda, pero se pierde la revisión y la firma del relay. Habría que
  diseñar otra forma de autorizar cada mensaje.
- **Clave derivada de un código que escribe la persona:** se evita confiar en el estado, pero suma un paso a cada
  teléfono.

## Lo que no entra

- Lo que el PC manda al teléfono (`estado`, `delta`, `consola`, `logrado`). Va por broadcast de cliente y qdd-movile no
  encontró filas guardadas, pero Supabase lo ve pasar. Cifrarlo es otro paso.
- Los metadatos: la clave de proyecto (`<carpeta>-<hash>`), los tiempos y los tamaños siguen visibles.

## Preguntas abiertas

1. **Teléfonos que no escanearon:** un segundo teléfono de la misma cuenta ve el PC por la cuenta, sin escanear el QR,
   así que no tiene la identidad desde fuera de Supabase. Hay dos caminos:
   - confiar en la primera clave que ve y avisar si cambia sin firma válida;
   - mostrar en el PC y en el teléfono una huella corta para compararla.

   Propuesta: confiar en la primera clave, con la huella visible en el detalle del PC.
2. **Latest:** el congelamiento de la 2.10.0 espera a que se cierre FND-321. ¿La 2.10.0 pasa a latest con FND-321
   resuelto en la política (decir qué se guarda y cuánto) y el cifrado después, o espera el cifrado? Lo decide la
   persona.

## Fases

1. Este RFC, revisado por qdd-movile y confirmado por la persona.
2. Fixture compartido con las claves de prueba, los mensajes cifrados y lo esperado al abrirlos. Va en los dos
   repositorios, sin cambios.
3. PC: claves, `clave` en la cabecera del estado, abrir antes de validar, motivo `clave` y capacidad `cifrado1`.
4. App: cifrar cuando ve `cifrado1` y reconfirmar la pública.
5. Prueba real con el teléfono de la persona: qdd-movile cuenta en `realtime.messages` que lo guardado ya no se lee. Al
   final, el PC deja de aceptar contenido en claro.
