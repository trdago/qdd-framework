import { expect, mock, test } from 'claude-code/testing'

// Sprint 8 (MEJ-057, MEJ-026): toda pregunta a la persona, de cualquier tipo, aparece en «▶ Para ti» mientras sigue
// abierta. Lo abierto en la terminal va arriba a la vista (MEJ-057); las acciones (pedidos y fichas) se ven; los ⏳
// cuentan en «? N» y se ven al desplegarlo. Todo cuenta en «N por responder». Una pregunta de la terminal aparece aunque QDD Móvil no esté activado, y aunque qdd no la registre.

const PANEL = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Tareas', isFocused: true, bodyColumns: 60, placement: 'dock', scroll: { offset: 0, bodyRows: 60 } },
  surface: 'terminal',
} as const

const ASK = {
  questions: [{ question: '¿Qué base usamos?', header: 'Base', multiSelect: false, options: [{ label: 'Postgres', description: '' }, { label: 'SQLite', description: '' }] }],
}

// Un ítem de cada tipo, como lo trae 'qdd consola panel --json --con pregunta' (TestParaTi_* en Go fija que lleven turno).
const TIPOS = {
  pedido: { id: 'pedido:1', tipo: 'pedido', etiqueta: 'Mergear el PR 1', detalle: 'gh pr merge 1', tecla: 'a', estado: 'pendiente', confirmar: true, turno: 1 },
  ficha: { id: 'aprobar:pagos', tipo: 'aprobar', etiqueta: 'Aprobar «Pagos»', detalle: 'qdd modules approve pagos', confirmar: true, turno: 2 },
  espera: { id: 'pregunta:espera-1', tipo: 'pregunta', origen: 'estado', estado: 'pendiente', etiqueta: '⏳ Base · Código: ¿Sigo?', detalle: '¿Sigo?', confirmar: true, turno: 3 },
  ia: { id: 'pregunta:ia1', tipo: 'pregunta', origen: 'ia', estado: 'pendiente', etiqueta: '❓ Base · ¿Qué base usamos?', detalle: '¿Qué base usamos?', confirmar: true, turno: 4 },
  permiso: { id: 'pregunta:pe1', tipo: 'pregunta', origen: 'permiso', estado: 'pendiente', etiqueta: '🔐 Bash: git push', detalle: 'git push', confirmar: true, turno: 5 },
  qdd: { id: 'pregunta:qd1', tipo: 'pregunta', origen: 'qdd', estado: 'pendiente', etiqueta: 'qdd incidentes publicar', detalle: '¿Lo publico?', confirmar: true, turno: 6 },
}
const ESPERA_EN_LA_TABLA = { pregunta: TIPOS.espera.id, atender: 'atender:base:0', donde: 'Base · Código', texto: '¿Sigo?' }

type Salida = { exitCode: number; stdout: string; stderr: string }
const ok = (d: unknown): Salida => ({ exitCode: 0, stdout: JSON.stringify(d), stderr: '' })
const pausa = () => new Promise((res) => setTimeout(res, 20))

// preparar: el panel trae r.acciones y r.esperas; 'qdd consola pregunta registrar' responde r.registrar (o falla);
// esperar da lo de r.esperas_pregunta en orden y después «pendiente».
function preparar(on, r) {
  r.reloj = mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('classic.PermissionRequest', () => ({}))
  // El motor: una herramienta pide permiso (PermissionRequest al lado del diálogo) y termina cuando r.soltar() la responde.
  const motor = $raiz
  on('tool.call', async (_, e) => {
    const dialogo = new Promise<void>((res) => (r.soltar = res))
    if (!r.sinPermiso) motor.$.classic.PermissionRequest({ tool_name: e.tool, tool_input: entradaDe(e) })
    await dialogo
    return { result: 'hecho' }
  })
  on('ui.notice', ($, e) => {
    r.notas.push(e)
    return { value: undefined }
  })
  on('process.run', async ($, e) => {
    const argv = [...e.argv]
    r.llamadas.push(argv)
    if (argv.includes('binario')) return { value: { exitCode: 1, stdout: '', stderr: '' } }
    if (argv[2] === 'panel') return { value: ok({ formato: 1, carpeta: '/work', estado: null, motivo: 'x', acciones: r.acciones, esperas_persona: r.esperas }) }
    if (argv[2] !== 'pregunta') return { value: ok({}) }
    if (argv[3] === 'registrar') {
      r.registros.push(JSON.parse(e.init?.stdin ?? '{}'))
      if (!r.registrar) return { value: { exitCode: 1, stdout: '', stderr: '[!] la cola de preguntas está tomada' } }
      return { value: ok(r.registrar) }
    }
    if (argv[3] === 'esperar') {
      r.esperados += 1
      if (r.esperados > 500) return { value: ok({ id: 'p1', estado: 'vencida', motivo: 'tope de la prueba' }) }
      return { value: ok(r.esperas_pregunta.shift() ?? { id: 'p1', estado: 'pendiente' }) }
    }
    return { value: ok({ id: 'p1', estado: 'respondida', respondida_en: 'pc' }) }
  })
}

