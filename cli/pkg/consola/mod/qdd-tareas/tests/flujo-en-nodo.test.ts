import { expect, mock, test } from 'claude-code/testing'

// Pedido de la persona (2026-11-08): «en el panel de flujo no me gusta tener que bajar hasta abajo para una acción al nodo
// del flujo; me gustaría que fuera directa sobre el elemento para no perder el hilo». qdd entrega cada línea partida por
// nodo (segmentos); el panel hace pulsable el nodo en el dibujo y abre su tarjeta justo debajo de él.

const FLUJOS = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-flujos',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Flujos', isFocused: true, bodyColumns: 60, placement: 'dock', scroll: { offset: 0, bodyRows: 30 }, viewportRows: 40 },
} as const

const PAGO = { id: 'pago', nombre: 'Cómo se cobra', nodos: 2, huella: 'h1' }
const GRANDE = { id: 'analisis', nombre: 'Análisis', nodos: 23, huella: 'h1' }

const conSegmentos = () => ({
  id: 'pago',
  nombre: 'Cómo se cobra',
  lineas: ['┌───────┐', '│ Llega │', '└───┬───┘', '    │', '  ( Fin )'],
  segmentos: [
    [{ texto: '┌───────┐', nodo: 'llega' }],
    [{ texto: '│ Llega │', nodo: 'llega' }],
    [{ texto: '└───┬───┘', nodo: 'llega' }],
    null,
    [{ texto: '  ' }, { texto: '( Fin )', nodo: 'fin' }],
  ],
  nodos: [
    { id: 'llega', tipo: 'paso', nombre: 'Llega', descripcion: 'la solicitud', siguen: ['fin'] },
    { id: 'fin', tipo: 'resultado', nombre: 'Fin' },
  ],
  por_enviar: [],
  puede_deshacer: false,
})

const panorama = () => ({
  id: 'analisis',
  nombre: 'Análisis',
  lineas: ['┌─────────┐', '│ Filtrar │', '└─────────┘'],
  segmentos: [[{ texto: '┌─────────┐', nodo: 'grupo-filtrar' }], [{ texto: '│ Filtrar │', nodo: 'grupo-filtrar' }], [{ texto: '└─────────┘', nodo: 'grupo-filtrar' }]],
  nodos: [{ id: 'region', tipo: 'paso', nombre: 'Región', grupo: 'Filtrar' }],
  grupos: ['Filtrar'],
  niveles: { region: 2 },
  por_enviar: [],
  puede_deshacer: false,
})

function preparar(on, r) {
  mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('classic.UserPromptSubmit', () => ({}))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('ui.close', () => ({ value: undefined }))
  on('process.run', ($, e) => {
    r.llamadas.push([...e.argv])
    if (e.argv.includes('binario')) return { value: { exitCode: 1, stdout: '', stderr: '' } }
    if (e.argv.includes('panel')) return { value: { exitCode: 0, stdout: JSON.stringify({ formato: 1, carpeta: '/work', acciones: [], herramientas: [], flujos: r.flujos }), stderr: '' } }
    if (e.argv[1] !== 'flujo') return { value: { exitCode: 0, stdout: '{}', stderr: '' } }
    if (e.argv[2] === 'ver') return { value: { exitCode: 0, stdout: JSON.stringify(r.dibujo), stderr: '' } }
    return { value: { exitCode: 0, stdout: '', stderr: '' } }
  })
}

const nuevo = (flujos: object[] = [PAGO], dibujo: object = conSegmentos()) => ({ llamadas: [] as string[][], flujos, dibujo })
const esperar = () => new Promise((res) => setTimeout(res, 20))
const comandos = (r, sub: string) => r.llamadas.filter((a) => a[1] === 'flujo' && a[2] === sub)
const claves = async (ui) => (await ui.findAll({ type: 'Button' })).map((b) => b.props.key ?? b.key)

async function abrir($, on, r) {
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...FLUJOS, surface: 'terminal' })
  await esperar()
  return ui
}

test('cada nodo del dibujo es pulsable donde se ve, y su trazo de unión no', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, r)
  const botones = await claves(ui)
  expect(botones).toContain('fl-dn:llega:1')
  expect(botones).toContain('fl-dn:fin:4')
  expect(await ui.find({ type: 'Button', label: '│ Llega │' })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: '    │' })).toBeDefined()
})

test('tocar un nodo en el dibujo abre su tarjeta justo debajo de él, y no otra abajo', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, r)
  await ui.press({ key: 'fl-dn:llega:1' })
  await esperar()
  expect(comandos(r, 'ver').at(-1)).toContain('--elegido')
  const orden = await claves(ui)
  const comentar = orden.indexOf('fl-comentar')
  expect(comentar).toBeGreaterThan(orden.indexOf('fl-dn:llega:2'))
  expect(comentar).toBeLessThan(orden.indexOf('fl-dn:fin:4'))
  expect(orden.filter((k) => k === 'fl-comentar').length).toBe(1)
  expect(await ui.find({ type: 'Text', text: 'la solicitud' })).toBeDefined()

  await ui.press({ key: 'fl-dn:llega:0' })
  await esperar()
  expect(await ui.find({ key: 'fl-comentar' })).toBeUndefined()
})

