// Mod de Claude Code con las tareas en curso del proyecto (QDD, RFC 0006 §3.4.1, §3.7, §3.9 y §3.10; blueprint
// consola-claude-code). Dibuja el panel liviano: una línea de resumen en color, la tabla de las 4 tareas pendientes más
// nuevas con su ✕ para descartarlas y, separadas por una línea fina, «▶ Para ti» (los comandos que la IA dejó para la
// persona), ❌ Fallas y la fila de Acciones (FND-090). Del proyecto se ve solo el nombre de la carpeta. La confirmación
// y el resultado de un botón van en su misma sección. Cada pedido se ve en su estado: pendiente (botón con su
// tecla), en espera de otro, ejecutando o hecho (en gris, sin botón: FND-070, FND-080), «todavía no» (pendiente, con
// su aviso en gris: FND-082) y con falla (en rojo, con «↻ Reintentar»). La lógica vive en la CLI: el mod lee 'qdd consola panel --json' cada INTERVALO_MS y solo dibuja; una
// acción se pide con 'qdd consola accion <id>', que qdd vuelve a comprobar. Lo que escribe pasa antes por «s  Sí».
// Sin qdd, con un qdd antiguo o sin .qdd/project/estado.yaml, el panel dice cómo crearlo y el resto de la consola
// sigue igual. QDD_BIN apunta a otro binario de qdd (p. ej. uno de prueba).

const PANE = 'qdd-tareas'
const TITULO = 'Tareas'
const INTERVALO_MS = 2000
const TOPE_MS = 3000
const TOPE_ACCION_MS = 60000
const TOPE_AUDITAR_MS = 600000
const LINEAS_RESULTADO = 4
// OPCIONALES son los tipos de acción opcionales que este panel dibuja; qdd los ofrece solo si se piden (FND-092).
const OPCIONALES = ['atender']
// Tarea en curso (RFC 0006 §3.13, blueprint estado-por-tareas): el ● late con un tic local de LATIDO_MS (cambia de
// fase cada 2 tics, 1 s) y destella un tic con el color de acento en cada pulso. Los pulsos del agente van a la CLI
// como mucho cada PULSO_MIN_MS.
const LATIDO_MS = 500
const PULSO_MIN_MS = 5000
const ACENTO = 'cyan'

let datos = null // { estado, motivo, acciones }
let falla = ''
let firma = ''
let confirmando = null // la acción que espera el segundo paso
let resultado = null // { titulo, ok, texto, zona }
let ocupado = false
let iniciado = false // ya pasó la primera lectura de session.start
// CODIGO_TODAVIA_NO es la salida de un pedido cuya condición aún no se cumple (qdd: consola.CodigoTodaviaNo).
const CODIGO_TODAVIA_NO = 75
const pedidosVistos = new Set() // ids de los pedidos que ya abrieron el panel
// locales: lo que el panel ya sabe de un pedido antes que qdd (o con un qdd anterior a 2.6.2, que no guarda el
// estado): id → { estado, fin }. Así un pedido aceptado nunca vuelve a ofrecerse.
const locales = new Map()
let tic = 0 // tics del latido; la fase cambia cada 2
const destellos = {} // id → tics de destello que quedan
const pulsos = {} // id → último pulso visto en el JSON
let ultimoEnvio = -Infinity // cuándo se envió el último pulso a la CLI

async function binario($) {
  const propio = await $.env.get('QDD_BIN')
  return propio || 'qdd'
}

// leer pide el panel a la CLI y redibuja solo si cambió.
async function leer($) {
  const bin = await binario($)
  let nuevo = null
  let motivo = ''
  try {
    let r = await $.process.run([bin, 'consola', 'panel', '--json', '--con', OPCIONALES.join(',')], { timeoutMs: TOPE_MS })
    // Un qdd anterior a --con no lo conoce: se lee sin los opcionales.
    if (r.exitCode !== 0 && /unknown flag/.test(r.stderr || '')) r = await $.process.run([bin, 'consola', 'panel', '--json'], { timeoutMs: TOPE_MS })
    motivo = r.exitCode === 0 ? '' : (r.stderr || '').trim() || 'esta versión de qdd no trae «qdd consola panel»: actualiza qdd'
    nuevo = r.exitCode === 0 ? JSON.parse(r.stdout) : null
  } catch (err) {
    motivo = 'no se pudo ejecutar ' + bin + ': ' + String(err?.message ?? err)
  }
  const huella = JSON.stringify([nuevo, motivo])
  if (huella === firma) return
  firma = huella
  datos = nuevo
  falla = motivo
  // Lo aceptado que qdd ya anotó (ejecutando, hecho o con falla) deja de ser local.
  for (const a of nuevo?.acciones || []) if (locales.has(a.id) && a.estado && a.estado !== 'pendiente') locales.delete(a.id)
  verPulsos()
  $.ui.invalidate('ui.render')
  await abrirSiLlegaUnPedido($)
}

