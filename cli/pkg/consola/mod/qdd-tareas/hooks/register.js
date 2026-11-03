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
// TOPE_CICLO_MS: el mensaje de la persona espera al ciclo (con lo que el panel espera de ella) a lo más esto.
const TOPE_CICLO_MS = 1500
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

// ── Materializar (pedido del usuario, 2026-11-01) ──────────────────────────────────────────────────────────────────
// Lo que cambia aparece como las ventanas del usuario (hypr-materialize): caracteres en el acento, bloques que se
// afinan y el texto nítido; lo que se va, al revés. Solo lo mínimo que cambió (una celda, un nombre, una tarjeta), y
// solo con los efectos encendidos ('qdd consola efectos'). qdd calcula los cuadros una vez por cambio ('qdd consola
// materializar'); el elemento Client del panel (materializar.js) solo los reproduce. Sin Client (otra superficie) o
// sin cuadros, se dibuja lo de siempre.
const vistos = new Map() // clave → huella de lo último dibujado
const animaciones = new Map() // clave → { huella, n, cuadros, ms, fallo } (lo que se materializa)
const fantasmas = new Map() // clave → { lineas, n, cuadros, ms, fallo } (un pedido que se va)
const tarjetasVistas = new Map() // id de pedido → sus líneas, para que al irse se vaya materializando
let Client = null
let contador = 0

function conEfectos() {
  return Boolean(Client && datos?.efectos?.activo)
}

// materializa: plano, o su materialización si cambió lo que muestra (huella). La primera vez que el panel dibuja algo
// también se materializa: el panel aparece como una ventana.
function materializa($, Text, clave, lineas, plano, huella = JSON.stringify(lineas)) {
  const antes = vistos.get(clave)
  vistos.set(clave, huella)
  if (!conEfectos()) return plano
  const a = animaciones.get(clave)
  if (a && a.huella === huella) return a.fallo ? plano : a.cuadros ? reproductor(clave, a) : enBlanco(Text, clave, lineas)
  if (antes === huella) return plano
  const nueva = { huella, n: ++contador }
  animaciones.set(clave, nueva)
  pedirCuadros($, nueva, lineas, true)
  return enBlanco(Text, clave, lineas)
}

function reproductor(clave, a) {
  return Client({ key: 'mat-' + clave + '-' + a.n, module: './materializar.js', props: { cuadros: a.cuadros, ms: a.ms, clave } })
}

// enBlanco ocupa lo mismo que el texto mientras llegan los cuadros (unos 30 ms): nada se mueve de lugar.
function enBlanco(Text, clave, lineas) {
  const blanco = (t) => [...t].map((c) => (c === '\uFE0F' ? '' : c.codePointAt(0) >= 0x1f000 || /[\u2300-\u27bf\u2b00-\u2bff]/.test(c) ? '  ' : ' ')).join('')
  return Text({ key: 'blanco-' + clave, children: [lineas.map((l) => l.map((x) => blanco(x.texto)).join('')).join('\n')] })
}

async function pedirCuadros($, anim, lineas, abrir) {
  try {
    const bin = await binario($)
    const pedido = { lineas, abrir, semilla: anim.n, acento: datos?.efectos?.acento || ACENTO }
    const r = await $.process.run([bin, 'consola', 'materializar'], { stdin: JSON.stringify(pedido), timeoutMs: TOPE_MS })
    const j = r.exitCode === 0 ? JSON.parse(r.stdout) : null
    anim.cuadros = Array.isArray(j?.cuadros) && j.cuadros.length ? j.cuadros : null
    anim.ms = j?.cuadro_ms || 40
    anim.fallo = !anim.cuadros
  } catch {
    anim.fallo = true
  }
  $.ui.invalidate('ui.render')
}

// seVan: los pedidos que estaban y ya no están se van materializando al revés (0,4 s), en el lugar del final.
function seVan($, Text, ids) {
  for (const [id, lineas] of tarjetasVistas) {
    if (ids.has(id)) continue
    tarjetasVistas.delete(id)
    if (!conEfectos()) continue
    const g = { lineas, n: ++contador }
    fantasmas.set('g:' + id, g)
    pedirCuadros($, g, lineas, false)
  }
  return [...fantasmas].map(([clave, g]) => (g.fallo ? null : g.cuadros ? reproductor(clave, g) : enBlanco(Text, clave, g.lineas))).filter(Boolean)
}

// lineasDeTarjeta: lo que se ve de un pedido, como líneas para materializar.
function lineasDeTarjeta(a) {
  const s = estadoDe(a)
  const marca = { espera: '◌', ejecutando: '⟳', hecho: '✓', fallo: '✗', armado: '◷' }[s.estado]
  const cabeza = marca ? [{ texto: marca + '  ' + a.etiqueta, tenue: s.estado !== 'fallo', color: s.estado === 'fallo' ? 'red' : undefined }] : [{ texto: a.tecla + '  ' + a.etiqueta, negrita: true }]
  return [cabeza, [{ texto: '   $ ' + primeraLinea(a.detalle), tenue: true }]]
}

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
  if (huella === firma) return avisar($)
  firma = huella
  datos = nuevo
  falla = motivo
  // Lo aceptado que qdd ya anotó (ejecutando, hecho o con falla) deja de ser local.
  for (const a of nuevo?.acciones || []) if (locales.has(a.id) && a.estado && a.estado !== 'pendiente') locales.delete(a.id)
  verPulsos()
  $.ui.invalidate('ui.render')
  await abrirSiLlegaUnPedido($)
  await avisar($)
}

// Aviso al chat (pedido del usuario, 2026-11-02: «el chat principal tiene que enterarse y verse una leyenda cuando
// acepto la tarea en el panel»). Al confirmar un pedido, el panel le dice a qdd qué sesión lo confirma y le deja a la
// IA una nota invisible de que corre aparte; la banda de arriba del prompt dice «▶ … en marcha» (o «◷ … armado»)
// mientras corre. Al terminar, qdd marca el pedido «por avisar» a esa sesión: el panel lo reclama con «qdd consola
// avisado» (sale con 0 solo la primera vez, aunque haya varias sesiones o un reload del mod), envía el mensaje de qdd
// a la IA, que sigue sola, y notifica en el escritorio.
let sesion = null // id de esta sesión; '' si no se pudo leer (entonces no hay aviso)
const teclasConfirmadas = new Map() // id de pedido → la tecla con que se confirmó (la banda la nombra)
const reclamados = new Set() // pedidos cuyo aviso ya se pidió a qdd en esta carga

async function sesionPropia($) {
  if (sesion !== null) return sesion
  try {
    sesion = String((await $.session.id()) || '')
  } catch {
    sesion = ''
  }
  return sesion
}

// notaParaLaIA: la IA sabe que el pedido ya corre (no lo repite ni lo vuelve a pedir) y que el resultado le llega solo.
async function notaParaLaIA($, a) {
  const texto = '[qdd] La persona confirmó en el panel el pedido «' + a.etiqueta + '»: corre aparte (o queda armado hasta que esté listo); no lo ejecutes ni lo pidas de nuevo. Cuando termine te llega su resultado.'
  try {
    await $.session.append({ message: { type: 'user', content: [{ type: 'text', text: texto }] } })
  } catch (err) {
    $.ui.log('no se anotó el pedido para la IA: ' + String(err?.message ?? err))
  }
}

// enMarchaAqui: los pedidos que esta sesión confirmó y todavía corren o esperan armados.
function enMarchaAqui() {
  return pedidos(datos?.acciones || []).filter((a) => {
    const { estado } = estadoDe(a)
    const propio = a.sesion ? a.sesion === sesion : teclasConfirmadas.has(a.id)
    return propio && (estado === 'ejecutando' || estado === 'armado')
  })
}

// banda: una línea por pedido en marcha, arriba del prompt; sin ninguno, la banda es la de siempre.
function banda($, e, next) {
  const lista = enMarchaAqui()
  if (e.props.hasSurvey || lista.length === 0) return next(e)
  const { Box, Text } = $.ui.resolve(e)
  const filas = lista.slice(0, Math.max(1, e.props.maxRows || 1)).map((a) => {
    const tecla = teclasConfirmadas.get(a.id)
    const nombre = (tecla ? '[' + tecla + '] ' : '') + a.etiqueta
    if (estadoDe(a).estado === 'armado') return Text({ key: 'banda-' + a.id, color: 'yellow', children: '◷ ' + nombre + ' · armado: corre solo cuando esté listo' })
    return Text({ key: 'banda-' + a.id, color: ACENTO, children: '▶ ' + nombre + ' · en marcha' })
  })
  return Box({ key: 'banda', flexDirection: 'column', children: filas })
}

// avisar reclama y entrega los avisos de esta sesión. Uno que qdd ya no ofrece (reintentado, ya avisado) se olvida.
async function avisar($) {
  const mia = await sesionPropia($)
  const listos = pedidos(datos?.acciones || []).filter((a) => a.avisar && a.mensaje && mia && a.sesion === mia)
  for (const id of [...reclamados]) if (!listos.some((a) => a.id === id)) reclamados.delete(id)
  for (const a of listos.filter((x) => !reclamados.has(x.id))) await entregar($, a, mia)
}