test('mover y flechas eligen el otro nodo con un clic en el dibujo', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, r)
  await ui.press({ key: 'fl-dn:llega:1' })
  await esperar()
  await ui.press({ key: 'fl-mover' })
  expect(await ui.find({ type: 'Text', text: 'Elige el nodo después del que va.' })).toBeDefined()
  await ui.press({ key: 'fl-dn:fin:4' })
  await esperar()
  expect(comandos(r, 'mover')[0]).toEqual(['qdd', 'flujo', 'mover', '--de', 'persona', '--despues-de', 'fin', '--', 'pago', 'llega'])
})

test('la lista de nodos sigue para el teclado, sin repetir la tarjeta', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, r)
  await ui.press({ key: 'fl-n-fin' })
  await esperar()
  const orden = await claves(ui)
  expect(orden.filter((k) => k === 'fl-comentar').length).toBe(1)
  expect(orden.indexOf('fl-comentar')).toBeLessThan(orden.indexOf('fl-n-llega'))
})

test('en el panorama, tocar la caja de un grupo lo abre', async ($, on) => {
  const r = nuevo([GRANDE], panorama())
  const ui = await abrir($, on, r)
  expect(comandos(r, 'ver').at(-1)).toEqual(['qdd', 'flujo', 'ver', 'analisis', '--ancho', '60', '--json', '--nivel', '1'])
  await ui.press({ key: 'fl-dn:grupo-filtrar:1' })
  await esperar()
  expect(comandos(r, 'ver').at(-1)).toEqual(['qdd', 'flujo', 'ver', 'analisis', '--ancho', '60', '--json', '--nivel', '1', '--abrir', 'Filtrar'])
})

test('un qdd anterior sin segmentos dibuja como antes y la tarjeta sigue en la lista', async ($, on) => {
  const d = conSegmentos() as any
  delete d.segmentos
  const r = nuevo([PAGO], d)
  const ui = await abrir($, on, r)
  expect(await ui.find({ key: 'fl-dn:llega:1' })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: '│ Llega │' })).toBeDefined()
  await ui.press({ key: 'fl-n-llega' })
  await esperar()
  expect(await ui.find({ key: 'fl-comentar' })).toBeDefined()
})

// Pedido de la persona (2026-11-08): «le falta peso visual […] usar distintos tamaños de letras, colores para marcar
// cosas, siempre respetando el branding». qdd marca el rol de cada trozo; el panel pinta con la marca sin mover columnas.
const conRoles = () => ({
  ...conSegmentos(),
  lineas: ['┌───────┐', '│ Llega │', '├───────┤', '│ la so │', '│ ✅ ok │', '└───┬───┘', '    │', '  ( Fin )'],
  segmentos: [
    [{ texto: '┌───────┐', nodo: 'llega' }],
    [{ texto: '│ ', nodo: 'llega' }, { texto: 'Llega', nodo: 'llega', rol: 'titulo' }, { texto: ' │', nodo: 'llega' }],
    [{ texto: '├───────┤', nodo: 'llega' }],
    [{ texto: '│ ', nodo: 'llega' }, { texto: 'la so', nodo: 'llega', rol: 'detalle' }, { texto: ' │', nodo: 'llega' }],
    [{ texto: '│ ', nodo: 'llega' }, { texto: '✅ ok', nodo: 'llega', rol: 'estado' }, { texto: ' │', nodo: 'llega' }],
    [{ texto: '└───┬───┘', nodo: 'llega' }],
    null,
    [{ texto: '  ' }, { texto: '( ', nodo: 'fin' }, { texto: 'Fin', nodo: 'fin', rol: 'titulo' }, { texto: ' )', nodo: 'fin' }],
  ],
})

const MARCA = { texto: '#d4be98', trazo: '#928374', secundario: '#a89984', acento: '#7daea3', bien: '#a9b665' }

test('el título va en negrita, el borde y el trazo en gris, y el elegido en el acento', async ($, on) => {
  const r = nuevo([PAGO], conRoles())
  const ui = await abrir($, on, r)
  const nodo = await ui.find({ key: 'fl-dn:llega:1' })
  expect(nodo).toMatchObject({ props: { label: '│ Llega │' } })
  expect(await ui.find({ type: 'Text', text: 'Llega' })).toMatchObject({ props: { weight: 'bold', color: MARCA.texto } })
  expect(await ui.find({ type: 'Text', text: '└───┬───┘' })).toMatchObject({ props: { color: MARCA.trazo } })
  expect(await ui.find({ type: 'Text', text: '    │' })).toMatchObject({ props: { color: MARCA.trazo } })
  expect(await ui.find({ type: 'Text', text: 'la so' })).toMatchObject({ props: { color: MARCA.secundario } })
  expect(await ui.find({ type: 'Text', text: '✅ ok' })).toMatchObject({ props: { color: MARCA.bien } })
  expect(await ui.find({ key: 'fl-dn:fin:7' })).toMatchObject({ props: { label: '( Fin )' } })
  await ui.press({ key: 'fl-dn:llega:1' })
  await esperar()
  expect(await ui.find({ type: 'Text', text: 'Llega' })).toMatchObject({ props: { weight: 'bold', color: MARCA.acento } })
  expect(await ui.find({ type: 'Text', text: '└───┬───┘' })).toMatchObject({ props: { color: MARCA.acento } })
})

test('un qdd sin roles dibuja los nodos como antes, sin apagar los títulos', async ($, on) => {
  const ui = await abrir($, on, nuevo())
  expect(await ui.find({ key: 'fl-dn:llega:1' })).toMatchObject({ props: { label: '│ Llega │' } })
  const texto = await ui.find({ type: 'Text', text: '│ Llega │' })
  expect(texto?.props?.color).not.toBe(MARCA.trazo)
})
