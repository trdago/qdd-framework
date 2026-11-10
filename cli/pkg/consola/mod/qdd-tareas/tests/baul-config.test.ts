import { expect, mock, test } from 'claude-code/testing'

// El baúl de accesos en Config (pedido de la persona, 2026-11-09: «el baúl de contraseñas se debe poder administrar
// desde config en el panel de la derecha: agregar, quitar, modificar»). El valor entra por el portapapeles (eligió esa
// opción): el panel nunca lo ve, solo le pide a qdd que lo lea, lo pruebe, lo guarde y vacíe el portapapeles.

const TAREAS = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Tareas', isFocused: true, bodyColumns: 70, placement: 'dock', scroll: { offset: 0, bodyRows: 40 }, view: {} },
} as const

const PANEL = {
  formato: 1,
  carpeta: '/work/bot',
  estado: { formato: 1, resumen: '', etapas: ['Código'], abreviaturas: ['Có'], frentes: [], esperas: [], fallas: [] },
  acciones: [],
}

const CF = { nombre: 'cloudflare', tipo: 'cloudflare', donde: 'llavero', variable: 'CLOUDFLARE_API_TOKEN', duenio: 'yo@x.cl', alcance: 'Workers', guardado: '2026-11-08T00:00:00Z' }
const GH = { nombre: 'gh-viejo', tipo: 'github', donde: '1password', variable: 'GITHUB_TOKEN', rechazado: true, guardado: '2026-10-01T00:00:00Z' }
const TIPOS = ['aws-sso', 'cloudflare', 'github', 'otro', 'resend', 'supabase']

type Salida = { exitCode: number; stdout: string; stderr: string }
const ok = (d: unknown): Salida => ({ exitCode: 0, stdout: typeof d === 'string' ? d : JSON.stringify(d), stderr: '' })

function preparar(on, r) {
  mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work/bot' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('ui.close', () => ({ value: undefined }))
  on('process.run', ($, e) => {
    const argv = [...e.argv]
    r.llamadas.push(argv)
    if (argv[1] === 'baul' && argv[2] === '--json') return { value: r.lista }
    if (argv[1] === 'baul') return { value: r.respuestas[argv[2]] ?? ok('') }
    if (argv[1] === 'herramienta') return { value: ok({ herramientas: [], problemas: null }) }
    return { value: ok(PANEL) }
  })
}

const nuevo = (tokens = [CF, GH], portapapeles = true) => ({
  llamadas: [] as string[][],
  lista: ok({ tokens, tipos: TIPOS, portapapeles }),
  respuestas: {} as Record<string, Salida>,
})
const esperar = () => new Promise((res) => setTimeout(res, 20))
const baul = (r) => r.llamadas.filter((a) => a[1] === 'baul')

async function abrirConfig($, on, r) {
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work/bot' })
  const ui = await $.ui.mount({ ...TAREAS, surface: 'terminal' })
  await ui.press({ key: 'config' })
  await esperar()
  return ui
}

test('Config lista los tokens del baúl, nunca un valor, y marca el rechazado', async ($, on) => {
  const r = nuevo()
  const ui = await abrirConfig($, on, r)
  expect(baul(r)).toContainEqual(['qdd', 'baul', '--json'])
  expect(await ui.find({ type: 'Text', text: 'cloudflare' })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /cloudflare · llavero · yo@x\.cl/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /RECHAZADO/ })).toBeDefined()
  expect(await ui.find({ key: 'b-agregar' })).toBeDefined()
})

