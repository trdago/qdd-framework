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

// FND-318: Claude Code quita del comando el «cd <carpeta actual> &&» inicial y la sesión pudo arrancar en otro checkout:
// la puerta mira la carpeta en que el comando va a correr, no la de arranque.
// contrato: puertas#91d6f84a
test('la puerta pregunta por la carpeta actual de la sesión, no por la de arranque', async ($, on) => {
  const r = nuevo({ exitCode: 2, stdout: RECHAZO, stderr: '' })
  preparar(on, r)
  on('session.cwd', () => ({ value: '/work/qdd/.claude/worktrees/otra' }))
  await iniciar($)
  expect(JSON.stringify(await correr($, "git commit -m 'feat: x'"))).not.toContain('corrió')
  expect(r.revisiones[0].slice(1, 5)).toEqual(['puertas', 'revisar', '--dir', '/work/qdd/.claude/worktrees/otra'])
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
// contrato: puertas#91d6f84a
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

// Puerta ficha (etapa 2): antes de editar, el panel le pasa a qdd el archivo y las rutas cuyo contexto ya pidió la sesión.
test('antes de editar pregunta con el archivo y lo que la sesión ya leyó', async ($, on) => {
  const r = nuevo({ exitCode: 0, stdout: '', stderr: '' })
  preparar(on, r)
  await iniciar($)
  await $.tool.call({ tool: 'mcp__qdd__qdd_project_context', paths: 'cli/pkg/baul, cli/cmd/baul.go\ncli/pkg/puertas', tool_use_id: 'u1' })
  await correr($, "cd cli && qdd modules context pkg/estado 'cmd/estado.go' --completo && go test ./...")
  await $.tool.call({ tool: 'Edit', file_path: '/work/qdd/cli/pkg/baul/verificar.go', old_string: 'a', new_string: 'b', tool_use_id: 'u2' })
  const editar = r.revisiones.find((a) => a.includes('--editar'))
  expect(editar).toEqual(['qdd', 'puertas', 'revisar', '--dir', '/work/qdd', '--sesion', 'sesion-1', '--editar', '/work/qdd/cli/pkg/baul/verificar.go',
    '--leida', 'cli/pkg/baul', '--leida', 'cli/cmd/baul.go', '--leida', 'cli/pkg/puertas', '--leida', 'pkg/estado', '--leida', 'cmd/estado.go'])
})

test('una edición que la puerta ficha rechaza no se hace', async ($, on) => {
  const r = nuevo({ exitCode: 2, stdout: 'Puerta de QDD: antes de editar cli/pkg/baul/verificar.go lee su ficha «baul»', stderr: '' })
  preparar(on, r)
  await iniciar($)
  for (const tool of ['Edit', 'Write', 'MultiEdit']) {
    const res = await $.tool.call({ tool, file_path: '/work/qdd/cli/pkg/baul/verificar.go', tool_use_id: 'u3' }).catch((err) => ({ deny: String(err) }))
    expect(JSON.stringify(res)).toContain('lee su ficha')
  }
  const nb = await $.tool.call({ tool: 'NotebookEdit', notebook_path: '/work/qdd/x.ipynb', tool_use_id: 'u4' }).catch((err) => ({ deny: String(err) }))
  expect(JSON.stringify(nb)).toContain('lee su ficha')
})
