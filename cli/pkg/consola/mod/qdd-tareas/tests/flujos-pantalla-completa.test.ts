import { expect, mock, test } from 'claude-code/testing'

// Pedido de la persona (2026-11-08): «cuando estamos en modo flujo debería iniciar de forma expandida usando toda la
// pantalla; el panel debe usar full la pantalla de la consola». Abrir Flujos pide todo el ancho de la terminal; Tareas
// vuelve al reparto de siempre, y ⤡ / ⤢ alterna en la pestaña Flujos.

const PANEL = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 180, rows: 50, isFullscreen: true },
  props: { title: 'Tareas', isFocused: true, bodyColumns: 60, placement: 'dock', scroll: { offset: 0, bodyRows: 30 }, viewportRows: 50 },
} as const

const FLUJOS = { ...PANEL, requestId: 'qdd-flujos', props: { ...PANEL.props, title: 'Flujos' } }
const PAGO = { id: 'pago', nombre: 'Cómo se cobra', nodos: 3, huella: 'h1' }

function preparar(on, r) {
  mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', ($, e) => {
    r.aperturas.push(e)
    return { value: { isPlaced: true } }
  })
  on('ui.close', () => ({ value: undefined }))
  on('process.run', ($, e) => {
    if (e.argv.includes('binario')) return { value: { exitCode: 1, stdout: '', stderr: '' } }
    if (e.argv.includes('panel')) return { value: { exitCode: 0, stdout: JSON.stringify({ formato: 1, carpeta: '/work', acciones: [], flujos: [PAGO], herramientas: [] }), stderr: '' } }
    if (e.argv.includes('flujo') && e.argv.includes('ver')) return { value: { exitCode: 0, stdout: JSON.stringify({ id: 'pago', nombre: 'Cómo se cobra', lineas: ['( dibujo )'], nodos: [] }), stderr: '' } }
    return { value: { exitCode: 0, stdout: '{}', stderr: '' } }
  })
}

const nuevo = () => ({ aperturas: [] as any[] })
const esperar = () => new Promise((res) => setTimeout(res, 20))
const de = (r, id: string) => r.aperturas.filter((e) => e.id === id)

test('el botón Flujos abre la pestaña con todo el ancho de la terminal', async ($, on) => {
  const r = nuevo()
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANEL, surface: 'terminal' })
  await ui.press({ key: 'flujos' })
  expect(de(r, 'qdd-flujos').at(-1)).toMatchObject({ title: 'Flujos', columns: 180 })
})

test('Tareas vuelve al reparto de siempre y ⤡ / ⤢ alterna el ancho de Flujos', async ($, on) => {
  const r = nuevo()
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...FLUJOS, surface: 'terminal' })
  await esperar()
  expect(await ui.find({ key: 'fl-ancho', label: /⤡/ })).toMatchObject({ props: { hotkey: 'z' } })
  await ui.press({ key: 'fl-ancho' })
  expect(de(r, 'qdd-flujos').at(-1).columns).toBeUndefined()
  expect(await ui.find({ key: 'fl-ancho', label: /⤢/ })).toBeDefined()
  await ui.press({ key: 'fl-ancho' })
  expect(de(r, 'qdd-flujos').at(-1)).toMatchObject({ columns: 180 })
  await ui.press({ key: 'fl-tareas' })
  expect(de(r, 'qdd-tareas').at(-1).columns).toBeUndefined()
  expect(de(r, 'qdd-tareas').at(-1)).toMatchObject({ focus: true })
})

test('sin pantalla completa (panel sobre el chat) no pide ancho', async ($, on) => {
  const r = nuevo()
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANEL, viewport: { columns: 180, rows: 50, isFullscreen: false }, props: { ...PANEL.props, placement: 'inline' }, surface: 'terminal' })
  await ui.press({ key: 'flujos' })
  expect(de(r, 'qdd-flujos').at(-1).columns).toBeUndefined()
})
