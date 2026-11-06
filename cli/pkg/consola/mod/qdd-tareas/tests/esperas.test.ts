import { expect, mock, test } from 'claude-code/testing'

// Issue #149: la tabla solo muestra el símbolo ⏳; «no veo las preguntas en el tablero». El panel muestra su texto
// recortado, lo abre entero al tocarlo y lo responde desde ahí, como QDD Móvil.

const PANEL = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Tareas', isFocused: true, bodyColumns: 40, placement: 'dock', scroll: { offset: 0, bodyRows: 30 }, viewportRows: 40 },
} as const

const LARGA = '¿Pruebas tú en Windows? Hace falta un PC real con la versión instalada, porque aquí no se puede.'
const PREGUNTA = { id: 'espera-a1', donde: 'Barra · PR', texto: LARGA }
const NOTA = { donde: 'Barra · Código', texto: 'Lo pruebas cuando puedas' }
const MENSAJE = 'Respondo a tu ⏳ «Barra · PR»:\n\nsí\n\nCuando lo resuelvas…'

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
    if (e.argv.includes('responder')) return { value: r.responde }
    if (e.argv.includes('panel')) return { value: { exitCode: 0, stdout: JSON.stringify({ formato: 1, carpeta: '/work', acciones: [], ...(r.esperas ? { esperas_persona: r.esperas } : {}) }), stderr: '' } }
    return { value: { exitCode: 0, stdout: '{}', stderr: '' } }
  })
}

const nuevo = (esperas?: object[]) => ({
  llamadas: [] as { argv: string[]; stdin?: string }[],
  enviados: [] as string[],
  esperas,
  responde: { exitCode: 0, stdout: JSON.stringify({ mensaje: MENSAJE }), stderr: '' },
})
const respuestas = (r) => r.llamadas.filter((l) => l.argv.includes('responder'))
const esperar = () => new Promise((res) => setTimeout(res, 20))

async function montado($, on, r) {
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await esperar()
  return $.ui.mount({ ...PANEL, surface: 'terminal' })
}

test('sin ⏳ de la persona no hay sección, y un qdd que no los trae tampoco', async ($, on) => {
  const ui = await montado($, on, nuevo())
  expect(await ui.find({ type: 'Text', text: /Te esperan/ })).toBeUndefined()
  expect(await ui.find({ key: 'espera-0' })).toBeUndefined()
})

test('un ⏳ se lee recortado al ancho, con su tarea y etapa', async ($, on) => {
  const ui = await montado($, on, nuevo([PREGUNTA]))
  expect(await ui.find({ type: 'Text', text: /⏳ Te esperan/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: 'Barra · PR' })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: LARGA })).toBeUndefined()
  const fila = await ui.find({ key: 'espera-0' })
  expect(fila.props.label.startsWith('¿Pruebas tú en Windows?')).toBe(true)
})

test('al tocarlo se ve entero y se responde: el texto va por stdin y la IA recibe el mensaje de qdd', async ($, on) => {
  const r = nuevo([PREGUNTA])
  const ui = await montado($, on, r)
  expect(await ui.find({ key: 'espera-respuesta' })).toBeUndefined()
  await ui.press({ key: 'espera-0' })
  expect(await ui.find({ type: 'Text', text: LARGA })).toBeDefined()

  r.esperas = []
  await ui.input({ key: 'espera-respuesta', text: 'sí' })
  await esperar()
  const [llamada] = respuestas(r)
  expect(llamada.argv).toEqual(['qdd', 'consola', 'pregunta', 'responder', 'espera-a1'])
  expect(llamada.stdin).toBe('sí')
  expect(r.enviados).toEqual([MENSAJE])
  expect(await ui.find({ type: 'Text', text: LARGA })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: /enviada a la IA/ })).toBeDefined()
})

test('una respuesta vacía no se manda', async ($, on) => {
  const r = nuevo([PREGUNTA])
  const ui = await montado($, on, r)
  await ui.press({ key: 'espera-0' })
  await ui.input({ key: 'espera-respuesta', text: '   ' })
  await esperar()
  expect(respuestas(r)).toEqual([])
  expect(await ui.find({ type: 'Text', text: LARGA })).toBeDefined()
})

test('si el teléfono ya la respondió, lo dice y no le envía nada a la IA', async ($, on) => {
  const r = nuevo([PREGUNTA])
  r.responde = { exitCode: 1, stdout: '', stderr: '[!] ya respondida en Pixel\n' }
  const ui = await montado($, on, r)
  await ui.press({ key: 'espera-0' })
  await ui.input({ key: 'espera-respuesta', text: 'sí' })
  await esperar()
  expect(r.enviados).toEqual([])
  expect(await ui.find({ type: 'Text', text: /ya respondida en Pixel/ })).toBeDefined()
})

test('una nota se abre entera pero no se responde desde aquí', async ($, on) => {
  const r = nuevo([NOTA])
  const ui = await montado($, on, r)
  await ui.press({ key: 'espera-0' })
  expect(await ui.find({ type: 'Text', text: NOTA.texto })).toBeDefined()
  expect(await ui.find({ key: 'espera-respuesta' })).toBeUndefined()
})

test('muestra 5 y «ver todas» trae el resto; tocar el abierto lo cierra', async ($, on) => {
  const r = nuevo(Array.from({ length: 7 }, (_, i) => ({ id: 'espera-' + i, donde: 'T' + i + ' · PR', texto: '¿Pregunta ' + i + '?' })))
  const ui = await montado($, on, r)
  expect(await ui.find({ key: 'espera-4' })).toBeDefined()
  expect(await ui.find({ key: 'espera-5' })).toBeUndefined()
  await ui.press({ key: 'esperas-todas' })
  expect(await ui.find({ key: 'espera-6' })).toBeDefined()
  await ui.press({ key: 'espera-1' })
  expect(await ui.find({ key: 'espera-respuesta' })).toBeDefined()
  await ui.press({ key: 'espera-1' })
  expect(await ui.find({ key: 'espera-respuesta' })).toBeUndefined()
})
