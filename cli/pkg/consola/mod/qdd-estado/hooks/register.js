// Mod de Claude Code para el estado por frente (QDD, RFC 0006 §3.4).
// Dibuja un panel con la tabla de frentes × etapas, lo que espera cada ⏳ y de quién, y las fallas. La lógica vive en
// la CLI: el mod lee 'qdd estado ver --json' cada INTERVALO_MS y solo dibuja; no interpreta reglas ni lee el YAML.
// Sin qdd, con un qdd antiguo o sin .qdd/project/estado.yaml, el panel dice cómo crearlo y el resto de la consola
// sigue igual. QDD_BIN apunta a otro binario de qdd (p. ej. uno de prueba).

const PANE = 'qdd-estado'
const INTERVALO_MS = 2000
const TOPE_MS = 3000

let datos = null
let falla = ''
let firma = ''

// leer pide el estado a la CLI y redibuja solo si cambió.
async function leer($) {
  const propio = await $.env.get('QDD_BIN')
  const bin = propio || 'qdd'
  let nuevo = null
  let motivo = ''
  try {
    const r = await $.process.run([bin, 'estado', 'ver', '--json'], { timeoutMs: TOPE_MS })
    motivo = r.exitCode === 0 ? '' : (r.stderr || '').trim()
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

function sinEstado(Box, Text) {
  const detalle = falla || 'No hay estado por frente en este proyecto.'
  return Box({
    flexDirection: 'column',
    children: [
      Text({ dimColor: true, wrap: 'wrap', children: [detalle] }),
      Text({ dimColor: true, wrap: 'wrap', children: ["Se crea con 'qdd estado init' (blueprint estado-por-frente, destino «archivo»)."] }),
    ],
  })
}

function lista(Box, Text, titulo, color, items, conQuien) {
  if (items.length === 0) return []
  const filas = [Text({ children: [' '] }), Text({ color, weight: 'semibold', children: [titulo] })]
  items.forEach((p, i) => {
    filas.push(Text({ key: titulo + '-f' + i, wrap: 'wrap', children: ['• ' + p.frente + ' › ' + p.etapa] }))
    const detalle = conQuien ? '  de ' + (p.de_quien || '¿?') + ': ' + p.texto : '  ' + p.texto
    filas.push(Text({ key: titulo + '-d' + i, dimColor: true, wrap: 'wrap', children: [detalle] }))
  })
  return filas
}

function dibujar(Box, Text, ancho) {
  if (!datos) return sinEstado(Box, Text)
  const columnas = 3 * datos.etapas.length
  const largo = Math.max('Frente'.length, ...datos.frentes.map((f) => f.nombre.length))
  const nombreAncho = Math.max(8, Math.min(largo, ancho - columnas - 1))
  const titulo = datos.resumen || '◧ sin frentes abiertos'
  const cuando = hora(datos.actualizado)

  const fila = (key, izquierda, derecha, dim) =>
    Box({
      key,
      flexDirection: 'row',
      columnGap: 1,
      children: [
        Box({ width: nombreAncho, children: [Text({ dimColor: dim, wrap: 'truncate-end', children: [izquierda] })] }),
        Text({ dimColor: dim, children: [derecha] }),
      ],
    })

  const leyenda = datos.etapas.map((e, i) => datos.abreviaturas[i] + '=' + e).join(' ')
  return Box({
    flexDirection: 'column',
    children: [
      Text({ weight: 'semibold', children: [titulo + (cuando ? ' · ' + cuando : '')] }),
      Text({ children: [' '] }),
      fila('cabecera', 'Frente', datos.abreviaturas.join(' '), true),
      ...datos.frentes.map((f) => fila('frente-' + f.id, f.nombre, celdas(f.simbolos), false)),
      Text({ dimColor: true, wrap: 'wrap', children: [leyenda] }),
      ...lista(Box, Text, '⏳ Esperando', 'yellow', datos.esperas, true),
      ...lista(Box, Text, '❌ Fallas', 'red', datos.fallas, false),
    ],
  })
}

export function register(on) {
  on('session.start', async ($, e, next) => {
    await leer($)
    $.clock.every(INTERVALO_MS, async () => {
      await leer($)
    })
    // Se abre solo si hay frentes; en una terminal angosta espera a que el usuario lo pida con /estado.
    if (datos && datos.frentes.length > 0) await $.ui.open({ id: PANE, title: 'Estado por frente' })
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
    await $.ui.open({ id: PANE, title: 'Estado por frente' })
    return {}
  })

  on('ui.render', { component: 'Pane' }, async ($, e, next) => {
    if (e.requestId !== PANE) return next(e)
    const { Box, Text } = $.ui.resolve(e)
    return dibujar(Box, Text, e.props.bodyColumns || 40)
  })
}
