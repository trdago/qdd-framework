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

const CF = { nombre: 'cloudflare-trdago', tipo: 'cloudflare', donde: 'llavero', variable: 'CLOUDFLARE_API_TOKEN', correo: 'yo@x.cl', cuenta: 'trdago', descripcion: 'DNS', ambiente: 'global', duenio: 'yo@x.cl', alcance: 'Workers', guardado: '2026-11-08T00:00:00Z' }
const GH = { nombre: 'gh-viejo', tipo: 'github', donde: '1password', variable: 'GITHUB_TOKEN', rechazado: true, guardado: '2026-10-01T00:00:00Z' }
const TIPOS = ['aws-sso', 'cloudflare', 'github', 'otro', 'resend', 'supabase']
const SITIOS = [
  { tipo: 'cloudflare', etiqueta: 'Cloudflare', host: 'cloudflare.com' },
  { tipo: 'github', etiqueta: 'GitHub', host: 'github.com' },
  { tipo: 'supabase', etiqueta: 'Supabase', host: 'supabase.com' },
  { tipo: 'resend', etiqueta: 'Resend', host: 'resend.com' },
  { tipo: 'aws-sso', etiqueta: 'AWS (perfil de SSO)', host: 'aws.amazon.com' },
]

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
  lista: ok({ tokens, tipos: TIPOS, ambientes: AMBIENTES, sitios: SITIOS, portapapeles }),
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

const claves = async (ui) => (await ui.findAll({ type: 'Button' })).map((b) => b.props.key ?? b.key)
const guardados = (r) => baul(r).filter((a) => a[2] === 'guardar' || a[2] === 'editar' || a[2] === 'renombrar')

test('Config lista los accesos con sitio, cuenta y correo, nunca un valor, y marca el rechazado', async ($, on) => {
  const r = nuevo()
  const ui = await abrirConfig($, on, r)
  expect(baul(r)).toContainEqual(['qdd', 'baul', '--json'])
  expect(await ui.find({ type: 'Text', text: 'cloudflare-trdago' })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /Cloudflare · trdago · yo@x\.cl · DNS · llavero/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /RECHAZADO/ })).toBeDefined()
  expect(await ui.find({ key: 'b-agregar' })).toBeDefined()
})

// Pedido de la persona (2026-11-20): «que me preguntara siempre el correo … sitio cloudflare en este caso seleccionable
// y si es otro que me pida agregar la url, usuario o id de cuenta, pass o token, descripción opcional».
test('el baúl va primero en Config, antes que las herramientas', async ($, on) => {
  const r = nuevo()
  const ui = await abrirConfig($, on, r)
  const orden = await claves(ui)
  expect(orden.indexOf('b-agregar')).toBeGreaterThanOrEqual(0)
  expect(orden.indexOf('b-agregar')).toBeLessThan(orden.indexOf('h-agregar'))
})

test('agregar pregunta correo, sitio, cuenta, descripción y ambiente; el nombre se arma solo', async ($, on) => {
  const r = nuevo([])
  r.respuestas.guardar = ok('[✓] cloudflare-trdago quedó en el llavero.\n')
  const ui = await abrirConfig($, on, r)
  expect(await ui.find({ type: 'Text', text: /El baúl está vacío/ })).toBeDefined()
  await ui.press({ key: 'b-agregar' })
  for (const k of ['b-correo', 'b-sitio', 'b-cuenta', 'b-descripcion', 'b-ambiente']) expect(await ui.find({ key: k })).toBeDefined()
  for (const k of ['b-nombre', 'b-en', 'b-direccion', 'b-tipo', 'b-url']) expect(await ui.find({ key: k })).toBeUndefined()
  await ui.press({ key: 'b-pegar' })
  expect(guardados(r)).toEqual([])
  expect(await ui.find({ type: 'Text', text: /falta el usuario o el correo/ })).toBeDefined()
  await ui.input({ key: 'b-correo', text: 'yo@x.cl', kind: 'change' })
  await ui.input({ key: 'b-cuenta', text: 'trdago', kind: 'change' })
  expect(await ui.find({ type: 'Text', text: /Se guarda como «cloudflare-trdago» en el llavero/ })).toBeDefined()
  await ui.press({ key: 'b-pegar' })
  await esperar()
  expect(guardados(r)).toEqual([['qdd', 'baul', 'guardar', 'cloudflare-trdago', '--tipo', 'cloudflare', '--en', 'llavero', '--portapapeles', '--nuevo', '--correo', 'yo@x.cl', '--cuenta', 'trdago', '--ambiente', 'global']])
  expect(await ui.find({ type: 'Text', text: /cloudflare-trdago quedó en el llavero/ })).toBeDefined()
})

