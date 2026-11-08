# RFC 0023: baúl de accesos y ambientes completos

**Estado:** aceptado (2026-11-08), con las decisiones de la persona. Lo dibuja el flujo `baul-de-accesos`.

## Problema

Con la IA, lo que antes se hacía una vez se repite en cada proyecto: crear la zona en Cloudflare, el proyecto en
Supabase, el usuario o el rol en AWS, el dominio en Resend, los secretos del repositorio en GitHub. Cada vez se busca
el token, se copia a un `.env` o se pega en el chat. Así terminan tokens con más permisos de los necesarios, `.env` con
valores reales en cada carpeta y llaves en la conversación de una IA.

La persona lo pidió así: «un baúl central de tokens … desde el DNS hasta el despliegue … qdd tiene que tener una forma
centralizada pero segura de almacenar y usar esos tokens para las tareas que requiera cada proyecto», «y también saber
cuándo es necesario y autorizar el acceso cada vez por sesión con fines específicos».

## Decisiones de la persona

1. **El permiso es por sesión y por fin**, y vale **hasta que se cierra la consola** (2026-11-07).
2. **Los tokens se guardan en el llavero del sistema** por defecto: Secret Service en Linux, Llavero en macOS y
   Administrador de credenciales en Windows. **1Password es una opción** para quien lo use (2026-11-08).
3. **AWS entra con IAM Identity Center** (`aws sso login`): las credenciales vencen solas y el baúl guarda solo el
   perfil. No hay llaves de larga vida (2026-11-08).

## Diseño

### Guardar: una vez, y nunca por el chat

`qdd baul guardar <nombre> --tipo <tipo> [--en llavero|1password]` pide el valor con entrada oculta **en la terminal de
la persona**. Si no hay terminal (la herramienta Bash de una IA, un pedido o el teléfono), se niega y dice dónde
correrlo. El valor no entra por argumentos, por variables de entorno ni por un archivo, y nunca pasa por el chat, un
pedido o el teléfono.

- **Llavero:** el mismo en los tres sistemas, con una biblioteca que habla con cada uno (`zalando/go-keyring`: D-Bus en
  Linux, `security` en macOS, `wincred` en Windows). QDD no inventa su propio cifrado ni guarda copias.
- **1Password:** con la CLI `op` (`op item create`, `op read`). Con la app de escritorio, cada sesión de `op` pide la
  huella o la contraseña: es la opción más fuerte.
- **AWS:** se guarda el nombre del perfil de `aws sso`, no una llave.

Tipos de la primera versión: `cloudflare`, `github`, `supabase`, `resend`, `aws-sso` y `otro` (con la variable de
entorno donde va).

### Verificar: QDD lo prueba y anota su alcance

Al guardar, y con `qdd baul verificar <nombre>`, una llamada barata de solo lectura dice de quién es, qué alcanza y
cuándo vence:

| Tipo | Llamada |
|---|---|
| cloudflare | `GET /user/tokens/verify` |
| github | `GET /user` (y los alcances de la cabecera) |
| supabase | `GET /v1/projects` |
| resend | `GET /domains` |
| aws-sso | `aws sts get-caller-identity --profile <perfil>` |

Lo anotado va al índice `<QDD_HOME>/baul/indice.json`: nombre, tipo, dónde está, alcance, dueño, vencimiento, cuándo
se usó. Ningún valor.

### Permitir: por sesión, por fin, hasta cerrar la consola

`qdd baul usar <nombre> --para "<fin>" -- <comando>` corre un comando con el token. Lo usan la persona y la IA.

1. La sesión sale de `CLAUDE_CODE_SESSION_ID`, y el proyecto, de la carpeta. Sin sesión (la terminal de la persona),
   el permiso es de esa terminal y dura lo que ella.
2. Si esa sesión no tiene permiso para ese token y ese fin en ese proyecto, `usar` no corre nada: deja un pedido en el
   panel, «¿Permitir que esta sesión use cloudflare para "crear la zona de ejemplo.cl" en notificador, hasta cerrar la
   consola?», y sale con 75 («todavía no»).
