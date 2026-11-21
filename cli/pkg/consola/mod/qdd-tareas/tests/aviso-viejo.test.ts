import { expect, mock, test } from 'claude-code/testing'

// FND-192 (MEJ-046, Windows rc.44): con la IA trabajando, el aviso de un ⏳ quedaba en la cola del chat y llegaba después
// de que la IA cerró la etapa; dos pulsaciones eran dos avisos. Ahora, con la IA en un turno, el panel guarda uno por
// etapa y al terminar el turno se lo vuelve a pedir a qdd: si la etapa ya no espera, no se envía.

const PANEL = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Tareas', isFocused: true, bodyColumns: 40, placement: 'dock', scroll: { offset: 0, bodyRows: 30 }, viewportRows: 40 },
} as const

const PREGUNTA = { atender: 'atender:branding:1', donde: 'Instalar blueprint de branding · Pruebas', texto: '¿Apruebas la ficha marca-lyd?' }
const MENSAJE = '⏳ Pruebas · Instalar blueprint de branding (tarea branding): espera ¿Apruebas la ficha marca-lyd? (persona).'
const YA_NO = { exitCode: 1, stdout: '', stderr: "[!] la acción 'atender:branding:1' no está entre las que ofrece\n" }

function preparar(on, r) {
  mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('turn.start', ($, e) => ({ turnId: e.turnId }))
  on('turn.complete', ($, e) => ({ text: e.answer }))
  on('prompt.submit', ($, e) => {
    r.enviados.push(e.text)
    return { text: e.text }
  })
  on('process.run', ($, e) => {
    if (e.argv.includes('accion')) r.acciones++
    if (e.argv.includes('binario')) return { value: { exitCode: 1, stdout: '', stderr: '' } }
    if (e.argv.includes('accion')) return { value: r.atiende }
    if (e.argv.includes('panel')) return { value: { exitCode: 0, stdout: JSON.stringify({ formato: 1, carpeta: '/work', acciones: [], esperas_persona: [PREGUNTA] }), stderr: '' } }
    return { value: { exitCode: 0, stdout: '{}', stderr: '' } }
  })
}

const nuevo = () => ({ enviados: [] as string[], acciones: 0, atiende: { exitCode: 0, stdout: MENSAJE + '\n', stderr: '' } })
const esperar = () => new Promise((res) => setTimeout(res, 20))

async function montado($, on, r) {
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await esperar()
  const ui = await $.ui.mount({ ...PANEL, surface: 'terminal' })
  if (await ui.find({ key: 'preguntas-ver' })) await ui.press({ key: 'preguntas-ver' })
  return ui
}

test('con la IA en un turno, el aviso espera a que termine y dos pulsaciones son uno solo', async ($, on) => {
  const r = nuevo()
  const ui = await montado($, on, r)
  await $.turn.start({ turnId: 't1' })
  await ui.press({ key: 'espera-0' })
  await esperar()
  await ui.press({ key: 'espera-0' })
  await esperar()
  expect(r.enviados).toEqual([])
  expect(await ui.find({ type: 'Text', text: /cuando la IA termine/ })).toBeDefined()
  await $.turn.complete({ turnId: 't1', answer: '' })
  await esperar()
  expect(r.acciones).toBe(1)
  expect(r.enviados).toEqual([MENSAJE])
})

test('si al terminar el turno la etapa ya no espera, el aviso no se envía', async ($, on) => {
  const r = nuevo()
  const ui = await montado($, on, r)
  await $.turn.start({ turnId: 't1' })
  await ui.press({ key: 'espera-0' })
  await esperar()
  r.atiende = YA_NO
  await $.turn.complete({ turnId: 't1', answer: '' })
  await esperar()
  expect(r.enviados).toEqual([])
  expect(await ui.find({ type: 'Text', text: /no está entre las que ofrece/ })).toBeDefined()
})

test('un subagente que termina no suelta el aviso: espera al turno de la conversación', async ($, on) => {
  const r = nuevo()
  const ui = await montado($, on, r)
  await $.turn.start({ turnId: 't1' })
  await ui.press({ key: 'espera-0' })
  await esperar()
  await $.turn.complete({ turnId: 's1', agentId: 'sub-1', answer: '' })
  await esperar()
  expect(r.enviados).toEqual([])
  await $.turn.complete({ turnId: 't1', answer: '' })
  await esperar()
  expect(r.enviados).toEqual([MENSAJE])
})

test('con la IA libre se envía en el acto, como siempre', async ($, on) => {
  const r = nuevo()
  const ui = await montado($, on, r)
  await ui.press({ key: 'espera-0' })
  await esperar()
  expect(r.enviados).toEqual([MENSAJE])
})
