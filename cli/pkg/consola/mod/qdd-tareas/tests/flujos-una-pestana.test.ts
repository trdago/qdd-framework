import { expect, mock, test } from 'claude-code/testing'

// Pedido de la persona (2026-11-07): con un proyecto de 20 flujos de prueba, una pestaña por flujo es inmanejable, y
// cerrarlas también. Todos van en una sola pestaña «Flujos», con un mapa de todos y sus pruebas, y se navega desde ahí.

const PANEL = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Tareas', isFocused: true, bodyColumns: 60, placement: 'dock', scroll: { offset: 0, bodyRows: 30 }, viewportRows: 40 },
} as const

const FLUJOS = { ...PANEL, requestId: 'qdd-flujos', props: { ...PANEL.props, title: 'Flujos' } }

const PAGO = { id: 'pago', nombre: 'Cómo se cobra', nodos: 3, huella: 'h1', pruebas: { pasa: 1, falla: 1, pendiente: 2 } }
const ENVIO = { id: 'envio', nombre: 'Cómo se despacha', nodos: 5, huella: 'h1' }
const ALTA = { id: 'alta', nombre: 'Alta de un cliente', nodos: 2, huella: 'h1' }

function panelCon(flujos: object[]) {
  return JSON.stringify({ formato: 1, carpeta: '/work', acciones: [], flujos, herramientas: [] })
}

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
  on('ui.close', ($, e) => {
    r.cerrados.push(e.id)
    return { value: undefined }
  })
  on('process.run', ($, e) => {
    r.llamadas.push([...e.argv])
    if (e.argv.includes('binario')) return { value: { exitCode: 1, stdout: '', stderr: '' } }
    if (e.argv.includes('panel')) return { value: { exitCode: 0, stdout: panelCon(r.flujos), stderr: '' } }
    if (e.argv.includes('flujo') && e.argv.includes('ver')) {
      const id = e.argv[3]
      const f = r.flujos.find((x) => x.id === id)
      return { value: { exitCode: 0, stdout: JSON.stringify({ id, nombre: f?.nombre || id, lineas: ['( dibujo de ' + id + ' )'], nodos: [] }), stderr: '' } }
    }
    return { value: { exitCode: 0, stdout: '{}', stderr: '' } }
  })
}

const nuevo = (flujos: object[]) => ({ llamadas: [] as string[][], aperturas: [] as any[], cerrados: [] as string[], flujos })
const esperar = () => new Promise((res) => setTimeout(res, 20))
const abiertos = (r) => r.aperturas.map((e) => e.id)

async function empezar($, on, r) {
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
}

test('el botón Flujos abre una sola pestaña para todos los flujos', async ($, on) => {
  const r = nuevo([PAGO, ENVIO, ALTA])
  await empezar($, on, r)
  const ui = await $.ui.mount({ ...PANEL, surface: 'terminal' })
  await ui.press({ key: 'flujos' })
  expect(abiertos(r)).toEqual(['qdd-flujos'])
  expect(r.aperturas[0]).toMatchObject({ title: 'Flujos' })
})

test('las pestañas de un flujo que dejó una versión anterior se cierran solas', async ($, on) => {
  const r = nuevo([PAGO, ENVIO])
  await empezar($, on, r)
  await $.ui.mount({ ...PANEL, surface: 'terminal' })
  await esperar()
  expect(r.cerrados).toContain('qdd-flujo-pago')
  expect(r.cerrados).toContain('qdd-flujo-envio')
})

test('el mapa muestra cada flujo con sus pruebas, y al elegir uno se dibuja ese', async ($, on) => {
  const r = nuevo([PAGO, ENVIO, ALTA])
  await empezar($, on, r)
  const ui = await $.ui.mount({ ...FLUJOS, surface: 'terminal' })
  await esperar()
  expect(await ui.find({ key: 'fl-ir-pago', label: /Cómo se cobra.*✓1 ✗1 ☐2/ })).toBeDefined()
  expect(await ui.find({ key: 'fl-ir-envio', label: /Cómo se despacha/ })).toBeDefined()
  await ui.press({ key: 'fl-ir-envio' })
  await esperar()
  expect(await ui.find({ type: 'Text', text: '( dibujo de envio )' })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: '( dibujo de pago )' })).toBeUndefined()
  expect(await ui.find({ key: 'fl-ir-pago' })).toBeUndefined()
})

test('con un flujo elegido, ‹ › pasan al anterior y al siguiente, y Mapa vuelve a la lista', async ($, on) => {
  const r = nuevo([PAGO, ENVIO, ALTA])
  await empezar($, on, r)
  const ui = await $.ui.mount({ ...FLUJOS, surface: 'terminal' })
  await esperar()
  await ui.press({ key: 'fl-ir-envio' })
  expect(await ui.find({ type: 'Text', text: /2 de 3/ })).toBeDefined()
  await ui.press({ key: 'fl-siguiente' })
  await esperar()
  expect(await ui.find({ type: 'Text', text: '( dibujo de alta )' })).toBeDefined()
  await ui.press({ key: 'fl-anterior' })
  await ui.press({ key: 'fl-anterior' })
  await esperar()
  expect(await ui.find({ type: 'Text', text: '( dibujo de pago )' })).toBeDefined()
  await ui.press({ key: 'fl-mapa' })
  expect(await ui.find({ key: 'fl-ir-alta' })).toBeDefined()
})

test('un flujo que cambió abre la pestaña Flujos y queda elegido', async ($, on) => {
  const r = nuevo([PAGO, ENVIO])
  await empezar($, on, r)
  const principal = await $.ui.mount({ ...PANEL, surface: 'terminal' })
  r.flujos = [PAGO, { ...ENVIO, huella: 'h2' }]
  await principal.press({ key: 'leer' })
  await esperar()
  expect(abiertos(r)).toEqual(['qdd-flujos'])
  expect(r.aperturas[0].focus).toBeFalsy()
  const ui = await $.ui.mount({ ...FLUJOS, surface: 'terminal' })
  await esperar()
  expect(await ui.find({ type: 'Text', text: '( dibujo de envio )' })).toBeDefined()
})

test('con un solo flujo no hay mapa: se dibuja directo', async ($, on) => {
  const r = nuevo([PAGO])
  await empezar($, on, r)
  const ui = await $.ui.mount({ ...FLUJOS, surface: 'terminal' })
  await esperar()
  expect(await ui.find({ type: 'Text', text: '( dibujo de pago )' })).toBeDefined()
  expect(await ui.find({ key: 'fl-mapa' })).toBeUndefined()
})
