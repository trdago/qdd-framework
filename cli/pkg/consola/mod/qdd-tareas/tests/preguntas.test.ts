import { expect, mock, test } from 'claude-code/testing'

// RFC 0013 §«Preguntas desde el teléfono»: lo que Claude Code le pregunta a la persona (AskUserQuestion o un permiso)
// también se responde desde QDD Móvil. El panel lo registra en qdd y espera al teléfono mientras el diálogo sigue en
// la terminal (el hook PermissionRequest corre a su lado); gana quien responde primero. Un permiso desde el teléfono
// solo se permite una vez o se niega: nunca «siempre» ni reglas.

const PANEL = { formato: 1, estado: null, motivo: 'sin estado', acciones: [] }
const ASK = {
  questions: [
    { question: '¿Qué base usamos?', header: 'Base', multiSelect: false, options: [{ label: 'Postgres', description: '' }, { label: 'SQLite', description: '' }] },
  ],
}
const SUGERENCIAS = [{ type: 'addRules', rules: [{ toolName: 'Bash', ruleContent: 'git push:*' }], behavior: 'allow', destination: 'localSettings' }]

type Salida = { exitCode: number; stdout: string; stderr: string }
const ok = (d: unknown): Salida => ({ exitCode: 0, stdout: typeof d === 'string' ? d : JSON.stringify(d), stderr: '' })

// preparar: 'qdd consola pregunta …' responde lo que diga r (registrar da el id; esperar, lo de r.esperas en orden y
// después «pendiente»); lo demás, el panel.
function preparar(on, r) {
  mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  // Lo de abajo: los hooks de la configuración (r.abajo, si uno decidió) y nada más.
  on('classic.PermissionRequest', () => r.abajo ?? {})
  on('ui.notice', ($, e) => {
    r.notas.push(e)
    return { value: undefined }
  })
  on('process.run', async ($, e) => {
    const argv = [...e.argv]
    r.llamadas.push(argv)
    if (argv[2] !== 'pregunta') return { value: ok(PANEL) }
    if (argv[3] === 'cerrar' && r.cerrarLento) await r.cerrarLento
    if (argv[3] === 'registrar') {
      r.registrados.push(JSON.parse(e.init.stdin))
      return { value: ok({ id: r.id, accion: 'pregunta:' + r.id }) }
    }
    if (argv[3] === 'esperar') {
      r.esperados += 1
      if (r.esperados > 500) return { value: ok({ id: r.id, estado: 'vencida', motivo: 'tope de la prueba' }) }
      return { value: ok(r.esperas.shift() ?? { id: r.id, estado: 'pendiente' }) }
    }
    return { value: ok({ id: r.id, estado: 'respondida', respondida_en: 'pc' }) }
  })
}

const nuevo = (extra = {}) => ({ id: 'p1', llamadas: [] as string[][], registrados: [] as any[], esperas: [] as any[], esperados: 0, notas: [] as any[], ...extra })
const deQDD = (r, sub: string) => r.llamadas.filter((a) => a[2] === 'pregunta' && a[3] === sub)

test('una pregunta de la IA que responde el teléfono llega a Claude Code con sus answers', async ($, on) => {
  const r = nuevo({ esperas: [{ id: 'p1', estado: 'pendiente' }, { id: 'p1', estado: 'respondida', respondida_en: 'telefono', lugar: 'Pixel', answers: { '¿Qué base usamos?': 'SQLite' } }] })
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const res = await $.classic.PermissionRequest({ tool_name: 'AskUserQuestion', tool_input: ASK })
  expect(res.decision).toEqual({ behavior: 'allow', updatedInput: { ...ASK, answers: { '¿Qué base usamos?': 'SQLite' } } })
  expect(r.registrados).toEqual([{ sesion: 'sesion-1', herramienta: 'AskUserQuestion', entrada: ASK }])
  expect(deQDD(r, 'esperar')[0]).toEqual(['qdd', 'consola', 'pregunta', 'esperar', 'p1', '--segundos', '10'])
  expect(r.esperados).toBe(2)
})

test('un permiso desde el teléfono se permite una vez, sin reglas, o se niega diciendo dónde', async ($, on) => {
  const r = nuevo({ esperas: [{ id: 'p1', estado: 'respondida', respondida_en: 'telefono', lugar: 'Pixel', decision: 'permitir' }] })
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const permitido = await $.classic.PermissionRequest({ tool_name: 'Bash', tool_input: { command: 'git push origin v2.9.0' }, permission_suggestions: SUGERENCIAS })
  expect(permitido.decision).toEqual({ behavior: 'allow' })
  expect(JSON.stringify(permitido)).not.toContain('updatedPermissions')

  r.esperas.push({ id: 'p1', estado: 'respondida', respondida_en: 'telefono', lugar: 'Pixel', decision: 'negar' })
  const negado = await $.classic.PermissionRequest({ tool_name: 'Bash', tool_input: { command: 'rm -rf /' } })
  expect(negado.decision).toMatchObject({ behavior: 'deny' })
  expect(negado.decision.message).toContain('Pixel')
})