// verPulsos marca un destello en cada tarea cuyo pulso cambió desde la lectura anterior (lo deja cualquier fuente).
function verPulsos() {
  for (const f of datos?.estado?.frentes || []) {
    const pulso = f.actividad?.pulso
    if (!pulso) continue
    if (pulsos[f.id] && pulsos[f.id] !== pulso) destellos[f.id] = 2
    pulsos[f.id] = pulso
  }
}

function hayEnCurso() {
  return (datos?.estado?.frentes || []).some((f) => f.actividad?.running)
}

// latir avanza el tic local y redibuja solo si hay una tarea en curso o un destello pendiente.
function latir($) {
  const habia = Object.values(destellos).some((n) => n > 0)
  for (const id of Object.keys(destellos)) destellos[id] = Math.max(0, destellos[id] - 1)
  if (!hayEnCurso() && !habia) return
  tic += 1
  $.ui.invalidate('ui.render')
}

// pulsar destella al instante la tarea actual y, como mucho cada PULSO_MIN_MS, deja el pulso en la CLI sin esperar:
// la herramienta o el subagente siguen su curso aunque qdd tarde o falle. Sin tarea en curso el pulso igual va: qdd
// no toca la actividad y lo anota como uso real (qdd uso).
async function pulsar($, fuente) {
  const actual = datos?.estado?.actual
  if (actual) {
    destellos[actual] = 2
    $.ui.invalidate('ui.render')
  }
  const ahora = await $.clock.now()
  if (ahora - ultimoEnvio < PULSO_MIN_MS) return
  ultimoEnvio = ahora
  const bin = await binario($)
  $.process.run([bin, 'estado', 'pulso', '--fuente', fuente], { timeoutMs: TOPE_MS }).catch(() => {})
}

// enCurso: «● running» de una tarea en curso; el ● alterna tenue y brillante, y destella en acento con un pulso.
function enCurso(Text, f) {
  if (!f.actividad?.running) return []
  const destella = (destellos[f.id] || 0) > 0
  const brillante = Math.floor(tic / 2) % 2 === 0
  return [
    Text({ key: 'run-' + f.id, color: destella ? ACENTO : colorFrente(f), dimColor: !destella && !brillante, children: ['●'] }),
    Text({ key: 'runt-' + f.id, dimColor: true, size: 'caption', children: ['running'] }),
  ]
}

// Un comando pedido por la IA abre el panel (sin quitarle el foco a la conversación), una vez por pedido.
async function abrirSiLlegaUnPedido($) {
  const nuevos = pedidos(datos?.acciones || []).filter((a) => ejecutable(a) && !pedidosVistos.has(a.id))
  for (const a of nuevos) pedidosVistos.add(a.id)
  if (!iniciado || nuevos.length === 0) return
  await $.ui.open({ id: PANE, title: TITULO })
}

function pedidos(lista) {
  return lista.filter((a) => a.tipo === 'pedido')
}

// estadoDe: el estado que manda es el de qdd, salvo mientras el panel lo ejecuta o cuando qdd aún lo da por pendiente.
// estadoDe: lo que dice qdd, salvo justo después de aceptar un pedido, mientras qdd todavía lo muestra pendiente
// (FND-070: nunca se ofrece dos veces). Un «todavía no» anotado después de aceptarlo manda: vuelve a tener botón.
function estadoDe(a) {
  const local = locales.get(a.id)
  const remoto = a.estado || 'pendiente'
  const avisoNuevo = local && a.aviso_en && new Date(a.aviso_en) >= new Date(local.inicio || 0)
  if (local && remoto === 'pendiente' && !avisoNuevo) return local
  return { estado: remoto, inicio: a.inicio, fin: a.fin, resultado: a.resultado }
}

function ejecutable(a) {
  const { estado } = estadoDe(a)
  return estado === 'pendiente' || estado === 'fallo'
}

function hora(iso) {
  const t = new Date(iso)
  if (Number.isNaN(t.getTime())) return ''
  const dos = (n) => String(n).padStart(2, '0')
  return dos(t.getHours()) + ':' + dos(t.getMinutes())
}

// celda alinea un símbolo con su abreviatura: ✅ ⏳ ❌ ocupan 2 columnas; — (no aplica), · (sin empezar) y ? una.
const ANGOSTOS = new Set(['—', '·', '?'])
function celda(s) {
  return ANGOSTOS.has(s) ? s + ' ' : s
}

// total: el de qdd (el mismo que la barra, FND-086); un qdd anterior no lo trae y se cuenta la lista.
function total(n, lista) {
  return typeof n === 'number' ? n : lista.length
}

