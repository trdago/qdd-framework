import { expect, mock, test } from 'claude-code/testing'

// 📦 Blueprints en el panel de tareas (pedido del usuario, 2026-11-01): ver los instalados y los disponibles, traer el
// catálogo, actualizar e instalar son comandos de qdd, sin la IA; solo ✦ Aplicar le escribe a la IA.

const TAREAS = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Tareas', isFocused: true, bodyColumns: 60, placement: 'dock', scroll: { offset: 0, bodyRows: 30 }, view: {} },
} as const

const PANEL = {
  formato: 1,
  carpeta: '/work/bot',
  estado: { formato: 1, resumen: '', etapas: ['Código'], abreviaturas: ['Có'], frentes: [{ id: 'a', nombre: 'Algo', estado: 'abierto', simbolos: ['⏳'] }], esperas: [], fallas: [] },
  acciones: [],
}

const ORIGEN = 'git@github.com:org/bp.git'
const entrada = (id: string, nombre: string, estado: string, instalada: string, publicada: string, extra = {}) => ({ id, nombre, origen: ORIGEN, subdir: id, instalada, publicada, ref: 'v3.0.0', estado, ...extra })

const ACTUALIZAR = { id: 'blueprint:actualizar:estado', tipo: 'comando', etiqueta: '⬆ Actualizar «Estado por tareas» a 2.0.1', detalle: 'qdd blueprint add estado', confirmar: true }
const APLICAR = { id: 'blueprint:aplicar:auth', tipo: 'atender', etiqueta: '✦ Aplicar «Usuarios y acceso» con la IA', detalle: 'Actualiza «Usuarios y acceso»…' }
const INSTALAR = { id: 'blueprint:instalar:pagos', tipo: 'comando', etiqueta: '＋ Instalar «Pagos»', detalle: 'qdd blueprint add ' + ORIGEN + ' pagos', confirmar: true }

const inventario = (estadoDeEstado = 'solo-lock', instaladaEstado = '2.0.0') => ({
  inventario: {
    entradas: [
      entrada('auth', 'Usuarios y acceso', 'con-cambios', '1.0.0', '1.1.0', { cambio: 'minor', razones: ['se agregó una invariante'] }),
      entrada('estado', 'Estado por tareas', estadoDeEstado, instaladaEstado, '2.0.1'),
      entrada('consola', 'Consola', 'al-dia', '1.0.0', '1.0.0'),
      entrada('pagos', 'Pagos', 'disponible', '', '1.0.0'),
    ],
    instalados: 3,
    disponibles: 1,
    sin_ia: estadoDeEstado === 'solo-lock' ? 1 : 0,
    con_ia: 1,
  },
  acciones: [
    { id: 'blueprint:sync', tipo: 'comando', etiqueta: '↻ Actualizar catálogo', detalle: 'qdd blueprint sync', confirmar: false },
    APLICAR,
    ...(estadoDeEstado === 'solo-lock' ? [ACTUALIZAR] : []),
    { id: 'blueprint:aplicar:estado', tipo: 'atender', etiqueta: '✦ Aplicar «Estado por tareas» con la IA', detalle: 'Aplica…' },
    { id: 'blueprint:aplicar:consola', tipo: 'atender', etiqueta: '✦ Aplicar «Consola» con la IA', detalle: 'Aplica…' },
    INSTALAR,
  ],
})

type Salida = { exitCode: number; stdout: string; stderr: string }
const ok = (d: unknown): Salida => ({ exitCode: 0, stdout: typeof d === 'string' ? d : JSON.stringify(d), stderr: '' })

function preparar(on, estado: { blueprints: unknown; acciones?: Record<string, Salida> }, registro: { llamadas: string[][]; enviados: string[] }) {
  mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work/bot' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('prompt.submit', ($, e) => {
    registro.enviados.push(e.text)
    return { text: e.text }
  })
  on('process.run', ($, e) => {
    const argv = [...e.argv]
    registro.llamadas.push(argv)
    if (argv[1] === 'herramienta') return { value: ok({ herramientas: [], problemas: null }) }
    if (argv[1] === 'consola' && argv[2] === 'blueprints') return { value: ok(estado.blueprints) }
    if (argv[1] === 'consola' && argv[2] === 'accion' && String(argv[3]).startsWith('blueprint:')) return { value: { ...(estado.acciones?.[argv[3]] ?? ok('')) } }
    return { value: ok(PANEL) }
  })
}

const nuevo = () => ({ llamadas: [] as string[][], enviados: [] as string[] })
const corrio = (r, id: string) => r.llamadas.filter((a) => a[2] === 'accion' && a[3] === id)

