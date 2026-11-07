import { expect, mock, test } from 'claude-code/testing'

// Botón 📷 del panel (pedido de la persona, 2026-11-07): qdd foto existía, pero la IA gastaba tokens en descubrir su
// propia herramienta. Con una imagen copiada, el panel ofrece 📷; al pulsarlo trae la foto con qdd foto y deja
// «mira <ruta>» en el chat, en el cursor, sin enviarlo: la persona sigue escribiendo.

const PANE = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Tareas', isFocused: true, bodyColumns: 60, placement: 'dock', scroll: { offset: 0, bodyRows: 30 }, view: {} },
} as const

const PANEL = {
  formato: 1,
  estado: { formato: 1, resumen: '', etapas: ['Código'], abreviaturas: ['Có'], frentes: [{ id: 'x', nombre: 'Algo', estado: 'abierto', simbolos: ['⏳'] }], esperas: [], fallas: [] },
  acciones: [],
}

type Registro = { llamadas: string[][]; rellenos: any[]; enviados: string[]; hay: string; trae: { exitCode: number; stdout: string } }

function preparar(on, r: Registro) {
  const clock = mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('prompt.fill', ($, e) => {
    r.rellenos.push(e)
    return { isFilled: true, text: e.text, cursor: 0 }
  })
  on('prompt.submit', ($, e) => {
    r.enviados.push(e.text)
    return { text: e.text }
  })
  on('process.run', ($, e) => {
    const argv = [...e.argv]
    r.llamadas.push(argv)
    if (argv[1] === 'foto' && argv.includes('--hay')) return { value: { exitCode: 0, stdout: r.hay + '\n', stderr: '' } }
    if (argv[1] === 'foto') return { value: { ...r.trae, stderr: '' } }
    if (argv[1] === 'herramienta') return { value: { exitCode: 0, stdout: JSON.stringify({ herramientas: [] }), stderr: '' } }
    return { value: { exitCode: 0, stdout: JSON.stringify(PANEL), stderr: '' } }
  })
  return clock
}

const nuevo = (hay: string, trae = { exitCode: 0, stdout: JSON.stringify({ ruta: '/work/tmp/fotos/a.png', ruta_corta: 'tmp/fotos/a.png', fuente: 'portapapeles (wl-paste)', tipo: 'image/png', bytes: 2048 }) }): Registro => ({
  llamadas: [],
  rellenos: [],
  enviados: [],
  hay,
  trae,
})

async function montar($, on, r: Registro) {
  const clock = preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await clock.advance(2000)
  return { ui, clock }
}

test('sin imagen copiada no hay botón; al copiar una aparece en la siguiente lectura', async ($, on) => {
  const r = nuevo('no')
  const { ui, clock } = await montar($, on, r)
  expect(await ui.find({ key: 'foto' })).toBeUndefined()
  r.hay = 'si'
  await clock.advance(2000)
  expect(await ui.find({ key: 'foto' })).toBeDefined()
})

test('pulsarlo trae la foto del portapapeles y deja «mira <ruta>» en el cursor, sin enviarla a la IA', async ($, on) => {
  const r = nuevo('si')
  const { ui } = await montar($, on, r)
  await ui.press({ key: 'foto' })
  expect(r.llamadas).toContainEqual(['qdd', 'foto', '--de', 'portapapeles', '--json'])
  expect(r.rellenos.at(-1)).toMatchObject({ text: 'mira tmp/fotos/a.png ', mode: 'insert' })
  expect(r.enviados).toEqual([])
  expect(await ui.find({ type: 'Text', text: /📷 tmp\/fotos\/a\.png quedó en el chat/ })).toBeDefined()
})

test('donde no se puede saber (Windows, macOS) el botón está siempre y trae también la última captura', async ($, on) => {
  const r = nuevo('no_se')
  const { ui, clock } = await montar($, on, r)
  expect(await ui.find({ key: 'foto' })).toBeDefined()
  const consultas = () => r.llamadas.filter((a) => a[1] === 'foto' && a.includes('--hay')).length
  const antes = consultas()
  await clock.advance(6000)
  expect(consultas()).toBe(antes)
  await ui.press({ key: 'foto' })
  expect(r.llamadas).toContainEqual(['qdd', 'foto', '--json'])
})

test('si qdd foto falla lo dice en el panel y no toca el chat', async ($, on) => {
  const r = nuevo('si', { exitCode: 1, stdout: JSON.stringify({ error: 'portapapeles: no hay imagen' }) })
  const { ui } = await montar($, on, r)
  await ui.press({ key: 'foto' })
  expect(r.rellenos).toEqual([])
  expect(await ui.find({ type: 'Text', text: /portapapeles: no hay imagen/ })).toBeDefined()
})

test('un qdd anterior sin --hay no deja un botón roto', async ($, on) => {
  const r = nuevo('')
  r.hay = 'Error: unknown flag: --hay'
  const { ui } = await montar($, on, r)
  expect(await ui.find({ key: 'foto' })).toBeUndefined()
})