// colorFrente: la tarea toma el color de su peor etapa.
function colorFrente(f) {
  if (f.simbolos.includes('❌')) return 'red'
  if (f.simbolos.includes('⏳')) return 'yellow'
  return 'green'
}

// nombre: el último segmento de una ruta (con / o \\): del proyecto se ve solo el nombre de la carpeta (FND-090).
function nombre(ruta) {
  const partes = String(ruta || '').split(/[\\/]+/).filter(Boolean)
  return partes[partes.length - 1] || String(ruta || '')
}

// enProyecto: la carpeta de un pedido nombrada desde la del proyecto («qdd/npm»); fuera de él, solo su nombre.
function enProyecto(ruta) {
  const raiz = datos?.carpeta || ''
  if (!ruta) return ''
  if (raiz && (ruta + '/').startsWith(raiz + '/')) return nombre(raiz) + ruta.slice(raiz.length).replace(/\\/g, '/')
  return nombre(ruta)
}

// zona: la sección donde van la confirmación y el resultado de una acción, la del botón que se pulsó.
function zona(a) {
  if (a.tipo === 'pedido') return 'pedidos'
  if (a.tipo === 'descartar') return 'tabla'
  return 'acciones'
}

// separador: una línea fina a todo el ancho entre secciones, con el título debajo y un dato atenuado a la derecha.
function separador(Box, Text, key, texto, color, derecha, ancho) {
  return Box({
    key,
    flexDirection: 'column',
    marginTop: 1,
    children: [
      Text({ key: key + '-linea', dimColor: true, wrap: 'truncate-end', children: ['─'.repeat(Math.max(8, ancho))] }),
      Box({
        flexDirection: 'row',
        justifyContent: 'space-between',
        columnGap: 1,
        children: [
          Box({ flexShrink: 0, children: [Text({ key: key + '-t', color, weight: 'bold', children: [texto] })] }),
          ...(derecha ? [Box({ flexShrink: 1, children: [Text({ dimColor: true, size: 'caption', wrap: 'truncate-start', children: [derecha] })] })] : []),
        ],
      }),
    ],
  })
}

// etapas: los ⏳ y ❌ son etapas, no tareas (FND-088): «1 etapa espera», «10 etapas esperan».
function etapas(n, uno, varios) {
  return n + (n === 1 ? ' etapa ' + uno : ' etapas ' + varios)
}

// cuantas: «4 tareas», o «4 de 9 tareas» si hay pendientes más antiguas que no entran.
function cuantas(e) {
  const n = e.frentes.length
  return e.ocultos > 0 ? n + ' de ' + (n + e.ocultos) + ' tareas' : n + (n === 1 ? ' tarea' : ' tareas')
}

// cabecera: el título con la hora y, a la derecha, la carpeta del proyecto (solo su nombre) con el botón para abrirla;
// debajo, el resumen en una línea con el color de cada conteo. Un conteo en cero no se muestra (FND-090). El panel no
// abre nada por su cuenta: pide 'abrir-carpeta', que qdd ofrece solo si el sistema tiene abridor.
function cabecera(Box, Text, Button, $, e, lista) {
  const cuando = e ? e.hace || hora(e.actualizado) : ''
  const abrirla = lista.find((a) => a.id === 'abrir-carpeta')
  const carpeta = datos.carpeta
    ? [
        Box({
          key: 'carpeta',
          flexDirection: 'row',
          flexShrink: 1,
          columnGap: 2,
          children: [
            Text({ key: 'ruta', dimColor: true, wrap: 'truncate-end', children: ['📂 ' + nombre(datos.carpeta)] }),
            ...(abrirla ? [Button({ key: 'b-abrir-carpeta', label: 'Abrir carpeta', hotkey: 'o', plain: true, onPress: () => pedir($, abrirla) })] : []),
          ],
        }),
      ]
    : []
  const fila = Box({
    flexDirection: 'row',
    justifyContent: 'space-between',
    columnGap: 2,
    children: [Box({ flexShrink: 0, children: [Text({ key: 'titulo', color: 'cyan', weight: 'bold', size: 'title', children: [TITULO + (cuando ? ' · ' + cuando : '')] })] }), ...carpeta],
  })
  if (!e) return Box({ key: 'cabecera', flexDirection: 'column', children: [fila] })
  const aviso = e.viejo
    ? [Text({ key: 'viejo', color: 'yellow', wrap: 'wrap', children: ['⚠ Sin actualizar ' + e.hace + ': lo que dice ⏳ puede haber cambiado'] })]
    : []
  const esperan = total(e.esperan, e.esperas)
  const fallan = total(e.fallan, e.fallas)
  const paraTiN = lista.filter((a) => a.tipo === 'pedido' && ejecutable(a)).length
  const conteos = [
    ...(esperan ? [Text({ key: 'r-esperas', color: 'yellow', weight: 'semibold', children: ['⏳ ' + etapas(esperan, 'espera', 'esperan')] })] : []),
    ...(fallan ? [Text({ key: 'r-fallas', color: 'red', weight: 'semibold', children: ['❌ ' + etapas(fallan, 'falla', 'fallan')] })] : []),
    ...(esperan || fallan || e.frentes.length === 0 ? [] : [Text({ key: 'r-limpio', color: 'green', children: ['✓ nada espera ni falla'] })]),
    ...(paraTiN ? [Text({ key: 'r-para-ti', color: 'yellow', weight: 'semibold', children: ['▶ ' + paraTiN + ' para ti'] })] : []),
  ]
  const punto = (i) => Text({ key: 'p-' + i, dimColor: true, children: ['·'] })
  return Box({
    key: 'cabecera',
    flexDirection: 'column',
    children: [
      fila,
      Box({
        key: 'resumen',
        flexDirection: 'row',
        flexWrap: 'wrap',
        columnGap: 1,
        children: [Text({ key: 'r-tareas', color: 'cyan', weight: 'semibold', children: [cuantas(e)] }), ...conteos.flatMap((t, i) => [punto(i), t])],
      }),
      ...aviso,
    ],
  })
}

