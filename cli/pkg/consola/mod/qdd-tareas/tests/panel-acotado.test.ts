import { expect, mock, test } from 'claude-code/testing'

// MEJ-014 (issue 114): lo que crece con el tiempo tiene un tope, y una lectura lenta no se suma a otra ni pisa a una más
// nueva.

const INICIO = { surface: 'terminal', isInteractive: true, cwd: '/work' }
const PEDIDO = { id: 'pedido:1', tipo: 'pedido', tecla: 'a', etiqueta: 'Merge', detalle: 'echo', confirmar: true, estado: 'pendiente' }
const FICHA = (etiqueta: string) => ({ id: 'aprobar:x', tipo: 'aprobar', etiqueta, detalle: '' })
const panel = (acciones: unknown[]) => JSON.stringify({ formato: 1, estado: null, motivo: '', acciones })
const ok = (stdout: string) => ({ exitCode: 0, stdout, stderr: '' })
const pausa = () => new Promise((res) => setTimeout(res, 20))

const TAREAS = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Tareas', isFocused: true, bodyColumns: 60, placement: 'dock', scroll: { offset: 0, bodyRows: 30 }, view: {} },
  surface: 'terminal',
} as const

function preparar(on, r) {
  const clock = mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => {
    r.abiertos += 1
    return { value: { isPlaced: true } }
  })
  on('tool.call', () => ({ result: 'hecho' }))
  on('process.spawn', async function* () {
    return { value: { code: 1, signal: null } }
  })
  on('process.run', async ($, e) => {
    if (e.argv[2] !== 'panel') return { value: ok('{}') }
    r.lecturas += 1
    const respuesta = r.panel
    if (r.lento) await r.lento
    return { value: ok(respuesta) }
  })
  return clock
}

function nuevo(): any {
  return { abiertos: 0, lecturas: 0, panel: panel([]), lento: null }
}

const compuerta = () => {
  let abrir: () => void = () => {}
  const abierta = new Promise<void>((res) => (abrir = res))
  return { abierta, abrir }
}

test('sin flujo, el reloj no lanza otra lectura mientras la anterior sigue en curso', async ($, on) => {
  const r = nuevo()
  const clock = preparar(on, r)
  await $.session.start(INICIO)
  await clock.advance(2000)
  await pausa()
  const lento = compuerta()
  r.lento = lento.abierta
  const antes = r.lecturas
  clock.advance(10000)
  for (let i = 0; i < 100 && r.lecturas === antes; i += 1) await pausa()
  await new Promise((res) => setTimeout(res, 300))
  expect(r.lecturas - antes).toBe(1)
  lento.abrir()
})

test('una respuesta vieja no pisa a una lectura más nueva', async ($, on) => {
  const r = nuevo()
  preparar(on, r)
  await $.session.start(INICIO)
  const ui = await $.ui.mount(TAREAS)
  const lento = compuerta()
  r.panel = panel([FICHA('vieja')])
  r.lento = lento.abierta
  await $.tool.call({ tool: 'mcp__qdd__qdd_estado' })
  await pausa()
  r.panel = panel([FICHA('nueva')])
  r.lento = null
  await ui.press({ key: 'leer' })
  await pausa()
  const etiqueta = async () => (await ui.find({ key: 'b-aprobar:x' })).props.label
  expect(await etiqueta()).toContain('nueva')
  lento.abrir()
  await pausa()
  expect(await etiqueta()).toContain('nueva')
})

test('un pedido que salió del panel se olvida: si vuelve, abre el panel otra vez', async ($, on) => {
  const r = nuevo()
  preparar(on, r)
  await $.session.start(INICIO)
  r.panel = panel([PEDIDO])
  await $.tool.call({ tool: 'mcp__qdd__qdd_estado' })
  await pausa()
  const abiertos = r.abiertos
  expect(abiertos).toBeGreaterThan(0)
  r.panel = panel([])
  await $.tool.call({ tool: 'mcp__qdd__qdd_estado' })
  await pausa()
  expect(r.abiertos).toBe(abiertos)
  r.panel = panel([PEDIDO])
  await $.tool.call({ tool: 'mcp__qdd__qdd_estado' })
  await pausa()
  expect(r.abiertos).toBe(abiertos + 1)
})
