import { expect, mock, test } from 'claude-code/testing'

// El panel con la marca de QDD Móvil (pedido del usuario, 2026-11-05): la paleta de docs/marca/tokens.json, el ámbar
// para lo que espera a la persona, una fila por tarea y una cabecera donde cada cosa tiene su lugar.

const AMBAR = '#d8a657'
const AZUL = '#7daea3'
const VERDE = '#a9b665'
const ROJO = '#ea6962'
const SOBRE_AMBAR = '#282828'

const PANE = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Tareas', isFocused: true, bodyColumns: 60, placement: 'dock', scroll: { offset: 0, bodyRows: 30 }, view: {} },
}

const ESTADO = {
  formato: 1,
  actualizado: '2026-11-05T07:00:00-03:00',
  hace: 'hace 2 min',
  resumen: '',
  etapas: ['Código', 'Pruebas', 'PR', 'Publicación'],
  abreviaturas: ['Có', 'Pe', 'PR', 'Pu'],
  objetivos: [
    { id: 'vacio', nombre: 'Un objetivo sin tareas a la vista', simbolo: '⏳', resumen: '0/3 tareas listas', pregunta: '¿Cómo vamos?' },
    { id: 'otro', nombre: 'Otro objetivo sin tareas a la vista', simbolo: '⏳', resumen: '1/3 tareas listas', pregunta: '¿Cómo vamos?' },
    { id: 'movil', nombre: 'Un objetivo con un nombre tan largo que no cabe nunca en la fila del panel', foco: true, simbolo: '⏳', resumen: '2/9 tareas listas', pregunta: '¿Cómo vamos?' },
  ],
  frentes: [
    {
      id: 'larga',
      nombre: 'FND-138 · el teléfono solo dice «entregada» si Claude Code la aceptó de verdad',
      estado: 'abierto',
      simbolos: ['·', '⏳', '·', '·'],
      objetivo: 'movil',
      prefijo: '└─ ',
      pregunta: '¿Cómo vamos?',
      actividad: { running: true, ultimo_pulso: '2026-11-05T07:00:00-03:00' },
    },
  ],
  ocultos: 3,
  tareas: 4,
  esperan: 1,
  fallan: 0,
  esperas: [],
  fallas: [],
}

const PEDIDO = { id: 'pedido:1', tipo: 'pedido', tecla: 'a', etiqueta: 'Publicar la prueba', detalle: 'git push origin v1', carpeta: '/work/tienda', confirmar: true }
const TODAS = [
  { id: 'abrir-carpeta', tipo: 'abrir', etiqueta: 'Abrir carpeta', detalle: 'xdg-open /work/tienda', confirmar: false },
  { id: 'revisar-estado', tipo: 'comando', etiqueta: 'Revisar estado', detalle: 'qdd estado check', confirmar: false },
  { id: 'auditar', tipo: 'comando', etiqueta: 'Auditar', detalle: 'qdd audit', confirmar: true },
  { id: 'mejoras-ciclo', tipo: 'comando', etiqueta: 'Ciclo de mejoras', detalle: 'qdd mejoras ciclo', confirmar: true },
  { id: 'goal-activar', tipo: 'comando', etiqueta: 'Modo goal', detalle: 'el modo goal', confirmar: true },
]
const PANEL = { formato: 2, version: 'v2.10.0', carpeta: '/work/tienda', estado: ESTADO, acciones: [PEDIDO, ...TODAS] }

async function abrir($, on, panel: unknown = PANEL) {
  mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work/tienda' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('process.run', () => ({ value: { exitCode: 0, stdout: JSON.stringify(panel), stderr: '' } }))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work/tienda' })
  return $.ui.mount({ ...PANE, surface: 'terminal' })
}

test('la cabecera: isotipo y título a la izquierda, carpeta y versión a la derecha, sin colores que no significan nada', async ($, on) => {
  const ui = await abrir($, on)
  expect(await ui.find({ type: 'Text', text: '▶_' })).toMatchObject({ props: { color: AMBAR } })
  const titulo = await ui.find({ type: 'Text', text: 'Tareas · hace 2 min' })
  expect(titulo).toMatchObject({ props: { weight: 'bold' } })
  expect(titulo.props.color).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: 'tienda · v2.10.0' })).toMatchObject({ props: { dimColor: true } })
  expect((await ui.find({ type: 'Text', text: '1 de 4 tareas' })).props.color).toBeUndefined()
})