// tabla: una fila por tarea con sus etapas y, al final, la ✕ que la descarta (pide «s  Sí»: escribe en estado.yaml).
// Cada etapa ⏳ o ❌ que qdd ofrece atender es un botón: le envía a la IA el mensaje de qdd para que se pronuncie.
// La ✕ y las etapas se pulsan con el mouse; las tareas más antiguas que no entran se cuentan debajo.
function tabla(Box, Text, Button, $, e, lista, ancho) {
  const columnas = 3 * e.etapas.length + 2
  const largo = Math.max('Tarea'.length, ...e.frentes.map((f) => f.nombre.length))
  const nombreAncho = Math.max(8, Math.min(largo, ancho - columnas - 3))
  const fila = (key, nombre, derecha, estilo, cierre) =>
    Box({
      key,
      flexDirection: 'row',
      columnGap: 1,
      children: [
        Box({ width: nombreAncho, children: [Text({ ...estilo, wrap: 'truncate-end', children: [nombre] })] }),
        ...derecha,
        ...cierre,
      ],
    })
  const etapasDe = (f) =>
    f.simbolos.map((s, i) => {
      const a = lista.find((x) => x.id === 'atender:' + f.id + ':' + i)
      if (a) return Button({ key: 'e-' + f.id + '-' + i, label: s, plain: true, onPress: () => atender($, a) })
      return Text({ key: 'c-' + f.id + '-' + i, color: colorFrente(f), children: [celda(s)] })
    })
  const equis = (f) => {
    const a = lista.find((x) => x.id === 'descartar:' + f.id)
    return a ? [Button({ key: 'x-' + f.id, label: '✕', plain: true, dimColor: true, onPress: () => pedir($, a) })] : []
  }
  const leyenda = e.etapas.map((etapa, i) => e.abreviaturas[i] + ' ' + etapa).join(' · ')
  const antiguas = e.ocultos > 0 ? [Text({ key: 'ocultas', dimColor: true, size: 'caption', children: ['+' + e.ocultos + ' más antiguas · qdd estado ver'] })] : []
  return Box({
    key: 'tabla',
    flexDirection: 'column',
    marginTop: 1,
    children: [
      fila('cab', 'Tarea', [Text({ dimColor: true, size: 'caption', children: [e.abreviaturas.join(' ')] })], { dimColor: true, size: 'caption' }, []),
      ...e.frentes.map((f) => fila('f-' + f.id, f.nombre, etapasDe(f), { color: colorFrente(f) }, [...enCurso(Text, f), ...equis(f)])),
      ...antiguas,
      Text({ dimColor: true, italic: true, size: 'caption', wrap: 'wrap', children: [leyenda] }),
      ...enZona(Box, Text, Button, $, 'tabla'),
    ],
  })
}

// fallas: cada etapa en ❌ como «• tarea › etapa» y debajo, atenuado, qué falla.
function fallas(Box, Text, items, ancho) {
  if (items.length === 0) return []
  return [
    separador(Box, Text, 's-fallas', '❌ Fallas', 'red', String(items.length), ancho),
    ...items.map((p, i) =>
      Box({
        key: 's-fallas-' + i,
        flexDirection: 'column',
        children: [
          Text({ color: 'red', wrap: 'wrap', children: ['• ' + p.frente + ' › ' + p.etapa] }),
          Text({ dimColor: true, wrap: 'wrap', children: ['  ' + p.texto] }),
        ],
      }),
    ),
  ]
}

