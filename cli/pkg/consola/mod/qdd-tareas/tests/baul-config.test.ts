import { expect, mock, test } from 'claude-code/testing'

// El baúl de accesos en Config (pedido de la persona, 2026-11-09: «el baúl de contraseñas se debe poder administrar
// desde config en el panel de la derecha: agregar, quitar, modificar»). El valor entra por el portapapeles (eligió esa
// opción): el panel nunca lo ve, solo le pide a qdd que lo lea, lo pruebe, lo guarde y vacíe el portapapeles.
// Agregar es un asistente de una pregunta por pantalla (la persona eligió la opción B, 2026-11-20, tras «cuando escribo
// encima y termino de escribir aprieto enter no se entiende qué pasó»): Enter responde y pasa a la siguiente.

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
  // El kit no simula $.ui.focus: se ve que el panel pidió las teclas (ui.open con focus).
  on('ui.open', ($, e) => {
    r.aperturas.push(e)
    return { value: { isPlaced: true } }
  })
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
  aperturas: [] as any[],
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
const visible = async (ui, key) => (await ui.find({ key })) !== undefined

// responder: lo que la persona escribe en la pregunta y Enter.
const responder = (ui, campo, text) => ui.input({ key: 'b-' + campo, text })

// hastaElFinal: agregar con Cloudflare, el correo y la cuenta dados, sin descripción y en global.
async function hastaElFinal(ui, correo, cuenta) {
  await ui.press({ key: 'b-agregar' })
  await ui.select({ key: 'b-tipo', value: 'cloudflare' })
  await responder(ui, 'correo', correo)
  await responder(ui, 'cuenta', cuenta)
  await responder(ui, 'descripcion', '')
  await ui.select({ key: 'b-ambiente', value: 'global' })
}

test('Config lista los accesos con sitio, cuenta y correo, nunca un valor, y marca el rechazado', async ($, on) => {
  const r = nuevo()
  const ui = await abrirConfig($, on, r)
  expect(baul(r)).toContainEqual(['qdd', 'baul', '--json'])
  expect(await ui.find({ type: 'Text', text: 'cloudflare-trdago' })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /Cloudflare · trdago · yo@x\.cl · DNS · llavero/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /RECHAZADO/ })).toBeDefined()
  expect(await ui.find({ key: 'b-agregar' })).toBeDefined()
})

// Lo que se vio al guardar el token de Cloudflare (2026-11-20): «1 accesos», el ID de cuenta largo tapando el correo
// en la fila, y el resultado de qdd en una sola línea con « · [✓]» en medio.
test('un acceso se cuenta en singular y un ID de cuenta largo se acorta para que se vea el correo', async ($, on) => {
  const CF2 = { ...CF, nombre: 'cloudflare-chilefplay', cuenta: '02d7f74f9a0ecb1439218b0631cf0aaa', correo: 'chilefplay@gmail.com', descripcion: '' }
  const r = nuevo([CF2])
  const ui = await abrirConfig($, on, r)
  expect(await ui.find({ type: 'Text', text: '1 acceso' })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /^Cloudflare · 02d7f74f… · chilefplay@gmail\.com · llavero/ })).toBeDefined()
})

test('lo que responde qdd va una línea debajo de otra', async ($, on) => {
  const r = nuevo([])
  r.respuestas.guardar = ok('[✓] cloudflare-trdago quedó en el llavero.\n[✓] cloudflare-trdago: la API lo acepta · token active\n')
  const ui = await abrirConfig($, on, r)
  await hastaElFinal(ui, 'yo@x.cl', 'trdago')
  await ui.press({ key: 'b-pegar' })
  await esperar()
  expect(await ui.find({ type: 'Text', text: /quedó en el llavero\.\n\[✓\] cloudflare-trdago: la API lo acepta/ })).toBeDefined()
})

test('el baúl va primero en Config, antes que las herramientas', async ($, on) => {
  const r = nuevo()
  const ui = await abrirConfig($, on, r)
  const orden = await claves(ui)
  expect(orden.indexOf('b-agregar')).toBeGreaterThanOrEqual(0)
  expect(orden.indexOf('b-agregar')).toBeLessThan(orden.indexOf('h-agregar'))
})

