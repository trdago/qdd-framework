import { expect, mock, test } from 'claude-code/testing'

// Modo goal (RFC 0015): con el modo activo, cada 5 s el panel pide «qdd goal ciclo» con el silencio de la consola. Qdd
// decide todo; el panel envía a la IA el mensaje 🎯 y a la persona el aviso. Solo la persona lo activa.

const ACTIVAR = { id: 'goal-activar', tipo: 'comando', etiqueta: 'Modo goal', detalle: 'qdd goal activar', confirmar: true }
const ESTADO = { actualizado: '2026-11-04', etapas: ['Código'], abreviaturas: ['Có'], frentes: [], objetivos: [], tareas: 0, esperan: 0, fallan: 0, ocultos: 0 }
const panel = (goal, tablero = false) => ({ formato: 1, estado: tablero ? ESTADO : null, motivo: tablero ? '' : 'sin estado', acciones: [ACTIVAR], goal })
const PANE = { plugin: 'qdd-tareas', component: 'Pane', requestId: 'qdd-tareas', viewport: { columns: 160, rows: 40 }, props: { title: 'Tareas' } }
const ACTIVO = { activo: true, estado: 'trabajando', recordatorios: 0, elegidas: [] }
const APAGADO = { activo: false, recordatorios: 0, elegidas: [] }

type Salida = { exitCode: number; stdout: string; stderr: string }
const ok = (d: unknown): Salida => ({ exitCode: 0, stdout: JSON.stringify(d), stderr: '' })

function preparar(on, r) {
  const clock = mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('agent.list', () => ({ value: r.agentes ?? [] }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('prompt.submit', ($, e) => {
    r.enviados.push(e.text)
    return { text: e.text }
  })
  on('ui.notify', ($, e) => {
    r.avisos.push(e.text ?? e.message)
    return { value: { isSent: true } }
  })
  on('process.run', ($, e) => {
    const argv = [...e.argv]
    r.llamadas.push({ argv, env: e.init?.env })
    if (argv[1] === 'goal') {
      if (r.viejo) return { value: { exitCode: 1, stdout: '', stderr: 'Error: unknown command "goal" for "qdd"' } }
      return { value: ok(r.ciclos.shift() ?? {}) }
    }
    if (argv[2] === 'telefono') return { value: ok({ activo: false, entregar: [] }) }
    if (argv[1] === 'mejoras') return { value: ok('') }
    return { value: ok(panel(r.goal, r.tablero)) }
  })
  return clock
}

const nuevo = (extra = {}) => ({ llamadas: [] as any[], enviados: [] as string[], avisos: [] as string[], ciclos: [] as any[], goal: ACTIVO, ...extra })
const ciclos = (r) => r.llamadas.filter((l) => l.argv[1] === 'goal').map((l) => l.argv)
const iniciar = ($) => $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })

test('apagado no gasta un proceso', async ($, on) => {
  const r = nuevo({ goal: APAGADO })
  const clock = preparar(on, r)
  await iniciar($)
  await clock.advance(60000)
  expect(ciclos(r)).toEqual([])
})

test('activo pide el ciclo con el silencio y la sesión', async ($, on) => {
  const r = nuevo()
  const clock = preparar(on, r)
  await iniciar($)
  await clock.advance(10000)
  expect(ciclos(r)[0]).toEqual(['qdd', 'goal', 'ciclo', '--json', '--silencio', '5', '--sesion', 'sesion-1'])
  expect(ciclos(r)[1]).toEqual(['qdd', 'goal', 'ciclo', '--json', '--silencio', '10', '--sesion', 'sesion-1'])
})

test('el mensaje de qdd llega a la IA y el aviso a la persona', async ($, on) => {
  const r = nuevo({ ciclos: [{ mensaje: '🎯 Modo goal: quedan 2 tareas.' }, { aviso: '🎯 Modo goal detenido' }] })
  const clock = preparar(on, r)
  await iniciar($)
  await clock.advance(5000)
  expect(r.enviados).toEqual(['🎯 Modo goal: quedan 2 tareas.'])
  await clock.advance(5000)
  expect(r.avisos.length).toBe(1)
  expect(r.enviados.length).toBe(1)
})

test('un mensaje que no empieza con 🎯 no se envía: podría leerse como un comando', async ($, on) => {
  const r = nuevo({ ciclos: [{ mensaje: '/clear' }, { mensaje: '!rm' }] })
  const clock = preparar(on, r)
  await iniciar($)
  await clock.advance(10000)
  expect(r.enviados).toEqual([])
})