// tecla: la que qdd da a cada pedido (a, b, d…), 1…9 para las fichas en el orden en que qdd las entrega, c para
// revisar el estado, u para auditar y o para abrir la carpeta.
function teclas(acciones) {
  const asignadas = {}
  let n = 0
  for (const a of acciones) {
    if (a.tecla) {
      asignadas[a.id] = a.tecla
      continue
    }
    if (a.tipo === 'aprobar') {
      n += 1
      asignadas[a.id] = String(n)
      continue
    }
    if (a.id === 'revisar-estado') asignadas[a.id] = 'c'
    if (a.id === 'auditar') asignadas[a.id] = 'u'
    if (a.id === 'abrir-carpeta') asignadas[a.id] = 'o'
  }
  return asignadas
}

async function ejecutar($, a) {
  const esPedido = a.tipo === 'pedido'
  let todaviaNo = false
  if (esPedido && !ejecutable(a)) return
  ocupado = true
  confirmando = null
  if (esPedido) locales.set(a.id, { estado: 'ejecutando', inicio: new Date().toISOString() })
  resultado = { titulo: a.etiqueta, ok: true, texto: 'ejecutando ' + primeraLinea(a.detalle) + '…', zona: zona(a) }
  $.ui.invalidate('ui.render')
  const bin = await binario($)
  const argv = [bin, 'consola', 'accion', a.id, ...(a.confirmar ? ['--confirmado'] : [])]
  let lanzado = false
  try {
    // Un pedido solo se lanza: qdd lo corre aparte y el panel queda libre (FND-089); el resultado llega por la cola.
    const r = await $.process.run(argv, { timeoutMs: a.id === 'auditar' ? TOPE_AUDITAR_MS : TOPE_ACCION_MS })
    const texto = ((r.stdout || '') + (r.stderr || '')).trim().split('\n').slice(-LINEAS_RESULTADO).join('\n')
    resultado = { titulo: a.etiqueta, ok: r.exitCode === 0, texto: texto || 'sin salida', zona: zona(a) }
    todaviaNo = r.exitCode === CODIGO_TODAVIA_NO
    if (todaviaNo) resultado = { ...resultado, todaviaNo: true }
    lanzado = esPedido && r.exitCode === 0
  } catch (err) {
    resultado = { titulo: a.etiqueta, ok: false, texto: String(err?.message ?? err), zona: zona(a) }
  }
  // Lanzado: la tarjeta del pedido dice cómo va; sin recuadro de resultado. Rechazado o «todavía no»: manda qdd.
  if (lanzado) resultado = null
  if (esPedido && !lanzado) locales.delete(a.id)
  ocupado = false
  firma = ''
  await leer($)
  $.ui.invalidate('ui.render')
}

// atender pide a qdd el mensaje de una etapa (qdd lo vuelve a calcular: si ya no espera, no hay nada que enviar) y lo
// envía a la conversación como un mensaje de la persona; la IA lo atiende en cuanto queda libre.
async function atender($, a) {
  if (ocupado) return
  ocupado = true
  confirmando = null
  const bin = await binario($)
  let texto = ''
  try {
    const r = await $.process.run([bin, 'consola', 'accion', a.id], { timeoutMs: TOPE_MS })
    texto = r.exitCode === 0 ? (r.stdout || '').trim() : ''
    resultado = { titulo: a.etiqueta, ok: false, texto: ((r.stdout || '') + (r.stderr || '')).trim() || 'sin salida', zona: 'tabla' }
  } catch (err) {
    resultado = { titulo: a.etiqueta, ok: false, texto: String(err?.message ?? err), zona: 'tabla' }
  }
  if (texto) {
    resultado = { titulo: a.etiqueta, ok: true, texto: '→ enviado a la IA', zona: 'tabla' }
    $.prompt.submit({ text: texto }).catch((err) => $.ui.log('no se envió a la IA: ' + String(err?.message ?? err)))
  }
  ocupado = false
  $.ui.invalidate('ui.render')
}

// descartar quita un pedido de la cola sin ejecutarlo.
async function descartar($, a) {
  ocupado = true
  confirmando = null
  const bin = await binario($)
  try {
    const r = await $.process.run([bin, 'consola', 'descartar', a.id.slice('pedido:'.length)], { timeoutMs: TOPE_MS })
    resultado = { titulo: 'Descartado: ' + a.etiqueta, ok: r.exitCode === 0, texto: ((r.stdout || '') + (r.stderr || '')).trim() || 'sin salida', zona: 'pedidos' }
  } catch (err) {
    resultado = { titulo: 'Descartar ' + a.etiqueta, ok: false, texto: String(err?.message ?? err), zona: 'pedidos' }
  }
  ocupado = false
  await leer($)
  $.ui.invalidate('ui.render')
}

