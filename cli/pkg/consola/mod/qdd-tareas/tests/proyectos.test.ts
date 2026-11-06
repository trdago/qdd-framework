import { expect, mock, test } from 'claude-code/testing'

// RFC 0018: la vista «Proyectos». qdd entrega las filas y el panel las dibuja: solo lee al abrir y con ↻, null se ve como
// «sin medir» y nada le escribe a la IA.

const tareas = (columnas: number) =>
  ({
    plugin: 'qdd-tareas',
    component: 'Pane',
    requestId: 'qdd-tareas',
    viewport: { columns: columnas, rows: 40 },
    props: { title: 'Tareas', isFocused: true, bodyColumns: columnas, placement: 'dock', scroll: { offset: 0, bodyRows: 30 }, view: {} },
  }) as const

const PANEL = {
  formato: 1,
  carpeta: '/work/bot',
  estado: { formato: 1, resumen: '', etapas: ['Código'], abreviaturas: ['Có'], frentes: [{ id: 'a', nombre: 'Algo', estado: 'abierto', simbolos: ['⏳'] }], esperas: [], fallas: [] },
  acciones: [],
}

const COMPLETO = {
  clave: 'c1',
  nombre: 'notificador',
  existe: true,
  puntaje: 90,
  hace: 'hace 3 h',
  findings_abiertos: 151,
  esperan: 6,
  fallan: 1,
  puntajes: [
    { fecha: '2026-11-05', puntaje: 40, otra_escala: true },
    { fecha: '2026-11-06', puntaje: 90 },
  ],
  findings_por_fecha: [
    { fecha: '2026-11-05', abiertos: 2, cerrados: 1 },
    { fecha: '2026-11-06', abiertos: 0, cerrados: 3, sin_estado: 1 },
  ],
  findings_sin_fecha: 4,
}
const SIN_MEDIR = { clave: 'c2', nombre: 'sin-datos', existe: true, puntaje: null, hace: '', findings_abiertos: null, esperan: null, fallan: null, puntajes: [], findings_por_fecha: [], findings_sin_fecha: 0 }
const EN_CERO = { ...SIN_MEDIR, clave: 'c3', nombre: 'en-cero', puntaje: 0, hace: 'hace 2 d', findings_abiertos: 0, esperan: 0, fallan: 0 }
const SE_FUE = { ...SIN_MEDIR, clave: 'c4', nombre: 'se-fue', existe: false }

type Salida = { exitCode: number; stdout: string; stderr: string }
const ok = (d: unknown): Salida => ({ exitCode: 0, stdout: JSON.stringify(d), stderr: '' })

function preparar(on, estado: { proyectos: Salida }, registro: { llamadas: string[][]; enviados: string[] }) {
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
    if (argv[1] === 'consola' && argv[2] === 'proyectos') return { value: { ...estado.proyectos } }
    return { value: ok(PANEL) }
  })
}

const nuevo = () => ({ llamadas: [] as string[][], enviados: [] as string[] })
const lecturas = (r) => r.llamadas.filter((a) => a[1] === 'consola' && a[2] === 'proyectos')

async function montar($, on, estado, r, columnas = 170) {
  preparar(on, estado, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work/bot' })
  return await $.ui.mount({ ...tareas(columnas), surface: 'terminal' })
}

async function abrir($, on, filas, r, columnas = 170) {
  const ui = await montar($, on, { proyectos: ok({ proyectos: filas }) }, r, columnas)
  await ui.press({ key: 'proyectos' })
  return ui
}

test('con la vista cerrada no se lee nada: el botón está y qdd no se llama', async ($, on) => {
  const r = nuevo()
  const ui = await montar($, on, { proyectos: ok({ proyectos: [COMPLETO] }) }, r)
  expect(await ui.find({ key: 'proyectos' })).toMatchObject({ props: { label: 'Proyectos', hotkey: 'y' } })
  expect(lecturas(r)).toEqual([])
})

test('al abrirla pide las filas a qdd y dibuja una por proyecto, con las mismas columnas', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, [COMPLETO], r)
  expect(lecturas(r)).toEqual([['qdd', 'consola', 'proyectos', '--json']])
  expect(await ui.find({ type: 'Text', text: /Puntaje.*Hace.*Findings.*⏳ y ❌/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /notificador\s+90\s+hace 3 h\s+151\s+⏳ 6 ❌ 1/ })).toBeDefined()
})