test('con un subagente corriendo la sesión está ocupada', async ($, on) => {
  const r = nuevo({ agentes: [{ id: 'a', status: 'running' }] })
  const clock = preparar(on, r)
  await iniciar($)
  await clock.advance(5000)
  expect(ciclos(r)[0]).toContain('--ocupada')
})

test('escribir o que corra una herramienta reinicia el silencio', async ($, on) => {
  const r = nuevo()
  const clock = preparar(on, r)
  on('tool.call', () => ({ result: 'hecho' }))
  await iniciar($)
  await clock.advance(10000)
  await $.tool.call({ tool: 'Read', file_path: '/work/a' })
  await clock.advance(5000)
  expect(ciclos(r).at(-1)).toContain('5')
  expect(ciclos(r).at(-1)).not.toContain('15')
})

test('sin terminal no vigila', async ($, on) => {
  const r = nuevo()
  const clock = preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: false, cwd: '/work' })
  await clock.advance(30000)
  expect(ciclos(r)).toEqual([])
})

test('terminado deja de preguntar', async ($, on) => {
  const r = nuevo({ goal: { ...ACTIVO, estado: 'terminado' } })
  const clock = preparar(on, r)
  await iniciar($)
  await clock.advance(30000)
  expect(ciclos(r)).toEqual([])
})

test('con un qdd sin «goal» deja de intentarlo', async ($, on) => {
  const r = nuevo({ viejo: true })
  const clock = preparar(on, r)
  await iniciar($)
  await clock.advance(30000)
  expect(ciclos(r).length).toBe(1)
})

test('el botón del panel pide confirmación y solo entonces lo activa, sin CLAUDECODE', async ($, on) => {
  const r = nuevo({ tablero: true, goal: APAGADO })
  preparar(on, r)
  await iniciar($)
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await ui.press({ key: 'b-goal-activar' })
  const accion = () => r.llamadas.filter((l) => l.argv[2] === 'accion')
  expect(accion()).toEqual([])
  await ui.press({ key: 'si' })
  expect(accion()[0].argv).toEqual(['qdd', 'consola', 'accion', 'goal-activar', '--confirmado'])
  expect(accion()[0].env).toEqual({ CLAUDECODE: '' })
})

test('el panel dice en qué está y lo último que eligió', async ($, on) => {
  const r = nuevo({ tablero: true, goal: { ...ACTIVO, estado: 'pausado', motivo: 'sin_avance', elegidas: [{ etiqueta: '¿Cuál?', eleccion: 'Sí (Recommended)', tipo: 'pregunta' }] } })
  preparar(on, r)
  await iniciar($)
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: /Modo goal · en pausa: 3 recordatorios sin avance/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /¿Cuál\? → Sí \(Recommended\)/ })).toBeDefined()
})

// FND-146: PowerShell no pasaba por esta regla, y una ruta entre comillas o qdd.cmd la esquivaban también por Bash.
// Queda la barrera de qdd (CLAUDECODE), pero esta debe verlos.
test('una IA no lo activa por PowerShell ni con una ruta entre comillas (FND-146)', async ($, on) => {
  const r = nuevo({ goal: APAGADO })
  preparar(on, r)
  await iniciar($)
  const comandos = [
    'qdd goal activar',
    '.\\qdd.exe goal activar',
    '& "C:\\x y\\qdd.exe" goal activar',
    "& 'C:\\x y\\qdd.cmd' consola accion goal-activar --confirmado",
    '"/opt/x y/qdd" goal activar',
    'qdd-rc10 goal activar',
    '$env:CLAUDECODE=""; qdd goal activar',
  ]
  for (const tool of ['Bash', 'PowerShell']) {
    for (const command of comandos) {
      const res = await $.tool.call({ tool, command, tool_use_id: 'u1' }).catch((err) => ({ deny: String(err) }))
      expect([tool, command, JSON.stringify(res).includes('solo la persona')]).toEqual([tool, command, true])
    }
  }
})

test('una IA no lo activa por Bash', async ($, on) => {
  const r = nuevo({ goal: APAGADO })
  preparar(on, r)
  await iniciar($)
  for (const command of ['qdd goal activar', 'cd x && qdd consola accion goal-activar --confirmado', 'CLAUDECODE= qdd goal activar']) {
    const res = await $.tool.call({ tool: 'Bash', command, tool_use_id: 'u1' }).catch((err) => ({ deny: String(err) }))
    expect(JSON.stringify(res)).toContain('solo la persona')
  }
})