async function entregar($, a, mia) {
  reclamados.add(a.id)
  const bin = await binario($)
  let r = null
  try {
    r = await $.process.run([bin, 'consola', 'avisado', a.id, '--sesion', mia], { timeoutMs: TOPE_MS })
  } catch (err) {
    reclamados.delete(a.id)
    $.ui.log('no se reclamó el aviso del pedido: ' + String(err?.message ?? err))
    return
  }
  if (r.exitCode !== 0) return
  teclasConfirmadas.delete(a.id)
  $.ui.invalidate('ui.render')
  $.prompt.submit({ text: a.mensaje }).catch((err) => $.ui.log('no se envió a la IA: ' + String(err?.message ?? err)))
  const fallo = a.estado === 'fallo'
  $.ui.notify((fallo ? '✗ ' : '✓ ') + a.etiqueta + (fallo ? ' falló' : ' terminó'), { title: 'qdd' }).catch((err) => $.ui.log('no se notificó: ' + String(err?.message ?? err)))
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
  // Un PR abierto (a.pr) es opcional: se ofrece, pero no abre el panel.
  if (!iniciado || !nuevos.some((a) => !a.pr)) return
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

// ejecutable: se puede confirmar. Uno que espera a otro también (con su tecla): queda armado y corre cuando el otro
// termine (armado.go).
function ejecutable(a) {
  const { estado } = estadoDe(a)
  return estado === 'pendiente' || estado === 'fallo' || (estado === 'espera' && Boolean(a.tecla))
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
    children: [
      Box({ flexShrink: 0, children: [Text({ key: 'titulo', color: 'cyan', weight: 'bold', size: 'title', children: [TITULO + (cuando ? ' · ' + cuando : '')] })] }),
      Box({ flexDirection: 'row', flexShrink: 1, columnGap: 2, children: [...carpeta, Button({ key: 'leer', label: '↻', hotkey: 'l', plain: true, onPress: () => releer($) }), Button({ key: 'blueprints', label: '📦 Blueprints', hotkey: 'p', plain: true, onPress: () => abrirBlueprints($) }), Button({ key: 'config', label: '⚙ Config', hotkey: 'k', plain: true, onPress: () => abrirConfig($) })] }),
    ],
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
// Árbol (blueprint estado-por-tareas, objetivos y dependencias): qdd entrega las filas en orden, cada una con su
// prefijo («│  └─ »), su objetivo y a quién espera; el panel pone la cabecera de cada objetivo (el foco primero) y
// deja las etapas en columnas a la derecha. Pinchar el nombre de una tarea o de un objetivo le envía a la IA la
// pregunta que trae qdd («¿Cómo vamos con…?»).
// Ciclos (estado-por-tareas, EPT-K05): un objetivo que sigue su propio ciclo trae sus etapas en e.ciclos; bajo su
// cabecera van las abreviaturas de ese ciclo y sus filas, un símbolo por etapa. La leyenda nombra cada ciclo.
function tabla(Box, Text, Button, $, e, lista, ancho) {
  const ciclos = e.ciclos || []
  const cicloDe = (o) => ciclos.find((c) => c.nombre === o.ciclo)
  const columnas = 3 * Math.max(e.etapas.length, ...ciclos.map((c) => c.etapas.length)) + 2
  const etiqueta = (f) => (f.prefijo || '') + f.nombre + (f.notas?.length ? ' · ' + f.notas.join(' · ') : '')
  const objetivos = e.objetivos || []
  const largo = Math.max('Tarea'.length, ...e.frentes.map((f) => etiqueta(f).length), ...objetivos.map((o) => o.nombre.length + 10))
  const nombreAncho = Math.max(8, Math.min(largo, ancho - columnas - 3))
  const fila = (key, nombre, derecha, estilo, cierre) =>
    Box({
      key,
      flexDirection: 'row',
      columnGap: 1,
      children: [
        Box({ width: nombreAncho, children: [nombre === null ? Text({ children: [''] }) : typeof nombre === 'string' ? Text({ ...estilo, wrap: 'truncate-end', children: [nombre] }) : nombre] }),
        ...derecha,
        ...cierre,
      ],
    })
  const etapasDe = (f) =>
    f.simbolos.map((s, i) => {
      const a = lista.find((x) => x.id === 'atender:' + f.id + ':' + i)
      const lineas = [[{ texto: celda(s), color: colorFrente(f) }]]
      if (a) return materializa($, Text, 'c:' + f.id + ':' + i, lineas, Button({ key: 'e-' + f.id + '-' + i, label: s, plain: true, onPress: () => atender($, a) }), s)
      return materializa($, Text, 'c:' + f.id + ':' + i, lineas, Text({ key: 'c-' + f.id + '-' + i, color: colorFrente(f), children: [celda(s)] }), s)
    })
  const equis = (f) => {
    const a = lista.find((x) => x.id === 'descartar:' + f.id)
    return a ? [Button({ key: 'x-' + f.id, label: '✕', plain: true, dimColor: true, onPress: () => pedir($, a) })] : []
  }
  // nombreDe: el prefijo atenuado, el nombre con su color y a quién espera; con pregunta de qdd, todo es un botón.
  const nombreDe = (f) => {
    const partes = [
      ...(f.prefijo ? [Text({ key: 'pre', dimColor: true, children: [f.prefijo] })] : []),
      Text({ key: 'nom', color: colorFrente(f), children: [f.nombre] }),
      ...(f.notas?.length ? [Text({ key: 'notas', color: 'yellow', dimColor: true, children: [' · ' + f.notas.join(' · ')] })] : []),
    ]
    const lineas = [[...(f.prefijo ? [{ texto: f.prefijo, tenue: true }] : []), { texto: f.nombre, color: colorFrente(f) }, ...(f.notas?.length ? [{ texto: ' · ' + f.notas.join(' · '), color: 'yellow', tenue: true }] : [])]]
    const plano = !f.pregunta ? (f.prefijo || f.notas?.length ? Text({ wrap: 'truncate-end', children: partes }) : f.nombre) : Button({ key: 'n-' + f.id, plain: true, onPress: () => preguntar($, f.pregunta), children: partes })
    return materializa($, Text, 'n:' + f.id, lineas, plano, etiqueta(f))
  }
  const filaTarea = (f) => fila('f-' + f.id, nombreDe(f), etapasDe(f), { color: colorFrente(f) }, [...enCurso(Text, f), ...equis(f)])
  const cabeceraObjetivo = (o) =>
    Box({
      key: 'o-' + o.id,
      flexDirection: 'row',
      columnGap: 1,
      marginTop: 1,
      children: [
        materializa(
          $,
          Text,
          'o:' + o.id,
          [[{ texto: '🎯 ' + o.nombre + (o.foco ? ' · foco' : ''), negrita: true, color: o.foco ? datos?.efectos?.acento || ACENTO : undefined }]],
          Button({ key: 'o-' + o.id + '-nombre', plain: true, onPress: () => preguntar($, o.pregunta), children: [Text({ key: 'o', weight: 'bold', color: o.foco ? ACENTO : undefined, children: ['🎯 ' + o.nombre + (o.foco ? ' · foco' : '')] })] }),
          o.nombre + (o.foco ? '·foco' : ''),
        ),
        Text({ key: 'o-estado', dimColor: true, wrap: 'truncate-end', children: [o.simbolo + ' ' + o.resumen] }),
      ],
    })
  const cabeceraCiclo = (o, c) => fila('cab-' + o.id, null, [Text({ dimColor: true, size: 'caption', children: [c.abreviaturas.join(' ')] })], {}, [])
  const filas = []
  if (objetivos.length === 0) filas.push(...e.frentes.map(filaTarea))
  for (const o of objetivos) {
    filas.push(cabeceraObjetivo(o))
    const c = cicloDe(o)
    if (c) filas.push(cabeceraCiclo(o, c))
    filas.push(...e.frentes.filter((f) => f.objetivo === o.id).map(filaTarea))
  }
  const sueltas = objetivos.length ? e.frentes.filter((f) => !f.objetivo) : []
  if (sueltas.length) {
    filas.push(Box({ key: 'o-sin', marginTop: 1, children: [Text({ dimColor: true, weight: 'bold', children: ['🎯 sin objetivo'] })] }))
    filas.push(...sueltas.map(filaTarea))
  }
  const nombrar = (etapas, abrev) => etapas.map((etapa, i) => abrev[i] + ' ' + etapa).join(' · ')
  const leyendas = [nombrar(e.etapas, e.abreviaturas), ...ciclos.map((c) => c.nombre + ': ' + nombrar(c.etapas, c.abreviaturas))]
  const antiguas = e.ocultos > 0 ? [Text({ key: 'ocultas', dimColor: true, size: 'caption', children: ['+' + e.ocultos + ' más antiguas · qdd estado ver'] })] : []
  return Box({
    key: 'tabla',
    flexDirection: 'column',
    marginTop: 1,
    children: [
      fila('cab', 'Tarea', [Text({ dimColor: true, size: 'caption', children: [e.abreviaturas.join(' ')] })], { dimColor: true, size: 'caption' }, []),
      ...filas,
      ...antiguas,
      ...leyendas.map((l, i) => Text({ key: 'leyenda-' + i, dimColor: true, italic: true, size: 'caption', wrap: 'wrap', children: [l] })),
      ...enZona(Box, Text, Button, $, 'tabla'),
    ],
  })
}

// preguntar envía a la conversación la pregunta que qdd da para una tarea u objetivo: la IA responde avance, bloqueos
// y siguiente paso, y actualiza el estado con evidencia.
function preguntar($, texto) {
  if (!texto) return
  resultado = { titulo: '¿Cómo vamos?', ok: true, texto: '→ enviado a la IA', zona: 'tabla' }
  $.prompt.submit({ text: texto }).catch((err) => $.ui.log('no se envió a la IA: ' + String(err?.message ?? err)))
  $.ui.invalidate('ui.render')
}

// releer: «l ↻» vuelve a leer el tablero en el acto, sin esperar a la próxima lectura.
async function releer($) {
  await leer($)
  $.ui.invalidate('ui.render')
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
  const mia = esPedido && a.confirmar ? await sesionPropia($) : ''
  const argv = [bin, 'consola', 'accion', a.id, ...(a.confirmar ? ['--confirmado'] : []), ...(mia ? ['--sesion', mia] : [])]
  let lanzado = false
  try {
    // Un pedido solo se lanza: qdd lo corre aparte y el panel queda libre (FND-089); el resultado llega por la cola.
    let r = await $.process.run(argv, { timeoutMs: a.id === 'auditar' ? TOPE_AUDITAR_MS : TOPE_ACCION_MS })
    // Un qdd anterior a --sesion no lo conoce: el pedido corre igual, sin aviso al terminar.
    if (mia && r.exitCode !== 0 && /unknown flag/.test(r.stderr || '')) r = await $.process.run(argv.slice(0, -2), { timeoutMs: TOPE_ACCION_MS })
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
  if (lanzado) teclasConfirmadas.set(a.id, a.tecla)
  if (lanzado) await notaParaLaIA($, a)
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
      ...(a.tipo === 'pedido' && a.estado === 'espera' ? [Text({ color: 'yellow', wrap: 'wrap', children: ['◷ Corre solo cuando ' + esperaDe(a) + ' termine: tu intención queda tomada.'] })] : []),
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

function esperaDe(a) {
  return (a.espera || []).map((t) => '[' + t + ']').join(' ') || 'otro pedido'
}

// desarmar es «✕ Dejar de esperar»: el pedido armado vuelve a pendiente (qdd consola desarmar).
async function desarmar($, a) {
  if (ocupado) return
  ocupado = true
  const bin = await binario($)
  try {
    const r = await $.process.run([bin, 'consola', 'desarmar', a.id.slice('pedido:'.length)], { timeoutMs: TOPE_MS })
    resultado = r.exitCode === 0 ? null : { titulo: 'Dejar de esperar ' + a.etiqueta, ok: false, texto: ((r.stdout || '') + (r.stderr || '')).trim() || 'sin salida', zona: 'pedidos' }
  } catch (err) {
    resultado = { titulo: 'Dejar de esperar ' + a.etiqueta, ok: false, texto: String(err?.message ?? err), zona: 'pedidos' }
  }
  locales.delete(a.id)
  ocupado = false
  firma = ''
  await leer($)
  $.ui.invalidate('ui.render')
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
  if (s.estado === 'espera' && !a.tecla) {
    return caja(gris('◌'), [comando, ...meta('espera a ' + esperaDe(a))])
  }
  // Ya se puede confirmar: queda armado y corre solo cuando termine lo que espera.
  if (s.estado === 'espera') {
    return caja(Button({ key: 'b-' + a.id, label: a.tecla + '  ' + a.etiqueta, hotkey: a.tecla, onPress: () => pedir($, a) }), [comando, ...meta('espera a ' + esperaDe(a) + ' · confírmalo y corre solo')])
  }
  // Armado (pedido del usuario, 2026-11-01): ya tiene la intención de la persona. Lo importante primero y en color; lo
  // que dijo el comando, atenuado debajo (lectura en F).
  if (s.estado === 'armado') {
    const cuando = a.espera?.length ? 'cuando ' + esperaDe(a) + ' termine' : 'cuando esté listo'
    const intento = a.intentos ? ' · intento ' + a.intentos : ''
    const proximo = !a.espera?.length && hora(a.proximo) ? ' · reintenta ' + hora(a.proximo) : ''
    const dijo = a.aviso ? [Text({ key: 'aviso', dimColor: true, size: 'caption', wrap: 'wrap', children: ['   ' + a.aviso.trim().split('\n').slice(-1)[0]] })] : []
    return caja(Text({ key: 'que', color: 'yellow', weight: 'bold', wrap: 'truncate-end', children: ['◷  ' + a.etiqueta] }), [
      Text({ key: 'armado', color: 'yellow', wrap: 'wrap', children: ['   Armado · corre solo ' + cuando + intento + proximo] }),
      ...dijo,
      comando,
      ...meta(),
      Box({ key: 'desarmar', paddingLeft: 3, children: [Button({ key: 'desarmar-' + a.id, label: '✕ Dejar de esperar', plain: true, onPress: () => desarmar($, a) })] }),
    ])
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
  // «todavía no» (FND-082) sin armar (dejó de esperar, o un qdd anterior): primero y en color qué pasa; atenuado, lo que
  // dijo. Se puede volver a confirmar.
  const aviso = a.aviso
    ? [
        Text({ key: 'aviso-t', color: 'yellow', weight: 'bold', size: 'caption', children: ['   ◷ Todavía no' + (hora(a.aviso_en) ? ' · ' + hora(a.aviso_en) : '')] }),
        Text({ key: 'aviso', dimColor: true, size: 'caption', wrap: 'wrap', children: ['   ' + a.aviso.trim().split('\n').slice(-1)[0]] }),
      ]
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
  const seVa = seVan($, Text, new Set(lista.map((a) => a.id)))
  for (const a of lista) tarjetasVistas.set(a.id, lineasDeTarjeta(a))
  if (lista.length === 0 && seVa.length === 0) return []
  const hechos = lista.filter((a) => estadoDe(a).estado === 'hecho')
  const abiertos = lista.filter((a) => estadoDe(a).estado !== 'hecho')
  const porHacer = lista.filter(ejecutable).length
  const enEspera = lista.filter((a) => estadoDe(a).estado === 'espera').length
  const corriendo = lista.filter((a) => estadoDe(a).estado === 'ejecutando').length
  const armados = lista.filter((a) => estadoDe(a).estado === 'armado').length
  const nada = corriendo ? [] : ['nada por hacer']
  const resumen = [...(porHacer ? [porHacer + ' por hacer'] : nada), ...(corriendo ? [corriendo + ' ejecutando'] : []), ...(armados ? [armados + (armados === 1 ? ' armado' : ' armados')] : []), ...(enEspera ? [enEspera + ' en espera'] : []), ...(hechos.length ? [hechos.length + (hechos.length === 1 ? ' hecho' : ' hechos')] : [])].join(' · ')
  const dibujo = (a) => materializa($, Text, 'p:' + a.id, lineasDeTarjeta(a), tarjeta(Box, Text, Button, $, a), estadoDe(a).estado + '|' + a.tecla + '|' + a.etiqueta)
  const cuerpo = [...[...abiertos, ...hechos].map((a) => (confirmando?.id === a.id ? confirmacion(Box, Text, Button, $) : dibujo(a))), ...seVa]
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
  return Text({ key: 'pie', dimColor: true, italic: true, size: 'caption', wrap: 'wrap', children: ['nombre → ¿cómo vamos? · ⏳/❌ clic → la IA lo atiende · l ↻ · botones: clic o tecla · foco: /tareas' + (version ? ' · ' + version : '')] })
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

function dibujar($, Box, Text, Button, Input, ancho) {
  if (!datos) return sinEstado(Box, Text, falla)
  const e = datos.estado
  const lista = datos.acciones || []
  if (config) return Box({ flexDirection: 'column', children: [cabecera(Box, Text, Button, $, e, lista), ...vistaConfig($, Box, Text, Button, Input, ancho)] })
  if (bps) return Box({ flexDirection: 'column', children: [cabecera(Box, Text, Button, $, e, lista), ...vistaBlueprints($, Box, Text, Button, ancho)] })
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

// ── Atajo desde el chat (pedido del usuario, 2026-11-01) ─────────────────────────────────────────────────────────
// Sin mouse y en cualquier terminal: con el chat vacío, la tecla de lo que te espera (un pedido a, b, d…, una ficha
// 1…9) y Enter abren su confirmación en el panel con el foco en «s  Sí»; otro Enter aprueba y el teclado vuelve solo al
// chat. Solo se toma un mensaje que es exactamente esa tecla («a.» va a la IA). Para un solo paso, el terminal o el
// sistema puede mapear AltGr+letra (o lo que se quiera) a un carácter privado, U+E000 + la letra: el panel lo consume
// sin escribirlo. Probado en Claude Code 2.1.300: Alt+letra no llega a los mods y el texto descartado vuelve al chat
// después del hook, por eso el panel espera un instante antes de vaciarlo y tomar el foco.
const PRIVADO = 0xe000
const ESPERA_ATAJO_MS = 40

function atajoDe(texto) {
  if (!datos || !/^[a-z0-9]$/.test(texto)) return null
  const lista = datos.acciones || []
  const pedido = pedidos(lista).find((a) => a.tecla === texto && ejecutable(a))
  if (pedido) return pedido
  const fichas = lista.filter((a) => a.tipo === 'aprobar')
  const asignadas = teclas(fichas)
  return fichas.find((a) => asignadas[a.id] === texto) || null
}

// privadoDe: la letra de un carácter privado (U+E061 → «a»), o '' si no es uno.
function privadoDe(texto) {
  const cp = [...String(texto || '')].length === 1 ? texto.codePointAt(0) : 0
  if (cp < PRIVADO + 0x30 || cp > PRIVADO + 0x7a) return ''
  return String.fromCharCode(cp - PRIVADO)
}

function teclaDe(a) {
  if (a.tecla) return a.tecla
  return teclas((datos?.acciones || []).filter((x) => x.tipo === 'aprobar'))[a.id] || ''
}

// atajo deja la confirmación de a en el panel con el foco en «s  Sí». vaciar: tras un envío descartado, el chat
// todavía tiene la tecla y el motor no le da el foco al panel sobre un chat con texto.
async function atajo($, a, vaciar) {
  if (vaciar) await $.clock.sleep(ESPERA_ATAJO_MS)
  bps = null
  config = null
  pedir($, a)
  if (vaciar) await $.prompt.fill({ text: '', mode: 'replace' }).catch((err) => $.ui.log('no se vació el chat: ' + String(err?.message ?? err)))
  await $.ui.open({ id: PANE, title: TITULO, focus: true })
  if (confirmando?.id !== a.id) return
  const r = await $.ui.focus({ requestId: PANE, key: 'si' }).catch((err) => ({ deny: String(err?.message ?? err) }))
  if (r?.deny) $.ui.log('el panel no tomó el foco: ' + r.deny)
}

// ── 📦 Blueprints (pedido del usuario, 2026-11-01) ───────────────────────────────────────────────────────────────
// Los instalados y los disponibles, con sus cuentas, los da 'qdd consola blueprints --json' sin red ni IA. ↻ trae el
// catálogo; ⬆ Actualizar y ＋ Instalar son comandos de qdd que escriben el lockfile (con «s  Sí»); ✦ Aplicar es lo único
// que le escribe a la IA, con el mensaje que da qdd. Después de cada acción se relee y se dice cuántos cambiaron.
let bps = null // null | { inv, acciones, confirmando, resultado, falla }
const SIMBOLO_BP = { 'con-cambios': ['✦', 'magenta'], 'solo-lock': ['⬆', 'yellow'], 'sin-medir': ['?', undefined], 'al-dia': ['✓', 'green'], disponible: ['＋', undefined] }
const ESTADO_BP = { 'con-cambios': 'cambia la receta', 'solo-lock': 'sin la IA', 'sin-medir': 'sin medir', 'al-dia': 'al día', disponible: 'disponible' }

async function abrirBlueprints($) {
  bps = bps ? null : {}
  config = null
  if (bps) await leerBlueprints($)
  $.ui.invalidate('ui.render')
}

async function leerBlueprints($) {
  const bin = await binario($)
  try {
    const r = await $.process.run([bin, 'consola', 'blueprints', '--json'], { timeoutMs: TOPE_MS })
    const j = r.exitCode === 0 ? JSON.parse(r.stdout) : null
    if (!j) throw new Error(((r.stdout || '') + (r.stderr || '')).trim() || 'sin salida')
    if (bps) bps = { ...bps, inv: j.inventario || { entradas: [] }, acciones: j.acciones || [], falla: '' }
  } catch (err) {
    if (bps) bps = { ...bps, falla: String(err?.message ?? err) }
  }
}

function pedirBP($, a) {
  if (ocupado || !bps) return
  if (a.tipo === 'atender') return aplicarBP($, a)
  if (!a.confirmar) return correrBP($, a)
  bps = { ...bps, confirmando: a }
  $.ui.invalidate('ui.render')
}

// correrBP corre la acción con qdd, relee y compara: cuántas versiones nuevas trajo el catálogo, o cuántos blueprints
// se actualizaron o instalaron.
async function correrBP($, a) {
  const antes = bps.inv
  ocupado = true
  bps = { ...bps, confirmando: null, resultado: { ok: true, titulo: a.etiqueta, texto: 'ejecutando ' + a.detalle + '…' } }
  $.ui.invalidate('ui.render')
  const bin = await binario($)
  let r = null
  try {
    r = await $.process.run([bin, 'consola', 'accion', a.id, ...(a.confirmar ? ['--confirmado'] : [])], { timeoutMs: TOPE_ACCION_MS })
  } catch (err) {
    r = { exitCode: 1, stdout: '', stderr: String(err?.message ?? err) }
  }
  await leerBlueprints($)
  const salida = ((r.stdout || '') + (r.stderr || '')).trim().split('\n').slice(-LINEAS_RESULTADO).join('\n')
  const ok = r.exitCode === 0
  const resumen = ok ? cambiosBP(antes, bps?.inv, a.id === PREFIJO_BP + 'sync') : ''
  if (bps) bps = { ...bps, resultado: { ok, titulo: a.etiqueta, resumen, texto: salida || 'sin salida' } }
  ocupado = false
  $.ui.invalidate('ui.render')
}

async function aplicarBP($, a) {
  ocupado = true
  bps = { ...bps, confirmando: null }
  const bin = await binario($)
  let texto = ''
  let resultado = { ok: false, titulo: a.etiqueta, texto: 'sin salida' }
  try {
    const r = await $.process.run([bin, 'consola', 'accion', a.id], { timeoutMs: TOPE_MS })
    texto = r.exitCode === 0 ? (r.stdout || '').trim() : ''
    resultado = { ok: false, titulo: a.etiqueta, texto: ((r.stdout || '') + (r.stderr || '')).trim() || 'sin salida' }
  } catch (err) {
    resultado = { ok: false, titulo: a.etiqueta, texto: String(err?.message ?? err) }
  }
  if (texto) {
    resultado = { ok: true, titulo: a.etiqueta, texto: '→ enviado a la IA' }
    $.prompt.submit({ text: texto }).catch((err) => $.ui.log('no se envió a la IA: ' + String(err?.message ?? err)))
  }
  if (bps) bps = { ...bps, resultado }
  ocupado = false
  $.ui.invalidate('ui.render')
}

const PREFIJO_BP = 'blueprint:'

// cambiosBP: tras ↻, cuántas versiones nuevas hay y qué se puede tomar; tras actualizar o instalar, cuántos cambiaron.
function cambiosBP(antes, despues, esSync) {
  if (!despues) return ''
  const previas = new Map((antes?.entradas || []).map((e) => [e.id, e]))
  if (esSync) {
    const nuevas = despues.entradas.filter((e) => e.publicada && previas.get(e.id)?.publicada !== e.publicada).length
    return [plural(nuevas, 'versión nueva', 'versiones nuevas'), despues.sin_ia + (despues.sin_ia === 1 ? ' se actualiza' : ' se actualizan') + ' sin la IA', plural(despues.con_ia, 'cambia la receta', 'cambian la receta')].join(' · ')
  }
  const actualizados = despues.entradas.filter((e) => e.instalada && previas.get(e.id)?.instalada && previas.get(e.id).instalada !== e.instalada).length
  const instalados = despues.entradas.filter((e) => e.instalada && !previas.get(e.id)?.instalada).length
  return [plural(actualizados, 'actualizado', 'actualizados'), plural(instalados, 'instalado', 'instalados')].join(' · ')
}

function plural(n, uno, varios) {
  return n + ' ' + (n === 1 ? uno : varios)
}

function vistaBlueprints($, Box, Text, Button, ancho) {
  const titulo = separador(Box, Text, 't-blueprints', '📦 Blueprints', undefined, 'p cierra', ancho)
  if (!bps.inv && !bps.falla) return [titulo, Text({ key: 'bp-leyendo', dimColor: true, children: ['leyendo…'] })]
  if (!bps.inv) return [titulo, Text({ key: 'bp-falla', color: 'red', wrap: 'wrap', children: ['✗ ' + bps.falla] })]
  const inv = bps.inv
  const acciones = bps.acciones || []
  const deEntrada = (id) => acciones.filter((a) => a.id.endsWith(':' + id) && a.id !== PREFIJO_BP + 'sync')
  const cuentas = [plural(inv.instalados, 'instalado', 'instalados'), inv.sin_ia + ' sin IA', inv.con_ia + ' con IA', plural(inv.disponibles, 'disponible', 'disponibles')].join(' · ')
  const generales = acciones.filter((a) => a.id === PREFIJO_BP + 'sync' || a.id === PREFIJO_BP + 'actualizar-sin-ia')
  const teclaGeneral = { [PREFIJO_BP + 'sync']: 'r', [PREFIJO_BP + 'actualizar-sin-ia']: 'u' }
  const botones = generales.map((a) => (bps.confirmando?.id === a.id ? confirmarBP($, Box, Text, Button) : Button({ key: 'bp-' + a.id, label: teclaGeneral[a.id] + '  ' + a.etiqueta, hotkey: teclaGeneral[a.id], plain: true, onPress: () => pedirBP($, a) })))
  const problemas = (inv.problemas || []).map((p, i) => Text({ key: 'bp-problema-' + i, color: 'yellow', wrap: 'wrap', children: ['⚠ ' + p] }))
  const grupos = [
    ['Para actualizar', inv.entradas.filter((e) => e.estado === 'con-cambios' || e.estado === 'solo-lock')],
    ['Instalados', inv.entradas.filter((e) => e.estado === 'al-dia' || e.estado === 'sin-medir')],
    ['Disponibles', inv.entradas.filter((e) => e.estado === 'disponible')],
  ].filter(([, es]) => es.length > 0)
  const cuerpo = grupos.flatMap(([nombre, es]) => [
    Text({ key: 'bp-g-' + nombre, dimColor: true, weight: 'semibold', size: 'caption', children: [nombre] }),
    ...es.map((e) => filaBP($, Box, Text, Button, e, deEntrada(e.id))),
  ])
  return [
    titulo,
    Text({ key: 'bp-cuentas', wrap: 'wrap', children: [cuentas] }),
    ...problemas,
    Box({ key: 'bp-generales', flexDirection: 'column', marginTop: 1, children: [Box({ flexDirection: 'row', flexWrap: 'wrap', columnGap: 2, children: botones })] }),
    ...resultadoBP(Box, Text),
    Box({ key: 'bp-lista', flexDirection: 'column', marginTop: 1, rowGap: 1, children: cuerpo }),
  ]
}

// filaBP: el símbolo y el nombre, las versiones y el estado; debajo, por qué cambió la receta y sus botones.
function filaBP($, Box, Text, Button, e, acciones) {
  const [simbolo, color] = SIMBOLO_BP[e.estado] || ['·', undefined]
  const versiones = e.instalada && e.publicada && e.instalada !== e.publicada ? e.instalada + ' → ' + e.publicada : e.instalada || e.publicada || ''
  const confirma = acciones.find((a) => bps.confirmando?.id === a.id)
  const botones = confirma
    ? [confirmarBP($, Box, Text, Button)]
    : [Box({ flexDirection: 'row', flexWrap: 'wrap', columnGap: 2, children: acciones.map((a) => Button({ key: 'bp-' + a.id, label: a.etiqueta.replace(/ «[^»]*»/, ''), plain: true, onPress: () => pedirBP($, a) })) })]
  return Box({
    key: 'bp-' + e.id,
    flexDirection: 'column',
    children: [
      Box({
        flexDirection: 'row',
        columnGap: 1,
        children: [
          Text({ color, dimColor: !color, children: [simbolo] }),
          Text({ weight: 'bold', wrap: 'truncate-end', children: [e.nombre || e.id] }),
          Text({ dimColor: true, wrap: 'truncate-end', children: [versiones + ' · ' + (ESTADO_BP[e.estado] || e.estado)] }),
        ],
      }),
      ...(e.razones?.length ? [Text({ dimColor: true, size: 'caption', wrap: 'wrap', children: ['  ' + e.razones.join(' · ')] })] : []),
      ...(acciones.length ? [Box({ paddingLeft: 2, children: botones })] : []),
    ],
  })
}

function confirmarBP($, Box, Text, Button) {
  const a = bps.confirmando
  return Box({
    key: 'bp-confirmar',
    flexDirection: 'column',
    borderStyle: 'single',
    borderColor: 'yellow',
    paddingX: 1,
    children: [
      Text({ color: 'yellow', dimColor: true, size: 'caption', children: ['Escribe en el proyecto'] }),
      Text({ color: 'yellow', weight: 'bold', wrap: 'wrap', children: ['¿' + a.etiqueta + '?'] }),
      Text({ mono: true, wrap: 'wrap', children: ['Ejecutará: ' + a.detalle] }),
      Box({
        flexDirection: 'row',
        columnGap: 2,
        marginTop: 1,
        children: [
          Button({ key: 'bp-si', label: 's  Sí', hotkey: 's', variant: 'primary', onPress: () => correrBP($, a) }),
          Button({
            key: 'bp-no',
            label: 'n  No',
            hotkey: 'n',
            onPress: () => {
              bps = { ...bps, confirmando: null }
              $.ui.invalidate('ui.render')
            },
          }),
        ],
      }),
    ],
  })
}

function resultadoBP(Box, Text) {
  const r = bps.resultado
  if (!r) return []
  const tinte = r.ok ? 'green' : 'red'
  return [
    Box({
      key: 'bp-resultado',
      flexDirection: 'column',
      borderStyle: 'single',
      borderColor: tinte,
      borderDimColor: true,
      paddingX: 1,
      marginTop: 1,
      children: [
        Text({ color: tinte, weight: 'semibold', wrap: 'wrap', children: [(r.ok ? '✓ ' : '✗ ') + r.titulo] }),
        ...(r.resumen ? [Text({ key: 'bp-resumen', wrap: 'wrap', children: [r.resumen] })] : []),
        Text({ dimColor: true, size: 'caption', wrap: 'wrap', children: [r.texto] }),
      ],
    }),
  ]
}

// ── Herramientas del proyecto (RFC 0011) ──────────────────────────────────────────────────────────────────────────
// Cada herramienta guardada (.qdd/herramientas/<id>.yaml) es su propio panel, 'qdd-herr-<id>', con su nombre como
// pestaña. Sus acciones son botones; cada una abre sus campos y ▶ Correr, que pide 'qdd herramienta correr' (sin shell:
// cada campo llega como un argumento). La salida va debajo: completa al terminar (una-vez) o mientras corre, hasta
// ■ Detener (en-vivo). Usarla no le pide nada a la IA. ⚙ Config, en el panel de tareas, las lista para abrirlas o
// quitarlas y guía para agregar una: la IA arma su definición una vez, con el mensaje que da qdd, y QDD la valida, la
// prueba y la guarda. La lógica vive en la CLI; el panel solo dibuja y pide.
const PANE_HERRAMIENTA = 'qdd-herr-'
const TOPE_HERRAMIENTA_MS = 600000
const LINEAS_SALIDA = 2000
const CADA_LECTURAS = 5 // las herramientas se releen cada 5 lecturas del panel (10 s)
let herramientas = []
let firmaHerramientas = ''
let lecturas = 0
let config = null // null | { vista: 'lista' | 'agregar' | 'ia', candidatos, elegido, quitando, guardada, aviso }
const sesionesHerr = {} // id → { accion, valores, confirmando, aviso, corrida, salida, fin }

function sesionHerr(id) {
  if (!sesionesHerr[id]) sesionesHerr[id] = { accion: '', valores: {}, confirmando: false, aviso: '', corrida: null, salida: '', fin: null }
  return sesionesHerr[id]
}

// leerHerramientas pide el listado a qdd. Una que apareció mientras la IA la configuraba cierra el paso 5 y queda como
// pestaña; una que ya no está cierra su pestaña. Un qdd sin 'herramienta' deja la lista vacía.
async function leerHerramientas($) {
  const bin = await binario($)
  let lista = herramientas
  try {
    const r = await $.process.run([bin, 'herramienta', 'listar', '--json'], { timeoutMs: TOPE_MS })
    lista = r.exitCode === 0 ? JSON.parse(r.stdout).herramientas || [] : []
  } catch {
    lista = []
  }
  const huella = JSON.stringify(lista)
  if (huella === firmaHerramientas) return
  firmaHerramientas = huella
  const antes = herramientas
  herramientas = lista
  const nuevas = lista.filter((h) => !antes.some((x) => x.id === h.id))
  if (config?.vista === 'ia' && nuevas.length > 0) {
    const guardada = nuevas.find((h) => h.comando.join(' ') === config.elegido) || nuevas[0]
    config = { vista: 'lista', guardada: guardada.nombre }
    await abrirHerramienta($, guardada)
  }
  for (const h of antes) {
    if (lista.some((x) => x.id === h.id)) continue
    delete sesionesHerr[h.id]
    await $.ui.close({ id: PANE_HERRAMIENTA + h.id }).catch(() => {})
  }
  $.ui.invalidate('ui.render')
}

async function abrirHerramienta($, h) {
  await $.ui.open({ id: PANE_HERRAMIENTA + h.id, title: h.nombre })
}

async function abrirHerramientas($) {
  for (const h of herramientas) await abrirHerramienta($, h)
}

function valorDe(s, c) {
  const v = s.valores[c.id]
  return v === undefined || v === '' ? c.por_defecto || '' : v
}

function elegirAccion($, id, accion) {
  const s = sesionHerr(id)
  if (s.corrida) return
  s.accion = accion
  s.valores = {}
  s.confirmando = false
  s.aviso = ''
  $.ui.invalidate('ui.render')
}

// correrHerramienta: lo requerido primero; lo que pide confirmar, después del «s  Sí». Solo van los campos con valor:
// qdd pone los por_defecto y vuelve a validar todo.
async function correrHerramienta($, id, confirmado) {
  const h = herramientas.find((x) => x.id === id)
  const s = sesionHerr(id)
  const a = h?.acciones.find((x) => x.id === s.accion)
  if (!a || s.corrida || h.aviso) return
  const falta = (a.campos || []).find((c) => c.requerido && !valorDe(s, c))
  s.aviso = falta ? 'falta «' + (falta.nombre || falta.id) + '»' : ''
  s.confirmando = !falta && a.confirmar && !confirmado
  $.ui.invalidate('ui.render')
  if (falta || s.confirmando) return
  const valores = (a.campos || []).filter((c) => (s.valores[c.id] ?? '') !== '').map((c) => c.id + '=' + s.valores[c.id])
  const argv = [await binario($), 'herramienta', 'correr', h.id, a.id, ...valores, ...(a.confirmar ? ['--si'] : [])]
  const corrida = { nombre: a.nombre, enVivo: a.modo === 'en-vivo', detenida: false, stream: null }
  s.corrida = corrida
  s.salida = ''
  s.fin = null
  $.ui.invalidate('ui.render')
  if (corrida.enVivo) {
    void enVivo($, s, corrida, argv)
    return
  }
  let fin = { ok: false }
  try {
    const r = await $.process.run(argv, { timeoutMs: TOPE_HERRAMIENTA_MS })
    s.salida = recortar((r.stdout || '') + (r.stderr || ''))
    fin = { ok: r.exitCode === 0, codigo: r.exitCode }
  } catch (err) {
    s.salida = String(err?.message ?? err)
  }
  terminar($, s, corrida, fin)
}

// enVivo lee la salida mientras corre; ■ Detener corta el proceso (qdd detiene la herramienta y lo que lanzó).
async function enVivo($, s, corrida, argv) {
  let fin = { ok: false }
  try {
    const stream = $.process.spawn({ argv })
    corrida.stream = stream
    for await (const { text } of stream) {
      if (corrida.detenida) break
      s.salida = recortar(s.salida + text)
      $.ui.invalidate('ui.render')
    }
    if (!corrida.detenida) {
      const r = await stream.result
      fin = { ok: r.code === 0, codigo: r.code }
    }
  } catch (err) {
    if (!corrida.detenida) s.salida = recortar(s.salida + String(err?.message ?? err))
  }
  terminar($, s, corrida, fin)
}

function detener($, s) {
  const corrida = s.corrida
  if (!corrida?.enVivo) return
  corrida.detenida = true
  corrida.stream?.return(undefined).catch(() => {})
  terminar($, s, corrida, { detenida: true })
}

// terminar cierra la corrida, salvo que ya la haya cerrado ■ o que otra la haya reemplazado.
function terminar($, s, corrida, fin) {
  if (s.corrida !== corrida) return
  s.corrida = null
  s.fin = { ...fin, nombre: corrida.nombre }
  $.ui.invalidate('ui.render')
}

// recortar deja las últimas LINEAS_SALIDA líneas: la salida no se guarda en disco.
function recortar(texto) {
  const lineas = texto.split('\n')
  return lineas.length > LINEAS_SALIDA ? lineas.slice(-LINEAS_SALIDA).join('\n') : texto
}

function campoDe($, Input, Select, s, c) {
  const etiqueta = (c.nombre || c.id) + (c.requerido ? ' *' : '')
  const guardar = (v) => {
    s.valores[c.id] = v
    $.ui.invalidate('ui.render')
  }
  if (c.tipo === 'opcion' || c.tipo === 'si-no') {
    const opciones = c.tipo === 'si-no' ? [{ value: 'no', label: 'no' }, { value: 'si', label: 'sí' }] : c.opciones.map((o) => ({ value: o, label: o }))
    const vacia = c.tipo === 'opcion' && !c.por_defecto ? [{ value: '', label: '—' }] : []
    return Select({ key: 'h-campo-' + c.id, label: etiqueta, options: [...vacia, ...opciones], value: valorDe(s, c) || (c.tipo === 'si-no' ? 'no' : ''), onSelect: guardar })
  }
  return Input({ key: 'h-campo-' + c.id, label: etiqueta, placeholder: c.tipo === 'numero' ? 'un número' : c.por_defecto || '', value: s.valores[c.id] ?? '', submitLabel: 'listo', onInput: guardar, onSubmit: guardar })
}

// formulario: los campos de la acción elegida, ▶ Correr o su «¿Seguro?», y lo que falta.
function formulario($, Box, Text, Button, Input, Select, h, s, a) {
  const campos = (a.campos || []).map((c) => campoDe($, Input, Select, s, c))
  const aviso = s.aviso ? [Text({ key: 'h-aviso', color: 'yellow', wrap: 'wrap', children: [s.aviso] })] : []
  const no = () => {
    s.confirmando = false
    $.ui.invalidate('ui.render')
  }
  const confirmar = Box({
    key: 'h-confirmar',
    flexDirection: 'column',
    borderStyle: 'single',
    borderColor: 'yellow',
    paddingX: 1,
    children: [
      Text({ color: 'yellow', weight: 'bold', wrap: 'wrap', children: ['¿Seguro? ' + a.nombre] }),
      Box({
        flexDirection: 'row',
        columnGap: 2,
        children: [
          Button({ key: 'h-si', label: 's  Sí', hotkey: 's', variant: 'primary', onPress: () => correrHerramienta($, h.id, true) }),
          Button({ key: 'h-no', label: 'n  No', hotkey: 'n', onPress: no }),
        ],
      }),
    ],
  })
  const boton = h.aviso || s.corrida ? [] : [Button({ key: 'h-correr', label: 'r  ▶ Correr', hotkey: 'r', variant: 'primary', onPress: () => correrHerramienta($, h.id, false) })]
  return [Box({ key: 'h-form', flexDirection: 'column', rowGap: 1, marginTop: 1, children: [...campos, ...aviso, ...(s.confirmando ? [confirmar] : boton)] })]
}

// salidaDe: la corrida en curso o la última, con su salida debajo.
function salidaDe($, Box, Text, Button, s) {
  if (!s.corrida && !s.fin) return []
  const cabeza = s.corrida
    ? Text({ key: 'h-estado', dimColor: true, children: ['⟳ ' + s.corrida.nombre + ' · corriendo…'] })
    : s.fin.detenida
      ? Text({ key: 'h-estado', dimColor: true, children: ['■ ' + s.fin.nombre + ' · detenida'] })
      : Text({ key: 'h-estado', color: s.fin.ok ? 'green' : 'red', weight: 'semibold', children: [(s.fin.ok ? '✓ ' : '✗ ') + s.fin.nombre + (s.fin.ok || s.fin.codigo === undefined ? '' : ' · salió con ' + s.fin.codigo)] })
  const detenerla = s.corrida?.enVivo ? [Button({ key: 'h-detener', label: 'd  ■ Detener', hotkey: 'd', onPress: () => detener($, s) })] : []
  const texto = s.salida.replace(/\n+$/, '')
  return [
    Box({ key: 'h-cabeza-salida', flexDirection: 'row', justifyContent: 'space-between', columnGap: 2, marginTop: 1, children: [cabeza, ...detenerla] }),
    ...(texto ? [Box({ key: 'h-salida', borderStyle: 'single', borderDimColor: true, paddingX: 1, children: [Text({ key: 'h-salida-texto', mono: true, wrap: 'wrap', children: [texto] })] })] : []),
  ]
}

function dibujarHerramienta($, Box, Text, Button, Input, Select, id) {
  const h = herramientas.find((x) => x.id === id)
  if (!h) return Text({ dimColor: true, wrap: 'wrap', children: ['Esta herramienta ya no está en el proyecto. ⚙ Config, en el panel de tareas, lista las que hay.'] })
  const s = sesionHerr(id)
  const a = h.acciones.find((x) => x.id === s.accion)
  const botones = h.acciones.map((x, i) =>
    Button({ key: 'h-accion-' + x.id, label: (i < 9 ? i + 1 + '  ' : '') + x.nombre, ...(i < 9 ? { hotkey: String(i + 1) } : {}), variant: x.id === s.accion ? 'primary' : undefined, onPress: () => elegirAccion($, id, x.id) }),
  )
  return Box({
    flexDirection: 'column',
    children: [
      Text({ key: 'h-nombre', color: 'cyan', weight: 'bold', size: 'title', children: [h.nombre] }),
      ...(h.descripcion ? [Text({ key: 'h-desc', dimColor: true, wrap: 'wrap', children: [h.descripcion] })] : []),
      ...(h.aviso ? [Text({ key: 'h-falta', color: 'yellow', wrap: 'wrap', children: ['⚠ ' + h.aviso] })] : []),
      Box({ key: 'h-acciones', flexDirection: 'row', flexWrap: 'wrap', columnGap: 2, marginTop: 1, children: botones }),
      ...(a ? formulario($, Box, Text, Button, Input, Select, h, s, a) : [Text({ key: 'h-elige', dimColor: true, size: 'caption', children: ['Elige una acción.'] })]),
      ...salidaDe($, Box, Text, Button, s),
      Box({ marginTop: 1, children: [Text({ key: 'h-pie', dimColor: true, italic: true, size: 'caption', wrap: 'wrap', children: ['corre en el proyecto, sin IA · ' + h.comando.join(' ')] })] }),
    ],
  })
}

// ⚙ Config (en el panel de tareas): las herramientas del proyecto y el paso a paso para agregar una.
async function abrirConfig($) {
  config = config ? null : { vista: 'lista' }
  bps = null
  if (config) await leerHerramientas($)
  $.ui.invalidate('ui.render')
}

async function agregar($) {
  const bin = await binario($)
  let candidatos = []
  try {
    const r = await $.process.run([bin, 'herramienta', 'candidatos', '--json'], { timeoutMs: TOPE_MS })
    candidatos = r.exitCode === 0 ? JSON.parse(r.stdout) || [] : []
  } catch {
    candidatos = []
  }
  config = { vista: 'agregar', candidatos: candidatos.filter((c) => !c.agregada) }
  $.ui.invalidate('ui.render')
}

// partir lee lo que la persona escribió como argumentos: separados por espacios, con comillas para juntar.
function partir(texto) {
  return (String(texto).match(/"[^"]*"|'[^']*'|\S+/g) || []).map((p) => p.replace(/^(["'])(.*)\1$/, '$2'))
}

// elegir pide a qdd el mensaje para la IA (paso 2) y se lo envía: la IA lee su ayuda, propone sus acciones a la
// persona y la guarda con qdd. El panel sigue el listado hasta que aparece (paso 5).
async function elegir($, argv) {
  if (argv.length === 0) return
  const bin = await binario($)
  let texto = ''
  let aviso = ''
  try {
    const r = await $.process.run([bin, 'herramienta', 'configurar', ...argv], { timeoutMs: TOPE_MS })
    texto = r.exitCode === 0 ? (r.stdout || '').trim() : ''
    aviso = r.exitCode === 0 ? '' : (r.stderr || '').trim() || 'esta versión de qdd no trae «qdd herramienta»: actualiza qdd'
  } catch (err) {
    aviso = String(err?.message ?? err)
  }
  config = { vista: texto ? 'ia' : 'agregar', candidatos: config?.candidatos || [], elegido: argv.join(' '), aviso }
  if (texto) $.prompt.submit({ text: texto }).catch((err) => $.ui.log('no se envió a la IA: ' + String(err?.message ?? err)))
  $.ui.invalidate('ui.render')
}

async function quitar($, h) {
  const bin = await binario($)
  let aviso = ''
  try {
    const r = await $.process.run([bin, 'herramienta', 'quitar', h.id, '--si'], { timeoutMs: TOPE_MS })
    aviso = r.exitCode === 0 ? '' : ((r.stdout || '') + (r.stderr || '')).trim()
  } catch (err) {
    aviso = String(err?.message ?? err)
  }
  config = { vista: 'lista', aviso }
  await leerHerramientas($)
  $.ui.invalidate('ui.render')
}

function filaHerramienta($, Box, Text, Button, h) {
  return Box({
    key: 'h-fila-' + h.id,
    flexDirection: 'row',
    justifyContent: 'space-between',
    columnGap: 2,
    children: [
      Box({
        flexDirection: 'column',
        flexShrink: 1,
        children: [
          Text({ key: 'h-n-' + h.id, color: h.aviso ? 'yellow' : undefined, weight: 'semibold', wrap: 'truncate-end', children: [h.nombre] }),
          Text({ key: 'h-d-' + h.id, dimColor: true, size: 'caption', wrap: 'truncate-end', children: [h.aviso ? '⚠ ' + h.aviso : h.acciones.length + (h.acciones.length === 1 ? ' acción' : ' acciones') + ' · ' + h.comando.join(' ')] }),
        ],
      }),
      Box({
        flexDirection: 'row',
        flexShrink: 0,
        columnGap: 2,
        children: [
          Button({ key: 'h-abrir-' + h.id, label: 'Abrir', plain: true, onPress: () => $.ui.open({ id: PANE_HERRAMIENTA + h.id, title: h.nombre, focus: true }) }),
          Button({
            key: 'h-quitar-' + h.id,
            label: 'Quitar',
            plain: true,
            onPress: () => {
              config = { vista: 'lista', quitando: h.id }
              $.ui.invalidate('ui.render')
            },
          }),
        ],
      }),
    ],
  })
}

function confirmarQuitar($, Box, Text, Button, h) {
  return Box({
    key: 'h-confirmar-quitar',
    flexDirection: 'column',
    borderStyle: 'single',
    borderColor: 'yellow',
    paddingX: 1,
    children: [
      Text({ color: 'yellow', weight: 'bold', wrap: 'wrap', children: ['¿Quitar «' + h.nombre + '»?'] }),
      Text({ dimColor: true, wrap: 'wrap', children: ['Borra ' + h.archivo + ' y cierra su pestaña.'] }),
      Box({
        flexDirection: 'row',
        columnGap: 2,
        marginTop: 1,
        children: [
          Button({ key: 'h-si', label: 's  Sí', hotkey: 's', variant: 'primary', onPress: () => quitar($, h) }),
          Button({
            key: 'h-no',
            label: 'n  No',
            hotkey: 'n',
            onPress: () => {
              config = { vista: 'lista' }
              $.ui.invalidate('ui.render')
            },
          }),
        ],
      }),
    ],
  })
}

const PASOS = ['La IA lee su ayuda y propone sus acciones', 'Las aceptas o pides cambios, en la conversación', 'QDD la valida y corre su prueba de humo', 'Queda guardada como pestaña']

function vistaConfig($, Box, Text, Button, Input, ancho) {
  const cancelar = Button({
    key: 'h-cancelar',
    label: 'Cancelar',
    plain: true,
    onPress: () => {
      config = { vista: 'lista' }
      $.ui.invalidate('ui.render')
    },
  })
  const aviso = config.aviso ? [Text({ key: 'h-config-aviso', color: 'red', wrap: 'wrap', children: [config.aviso] })] : []
  let cuerpo = []
  if (config.vista === 'lista') {
    const filas = herramientas.map((h) => (config.quitando === h.id ? confirmarQuitar($, Box, Text, Button, h) : filaHerramienta($, Box, Text, Button, h)))
    cuerpo = [
      ...(config.guardada ? [Text({ key: 'h-guardada', color: 'green', wrap: 'wrap', children: ['✓ ' + config.guardada + ' quedó guardada: ya es una pestaña'] })] : []),
      ...(filas.length ? filas : [Text({ key: 'h-vacio', dimColor: true, wrap: 'wrap', children: ['No hay herramientas. Agrega un ejecutable del proyecto (un simulador, un visor de logs, un migrador) y queda como pestaña.'] })]),
      Box({ key: 'h-agregar-fila', children: [Button({ key: 'h-agregar', label: '+ Agregar herramienta', variant: 'primary', onPress: () => agregar($) })] }),
    ]
  }
  if (config.vista === 'agregar') {
    const cands = (config.candidatos || []).map((c, i) => Button({ key: 'h-cand-' + i, label: c.comando.join(' ') + ' · ' + c.origen, plain: true, onPress: () => elegir($, c.comando) }))
    cuerpo = [
      Text({ key: 'h-paso-1', weight: 'semibold', wrap: 'wrap', children: ['1 · Elige un ejecutable del proyecto o escribe su ruta'] }),
      ...(cands.length ? [Box({ key: 'h-cands', flexDirection: 'column', children: cands })] : [Text({ key: 'h-sin-cands', dimColor: true, wrap: 'wrap', children: ['No encontré ejecutables en bin/, scripts/, cmd/*, target/release, package.json ni el Makefile.'] })]),
      Input({ key: 'h-ruta', label: 'Ruta', placeholder: './bin/mi-herramienta --flag', submitLabel: 'agregar', onSubmit: (v) => elegir($, partir(v)) }),
      cancelar,
    ]
  }
  if (config.vista === 'ia') {
    cuerpo = [
      Text({ key: 'h-paso-1', color: 'green', wrap: 'truncate-end', children: ['✓ 1 · ' + config.elegido] }),
      ...PASOS.map((p, i) => Text({ key: 'h-paso-' + (i + 2), dimColor: i > 0, color: i === 0 ? 'yellow' : undefined, wrap: 'wrap', children: [(i === 0 ? '⟳ ' : '· ') + (i + 2) + ' · ' + p] })),
      cancelar,
    ]
  }
  return [
    separador(Box, Text, 't-config', '⚙ Herramientas del proyecto', undefined, 'k cierra', ancho),
    Box({ key: 'h-config', flexDirection: 'column', rowGap: 1, children: [...aviso, ...cuerpo] }),
  ]
}

// COMANDO_QDD: qdd en posición de comando (al inicio o tras ; & | ( ` o un salto de línea), con asignaciones VAR=…
// delante, con o sin ruta (./qdd, $S/qdd, ~/.local/bin/qdd) y con o sin sufijo (qdd-rc10, qdd.exe). Una carpeta o un
// archivo llamado qdd en los argumentos no cuenta (FND-106).
const COMANDO_QDD = /(?:^|[;&|(\n`])\s*(?:\w+=\S*\s+)*(?:\S*\/)?qdd(?:[-.][\w.-]*)?(?=\s|$)/

// cambiaElPanel: una tool MCP de qdd o un comando de shell que corre qdd.
function cambiaElPanel(e) {
  if (/__qdd_/.test(String(e.tool || ''))) return true
  return e.tool === 'Bash' && COMANDO_QDD.test(String(e.command || ''))
}

// cicloConContexto marca el ciclo y devuelve lo que el panel espera de la persona ('' si nada cambió). Un qdd anterior
// a --contexto igual marca el ciclo.
async function cicloConContexto($, bin, modo) {
  try {
    const r = await $.process.run([bin, 'consola', 'ciclo', '--modo', modo, '--contexto'], { timeoutMs: TOPE_CICLO_MS })
    if (r.exitCode === 0) return (r.stdout || '').trim()
    if (/unknown flag/.test(r.stderr || '')) await $.process.run([bin, 'consola', 'ciclo', '--modo', modo], { timeoutMs: TOPE_MS })
  } catch (err) {
    $.ui.log('no se marcó el ciclo: ' + String(err?.message ?? err))
  }
  return ''
}

function hayQueMostrar() {
  if (!datos) return false
  const frentes = datos.estado ? datos.estado.frentes.length : 0
  const fichas = (datos.acciones || []).filter((a) => a.tipo === 'aprobar' || (a.tipo === 'pedido' && ejecutable(a))).length
  return frentes + fichas > 0
}

// ── Preguntas desde el teléfono (RFC 0013 §«Preguntas desde el teléfono») ─────────────────────────────────────────
// Lo que Claude Code le pregunta a la persona en esta sesión (AskUserQuestion o un permiso de herramienta) también se
// responde desde QDD Móvil. El panel lo registra con «qdd consola pregunta registrar» (sin QDD Móvil activado, qdd no
// registra nada) y espera la respuesta de un teléfono con «qdd consola pregunta esperar» mientras el diálogo de la
// terminal sigue abierto: Claude Code corre el hook PermissionRequest al lado del diálogo y gana quien responde
// primero. Si gana el teléfono, el hook decide: AskUserQuestion con sus answers; un permiso, permitir una vez o negar
// (nunca «siempre» ni reglas). Si gana la terminal, la llamada de la herramienta termina y el panel la cierra «en el
// PC». Solo en la conversación principal de una sesión interactiva: en un subagente o sin terminal Claude Code puede
// esperar al hook antes de mostrar el diálogo, y la terminal quedaría sin preguntar.
const ESPERA_PREGUNTA_S = 10
const PREGUNTAR = 'AskUserQuestion'
const RESERVADAS = new Set(['tool', 'tool_use_id', 'agentId', 'requestMeta', 'consent'])
let interactiva = false
let huboPreguntas = false // esta carga registró alguna (al cerrar la sesión, qdd vence las que queden)
const llamadas = new Map() // tool_use_id → { tool, entrada }: las herramientas en curso de la conversación principal
const preguntasVivas = new Map() // id → { llamada, candidatas, cerrada, aplicada }

// estable: el JSON de un valor con las claves ordenadas, para reconocer la misma entrada.
function estable(v) {
  if (Array.isArray(v)) return '[' + v.map(estable).join(',') + ']'
  if (v && typeof v === 'object') return '{' + Object.keys(v).sort().map((k) => JSON.stringify(k) + ':' + estable(v[k])).join(',') + '}'
  return JSON.stringify(v)
}

function entradaDe(e) {
  const out = {}
  for (const [k, v] of Object.entries(e)) if (!RESERVADAS.has(k)) out[k] = v
  return out
}

// llamadaDe: la herramienta en curso que pide este permiso (la de la misma entrada, sin otra pregunta viva); si no, las
// de esa herramienta quedan como candidatas.
function llamadaDe(tool, entrada) {
  const usadas = new Set([...preguntasVivas.values()].map((v) => v.llamada))
  const libres = [...llamadas].filter(([id, l]) => l.tool === tool && !usadas.has(id))
  const clave = estable(entrada)
  const misma = libres.find(([, l]) => estable(l.entrada) === clave)
  if (misma) return { llamada: misma[0], candidatas: new Set() }
  return { llamada: '', candidatas: new Set(libres.map(([id]) => id)) }
}

async function registrarPregunta($, bin, mia, e) {
  try {
    const r = await $.process.run([bin, 'consola', 'pregunta', 'registrar'], { stdin: JSON.stringify({ sesion: mia, herramienta: e.tool_name, entrada: e.tool_input }), timeoutMs: TOPE_MS })
    if (r.exitCode !== 0) return ''
    return String(JSON.parse(r.stdout).id || '')
  } catch {
    return ''
  }
}

// esperarTelefono pide a qdd, de a ESPERA_PREGUNTA_S, la respuesta de un teléfono; null si la terminal ganó, venció o
// Claude Code dejó de esperar al hook.
async function esperarTelefono($, bin, id, viva, signal) {
  while (!viva.cerrada && !signal?.aborted) {
    let r
    try {
      r = await $.process.run([bin, 'consola', 'pregunta', 'esperar', id, '--segundos', String(ESPERA_PREGUNTA_S)], { timeoutMs: (ESPERA_PREGUNTA_S + 5) * 1000 })
    } catch (err) {
      $.ui.log('qdd: dejó de esperar la respuesta del teléfono: ' + String(err?.message ?? err), { to: 'debug' })
      return null
    }
    if (viva.cerrada || r.exitCode !== 0) return null
    let p = null
    try {
      p = JSON.parse(r.stdout)
    } catch {
      return null
    }
    if (p.estado === 'pendiente') continue
    if (p.estado !== 'respondida' || p.respondida_en !== 'telefono') return null
    return p
  }
  return null
}

// decisionDe: lo que el teléfono respondió, como lo decide un hook PermissionRequest. Un permiso solo se permite una
// vez (sin updatedPermissions) o se niega.
function decisionDe(e, p) {
  const donde = p.lugar || 'el teléfono'
  if (e.tool_name === PREGUNTAR && p.answers) return { behavior: 'allow', updatedInput: { ...e.tool_input, answers: p.answers } }
  if (e.tool_name !== PREGUNTAR && p.decision === 'permitir') return { behavior: 'allow' }
  return { behavior: 'deny', message: 'La persona lo negó desde ' + donde + ' (QDD Móvil).' }
}

async function preguntarAlTelefono($, e, next) {
  const r = await next(e)
  if (r?.decision || !interactiva || e.agent_id) return r
  const bin = await binario($)
  const id = await registrarPregunta($, bin, await sesionPropia($), e)
  if (!id) return r
  huboPreguntas = true
  const viva = { ...llamadaDe(e.tool_name, e.tool_input), cerrada: false, aplicada: false }
  preguntasVivas.set(id, viva)
  if (viva.llamada) {
    try {
      $.ui.notice(viva.llamada, '📱 También se puede responder desde QDD Móvil')
    } catch {}
  }
  const p = await esperarTelefono($, bin, id, viva, next.signal)
  if (!p || viva.cerrada) return r
  viva.aplicada = true
  preguntasVivas.delete(id)
  return { ...(r || {}), decision: decisionDe(e, p) }
}

// cerrarPreguntasDe: terminó la herramienta que preguntaba sin que el teléfono decidiera: la respondió la terminal.
async function cerrarPreguntasDe($, llamada) {
  const vivas = [...preguntasVivas].filter(([, v]) => v.llamada === llamada || v.candidatas.has(llamada))
  if (vivas.length === 0) return
  const bin = await binario($)
  for (const [id, v] of vivas) {
    preguntasVivas.delete(id)
    v.cerrada = true
    await $.process.run([bin, 'consola', 'pregunta', 'cerrar', id], { timeoutMs: TOPE_MS }).catch((err) => $.ui.log('qdd: no se cerró la pregunta: ' + String(err?.message ?? err), { to: 'debug' }))
  }
}

// vencerPreguntas: la sesión se cierra; lo que esperaba al teléfono ya no se puede responder.
async function vencerPreguntas($) {
  for (const v of preguntasVivas.values()) v.cerrada = true
  preguntasVivas.clear()
  if (!huboPreguntas) return
  const mia = await sesionPropia($)
  if (!mia) return
  const bin = await binario($)
  await $.process.run([bin, 'consola', 'pregunta', 'vencer', '--sesion', mia], { timeoutMs: 2000 }).catch(() => {})
}

// ── Instrucciones desde el teléfono (RFC 0013 §«Ejecutar desde el teléfono», solo Pro) ──────────────────────────────
// Con Pro, la persona puede mandar desde QDD Móvil una instrucción para la IA del proyecto. Cada TELEFONO_MS, la
// sesión interactiva late con «qdd consola telefono latir» (así el teléfono sabe que hay una sesión abierta) y recibe
// lo que le toca, una sola vez aunque haya varias sesiones abiertas: qdd lo decide con la cola bloqueada. Lo envía a
// la IA como un mensaje de la persona que dice que viene de su teléfono; el mensaje empieza con 📱, así que nunca se
// lee como un comando de shell («!») ni de Claude Code («/»), y se notifica en el escritorio. Sin QDD Móvil activado,
// late cada TELEFONO_PAUSA latidos; con un qdd anterior a «consola telefono», deja de latir.
const TELEFONO_MS = 5000
const TELEFONO_PAUSA = 12
const telefono = { soportado: true, latio: false, saltar: 0, latiendo: false }

async function latirTelefono($) {
  if (!interactiva || !telefono.soportado || telefono.latiendo) return
  if (telefono.saltar > 0) {
    telefono.saltar -= 1
    return
  }
  const mia = await sesionPropia($)
  if (!mia) return
  telefono.latiendo = true
  try {
    await recibirDelTelefono($, mia)
  } finally {
    telefono.latiendo = false
  }
}

async function recibirDelTelefono($, mia) {
  const bin = await binario($)
  let r
  try {
    r = await $.process.run([bin, 'consola', 'telefono', 'latir', '--sesion', mia], { timeoutMs: TOPE_MS })
  } catch (err) {
    $.ui.log('qdd: no latió para el teléfono: ' + String(err?.message ?? err), { to: 'debug' })
    return
  }
  if (r.exitCode !== 0 && /unknown command|unknown flag/.test(r.stderr || '')) telefono.soportado = false
  if (r.exitCode !== 0) return
  let l = null
  try {
    l = JSON.parse(r.stdout)
  } catch {
    return
  }
  if (!l.activo) {
    telefono.saltar = TELEFONO_PAUSA - 1
    return
  }
  telefono.latio = true
  for (const x of l.entregar || []) await entregarDelTelefono($, bin, mia, x)
}

// entregarDelTelefono le da la instrucción a la IA; si Claude Code no la acepta, qdd la marca sin entregar y el
// teléfono lo ve.
async function entregarDelTelefono($, bin, mia, x) {
  if (!x?.mensaje || !String(x.mensaje).startsWith('📱')) return
  try {
    await $.prompt.submit({ text: x.mensaje })
  } catch (err) {
    const motivo = String(err?.message ?? err)
    $.ui.log('no se envió a la IA la instrucción del teléfono: ' + motivo)
    await $.process.run([bin, 'consola', 'telefono', 'fallo', x.solicitud, '--sesion', mia, '--motivo', motivo], { timeoutMs: TOPE_MS }).catch(() => {})
    return
  }
  $.ui.notify('📱 Instrucción desde ' + (x.nombre || 'el teléfono'), { title: 'qdd' }).catch((err) => $.ui.log('no se notificó: ' + String(err?.message ?? err)))
}

// cerrarTelefono: la sesión se cierra; deja de contar como abierta en el acto.
async function cerrarTelefono($) {
  if (!telefono.latio) return
  const mia = await sesionPropia($)
  if (!mia) return
  const bin = await binario($)
  await $.process.run([bin, 'consola', 'telefono', 'latir', '--sesion', mia, '--fin'], { timeoutMs: 2000 }).catch(() => {})
}

// ── Cola de mejoras (pkg/mejoras) ──────────────────────────────────────────────────────────────────────────────────
// Mientras hay una sesión interactiva, cada MEJORAS_MS pide «qdd mejoras ciclo --si-toca»: qdd decide si pasaron sus
// 10 min y, si el ciclo entregó tareas del objetivo mejora-continua, devuelve el mensaje para la IA, que empieza con 🔁
// (nunca se lee como «!» ni «/»). Con un qdd sin «mejoras», deja de pedirlo. El botón «Ciclo de mejoras» del panel lo
// fuerza sin esperar.
const MEJORAS_MS = 60000
const mejoras = { soportado: true, corriendo: false }

async function cicloMejoras($) {
  if (!interactiva || !mejoras.soportado || mejoras.corriendo) return
  mejoras.corriendo = true
  try {
    await pedirCicloMejoras($)
  } finally {
    mejoras.corriendo = false
  }
}

async function pedirCicloMejoras($) {
  const bin = await binario($)
  let r
  try {
    r = await $.process.run([bin, 'mejoras', 'ciclo', '--si-toca', '--json'], { timeoutMs: TOPE_MS })
  } catch (err) {
    $.ui.log('qdd: el ciclo de mejoras no corrió: ' + String(err?.message ?? err), { to: 'debug' })
    return
  }
  if (r.exitCode !== 0 && /unknown command|unknown flag/.test(r.stderr || '')) mejoras.soportado = false
  if (r.exitCode !== 0 || !String(r.stdout || '').trim()) return
  let c = null
  try {
    c = JSON.parse(r.stdout)
  } catch {
    return
  }
  if (!c?.mensaje || !String(c.mensaje).startsWith('🔁')) return
  try {
    await $.prompt.submit({ text: c.mensaje })
  } catch (err) {
    $.ui.log('no se envió a la IA el aviso de la cola de mejoras: ' + String(err?.message ?? err))
    return
  }
  $.ui.notify('🔁 Mejoras nuevas para trabajar: ' + (c.tareas || []).join(', '), { title: 'qdd' }).catch((err) => $.ui.log('no se notificó: ' + String(err?.message ?? err)))
}

export function register(on) {
  on('session.start', async ($, e, next) => {
    interactiva = e.isInteractive === true
    await leer($)
    await leerHerramientas($)
    if (interactiva) {
      await latirTelefono($)
      $.clock.every(TELEFONO_MS, async () => {
        await latirTelefono($)
      })
      $.clock.every(MEJORAS_MS, async () => {
        await cicloMejoras($)
      })
    }
    $.clock.every(INTERVALO_MS, async () => {
      await leer($)
      lecturas += 1
      if (lecturas % CADA_LECTURAS === 0) await leerHerramientas($)
    })
    $.clock.every(LATIDO_MS, async () => {
      latir($)
    })
    // Se abre solo si hay tareas, fichas por aprobar o comandos pedidos; después, cada pedido nuevo lo vuelve a abrir.
    if (hayQueMostrar()) await $.ui.open({ id: PANE, title: TITULO })
    // Cada herramienta guardada es una pestaña, sin quitarle el foco a la conversación (RFC 0011).
    await abrirHerramientas($)
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
    await leerHerramientas($)
    await abrirHerramientas($)
    return {}
  })

  // Pulsos de vida (§3.13): cada herramienta del agente y cada subagente laten la tarea en curso. next(e) va primero
  // y sin esperar al pulso.
  // Una tool de qdd o un comando qdd cambian lo que el panel muestra: se relee en el acto, sin esperar a la próxima
  // lectura (sincronía panel–chat, 2026-11-01).
  // Cada herramienta de la conversación principal queda en curso hasta que termina: si preguntó algo (un permiso) y el
  // teléfono no decidió, la respondió la terminal.
  on('tool.call', async ($, e, next) => {
    pulsar($, 'agente')
    const seguir = !e.agentId && e.tool_use_id
    if (seguir) llamadas.set(e.tool_use_id, { tool: e.tool, entrada: entradaDe(e) })
    try {
      if (!cambiaElPanel(e)) return await next(e)
      const r = await next(e)
      await leer($)
      return r
    } finally {
      if (seguir) {
        llamadas.delete(e.tool_use_id)
        await cerrarPreguntasDe($, e.tool_use_id)
      }
    }
  })

  // Preguntas desde el teléfono: corre al lado del diálogo de la terminal; gana quien responde primero.
  on('classic.PermissionRequest', async ($, e, next) => preguntarAlTelefono($, e, next))

  on('session.end', async ($, e, next) => {
    await vencerPreguntas($)
    await cerrarTelefono($)
    return next(e)
  })

  on('agent.spawn', async ($, e, next) => {
    pulsar($, 'subagente')
    return next(e)
  })

  // Un ciclo por mensaje (§3.9): lo que ya se vio (el resultado y los pedidos hechos) sale del tablero, y qdd anota el
  // modo de permisos de la IA para decirle cuándo pedir autorización. Con --contexto, qdd dice además qué espera el
  // panel de la persona (solo si cambió) y va a la IA con el mensaje: si ella lo resolvió en el chat, la IA lo refleja.
  on('classic.UserPromptSubmit', async ($, e, next) => {
    resultado = null
    for (const [id, l] of locales) if (l.estado === 'hecho') locales.delete(id)
    $.ui.invalidate('ui.render')
    const bin = await binario($)
    const contexto = await cicloConContexto($, bin, String(e.permission_mode || ''))
    const r = await next(e)
    if (!contexto) return r
    return { ...(r || {}), additionalContext: [...((r && r.additionalContext) || []), contexto] }
  })

  // Atajo desde el chat: la tecla de lo que te espera y Enter lo abren en el panel, sin enviarle nada a la IA.
  on('prompt.submit', async ($, e, next) => {
    if (e.command || e.isMeta) return next(e)
    const a = atajoDe(String(e.text || '').trim())
    if (!a) return next(e)
    atajo($, a, true).catch((err) => $.ui.log('atajo: ' + String(err?.message ?? err)))
    return { drop: '▶ ' + teclaDe(a) + ' · ' + a.etiqueta + ': Enter confirma en el panel, n cancela' }
  })

  // Mientras se escribe la tecla, el chat dice qué abre Enter; un carácter privado lo abre en el acto sin escribirse.
  on('prompt.edit', async ($, e, next) => {
    const privado = privadoDe(e.inputText)
    if (privado) {
      const a = atajoDe(privado)
      if (a) atajo($, a, false).catch((err) => $.ui.log('atajo: ' + String(err?.message ?? err)))
      return { text: e.text, cursor: e.cursor }
    }
    const r = await next(e)
    const a = atajoDe(String(r.text || '').trim())
    if (!a) return r
    const fin = r.text.length
    return { ...r, decorations: [...(r.decorations ?? []), { start: fin, end: fin, virtualText: '  ↵ ' + a.etiqueta, dimColor: true }] }
  })

  // El reproductor terminó: se vuelve a dibujar lo de siempre (o se quita lo que se iba).
  on('ui.message', async ($, e, next) => {
    const fin = e.data && typeof e.data === 'object' ? e.data.fin : undefined
    if (typeof fin !== 'string') return next(e)
    animaciones.delete(fin)
    fantasmas.delete(fin)
    $.ui.invalidate('ui.render')
    return {}
  })

  on('ui.render', { component: 'AbovePrompt' }, async ($, e, next) => banda($, e, next))

  on('ui.render', { component: 'Pane' }, async ($, e, next) => {
    const { Box, Text, Button, Input, Select, Client: cliente } = $.ui.resolve(e)
    Client = cliente || null
    if (String(e.requestId).startsWith(PANE_HERRAMIENTA)) return dibujarHerramienta($, Box, Text, Button, Input, Select, e.requestId.slice(PANE_HERRAMIENTA.length))
    if (e.requestId !== PANE) return next(e)
    return dibujar($, Box, Text, Button, Input, e.props.bodyColumns || 40)
  })
}