test('otro sitio pide su URL y el nombre sale de su dominio, con el ambiente si no es global', async ($, on) => {
  const r = nuevo([])
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-agregar' })
  await ui.select({ key: 'b-sitio', value: 'otro' })
  await ui.input({ key: 'b-cuenta', text: 'ventas', kind: 'change' })
  await ui.press({ key: 'b-pegar' })
  expect(await ui.find({ type: 'Text', text: /falta la URL del sitio/ })).toBeDefined()
  await ui.input({ key: 'b-url', text: 'https://app.sendgrid.com/settings', kind: 'change' })
  await ui.input({ key: 'b-descripcion', text: 'envíos del sitio', kind: 'change' })
  await ui.select({ key: 'b-ambiente', value: 'prod' })
  expect(await ui.find({ type: 'Text', text: /«sendgrid-ventas-prod»/ })).toBeDefined()
  await ui.press({ key: 'b-pegar' })
  await esperar()
  expect(guardados(r)).toEqual([['qdd', 'baul', 'guardar', 'sendgrid-ventas-prod', '--tipo', 'otro', '--en', 'llavero', '--portapapeles', '--nuevo', '--cuenta', 'ventas', '--descripcion', 'envíos del sitio', '--ambiente', 'prod', '--url', 'https://app.sendgrid.com/settings']])
})

test('agregar no pisa un acceso que ya existe: avisa y no corre nada', async ($, on) => {
  const r = nuevo()
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-agregar' })
  await ui.input({ key: 'b-cuenta', text: 'trdago', kind: 'change' })
  await ui.press({ key: 'b-pegar' })
  expect(guardados(r)).toEqual([])
  expect(await ui.find({ type: 'Text', text: /ya tienes un acceso llamado «cloudflare-trdago»/ })).toBeDefined()
})

test('más opciones: el nombre, dónde se guarda y la variable de otro', async ($, on) => {
  const r = nuevo([])
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-agregar' })
  await ui.select({ key: 'b-sitio', value: 'otro' })
  await ui.input({ key: 'b-url', text: 'mi-api.cl', kind: 'change' })
  await ui.input({ key: 'b-correo', text: 'yo@x.cl', kind: 'change' })
  await ui.press({ key: 'b-mas' })
  await ui.input({ key: 'b-nombre', text: 'mi-api', kind: 'change' })
  await ui.select({ key: 'b-en', value: '1password' })
  await ui.input({ key: 'b-variable', text: 'MI_API_KEY', kind: 'change' })
  await ui.input({ key: 'b-vence', text: '31/01/2027', kind: 'change' })
  await ui.press({ key: 'b-pegar' })
  expect(await ui.find({ type: 'Text', text: /AAAA-MM-DD/ })).toBeDefined()
  expect(guardados(r)).toEqual([])
  await ui.input({ key: 'b-vence', text: '2027-01-31', kind: 'change' })
  await ui.press({ key: 'b-pegar' })
  await esperar()
  expect(guardados(r)).toEqual([['qdd', 'baul', 'guardar', 'mi-api', '--tipo', 'otro', '--en', '1password', '--portapapeles', '--nuevo', '--variable', 'MI_API_KEY', '--correo', 'yo@x.cl', '--ambiente', 'global', '--vence', '2027-01-31', '--url', 'mi-api.cl']])
})

