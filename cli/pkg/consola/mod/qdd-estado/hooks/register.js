// Mod de Claude Code para el estado por frente (QDD, RFC 0006 §3.4.1 y §3.7; blueprint consola-claude-code v3.0.0).
// Dibuja el panel en cuatro niveles (resumen en números grandes, títulos, texto y detalle atenuado) y los botones de
// las acciones. La lógica vive en la CLI: el mod lee 'qdd consola panel --json' cada INTERVALO_MS y solo dibuja; una
// acción se pide con 'qdd consola accion <id>', que qdd vuelve a comprobar. Lo que escribe pasa antes por «s  Sí».
// Sin qdd, con un qdd antiguo o sin .qdd/project/estado.yaml, el panel dice cómo crearlo y el resto de la consola
// sigue igual. QDD_BIN apunta a otro binario de qdd (p. ej. uno de prueba).

const PANE = 'qdd-estado'
const TITULO = 'Estado por frente'
const INTERVALO_MS = 2000
const TOPE_MS = 3000
const TOPE_ACCION_MS = 60000
const TOPE_AUDITAR_MS = 600000
const LINEAS_RESULTADO = 4

let datos = null // { estado, motivo, acciones }
let falla = ''
let firma = ''
let confirmando = null // la acción que espera el segundo paso
let resultado = null // { titulo, ok, texto }
let ocupado = false

// Números de 3 filas con caracteres de bloque: la terminal tiene un solo tamaño de letra.
const DIGITOS = {
  0: ['█▀█', '█ █', '▀▀▀'],
  1: ['▀█ ', ' █ ', '▀▀▀'],
  2: ['▀▀█', '█▀▀', '▀▀▀'],
  3: ['▀▀█', ' ▀█', '▀▀▀'],
  4: ['█ █', '▀▀█', '  ▀'],
  5: ['█▀▀', '▀▀█', '▀▀▀'],
  6: ['█▀▀', '█▀█', '▀▀▀'],
  7: ['▀▀█', '  █', '  ▀'],
  8: ['█▀█', '█▀█', '▀▀▀'],
  9: ['█▀█', '▀▀█', '▀▀▀'],
}

function grande(n) {
  const cifras = String(n).split('')
  return [0, 1, 2].map((fila) => cifras.map((c) => DIGITOS[c][fila]).join(' '))
}

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
  $.ui.invalidate('ui.render')
}

function hora(iso) {
  const t = new Date(iso)
  if (Number.isNaN(t.getTime())) return ''
  const dos = (n) => String(n).padStart(2, '0')
  return dos(t.getHours()) + ':' + dos(t.getMinutes())
}

// celdas alinea los símbolos con las abreviaturas: ✅ ⏳ ❌ ocupan 2 columnas y — una.
function celdas(simbolos) {
  return simbolos.map((s) => (s === '—' || s === '?' ? s + ' ' : s)).join(' ')
}

// colorFrente: el frente toma el color de su peor etapa.
function colorFrente(f) {
  if (f.simbolos.includes('❌')) return 'red'
  if (f.simbolos.includes('⏳')) return 'yellow'
  return 'green'
}

function titulo(Text, key, texto, color) {
  return Text({ key, color, weight: 'bold', size: 'title', children: [texto] })
}

function contador(Box, Text, key, n, etiqueta, color) {
  return Box({
    key,
    flexDirection: 'column',
    alignItems: 'center',
    children: [
      ...grande(n).map((l, i) => Text({ key: key + '-' + i, color, weight: 'bold', children: [l] })),
      Text({ dimColor: true, size: 'caption', children: [etiqueta] }),
    ],
  })
}

function cabecera(Box, Text, e) {
  const cuando = hora(e.actualizado)
  return Box({
    key: 'cabecera',
    flexDirection: 'column',
    borderStyle: 'round',
    borderColor: 'cyan',
    paddingX: 1,
    children: [
      Box({
        flexDirection: 'row',
        justifyContent: 'space-between',
        children: [
          Text({ color: 'cyan', weight: 'bold', size: 'title', children: ['◧ ' + TITULO] }),
          Text({ dimColor: true, size: 'caption', children: [cuando ? 'al día ' + cuando : ''] }),
        ],
      }),
      Box({
        flexDirection: 'row',
        justifyContent: 'space-around',
        marginTop: 1,
        children: [
          contador(Box, Text, 'c-frentes', e.frentes.length, 'frentes', 'cyan'),
          contador(Box, Text, 'c-esperas', e.esperas.length, '⏳ esperan', 'yellow'),
          contador(Box, Text, 'c-fallas', e.fallas.length, '❌ fallan', 'red'),
        ],
      }),
    ],
  })
}