// Pedido de la persona (2026-11-20): «que me preguntara siempre el correo … sitio cloudflare en este caso seleccionable
// y si es otro que me pida agregar la url, usuario o id de cuenta, pass o token, descripción opcional».
test('agregar va de a una pregunta y Enter pasa a la siguiente: sitio, correo, cuenta, descripción, ambiente, token', async ($, on) => {
  const r = nuevo([])
  r.respuestas.guardar = ok('[✓] cloudflare-trdago quedó en el llavero.\n')
  const ui = await abrirConfig($, on, r)
  expect(await ui.find({ type: 'Text', text: /El baúl está vacío/ })).toBeDefined()
  await ui.press({ key: 'b-agregar' })
  expect(await ui.find({ type: 'Text', text: /Agregar un acceso · 1 de 6/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /¿De qué sitio es el acceso\?/ })).toBeDefined()
  for (const k of ['b-correo', 'b-cuenta', 'b-descripcion', 'b-ambiente', 'b-pegar', 'b-nombre', 'b-url']) expect(await visible(ui, k)).toBe(false)

  await ui.select({ key: 'b-tipo', value: 'cloudflare' })
  expect(await visible(ui, 'b-tipo')).toBe(false)
  expect(await ui.find({ type: 'Text', text: /2 de 6/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /¿Con qué correo entras a Cloudflare\?/ })).toBeDefined()
  expect((await ui.find({ key: 'b-ir-tipo' })).props.label).toMatch(/✓ Sitio: Cloudflare/)

  await responder(ui, 'correo', 'yo@x.cl')
  expect(await ui.find({ type: 'Text', text: /¿Cuál es el ID de cuenta de Cloudflare\?/ })).toBeDefined()
  await responder(ui, 'cuenta', 'trdago')
  expect(await ui.find({ type: 'Text', text: /¿Para qué es\?/ })).toBeDefined()
  await responder(ui, 'descripcion', '')
  expect(await ui.find({ type: 'Text', text: /¿En qué ambiente se usa\?/ })).toBeDefined()
  await ui.select({ key: 'b-ambiente', value: 'global' })

  expect(await ui.find({ type: 'Text', text: /6 de 6/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /Se guarda como «cloudflare-trdago» en el llavero/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /Copia el token o la contraseña de Cloudflare/ })).toBeDefined()
  expect(guardados(r)).toEqual([])
  await ui.press({ key: 'b-pegar' })
  await esperar()
  expect(guardados(r)).toEqual([['qdd', 'baul', 'guardar', 'cloudflare-trdago', '--tipo', 'cloudflare', '--en', 'llavero', '--portapapeles', '--nuevo', '--correo', 'yo@x.cl', '--cuenta', 'trdago', '--ambiente', 'global']])
  expect(await ui.find({ type: 'Text', text: /cloudflare-trdago quedó en el llavero/ })).toBeDefined()
})

test('al abrir el asistente y tras cada Enter el panel pide las teclas para la pregunta a la vista', async ($, on) => {
  const r = nuevo([])
  const ui = await abrirConfig($, on, r)
  const alAbrir = r.aperturas.filter((e) => e.focus).length
  await ui.press({ key: 'b-agregar' })
  await esperar()
  expect(r.aperturas.filter((e) => e.focus).length).toBeGreaterThan(alAbrir)
  const antes = r.aperturas.filter((e) => e.focus).length
  await ui.select({ key: 'b-tipo', value: 'github' })
  await esperar()
  expect(r.aperturas.filter((e) => e.focus).length).toBeGreaterThan(antes)
})

// La persona (2026-11-20) escribió en Cuenta el ID de cuenta de Cloudflare asociado a su token: el nombre salía
// «cloudflare-02d7f74f9a0ecb1439218b0631cf0», ilegible. Un ID largo no sirve de nombre: va la parte del correo.
test('con un ID de cuenta largo el nombre sale del correo, y en Cloudflare la pregunta pide el ID de cuenta', async ($, on) => {
  const r = nuevo([])
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-agregar' })
  await ui.select({ key: 'b-tipo', value: 'cloudflare' })
  await responder(ui, 'correo', 'chilefplay@gmail.com')
  expect(await ui.find({ type: 'Text', text: /¿Cuál es el ID de cuenta de Cloudflare\?/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /32 caracteres/ })).toBeDefined()
  await responder(ui, 'cuenta', '02d7f74f9a0ecb1439218b0631cf0aaa')
  await responder(ui, 'descripcion', '')
  await ui.select({ key: 'b-ambiente', value: 'global' })
  expect(await ui.find({ type: 'Text', text: /«cloudflare-chilefplay»/ })).toBeDefined()
})

