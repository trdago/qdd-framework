import { expect, mock, test } from 'claude-code/testing'

// Pedido de la persona (2026-11-08): «deberían quedar en secciones distintas; un botón en la sección Para ti que puedo
// expandir si quiero y si no solo veo el número con un símbolo de pregunta, pero lo que estamos trabajando siempre
// primero». Los pedidos y las fichas (acciones) van a la vista en «▶ Para ti»; las preguntas, plegadas en «? N».

const PANEL = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Tareas', isFocused: true, bodyColumns: 60, placement: 'dock', scroll: { offset: 0, bodyRows: 40 } },
} as const

const PEDIDO = { id: 'pedido:1', tipo: 'pedido', etiqueta: 'Mergear', detalle: 'gh pr merge', tecla: 'a', estado: 'pendiente', confirmar: true, turno: 12, siguiente: true }
const FICHA = { id: 'aprobar:x', tipo: 'aprobar', etiqueta: 'Aprobar «x»', detalle: 'qdd modules approve x', confirmar: true, turno: 13 }
const ESPERA = { id: 'pregunta:espera-1', tipo: 'pregunta', origen: 'estado', estado: 'pendiente', etiqueta: '⏳', detalle: '¿Sigo?', confirmar: true, turno: 14 }
const DIALOGO = { id: 'pregunta:abc', tipo: 'pregunta', origen: 'ia', estado: 'pendiente', etiqueta: '¿Cuál prefieres?', detalle: '¿Cuál prefieres?', confirmar: true, turno: 15 }
const EN_LA_TABLA = { pregunta: ESPERA.id, atender: 'atender:barra:1', donde: 'Barra · PR', texto: '¿Sigo?' }

function preparar(on, r) {
  mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('process.run', ($, e) => {
    if (e.argv.includes('binario')) return { value: { exitCode: 1, stdout: '', stderr: '' } }
    if (e.argv.includes('panel')) return { value: { exitCode: 0, stdout: JSON.stringify({ formato: 1, carpeta: '/work', estado: null, motivo: 'x', acciones: r.acciones, esperas_persona: r.esperas }), stderr: '' } }
    return { value: { exitCode: 0, stdout: '{}', stderr: '' } }
  })
}

const esperar = () => new Promise((res) => setTimeout(res, 20))

async function montado($, on, acciones, esperas = [EN_LA_TABLA]) {
  preparar(on, { acciones, esperas })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await esperar()
  return $.ui.mount({ ...PANEL, surface: 'terminal' })
}

async function claves(ui) {
  return (await ui.findAll({ type: 'Button' })).map((b) => b.props.key ?? b.key)
}

test('las acciones se ven; las preguntas quedan plegadas en «? N», después de las acciones', async ($, on) => {
  const ui = await montado($, on, [PEDIDO, FICHA, ESPERA, DIALOGO])
  const boton = await ui.find({ key: 'preguntas-ver' })
  expect(boton.props.label).toContain('? 2')
  expect(await ui.find({ key: 'espera-0' })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: /en la terminal/ })).toBeUndefined()
  const orden = (await claves(ui)).filter((k) => ['b-' + PEDIDO.id, 'b-' + FICHA.id, 'preguntas-ver'].includes(k))
  expect(orden).toEqual(['b-' + PEDIDO.id, 'b-' + FICHA.id, 'preguntas-ver'])
})

test('al tocar «? N» se despliegan, con su número y sus botones; otro toque las pliega', async ($, on) => {
  const ui = await montado($, on, [PEDIDO, ESPERA, DIALOGO])
  await ui.press({ key: 'preguntas-ver' })
  expect(await ui.find({ key: 'espera-0' })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /#15 ¿Cuál prefieres\? · en la terminal/ })).toBeDefined()
  await ui.press({ key: 'preguntas-ver' })
  expect(await ui.find({ key: 'espera-0' })).toBeUndefined()
})

test('solo preguntas: «Para ti» muestra solo el «? N»', async ($, on) => {
  const ui = await montado($, on, [ESPERA])
  expect((await ui.find({ key: 'preguntas-ver' })).props.label).toContain('? 1')
  expect(await ui.find({ key: 'b-' + PEDIDO.id })).toBeUndefined()
})

test('sin preguntas no hay botón «? N»', async ($, on) => {
  const ui = await montado($, on, [PEDIDO], [])
  expect(await ui.find({ key: 'b-' + PEDIDO.id })).toBeDefined()
  expect(await ui.find({ key: 'preguntas-ver' })).toBeUndefined()
})