function pedir($, a) {
  if (ocupado) return
  if (a.tipo === 'pedido' && !ejecutable(a)) return
  if (!a.confirmar) {
    ejecutar($, a)
    return
  }
  confirmando = a
  $.ui.invalidate('ui.render')
}

function confirmacion(Box, Text, Button, $) {
  const a = confirmando
  return Box({
    key: 'confirmar',
    flexDirection: 'column',
    borderStyle: 'single',
    borderColor: 'yellow',
    paddingX: 1,
    children: [
      Text({ color: 'yellow', dimColor: true, size: 'caption', children: [a.tipo === 'pedido' ? 'Confirma antes de ejecutar' : 'Escribe en el proyecto'] }),
      Text({ color: 'yellow', weight: 'bold', wrap: 'wrap', children: ['¿' + a.etiqueta + '?'] }),
      Text({ dimColor: a.tipo !== 'pedido', mono: a.tipo === 'pedido', wrap: 'wrap', children: ['Ejecutará: ' + a.detalle] }),
      ...(a.carpeta ? [Text({ dimColor: true, size: 'caption', wrap: 'truncate-start', children: ['En: ' + enProyecto(a.carpeta)] })] : []),
      Box({
        flexDirection: 'row',
        columnGap: 2,
        marginTop: 1,
        children: [
          Button({ key: 'si', label: 's  Sí', hotkey: 's', variant: 'primary', onPress: () => ejecutar($, a) }),
          Button({
            key: 'no',
            label: 'n  No',
            hotkey: 'n',
            onPress: () => {
              confirmando = null
              $.ui.invalidate('ui.render')
            },
          }),
          ...(a.tipo === 'pedido' ? [Button({ key: 'descartar', label: 'x  Descartar', hotkey: 'x', onPress: () => descartar($, a) })] : []),
        ],
      }),
    ],
  })
}

// primeraLinea: un comando de varias líneas se ve en una, con cuántas faltan; entero solo al confirmarlo.
function primeraLinea(comando) {
  const lineas = String(comando || '').trim().split('\n')
  return lineas.length > 1 ? lineas[0] + '  … (+' + (lineas.length - 1) + ' líneas)' : lineas[0]
}

// tarjeta dibuja un pedido: qué (con su marca o su botón), el comando exacto en una línea y, atenuado, cómo va y en
// qué carpeta (desde la del proyecto, FND-090). Solo pendiente y con falla tienen botón; en espera y ejecutando van en
// gris; un hecho ocupa una línea.
function tarjeta(Box, Text, Button, $, a) {
  const s = estadoDe(a)
  const comando = Text({ key: 'cmd', mono: true, dimColor: true, wrap: 'truncate-end', children: ['   $ ' + primeraLinea(a.detalle)] })
  const donde = enProyecto(a.carpeta)
  const meta = (...partes) => {
    const texto = [...partes, donde ? 'en ' + donde : ''].filter(Boolean).join(' · ')
    return texto ? [Text({ key: 'meta', dimColor: true, size: 'caption', wrap: 'truncate-end', children: ['   ' + texto] })] : []
  }
  const caja = (cabeza, resto) => Box({ key: 'pedido-' + a.id, flexDirection: 'column', children: [cabeza, ...resto] })
  const gris = (marca) => Text({ key: 'que', dimColor: true, wrap: 'truncate-end', children: [marca + '  ' + a.etiqueta] })
  if (s.estado === 'espera') {
    const de = (a.espera || []).map((t) => '[' + t + ']').join(' ')
    return caja(gris('◌'), [comando, ...meta('espera a ' + (de || 'otro pedido'))])
  }
  if (s.estado === 'ejecutando') {
    return caja(gris('⟳'), [comando, ...meta(hora(s.inicio) ? 'ejecutando desde ' + hora(s.inicio) : 'ejecutando…')])
  }
  if (s.estado === 'hecho') {
    return Box({
      key: 'pedido-' + a.id,
      flexDirection: 'row',
      justifyContent: 'space-between',
      columnGap: 2,
      children: [
        Box({ flexShrink: 1, children: [gris('✓')] }),
        Box({ flexShrink: 0, children: [Text({ key: 'meta', dimColor: true, size: 'caption', children: [hora(s.fin) ? 'hecho ' + hora(s.fin) : 'hecho'] })] }),
      ],
    })
  }
  if (s.estado === 'fallo') {
    const error = (s.resultado || '').trim().split('\n').slice(-2).join('\n')
    return caja(Text({ key: 'que', color: 'red', weight: 'semibold', wrap: 'wrap', children: ['✗  ' + a.etiqueta] }), [
      comando,
      ...(error ? [Text({ key: 'error', color: 'red', dimColor: true, size: 'caption', wrap: 'wrap', children: ['   ' + error] })] : []),
      ...meta(hora(s.fin) ? 'falló ' + hora(s.fin) : 'falló'),
      Box({ key: 'reintentar', paddingLeft: 3, children: [Button({ key: 'b-' + a.id, label: a.tecla + '  ↻ Reintentar', hotkey: a.tecla, onPress: () => pedir($, a) })] }),
    ])
  }
  // «todavía no» (FND-082): el comando dijo que su condición aún no se cumple; se ve en gris y se puede volver a probar.
  const aviso = a.aviso
    ? [Text({ key: 'aviso', dimColor: true, size: 'caption', wrap: 'wrap', children: ['   todavía no' + (hora(a.aviso_en) ? ' · ' + hora(a.aviso_en) : '') + ' — ' + a.aviso.trim().split('\n').slice(-1)[0]] })]
    : []
  return caja(Button({ key: 'b-' + a.id, label: a.tecla + '  ' + a.etiqueta, hotkey: a.tecla, variant: 'primary', onPress: () => pedir($, a) }), [comando, ...aviso, ...meta()])
}