test('Enter con un dato que no vale se queda en la pregunta y dice por qué', async ($, on) => {
  const r = nuevo([])
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-agregar' })
  await ui.select({ key: 'b-tipo', value: 'cloudflare' })
  await responder(ui, 'correo', 'yo')
  expect(await ui.find({ type: 'Text', text: /«yo» no parece un correo/ })).toBeDefined()
  expect(await visible(ui, 'b-correo')).toBe(true)
  await responder(ui, 'correo', '')
  await responder(ui, 'cuenta', '')
  expect(await ui.find({ type: 'Text', text: /Falta el usuario o el correo/ })).toBeDefined()
  expect(await visible(ui, 'b-cuenta')).toBe(true)
  await responder(ui, 'cuenta', 'trdago')
  expect(await visible(ui, 'b-descripcion')).toBe(true)
})

test('otro sitio pide su URL y el nombre sale de su dominio, con el ambiente si no es global', async ($, on) => {
  const r = nuevo([])
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-agregar' })
  await ui.select({ key: 'b-tipo', value: 'otro' })
  expect(await ui.find({ type: 'Text', text: /2 de 7/ })).toBeDefined()
  await responder(ui, 'url', '')
  expect(await ui.find({ type: 'Text', text: /Falta la dirección del sitio/ })).toBeDefined()
  await responder(ui, 'url', 'https://app.sendgrid.com/settings')
  expect(await ui.find({ type: 'Text', text: /¿Con qué correo entras a app\.sendgrid\.com\?/ })).toBeDefined()
  await responder(ui, 'correo', '')
  expect(await ui.find({ type: 'Text', text: /¿Cuál es tu usuario o ID de cuenta en app\.sendgrid\.com\?/ })).toBeDefined()
  await responder(ui, 'cuenta', 'ventas')
  await responder(ui, 'descripcion', 'envíos del sitio')
  await ui.select({ key: 'b-ambiente', value: 'prod' })
  expect(await ui.find({ type: 'Text', text: /«sendgrid-ventas-prod»/ })).toBeDefined()
  await ui.press({ key: 'b-pegar' })
  await esperar()
  expect(guardados(r)).toEqual([['qdd', 'baul', 'guardar', 'sendgrid-ventas-prod', '--tipo', 'otro', '--en', 'llavero', '--portapapeles', '--nuevo', '--cuenta', 'ventas', '--descripcion', 'envíos del sitio', '--ambiente', 'prod', '--url', 'https://app.sendgrid.com/settings']])
})

test('lo ya respondido se puede cambiar: el resumen lleva a esa pregunta y Enter vuelve a donde ibas', async ($, on) => {
  const r = nuevo([])
  const ui = await abrirConfig($, on, r)
  await hastaElFinal(ui, 'yo@x.cl', 'trdago')
  expect(await visible(ui, 'b-pegar')).toBe(true)
  await ui.press({ key: 'b-ir-correo' })
  expect((await ui.find({ key: 'b-correo' })).props.value).toBe('yo@x.cl')
  await responder(ui, 'correo', 'otra@x.cl')
  expect(await visible(ui, 'b-pegar')).toBe(true)
  expect((await ui.find({ key: 'b-ir-correo' })).props.label).toMatch(/otra@x\.cl/)
  await ui.press({ key: 'b-atras' })
  expect(await visible(ui, 'b-ambiente')).toBe(true)
  await ui.press({ key: 'b-atras' })
  expect((await ui.find({ key: 'b-descripcion' })).props.value).toBe('')
})

test('cambiar a otro sitio desde el resumen pide la URL que faltaba antes de volver', async ($, on) => {
  const r = nuevo([])
  const ui = await abrirConfig($, on, r)
  await hastaElFinal(ui, 'yo@x.cl', 'trdago')
  await ui.press({ key: 'b-ir-tipo' })
  await ui.select({ key: 'b-tipo', value: 'otro' })
  expect(await visible(ui, 'b-url')).toBe(true)
  await responder(ui, 'url', 'mi-api.cl')
  expect(await visible(ui, 'b-pegar')).toBe(true)
})