test('null se ve «sin medir» y un cero medido se ve como cero', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, [SIN_MEDIR, EN_CERO], r)
  expect(await ui.find({ type: 'Text', text: /sin-datos\s+sin medir\s+sin medir\s+sin medir\s+sin medir/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /en-cero\s+0\s+hace 2 d\s+0\s+⏳ 0 ❌ 0/ })).toBeDefined()
})

test('sin ratón: cada fila lleva su número como tecla, y la décima ya no', async ($, on) => {
  const r = nuevo()
  const filas = Array.from({ length: 10 }, (_, i) => ({ ...SIN_MEDIR, clave: 'k' + i, nombre: 'p' + i }))
  const ui = await abrir($, on, filas, r)
  expect(await ui.find({ key: 'proy-k0' })).toMatchObject({ props: { hotkey: '1' } })
  expect(await ui.find({ key: 'proy-k8' })).toMatchObject({ props: { hotkey: '9' } })
  expect((await ui.find({ key: 'proy-k9' })).props.hotkey).toBeUndefined()
  // El motor ya dibuja la tecla como «1: » antes del texto: la fila no la repite, y sin tecla deja ese hueco para alinear.
  expect(await ui.find({ type: 'Text', text: /^p0\s/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /^   p9\s/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /^   Proyecto\s/ })).toBeDefined()
})

test('el contador más ancho cabe entero a 40 columnas, y el nombre completo sale al elegir la fila', async ($, on) => {
  const r = nuevo()
  const largo = { ...COMPLETO, nombre: 'notificador_de_ofertas_del_estado', esperan: 29, fallan: 10 }
  const ui = await abrir($, on, [largo], r, 36)
  const fila = await ui.find({ type: 'Text', text: /⏳ 29 ❌ 10/ })
  expect(fila).toBeDefined()
  // El motor suma «1: » antes del texto.
  expect([...String(fila.props.children)].reduce((n, c) => n + (/\p{Emoji_Presentation}/u.test(c) ? 2 : 1), 3)).toBeLessThanOrEqual(36)
  await ui.press({ key: 'proy-c1' })
  expect(await ui.find({ type: 'Text', text: /notificador_de_ofertas_del_estado · hace 3 h/ })).toBeDefined()
})

test('una carpeta que ya no está dice «no existe» y no finge datos', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, [SE_FUE], r)
  expect(await ui.find({ type: 'Text', text: /se-fue\s+no existe/ })).toBeDefined()
  await ui.press({ key: 'proy-c4' })
  expect(await ui.find({ type: 'Text', text: /La carpeta ya no existe/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /Puntajes por día/ })).toBeUndefined()
})

test('a 40 columnas quedan proyecto, puntaje y ⏳ y ❌; lo demás, al elegir la fila', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, [COMPLETO], r, 40)
  const fila = await ui.find({ type: 'Text', text: /notificador/ })
  expect(fila).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /notificador\s+90\s+⏳ 6 ❌ 1/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /notificador.*hace 3 h/ })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: /notificador.*151/ })).toBeUndefined()
  await ui.press({ key: 'proy-c1' })
  expect(await ui.find({ type: 'Text', text: /hace 3 h · 151 findings abiertos/ })).toBeDefined()
})

test('un nombre largo no rompe la fila a 40 columnas', async ($, on) => {
  const r = nuevo()
  const largo = { ...COMPLETO, nombre: 'un-proyecto-con-un-nombre-larguisimo-de-verdad' }
  const ui = await abrir($, on, [largo], r, 40)
  expect(await ui.find({ type: 'Text', text: /un-proyecto.*…\s+90\s+⏳ 6 ❌ 1/ })).toBeDefined()
})

