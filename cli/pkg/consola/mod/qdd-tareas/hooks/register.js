// Mod de Claude Code con las tareas en curso del proyecto (QDD, RFC 0006 §3.4.1, §3.7, §3.9 y §3.10; blueprint
// consola-claude-code). Dibuja el panel liviano: una línea de resumen en color, la tabla de las 4 tareas pendientes más
// nuevas con su ✕ para descartarlas y los botones de las acciones. Orden: tareas, Acciones, «▶ Para ti» (los comandos
// que la IA dejó para la persona), ⏳ Esperando y ❌ Fallas. Cada pedido se ve en su estado: pendiente (botón con su
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
let resultado = null // { titulo, ok, texto }
let ocupado = false
let iniciado = false // ya pasó la primera lectura de session.start
// CODIGO_TODAVIA_NO es la salida de un pedido cuya condición aún no se cumple (qdd: consola.CodigoTodaviaNo).
const CODIGO_TODAVIA_NO = 75
const pedidosVistos = new Set() // ids de los pedidos que ya abrieron el panel
// locales: lo que el panel ya sabe de un pedido antes que qdd (o con un qdd anterior a 2.6.2, que no guarda el
// estado): id → { estado, fin }. Así un pedido aceptado nunca vuelve a ofrecerse.
const locales = new Map()
let inicio = '' // HOME, para acortar la carpeta a ~/…
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
    const r = await $.process.run([bin, 'consola', 'panel', '--json'], { timeoutMs: TOPE_MS })
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
function estadoDe(a) {
  const local = locales.get(a.id)
  const remoto = a.estado || 'pendiente'
  if (local && (local.estado === 'ejecutando' || remoto === 'pendiente')) return local
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

// celdas alinea los símbolos con las abreviaturas: ✅ ⏳ ❌ ocupan 2 columnas; — (no aplica), · (sin empezar) y ? una.
const ANGOSTOS = new Set(['—', '·', '?'])
function celdas(simbolos) {
  return simbolos.map((s) => (ANGOSTOS.has(s) ? s + ' ' : s)).join(' ')
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

function titulo(Text, key, texto, color) {
  return Text({ key, color, weight: 'bold', size: 'title', children: [texto] })
}

// corta deja la carpeta en ~/… y, si no cabe, muestra el final (lo que distingue un proyecto de otro).
function corta(ruta, ancho) {
  const casa = inicio && (ruta === inicio || ruta.startsWith(inicio + '/')) ? '~' + ruta.slice(inicio.length) : ruta
  if (casa.length <= ancho) return casa
  return '…' + casa.slice(casa.length - ancho + 1)
}

// carpeta (§3.9): a la derecha, la carpeta del proyecto y el botón para abrirla con el gestor de archivos del sistema.
// El panel no abre nada por su cuenta: pide 'abrir-carpeta', que qdd ofrece solo si el sistema tiene abridor.
function carpeta(Box, Text, Button, $, ruta, abrirla, ancho) {
  if (!ruta) return []
  const boton = abrirla ? [Button({ key: 'b-abrir-carpeta', label: 'o  Abrir carpeta', hotkey: 'o', plain: true, onPress: () => pedir($, abrirla) })] : []
  const espacio = Math.max(12, ancho - (abrirla ? 20 : 4))
  return [
    Box({
      key: 'carpeta',
      flexDirection: 'row',
      justifyContent: 'flex-end',
      columnGap: 1,
      children: [Text({ key: 'ruta', dimColor: true, wrap: 'truncate-start', children: ['📂 ' + corta(ruta, espacio)] }), ...boton],
    }),
  ]
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

// cabecera: título con la hora y la versión de qdd a la derecha, y el resumen en una línea con el color de cada conteo.
function cabecera(Box, Text, e, version, lista = []) {
  const cuando = e.hace || hora(e.actualizado)
  const aviso = e.viejo
    ? [Text({ key: 'viejo', color: 'yellow', wrap: 'wrap', children: ['⚠ Sin actualizar ' + e.hace + ': lo que dice ⏳ puede haber cambiado'] })]
    : []
  const paraTiN = lista.filter((a) => a.tipo === 'pedido' && ejecutable(a)).length
  const paraTiTexto = paraTiN > 0
    ? [Text({ dimColor: true, children: ['·'] }), Text({ key: 'r-para-ti', color: 'yellow', weight: 'semibold', children: ['▶ ' + paraTiN + ' para ti'] })]
    : []
  return Box({
    key: 'cabecera',
    flexDirection: 'column',
    marginTop: 1,
    children: [
      Box({
        flexDirection: 'row',
        justifyContent: 'space-between',
        children: [
          Text({ key: 'titulo', color: 'cyan', weight: 'bold', size: 'title', children: [TITULO + (cuando ? ' · ' + cuando : '')] }),
          Text({ key: 'version', dimColor: true, size: 'caption', children: [version || ''] }),
        ],
      }),
      Box({
        key: 'resumen',
        flexDirection: 'row',
        columnGap: 1,
        children: [
          Text({ key: 'r-tareas', color: 'cyan', weight: 'semibold', children: [cuantas(e)] }),
          Text({ dimColor: true, children: ['·'] }),
          Text({ key: 'r-esperas', color: 'yellow', weight: 'semibold', children: ['⏳ ' + etapas(total(e.esperan, e.esperas), 'espera', 'esperan')] }),
          Text({ dimColor: true, children: ['·'] }),
          Text({ key: 'r-fallas', color: 'red', weight: 'semibold', children: ['❌ ' + etapas(total(e.fallan, e.fallas), 'falla', 'fallan')] }),
          ...paraTiTexto,
        ],
      }),
      ...aviso,
    ],
  })
}

// tabla: una fila por tarea con sus etapas y, al final, la ✕ que la descarta (pide «s  Sí»: escribe en estado.yaml).
// La ✕ se pulsa con el mouse; las tareas más antiguas que no entran se cuentan debajo.
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
        Text({ ...estilo, children: [derecha] }),
        ...cierre,
      ],
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
      fila('cab', 'Tarea', e.abreviaturas.join(' '), { dimColor: true, size: 'caption' }, []),
      ...e.frentes.map((f) => fila('f-' + f.id, f.nombre, celdas(f.simbolos), { color: colorFrente(f) }, [...enCurso(Text, f), ...equis(f)])),
      ...antiguas,
      Text({ dimColor: true, italic: true, size: 'caption', wrap: 'wrap', children: [leyenda] }),
    ],
  })
}

