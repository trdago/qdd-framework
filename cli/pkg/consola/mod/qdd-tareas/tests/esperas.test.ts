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
    if (e.argv.includes('panel')) return { value: { exitCode: 0, stdout: JSON.stringify({ formato: 1, carpeta: '/work', acciones: [], ...(r.esperas ? { esperas_persona: r.esperas } : {}) }), stderr: '' } }
    return { value: { exitCode: 0, stdout: '{}', stderr: '' } }
  })
}

const nuevo = (esperas?: object[]) => ({
  llamadas: [] as { argv: string[]; stdin?: string }[],
  enviados: [] as string[],
  esperas,
  atiende: { exitCode: 0, stdout: MENSAJE + '\n', stderr: '' },
})
const acciones = (r) => r.llamadas.filter((l) => l.argv.includes('accion'))
const esperar = () => new Promise((res) => setTimeout(res, 20))

async function montado($, on, r) {
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await esperar()
  return $.ui.mount({ ...PANEL, surface: 'terminal' })
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