test('al elegir una fila muestra su historial; otra escala va marcada y sin estado aparte', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, [COMPLETO, SIN_MEDIR], r)
  expect(await ui.find({ type: 'Text', text: /Puntajes por día/ })).toBeUndefined()
  await ui.press({ key: 'proy-c1' })
  expect(await ui.find({ type: 'Text', text: /2026-11-06\s+90$/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /2026-11-05\s+40 · otra escala, no se compara/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /2026-11-05\s+2 abiertos · 1 cerrado/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /2026-11-06\s+0 abiertos · 3 cerrados · 1 sin estado/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /4 sin fecha/ })).toBeDefined()
  // Otra pulsación lo cierra.
  await ui.press({ key: 'proy-c1' })
  expect(await ui.find({ type: 'Text', text: /Puntajes por día/ })).toBeUndefined()
})

test('un historial sin datos dice «sin medir», no una lista vacía', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, [SIN_MEDIR], r)
  await ui.press({ key: 'proy-c2' })
  expect(await ui.find({ type: 'Text', text: /puntaje sin medir · findings sin medir/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /^sin medir$/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /ninguno con fecha/ })).toBeUndefined()
})

test('↻ relee las filas; cerrarla no vuelve a leer', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, [COMPLETO], r)
  expect(lecturas(r).length).toBe(1)
  await ui.press({ key: 'leer' })
  expect(lecturas(r).length).toBe(2)
  await ui.press({ key: 'proyectos' })
  expect(await ui.find({ type: 'Text', text: /notificador/ })).toBeUndefined()
  await ui.press({ key: 'leer' })
  expect(lecturas(r).length).toBe(2)
})

test('el aviso de qdd y una falla se muestran', async ($, on) => {
  const r = nuevo()
  const conAviso = await montar($, on, { proyectos: ok({ proyectos: [], aviso: 'No se pudo leer el registro de proyectos: qdd doctor --fix lo repara.' }) }, r)
  await conAviso.press({ key: 'proyectos' })
  expect(await conAviso.find({ type: 'Text', text: /qdd doctor --fix lo repara/ })).toBeDefined()
})

test('si qdd falla lo dice y no deja el panel sin dibujar', async ($, on) => {
  const r = nuevo()
  const ui = await montar($, on, { proyectos: { exitCode: 1, stdout: '', stderr: 'algo se rompió' } }, r)
  await ui.press({ key: 'proyectos' })
  expect(await ui.find({ type: 'Text', text: /✗ algo se rompió/ })).toBeDefined()
})

test('una carpeta de findings sin fechas no dice sin medir: los midió y no hay', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, [{ ...SIN_MEDIR, findings_abiertos: 0 }], r)
  await ui.press({ key: 'proy-c2' })
  expect(await ui.find({ type: 'Text', text: /ninguno con fecha/ })).toBeDefined()
})

test('sin proyectos lo dice', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, [], r)
  expect(await ui.find({ type: 'Text', text: /Todavía no hay proyectos/ })).toBeDefined()
})

test('nada le escribe a la IA, ni abrir ni elegir ni releer', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, [COMPLETO], r)
  await ui.press({ key: 'proy-c1' })
  await ui.press({ key: 'leer' })
  expect(r.enviados).toEqual([])
})

test('es una vista a la vez: Blueprints la cierra, y al abrirla cierra Blueprints', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, [COMPLETO], r)
  await ui.press({ key: 'blueprints' })
  expect(await ui.find({ type: 'Text', text: /notificador/ })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: /p cierra/ })).toBeDefined()
  await ui.press({ key: 'proyectos' })
  expect(await ui.find({ type: 'Text', text: /notificador/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /p cierra/ })).toBeUndefined()
})