// enZona: la confirmación pendiente y el último resultado, si son de esta sección (FND-090). El recuadro del resultado
// es verde (✓), rojo (✗) o, si el comando dijo «todavía no», gris (◌, FND-082).
function enZona(Box, Text, Button, $, aqui) {
  const pregunta = confirmando && zona(confirmando) === aqui && aqui !== 'pedidos' ? [confirmacion(Box, Text, Button, $)] : []
  if (!resultado || resultado.zona !== aqui) return pregunta
  const marca = resultado.todaviaNo ? '◌ ' : resultado.ok ? '✓ ' : '✗ '
  const tinte = resultado.todaviaNo ? undefined : resultado.ok ? 'green' : 'red'
  return [
    ...pregunta,
    Box({
      key: 'resultado',
      flexDirection: 'column',
      borderStyle: 'single',
      borderColor: tinte,
      borderDimColor: true,
      paddingX: 1,
      children: [
        Text({ color: tinte, dimColor: !tinte, weight: 'semibold', wrap: 'wrap', children: [marca + resultado.titulo + (resultado.todaviaNo ? ' · todavía no' : '')] }),
        Text({ dimColor: true, size: 'caption', wrap: 'wrap', children: [resultado.texto] }),
      ],
    }),
  ]
}

// paraTi lista los comandos que la IA dejó para la persona: primero los abiertos, en el orden que da qdd (cada uno
// después de lo que espera, FND-080), y al final los hechos. El que se está confirmando cambia su tarjeta por la
// confirmación; los demás siguen en su lugar.
function paraTi(Box, Text, Button, $, lista, ancho) {
  if (lista.length === 0) return []
  const hechos = lista.filter((a) => estadoDe(a).estado === 'hecho')
  const abiertos = lista.filter((a) => estadoDe(a).estado !== 'hecho')
  const porHacer = lista.filter(ejecutable).length
  const enEspera = lista.filter((a) => estadoDe(a).estado === 'espera').length
  const corriendo = lista.filter((a) => estadoDe(a).estado === 'ejecutando').length
  const nada = corriendo ? [] : ['nada por hacer']
  const resumen = [...(porHacer ? [porHacer + ' por hacer'] : nada), ...(corriendo ? [corriendo + ' ejecutando'] : []), ...(enEspera ? [enEspera + ' en espera'] : []), ...(hechos.length ? [hechos.length + (hechos.length === 1 ? ' hecho' : ' hechos')] : [])].join(' · ')
  const cuerpo = [...abiertos, ...hechos].map((a) => (confirmando?.id === a.id ? confirmacion(Box, Text, Button, $) : tarjeta(Box, Text, Button, $, a)))
  return [
    separador(Box, Text, 't-pedidos', '▶ Para ti', porHacer ? 'yellow' : undefined, resumen, ancho),
    Box({ key: 'pedidos', flexDirection: 'column', rowGap: 1, children: cuerpo }),
    ...enZona(Box, Text, Button, $, 'pedidos'),
  ]
}

// acciones: las herramientas del proyecto en una fila de botones (FND-090); el comando que corre cada una se ve al
// confirmarla. «Abrir carpeta» va arriba, junto a la carpeta, cada ✕ en la fila de su tarea y los pedidos en «Para ti».
// Solo los tipos que se dibujan aquí y con tecla: uno que este panel no conoce no sale como «undefined» (FND-092).
function acciones(Box, Text, Button, $, todas, ancho) {
  const conocidas = todas.filter((a) => a.tipo === 'aprobar' || a.tipo === 'comando')
  const todasTeclas = teclas(conocidas)
  const lista = conocidas.filter((a) => todasTeclas[a.id])
  if (lista.length === 0) return []
  const asignadas = teclas(lista)
  const botones = lista.map((a) => Button({ key: 'b-' + a.id, label: asignadas[a.id] + '  ' + a.etiqueta, hotkey: asignadas[a.id], onPress: () => pedir($, a) }))
  return [
    separador(Box, Text, 't-acciones', 'Acciones', undefined, '', ancho),
    Box({ key: 'botones', flexDirection: 'row', flexWrap: 'wrap', columnGap: 2, children: botones }),
    ...enZona(Box, Text, Button, $, 'acciones'),
  ]
}