function tabla(Box, Text, e, ancho) {
  const columnas = 3 * e.etapas.length
  const largo = Math.max('Frente'.length, ...e.frentes.map((f) => f.nombre.length))
  const nombreAncho = Math.max(8, Math.min(largo, ancho - columnas - 3))
  const fila = (key, nombre, derecha, estilo) =>
    Box({
      key,
      flexDirection: 'row',
      columnGap: 1,
      children: [
        Box({ width: nombreAncho, children: [Text({ ...estilo, wrap: 'truncate-end', children: [nombre] })] }),
        Text({ ...estilo, children: [derecha] }),
      ],
    })
  const leyenda = e.etapas.map((etapa, i) => e.abreviaturas[i] + ' ' + etapa).join(' · ')
  return Box({
    key: 'tabla',
    flexDirection: 'column',
    marginTop: 1,
    children: [
      titulo(Text, 't-frentes', 'Frentes', undefined),
      fila('cab', 'Frente', e.abreviaturas.join(' '), { dimColor: true, size: 'caption' }),
      ...e.frentes.map((f) => fila('f-' + f.id, f.nombre, celdas(f.simbolos), { color: colorFrente(f) })),
      Text({ dimColor: true, italic: true, size: 'caption', wrap: 'wrap', children: [leyenda] }),
    ],
  })
}

function seccion(Box, Text, key, nombre, color, items, conQuien) {
  if (items.length === 0) return []
  return [
    Box({
      key,
      flexDirection: 'column',
      marginTop: 1,
      rowGap: 1,
      children: [
        titulo(Text, key + '-t', nombre + ' (' + items.length + ')', color),
        ...items.map((p, i) =>
          Box({
            key: key + '-' + i,
            flexDirection: 'column',
            paddingLeft: 1,
            children: [
              Text({ weight: 'semibold', wrap: 'wrap', children: [p.frente] }),
              Text({ dimColor: true, size: 'caption', wrap: 'wrap', children: [p.etapa + (conQuien ? ' · de ' + (p.de_quien || '¿?') : '')] }),
              Text({ wrap: 'wrap', children: ['→ ' + p.texto] }),
            ],
          }),
        ),
      ],
    }),
  ]
}

// tecla: 1…9 para las fichas en el orden en que qdd las entrega, c para revisar el estado y u para auditar.
function teclas(acciones) {
  const asignadas = {}
  let n = 0
  for (const a of acciones) {
    if (a.tipo === 'aprobar') {
      n += 1
      asignadas[a.id] = String(n)
      continue
    }
    if (a.id === 'revisar-estado') asignadas[a.id] = 'c'
    if (a.id === 'auditar') asignadas[a.id] = 'u'
  }
  return asignadas
}

async function ejecutar($, a) {
  ocupado = true
  confirmando = null
  resultado = { titulo: a.etiqueta, ok: true, texto: 'ejecutando ' + a.detalle + '…' }
  $.ui.invalidate('ui.render')
  const bin = await binario($)
  const argv = [bin, 'consola', 'accion', a.id, ...(a.confirmar ? ['--confirmado'] : [])]
  try {
    const r = await $.process.run(argv, { timeoutMs: a.id === 'auditar' ? TOPE_AUDITAR_MS : TOPE_ACCION_MS })
    const texto = ((r.stdout || '') + (r.stderr || '')).trim().split('\n').slice(-LINEAS_RESULTADO).join('\n')
    resultado = { titulo: a.etiqueta, ok: r.exitCode === 0, texto: texto || 'sin salida' }
  } catch (err) {
    resultado = { titulo: a.etiqueta, ok: false, texto: String(err?.message ?? err) }
  }
  ocupado = false
  await leer($)
  $.ui.invalidate('ui.render')
}

