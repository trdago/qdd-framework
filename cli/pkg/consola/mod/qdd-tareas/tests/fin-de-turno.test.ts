import { expect, mock, test } from 'claude-code/testing'

// MEJ-026 (issue #150): lo que necesita a la persona llega siempre al panel y al teléfono; qdd lo exige en vez de confiar
// en la IA. Al terminar un turno de la conversación principal, el panel le pasa a qdd el último mensaje de la IA y las
// herramientas que llamó (qdd consola fin-de-turno) y, si qdd dice que falta algo, le devuelve el turno una vez.

const PANEL = JSON.stringify({ formato: 1, estado: null, motivo: '', acciones: [] })
const FALTA = 'Terminaste con una pregunta para la persona («¿Parto?») que no quedó en el panel.'
const PRESUPUESTO_MS = 400

const ok = (stdout: string) => ({ exitCode: 0, stdout, stderr: '' })
const pausa = () => new Promise((res) => setTimeout(res, 20))

function preparar(on, r) {
  const clock = mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('tool.call', () => ({ result: 'hecho' }))
  on('classic.Stop', () => ({}))
  on('turn.start', ($, e) => ({ turnId: e.turnId }))
  on('ui.log', () => ({ value: undefined }))
  on('process.run', async ($, e) => {
    if (e.argv[2] !== 'fin-de-turno') return { value: ok(PANEL) }
    r.turnos.push(JSON.parse(e.init.stdin))
    if (r.retener) await r.retener
    return { value: ok(JSON.stringify({ devolver: r.devolver })) }
  })
  return clock
}

const nuevo = (): any => ({ turnos: [] as any[], devolver: FALTA, retener: null })

test('una pregunta que quedó solo en el chat devuelve el turno con lo que dice qdd', async ($, on) => {
  const r = nuevo()
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await $.turn.start({ turnId: 't1' })
  await $.tool.call({ tool: 'Read', file_path: '/work/a', tool_use_id: 'u1' })
  await $.tool.call({ tool: 'mcp__qdd__qdd_estado', accion: 'set', estado: 'ok', tool_use_id: 'u2' })
  await $.tool.call({ tool: 'Read', file_path: '/work/b', tool_use_id: 'u3', agentId: 'sub-1' })
  const res = await $.classic.Stop({ stop_hook_active: false, last_assistant_message: '¿Parto?' })
  expect(res.block).toBe(FALTA)
  expect(r.turnos).toHaveLength(1)
  const t = r.turnos[0]
  expect(t.mensaje).toBe('¿Parto?')
  expect(t.ya_devuelto).toBe(false)
  expect(t.herramientas.map((h) => h.tool)).toEqual(['Read', 'mcp__qdd__qdd_estado'])
  expect(t.herramientas[0].entrada).toBeUndefined()
  expect(t.herramientas[1].entrada).toMatchObject({ accion: 'set', estado: 'ok' })
})

test('el turno siguiente empieza sin las herramientas del anterior', async ($, on) => {
  const r = nuevo()
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await $.turn.start({ turnId: 't1' })
  await $.tool.call({ tool: 'AskUserQuestion', questions: [], tool_use_id: 'u1' })
  await $.classic.Stop({ stop_hook_active: false, last_assistant_message: 'Listo.' })
  await $.turn.start({ turnId: 't2' })
  await $.classic.Stop({ stop_hook_active: false, last_assistant_message: '¿Parto?' })
  expect(r.turnos[1].herramientas).toEqual([])
})

test('nada que dejar, ya devuelto, un subagente o sin terminal: termina como siempre', async ($, on) => {
  const r = nuevo()
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  r.devolver = ''
  expect((await $.classic.Stop({ stop_hook_active: false, last_assistant_message: 'Listo.' })).block).toBeUndefined()
  r.devolver = FALTA
  expect((await $.classic.Stop({ stop_hook_active: true, last_assistant_message: '¿Parto?' })).block).toBeUndefined()
  expect((await $.classic.Stop({ stop_hook_active: false, last_assistant_message: '¿Parto?', agent_id: 'sub-1' })).block).toBeUndefined()
  expect(r.turnos).toHaveLength(1)
})

test('sin terminal (claude -p) no hay persona mirando: no se pregunta a qdd', async ($, on) => {
  const r = nuevo()
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: false, cwd: '/work' })
  expect((await $.classic.Stop({ stop_hook_active: false, last_assistant_message: '¿Parto?' })).block).toBeUndefined()
  expect(r.turnos).toHaveLength(0)
})

test('un qdd lento no retiene el final del turno más que el presupuesto', async ($, on) => {
  const r = nuevo()
  const clock = preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  r.retener = new Promise(() => {})
  let listo = false
  const fin = $.classic.Stop({ stop_hook_active: false, last_assistant_message: '¿Parto?' }).then((x) => {
    listo = true
    return x
  })
  await pausa()
  await clock.advance(PRESUPUESTO_MS)
  await pausa()
  expect(listo).toBe(true)
  expect((await fin).block).toBeUndefined()
})
