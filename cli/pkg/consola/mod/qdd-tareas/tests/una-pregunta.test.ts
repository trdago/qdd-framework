import { expect, mock, test } from 'claude-code/testing'

// MEJ-104 (REP-822): un AskUserQuestion con 2 o más preguntas termina en la pantalla «Review your answers / Submit
// answers» de Claude Code. En Windows quedó ahí con la persona lejos: el teléfono decía «esperando consola» y la IA
// estuvo detenida hasta que volvió al PC. Con una pregunta por llamada esa pantalla no aparece, cada pregunta tiene sus
// botones en «▶ Para ti» y el teléfono la responde sola. El mod rechaza el diálogo de varias antes de mostrarlo y le dice
// a la IA que las haga de a una (elegido por el modo goal, opción recomendada, 2026-11-26).

type Salida = { exitCode: number; stdout: string; stderr: string }
const ok = (d: unknown): Salida => ({ exitCode: 0, stdout: JSON.stringify(d), stderr: '' })

const pregunta = (texto: string) => ({ question: texto, header: 'Base', multiSelect: false, options: [{ label: 'Sí', description: '' }, { label: 'No', description: '' }] })

function preparar(on, r) {
  mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('classic.PermissionRequest', () => ({}))
  on('tool.call', (_, e) => {
    r.mostradas.push(e.questions?.length ?? 0)
    return { result: 'hecho' }
  })
  on('process.run', () => ({ value: ok({ formato: 1, estado: null, motivo: 'sin estado', acciones: [] }) }))
}

const iniciar = ($) => $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
const preguntar = ($, questions, extra = {}) => $.tool.call({ tool: 'AskUserQuestion', questions, tool_use_id: 'u1', ...extra }).catch((err) => ({ deny: String(err) }))

test('un diálogo con varias preguntas no se muestra: la IA las hace de a una', async ($, on) => {
  const r = { mostradas: [] as number[] }
  preparar(on, r)
  await iniciar($)
  const res = await preguntar($, [pregunta('¿Postgres?'), pregunta('¿Con réplicas?')])
  expect(JSON.stringify(res)).toContain('una pregunta por llamada')
  expect(r.mostradas).toEqual([])
})

test('una pregunta sola sigue igual', async ($, on) => {
  const r = { mostradas: [] as number[] }
  preparar(on, r)
  await iniciar($)
  const res = await preguntar($, [pregunta('¿Postgres?')])
  expect(JSON.stringify(res)).not.toContain('una pregunta por llamada')
  expect(r.mostradas).toEqual([1])
})

test('también en un subagente', async ($, on) => {
  const r = { mostradas: [] as number[] }
  preparar(on, r)
  await iniciar($)
  const res = await preguntar($, [pregunta('¿Postgres?'), pregunta('¿Con réplicas?')], { agentId: 'a1' })
  expect(JSON.stringify(res)).toContain('una pregunta por llamada')
  expect(r.mostradas).toEqual([])
})
