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
  expect(latidos(r)[0]).toEqual(['qdd', 'consola', 'telefono', 'latir', '--sesion', 'sesion-1'])
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