// FND-140: los botones compartían fila con el título y, si no cabían, se pisaban («l: ↻brp: 📦 Blueprints»).
test('los botones de la cabecera van en su propia fila, que se dobla si no cabe, sin emojis anchos (FND-140)', async ($, on) => {
  const ui = await abrir($, on)
  expect(await ui.find({ key: 'herramientas' })).toMatchObject({ props: { flexDirection: 'row', flexWrap: 'wrap' } })
  expect(await ui.find({ key: 'leer' })).toMatchObject({ props: { label: '↻', hotkey: 'l', plain: true } })
  expect(await ui.find({ key: 'blueprints' })).toMatchObject({ props: { label: 'Blueprints', hotkey: 'p', plain: true } })
  expect(await ui.find({ key: 'config' })).toMatchObject({ props: { label: 'Config', hotkey: 'k', plain: true } })
  expect(await ui.find({ key: 'b-abrir-carpeta' })).toMatchObject({ props: { label: 'Abrir carpeta', hotkey: 'o', plain: true } })
  const textos = (await ui.findAll({ type: 'Text' })).map((n) => n.text).join('\n')
  expect(textos).not.toMatch(/📂|📦|⚙/)
})

// FND-139: «k» era a la vez Config y Modo goal.
test('ningún botón del panel comparte su tecla con otro (FND-139)', async ($, on) => {
  const nueve = 'abdefghij'.split('').map((t, i) => ({ ...PEDIDO, id: 'pedido:' + i, tecla: t }))
  const ui = await abrir($, on, { ...PANEL, acciones: [...nueve, ...TODAS] })
  const teclas = (await ui.findAll({ type: 'Button' })).map((b) => b.props.hotkey).filter(Boolean)
  expect(teclas).toContain('k')
  expect(teclas.length).toBeGreaterThan(14)
  expect(teclas.filter((t, i) => teclas.indexOf(t) !== i)).toEqual([])
})

test('«▶ para ti» va primero en el resumen y relleno en ámbar, como en la app', async ($, on) => {
  const ui = await abrir($, on)
  expect(await ui.find({ type: 'Text', text: '▶ 1 para ti' })).toMatchObject({ props: { backgroundColor: AMBAR, color: SOBRE_AMBAR, weight: 'bold' } })
  const resumen = await ui.find({ key: 'resumen' })
  expect(resumen.children[0]).toMatchObject({ props: { backgroundColor: AMBAR } })
})

test('cada estado usa su color de marca', async ($, on) => {
  const estado = { ...ESTADO, esperan: 1, fallan: 1, fallas: [{ frente: 'x', etapa: 'y', texto: 'z' }] }
  const ui = await abrir($, on, { ...PANEL, estado })
  expect(await ui.find({ type: 'Text', text: '⏳ 1 etapa espera' })).toMatchObject({ props: { color: AMBAR } })
  expect(await ui.find({ type: 'Text', text: '❌ 1 etapa falla' })).toMatchObject({ props: { color: ROJO } })
  expect(await ui.find({ type: 'Text', text: '●' })).toMatchObject({ props: { color: AZUL } })
  expect(await ui.find({ type: 'Text', text: 'running' })).toMatchObject({ props: { color: AZUL } })
  expect(await ui.find({ type: 'Text', text: /Un objetivo con un nombre/ })).toMatchObject({ props: { color: AZUL, weight: 'bold' } })
})

test('sin esperas ni fallas, el resumen lo dice en el verde de la marca', async ($, on) => {
  const estado = { ...ESTADO, esperan: 0, frentes: [{ ...ESTADO.frentes[0], simbolos: ['✅', '·', '·', '·'] }] }
  const ui = await abrir($, on, { ...PANEL, acciones: [], estado })
  expect(await ui.find({ type: 'Text', text: '✓ nada espera ni falla' })).toMatchObject({ props: { color: VERDE } })
})

