import { expect, mock, test } from 'claude-code/testing'

// «Para ti» de todos los proyectos (#256, 2026-11-15): la persona no vio las fichas ni las preguntas de notificador,
// abiertas en otra sesión, y el panel pasó a mostrar debajo de lo propio lo que la espera en los demás proyectos.
// MEJ-086 (2026-11-15): así el tablero de cada consola mostraba proyectos que no eran el de la sesión. La persona eligió
// «Solo este + una línea»: «Para ti» muestra solo este proyecto y, si otros la esperan, una línea que los cuenta y abre
// la vista Proyectos (y), donde está el detalle.

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
    if (argv[2] === 'panel') return { value: ok({ formato: 1, carpeta: '/work', estado: null, motivo: 'x', acciones: r.propias || [], ...(r.otros ? { otros: r.otros } : {}) }) }
    if (argv.includes('responder')) return { value: ok({ aplicado: 'listo', mensaje: 'Respondo a tu ⏳…' }) }
    return { value: ok({}) }
  })
}

async function montado($, on, otros, propias?) {
  const r = { otros, propias, llamadas: [] as string[][], enviados: [] as string[] }
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await pausa()
  return { r, ui: await $.ui.mount({ ...PANEL }) }
}

const OTRO_MAS = { proyecto: 'fairplay_agent', carpeta: '/home/p/fairplay_agent', acciones: [FICHA], esperas: [] }

test('el panel pide lo de los demás proyectos para contarlos', async ($, on) => {
  const { r } = await montado($, on, [])
  const panel = r.llamadas.find((a) => a[2] === 'panel')
  expect(panel?.[panel.indexOf('--con') + 1].split(',')).toContain('otros')
})

test('lo de otro proyecto no se dibuja en «Para ti»: una línea lo cuenta (MEJ-086)', async ($, on) => {
  const { ui } = await montado($, on, [OTRO], [{ ...PEDIDO, id: 'pedido:9', etiqueta: 'Publicar aquí', tecla: 'b' }])
  expect(await ui.find({ type: 'Button', text: /Publicar aquí/ })).toBeDefined()
  expect(await ui.find({ type: 'Button', text: /Subir d5b19cc4/ })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: /¿Subo a producción\?/ })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: /◦ / })).toBeUndefined()
  expect(await ui.find({ key: 'pt-otros' })).toMatchObject({ props: { label: '+ 1 proyecto te espera: notificador_de_ofertas · ver en Proyectos (y)', plain: true } })
})

test('con varios proyectos, la línea los cuenta y los nombra', async ($, on) => {
  const { ui } = await montado($, on, [OTRO, OTRO_MAS])
  expect(await ui.find({ key: 'pt-otros' })).toMatchObject({ props: { label: '+ 2 proyectos te esperan: notificador_de_ofertas, fairplay_agent · ver en Proyectos (y)' } })
})

test('la línea abre la vista Proyectos', async ($, on) => {
  const { r, ui } = await montado($, on, [OTRO])
  await ui.press({ key: 'pt-otros' })
  await pausa()
  expect(r.llamadas.some((a) => a[1] === 'consola' && a[2] === 'proyectos')).toBe(true)
})

test('sin otros proyectos (o con un qdd anterior) no hay línea', async ($, on) => {
  const { ui } = await montado($, on, undefined, [PEDIDO])
  expect(await ui.find({ type: 'Button', text: /Subir d5b19cc4/ })).toBeDefined()
  expect(await ui.find({ key: 'pt-otros' })).toBeUndefined()
})