3. Con el «Sí» de la persona en el panel, el permiso queda en `<QDD_HOME>/baul/permisos.json` con la sesión, el
   proyecto, el token y el fin.
4. El permiso muere cuando la sesión se cierra (`CerrarSesion`) o deja de latir (`VidaSesion`), lo que pase primero.

Un fin distinto es otro permiso. Un proyecto nunca alcanza un token que no declaró: el permiso se pide en el proyecto
donde se usa.

### Usar: el token vive lo que dura el comando

`usar` pone el token solo en el entorno del comando (`CLOUDFLARE_API_TOKEN`, `GITHUB_TOKEN`, `SUPABASE_ACCESS_TOKEN`,
`RESEND_API_KEY`, `AWS_PROFILE`) y no lo escribe en ningún archivo. Toda la salida pasa por el ocultador, que además
tapa cualquier valor guardado en el baúl. No existe `qdd baul ver`: la IA nunca ve el valor.

### Registro: cada uso deja huella

`<QDD_HOME>/baul/registro.jsonl` solo agrega. Cada línea lleva la huella de la anterior, así que si alguien borra una,
`doctor` lo ve. Dice quién (la persona, la IA de tal sesión, el teléfono), el proyecto, el token, el fin, el comando,
cómo terminó y cuándo. Nunca un valor. Se ve en el panel y en el teléfono.

### Doctor

Avisa de:

- tokens por vencer o rechazados;
- permisos de más (un token de Cloudflare con todas las zonas);
- tokens sin uso en 90 días;
- un registro roto;
- `.env` con valores reales en los proyectos de este PC, para pasarlos al baúl. Lista solo los nombres de las claves,
  nunca un valor.

### Después de la base: ambientes completos

`.qdd/ambientes.yaml`, versionado y sin valores, declara por ambiente (dev, prod) el dominio y la zona, el proyecto de
Supabase, la cuenta y el rol de AWS, el dominio de Resend y el repositorio. `qdd ambiente levantar dev|prod` recorre la
receta en orden (DNS, base de datos, backend, correo, secretos del despliegue, despliegue), mira qué existe, muestra el
plan y espera el clic. Producción y lo que no se deshace siempre piden confirmación. Los secretos que la app necesita
al correr van al gestor de donde corre (GitHub Actions, AWS Secrets Manager, secretos de Supabase o de Cloudflare
Workers), no a un `.env`. Rotar un token es cambiarlo una vez en el baúl y volver a levantar.

## Fases

1. **Base:** `guardar` (llavero y 1Password), `verificar`, `usar` con permiso por sesión y fin, ocultador, registro
   encadenado, `doctor`. Sin tool MCP nueva: tools/list está en 15.989 de 16.000 caracteres (RFC 0022), y la IA lo
   conoce por `qdd baul --help` y la forma de trabajo.
2. **Panel y teléfono:** el pedido de permiso, la lista de tokens con su alcance y vencimiento, y el registro.
3. **Ambientes:** `.qdd/ambientes.yaml` y `qdd ambiente levantar`, un proveedor a la vez (Cloudflare DNS primero).

## Lo que el baúl no protege

Es honesto decirlo: en Linux, el llavero abierto lo lee cualquier proceso del mismo usuario, incluida una IA que llame
a `secret-tool` directo. El baúl evita lo accidental (`.env` regados, tokens en el chat, permisos de más) y deja huella
de cada uso hecho por QDD. No es una caja fuerte contra un proceso del mismo usuario que quiera saltárselo. Para eso,
1Password con aprobación por sesión es más fuerte, y por eso queda como opción.

## Pruebas

Ninguna toca el llavero ni el 1Password reales: van con un llavero de prueba (`keyring.MockInit`) y un `op` falso, con
`HOME` y `QDD_HOME` temporales. Lo mínimo:

- `guardar` sin terminal se niega;
- `usar` sin permiso deja el pedido y sale con 75;
- el permiso muere al cerrar la sesión;
- el ocultador tapa el valor en la salida;
- borrar una línea del registro se detecta;
- un `.env` se revisa sin imprimir valores.
