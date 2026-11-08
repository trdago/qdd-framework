import { expect, mock, test } from 'claude-code/testing'

// Pedido del usuario (2026-11-02): «el chat principal tiene que enterarse y verse una leyenda cuando acepto la tarea en
// el panel». Al confirmar, el panel dice su sesión a qdd, le deja a la IA una nota de que el pedido corre aparte y una
// banda arriba del prompt dice «▶ … en marcha» mientras corre. Al terminar, la sesión que lo confirmó (y solo ella, una
// vez: lo decide qdd con «consola avisado») le envía el resultado a la IA, que sigue sola, y sale una notificación.

const PANE = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Tareas', isFocused: true, bodyColumns: 60, placement: 'dock', scroll: { offset: 0, bodyRows: 30 }, view: {} },
} as const

const BANDA = {
  plugin: 'qdd-tareas',
  component: 'AbovePrompt',
  viewport: { columns: 120, rows: 40 },
  props: { hasSurvey: false, isWorking: false, maxRows: 10, scroll: { offset: 0, bodyRows: 10 } },
} as const

const ESTADO = { formato: 1, resumen: '', etapas: ['Código'], abreviaturas: ['Có'], frentes: [{ id: 'x', nombre: 'Algo', estado: 'abierto', simbolos: ['⏳'] }], esperas: [], fallas: [] }
const PUBLICAR = { id: 'pedido:1', tipo: 'pedido', tecla: 'a', etiqueta: 'Publicar rc.16', detalle: 'bash paso-rc16.sh', confirmar: true, estado: 'pendiente' }
const MENSAJE = '✓ Pedido [a] «Publicar rc.16» terminó bien (exit 0). Salida: publicado Sigue con lo que venía después.'

function preparar(on, r) {
  const clock = mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  // La banda de siempre (la del motor): sin pedidos en marcha, el panel la deja pasar.
  on('ui.render', { component: 'AbovePrompt' }, ($, e) => $.ui.resolve(e).Box({ key: 'motor', children: [] }))
  on('session.append', ($, e, next) => {
    r.notas.push(e.message)
    return next(e)
  })
  on('prompt.submit', ($, e) => {
    r.enviados.push(e.text)
    return { text: e.text }
  })
  on('ui.notify', ($, e) => {
    r.notificaciones.push(e)
    return { value: { isSent: true } }
  })
  on('process.run', ($, e) => {
    r.llamadas.push([...e.argv])
    if (r.viejo && e.argv.includes('--sesion')) return { value: { exitCode: 1, stdout: '', stderr: 'Error: unknown flag: --sesion' } }
    if (e.argv[2] === 'accion') return { value: { exitCode: 0, stdout: '[+] En marcha', stderr: '' } }
    if (e.argv[2] === 'avisado') return { value: { exitCode: r.reclamos++ === 0 ? 0 : 1, stdout: r.alReclamar ?? '', stderr: '' } }
    if (e.argv[1] === 'herramienta') return { value: { exitCode: 0, stdout: JSON.stringify({ herramientas: [] }), stderr: '' } }
    return { value: { exitCode: 0, stdout: JSON.stringify({ formato: 1, estado: ESTADO, acciones: [r.pedido] }), stderr: '' } }
  })
  return clock
}

const nuevo = (pedido) => ({ pedido, llamadas: [] as string[][], notas: [] as any[], enviados: [] as string[], notificaciones: [] as any[], reclamos: 0 })

test('al confirmar dice su sesión, la IA sabe que corre aparte y la banda dice «en marcha»', async ($, on) => {
  const r = nuevo(PUBLICAR)
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await ui.advance(1000)
  await ui.press({ key: 'b-pedido:1' })
  r.pedido = { ...PUBLICAR, tecla: undefined, estado: 'ejecutando', sesion: 'sesion-1', inicio: '2026-11-02T01:00:00-03:00' }
  await ui.press({ key: 'si' })
  expect(r.llamadas).toContainEqual(['qdd', 'consola', 'accion', 'pedido:1', '--confirmado', '--sesion', 'sesion-1'])
  const nota = JSON.stringify(r.notas)
  expect(nota).toContain('Publicar rc.16')
  expect(nota).toMatch(/corre aparte/)
  expect(r.notas[0]).toMatchObject({ type: 'user' })
  expect(r.enviados).toEqual([])
  const banda = await $.ui.mount({ ...BANDA, surface: 'terminal' })
  expect(await banda.find({ type: 'Text', text: /▶ \[a\] Publicar rc\.16 · en marcha/ })).toBeDefined()
})