test('modificar: «Guardar cambios» guarda los datos sin pedir el token, y borrar uno lo borra', async ($, on) => {
  const r = nuevo()
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-modificar-cloudflare-trdago' })
  expect(await ui.find({ type: 'Text', text: /Modificar «cloudflare-trdago»/ })).toBeDefined()
  await ui.input({ key: 'b-descripcion', text: '', kind: 'change' })
  await ui.input({ key: 'b-correo', text: 'otra@x.cl', kind: 'change' })
  await ui.press({ key: 'b-guardar' })
  await esperar()
  expect(guardados(r)).toEqual([['qdd', 'baul', 'editar', 'cloudflare-trdago', '--correo', 'otra@x.cl', '--cuenta', 'trdago', '--descripcion', '', '--ambiente', 'global', '--vence', '']])
  expect(await ui.find({ key: 'b-guardar' })).toBeUndefined()
})

test('modificar: cambiar el nombre lo renombra y después guarda los datos', async ($, on) => {
  const r = nuevo()
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-modificar-cloudflare-trdago' })
  await ui.press({ key: 'b-mas' })
  await ui.input({ key: 'b-nombre', text: 'cf-dns', kind: 'change' })
  await ui.press({ key: 'b-guardar' })
  await esperar()
  expect(guardados(r)).toEqual([
    ['qdd', 'baul', 'renombrar', 'cloudflare-trdago', 'cf-dns'],
    ['qdd', 'baul', 'editar', 'cf-dns', '--correo', 'yo@x.cl', '--cuenta', 'trdago', '--descripcion', 'DNS', '--ambiente', 'global', '--vence', ''],
  ])
})

test('si qdd no pudo renombrar, no sigue', async ($, on) => {
  const r = nuevo()
  r.respuestas.renombrar = { exitCode: 1, stdout: '', stderr: 'ya hay un token llamado gh-viejo: quítalo o elige otro nombre' }
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-modificar-cloudflare-trdago' })
  await ui.press({ key: 'b-mas' })
  await ui.input({ key: 'b-nombre', text: 'gh-viejo', kind: 'change' })
  await ui.press({ key: 'b-pegar' })
  await esperar()
  expect(guardados(r).map((a) => a[2])).toEqual(['renombrar'])
  expect(await ui.find({ type: 'Text', text: /ya hay un token llamado gh-viejo/ })).toBeDefined()
})

test('modificar: «Pegar token nuevo» reemplaza el valor sin --nuevo, y cambiar el sitio pide el token', async ($, on) => {
  const r = nuevo()
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-modificar-gh-viejo' })
  await ui.select({ key: 'b-sitio', value: 'otro' })
  await ui.input({ key: 'b-url', text: 'gitlab.com', kind: 'change' })
  await ui.press({ key: 'b-guardar' })
  expect(guardados(r)).toEqual([])
  expect(await ui.find({ type: 'Text', text: /pide el token de nuevo/ })).toBeDefined()
  await ui.select({ key: 'b-sitio', value: 'github' })
  await ui.press({ key: 'b-mas' })
  await ui.select({ key: 'b-en', value: 'llavero' })
  await ui.press({ key: 'b-pegar' })
  await esperar()
  expect(guardados(r)).toEqual([['qdd', 'baul', 'guardar', 'gh-viejo', '--tipo', 'github', '--en', 'llavero', '--portapapeles', '--ambiente', 'global']])
})

