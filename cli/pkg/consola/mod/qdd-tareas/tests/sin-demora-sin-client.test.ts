import { expect, mock, test } from 'claude-code/testing'

// Contrato de la ficha estado: los efectos solo decoran (sin el elemento Client en la superficie el panel no pide
// cuadros y dibuja lo de siempre), y un pulso nunca demora la herramienta ni el subagente que lo produce.

const PANE = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Tareas', isFocused: true, bodyColumns: 60, placement: 'dock', scroll: { offset: 0, bodyRows: 30 }, view: {} },
} as const

const ESTADO = {
  formato: 1,
  resumen: '◧ 1 tarea · ⏳1',
  etapas: ['Código', 'Pruebas'],
  abreviaturas: ['Có', 'Pu'],
  frentes: [{ id: 'gemini', nombre: 'Gemini retirado', estado: 'abierto', simbolos: ['✅', '⏳'] }],
  esperas: [],
  fallas: [],
}
const EFECTOS = { activo: true, acento: '#7daea3', abrir_ms: 800, cerrar_ms: 400, cuadro_ms: 40, origen: 'hypr-materialize' }
const PANEL = { formato: 1, estado: ESTADO, acciones: [], efectos: EFECTOS }
const ok = (d: unknown) => ({ exitCode: 0, stdout: JSON.stringify(d), stderr: '' })

function preparar(on, llamadas: string[][], pulso: () => unknown = () => ({ value: ok({}) })) {
  const clock = mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('process.run', ($, e) => {
    llamadas.push([...e.argv])
    if (e.argv[1] === 'estado' && e.argv[2] === 'pulso') return pulso()
    if (e.argv[2] === 'materializar') {
      const { lote } = JSON.parse(e.init.stdin)
      return { value: ok({ lote: lote.map((p) => ({ cuadro_ms: 40, cuadros: [[[{ texto: '#' }]], p.lineas] })) }) }
    }
    return { value: ok(PANEL) }
  })
  return clock
}

// FND-272: la tabla de elementos de una superficie viene completa con todos los nombres, y uno que no tiene dibuja un
// fragmento. En vscode o mobile `Client` existía igual: con los efectos encendidos el panel pedía cuadros, dibujaba la
// fila en blanco y después un fragmento, y «Gemini retirado» no volvía a verse.
// contrato: estado#9376e8fe
for (const surface of ['vscode', 'mobile'] as const) {
  test(`con los efectos encendidos pero sin Client (${surface}), no pide cuadros y dibuja lo de siempre (FND-272)`, async ($, on) => {
    const llamadas: string[][] = []
    const clock = preparar(on, llamadas)
    await $.session.start({ surface, isInteractive: true, cwd: '/work' })
    const ui = await $.ui.mount({ ...PANE, surface })
    await clock.advance(100)
    expect(llamadas.some((a) => a[2] === 'materializar')).toBe(false)
    expect(await ui.find({ type: 'Text', text: 'Gemini retirado' })).toBeDefined()
  })
}

// contrato: estado#8c838d48
test('un «qdd estado pulso» que no termina no demora la herramienta ni el subagente', async ($, on) => {
  const llamadas: string[][] = []
  const clock = preparar(on, llamadas, () => new Promise(() => {}))
  on('tool.call', () => ({ result: 'hecho' }))
  on('agent.spawn', () => ({ model: 'sonnet' }))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const tarde = Symbol('tarde')
  const aTiempo = (p: Promise<unknown>) => Promise.race([p, new Promise((r) => setTimeout(() => r(tarde), 1000))])
  expect(await aTiempo($.tool.call({ tool: 'Bash', command: 'ls' }))).toMatchObject({ result: 'hecho' })
  await clock.advance(6000)
  expect(await aTiempo($.agent.spawn({ prompt: 'revisa' }))).toMatchObject({ model: 'sonnet' })
  await clock.advance(1)
  const pulsos = llamadas.filter((a) => a[1] === 'estado' && a[2] === 'pulso').map((a) => a[4])
  expect(pulsos).toEqual(['agente', 'subagente'])
})
