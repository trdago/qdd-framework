import { expect, mock, test } from 'claude-code/testing'

// Que una pregunta no vuelva a dejar la consola entrampada (después de FND-165): un diálogo de la terminal
// (AskUserQuestion o un permiso) que sigue abierto al minuto avisa una vez en el escritorio, con el proyecto y la
// pregunta. Uno que la persona ya respondió, en la terminal o en el teléfono, no avisa.

const PANEL = { formato: 1, estado: null, motivo: 'sin estado', acciones: [] }
const ASK = {
  questions: [
    { question: '¿Qué base usamos?', header: 'Base', multiSelect: false, options: [{ label: 'Postgres', description: '' }, { label: 'SQLite', description: '' }] },
  ],
}

type Salida = { exitCode: number; stdout: string; stderr: string }
const ok = (d: unknown): Salida => ({ exitCode: 0, stdout: JSON.stringify(d), stderr: '' })

function preparar($$, on, r) {
  // Un reloj propio, sin los ciclos del panel (latido, lecturas, teléfono…): el aviso es un solo sleep, y un minuto
  // simulado con ciclos tarda segundos reales.
  const clock = { ahora: 0, dormidos: [] as { hasta: number; listo: () => void }[] }
  const avanzar = async (ms: number) => {
    clock.ahora += ms
    const listos = clock.dormidos.filter((d) => d.hasta <= clock.ahora)
    clock.dormidos = clock.dormidos.filter((d) => d.hasta > clock.ahora)
    for (const d of listos) d.listo()
    await new Promise((res) => setTimeout(res, 20))
  }
  on('clock.every', () => ({ value: undefined }))
  on('clock.now', () => ({ value: clock.ahora }))
  on('clock.sleep', ($, e) => {
    return new Promise((listo) => clock.dormidos.push({ hasta: clock.ahora + Number(e.ms), listo: () => listo({ value: undefined }) }))
  })
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/home/ana/proyectos/tienda' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: false } })) // sin panel abierto: el aviso no depende de él, y la prueba no lo redibuja
  on('classic.PermissionRequest', () => ({}))
  on('ui.notice', () => ({ value: undefined }))
  on('ui.log', () => ({ value: undefined }))
  // El motor: la llamada pide permiso (PermissionRequest al lado del diálogo) y queda abierta hasta soltar().
  const motor = $$
  on('tool.call', async (_, e) => {
    motor.classic.PermissionRequest({ tool_name: e.tool, tool_input: e.tool === 'Bash' ? { command: e.command } : { questions: e.questions } })
    await r.abierto
    return { result: 'hecho' }
  })
  on('ui.notify', ($, e) => {
    r.avisos.push(e)
    return { value: { isSent: true } }
  })
  on('process.run', async ($, e) => {
    const argv = [...e.argv]
    if (argv[2] !== 'pregunta') return { value: ok(PANEL) }
    if (argv[3] === 'registrar') {
      r.registrados += 1
      if (!r.id) return { value: { exitCode: 1, stdout: '', stderr: 'cola tomada' } }
      return { value: ok({ id: r.id, accion: 'pregunta:' + r.id }) }
    }
    if (argv[3] === 'esperar') {
      r.esperados += 1
      if (r.esperados > 3) return { value: ok({ id: r.id, estado: 'vencida', motivo: 'tope de la prueba' }) }
      return { value: ok(r.esperas.shift() ?? { id: r.id, estado: 'pendiente' }) }
    }
    return { value: ok({ id: r.id, estado: 'respondida', respondida_en: 'pc' }) }
  })
  return { advance: avanzar }
}

const nuevo = (extra = {}) => ({ id: 'p1', registrados: 0, esperados: 0, esperas: [] as any[], avisos: [] as any[], ...extra })
const hasta = async (cond: () => boolean) => {
  for (let i = 0; i < 500 && !cond(); i++) await new Promise((res) => setTimeout(res, 1))
}

// dialogo: una llamada de la conversación principal que pide permiso y queda abierta hasta soltar().
function dialogo($, r, entrada: { tool: string; [k: string]: unknown }) {
  let soltar: () => void = () => {}
  r.abierto = new Promise<void>((res) => (soltar = res))
  const llamada = $.tool.call({ ...entrada, tool_use_id: 't1' })
  return { soltar, llamada }
}

test('un AskUserQuestion sin responder al minuto avisa una vez en el escritorio, con el proyecto y la pregunta', async ($, on) => {
  const r = nuevo()
  const clock = preparar($, on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/home/ana/proyectos/tienda' })
  const d = dialogo($, r, { tool: 'AskUserQuestion', ...ASK })
  try {
    await hasta(() => r.registrados > 0)
    for (let i = 0; i < 11; i++) await clock.advance(5_000)
    expect(r.avisos).toEqual([])
    await clock.advance(5_000)
    await hasta(() => r.avisos.length > 0)
    expect(r.avisos.length).toBe(1)
    expect(r.avisos[0]).toMatchObject({ title: 'qdd · tienda', text: expect.stringContaining('¿Qué base usamos?') })
    await clock.advance(5_000)
    expect(r.avisos.length).toBe(1)
  } finally {
    d.soltar()
    await d.llamada
  }
})

test('un permiso que la persona responde en la terminal antes del minuto no avisa', async ($, on) => {
  const r = nuevo()
  const clock = preparar($, on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/w' })
  const d = dialogo($, r, { tool: 'Bash', command: 'git push' })
  await hasta(() => r.registrados > 0)
  await clock.advance(5_000)
  d.soltar()
  await d.llamada
  await clock.advance(56_000)
  expect(r.avisos).toEqual([])
})

test('una pregunta respondida desde el teléfono no avisa', async ($, on) => {
  const r = nuevo({ esperas: [{ id: 'p1', estado: 'respondida', respondida_en: 'telefono', lugar: 'Pixel', answers: { '¿Qué base usamos?': 'SQLite' } }] })
  const clock = preparar($, on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/w' })
  const res = await $.classic.PermissionRequest({ tool_name: 'AskUserQuestion', tool_input: ASK })
  expect(res.decision?.behavior).toBe('allow')
  await clock.advance(61_000)
  expect(r.avisos).toEqual([])
})

test('lo que qdd no alcanzó a registrar también avisa si sigue abierto', async ($, on) => {
  const r = nuevo({ id: '' })
  const clock = preparar($, on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/w' })
  const d = dialogo($, r, { tool: 'Bash', command: 'rm -rf build' })
  await hasta(() => r.registrados > 0)
  await clock.advance(60_000)
  await hasta(() => r.avisos.length > 0)
  expect(r.avisos.length).toBe(1)
  expect(r.avisos[0]).toMatchObject({ text: expect.stringContaining('rm -rf build') })
  d.soltar()
  await d.llamada
})

test('lo que qdd no registró y la terminal ya respondió no avisa', async ($, on) => {
  const r = nuevo({ id: '' })
  const clock = preparar($, on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/w' })
  const d = dialogo($, r, { tool: 'Bash', command: 'ls' })
  await hasta(() => r.registrados > 0)
  d.soltar()
  await d.llamada
  await clock.advance(60_000)
  expect(r.avisos).toEqual([])
})