test('quitar pregunta antes y borra con --si; verificar prueba el token', async ($, on) => {
  const r = nuevo()
  r.respuestas.verificar = ok('[✓] cloudflare-trdago: dueño yo@x.cl · Workers · vence -\n')
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-verificar-cloudflare-trdago' })
  await esperar()
  expect(baul(r)).toContainEqual(['qdd', 'baul', 'verificar', 'cloudflare-trdago'])
  expect(await ui.find({ type: 'Text', text: /dueño yo@x\.cl/ })).toBeDefined()
  await ui.press({ key: 'b-quitar-cloudflare-trdago' })
  expect(await ui.find({ type: 'Text', text: /¿Quitar «cloudflare-trdago» del baúl\?/ })).toBeDefined()
  await ui.press({ key: 'b-no' })
  expect(baul(r).some((a) => a[2] === 'quitar')).toBe(false)
  await ui.press({ key: 'b-quitar-cloudflare-trdago' })
  await ui.press({ key: 'b-si' })
  await esperar()
  expect(baul(r)).toContainEqual(['qdd', 'baul', 'quitar', 'cloudflare-trdago', '--si'])
})

test('sin portapapeles dice cómo guardarlo en la terminal, con los datos escritos', async ($, on) => {
  const r = nuevo([], false)
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-agregar' })
  await ui.input({ key: 'b-cuenta', text: 'trdago', kind: 'change' })
  expect(await ui.find({ key: 'b-pegar' })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: /qdd baul guardar cloudflare-trdago --tipo cloudflare --cuenta trdago --ambiente global/ })).toBeDefined()
})

test('un qdd sin el baúl en el panel no rompe Config', async ($, on) => {
  const r = nuevo()
  r.lista = { exitCode: 1, stdout: '', stderr: 'unknown flag: --json' }
  const ui = await abrirConfig($, on, r)
  expect(await ui.find({ type: 'Text', text: /actualiza qdd/ })).toBeDefined()
  expect(await ui.find({ key: 'h-agregar' })).toBeDefined()
})

test('un qdd anterior, sin sitios ni ambientes en la lista, igual ofrece los de siempre', async ($, on) => {
  const r = nuevo([])
  r.lista = ok({ tokens: [], tipos: TIPOS, portapapeles: true })
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-agregar' })
  const sitio = await ui.find({ key: 'b-sitio' })
  expect(sitio.props.options.map((o) => o.value)).toEqual(['cloudflare', 'github', 'supabase', 'resend', 'aws-sso', 'otro'])
  const ambiente = await ui.find({ key: 'b-ambiente' })
  expect(ambiente.props.options.map((o) => o.value)).toEqual(['dev', 'qa', 'prod', 'global'])
})

test('el vencimiento que declaró la persona se ve en la fila y vuelve al modificar', async ($, on) => {
  const MIO = { nombre: 'mi-api', tipo: 'otro', donde: 'llavero', variable: 'MI_API_TOKEN', url: 'https://mi-api.cl', cuenta: 'yo', vence_declarado: '2027-01-31T00:00:00-03:00', guardado: '2026-11-20T00:00:00Z' }
  const r = nuevo([MIO])
  const ui = await abrirConfig($, on, r)
  expect(await ui.find({ type: 'Text', text: /mi-api\.cl · yo · llavero · vence 2027-01-31/ })).toBeDefined()
  await ui.press({ key: 'b-modificar-mi-api' })
  await ui.press({ key: 'b-guardar' })
  await esperar()
  expect(guardados(r)).toEqual([['qdd', 'baul', 'editar', 'mi-api', '--correo', '', '--cuenta', 'yo', '--descripcion', '', '--ambiente', 'global', '--vence', '2027-01-31', '--url', 'https://mi-api.cl']])
})

test('una entrada con la dirección de antes la sigue mostrando', async ($, on) => {
  const PG = { nombre: 'pg-app', tipo: 'otro', donde: 'llavero', variable: 'PGPASSWORD', direccion: 'prod/db-prod-01/notificador/postgres/app_user', guardado: '2026-11-13T00:00:00Z' }
  const r = nuevo([PG])
  const ui = await abrirConfig($, on, r)
  expect(await ui.find({ type: 'Text', text: /prod\/db-prod-01\/notificador\/postgres\/app_user · llavero/ })).toBeDefined()
})