test('una tarea ocupa una sola fila, aunque su nombre sea largo', async ($, on) => {
  const ui = await abrir($, on)
  const nombre = await ui.find({ type: 'Text', text: /^FND-138/ })
  expect(nombre.text.length + '└─ '.length).toBeLessThanOrEqual(60 - 14 - 3 - 10)
  expect(nombre.text.endsWith('…')).toBe(true)
})

test('la tarea en curso cabe: su columna de nombre descuenta «● running» (ninguna línea más ancha que el panel)', async ($, on) => {
  const ui = await abrir($, on)
  const fila = await ui.find({ key: 'f-larga' })
  // 60 de ancho: 14 de etapas, 3 de la ✕ y 10 de «● running».
  expect(fila.children[0].props.width).toBeLessThanOrEqual(60 - 14 - 3 - 10)
})

test('un objetivo: el nombre se recorta y su estado queda a la derecha, siempre visible', async ($, on) => {
  const ui = await abrir($, on)
  const fila = await ui.find({ key: 'o-movil' })
  expect(fila).toMatchObject({ props: { flexDirection: 'row', justifyContent: 'space-between' } })
  expect(fila.children[0]).toMatchObject({ props: { flexShrink: 1 } })
  expect(fila.children[1]).toMatchObject({ props: { flexShrink: 0 } })
  const nombre = await ui.find({ type: 'Text', text: /Un objetivo con un nombre/ })
  const estado = await ui.find({ type: 'Text', text: '⏳ 2/9 tareas listas' })
  // El 🎯 ya mide 2 en .length; el ⏳ ocupa una columna más. Entre los dos hay 2 de separación.
  expect(nombre.text.length + estado.text.length + 1 + 2).toBeLessThanOrEqual(60)
  expect(nombre.text.endsWith('…')).toBe(true)
})

test('los objetivos sin tareas a la vista van juntos, sin una línea en blanco entre cada uno', async ($, on) => {
  const ui = await abrir($, on)
  expect((await ui.find({ key: 'o-vacio' })).props.marginTop).toBe(1)
  expect((await ui.find({ key: 'o-otro' })).props.marginTop).toBe(0)
  expect((await ui.find({ key: 'o-movil' })).props.marginTop).toBe(1)
})

test('el pie es una línea corta, sin cursiva, y no repite lo que ya dicen los botones', async ($, on) => {
  const ui = await abrir($, on)
  const pie = await ui.find({ type: 'Text', text: 'nombre → ¿cómo vamos? · ⏳/❌ → la IA · foco: /tareas' })
  expect(pie.props.italic).toBeUndefined()
  const leyenda = await ui.find({ type: 'Text', text: /^Có Código/ })
  expect(leyenda).toMatchObject({ props: { dimColor: true } })
  expect(leyenda.props.italic).toBeUndefined()
})

test('una versión nueva sin botón (Windows) se avisa en la cabecera con lo que dice qdd', async ($, on) => {
  const aviso = '⬆ qdd 2.9.7 disponible: cierra las sesiones de Claude Code y corre npm i -g qdd-framework@2.9.7'
  const ui = await abrir($, on, { ...PANEL, aviso_version: aviso })
  expect(await ui.find({ type: 'Text', text: aviso })).toMatchObject({ props: { color: AMBAR, wrap: 'wrap' } })
})

test('la actualización de qdd se ofrece como pedido, pero no abre el panel', async ($, on) => {
  const aperturas: unknown[] = []
  const clock = mock.clock(on)
  mock.env(on, {})
  const salida = { exitCode: 0, stdout: JSON.stringify({ ...PANEL, acciones: [] }), stderr: '' }
  on('session.start', () => ({ cwd: '/work/tienda' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', ($, e) => {
    aperturas.push(e)
    return { value: { isPlaced: true } }
  })
  on('process.run', () => ({ value: { ...salida } }))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work/tienda' })
  const antes = aperturas.length
  salida.stdout = JSON.stringify({ ...PANEL, acciones: [{ ...PEDIDO, id: 'pedido:9', tecla: 'b', etiqueta: '⬆ Actualizar qdd a 2.9.7', version_nueva: '2.9.7' }] })
  await clock.advance(2000)
  expect(aperturas.length).toBe(antes)
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ key: 'b-pedido:9' })).toMatchObject({ props: { hotkey: 'b' } })
})
