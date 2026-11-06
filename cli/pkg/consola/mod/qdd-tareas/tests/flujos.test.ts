import { expect, mock, test } from 'claude-code/testing'

// Un flujo del proyecto (RFC 0012, ampliación) se abre en su pestaña. qdd lo dibuja; el panel solo muestra las líneas.

const PANEL = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Tareas', isFocused: true, bodyColumns: 60, placement: 'dock', scroll: { offset: 0, bodyRows: 30 }, viewportRows: 40 },
} as const

const pestana = (id: string, columnas = 60) => ({ ...PANEL, requestId: 'qdd-flujo-' + id, props: { ...PANEL.props, title: id, bodyColumns: columnas } })

const PAGO = { id: 'pago', nombre: 'Cómo se cobra', nodos: 3, huella: 'h1' }

function panelCon(flujos: object[] | undefined) {
  return JSON.stringify({ formato: 1, carpeta: '/work', acciones: [], ...(flujos ? { flujos, herramientas: [] } : {}) })
}

// preparar deja al panel con una lectura y responde «flujo ver» con el dibujo del ancho que se pidió.
function preparar(on, r) {
  mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', ($, e) => {
    r.abiertos.push(e.id)
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
    if (e.argv.includes('flujo') && r.falla) return { value: { exitCode: 1, stdout: '', stderr: r.falla } }
    if (e.argv.includes('flujo')) {
      const ancho = e.argv[e.argv.indexOf('--ancho') + 1]
      return { value: { exitCode: 0, stdout: JSON.stringify({ id: e.argv[3], nombre: 'Cómo se cobra', lineas: ['( Llega ) a ' + ancho, '   │', r.ultima || '( Fin )'] }), stderr: '' } }
    }
    return { value: { exitCode: 0, stdout: '{}', stderr: '' } }
  })
}

const nuevo = (flujos?: object[]) => ({ llamadas: [] as string[][], abiertos: [] as string[], aperturas: [] as any[], cerrados: [] as string[], flujos, ultima: '' })
const dibujos = (r) => r.llamadas.filter((a) => a.includes('flujo') && a.includes('ver'))
const esperar = () => new Promise((res) => setTimeout(res, 20))

async function empezar($, on, r) {
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
}

test('un flujo que ya estaba no abre una pestaña sola; el botón Flujos la abre', async ($, on) => {
  const r = nuevo([PAGO])
  await empezar($, on, r)
  expect(r.abiertos).not.toContain('qdd-flujo-pago')
  const ui = await $.ui.mount({ ...PANEL, surface: 'terminal' })
  await ui.press({ key: 'flujos' })
  expect(r.abiertos).toContain('qdd-flujo-pago')
})

test('un qdd que no trae flujos no ofrece el botón', async ($, on) => {
  const r = nuevo()
  await empezar($, on, r)
  const ui = await $.ui.mount({ ...PANEL, surface: 'terminal' })
  expect(await ui.find({ key: 'flujos' })).toBeUndefined()
})

test('un flujo nuevo o cambiado abre su pestaña sin quitarle el foco a la conversación', async ($, on) => {
  const r = nuevo([])
  const abiertos = r.aperturas
  await empezar($, on, r)
  const ui = await $.ui.mount({ ...PANEL, surface: 'terminal' })
  r.flujos = [PAGO]
  await ui.press({ key: 'leer' })
  await esperar()
  const abierto = abiertos.find((e) => e.id === 'qdd-flujo-pago')
  expect(abierto).toMatchObject({ title: 'Cómo se cobra' })
  expect(abierto.focus).toBeFalsy()
  abiertos.length = 0
  r.flujos = [{ ...PAGO, huella: 'h2' }]
  await ui.press({ key: 'leer' })
  await esperar()
  expect(abiertos.some((e) => e.id === 'qdd-flujo-pago')).toBe(true)
})

test('la pestaña muestra las líneas de qdd, y se dibuja de nuevo solo si cambia el flujo o el ancho', async ($, on) => {
  const r = nuevo([PAGO])
  await empezar($, on, r)
  const ui = await $.ui.mount({ ...pestana('pago'), surface: 'terminal' })
  await esperar()
  expect(dibujos(r)[0]).toEqual(['qdd', 'flujo', 'ver', 'pago', '--ancho', '60', '--json'])
  expect(await ui.find({ type: 'Text', text: '( Llega ) a 60' })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: '( Fin )' })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: 'Cómo se cobra' })).toBeDefined()
  expect(dibujos(r).length).toBe(1)

  await ui.unmount()
  const angosta = await $.ui.mount({ ...pestana('pago', 40), surface: 'terminal' })
  await esperar()
  expect(dibujos(r).length).toBe(2)
  expect(await angosta.find({ type: 'Text', text: '( Llega ) a 40' })).toBeDefined()
})

test('un cambio en el flujo se ve sin un proceso de más', async ($, on) => {
  const r = nuevo([PAGO])
  await empezar($, on, r)
  const principal = await $.ui.mount({ ...PANEL, surface: 'terminal' })
  const ui = await $.ui.mount({ ...pestana('pago'), surface: 'terminal' })
  await esperar()
  r.ultima = '( Otro fin )'
  r.flujos = [{ ...PAGO, huella: 'h2' }]
  await principal.press({ key: 'leer' })
  await esperar()
  expect(await ui.find({ type: 'Text', text: '( Otro fin )' })).toBeDefined()
  expect(dibujos(r).length).toBe(2)
})

test('un archivo roto dice su problema y no pide el dibujo', async ($, on) => {
  const r = nuevo([{ id: 'roto', nombre: 'roto', nodos: 0, huella: 'x', problema: 'el flujo no tiene nombre' }])
  await empezar($, on, r)
  const ui = await $.ui.mount({ ...pestana('roto'), surface: 'terminal' })
  await esperar()
  expect(await ui.find({ type: 'Text', text: /el flujo no tiene nombre/ })).toBeDefined()
  expect(dibujos(r).length).toBe(0)
})

test('si qdd no puede dibujarlo, lo dice y no lo reintenta en cada dibujo', async ($, on) => {
  const r: any = { ...nuevo([PAGO]), falla: 'no hay un flujo' }
  await empezar($, on, r)
  const ui = await $.ui.mount({ ...pestana('pago'), surface: 'terminal' })
  await esperar()
  await esperar()
  expect(await ui.find({ type: 'Text', text: /no hay un flujo/ })).toBeDefined()
  expect(dibujos(r).length).toBe(1)
})

test('un flujo que se borró cierra su pestaña', async ($, on) => {
  const r = nuevo([PAGO])
  await empezar($, on, r)
  const ui = await $.ui.mount({ ...PANEL, surface: 'terminal' })
  r.flujos = []
  await ui.press({ key: 'leer' })
  await esperar()
  expect(r.cerrados).toContain('qdd-flujo-pago')
})