// seccion: «• tarea › etapa» y debajo «→ texto», con de quién atenuado en las esperas.
function seccion(Box, Text, key, nombre, color, items, conQuien) {
  if (items.length === 0) return []
  return [
    Box({
      key,
      flexDirection: 'column',
      marginTop: 1,
      children: [
        titulo(Text, key + '-t', nombre + ' (' + items.length + ')', color),
        ...items.map((p, i) =>
          Box({
            key: key + '-' + i,
            flexDirection: 'column',
            children: [
              Text({ wrap: 'wrap', children: ['• ' + p.frente + ' › ' + p.etapa] }),
              Text({ dimColor: true, wrap: 'wrap', children: ['  → ' + p.texto + (conQuien ? ' · de ' + (p.de_quien || '¿?') : '')] }),
            ],
          }),
        ),
      ],
    }),
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
  resultado = { titulo: a.etiqueta, ok: true, texto: 'ejecutando ' + a.detalle + '…' }
  $.ui.invalidate('ui.render')
  const bin = await binario($)
  const argv = [bin, 'consola', 'accion', a.id, ...(a.confirmar ? ['--confirmado'] : [])]
  try {
    const largo = a.id === 'auditar' || a.tipo === 'pedido'
    const r = await $.process.run(argv, { timeoutMs: largo ? TOPE_AUDITAR_MS : TOPE_ACCION_MS })
    const texto = ((r.stdout || '') + (r.stderr || '')).trim().split('\n').slice(-LINEAS_RESULTADO).join('\n')
    resultado = { titulo: a.etiqueta, ok: r.exitCode === 0, texto: texto || 'sin salida' }
    todaviaNo = r.exitCode === CODIGO_TODAVIA_NO
    if (todaviaNo) resultado = { ...resultado, todaviaNo: true }
  } catch (err) {
    resultado = { titulo: a.etiqueta, ok: false, texto: String(err?.message ?? err) }
  }
  if (esPedido) locales.set(a.id, { estado: resultado.ok ? 'hecho' : 'fallo', fin: new Date().toISOString(), resultado: resultado.texto })
  // «todavía no» (código 75, FND-082): vuelve a pendiente; manda lo que diga qdd, con su aviso.
  if (esPedido && todaviaNo) locales.delete(a.id)
  ocupado = false
  firma = ''
  await leer($)
  $.ui.invalidate('ui.render')
}

// descartar quita un pedido de la cola sin ejecutarlo.
async function descartar($, a) {
  ocupado = true
  confirmando = null
  const bin = await binario($)
  try {
    const r = await $.process.run([bin, 'consola', 'descartar', a.id.slice('pedido:'.length)], { timeoutMs: TOPE_MS })
    resultado = { titulo: 'Descartado: ' + a.etiqueta, ok: r.exitCode === 0, texto: ((r.stdout || '') + (r.stderr || '')).trim() || 'sin salida' }
  } catch (err) {
    resultado = { titulo: 'Descartar ' + a.etiqueta, ok: false, texto: String(err?.message ?? err) }
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
      ...(a.carpeta ? [Text({ dimColor: true, size: 'caption', wrap: 'truncate-start', children: ['En: ' + a.carpeta] })] : []),
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

// tarjeta dibuja un pedido en tres líneas iguales para todos los estados: qué (con su marca o su botón), el comando
// exacto y dónde/cuándo. Solo pendiente y con falla tienen botón; en espera, ejecutando y hecho van en gris.
function tarjeta(Box, Text, Button, $, a) {
  const s = estadoDe(a)
  const apagado = s.estado === 'espera' || s.estado === 'ejecutando' || s.estado === 'hecho'
  const comando = Text({ key: 'cmd', mono: true, dimColor: apagado, wrap: 'wrap', children: ['   $ ' + a.detalle] })
  const donde = (extra) => Text({ key: 'en', dimColor: true, size: 'caption', wrap: 'truncate-start', children: ['   en ' + a.carpeta + (extra ? ' · ' + extra : '')] })
  const caja = (cabeza, resto) => Box({ key: 'pedido-' + a.id, flexDirection: 'column', children: [cabeza, comando, ...resto] })
  if (s.estado === 'espera') {
    const de = (a.espera || []).map((t) => '[' + t + ']').join(' ')
    return caja(Text({ key: 'que', dimColor: true, wrap: 'wrap', children: ['◌  ' + a.etiqueta + '  · espera a ' + (de || 'otro pedido')] }), [donde('')])
  }
  if (s.estado === 'ejecutando') {
    return caja(Text({ key: 'que', dimColor: true, wrap: 'wrap', children: ['⟳  ' + a.etiqueta + '  · ejecutando…'] }), [donde(hora(s.inicio) ? 'desde ' + hora(s.inicio) : '')])
  }
  if (s.estado === 'hecho') {
    return caja(Text({ key: 'que', dimColor: true, wrap: 'wrap', children: ['✓  ' + a.etiqueta] }), [donde(hora(s.fin) ? 'hecho ' + hora(s.fin) : 'hecho')])
  }
  if (s.estado === 'fallo') {
    const error = (s.resultado || '').trim().split('\n').slice(-2).join('\n')
    return caja(Text({ key: 'que', color: 'red', weight: 'semibold', wrap: 'wrap', children: ['✗  ' + a.etiqueta] }), [
      ...(error ? [Text({ key: 'error', color: 'red', dimColor: true, size: 'caption', wrap: 'wrap', children: ['   ' + error] })] : []),
      donde(hora(s.fin) ? 'falló ' + hora(s.fin) : 'falló'),
      Box({ key: 'reintentar', paddingLeft: 3, children: [Button({ key: 'b-' + a.id, label: a.tecla + '  ↻ Reintentar', hotkey: a.tecla, onPress: () => pedir($, a) })] }),
    ])
  }
  // «todavía no» (FND-082): el comando dijo que su condición aún no se cumple; se ve en gris y se puede volver a probar.
  const aviso = a.aviso
    ? [Text({ key: 'aviso', dimColor: true, size: 'caption', wrap: 'wrap', children: ['   todavía no' + (hora(a.aviso_en) ? ' · ' + hora(a.aviso_en) : '') + ' — ' + a.aviso.trim().split('\n').slice(-1)[0]] })]
    : []
  return caja(Button({ key: 'b-' + a.id, label: a.tecla + '  ' + a.etiqueta, hotkey: a.tecla, variant: 'primary', onPress: () => pedir($, a) }), [...aviso, donde('')])
}

// paraTi lista los comandos que la IA dejó para la persona: primero los abiertos, en el orden que da qdd (cada uno
// después de lo que espera, FND-080), y al final los hechos. Mientras se confirma uno, en su lugar va la confirmación.
function paraTi(Box, Text, Button, $, lista) {
  if (lista.length === 0) return []
  const hechos = lista.filter((a) => estadoDe(a).estado === 'hecho')
  const abiertos = lista.filter((a) => estadoDe(a).estado !== 'hecho')
  const porHacer = lista.filter(ejecutable).length
  const enEspera = lista.filter((a) => estadoDe(a).estado === 'espera').length
  const resumen = [porHacer ? porHacer + ' por hacer' : 'nada por hacer', ...(enEspera ? [enEspera + ' en espera'] : []), ...(hechos.length ? [hechos.length + (hechos.length === 1 ? ' hecho' : ' hechos')] : [])].join(' · ')
  const cuerpo = confirmando
    ? confirmando.tipo === 'pedido'
      ? [confirmacion(Box, Text, Button, $)]
      : []
    : [...abiertos, ...hechos].map((a) => tarjeta(Box, Text, Button, $, a))
  return [
    Box({
      key: 't-pedidos',
      flexDirection: 'row',
      justifyContent: 'space-between',
      children: [titulo(Text, 't-pedidos-t', '▶ Para ti', porHacer ? 'yellow' : undefined), Text({ dimColor: true, size: 'caption', children: [resumen] })],
    }),
    ...cuerpo,
  ]
}

function acciones(Box, Text, Button, $, todas) {
  // «Abrir carpeta» va arriba, junto a la carpeta, y cada ✕ en la fila de su tarea.
  const lista = todas.filter((a) => a.tipo !== 'abrir' && a.tipo !== 'descartar')
  if (lista.length === 0) return []
  const asignadas = teclas(lista)
  const filas = lista.filter((a) => a.tipo !== 'pedido').map((a) =>
    Box({
      key: 'fila-' + a.id,
      flexDirection: 'row',
      columnGap: 1,
      children: [
        Button({ key: 'b-' + a.id, label: asignadas[a.id] + '  ' + a.etiqueta, hotkey: asignadas[a.id], onPress: () => pedir($, a) }),
        Text({ dimColor: true, size: 'caption', wrap: 'truncate-end', children: [a.detalle + (a.confirmar ? '' : ' · solo lectura')] }),
      ],
    }),
  )
  // El recuadro del último resultado: verde (✓), rojo (✗) o, si el comando dijo «todavía no», gris (◌, FND-082).
  const marca = resultado?.todaviaNo ? '◌ ' : resultado?.ok ? '✓ ' : '✗ '
  const tinte = resultado?.todaviaNo ? undefined : resultado?.ok ? 'green' : 'red'
  const salida = resultado
    ? [
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
    : []
  return [
    Box({
      key: 'acciones',
      flexDirection: 'column',
      marginTop: 1,
      rowGap: 1,
      children: [
        titulo(Text, 't-acciones', 'Acciones', 'magenta'),
        ...(confirmando ? (confirmando.tipo === 'pedido' ? [] : [confirmacion(Box, Text, Button, $)]) : filas),
        ...paraTi(Box, Text, Button, $, pedidos(lista)),
        ...salida,
        Text({ dimColor: true, italic: true, size: 'caption', wrap: 'wrap', children: ['Clic en un botón o en la ✕ de una tarea; con el foco en el panel (clic o /tareas), también la tecla de cada botón.'] }),
      ],
    }),
  ]
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
  const arriba = carpeta(Box, Text, Button, $, datos.carpeta, lista.find((a) => a.id === 'abrir-carpeta'), ancho)
  if (!e) return Box({ flexDirection: 'column', children: [...arriba, sinEstado(Box, Text, datos.motivo), ...acciones(Box, Text, Button, $, lista)] })
  return Box({
    flexDirection: 'column',
    children: [
      ...arriba,
      cabecera(Box, Text, e, datos.version, lista),
      tabla(Box, Text, Button, $, e, lista, ancho),
      ...acciones(Box, Text, Button, $, lista),
      ...seccion(Box, Text, 's-fallas', '❌ Fallas', 'red', e.fallas, false),
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
    inicio = (await $.env.get('HOME')) || ''
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
