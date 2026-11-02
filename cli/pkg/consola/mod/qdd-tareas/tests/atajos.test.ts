import { expect, mock, test } from 'claude-code/testing'

// Atajo desde el chat (pedido del usuario, 2026-11-01): sin mouse y en cualquier terminal. Con el chat vacío, la tecla
// de lo que te espera y Enter abren su confirmación en el panel con el foco en «s  Sí»; otro Enter aprueba y el teclado
// vuelve al chat. Un carácter privado (U+E000 + la letra) hace lo mismo en un solo paso, para quien mapee AltGr+letra.

const PANE = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Tareas', isFocused: true, bodyColumns: 60, placement: 'dock', scroll: { offset: 0, bodyRows: 30 }, view: {} },
} as const

const PEDIDO = { id: 'pedido:p1', tipo: 'pedido', tecla: 'a', etiqueta: 'Publicar v1', detalle: 'git push origin v1', confirmar: true, estado: 'pendiente' }
const HECHO = { id: 'pedido:p0', tipo: 'pedido', tecla: 'b', etiqueta: 'Ya corrió', detalle: 'true', confirmar: true, estado: 'hecho' }
const FICHA = { id: 'aprobar:pagos', tipo: 'aprobar', etiqueta: 'Aprobar «Pagos»', detalle: 'qdd modules approve pagos', confirmar: true }
const PANEL = {
  formato: 1,
  estado: { formato: 1, resumen: '', etapas: ['Código'], abreviaturas: ['Có'], frentes: [{ id: 'x', nombre: 'Algo', estado: 'abierto', simbolos: ['⏳'] }], esperas: [], fallas: [] },
  acciones: [PEDIDO, HECHO, FICHA],
}

function preparar(on, r: { llamadas: string[][]; aperturas: any[]; registros: any[]; rellenos: any[]; enviados: string[] }) {
  const clock = mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', ($, e) => {
    r.aperturas.push(e)
    return { value: { isPlaced: true } }
  })
  // El kit no simula $.ui.focus (el foco en «s  Sí» se probó en Claude Code 2.1.300, en tmux): el panel lo anota.
  on('ui.log', ($, e) => {
    r.registros.push(e)
    return { value: undefined }
  })
  on('prompt.fill', ($, e) => {
    r.rellenos.push(e)
    return { isFilled: true, text: e.text, cursor: 0 }
  })
  on('prompt.submit', ($, e) => {
    r.enviados.push(e.text)
    return { text: e.text }
  })
  on('prompt.edit', ($, e) => {
    const text = e.text.slice(0, e.start) + e.inputText + e.text.slice(e.end)
    return { text, cursor: e.start + e.inputText.length }
  })
  on('process.run', ($, e) => {
    r.llamadas.push([...e.argv])
    if (e.argv[2] === 'accion') return { value: { exitCode: 0, stdout: '', stderr: '' } }
    if (e.argv[1] === 'herramienta') return { value: { exitCode: 0, stdout: JSON.stringify({ herramientas: [] }), stderr: '' } }
    return { value: { exitCode: 0, stdout: JSON.stringify(PANEL), stderr: '' } }
  })
  return clock
}

const nuevo = () => ({ llamadas: [] as string[][], aperturas: [] as any[], registros: [] as any[], rellenos: [] as any[], enviados: [] as string[] })

test('la tecla de un pedido y Enter abren su confirmación con el foco en «s  Sí», sin enviarle nada a la IA', async ($, on) => {
  const r = nuevo()
  const clock = preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  const envio = await $.prompt.submit({ text: 'a' })
  expect(envio).toMatchObject({ drop: expect.stringContaining('Publicar v1') })
  await clock.advance(100)
  expect(r.rellenos.at(-1)).toMatchObject({ text: '' })
  expect(r.aperturas.at(-1)).toMatchObject({ id: 'qdd-tareas', focus: true })
  expect(await ui.find({ type: 'Text', text: '¿Publicar v1?' })).toBeDefined()
  expect(r.llamadas.some((a) => a[2] === 'accion')).toBe(false)
  expect(r.enviados).toEqual([])
  await ui.press({ key: 'si' })
  expect(r.llamadas).toContainEqual(['qdd', 'consola', 'accion', 'pedido:p1', '--confirmado'])
})

test('un dígito y Enter abren la ficha por aprobar con esa tecla', async ($, on) => {
  const r = nuevo()
  const clock = preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await $.prompt.submit({ text: '1' })).toMatchObject({ drop: expect.stringContaining('Pagos') })
  await clock.advance(100)
  expect(await ui.find({ type: 'Text', text: '¿Aprobar «Pagos»?' })).toBeDefined()
})

test('lo demás va a la IA como siempre: otra palabra, una tecla sin pendiente o un pedido ya hecho', async ($, on) => {
  const r = nuevo()
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await $.ui.mount({ ...PANE, surface: 'terminal' })
  for (const texto of ['a.', 'hola', 'z', 'b', ' a b']) {
    const envio = await $.prompt.submit({ text: texto })
    expect(envio).toMatchObject({ text: texto })
  }
  expect(r.aperturas.filter((a) => a.focus)).toEqual([])
  expect(r.enviados).toEqual(['a.', 'hola', 'z', 'b', ' a b'])
})

test('mientras escribes la tecla, el chat dice qué abre Enter', async ($, on) => {
  const r = nuevo()
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await $.ui.mount({ ...PANE, surface: 'terminal' })
  const conTecla = await $.prompt.edit({ origin: { kind: 'composer' }, key: { key: 'a' }, text: '', cursor: 0, start: 0, end: 0, inputText: 'a' })
  expect(conTecla.decorations?.[0]).toMatchObject({ virtualText: expect.stringContaining('Publicar v1'), dimColor: true })
  const otra = await $.prompt.edit({ origin: { kind: 'composer' }, key: { key: 'z' }, text: '', cursor: 0, start: 0, end: 0, inputText: 'z' })
  expect(otra.decorations ?? []).toEqual([])
})

test('U+E061 (AltGr+a mapeado por el terminal) abre el pedido a en un paso y no escribe nada en el chat', async ($, on) => {
  const r = nuevo()
  const clock = preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  const editado = await $.prompt.edit({ origin: { kind: 'composer' }, text: '', cursor: 0, start: 0, end: 0, inputText: '\uE061' })
  expect(editado).toMatchObject({ text: '', cursor: 0 })
  await clock.advance(100)
  expect(r.aperturas.at(-1)).toMatchObject({ id: 'qdd-tareas', focus: true })
  expect(r.rellenos).toEqual([])
  expect(await ui.find({ type: 'Text', text: '¿Publicar v1?' })).toBeDefined()
})
