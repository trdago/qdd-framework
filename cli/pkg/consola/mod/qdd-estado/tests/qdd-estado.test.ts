import { expect, mock, test } from 'claude-code/testing'

const PANE = {
  plugin: 'qdd-estado',
  component: 'Pane',
  requestId: 'qdd-estado',
  viewport: { columns: 160, rows: 40 },
  props: {
    title: 'Estado por frente',
    isFocused: true,
    bodyColumns: 60,
    placement: 'dock',
    scroll: { offset: 0, bodyRows: 30 },
    view: {},
  },
} as const

const ESTADO = {
  formato: 1,
  actualizado: '2026-10-29T15:13:01-03:00',
  resumen: '◧ 2 frentes · ⏳1 ❌1',
  etapas: ['Código', 'Pruebas', 'Producción', 'Validación real'],
  abreviaturas: ['Có', 'Pu', 'Po', 'Va'],
  frentes: [
    { id: 'gemini', nombre: 'Gemini retirado', estado: 'abierto', simbolos: ['✅', '✅', '✅', '⏳'] },
    { id: 'ticket', nombre: 'Ticket de MP en logs', estado: 'abierto', simbolos: ['❌', '—', '—', '—'] },
  ],
  esperas: [{ frente: 'Gemini retirado', etapa: 'Validación real', texto: 'ver un análisis con imagen', de_quien: 'uso real' }],
  fallas: [{ frente: 'Ticket de MP en logs', etapa: 'Código', texto: 'la URL con el ticket se imprime' }],
}

const ACCIONES = [
  { id: 'aprobar:pagos', tipo: 'aprobar', etiqueta: 'Aprobar «Pagos»', detalle: 'qdd modules approve pagos', confirmar: true },
  { id: 'revisar-estado', tipo: 'comando', etiqueta: 'Revisar estado', detalle: 'qdd estado check', confirmar: false },
  { id: 'auditar', tipo: 'comando', etiqueta: 'Auditar', detalle: 'qdd audit', confirmar: true },
]

const PANEL = { formato: 1, estado: ESTADO, acciones: ACCIONES }

type Salida = { exitCode: number; stdout: string; stderr: string }

// Prepara los stubs comunes: 'panel' es lo que responde 'qdd consola panel --json' y 'accion' lo de 'qdd consola accion'.
function preparar(on, panel: Salida, llamadas: string[][] = [], accion: Salida = { exitCode: 0, stdout: '', stderr: '' }) {
  const clock = mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('process.run', ($, e) => {
    llamadas.push([...e.argv])
    if (e.argv[2] === 'accion') return { value: { ...accion } }
    return { value: { ...panel } }
  })
  return clock
}

const ok = (d: unknown): Salida => ({ exitCode: 0, stdout: JSON.stringify(d), stderr: '' })

test('dibuja la cabecera con números grandes, la tabla, lo que espera cada ⏳ y las fallas (CCC-C13)', async ($, on) => {
  const llamadas: string[][] = []
  preparar(on, ok(PANEL), llamadas)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  expect(llamadas[0]).toEqual(['qdd', 'consola', 'panel', '--json'])

  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: '◧ Estado por frente' })).toBeDefined()
  // 2 frentes: la primera fila del 2 grande, en cian; 1 ⏳: la del 1, en amarillo.
  expect(await ui.find({ type: 'Text', text: '▀▀█' })).toMatchObject({ props: { color: 'cyan', weight: 'bold' } })
  expect(await ui.find({ type: 'Text', text: '▀█ ' })).toMatchObject({ props: { color: 'yellow' } })
  expect(await ui.find({ type: 'Text', text: /Có Pu Po Va/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: 'Gemini retirado' })).toMatchObject({ props: { color: 'yellow' } })
  expect(await ui.find({ type: 'Text', text: 'Ticket de MP en logs' })).toMatchObject({ props: { color: 'red' } })
  expect(await ui.find({ type: 'Text', text: /Validación real · de uso real/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /la URL con el ticket se imprime/ })).toBeDefined()
  expect(await ui.find({ key: 'b-aprobar:pagos' })).toMatchObject({ props: { hotkey: '1' } })
  expect(await ui.find({ key: 'b-revisar-estado' })).toMatchObject({ props: { hotkey: 'c' } })
  expect(await ui.find({ key: 'b-auditar' })).toMatchObject({ props: { hotkey: 'u' } })
})

test('aprobar una ficha pide «s  Sí» y solo entonces llama a qdd con --confirmado (CCC-C10)', async ($, on) => {
  const llamadas: string[][] = []
  preparar(on, ok(PANEL), llamadas, { exitCode: 0, stdout: "[+] Ficha 'pagos' aprobada (revisada en abc123).", stderr: '' })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })

  await ui.press({ key: 'b-aprobar:pagos' })
  expect(await ui.find({ type: 'Text', text: '¿Aprobar «Pagos»?' })).toBeDefined()
  expect(llamadas.some((a) => a[2] === 'accion')).toBe(false)

  await ui.press({ key: 'si' })
  expect(llamadas).toContainEqual(['qdd', 'consola', 'accion', 'aprobar:pagos', '--confirmado'])
  expect(await ui.find({ type: 'Text', text: /Ficha 'pagos' aprobada/ })).toBeDefined()
})

test('con «n  No» no se ejecuta nada', async ($, on) => {
  const llamadas: string[][] = []
  preparar(on, ok(PANEL), llamadas)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await ui.press({ key: 'b-auditar' })
  await ui.press({ key: 'no' })
  expect(llamadas.some((a) => a[2] === 'accion')).toBe(false)
  expect(await ui.find({ key: 'b-auditar' })).toBeDefined()
})

test('revisar el estado corre sin confirmación y muestra el resultado (CCC-C12)', async ($, on) => {
  const llamadas: string[][] = []
  preparar(on, ok(PANEL), llamadas, { exitCode: 1, stdout: '', stderr: '[✗] EPF-C03: falta la evidencia' })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await ui.press({ key: 'b-revisar-estado' })
  expect(llamadas).toContainEqual(['qdd', 'consola', 'accion', 'revisar-estado'])
  expect(await ui.find({ type: 'Text', text: '✗ Revisar estado' })).toMatchObject({ props: { color: 'red' } })
  expect(await ui.find({ type: 'Text', text: /EPF-C03: falta la evidencia/ })).toBeDefined()
})

test('sin estado.yaml dice cómo crearlo y sigue ofreciendo las fichas por aprobar', async ($, on) => {
  const sinEstado = { formato: 1, estado: null, motivo: "el proyecto no tiene .qdd/project/estado.yaml: créalo con 'qdd estado init'", acciones: [ACCIONES[0]] }
  preparar(on, ok(sinEstado))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: /qdd estado init/ })).toBeDefined()
  expect(await ui.find({ key: 'b-aprobar:pagos' })).toBeDefined()
})

test('sin qdd o con un qdd antiguo no falla', async ($, on) => {
  preparar(on, { exitCode: 1, stdout: '', stderr: '' })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: /actualiza qdd/ })).toBeDefined()
})

test('se redibuja solo cuando el panel cambia', async ($, on) => {
  const salida = ok(PANEL)
  const clock = preparar(on, salida)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  salida.stdout = JSON.stringify({ ...PANEL, estado: { ...ESTADO, esperas: [], fallas: [] } })
  await clock.advance(2000)
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: /Esperando/ })).toBeUndefined()
  // 0 ⏳ y 0 ❌: el 0 grande reemplaza al 1.
  expect(await ui.find({ type: 'Text', text: '▀█ ' })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: '█▀█' })).toMatchObject({ props: { color: 'yellow' } })
})
