import { expect, mock, test } from 'claude-code/testing'

// Issue #149: la tabla solo muestra el símbolo ⏳; «no veo las preguntas en el tablero». Las preguntas van en «▶ Para ti»
// con su texto, y al tocarlas llegan al chat, donde se responden. La persona descartó una sección con su propio campo.

const PANEL = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Tareas', isFocused: true, bodyColumns: 40, placement: 'dock', scroll: { offset: 0, bodyRows: 30 }, viewportRows: 40 },
} as const

const LARGA = '¿Pruebas tú en Windows? Hace falta un PC real con la versión instalada, porque aquí no se puede.'
const PREGUNTA = { atender: 'atender:barra:1', donde: 'Barra · PR', texto: LARGA }
const MENSAJE = '⏳ PR · Barra (tarea barra): espera ' + LARGA

function preparar(on, r) {
  mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('prompt.submit', ($, e) => {
    r.enviados.push(e.text)
    return { text: e.text }
  })
  on('process.run', ($, e) => {
    r.llamadas.push({ argv: [...e.argv], stdin: e.init?.stdin })
    if (e.argv.includes('binario')) return { value: { exitCode: 1, stdout: '', stderr: '' } }
    if (e.argv.includes('accion')) return { value: r.atiende }
    if (e.argv.includes('responder')) return { value: r.responde }
    if (e.argv.includes('panel')) return { value: { exitCode: 0, stdout: JSON.stringify({ formato: 1, carpeta: '/work', acciones: [], ...(r.esperas ? { esperas_persona: r.esperas } : {}) }), stderr: '' } }
    return { value: { exitCode: 0, stdout: '{}', stderr: '' } }
  })
}

const nuevo = (esperas?: object[]) => ({
  llamadas: [] as { argv: string[]; stdin?: string }[],
  enviados: [] as string[],
  esperas,
  atiende: { exitCode: 0, stdout: MENSAJE + '\n', stderr: '' },
  responde: { exitCode: 0, stdout: JSON.stringify({ aplicado: 'el ⏳ pasa a la IA para que lo verifique', mensaje: 'Respondo a tu ⏳: Ya lo hice.' }) + '\n', stderr: '' },
})
const acciones = (r) => r.llamadas.filter((l) => l.argv.includes('accion'))
const esperar = () => new Promise((res) => setTimeout(res, 20))

async function montado($, on, r) {
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await esperar()
  const ui = await $.ui.mount({ ...PANEL, surface: 'terminal' })
  // Las preguntas quedan plegadas en «? N» (pedido de la persona, 2026-11-08): se abren para verlas.
  if (await ui.find({ key: 'preguntas-ver' })) await ui.press({ key: 'preguntas-ver' })
  return ui
}

test('sin preguntas no se agrega nada a «Para ti», y un qdd que no las trae tampoco', async ($, on) => {
  const ui = await montado($, on, nuevo())
  expect(await ui.find({ type: 'Text', text: /Para ti/ })).toBeUndefined()
  expect(await ui.find({ key: 'espera-0' })).toBeUndefined()
})

test('una pregunta sale en «Para ti», recortada al ancho, con su tarea y etapa; «Te esperan» ya no existe', async ($, on) => {
  const ui = await montado($, on, nuevo([PREGUNTA]))
  expect(await ui.find({ type: 'Text', text: /▶ Para ti/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /1 por responder/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /nada por hacer/ })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: /Te esperan/ })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: 'Barra · PR' })).toBeDefined()
  const fila = await ui.find({ key: 'espera-0' })
  expect(fila.props.label.startsWith('¿Pruebas tú en Windows?')).toBe(true)
})

test('al tocarla, la IA recibe el mensaje de qdd y se responde en el chat: no hay campo aparte', async ($, on) => {
  const r = nuevo([PREGUNTA])
  const ui = await montado($, on, r)
  await ui.press({ key: 'espera-0' })
  await esperar()
  expect(acciones(r)[0].argv).toEqual(['qdd', 'consola', 'accion', 'atender:barra:1'])
  expect(r.enviados).toEqual([MENSAJE])
  expect(await ui.find({ key: 'espera-respuesta' })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: /enviado a la IA/ })).toBeDefined()
})

