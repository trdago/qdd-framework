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

const AMBIENTES = ['dev', 'qa', 'prod', 'global']

const nuevo = (tokens = [CF, GH], portapapeles = true) => ({
  llamadas: [] as string[][],
  lista: ok({ tokens, tipos: TIPOS, ambientes: AMBIENTES, portapapeles }),
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

// Pedido de la persona (2026-11-13): cada acceso con su dirección estándar ambiente/servidor/servicio/clase/cuenta, para que no
// se guarde dos veces lo mismo.
test('agregar lleva la dirección; la fila la muestra y modificar la conserva', async ($, on) => {
  const PG = { nombre: 'pg-app', tipo: 'otro', donde: 'llavero', variable: 'PGPASSWORD', direccion: 'prod/db-prod-01/notificador/postgres/app_user', guardado: '2026-11-13T00:00:00Z' }
  const r = nuevo([PG])
  const ui = await abrirConfig($, on, r)
  expect(await ui.find({ type: 'Text', text: /prod\/db-prod-01\/notificador\/postgres\/app_user · otro · llavero/ })).toBeDefined()
  await ui.press({ key: 'b-agregar' })
  await ui.input({ key: 'b-nombre', text: 'pg-admin', kind: 'change' })
  await ui.select({ key: 'b-ambiente', value: 'prod' })
  await ui.input({ key: 'b-servidor', text: 'db-prod-01', kind: 'change' })
  await ui.input({ key: 'b-servicio', text: 'notificador', kind: 'change' })
  await ui.input({ key: 'b-clase', text: 'postgres', kind: 'change' })
  await ui.input({ key: 'b-cuenta', text: 'admin', kind: 'change' })
  await ui.select({ key: 'b-tipo', value: 'otro' })
  await ui.input({ key: 'b-variable', text: 'PGPASSWORD', kind: 'change' })
  await ui.press({ key: 'b-pegar' })
  await esperar()
  expect(baul(r)).toContainEqual(['qdd', 'baul', 'guardar', 'pg-admin', '--tipo', 'otro', '--en', 'llavero', '--portapapeles', '--variable', 'PGPASSWORD', '--direccion', 'prod/db-prod-01/notificador/postgres/admin'])
  await ui.press({ key: 'b-modificar-pg-app' })
  await ui.press({ key: 'b-pegar' })
  await esperar()
  expect(baul(r)).toContainEqual(['qdd', 'baul', 'guardar', 'pg-app', '--tipo', 'otro', '--en', 'llavero', '--portapapeles', '--variable', 'PGPASSWORD', '--direccion', 'prod/db-prod-01/notificador/postgres/app_user'])
})

// Pedido de la persona (2026-11-20): «en config debería estar el vault y permitir ingresar de forma fácil y de forma
// estándar las keys al vault y gestionarlas, o sea quitar, agregar, modificar los nombres».
const claves = async (ui) => (await ui.findAll({ type: 'Button' })).map((b) => b.props.key ?? b.key)

test('el baúl va primero en Config, antes que las herramientas', async ($, on) => {
  const r = nuevo()
  const ui = await abrirConfig($, on, r)
  const orden = await claves(ui)
  expect(orden.indexOf('b-agregar')).toBeGreaterThanOrEqual(0)
  expect(orden.indexOf('b-agregar')).toBeLessThan(orden.indexOf('h-agregar'))
})

test('modificar cambia el nombre sin pedir el valor de nuevo', async ($, on) => {
  const r = nuevo()
  r.respuestas.renombrar = ok('[✓] cloudflare ahora se llama cf-prod.\n')
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-modificar-cloudflare' })
  expect(await ui.find({ key: 'b-renombrar' })).toBeUndefined()
  await ui.input({ key: 'b-nombre', text: 'cf-prod', kind: 'change' })
  await ui.press({ key: 'b-renombrar' })
  await esperar()
  expect(baul(r)).toContainEqual(['qdd', 'baul', 'renombrar', 'cloudflare', 'cf-prod'])
  expect(baul(r).some((a) => a[2] === 'guardar')).toBe(false)
  expect(await ui.find({ type: 'Text', text: /ahora se llama cf-prod/ })).toBeDefined()
  expect(await ui.find({ key: 'b-pegar' })).toBeUndefined()
})

test('cambiar el nombre y pegar un valor nuevo renombra primero y guarda con el nombre nuevo', async ($, on) => {
  const r = nuevo()
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-modificar-cloudflare' })
  await ui.input({ key: 'b-nombre', text: 'cf-prod', kind: 'change' })
  await ui.press({ key: 'b-pegar' })
  await esperar()
  const hechos = baul(r).filter((a) => a[2] === 'renombrar' || a[2] === 'guardar')
  expect(hechos).toEqual([
    ['qdd', 'baul', 'renombrar', 'cloudflare', 'cf-prod'],
    ['qdd', 'baul', 'guardar', 'cf-prod', '--tipo', 'cloudflare', '--en', 'llavero', '--portapapeles'],
  ])
})

test('si qdd no pudo renombrar, no guarda el valor con el nombre nuevo', async ($, on) => {
  const r = nuevo()
  r.respuestas.renombrar = { exitCode: 1, stdout: '', stderr: 'ya hay un token llamado gh-viejo: quítalo o elige otro nombre' }
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-modificar-cloudflare' })
  await ui.input({ key: 'b-nombre', text: 'gh-viejo', kind: 'change' })
  await ui.press({ key: 'b-pegar' })
  await esperar()
  expect(baul(r).some((a) => a[2] === 'guardar')).toBe(false)
  expect(await ui.find({ type: 'Text', text: /ya hay un token llamado gh-viejo/ })).toBeDefined()
})

test('la dirección se arma con campos: ambiente de la lista y las demás partes, todas o ninguna', async ($, on) => {
  const r = nuevo([])
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-agregar' })
  expect(await ui.find({ key: 'b-direccion' })).toBeUndefined()
  expect(await ui.find({ key: 'b-servidor' })).toBeUndefined()
  await ui.input({ key: 'b-nombre', text: 'gh-ci', kind: 'change' })
  await ui.select({ key: 'b-tipo', value: 'github' })
  await ui.select({ key: 'b-ambiente', value: 'global' })
  await ui.input({ key: 'b-servidor', text: 'github.com', kind: 'change' })
  await ui.press({ key: 'b-pegar' })
  expect(baul(r).some((a) => a[2] === 'guardar')).toBe(false)
  expect(await ui.find({ type: 'Text', text: /falta servicio, clase y cuenta/ })).toBeDefined()
  await ui.input({ key: 'b-servicio', text: 'ci', kind: 'change' })
  await ui.input({ key: 'b-clase', text: 'token', kind: 'change' })
  await ui.input({ key: 'b-cuenta', text: 'trdago', kind: 'change' })
  await ui.press({ key: 'b-pegar' })
  await esperar()
  expect(baul(r)).toContainEqual(['qdd', 'baul', 'guardar', 'gh-ci', '--tipo', 'github', '--en', 'llavero', '--portapapeles', '--direccion', 'global/github.com/ci/token/trdago'])
})

test('un qdd anterior, sin ambientes en la lista, igual ofrece los cuatro', async ($, on) => {
  const r = nuevo([])
  r.lista = ok({ tokens: [], tipos: TIPOS, portapapeles: true })
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-agregar' })
  const ambiente = await ui.find({ key: 'b-ambiente' })
  expect(ambiente.props.options.map((o) => o.value)).toEqual(['', 'dev', 'qa', 'prod', 'global'])
})