test('si la terminal responde primero, el panel la cierra en el PC y no decide nada', async ($, on) => {
  const r = nuevo()
  preparar(on, r)
  let soltar: () => void = () => {}
  const diálogo = new Promise<void>((res) => (soltar = res))
  // El motor: la llamada pide permiso (PermissionRequest al lado del diálogo) y termina cuando la persona responde
  // en la terminal.
  let permiso: Promise<any> = Promise.resolve(null)
  const motor = $
  on('tool.call', async (_, e) => {
    permiso = motor.classic.PermissionRequest({ tool_name: 'Bash', tool_input: { command: e.command } })
    await diálogo
    return { result: { stdout: 'ok', stderr: '', interrupted: false } }
  })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const llamada = $.tool.call({ tool: 'Bash', command: 'git push' })
  while (r.esperados < 3) await new Promise((res) => setTimeout(res, 1))
  expect(r.notas[0]?.text).toContain('QDD Móvil')
  soltar()
  await llamada
  const res = await permiso
  expect(res?.decision).toBeUndefined()
  await new Promise((res) => setTimeout(res, 20))
  expect(deQDD(r, 'cerrar')).toEqual([['qdd', 'consola', 'pregunta', 'cerrar', 'p1']])
})

// MEJ-015: la herramienta ya terminó; cerrar su pregunta en el PC no la hace esperar.
test('el final de una herramienta no espera a que qdd cierre su pregunta', async ($, on) => {
  let soltarCierre: () => void = () => {}
  const r = nuevo({ cerrarLento: new Promise<void>((res) => (soltarCierre = res)) })
  preparar(on, r)
  let soltar: () => void = () => {}
  const diálogo = new Promise<void>((res) => (soltar = res))
  const motor = $
  on('tool.call', async (_, e) => {
    motor.classic.PermissionRequest({ tool_name: 'Bash', tool_input: { command: e.command } })
    await diálogo
    return { result: 'hecho' }
  })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const llamada = $.tool.call({ tool: 'Bash', command: 'git push', tool_use_id: 't1' })
  while (r.esperados < 3) await new Promise((res) => setTimeout(res, 1))
  soltar()
  const fin = await Promise.race([llamada, new Promise((res) => setTimeout(() => res('esperó el cierre'), 20))])
  expect(fin).toEqual({ result: 'hecho' })
  soltarCierre()
})

test('sin QDD Móvil, en un subagente o sin terminal no se registra ni se espera nada', async ($, on) => {
  const r = nuevo({ id: '' })
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  expect((await $.classic.PermissionRequest({ tool_name: 'Bash', tool_input: { command: 'ls' } })).decision).toBeUndefined()
  expect(deQDD(r, 'registrar').length).toBe(1)
  expect(deQDD(r, 'esperar').length).toBe(0)

  r.id = 'p1'
  await $.classic.PermissionRequest({ tool_name: 'Bash', tool_input: { command: 'ls' }, agent_id: 'sub-1' })
  expect(deQDD(r, 'registrar').length).toBe(1)
})

test('una sesión sin terminal no pregunta al teléfono', async ($, on) => {
  const r = nuevo()
  preparar(on, r)
  await $.session.start({ surface: null, isInteractive: false, cwd: '/work' })
  await $.classic.PermissionRequest({ tool_name: 'Bash', tool_input: { command: 'ls' } })
  expect(deQDD(r, 'registrar').length).toBe(0)
})

test('un hook de la configuración que ya decidió manda', async ($, on) => {
  const r = nuevo({ abajo: { decision: { behavior: 'deny', message: 'no' } } })
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const res = await $.classic.PermissionRequest({ tool_name: 'Bash', tool_input: { command: 'ls' } })
  expect(res.decision).toEqual({ behavior: 'deny', message: 'no' })
  expect(deQDD(r, 'registrar').length).toBe(0)
})

test('al cerrarse la sesión vencen sus preguntas', async ($, on) => {
  const r = nuevo({ esperas: [{ id: 'p1', estado: 'vencida', motivo: 'x' }] })
  preparar(on, r)
  on('session.end', () => ({ sessionId: 'sesion-1' }))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await $.classic.PermissionRequest({ tool_name: 'Bash', tool_input: { command: 'ls' } })
  await $.session.end({ reason: 'exit' })
  expect(deQDD(r, 'vencer')).toEqual([['qdd', 'consola', 'pregunta', 'vencer', '--sesion', 'sesion-1']])
})
