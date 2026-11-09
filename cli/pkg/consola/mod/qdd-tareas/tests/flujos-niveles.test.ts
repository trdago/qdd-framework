import { expect, mock, test } from 'claude-code/testing'

// Flujos grandes, partes B y C (#186; la persona eligió «Los tres, en orden»). Un flujo grande abre en el panorama
// (nivel 1, un grupo por caja), se cambia de nivel con 1, 2 y 3, cada grupo se abre solo, y la tarjeta del nodo muestra
// y corrige su nivel y su grupo.

const FLUJOS = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-flujos',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Flujos', isFocused: true, bodyColumns: 60, placement: 'dock', scroll: { offset: 0, bodyRows: 30 }, viewportRows: 40 },
} as const

const GRANDE = { id: 'analisis', nombre: 'Análisis', nodos: 23, huella: 'h1' }
const CHICO = { id: 'pago', nombre: 'Cómo se cobra', nodos: 3, huella: 'h1' }

const dibujo = (id: string) => ({
  id,
  nombre: id,
  lineas: ['( dibujo de ' + id + ' )'],
  nodos: [
    { id: 'region', tipo: 'paso', nombre: 'Región', grupo: 'Filtrar' },
    { id: 'rechazo', tipo: 'resultado', nombre: 'Rechazo', grupo: 'Filtrar' },
  ],
  grupos: ['Descubrir', 'Filtrar'],
  niveles: { region: 2, rechazo: 3 },
  por_enviar: [],
  puede_deshacer: false,
})

function preparar(on, r) {
  mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('ui.close', () => ({ value: undefined }))
  on('process.run', ($, e) => {
    r.llamadas.push([...e.argv])
    if (e.argv.includes('binario')) return { value: { exitCode: 1, stdout: '', stderr: '' } }
    if (e.argv.includes('panel')) return { value: { exitCode: 0, stdout: JSON.stringify({ formato: 1, carpeta: '/work', acciones: [], herramientas: [], flujos: r.flujos }), stderr: '' } }
    if (e.argv[1] === 'flujo' && e.argv[2] === 'ver') return { value: { exitCode: 0, stdout: JSON.stringify(dibujo(e.argv[3])), stderr: '' } }
    return { value: { exitCode: 0, stdout: '', stderr: '' } }
  })
}

const esperar = () => new Promise((res) => setTimeout(res, 20))
const ver = (r) => r.llamadas.filter((a) => a[1] === 'flujo' && a[2] === 'ver')
const valor = (argv: string[], flag: string) => (argv.includes(flag) ? argv[argv.indexOf(flag) + 1] : undefined)

async function abrir($, on, flujos: object[]) {
  const r = { llamadas: [] as string[][], flujos }
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...FLUJOS, surface: 'terminal' })
  await esperar()
  return { r, ui }
}

test('un flujo grande abre en el panorama y uno chico entero', async ($, on) => {
  const grande = await abrir($, on, [GRANDE])
  expect(valor(ver(grande.r).at(-1), '--nivel')).toBe('1')
  expect(await grande.ui.find({ key: 'fl-nivel-1', label: /Panorama/ })).toBeDefined()
})

test('en un flujo chico no cambia lo que se pide', async ($, on) => {
  const { r } = await abrir($, on, [CHICO])
  expect(ver(r).at(-1)).toEqual(['qdd', 'flujo', 'ver', 'pago', '--ancho', '60', '--json'])
})

test('1, 2 y 3 cambian el nivel del dibujo', async ($, on) => {
  const { r, ui } = await abrir($, on, [GRANDE])
  await ui.press({ key: 'fl-nivel-2' })
  await esperar()
  expect(valor(ver(r).at(-1), '--nivel')).toBe('2')
  await ui.press({ key: 'fl-nivel-3' })
  await esperar()
  expect(ver(r).at(-1)).not.toContain('--nivel')
})

test('en el panorama cada grupo se abre y se cierra solo', async ($, on) => {
  const { r, ui } = await abrir($, on, [GRANDE])
  await ui.press({ key: 'fl-grupo-filtrar' })
  await esperar()
  expect(valor(ver(r).at(-1), '--abrir')).toBe('Filtrar')
  expect(await ui.find({ key: 'fl-grupo-filtrar', label: /▾ Filtrar/ })).toBeDefined()
  await ui.press({ key: 'fl-grupo-filtrar' })
  await esperar()
  expect(ver(r).at(-1)).not.toContain('--abrir')
  await ui.press({ key: 'fl-nivel-2' })
  await esperar()
  expect(await ui.find({ key: 'fl-grupo-filtrar' })).toBeUndefined()
})

test('la tarjeta muestra el nivel y el grupo, y editarlos los guarda', async ($, on) => {
  const { r, ui } = await abrir($, on, [CHICO])
  await ui.press({ key: 'fl-n-rechazo' })
  await esperar()
  expect(await ui.find({ type: 'Text', text: /Nivel 3 · desvío \(calculado\) · grupo Filtrar/ })).toBeDefined()
  await ui.press({ key: 'fl-editar' })
  await ui.select({ key: 'fl-nivel', value: '2' })
  await ui.input({ key: 'fl-grupo', text: 'Revisar', kind: 'change' })
  await ui.press({ key: 'fl-guardar' })
  await esperar()
  const m = r.llamadas.find((a) => a[2] === 'modificar')
  expect(valor(m, '--nivel')).toBe('2')
  expect(valor(m, '--grupo')).toBe('Revisar')
})

// Pedido de la persona (2026-11-08): «dentro de los nodos tiene que existir un título visible […] debería estar solo
// el título primero y, si quiero ver el detalle, verlo». qdd dibuja cada nodo con su título; h pide el detalle.
test('cada nodo muestra su título y h trae o quita la descripción', async ($, on) => {
  const r = { llamadas: [] as string[][], flujos: [CHICO] }
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...FLUJOS, surface: 'terminal' })
  await esperar()
  expect(ver(r).at(-1)).not.toContain('--detalle')
  expect(await ui.find({ key: 'fl-ver-detalle', label: /☰ Ver detalle/ })).toMatchObject({ props: { hotkey: 'h' } })
  await ui.press({ key: 'fl-ver-detalle' })
  await esperar()
  expect(ver(r).at(-1)).toContain('--detalle')
  expect(await ui.find({ key: 'fl-ver-detalle', label: /▭ Solo títulos/ })).toBeDefined()
  await ui.press({ key: 'fl-ver-detalle' })
  await esperar()
  expect(ver(r).at(-1)).not.toContain('--detalle')
})
