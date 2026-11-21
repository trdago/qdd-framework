import { expect, mock, test } from 'claude-code/testing'

// Puertas de QDD, etapa 1 (ficha puertas): antes de un comando de la IA que commitea o publica, el panel le pregunta a
// 'qdd puertas revisar'. Con 2 lo rechaza con el mensaje de qdd; con cualquier otra cosa (o si qdd no responde) pasa.
// Un comando que no nombra nada de eso no gasta un proceso.

const RECHAZO = 'Puerta de QDD: no hay un qdd audit aprobado para este árbol. Corre qdd audit en esta carpeta y commitea sin cambiar nada después.'

function preparar(on, r) {
  const clock = mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work/qdd' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('agent.list', () => ({ value: [] }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('tool.call', () => ({ result: 'corrió' }))
  on('process.run', ($, e) => {
    const argv = [...e.argv]
    if (argv[1] !== 'puertas') return { value: { exitCode: 0, stdout: JSON.stringify({ formato: 1, estado: null, motivo: 'sin estado', acciones: [] }), stderr: '' } }
    r.revisiones.push(argv)
    if (r.cuelga) return new Promise(() => {})
    return { value: r.respuesta }
  })
  return clock
}

const nuevo = (respuesta) => ({ revisiones: [] as string[][], respuesta, cuelga: false })
const iniciar = ($) => $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work/qdd' })
const correr = ($, command, tool = 'Bash') => $.tool.call({ tool, command, tool_use_id: 'u1' }).catch((err) => ({ deny: String(err) }))

test('un commit que la puerta rechaza no corre y la IA lee qué falta', async ($, on) => {
  const r = nuevo({ exitCode: 2, stdout: RECHAZO + '\n', stderr: '' })
  preparar(on, r)
  await iniciar($)
  const res = await correr($, "git add -A && git commit -m 'feat: x'")
  expect(JSON.stringify(res)).toContain('no hay un qdd audit aprobado')
  expect(JSON.stringify(res)).not.toContain('corrió')
  expect(r.revisiones).toEqual([['qdd', 'puertas', 'revisar', '--dir', '/work/qdd', '--sesion', 'sesion-1', '--', "git add -A && git commit -m 'feat: x'"]])
})

test('lo que la puerta deja pasar corre igual', async ($, on) => {
  const r = nuevo({ exitCode: 0, stdout: '', stderr: '' })
  preparar(on, r)
  await iniciar($)
  expect(JSON.stringify(await correr($, 'gh pr merge 3 --merge'))).toContain('corrió')
})

test('un comando que no commitea ni publica no gasta un proceso', async ($, on) => {
  const r = nuevo({ exitCode: 2, stdout: RECHAZO, stderr: '' })
  preparar(on, r)
  await iniciar($)
  for (const command of ['go test ./...', 'git status', 'gh pr view 3', 'npm view qdd-framework version']) {
    expect(JSON.stringify(await correr($, command))).toContain('corrió')
  }
  expect(JSON.stringify(await $.tool.call({ tool: 'Read', file_path: '/x', tool_use_id: 'u2' }))).toContain('corrió')
  expect(r.revisiones).toEqual([])
})

// Invariante: una puerta que no puede decidir deja pasar y nunca bloquea por una falla propia.
test('si qdd falla o no responde a tiempo, el comando pasa', async ($, on) => {
  const r = nuevo({ exitCode: 1, stdout: 'Error: unknown command "puertas" for "qdd"', stderr: '' })
  const clock = preparar(on, r)
  await iniciar($)
  expect(JSON.stringify(await correr($, 'git commit -m x'))).toContain('corrió')
  r.cuelga = true
  const res = correr($, 'git commit -m x')
  await clock.advance(2500)
  expect(JSON.stringify(await res)).toContain('corrió')
})

test('PowerShell también pasa por las puertas', async ($, on) => {
  const r = nuevo({ exitCode: 2, stdout: RECHAZO, stderr: '' })
  preparar(on, r)
  await iniciar($)
  expect(JSON.stringify(await correr($, 'git commit -m x', 'PowerShell'))).toContain('no hay un qdd audit')
})