test('si qdd ya no la ofrece, lo dice y no le envía nada a la IA', async ($, on) => {
  const r = nuevo([PREGUNTA])
  r.atiende = { exitCode: 1, stdout: '', stderr: '[!] esa etapa ya no espera\n' }
  const ui = await montado($, on, r)
  await ui.press({ key: 'espera-0' })
  await esperar()
  expect(r.enviados).toEqual([])
  expect(await ui.find({ type: 'Text', text: /ya no espera/ })).toBeDefined()
})

test('muestra 5 y «ver las N» trae el resto', async ($, on) => {
  const r = nuevo(Array.from({ length: 7 }, (_, i) => ({ atender: 'atender:t' + i + ':0', donde: 'T' + i + ' · PR', texto: '¿Pregunta ' + i + '?' })))
  const ui = await montado($, on, r)
  expect(await ui.find({ key: 'espera-4' })).toBeDefined()
  expect(await ui.find({ key: 'espera-5' })).toBeUndefined()
  await ui.press({ key: 'esperas-todas' })
  expect(await ui.find({ key: 'espera-6' })).toBeDefined()
})

// Botones de los ⏳ (propuesta de qdd-movile aprobada por la persona, 2026-11-07): los mismos tres del teléfono.
const CON_BOTONES = { ...PREGUNTA, pregunta: 'pregunta:espera-abc' }
const respuestas = (r) => r.llamadas.filter((l) => l.argv.includes('responder'))

test('cada ⏳ trae «Ya lo hice», «Todavía no» y «Ya no va»', async ($, on) => {
  const ui = await montado($, on, nuevo([CON_BOTONES]))
  expect((await ui.find({ key: 'espera-hecho-0' })).props.label).toBe('Ya lo hice')
  expect((await ui.find({ key: 'espera-todavia_no-0' })).props.label).toBe('Todavía no')
  expect((await ui.find({ key: 'espera-ya_no_va-0' })).props.label).toBe('Ya no va')
})

test('«Ya lo hice» lo deja en el estado, se lo envía a la IA y dice qué quedó', async ($, on) => {
  const r = nuevo([CON_BOTONES])
  const ui = await montado($, on, r)
  await ui.press({ key: 'espera-hecho-0' })
  await esperar()
  expect(respuestas(r)[0].argv).toEqual(['qdd', 'consola', 'pregunta', 'responder', 'pregunta:espera-abc', 'hecho'])
  expect(r.enviados).toEqual(['Respondo a tu ⏳: Ya lo hice.'])
  expect(await ui.find({ type: 'Text', text: /pasa a la IA para que lo verifique/ })).toBeDefined()
})

test('«Todavía no» no le envía nada a la IA', async ($, on) => {
  const r = nuevo([CON_BOTONES])
  r.responde = { exitCode: 0, stdout: JSON.stringify({ aplicado: 'vuelve a preguntarse en 24 h', mensaje: '' }) + '\n', stderr: '' }
  const ui = await montado($, on, r)
  await ui.press({ key: 'espera-todavia_no-0' })
  await esperar()
  expect(respuestas(r)[0].argv.at(-1)).toBe('todavia_no')
  expect(r.enviados).toEqual([])
  expect(await ui.find({ type: 'Text', text: /vuelve a preguntarse/ })).toBeDefined()
})

test('si qdd no la acepta, lo dice y no envía nada', async ($, on) => {
  const r = nuevo([CON_BOTONES])
  r.responde = { exitCode: 1, stdout: '', stderr: '[!] la pregunta ya no está\n' }
  const ui = await montado($, on, r)
  await ui.press({ key: 'espera-ya_no_va-0' })
  await esperar()
  expect(r.enviados).toEqual([])
  expect(await ui.find({ type: 'Text', text: /ya no está/ })).toBeDefined()
})

test('un qdd que no trae el id de la pregunta no muestra los botones', async ($, on) => {
  const ui = await montado($, on, nuevo([PREGUNTA]))
  expect(await ui.find({ key: 'espera-0' })).toBeDefined()
  expect(await ui.find({ key: 'espera-hecho-0' })).toBeUndefined()
})
