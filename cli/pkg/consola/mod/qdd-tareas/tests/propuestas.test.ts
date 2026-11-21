import { expect, mock, test } from 'claude-code/testing'

// Flujo para conversar (RFC 0025, fase 3): cada 15 s el panel pide «qdd flujo propuestas ciclo --si-toca» y qdd decide
// si hay invitaciones vivas que leer. Sin terminal no corre; un qdd sin el comando deja de intentarlo.

const PANEL = { formato: 1, estado: null, motivo: 'sin estado', acciones: [] }
const ok = (d: unknown) => ({ exitCode: 0, stdout: JSON.stringify(d), stderr: '' })

function preparar(on, r) {
  const clock = mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('process.run', ($, e) => {
    const argv = [...e.argv]
    r.llamadas.push(argv)
    if (argv[1] === 'flujo' && argv[2] === 'propuestas' && r.viejo) return { value: { exitCode: 1, stdout: '', stderr: 'Error: unknown command "propuestas" for "qdd flujo"' } }
    if (argv[1] === 'flujo' && argv[2] === 'propuestas') return { value: ok({ nuevas: 0, pedidas: 0, rechazadas: 0, esperan: 0 }) }
    if (argv[2] === 'telefono') return { value: ok({ activo: false, entregar: [] }) }
    if (argv[1] === 'estado' && argv[2] === 'verificar') return { value: ok({ revisadas: [], cumplidas: [], avisos: [] }) }
    return { value: ok(argv[1] === 'mejoras' ? {} : PANEL) }
  })
  return clock
}

const ciclos = (r) => r.llamadas.filter((a) => a[1] === 'flujo' && a[2] === 'propuestas')

test('cada 15 s pide el ciclo de propuestas con --si-toca', { timeoutMs: 30000 }, async ($, on) => {
  const r = { llamadas: [] as string[][] }
  const clock = preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await clock.advance(15000)
  expect(ciclos(r)[0]).toEqual(['qdd', 'flujo', 'propuestas', 'ciclo', '--si-toca', '--json'])
  await clock.advance(15000)
  expect(ciclos(r).length).toBe(2)
})

test('un qdd sin el comando deja de intentarlo', { timeoutMs: 30000 }, async ($, on) => {
  const r = { llamadas: [] as string[][], viejo: true }
  const clock = preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await clock.advance(15000)
  await clock.advance(15000)
  expect(ciclos(r).length).toBe(1)
})

test('sin terminal no corre', { timeoutMs: 30000 }, async ($, on) => {
  const r = { llamadas: [] as string[][] }
  const clock = preparar(on, r)
  await $.session.start({ surface: 'sdk', isInteractive: false, cwd: '/work' })
  await clock.advance(30000)
  expect(ciclos(r).length).toBe(0)
})