test('agregar no pisa un acceso que ya existe: avisa y no corre nada', async ($, on) => {
  const r = nuevo()
  const ui = await abrirConfig($, on, r)
  await hastaElFinal(ui, '', 'trdago')
  await ui.press({ key: 'b-pegar' })
  expect(guardados(r)).toEqual([])
  expect(await ui.find({ type: 'Text', text: /ya tienes un acceso llamado «cloudflare-trdago»/ })).toBeDefined()
})

test('cancelar sale del asistente sin guardar nada', async ($, on) => {
  const r = nuevo([])
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-agregar' })
  await ui.select({ key: 'b-tipo', value: 'cloudflare' })
  await ui.press({ key: 'b-cancelar' })
  expect(await visible(ui, 'b-correo')).toBe(false)
  expect(await visible(ui, 'b-agregar')).toBe(true)
  expect(guardados(r)).toEqual([])
})

test('más opciones, al final: nombre, dónde, vencimiento y la variable de otro abren su pregunta y Enter vuelve', async ($, on) => {
  const r = nuevo([])
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-agregar' })
  await ui.select({ key: 'b-tipo', value: 'otro' })
  await responder(ui, 'url', 'mi-api.cl')
  await responder(ui, 'correo', 'yo@x.cl')
  await responder(ui, 'cuenta', '')
  await responder(ui, 'descripcion', '')
  await ui.select({ key: 'b-ambiente', value: 'global' })
  expect(await visible(ui, 'b-ir-nombre')).toBe(false)
  await ui.press({ key: 'b-mas' })
  expect((await ui.find({ key: 'b-ir-nombre' })).props.label).toBe('Nombre: mi-api-yo')
  expect((await ui.find({ key: 'b-ir-vence' })).props.label).toBe('Vence el: —')
  await ui.press({ key: 'b-ir-nombre' })
  expect((await ui.find({ key: 'b-nombre' })).props.value).toBe('mi-api-yo')
  await responder(ui, 'nombre', 'Mi API')
  expect(await ui.find({ type: 'Text', text: /«Mi API» no vale/ })).toBeDefined()
  await responder(ui, 'nombre', 'mi-api')
  expect(await visible(ui, 'b-pegar')).toBe(true)
  await ui.press({ key: 'b-ir-en' })
  await ui.select({ key: 'b-en', value: '1password' })
  await ui.press({ key: 'b-ir-variable' })
  await responder(ui, 'variable', 'MI_API_KEY')
  await ui.press({ key: 'b-ir-vence' })
  await responder(ui, 'vence', '31/01/2027')
  expect(await ui.find({ type: 'Text', text: /AAAA-MM-DD \(2027-01-31\)/ })).toBeDefined()
  await responder(ui, 'vence', '2027-01-31')
  expect((await ui.find({ key: 'b-ir-vence' })).props.label).toBe('Vence el: 2027-01-31')
  expect(await ui.find({ type: 'Text', text: /Se guarda como «mi-api» en el 1Password/ })).toBeDefined()
  await ui.press({ key: 'b-pegar' })
  await esperar()
  expect(guardados(r)).toEqual([['qdd', 'baul', 'guardar', 'mi-api', '--tipo', 'otro', '--en', '1password', '--portapapeles', '--nuevo', '--variable', 'MI_API_KEY', '--correo', 'yo@x.cl', '--ambiente', 'global', '--vence', '2027-01-31', '--url', 'mi-api.cl']])
})

test('desde una pregunta de más opciones, Atrás vuelve al final', async ($, on) => {
  const r = nuevo([])
  const ui = await abrirConfig($, on, r)
  await hastaElFinal(ui, 'yo@x.cl', 'trdago')
  await ui.press({ key: 'b-mas' })
  await ui.press({ key: 'b-ir-vence' })
  expect(await ui.find({ type: 'Text', text: /Agregar un acceso$/ })).toBeDefined()
  await ui.press({ key: 'b-atras' })
  expect(await visible(ui, 'b-pegar')).toBe(true)
})