async function abrir($, on, estado, r) {
  preparar(on, estado, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work/bot' })
  const ui = await $.ui.mount({ ...TAREAS, surface: 'terminal' })
  expect(await ui.find({ key: 'blueprints' })).toMatchObject({ props: { label: '📦 Blueprints', hotkey: 'p' } })
  await ui.press({ key: 'blueprints' })
  return ui
}

test('📦 Blueprints lista los instalados y los disponibles con sus cuentas, sin red ni IA', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, { blueprints: inventario() }, r)
  expect(r.llamadas).toContainEqual(['qdd', 'consola', 'blueprints', '--json'])
  expect(await ui.find({ type: 'Text', text: '3 instalados · 1 sin IA · 1 con IA · 1 disponible' })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /Usuarios y acceso/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /1\.0\.0 → 1\.1\.0/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /se agregó una invariante/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /Pagos/ })).toBeDefined()
  expect(r.enviados).toEqual([])
  // p vuelve al tablero.
  await ui.press({ key: 'blueprints' })
  expect(await ui.find({ type: 'Text', text: /instalados ·/ })).toBeUndefined()
})

test('↻ Actualizar catálogo corre sin confirmar, relee y dice cuántas versiones nuevas trajo', async ($, on) => {
  const r = nuevo()
  const antes = inventario('al-dia')
  antes.inventario.entradas[1].publicada = '2.0.0'
  const estado = { blueprints: antes as unknown, acciones: { 'blueprint:sync': ok('[+] sincronizado: última versión publicada v3.0.0.\n') } }
  const ui = await abrir($, on, estado, r)
  estado.blueprints = inventario()
  await ui.press({ key: 'bp-blueprint:sync' })
  expect(corrio(r, 'blueprint:sync')).toEqual([['qdd', 'consola', 'accion', 'blueprint:sync']])
  expect(r.llamadas.filter((a) => a[2] === 'blueprints').length).toBe(2)
  expect(await ui.find({ type: 'Text', text: /1 versión nueva · 1 se actualiza sin la IA/ })).toBeDefined()
  expect(r.enviados).toEqual([])
})

test('⬆ Actualizar e ＋ Instalar escriben el lockfile: piden «s  Sí» y no le escriben a la IA', async ($, on) => {
  const r = nuevo()
  const estado = { blueprints: inventario() as unknown, acciones: { 'blueprint:actualizar:estado': ok('[+] estado-por-tareas 2.0.1\n') } }
  const ui = await abrir($, on, estado, r)
  await ui.press({ key: 'bp-' + ACTUALIZAR.id })
  expect(corrio(r, ACTUALIZAR.id)).toEqual([])
  expect(await ui.find({ type: 'Text', text: '¿' + ACTUALIZAR.etiqueta + '?' })).toBeDefined()
  await ui.press({ key: 'bp-no' })
  expect(corrio(r, ACTUALIZAR.id)).toEqual([])
  await ui.press({ key: 'bp-' + ACTUALIZAR.id })
  estado.blueprints = inventario('al-dia', '2.0.1')
  await ui.press({ key: 'bp-si' })
  expect(corrio(r, ACTUALIZAR.id)).toEqual([['qdd', 'consola', 'accion', ACTUALIZAR.id, '--confirmado']])
  expect(await ui.find({ type: 'Text', text: /1 actualizado/ })).toBeDefined()

  await ui.press({ key: 'bp-' + INSTALAR.id })
  await ui.press({ key: 'bp-si' })
  expect(corrio(r, INSTALAR.id)).toEqual([['qdd', 'consola', 'accion', INSTALAR.id, '--confirmado']])
  expect(r.enviados).toEqual([])
})

test('✦ Aplicar con la IA es lo único que le escribe a la IA, con el mensaje que da qdd', async ($, on) => {
  const r = nuevo()
  const estado = { blueprints: inventario(), acciones: { 'blueprint:aplicar:auth': ok('Actualiza «Usuarios y acceso» con qdd blueprint add auth…\n') } }
  const ui = await abrir($, on, estado, r)
  await ui.press({ key: 'bp-' + APLICAR.id })
  expect(corrio(r, APLICAR.id)).toEqual([['qdd', 'consola', 'accion', APLICAR.id]])
  expect(r.enviados).toEqual(['Actualiza «Usuarios y acceso» con qdd blueprint add auth…'])
  expect(await ui.find({ type: 'Text', text: /enviado a la IA/ })).toBeDefined()
})

test('sin copia central lo dice y ofrece traerla; lo instalado queda sin medir', async ($, on) => {
  const r = nuevo()
  const blueprints = {
    inventario: { entradas: [entrada('estado', 'Estado por tareas', 'sin-medir', '2.0.0', '')], instalados: 1, disponibles: 0, sin_ia: 0, con_ia: 0, problemas: ['sin copia central de ' + ORIGEN + ': ↻ Actualizar catálogo (qdd blueprint sync)'] },
    acciones: [{ id: 'blueprint:sync', tipo: 'comando', etiqueta: '↻ Actualizar catálogo', detalle: 'qdd blueprint sync', confirmar: false }],
  }
  const ui = await abrir($, on, { blueprints }, r)
  expect(await ui.find({ type: 'Text', text: /sin copia central/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /sin medir/ })).toBeDefined()
  expect(await ui.find({ key: 'bp-blueprint:sync' })).toBeDefined()
})
