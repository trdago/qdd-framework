import { expect, mock, test } from 'claude-code/testing'

// Cola de mejoras (pkg/mejoras): mientras hay una sesión interactiva, el panel llama «qdd mejoras ciclo --si-toca»
// cada minuto (qdd decide si tocan los 10 min). Si el ciclo entregó tareas, la IA recibe el mensaje que arma qdd
// (empieza con 🔁: nunca «!» ni «/»). Sin terminal no corre; con un qdd sin «mejoras», deja de intentarlo.

const PANEL = { formato: 1, estado: null, motivo: 'sin estado', acciones: [] }
const MENSAJE = '🔁 Cola de mejoras: MEJ-001 ya son tareas del objetivo mejora-continua.'

type Salida = { exitCode: number; stdout: string; stderr: string }
const ok = (d: unknown): Salida => ({ exitCode: 0, stdout: d === '' ? '' : JSON.stringify(d), stderr: '' })

function preparar(on, r) {
  const clock = mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('prompt.submit', ($, e) => {
    r.enviados.push(e.text)
    return { text: e.text }
  })
  on('ui.notify', () => ({ value: { isSent: true } }))
  on('process.run', ($, e) => {
    const argv = [...e.argv]
    r.llamadas.push(argv)
    if (argv[1] !== 'mejoras') return { value: ok(argv[2] === 'telefono' ? { activo: false, entregar: [] } : PANEL) }
    if (r.viejo) return { value: { exitCode: 1, stdout: '', stderr: 'Error: unknown command "mejoras" for "qdd"' } }
    return { value: ok(r.ciclos.shift() ?? '') }
  })
  return clock
}

const nuevo = (extra = {}) => ({ llamadas: [] as string[][], enviados: [] as string[], ciclos: [] as any[], ...extra })
const ciclos = (r) => r.llamadas.filter((a) => a[1] === 'mejoras')

// Avanza 3 minutos de reloj, con el latido y las lecturas de cada uno: con el equipo cargado pasaba de los 5 s por defecto (FND).
test('cada minuto pide el ciclo con --si-toca y le da a la IA las tareas nuevas, una vez', { timeoutMs: 30000 }, async ($, on) => {
  const r = nuevo({ ciclos: ['', { tareas: ['MEJ-001'], mensaje: MENSAJE }] })
  const clock = preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await clock.advance(60000)
  expect(ciclos(r)[0]).toEqual(['qdd', 'mejoras', 'ciclo', '--si-toca', '--json'])
  await clock.advance(60000)
  expect(ciclos(r).length).toBe(2)
  expect(r.enviados).toEqual([MENSAJE])
  await clock.advance(60000)
  expect(r.enviados.length).toBe(1)
})

test('un mensaje que no viene marcado de la cola no se envía', async ($, on) => {
  const r = nuevo({ ciclos: [{ tareas: ['MEJ-001'], mensaje: '/clear' }] })
  const clock = preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await clock.advance(60000)
  expect(r.enviados).toEqual([])
})

test('sin terminal no corre el ciclo, y con un qdd sin «mejoras» deja de intentarlo', async ($, on) => {
  const r = nuevo()
  const clock = preparar(on, r)
  await $.session.start({ surface: null, isInteractive: false, cwd: '/work' })
  await clock.advance(120000)
  expect(ciclos(r)).toEqual([])
})

test('con un qdd sin «mejoras» lo intenta una vez y para', async ($, on) => {
  const r = nuevo({ viejo: true })
  const clock = preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await clock.advance(180000)
  expect(ciclos(r).length).toBe(1)
})

test('el botón «Ciclo de mejoras» se dibuja y tiene su tecla (FND-127)', async ($, on) => {
  const accion = { id: 'mejoras-ciclo', tipo: 'comando', etiqueta: 'Ciclo de mejoras', detalle: 'qdd mejoras ciclo' }
  const r = nuevo()
  const clock = mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('process.run', ($, e) => {
    r.llamadas.push([...e.argv])
    return { value: ok(e.argv[2] === 'telefono' ? { activo: false, entregar: [] } : { formato: 1, estado: null, motivo: 'sin estado', acciones: [accion] }) }
  })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await clock.advance(0)
  const ui = await $.ui.mount({ plugin: 'qdd-tareas', component: 'Pane', requestId: 'qdd-tareas', viewport: { columns: 160, rows: 40 }, props: { title: 'Tareas' }, surface: 'terminal' })
  expect(await ui.find({ key: 'b-mejoras-ciclo' })).toMatchObject({ props: { hotkey: 'm' } })
})