function pedir($, a) {
  if (ocupado) return
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
      Text({ color: 'yellow', weight: 'bold', wrap: 'wrap', children: ['¿' + a.etiqueta + '?'] }),
      Text({ dimColor: true, size: 'caption', wrap: 'wrap', children: ['Ejecutará: ' + a.detalle] }),
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
        ],
      }),
    ],
  })
}

function acciones(Box, Text, Button, $, lista) {
  if (lista.length === 0) return []
  const asignadas = teclas(lista)
  const filas = lista.map((a) =>
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
  const salida = resultado
    ? [
        Box({
          key: 'resultado',
          flexDirection: 'column',
          borderStyle: 'single',
          borderColor: resultado.ok ? 'green' : 'red',
          borderDimColor: true,
          paddingX: 1,
          children: [
            Text({ color: resultado.ok ? 'green' : 'red', weight: 'semibold', wrap: 'wrap', children: [(resultado.ok ? '✓ ' : '✗ ') + resultado.titulo] }),
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
        ...(confirmando ? [confirmacion(Box, Text, Button, $)] : filas),
        ...salida,
        Text({ dimColor: true, italic: true, size: 'caption', wrap: 'wrap', children: ['Clic en el panel (o /estado) para darle el foco; luego la tecla de cada botón.'] }),
      ],
    }),
  ]
}

function sinEstado(Box, Text, motivo) {
  return Box({
    key: 'sin-estado',
    flexDirection: 'column',
    children: [
      Text({ dimColor: true, wrap: 'wrap', children: [motivo || 'No hay estado por frente en este proyecto.'] }),
      Text({ dimColor: true, wrap: 'wrap', children: ["Se crea con 'qdd estado init' (blueprint estado-por-frente, destino «archivo»)."] }),
    ],
  })
}

function dibujar($, Box, Text, Button, ancho) {
  if (!datos) return sinEstado(Box, Text, falla)
  const e = datos.estado
  const lista = datos.acciones || []
  if (!e) return Box({ flexDirection: 'column', children: [sinEstado(Box, Text, datos.motivo), ...acciones(Box, Text, Button, $, lista)] })
  return Box({
    flexDirection: 'column',
    children: [
      cabecera(Box, Text, e),
      tabla(Box, Text, e, ancho),
      ...seccion(Box, Text, 's-esperas', '⏳ Esperando', 'yellow', e.esperas, true),
      ...seccion(Box, Text, 's-fallas', '❌ Fallas', 'red', e.fallas, false),
      ...acciones(Box, Text, Button, $, lista),
    ],
  })
}

function hayQueMostrar() {
  if (!datos) return false
  const frentes = datos.estado ? datos.estado.frentes.length : 0
  const fichas = (datos.acciones || []).filter((a) => a.tipo === 'aprobar').length
  return frentes + fichas > 0
}

export function register(on) {
  on('session.start', async ($, e, next) => {
    await leer($)
    $.clock.every(INTERVALO_MS, async () => {
      await leer($)
    })
    // Se abre solo si hay frentes o fichas por aprobar; en una terminal angosta espera a que el usuario lo pida con /estado.
    if (hayQueMostrar()) await $.ui.open({ id: PANE, title: TITULO })
    try {
      await $.command.register({ name: 'estado', description: 'Abre el panel de estado por frente (/estado cerrar lo cierra)', immediate: true })
    } catch (err) {
      $.ui.log('no se registró /estado: ' + String(err?.message ?? err))
    }
    return next(e)
  })

  on('command.run', { command: 'estado' }, async ($, e) => {
    if ((e.args || '').trim() === 'cerrar') {
      await $.ui.close({ id: PANE })
      return {}
    }
    await leer($)
    await $.ui.open({ id: PANE, title: TITULO, focus: true })
    return {}
  })

  on('ui.render', { component: 'Pane' }, async ($, e, next) => {
    if (e.requestId !== PANE) return next(e)
    const { Box, Text, Button } = $.ui.resolve(e)
    return dibujar($, Box, Text, Button, e.props.bodyColumns || 40)
  })
}