const RESERVADAS = ['tool', 'tool_use_id', 'agentId', 'requestMeta', 'consent']
const entradaDe = (e) => Object.fromEntries(Object.entries(e).filter(([k]) => !RESERVADAS.includes(k)))
const $raiz: { $: any } = { $: null }
const nuevo = (extra = {}) => ({ acciones: [] as any[], esperas: [] as any[], registrar: { id: 'p1', accion: 'pregunta:p1' } as any, esperas_pregunta: [] as any[], esperados: 0, soltar: null as null | (() => void), llamadas: [] as string[][], notas: [] as any[], registros: [] as any[], ...extra })

async function montado($, on, r) {
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await pausa()
  return $.ui.mount({ ...PANEL })
}

// unDialogo abre una herramienta que pide permiso y queda abierta hasta cerrar().
function unDialogo($, r, herramienta: string, entrada: any) {
  $raiz.$ = $
  const llamada = $.tool.call({ tool: herramienta, ...entrada, tool_use_id: 't1' })
  return {
    cerrar: async () => {
      while (!r.soltar) await new Promise((res) => setTimeout(res, 1))
      r.soltar()
      await llamada
      await pausa()
    },
  }
}

test('cada tipo de pregunta aparece en «Para ti»: lo de la terminal y las acciones a la vista, los ⏳ en «? N»', async ($, on) => {
  const r = nuevo({ acciones: Object.values(TIPOS), esperas: [ESPERA_EN_LA_TABLA] })
  const ui = await montado($, on, r)
  expect(await ui.find({ key: 'b-' + TIPOS.pedido.id })).toBeDefined()
  expect(await ui.find({ key: 'b-' + TIPOS.ficha.id })).toBeDefined()
  for (const a of [TIPOS.ia, TIPOS.permiso, TIPOS.qdd]) {
    expect(await ui.find({ type: 'Text', text: new RegExp('#' + a.turno + ' ') })).toBeDefined()
  }
  expect(await ui.find({ type: 'Text', text: /4 por responder/ })).toBeDefined()
  expect((await ui.find({ key: 'preguntas-ver' })).props.label).toContain('? 1')
  expect(await ui.find({ key: 'espera-0' })).toBeDefined()
})

for (const tipo of ['espera', 'ia', 'permiso', 'qdd']) {
  test('una sola pregunta (' + tipo + ') ya enciende «Para ti» y dice cuántas hay por responder', async ($, on) => {
    const r = nuevo({ acciones: [TIPOS[tipo]], esperas: tipo === 'espera' ? [ESPERA_EN_LA_TABLA] : [] })
    const ui = await montado($, on, r)
    if (tipo === 'espera') expect((await ui.find({ key: 'preguntas-ver' }))?.props.label).toContain('? 1')
    if (tipo !== 'espera') expect(await ui.find({ type: 'Text', text: new RegExp('#' + TIPOS[tipo].turno + ' ') })).toBeDefined()
    expect(await ui.find({ type: 'Text', text: /1 por responder/ })).toBeDefined()
  })
}

