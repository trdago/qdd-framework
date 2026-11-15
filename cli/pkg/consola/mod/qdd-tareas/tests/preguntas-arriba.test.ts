import { expect, mock, test } from 'claude-code/testing'

// MEJ-057 (la persona eligió «Arriba con botones», 2026-11-15): una pregunta que la IA hace en la terminal va primero
// en «▶ Para ti», con su texto y un botón por opción; no queda plegada en «? N», donde estuvo 19 min sin que la persona
// la viera. Los ⏳ siguen plegados. Varias preguntas juntas, o una que qdd no registró, se responden en la terminal.

const PANEL = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Tareas', isFocused: true, bodyColumns: 60, placement: 'dock', scroll: { offset: 0, bodyRows: 60 } },
  surface: 'terminal',
} as const

const PEDIDO = { id: 'pedido:1', tipo: 'pedido', etiqueta: 'Mergear', detalle: 'gh pr merge', tecla: 'a', estado: 'pendiente', confirmar: true, turno: 12 }
const ESPERA = { id: 'pregunta:espera-1', tipo: 'pregunta', origen: 'estado', estado: 'pendiente', etiqueta: '⏳', detalle: '¿Sigo?', confirmar: true, turno: 14 }
const EN_LA_TABLA = { pregunta: ESPERA.id, atender: 'atender:barra:1', donde: 'Barra · PR', texto: '¿Sigo?' }
const ASK = {
  id: 'pregunta:abc', tipo: 'pregunta', origen: 'ia', estado: 'pendiente', etiqueta: '❓ Base · ¿Qué base usamos?', detalle: '¿Qué base usamos?', confirmar: true, turno: 15, siguiente: true,
  preguntas: [{ id: 'q1', encabezado: 'Base', texto: '¿Qué base usamos?', multiple: false, otra: true, opciones: [{ valor: 'a', etiqueta: 'Postgres (Recommended)', descripcion: '' }, { valor: 'b', etiqueta: 'SQLite', descripcion: '' }] }],
}
const PERMISO = {
  id: 'pregunta:pe1', tipo: 'pregunta', origen: 'permiso', estado: 'pendiente', etiqueta: '🔐 Bash: git push', detalle: 'git push', confirmar: true, turno: 16,
  preguntas: [{ id: 'q1', encabezado: 'Permiso', texto: '¿Permites que Claude use Bash?', opciones: [{ valor: 'permitir', etiqueta: 'Permitir una vez' }, { valor: 'negar', etiqueta: 'Negar' }] }],
}
const DOS = { ...ASK, id: 'pregunta:dos', turno: 17, siguiente: false, preguntas: [ASK.preguntas[0], { ...ASK.preguntas[0], id: 'q2', texto: '¿Y la caché?' }] }

type Salida = { exitCode: number; stdout: string; stderr: string }
const ok = (d: unknown): Salida => ({ exitCode: 0, stdout: JSON.stringify(d), stderr: '' })
const pausa = () => new Promise((res) => setTimeout(res, 20))

function preparar(on, r) {
  mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('prompt.submit', ($, e) => {
    r.enviados.push(e.text)
    return { value: undefined }
  })
  on('process.run', ($, e) => {
    const argv = [...e.argv]
    r.llamadas.push(argv)
    if (argv.includes('binario')) return { value: { exitCode: 1, stdout: '', stderr: '' } }
    if (argv[2] === 'panel') return { value: ok({ formato: 1, carpeta: '/work', estado: null, motivo: 'x', acciones: r.acciones, esperas_persona: r.esperas }) }
    if (argv[2] === 'pregunta' && argv[3] === 'responder') return { value: ok({ aplicado: 'Respondida desde el panel: SQLite' }) }
    return { value: ok({}) }
  })
}

async function montado($, on, acciones, esperas: any[] = []) {
  const r = { acciones, esperas, llamadas: [] as string[][], enviados: [] as string[] }
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await pausa()
  return { r, ui: await $.ui.mount({ ...PANEL }) }
}

async function claves(ui) {
  return (await ui.findAll({ type: 'Button' })).map((b) => b.props.key ?? b.key)
}

test('una pregunta de la terminal va primero en «Para ti», a la vista y con un botón por opción', async ($, on) => {
  const { ui } = await montado($, on, [PEDIDO, ESPERA, ASK], [EN_LA_TABLA])
  expect(await ui.find({ type: 'Text', text: /▶ #15 ❓ Base · ¿Qué base usamos\?/ })).toBeDefined()
  const orden = (await claves(ui)).filter((k) => ['pq-' + ASK.id + '-a', 'b-' + PEDIDO.id, 'preguntas-ver'].includes(k))
  expect(orden).toEqual(['pq-' + ASK.id + '-a', 'b-' + PEDIDO.id, 'preguntas-ver'])
  expect((await ui.find({ key: 'pq-' + ASK.id + '-a' })).props.label).toBe('Postgres (Recommended)')
  expect((await ui.find({ key: 'pq-' + ASK.id + '-b' })).props.label).toBe('SQLite')
  expect((await ui.find({ key: 'preguntas-ver' })).props.label).toContain('? 1')
  expect(await ui.find({ type: 'Text', text: /2 por responder/ })).toBeDefined()
})

test('tocar una opción la responde desde el panel con qdd, sin escribirle nada a la IA', async ($, on) => {
  const { r, ui } = await montado($, on, [ASK])
  await ui.press({ key: 'pq-' + ASK.id + '-b' })
  await pausa()
  expect(r.llamadas.filter((a) => a[3] === 'responder')).toEqual([['qdd', 'consola', 'pregunta', 'responder', ASK.id, 'b']])
  expect(r.enviados).toEqual([])
  expect(await ui.find({ type: 'Text', text: /Respondida desde el panel: SQLite/ })).toBeDefined()
})

test('un permiso abierto va arriba con Permitir una vez y Negar', async ($, on) => {
  const { r, ui } = await montado($, on, [PERMISO])
  expect(await ui.find({ type: 'Text', text: /#16 🔐 Bash: git push/ })).toBeDefined()
  await ui.press({ key: 'pq-' + PERMISO.id + '-negar' })
  await pausa()
  expect(r.llamadas.filter((a) => a[3] === 'responder')).toEqual([['qdd', 'consola', 'pregunta', 'responder', PERMISO.id, 'negar']])
})

test('varias preguntas juntas van arriba, pero se responden en la terminal', async ($, on) => {
  const { ui } = await montado($, on, [DOS])
  expect(await ui.find({ type: 'Text', text: /#17 ❓ Base/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /respóndela en la terminal/ })).toBeDefined()
  expect((await claves(ui)).filter((k) => String(k).startsWith('pq-'))).toEqual([])
})

test('sin preguntas de la terminal, «Para ti» queda como antes', async ($, on) => {
  const { ui } = await montado($, on, [PEDIDO, ESPERA], [EN_LA_TABLA])
  expect((await claves(ui)).filter((k) => String(k).startsWith('pq-'))).toEqual([])
  expect((await ui.find({ key: 'preguntas-ver' })).props.label).toContain('? 1')
})