test('un pedido de otra sesión no va en la banda de esta', async ($, on) => {
  const r = nuevo({ ...PUBLICAR, tecla: undefined, estado: 'ejecutando', sesion: 'otra' })
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await ui.advance(1000)
  const banda = await $.ui.mount({ ...BANDA, surface: 'terminal' })
  expect(await banda.find({ type: 'Text', text: /en marcha/ })).toBeUndefined()
})

test('al terminar, la sesión que lo confirmó lo reclama una vez, se lo envía a la IA y notifica', async ($, on) => {
  const r = nuevo({ ...PUBLICAR, tecla: undefined, estado: 'hecho', sesion: 'sesion-1', avisar: true, mensaje: MENSAJE, resultado: 'publicado' })
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await ui.advance(1000)
  await ui.advance(5000)
  expect(r.llamadas.filter((l) => l[2] === 'avisado')[0]).toEqual(['qdd', 'consola', 'avisado', 'pedido:1', '--sesion', 'sesion-1'])
  expect(r.enviados).toEqual([MENSAJE])
  expect(r.notificaciones).toHaveLength(1)
  expect(r.notificaciones[0]).toMatchObject({ text: expect.stringContaining('Publicar rc.16') })
  const banda = await $.ui.mount({ ...BANDA, surface: 'terminal' })
  expect(await banda.find({ type: 'Text', text: /en marcha/ })).toBeUndefined()
})

// FND-187: el panel leyó el pedido como cortado y, al reclamarlo, ya había terminado bien. Va lo que dice qdd al
// reclamarlo, no lo de la lectura anterior.
test('a la IA le llega el resultado que qdd da al reclamarlo, no el de la lectura anterior', async ($, on) => {
  const cortado = '✗ Pedido [a] «Publicar rc.16» falló (exit -1). Salida: se cortó'
  const r = { ...nuevo({ ...PUBLICAR, tecla: undefined, estado: 'fallo', sesion: 'sesion-1', avisar: true, mensaje: cortado }), alReclamar: MENSAJE + '\n' }
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await ui.advance(5000)
  expect(r.enviados).toEqual([MENSAJE])
  expect(r.notificaciones[0]).toMatchObject({ text: expect.stringContaining('terminó') })
})

test('si qdd dice que ya se avisó (otra sesión, un reload), no se envía de nuevo', async ($, on) => {
  const r = nuevo({ ...PUBLICAR, tecla: undefined, estado: 'fallo', sesion: 'sesion-1', avisar: true, mensaje: '✗ falló' })
  r.reclamos = 1
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await ui.advance(5000)
  expect(r.enviados).toEqual([])
  expect(r.notificaciones).toEqual([])
})

test('el aviso de otra sesión no se reclama aquí', async ($, on) => {
  const r = nuevo({ ...PUBLICAR, tecla: undefined, estado: 'hecho', sesion: 'otra', avisar: true, mensaje: MENSAJE })
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await ui.advance(5000)
  expect(r.llamadas.filter((l) => l[2] === 'avisado')).toEqual([])
  expect(r.enviados).toEqual([])
})

test('con un qdd anterior a --sesion, el pedido corre igual (sin aviso)', async ($, on) => {
  const r = { ...nuevo(PUBLICAR), viejo: true }
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await ui.advance(1000)
  await ui.press({ key: 'b-pedido:1' })
  await ui.press({ key: 'si' })
  expect(r.llamadas).toContainEqual(['qdd', 'consola', 'accion', 'pedido:1', '--confirmado'])
})