// La persona no veía ninguna de 30 preguntas (2026-11-25): todas se ven, sin tope ni «ver las N» (el modo goal eligió
// la opción recomendada «Desplegadas, sin tope», 2026-11-26).
test('muchas ⏳: «? N» las cuenta todas y se ven todas, sin tope', async ($, on) => {
  const esperas = Array.from({ length: 30 }, (_, i) => ({ pregunta: 'pregunta:e' + i, atender: 'atender:t:' + i, donde: 'T · ' + i, texto: '¿' + i + '?' }))
  const acciones = esperas.map((q, i) => ({ ...TIPOS.espera, id: q.pregunta, turno: 10 + i }))
  const ui = await montado($, on, nuevo({ acciones, esperas }))
  expect((await ui.find({ key: 'preguntas-ver' })).props.label).toContain('? 30')
  expect(await ui.find({ key: 'espera-29' })).toBeDefined()
  expect(await ui.find({ key: 'esperas-todas' })).toBeUndefined()
})

test('sin QDD Móvil la pregunta se registra y su sesión la sigue esperando, sin ofrecer el teléfono (MEJ-057)', async ($, on) => {
  const r = nuevo({ registrar: { id: 'p1', accion: 'pregunta:p1', telefono: false } })
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const d = unDialogo($, r, 'Bash', { command: 'git push' })
  while (r.esperados < 3) await new Promise((res) => setTimeout(res, 1))
  expect(r.llamadas.filter((a) => a[3] === 'registrar').length).toBe(1)
  expect(r.notas.filter((n) => /QDD Móvil/.test(n.text ?? ''))).toEqual([])
  await d.cerrar()
  expect(r.llamadas.filter((a) => a[3] === 'cerrar')).toEqual([['qdd', 'consola', 'pregunta', 'cerrar', 'p1']])
})

test('con QDD Móvil avisa que también se responde desde el teléfono', async ($, on) => {
  const r = nuevo({ registrar: { id: 'p1', accion: 'pregunta:p1', telefono: true } })
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const d = unDialogo($, r, 'Bash', { command: 'git push' })
  while (r.esperados < 1) await new Promise((res) => setTimeout(res, 1))
  expect(r.notas[0]?.text).toContain('QDD Móvil')
  await d.cerrar()
})

test('si qdd no registra la pregunta, «Para ti» igual la cuenta en la terminal hasta que se responde', async ($, on) => {
  const r = nuevo({ registrar: null })
  const ui = await montado($, on, r)
  expect(await ui.find({ key: 'preguntas-ver' })).toBeUndefined()
  const d = unDialogo($, r, 'AskUserQuestion', ASK)
  await pausa()
  expect(await ui.find({ type: 'Text', text: /Base · ¿Qué base usamos\?/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /respóndela en la terminal/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /1 por responder/ })).toBeDefined()
  await d.cerrar()
  expect(await ui.find({ type: 'Text', text: /Base · ¿Qué base usamos\?/ })).toBeUndefined()
})

test('un qdd anterior que sin QDD Móvil no registra (id vacío) también deja la pregunta en «Para ti»', async ($, on) => {
  const r = nuevo({ registrar: { id: '', motivo: 'QDD Móvil no está activado en este PC (qdd remoto activar)' } })
  const ui = await montado($, on, r)
  const d = unDialogo($, r, 'Bash', { command: 'git push origin main' })
  await pausa()
  expect(await ui.find({ type: 'Text', text: /Bash: git push origin main/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /respóndela en la terminal/ })).toBeDefined()
  await d.cerrar()
})

// MEJ-094 (REP-510): Claude Code no garantiza el hook PermissionRequest para AskUserQuestion (anthropics/claude-code
// #15400, #59908) y en Windows corre con otras reglas. Sin él, qdd no registraba nada: la pregunta no salía en el panel
// ni en el teléfono, y la sesión estuvo 3 horas esperando sin que nadie lo supiera.
const cerradas = (r) => r.llamadas.filter((a) => a[3] === 'cerrar').map((a) => a[4])

test('si Claude Code no pide el permiso de AskUserQuestion, al rato qdd la registra para responder en el PC (MEJ-094)', async ($, on) => {
  const r = nuevo({ sinPermiso: true })
  await montado($, on, r)
  const d = unDialogo($, r, 'AskUserQuestion', ASK)
  await pausa()
  expect(r.registros).toEqual([])
  await r.reloj.advance(6_000)
  await pausa()
  expect(r.registros.length).toBe(1)
  expect(r.registros[0]).toMatchObject({ herramienta: 'AskUserQuestion', entrada: ASK, responder_en: 'pc' })
  expect(r.llamadas.filter((a) => a[3] === 'esperar')).toEqual([])
  await d.cerrar()
  expect(cerradas(r)).toEqual(['p1'])
})

