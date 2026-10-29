import { expect, mock, test } from 'claude-code/testing'

const PANE = {
  plugin: 'qdd-estado',
  component: 'Pane',
  requestId: 'qdd-estado',
  viewport: { columns: 160, rows: 40 },
  props: {
    title: 'Estado por frente',
    isFocused: false,
    bodyColumns: 60,
    placement: 'dock',
    scroll: { offset: 0, bodyRows: 30 },
    view: {},
  },
} as const

const DATOS = {
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

// Prepara los stubs comunes; `salida` es lo que responde la CLI y la prueba puede cambiarlo entre lecturas.
function preparar(on, salida: { exitCode: number; stdout: string; stderr: string }, llamadas: string[][] = []) {
  const clock = mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('process.run', ($, e) => {
    llamadas.push([...e.argv])
    return { value: { ...salida } }
  })
  return clock
}

test('dibuja la tabla, lo que espera cada ⏳ y las fallas', async ($, on) => {
  const llamadas: string[][] = []
  preparar(on, { exitCode: 0, stdout: JSON.stringify(DATOS), stderr: '' }, llamadas)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  expect(llamadas[0]).toEqual(['qdd', 'estado', 'ver', '--json'])

  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: /◧ 2 frentes · ⏳1 ❌1/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /Có Pu Po Va/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: 'Gemini retirado' })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /de uso real: ver un análisis con imagen/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /la URL con el ticket se imprime/ })).toBeDefined()
})

test('sin estado.yaml o sin qdd dice cómo crearlo y no falla', async ($, on) => {
  preparar(on, { exitCode: 1, stdout: '', stderr: "[!] el proyecto no tiene .qdd/project/estado.yaml: créalo con 'qdd estado init'" })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: /qdd estado init/ })).toBeDefined()
})

test('se redibuja solo cuando el archivo cambia', async ($, on) => {
  const salida = { exitCode: 0, stdout: JSON.stringify(DATOS), stderr: '' }
  const clock = preparar(on, salida)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  salida.stdout = JSON.stringify({ ...DATOS, resumen: '◧ 2 frentes · ✅', esperas: [], fallas: [] })
  await clock.advance(2000)
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: /◧ 2 frentes · ✅/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /Esperando/ })).toBeUndefined()
})
