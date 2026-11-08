import { expect, mock, test } from 'claude-code/testing'

// RFC 0019: «no puedo estar adivinando cuál debería contestar primero». Para ti es una sola lista en el orden que da qdd,
// con el número de cada cosa, lo que sigue y lo que espera. Desde el 2026-11-08 (pedido de la persona) las acciones van
// primero y las preguntas quedan plegadas en «? N», sin perder su número.

const PANEL = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Tareas', isFocused: true, bodyColumns: 60, placement: 'dock', scroll: { offset: 0, bodyRows: 40 } },
} as const

const PEDIDO = { id: 'pedido:1', tipo: 'pedido', etiqueta: 'Mergear', detalle: 'gh pr merge', tecla: 'a', estado: 'pendiente', confirmar: true, turno: 12, siguiente: true }
const ATADO = { id: 'pedido:2', tipo: 'pedido', etiqueta: 'Publicar', detalle: 'npm publish', tecla: 'b', estado: 'espera', espera: ['a'], confirmar: true, turno: 15, espera_turno: [12] }
const FICHA = { id: 'aprobar:x', tipo: 'aprobar', etiqueta: 'Aprobar «x»', detalle: 'qdd modules approve x', confirmar: true, turno: 13 }
const ESPERA = { id: 'pregunta:espera-1', tipo: 'pregunta', origen: 'estado', estado: 'pendiente', etiqueta: '⏳', detalle: '¿Sigo?', confirmar: true, turno: 14, espera_turno: [12] }
const DIALOGO = { id: 'pregunta:abc', tipo: 'pregunta', origen: 'ia', estado: 'pendiente', etiqueta: '¿Cuál prefieres?', detalle: '¿Cuál prefieres?', confirmar: true, turno: 16 }
const CERRADA = { id: 'pregunta:vieja', tipo: 'pregunta', origen: 'ia', estado: 'respondida', etiqueta: 'Vieja', detalle: 'Vieja', confirmar: true }
const EN_LA_TABLA = { pregunta: ESPERA.id, atender: 'atender:barra:1', donde: 'Barra · PR', texto: '¿Sigo?' }

function preparar(on, r) {
  mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('process.run', ($, e) => {
    r.llamadas.push([...e.argv])
    if (e.argv.includes('binario')) return { value: { exitCode: 1, stdout: '', stderr: '' } }
    if (e.argv.includes('panel')) {
      return { value: { exitCode: 0, stdout: JSON.stringify({ formato: 1, carpeta: '/work', estado: r.estado ?? null, motivo: 'x', acciones: r.acciones, esperas_persona: r.esperas }), stderr: '' } }
    }
    return { value: { exitCode: 0, stdout: '{}', stderr: '' } }
  })
}

const esperar = () => new Promise((res) => setTimeout(res, 20))

async function montado($, on, acciones, esperas = [EN_LA_TABLA], estado = null) {
  const r = { llamadas: [] as string[][], acciones, esperas, estado }
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await esperar()
  const ui = await $.ui.mount({ ...PANEL, surface: 'terminal' })
  if (await ui.find({ key: 'preguntas-ver' })) await ui.press({ key: 'preguntas-ver' })
  return { ui, r }
}

async function textoDe(ui) {
  const nodos = [...(await ui.findAll({ type: 'Text' })).map((n) => n.text), ...(await ui.findAll({ type: 'Button' })).map((b) => b.props.label)]
  return nodos.join('\n')
}

test('las acciones en el orden de qdd y después las preguntas, todo con su número', async ($, on) => {
  const { ui } = await montado($, on, [PEDIDO, FICHA, ESPERA, ATADO, DIALOGO, CERRADA])
  const texto = await textoDe(ui)
  for (const n of ['#12', '#13', '#14', '#15', '#16']) expect(texto).toContain(n)
  expect(texto).not.toContain('Vieja')
  const botones = (await ui.findAll({ type: 'Button' })).map((b) => b.props.key ?? b.key)
  const enLista = ['b-' + PEDIDO.id, 'b-' + FICHA.id, 'b-' + ATADO.id, 'preguntas-ver', 'espera-0']
  expect(botones.filter((k) => enLista.includes(k))).toEqual(enLista)
})

test('lo que sigue se marca, y solo eso', async ($, on) => {
  const { ui } = await montado($, on, [PEDIDO, ATADO])
  const texto = await textoDe(ui)
  expect(texto).toContain('▶ #12')
  expect(texto).not.toContain('▶ #15')
})

test('cada cosa dice qué turno espera', async ($, on) => {
  const { ui } = await montado($, on, [PEDIDO, ESPERA, ATADO])
  const texto = await textoDe(ui)
  expect(texto).toContain('espera a #12')
  expect(texto).toContain('espera #12')
})

test('una ficha con turno se aprueba desde su lugar en la lista y no sale otra vez en Acciones', async ($, on) => {
  const { ui } = await montado($, on, [PEDIDO, FICHA], [])
  const boton = await ui.find({ key: 'b-' + FICHA.id })
  expect(boton).toBeDefined()
  const texto = await textoDe(ui)
  expect(texto).toContain('#13 Aprobar «x»')
  expect(texto.split('Aprobar «x»').length).toBe(2)
})

test('una pregunta abierta en la terminal se ve con su número y dice dónde se responde', async ($, on) => {
  const { ui } = await montado($, on, [DIALOGO], [])
  const texto = await textoDe(ui)
  expect(texto).toContain('#16')
  expect(texto).toContain('en la terminal')
})

test('un panel pide las preguntas para numerar la lista completa', async ($, on) => {
  const { r } = await montado($, on, [])
  const panel = r.llamadas.find((l) => l.includes('panel'))
  expect(panel[panel.indexOf('--con') + 1].split(',')).toContain('pregunta')
})

test('el resumen de arriba cuenta lo mismo que la lista', async ($, on) => {
  const estado = { formato: 1, resumen: '', etapas: ['Código'], abreviaturas: ['Có'], frentes: [], esperas: [], fallas: [] }
  const { ui } = await montado($, on, [PEDIDO, FICHA, ESPERA, DIALOGO], [EN_LA_TABLA], estado)
  expect(await ui.find({ type: 'Text', text: '▶ 4 para ti' })).toBeDefined()
})

test('sin turnos (un qdd anterior) todo sigue como antes', async ($, on) => {
  const viejo = { ...PEDIDO, turno: undefined, siguiente: undefined }
  const { ui } = await montado($, on, [viejo], [{ ...EN_LA_TABLA, pregunta: undefined }])
  const texto = await textoDe(ui)
  expect(texto).toContain('Mergear')
  expect(texto).toContain('¿Sigo?')
  expect(texto).not.toContain('#')
})