test('agregar: nombre, tipo y lugar, y qdd lee el token del portapapeles', async ($, on) => {
  const r = nuevo([])
  r.respuestas.guardar = ok('[✓] resend quedó en el llavero.\n')
  const ui = await abrirConfig($, on, r)
  expect(await ui.find({ type: 'Text', text: /El baúl está vacío/ })).toBeDefined()
  await ui.press({ key: 'b-agregar' })
  expect(await ui.find({ type: 'Text', text: /Copia el token/ })).toBeDefined()
  await ui.press({ key: 'b-pegar' })
  expect(baul(r).some((a) => a[2] === 'guardar')).toBe(false)
  expect(await ui.find({ type: 'Text', text: /falta el nombre/ })).toBeDefined()
  await ui.input({ key: 'b-nombre', text: 'resend', kind: 'change' })
  await ui.select({ key: 'b-tipo', value: 'resend' })
  await ui.press({ key: 'b-pegar' })
  await esperar()
  expect(baul(r)).toContainEqual(['qdd', 'baul', 'guardar', 'resend', '--tipo', 'resend', '--en', 'llavero', '--portapapeles'])
  expect(await ui.find({ type: 'Text', text: /resend quedó en el llavero/ })).toBeDefined()
  expect(baul(r).filter((a) => a[2] === '--json').length).toBe(2)
})

test('otro pide su variable', async ($, on) => {
  const r = nuevo([])
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-agregar' })
  await ui.input({ key: 'b-nombre', text: 'mio', kind: 'change' })
  await ui.select({ key: 'b-tipo', value: 'otro' })
  await ui.press({ key: 'b-pegar' })
  expect(await ui.find({ type: 'Text', text: /falta la variable/ })).toBeDefined()
  await ui.input({ key: 'b-variable', text: 'MI_TOKEN', kind: 'change' })
  await ui.press({ key: 'b-pegar' })
  await esperar()
  expect(baul(r)).toContainEqual(['qdd', 'baul', 'guardar', 'mio', '--tipo', 'otro', '--en', 'llavero', '--portapapeles', '--variable', 'MI_TOKEN'])
})

test('modificar reemplaza el valor con lo copiado, y puede cambiarlo de lugar', async ($, on) => {
  const r = nuevo()
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-modificar-gh-viejo' })
  expect(await ui.find({ key: 'b-nombre' })).toBeUndefined()
  await ui.select({ key: 'b-en', value: 'llavero' })
  await ui.press({ key: 'b-pegar' })
  await esperar()
  expect(baul(r)).toContainEqual(['qdd', 'baul', 'guardar', 'gh-viejo', '--tipo', 'github', '--en', 'llavero', '--portapapeles'])
})

test('quitar pregunta antes y borra con --si; verificar prueba el token', async ($, on) => {
  const r = nuevo()
  r.respuestas.verificar = ok('[✓] cloudflare: dueño yo@x.cl · Workers · vence -\n')
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-verificar-cloudflare' })
  await esperar()
  expect(baul(r)).toContainEqual(['qdd', 'baul', 'verificar', 'cloudflare'])
  expect(await ui.find({ type: 'Text', text: /dueño yo@x\.cl/ })).toBeDefined()
  await ui.press({ key: 'b-quitar-cloudflare' })
  expect(await ui.find({ type: 'Text', text: /¿Quitar «cloudflare» del baúl\?/ })).toBeDefined()
  await ui.press({ key: 'b-no' })
  expect(baul(r).some((a) => a[2] === 'quitar')).toBe(false)
  await ui.press({ key: 'b-quitar-cloudflare' })
  await ui.press({ key: 'b-si' })
  await esperar()
  expect(baul(r)).toContainEqual(['qdd', 'baul', 'quitar', 'cloudflare', '--si'])
})

test('lo que qdd rechaza se ve, y sin portapapeles dice cómo guardarlo en la terminal', async ($, on) => {
  const r = nuevo([], false)
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-agregar' })
  expect(await ui.find({ key: 'b-pegar' })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: /qdd baul guardar/ })).toBeDefined()
})

test('un qdd sin el baúl en el panel no rompe Config', async ($, on) => {
  const r = nuevo()
  r.lista = { exitCode: 1, stdout: '', stderr: 'unknown flag: --json' }
  const ui = await abrirConfig($, on, r)
  expect(await ui.find({ type: 'Text', text: /actualiza qdd/ })).toBeDefined()
  expect(await ui.find({ key: 'h-agregar' })).toBeDefined()
})