test('modificar abre el resumen con todo respondido; cambiar un dato y «Guardar cambios» no pide el token', async ($, on) => {
  const r = nuevo()
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-modificar-cloudflare-trdago' })
  expect(await ui.find({ type: 'Text', text: /Modificar «cloudflare-trdago»/ })).toBeDefined()
  for (const [k, v] of [['tipo', /Cloudflare/], ['correo', /yo@x\.cl/], ['cuenta', /trdago/], ['descripcion', /DNS/], ['ambiente', /global/]]) expect((await ui.find({ key: 'b-ir-' + k })).props.label).toMatch(v)
  await ui.press({ key: 'b-ir-descripcion' })
  await responder(ui, 'descripcion', '')
  await ui.press({ key: 'b-ir-correo' })
  await responder(ui, 'correo', 'otra@x.cl')
  await ui.press({ key: 'b-guardar' })
  await esperar()
  expect(guardados(r)).toEqual([['qdd', 'baul', 'editar', 'cloudflare-trdago', '--correo', 'otra@x.cl', '--cuenta', 'trdago', '--descripcion', '', '--ambiente', 'global', '--vence', '']])
  expect(await ui.find({ key: 'b-guardar' })).toBeUndefined()
})

test('modificar: en el resumen no hay Atrás, y desde una pregunta Atrás vuelve al resumen', async ($, on) => {
  const r = nuevo()
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-modificar-cloudflare-trdago' })
  expect(await visible(ui, 'b-atras')).toBe(false)
  expect(await ui.find({ type: 'Text', text: /copia el nuevo token o contraseña de Cloudflare/ })).toBeDefined()
  await ui.press({ key: 'b-ir-cuenta' })
  await ui.press({ key: 'b-atras' })
  expect(await visible(ui, 'b-guardar')).toBe(true)
})

test('modificar: cambiar el nombre lo renombra y después guarda los datos', async ($, on) => {
  const r = nuevo()
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-modificar-cloudflare-trdago' })
  await ui.press({ key: 'b-mas' })
  await ui.press({ key: 'b-ir-nombre' })
  await responder(ui, 'nombre', 'cf-dns')
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
  await ui.press({ key: 'b-ir-nombre' })
  await responder(ui, 'nombre', 'gh-viejo')
  await ui.press({ key: 'b-pegar' })
  await esperar()
  expect(guardados(r).map((a) => a[2])).toEqual(['renombrar'])
  expect(await ui.find({ type: 'Text', text: /ya hay un token llamado gh-viejo/ })).toBeDefined()
})

test('modificar: «Pegar token nuevo» reemplaza el valor sin --nuevo, y cambiar el sitio pide el token', async ($, on) => {
  const r = nuevo()
  const ui = await abrirConfig($, on, r)
  await ui.press({ key: 'b-modificar-gh-viejo' })
  await ui.press({ key: 'b-ir-tipo' })
  await ui.select({ key: 'b-tipo', value: 'otro' })
  await responder(ui, 'url', 'gitlab.com')
  await ui.press({ key: 'b-guardar' })
  expect(guardados(r)).toEqual([])
  expect(await ui.find({ type: 'Text', text: /pide el token de nuevo/ })).toBeDefined()
  await ui.press({ key: 'b-ir-tipo' })
  await ui.select({ key: 'b-tipo', value: 'github' })
  await ui.press({ key: 'b-mas' })
  await ui.press({ key: 'b-ir-en' })
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

test('sin portapapeles el último paso dice cómo guardarlo en la terminal, con los datos escritos', async ($, on) => {
  const r = nuevo([], false)
  const ui = await abrirConfig($, on, r)
  await hastaElFinal(ui, '', 'trdago')
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
  const sitio = await ui.find({ key: 'b-tipo' })
  expect(sitio.props.options.map((o) => o.value)).toEqual(['cloudflare', 'github', 'supabase', 'resend', 'aws-sso', 'otro'])
  await ui.select({ key: 'b-tipo', value: 'github' })
  await responder(ui, 'correo', 'yo@x.cl')
  await responder(ui, 'cuenta', '')
  await responder(ui, 'descripcion', '')
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
  const PG = { nombre: 'pg-app', tipo: 'otro', donde: 'llavero', variable: 'PGPASSWORD', direccion: 'prod/db-prod-01/proyecto-b/postgres/app_user', guardado: '2026-11-13T00:00:00Z' }
  const r = nuevo([PG])
  const ui = await abrirConfig($, on, r)
  expect(await ui.find({ type: 'Text', text: /prod\/db-prod-01\/proyecto-b\/postgres\/app_user · llavero/ })).toBeDefined()
})
