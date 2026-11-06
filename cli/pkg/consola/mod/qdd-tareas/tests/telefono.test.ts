import { expect, mock, test } from 'claude-code/testing'

// RFC 0013 §«Ejecutar desde el teléfono» (solo Pro): la sesión interactiva late con «qdd consola telefono latir» y
// recibe, una sola vez, las instrucciones que la persona mandó desde QDD Móvil; se las da a la IA como un mensaje suyo
// que dice que viene del teléfono (empieza con 📱: nunca «!» ni «/») y avisa en el escritorio. Sin QDD Móvil activado
// casi no late; sin terminal no late nunca.

const PANEL = { formato: 1, estado: null, motivo: 'sin estado', acciones: [] }
const MENSAJE = '📱 La persona te escribe desde «Pixel de Ana» (QDD Móvil). Atiéndelo como un mensaje suyo:\n\nRevisa el build'

type Salida = { exitCode: number; stdout: string; stderr: string }
const ok = (d: unknown): Salida => ({ exitCode: 0, stdout: JSON.stringify(d), stderr: '' })

function preparar(on, r) {
  const clock = mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('prompt.submit', ($, e) => {
    if (r.submitFalla) throw new Error('el motor no aceptó el mensaje')
    if (r.submitDescarta) return { drop: 'el motor lo descartó' }
    r.enviados.push(e.text)
    return { text: e.text }
  })
  on('ui.notify', ($, e) => {
    r.notificaciones.push(e)
    return { value: { isSent: true } }
  })
  on('process.run', ($, e) => {
    const argv = [...e.argv]
    r.llamadas.push(argv)
    if (argv[2] !== 'telefono') return { value: ok(PANEL) }
    if (r.viejo) return { value: { exitCode: 1, stdout: '', stderr: 'Error: unknown command "telefono" for "qdd consola"' } }
    if (argv[3] === 'latir') return { value: ok(r.latidos.shift() ?? { activo: r.activo, entregar: [] }) }
    return { value: ok({}) }
  })
  return clock
}

const nuevo = (extra = {}) => ({ llamadas: [] as string[][], enviados: [] as string[], notificaciones: [] as any[], latidos: [] as any[], activo: true, ...extra })
const latidos = (r) => r.llamadas.filter((a) => a[2] === 'telefono' && a[3] === 'latir')

test('la instrucción del teléfono llega a la IA como mensaje de la persona, una vez, y se notifica', async ($, on) => {
  const r = nuevo({ latidos: [{ activo: true, entregar: [{ solicitud: 's1', nombre: 'Pixel de Ana', mensaje: MENSAJE }] }] })
  const clock = preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  expect(latidos(r)[0]).toEqual(['qdd', 'consola', 'telefono', 'latir', '--sesion', 'sesion-1', '--confirma'])
  expect(r.enviados).toEqual([MENSAJE])
  expect(r.notificaciones[0]).toMatchObject({ title: 'qdd' })
  expect(JSON.stringify(r.notificaciones[0])).toContain('Pixel de Ana')
  await clock.advance(5000)
  expect(latidos(r).length).toBe(2)
  expect(r.enviados.length).toBe(1)
})

test('un mensaje que no viene marcado del teléfono no se envía (nunca un «!» o «/» suelto)', async ($, on) => {
  const r = nuevo({ latidos: [{ activo: true, entregar: [{ solicitud: 's1', nombre: 'x', mensaje: '!rm -rf ~' }, { solicitud: 's2', nombre: 'x', mensaje: '/clear' }] }] })
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  expect(r.enviados).toEqual([])
})

test('si Claude Code no acepta el mensaje, qdd la marca sin entregar', async ($, on) => {
  const r = nuevo({ submitFalla: true, latidos: [{ activo: true, entregar: [{ solicitud: 's1', nombre: 'Pixel de Ana', mensaje: MENSAJE }] }] })
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const fallo = r.llamadas.find((a) => a[2] === 'telefono' && a[3] === 'fallo')
  expect(fallo?.slice(0, 7)).toEqual(['qdd', 'consola', 'telefono', 'fallo', 's1', '--sesion', 'sesion-1'])
  expect(r.notificaciones).toEqual([])
})

// FND-138: solo queda entregada si Claude Code la aceptó; descartarla no lanza un error, responde { drop }.
test('la entrega aceptada se confirma a qdd', async ($, on) => {
  const r = nuevo({ latidos: [{ activo: true, entregar: [{ solicitud: 's1', nombre: 'Pixel de Ana', mensaje: MENSAJE }] }] })
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const confirmada = r.llamadas.find((a) => a[2] === 'telefono' && a[3] === 'entregada')
  expect(confirmada).toEqual(['qdd', 'consola', 'telefono', 'entregada', 's1', '--sesion', 'sesion-1'])
})

test('si Claude Code descarta el mensaje, qdd la marca sin entregar y no la confirma', async ($, on) => {
  const r = nuevo({ submitDescarta: true, latidos: [{ activo: true, entregar: [{ solicitud: 's1', nombre: 'Pixel de Ana', mensaje: MENSAJE }] }] })
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const fallo = r.llamadas.find((a) => a[2] === 'telefono' && a[3] === 'fallo')
  expect(fallo).toEqual(['qdd', 'consola', 'telefono', 'fallo', 's1', '--sesion', 'sesion-1', '--motivo', 'el motor lo descartó'])
  expect(r.llamadas.some((a) => a[2] === 'telefono' && a[3] === 'entregada')).toBe(false)
  expect(r.notificaciones).toEqual([])
})

