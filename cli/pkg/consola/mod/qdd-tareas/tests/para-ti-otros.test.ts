import { expect, mock, test } from 'claude-code/testing'

// «Para ti» de todos los proyectos (2026-11-15): la persona mira el panel de una sesión y no vio las fichas ni las
// preguntas de notificador, abiertas en otra. El panel muestra, debajo de lo propio, lo que espera a la persona en los
// demás proyectos (qdd consola panel --con otros) y lo responde ahí mismo con 'qdd consola --dir <carpeta> …'. Como en
// lo propio: lo de la terminal y las acciones a la vista, los ⏳ plegados.

const PANEL = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Tareas', isFocused: true, bodyColumns: 60, placement: 'dock', scroll: { offset: 0, bodyRows: 80 } },
  surface: 'terminal',
} as const

const CARPETA = '/home/p/notificador_de_ofertas'
const FICHA = { id: 'aprobar:usuarios', tipo: 'aprobar', etiqueta: 'Aprobar «Usuarios»', detalle: 'qdd modules approve usuarios', confirmar: true, turno: 3 }
const PEDIDO = { id: 'pedido:1', tipo: 'pedido', etiqueta: 'Subir d5b19cc4', detalle: 'git push', tecla: 'a', estado: 'pendiente', confirmar: true, turno: 1 }
const ASK = {
  id: 'pregunta:abc', tipo: 'pregunta', origen: 'ia', estado: 'pendiente', etiqueta: '❓ Push · ¿Subo a producción?', detalle: '¿Subo a producción?', confirmar: true, turno: 2,
  preguntas: [{ id: 'q1', encabezado: 'Push', texto: '¿Subo a producción?', opciones: [{ valor: 'a', etiqueta: 'Sí (Recommended)' }, { valor: 'b', etiqueta: 'No' }] }],
}
const ESPERA = { id: 'pregunta:espera-9', tipo: 'pregunta', origen: 'estado', estado: 'pendiente', etiqueta: '⏳', detalle: '¿SES listo?', confirmar: true, turno: 4 }
const OTRO = { proyecto: 'notificador_de_ofertas', carpeta: CARPETA, acciones: [PEDIDO, ASK, FICHA, ESPERA], esperas: [{ pregunta: ESPERA.id, atender: 'atender:ses:0', donde: 'SES · Producción', texto: '¿SES listo?' }] }

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
    if (argv[2] === 'panel') return { value: ok({ formato: 1, carpeta: '/work', estado: null, motivo: 'x', acciones: [], ...(r.otros ? { otros: r.otros } : {}) }) }
    if (argv.includes('responder')) return { value: ok({ aplicado: 'listo', mensaje: 'Respondo a tu ⏳…' }) }
    return { value: ok({}) }
  })
}

async function montado($, on, otros) {
  const r = { otros, llamadas: [] as string[][], enviados: [] as string[] }
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await pausa()
  return { r, ui: await $.ui.mount({ ...PANEL }) }
}

const k = (a) => 'ot-' + CARPETA + '-' + a.id
const deQDD = (r, que: string) => r.llamadas.filter((a) => a.includes('--dir') && a.includes(que))

test('el panel pide lo de los demás proyectos', async ($, on) => {
  const { r } = await montado($, on, [])
  const panel = r.llamadas.find((a) => a[2] === 'panel')
  expect(panel?.[panel.indexOf('--con') + 1]).toContain('otros')
})

test('lo de otro proyecto va en «Para ti» bajo su nombre: la terminal y las acciones a la vista, los ⏳ plegados', async ($, on) => {
  const { ui } = await montado($, on, [OTRO])
  expect(await ui.find({ type: 'Text', text: /◦ notificador_de_ofertas/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /❓ Push · ¿Subo a producción\?/ })).toBeDefined()
  expect(await ui.find({ key: k(FICHA) })).toBeDefined()
  expect(await ui.find({ key: k(PEDIDO) })).toBeDefined()
  expect((await ui.find({ key: 'ot-ver-' + CARPETA })).props.label).toContain('? 1')
  expect(await ui.find({ type: 'Text', text: /4 en otros proyectos/ })).toBeDefined()
})

test('aprobar una ficha de otro proyecto pide confirmación, nombra el proyecto y corre allá', async ($, on) => {
  const { r, ui } = await montado($, on, [OTRO])
  await ui.press({ key: k(FICHA) })
  expect(await ui.find({ type: 'Text', text: /notificador_de_ofertas/ })).toBeDefined()
  expect(deQDD(r, 'accion')).toEqual([])
  await ui.press({ key: 'si' })
  await pausa()
  expect(deQDD(r, 'accion')).toEqual([['qdd', 'consola', '--dir', CARPETA, 'accion', FICHA.id, '--confirmado']])
})

test('un pedido de otro proyecto se confirma desde aquí y corre en su proyecto', async ($, on) => {
  const { r, ui } = await montado($, on, [OTRO])
  await ui.press({ key: k(PEDIDO) })
  await ui.press({ key: 'si' })
  await pausa()
  expect(deQDD(r, 'accion')).toEqual([['qdd', 'consola', '--dir', CARPETA, 'accion', PEDIDO.id, '--confirmado']])
  expect(r.enviados).toEqual([])
})

test('una pregunta de la terminal de otro proyecto se responde con su botón', async ($, on) => {
  const { r, ui } = await montado($, on, [OTRO])
  await ui.press({ key: 'pq-' + CARPETA + '-' + ASK.id + '-b' })
  await pausa()
  expect(deQDD(r, 'responder')).toEqual([['qdd', 'consola', '--dir', CARPETA, 'pregunta', 'responder', ASK.id, 'b']])
})

test('un ⏳ de otro proyecto se despliega y se responde allá, sin escribirle a la IA de esta sesión', async ($, on) => {
  const { r, ui } = await montado($, on, [OTRO])
  await ui.press({ key: 'ot-ver-' + CARPETA })
  expect(await ui.find({ type: 'Text', text: /¿SES listo\?/ })).toBeDefined()
  await ui.press({ key: 'ot-espera-' + CARPETA + '-0-hecho' })
  await pausa()
  expect(deQDD(r, 'responder')).toEqual([['qdd', 'consola', '--dir', CARPETA, 'pregunta', 'responder', ESPERA.id, 'hecho']])
  expect(r.enviados).toEqual([])
})

test('sin otros proyectos (o con un qdd anterior) «Para ti» queda como antes', async ($, on) => {
  const { ui } = await montado($, on, undefined)
  expect(await ui.find({ type: 'Text', text: /◦ / })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: /otros proyectos/ })).toBeUndefined()
})