// pie: cómo se usa y la versión de qdd, en una línea atenuada.
function pie(Text, version) {
  return Text({ key: 'pie', dimColor: true, italic: true, size: 'caption', wrap: 'wrap', children: ['⏳/❌ clic → la IA lo atiende · botones: clic o tecla · foco: /tareas' + (version ? ' · ' + version : '')] })
}

function sinEstado(Box, Text, motivo) {
  return Box({
    key: 'sin-estado',
    flexDirection: 'column',
    children: [
      Text({ dimColor: true, wrap: 'wrap', children: [motivo || 'No hay tareas en este proyecto.'] }),
      Text({ dimColor: true, wrap: 'wrap', children: ["Se crea con 'qdd estado init' (blueprint estado-por-tareas, destino «archivo»)."] }),
    ],
  })
}

function dibujar($, Box, Text, Button, ancho) {
  if (!datos) return sinEstado(Box, Text, falla)
  const e = datos.estado
  const lista = datos.acciones || []
  const cuerpo = e ? [tabla(Box, Text, Button, $, e, lista, ancho)] : [sinEstado(Box, Text, datos.motivo)]
  return Box({
    flexDirection: 'column',
    children: [
      cabecera(Box, Text, Button, $, e, lista),
      ...cuerpo,
      ...paraTi(Box, Text, Button, $, pedidos(lista), ancho),
      ...fallas(Box, Text, e?.fallas || [], ancho),
      ...acciones(Box, Text, Button, $, lista, ancho),
      Box({ marginTop: 1, children: [pie(Text, datos.version)] }),
    ],
  })
}

function hayQueMostrar() {
  if (!datos) return false
  const frentes = datos.estado ? datos.estado.frentes.length : 0
  const fichas = (datos.acciones || []).filter((a) => a.tipo === 'aprobar' || (a.tipo === 'pedido' && ejecutable(a))).length
  return frentes + fichas > 0
}

export function register(on) {
  on('session.start', async ($, e, next) => {
    await leer($)
    $.clock.every(INTERVALO_MS, async () => {
      await leer($)
    })
    $.clock.every(LATIDO_MS, async () => {
      latir($)
    })
    // Se abre solo si hay tareas, fichas por aprobar o comandos pedidos; después, cada pedido nuevo lo vuelve a abrir.
    if (hayQueMostrar()) await $.ui.open({ id: PANE, title: TITULO })
    iniciado = true
    try {
      await $.command.register({ name: 'tareas', description: 'Abre el panel de tareas (/tareas cerrar lo cierra)', immediate: true })
    } catch (err) {
      $.ui.log('no se registró /tareas: ' + String(err?.message ?? err))
    }
    return next(e)
  })

  on('command.run', { command: 'tareas' }, async ($, e) => {
    if ((e.args || '').trim() === 'cerrar') {
      await $.ui.close({ id: PANE })
      return {}
    }
    await leer($)
    await $.ui.open({ id: PANE, title: TITULO, focus: true })
    return {}
  })

  // Pulsos de vida (§3.13): cada herramienta del agente y cada subagente laten la tarea en curso. next(e) va primero
  // y sin esperar al pulso.
  on('tool.call', async ($, e, next) => {
    pulsar($, 'agente')
    return next(e)
  })

  on('agent.spawn', async ($, e, next) => {
    pulsar($, 'subagente')
    return next(e)
  })

  // Un ciclo por mensaje (§3.9): lo que ya se vio (el resultado y los pedidos hechos) sale del tablero, y qdd anota el
  // modo de permisos de la IA para decirle cuándo pedir autorización. next(e) no espera a qdd.
  on('classic.UserPromptSubmit', async ($, e, next) => {
    resultado = null
    for (const [id, l] of locales) if (l.estado === 'hecho') locales.delete(id)
    $.ui.invalidate('ui.render')
    const bin = await binario($)
    $.process.run([bin, 'consola', 'ciclo', '--modo', String(e.permission_mode || '')], { timeoutMs: TOPE_MS }).catch(() => {})
    return next(e)
  })

  on('ui.render', { component: 'Pane' }, async ($, e, next) => {
    if (e.requestId !== PANE) return next(e)
    const { Box, Text, Button } = $.ui.resolve(e)
    return dibujar($, Box, Text, Button, e.props.bodyColumns || 40)
  })
}