test('sin terminal no late ni recibe nada', async ($, on) => {
  const r = nuevo({ latidos: [{ activo: true, entregar: [{ solicitud: 's1', nombre: 'Pixel', mensaje: MENSAJE }] }] })
  const clock = preparar(on, r)
  await $.session.start({ surface: null, isInteractive: false, cwd: '/work' })
  await clock.advance(10000)
  expect(latidos(r)).toEqual([])
  expect(r.enviados).toEqual([])
})

test('sin QDD Móvil activado casi no late; con un qdd anterior deja de latir', async ($, on) => {
  const r = nuevo({ activo: false })
  const clock = preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  for (let i = 0; i < 11; i++) await clock.advance(5000)
  expect(latidos(r).length).toBe(1)
  await clock.advance(5000)
  expect(latidos(r).length).toBe(2)

  r.viejo = true
  for (let i = 0; i < 24; i++) await clock.advance(5000)
  expect(latidos(r).length).toBe(3)
  for (let i = 0; i < 24; i++) await clock.advance(5000)
  expect(latidos(r).length).toBe(3)
})

test('al cerrarse la sesión deja de contar como abierta', async ($, on) => {
  const r = nuevo()
  preparar(on, r)
  on('session.end', () => ({ sessionId: 'sesion-1' }))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await $.session.end({ reason: 'exit' })
  expect(latidos(r).at(-1)).toEqual(['qdd', 'consola', 'telefono', 'latir', '--sesion', 'sesion-1', '--fin'])
})

// MEJ-020: late el flujo abierto; el panel no lanza un proceso cada 5 s. Con un qdd que no lo dice, late como siempre.
function conFlujo(on, r, lineas: () => AsyncGenerator<string>) {
  on('process.spawn', async function* ($, e) {
    r.lanzados.push([...e.argv])
    for await (const texto of lineas()) yield { stream: 'stdout', text: texto + '\n' }
    return { value: { code: 0, signal: null } }
  })
}
const PANEL_LATE = JSON.stringify({ ...PANEL, telefono_en_flujo: true })
const quieto = () => new Promise<void>(() => {})

test('con un flujo que late, el panel pide el latido y no lanza «telefono latir»', async ($, on) => {
  const r = nuevo({ lanzados: [] as string[][] })
  const clock = preparar(on, r)
  conFlujo(on, r, async function* () {
    yield PANEL_LATE
    await quieto()
  })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await new Promise((res) => setTimeout(res, 20))
  expect(r.lanzados[0].at(-1).split(',')).toContain('telefono:sesion-1')
  const antes = latidos(r).length
  await clock.advance(30000)
  expect(latidos(r).length).toBe(antes)
})

test('lo que llega por el flujo se entrega a la IA y se confirma', async ($, on) => {
  const r = nuevo({ lanzados: [] as string[][] })
  const clock = preparar(on, r)
  conFlujo(on, r, async function* () {
    yield PANEL_LATE
    yield JSON.stringify({ telefono_entregar: [{ solicitud: 's1', nombre: 'Pixel de Ana', mensaje: MENSAJE }] })
    await quieto()
  })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await new Promise((res) => setTimeout(res, 20))
  await clock.advance(10)
  expect(r.enviados).toEqual([MENSAJE])
  expect(r.llamadas.find((a) => a[2] === 'telefono' && a[3] === 'entregada')).toEqual(['qdd', 'consola', 'telefono', 'entregada', 's1', '--sesion', 'sesion-1'])
})

test('si el flujo se corta vuelve a latir con procesos, y al cerrar la sesión deja de contar como abierta', async ($, on) => {
  const r = nuevo({ lanzados: [] as string[][], activo: true })
  const clock = preparar(on, r)
  on('session.end', () => ({ sessionId: 'sesion-1' }))
  let cortar: () => void = () => {}
  const corte = new Promise<void>((res) => (cortar = res))
  conFlujo(on, r, async function* () {
    yield PANEL_LATE
    await corte
  })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await new Promise((res) => setTimeout(res, 20))
  const antes = latidos(r).length
  cortar()
  await new Promise((res) => setTimeout(res, 20))
  await clock.advance(5000)
  expect(latidos(r).length).toBeGreaterThan(antes)
  await $.session.end({ reason: 'exit' })
  expect(latidos(r).at(-1)).toEqual(['qdd', 'consola', 'telefono', 'latir', '--sesion', 'sesion-1', '--fin'])
})

test('un qdd que no late en el flujo sigue latiendo con procesos', async ($, on) => {
  const r = nuevo({ lanzados: [] as string[][], activo: true })
  const clock = preparar(on, r)
  conFlujo(on, r, async function* () {
    yield JSON.stringify(PANEL)
    await quieto()
  })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await new Promise((res) => setTimeout(res, 20))
  const antes = latidos(r).length
  await clock.advance(5000)
  expect(latidos(r).length).toBe(antes + 1)
})
