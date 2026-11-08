import { expect, mock, test } from 'claude-code/testing'

// Esperas que se responden solas (pedido de la persona, 2026-11-08): cada minuto el panel pide «qdd estado verificar
// --si-toca» y qdd decide qué ⏳ toca revisar. Sin terminal no corre; un qdd sin el comando deja de intentarlo.

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
    if (argv[1] === 'estado' && argv[2] === 'verificar' && r.viejo) return { value: { exitCode: 1, stdout: '', stderr: 'Error: unknown command "verificar" for "qdd estado"' } }
    if (argv[1] === 'estado' && argv[2] === 'verificar') return { value: ok({ revisadas: [], cumplidas: [], avisos: [] }) }
    if (argv[2] === 'telefono') return { value: ok({ activo: false, entregar: [] }) }
    return { value: ok(argv[1] === 'mejoras' ? {} : PANEL) }
  })
  return clock
}

const verificaciones = (r) => r.llamadas.filter((a) => a[1] === 'estado' && a[2] === 'verificar')

test('cada minuto pide verificar con --si-toca', { timeoutMs: 30000 }, async ($, on) => {
  const r = { llamadas: [] as string[][] }
  const clock = preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await clock.advance(60000)
  expect(verificaciones(r)[0]).toEqual(['qdd', 'estado', 'verificar', '--si-toca', '--json'])
  await clock.advance(60000)
  expect(verificaciones(r).length).toBe(2)
})

test('un qdd sin el comando deja de intentarlo', { timeoutMs: 30000 }, async ($, on) => {
  const r = { llamadas: [] as string[][], viejo: true }
  const clock = preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await clock.advance(60000)
  await clock.advance(60000)
  expect(verificaciones(r).length).toBe(1)
})