test('sin permiso y sin que qdd la registre, igual sale en «Para ti» (MEJ-094)', async ($, on) => {
  const r = nuevo({ sinPermiso: true, registrar: null })
  const ui = await montado($, on, r)
  const d = unDialogo($, r, 'AskUserQuestion', ASK)
  await r.reloj.advance(6_000)
  await pausa()
  expect(await ui.find({ type: 'Text', text: /Base · ¿Qué base usamos\?/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /respóndela en la terminal/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /1 por responder/ })).toBeDefined()
  await d.cerrar()
  expect(await ui.find({ type: 'Text', text: /Base · ¿Qué base usamos\?/ })).toBeUndefined()
})

test('una pregunta que solo se responde en el PC no ofrece botones en «Para ti» (MEJ-094)', async ($, on) => {
  const r = nuevo({ acciones: [{ ...TIPOS.ia, responder_en: 'pc', preguntas: [{ id: 'q1', texto: '¿Qué base usamos?', opciones: [{ valor: 'a', etiqueta: 'Postgres' }] }] }] })
  const ui = await montado($, on, r)
  expect(await ui.find({ type: 'Text', text: /respóndela en la terminal/ })).toBeDefined()
  expect(await ui.find({ key: 'pq-' + TIPOS.ia.id + '-a' })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: /1 por responder/ })).toBeDefined()
})

test('con el permiso pedido a tiempo, el respaldo no registra otra (MEJ-094)', async ($, on) => {
  const r = nuevo()
  await montado($, on, r)
  const d = unDialogo($, r, 'AskUserQuestion', ASK)
  await r.reloj.advance(6_000)
  await pausa()
  expect(r.registros.length).toBe(1)
  expect(r.registros[0].responder_en).toBeUndefined()
  await d.cerrar()
})

test('una herramienta que no es AskUserQuestion sin permiso no es una pregunta (MEJ-094)', async ($, on) => {
  const r = nuevo({ sinPermiso: true })
  const ui = await montado($, on, r)
  const d = unDialogo($, r, 'Bash', { command: 'make test' })
  await r.reloj.advance(6_000)
  await pausa()
  expect(r.registros).toEqual([])
  expect(await ui.find({ type: 'Text', text: /por responder/ })).toBeUndefined()
  await d.cerrar()
})

test('si el permiso llega después del respaldo, cierra la del respaldo y registra la que se responde desde el teléfono (MEJ-094)', async ($, on) => {
  const r = nuevo({ sinPermiso: true })
  await montado($, on, r)
  const d = unDialogo($, r, 'AskUserQuestion', ASK)
  await r.reloj.advance(6_000)
  await pausa()
  expect(r.registros.map((x) => x.responder_en)).toEqual(['pc'])
  $.classic.PermissionRequest({ tool_name: 'AskUserQuestion', tool_input: ASK })
  while (r.esperados < 1) await new Promise((res) => setTimeout(res, 1))
  expect(cerradas(r)).toEqual(['p1'])
  expect(r.registros.map((x) => x.responder_en)).toEqual(['pc', undefined])
  await d.cerrar()
})

test('sin registrar, si el permiso llega después del respaldo, la pregunta no queda dos veces (MEJ-094)', async ($, on) => {
  const r = nuevo({ sinPermiso: true, registrar: null })
  const ui = await montado($, on, r)
  const d = unDialogo($, r, 'AskUserQuestion', ASK)
  await r.reloj.advance(6_000)
  await pausa()
  expect(await ui.find({ type: 'Text', text: /1 por responder/ })).toBeDefined()
  $.classic.PermissionRequest({ tool_name: 'AskUserQuestion', tool_input: ASK })
  await pausa()
  await r.reloj.advance(6_000)
  await pausa()
  expect(await ui.find({ type: 'Text', text: /1 por responder/ })).toBeDefined()
  await d.cerrar()
  expect(await ui.find({ type: 'Text', text: /por responder/ })).toBeUndefined()
})
