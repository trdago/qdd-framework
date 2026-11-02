import { expect, mock, test } from 'claude-code/testing'

// Pedido del usuario (2026-11-01): confirmado un pedido, la intención ya está tomada. Si su comando dice «todavía no»
// o espera a otro, qdd lo deja ◷ armado y lo reintenta solo; el panel lo dice con jerarquía (lo importante primero y
// en color, el detalle atenuado debajo) y ofrece «Dejar de esperar».

const PANE = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Tareas', isFocused: true, bodyColumns: 60, placement: 'dock', scroll: { offset: 0, bodyRows: 30 }, view: {} },
} as const

const ESTADO = { formato: 1, resumen: '', etapas: ['Código'], abreviaturas: ['Có'], frentes: [{ id: 'x', nombre: 'Algo', estado: 'abierto', simbolos: ['⏳'] }], esperas: [], fallas: [] }
const PUBLICAR = { id: 'pedido:1', tipo: 'pedido', tecla: 'a', etiqueta: 'Publicar rc.13', detalle: 'bash paso-rc13.sh', confirmar: true, estado: 'pendiente' }
const INSTALAR = { id: 'pedido:2', tipo: 'pedido', tecla: 'b', etiqueta: 'Instalar rc.13', detalle: 'bash instalar.sh', confirmar: true, estado: 'espera', espera: ['a'] }
const ARMADO = {
  id: 'pedido:3',
  tipo: 'pedido',
  etiqueta: 'Instalar rc.12',
  detalle: 'bash instalar.sh',
  confirmar: true,
  estado: 'armado',
  aviso: 'Todavía no: npm preview tiene 2.8.0-rc.11; el workflow tarda unos 5 min',
  aviso_en: '2026-11-01T22:00:00-03:00',
  proximo: '2026-11-01T22:00:30-03:00',
  intentos: 2,
}

function preparar(on, acciones: () => unknown[], llamadas: string[][]) {
  const clock = mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('process.run', ($, e) => {
    llamadas.push([...e.argv])
    if (e.argv[2] === 'accion' || e.argv[2] === 'desarmar') return { value: { exitCode: 0, stdout: '', stderr: '' } }
    if (e.argv[1] === 'herramienta') return { value: { exitCode: 0, stdout: JSON.stringify({ herramientas: [] }), stderr: '' } }
    return { value: { exitCode: 0, stdout: JSON.stringify({ formato: 1, estado: ESTADO, acciones: acciones() }), stderr: '' } }
  })
  return clock
}

test('un pedido armado dice primero y en color que ya tiene tu intención; el detalle va atenuado debajo', async ($, on) => {
  const llamadas: string[][] = []
  preparar(on, () => [ARMADO], llamadas)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await ui.advance(1000)
  expect(await ui.find({ type: 'Text', text: '◷  Instalar rc.12' })).toMatchObject({ props: { color: 'yellow', weight: 'bold' } })
  expect(await ui.find({ type: 'Text', text: /^\s*Armado · corre solo cuando esté listo · intento 2/ })).toMatchObject({ props: { color: 'yellow' } })
  expect(await ui.find({ type: 'Text', text: /npm preview tiene 2\.8\.0-rc\.11/ })).toMatchObject({ props: { dimColor: true } })
  // Sin tecla: no se acepta dos veces; sí se puede dejar de esperar.
  expect(await ui.find({ key: 'b-pedido:3' })).toBeUndefined()
  await ui.press({ key: 'desarmar-pedido:3' })
  expect(llamadas).toContainEqual(['qdd', 'consola', 'desarmar', '3'])
})

test('uno que espera a otro se confirma antes: la confirmación dice que corre solo cuando el otro termine', async ($, on) => {
  const llamadas: string[][] = []
  preparar(on, () => [PUBLICAR, INSTALAR], llamadas)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await ui.advance(1000)
  expect(await ui.find({ type: 'Text', text: /espera a \[a\]/ })).toBeDefined()
  await ui.press({ key: 'b-pedido:2' })
  expect(await ui.find({ type: 'Text', text: /Corre solo cuando \[a\] termine/ })).toBeDefined()
  await ui.press({ key: 'si' })
  expect(llamadas).toContainEqual(['qdd', 'consola', 'accion', 'pedido:2', '--confirmado'])
})

test('un «todavía no» sin armar se lee primero y en color, no en gris', async ($, on) => {
  const llamadas: string[][] = []
  const pendiente = { ...PUBLICAR, aviso: 'Dejó de esperar tras 2h0m0s: confírmalo de nuevo cuando esté listo.', aviso_en: '2026-11-01T22:00:00-03:00' }
  preparar(on, () => [pendiente], llamadas)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await ui.advance(1000)
  expect(await ui.find({ type: 'Text', text: /^\s*◷ Todavía no/ })).toMatchObject({ props: { color: 'yellow', weight: 'bold' } })
  expect(await ui.find({ type: 'Text', text: /Dejó de esperar tras 2h0m0s/ })).toMatchObject({ props: { dimColor: true } })
})
