import { expect, mock, test } from 'claude-code/testing'

// El panel recibe los cambios de un solo proceso, 'qdd consola seguir', en vez de lanzar 'qdd consola panel' cada 2 s
// (MEJ-020: 2.280 procesos por hora por sesión aunque nada cambie; RFC 0016).

const PANE = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Tareas', isFocused: true, bodyColumns: 60, placement: 'dock', scroll: { offset: 0, bodyRows: 30 }, viewportRows: 30 },
} as const

function panel(nombre: string) {
  return JSON.stringify({ formato: 1, carpeta: '/work', acciones: [], estado: { formato: 1, resumen: '', etapas: ['Código'], abreviaturas: ['Có'], frentes: [{ id: 'x', nombre, estado: 'abierto', simbolos: ['·'], pregunta: '' }], ocultos: 0, tareas: 1, esperan: 0, fallan: 0, esperas: [], fallas: [] } })
}

function preparar(on, r) {
  const clock = mock.clock(on)
  mock.env(on, r.env || {})
  on('session.start', () => ({ cwd: '/work' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('process.run', ($, e) => {
    r.llamadas.push([...e.argv])
    if (e.argv.includes('binario')) return { value: { exitCode: r.binario ? 0 : 1, stdout: (r.binario || '') + '\n', stderr: '' } }
    if (e.argv.includes('panel') && r.lectura) return r.lectura
    if (e.argv.includes('panel')) return { value: { exitCode: 0, stdout: panel('leído a mano'), stderr: '' } }
    return { value: { exitCode: 0, stdout: '{}', stderr: '' } }
  })
  return clock
}

const lecturas = (r) => r.llamadas.filter((a) => a.includes('panel')).length

test('con el flujo abierto los cambios llegan solos y no se lanza un proceso por lectura', async ($, on) => {
  const r = { llamadas: [] as string[][], lanzados: [] as string[][] }
  const clock = preparar(on, r)
  let soltar: () => void = () => {}
  const espera = new Promise<void>((res) => (soltar = res))
  on('process.spawn', async function* ($, e) {
    r.lanzados.push([...e.argv])
    yield { stream: 'stdout', text: panel('primero') + '\n' }
    await espera
    yield { stream: 'stdout', text: panel('segundo') + '\n' }
    await new Promise(() => {})
    return { value: { code: 0, signal: null } }
  })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await clock.advance(2000)
  await new Promise((res) => setTimeout(res, 20))
  expect(r.lanzados).toEqual([['qdd', 'consola', 'seguir', '--con', 'atender,herramientas,mejoras,flujos,esperas_persona,telefono:sesion-1']])
  expect(await ui.find({ type: 'Text', text: /primero/ })).toBeDefined()

  const antes = lecturas(r)
  await clock.advance(60000)
  expect(lecturas(r)).toBe(antes)
  expect(r.lanzados.length).toBe(1)

  soltar()
  await new Promise((res) => setTimeout(res, 20))
  expect(await ui.find({ type: 'Text', text: /segundo/ })).toBeDefined()
})

test('una línea partida en dos trozos se lee entera', async ($, on) => {
  const r = { llamadas: [] as string[][] }
  const clock = preparar(on, r)
  const linea = panel('partida') + '\n'
  on('process.spawn', async function* () {
    yield { stream: 'stdout', text: linea.slice(0, 40) }
    yield { stream: 'stdout', text: linea.slice(40) }
    await new Promise(() => {})
    return { value: { code: 0, signal: null } }
  })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await clock.advance(2000)
  await new Promise((res) => setTimeout(res, 20))
  expect(await ui.find({ type: 'Text', text: /partida/ })).toBeDefined()
})

test('un qdd sin «consola seguir» sigue leyendo como siempre y no reintenta el flujo', async ($, on) => {
  const r = { llamadas: [] as string[][], lanzados: [] as string[][] }
  const clock = preparar(on, r)
  on('process.spawn', async function* ($, e) {
    r.lanzados.push([...e.argv])
    yield { stream: 'stderr', text: 'Error: unknown command "seguir"\n' }
    return { value: { code: 1, signal: null } }
  })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await clock.advance(2000)
  await new Promise((res) => setTimeout(res, 20))
  const antes = lecturas(r)
  await clock.advance(10000)
  expect(lecturas(r)).toBeGreaterThan(antes)
  expect(r.lanzados.length).toBe(1)
})

test('si un flujo que funcionaba se corta, el panel vuelve a leer y lo abre de nuevo', async ($, on) => {
  const r = { llamadas: [] as string[][], lanzados: [] as string[][] }
  const clock = preparar(on, r)
  on('process.spawn', async function* ($, e) {
    r.lanzados.push([...e.argv])
    yield { stream: 'stdout', text: panel('primero') + '\n' }
    return { value: { code: 1, signal: null } }
  })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await clock.advance(2000)
  await new Promise((res) => setTimeout(res, 20))
  const antes = lecturas(r)
  await clock.advance(2000)
  await new Promise((res) => setTimeout(res, 20))
  expect(lecturas(r)).toBeGreaterThan(antes)
  expect(r.lanzados.length).toBeGreaterThan(1)
})

test('con la ruta del binario, las llamadas ya no pasan por el envoltorio', async ($, on) => {
  const r = { llamadas: [] as string[][], binario: '/opt/qdd/bin/qdd' }
  preparar(on, r)
  on('process.spawn', async function* () {
    await new Promise(() => {})
    return { value: { code: 0, signal: null } }
  })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const lectura = r.llamadas.find((a) => a.includes('panel'))
  expect(r.llamadas[0]).toEqual(['qdd', 'consola', 'binario'])
  expect(lectura?.[0]).toBe('/opt/qdd/bin/qdd')
})

test('una respuesta rara a «consola binario» se ignora: sigue con qdd', async ($, on) => {
  const r = { llamadas: [] as string[][], binario: 'Usage:\n  qdd consola [command]' }
  preparar(on, r)
  on('process.spawn', async function* () {
    await new Promise(() => {})
    return { value: { code: 0, signal: null } }
  })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  expect(r.llamadas.find((a) => a.includes('panel'))?.[0]).toBe('qdd')
})

test('la ruta de Windows del binario también vale', async ($, on) => {
  const r = { llamadas: [] as string[][], binario: 'C:\\Program Files\\nodejs\\node_modules\\qdd-framework\\bin\\qdd.exe' }
  preparar(on, r)
  on('process.spawn', async function* () {
    await new Promise(() => {})
    return { value: { code: 0, signal: null } }
  })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  expect(r.llamadas.find((a) => a.includes('panel'))?.[0]).toBe(r.binario)
})

test('el latido del flujo (una línea vacía) no cuenta como cambio ni lo corta', async ($, on) => {
  const r = { llamadas: [] as string[][], lanzados: [] as string[][] }
  const clock = preparar(on, r)
  on('process.spawn', async function* ($, e) {
    r.lanzados.push([...e.argv])
    yield { stream: 'stdout', text: panel('primero') + '\n\n\n' }
    await new Promise(() => {})
    return { value: { code: 0, signal: null } }
  })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await clock.advance(2000)
  await new Promise((res) => setTimeout(res, 20))
  expect(await ui.find({ type: 'Text', text: /primero/ })).toBeDefined()
  await clock.advance(10000)
  await new Promise((res) => setTimeout(res, 20))
  expect(r.lanzados.length).toBe(1)
})

test('una sesión sin terminal (claude -p) no abre el flujo ni lee cada 2 s (MEJ-020)', async ($, on) => {
  const r = { llamadas: [] as string[][], lanzados: [] as string[][] }
  const clock = preparar(on, r)
  on('process.spawn', async function* ($, e) {
    r.lanzados.push([...e.argv])
    return { value: { code: 0, signal: null } }
  })
  await $.session.start({ surface: null, isInteractive: false, cwd: '/work' })
  const antes = lecturas(r)
  await clock.advance(60000)
  expect(r.lanzados).toEqual([])
  expect(lecturas(r)).toBe(antes)
})

test('una lectura lenta no pisa lo más nuevo que trajo el flujo', async ($, on) => {
  const r: any = { llamadas: [] as string[][] }
  const clock = preparar(on, r)
  let soltarFlujo: () => void = () => {}
  const esperaFlujo = new Promise<void>((res) => (soltarFlujo = res))
  on('process.spawn', async function* () {
    yield { stream: 'stdout', text: panel('primero') + '\n' }
    await esperaFlujo
    yield { stream: 'stdout', text: panel('lo más nuevo') + '\n' }
    await new Promise(() => {})
    return { value: { code: 0, signal: null } }
  })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await clock.advance(2000)
  await new Promise((res) => setTimeout(res, 20))
  let soltarLectura: (v: any) => void = () => {}
  r.lectura = new Promise((res) => (soltarLectura = res))
  const releer = ui.press({ key: 'leer' })
  await new Promise((res) => setTimeout(res, 20))
  soltarFlujo()
  await new Promise((res) => setTimeout(res, 20))
  soltarLectura({ value: { exitCode: 0, stdout: panel('lectura vieja'), stderr: '' } })
  await releer
  await new Promise((res) => setTimeout(res, 20))
  expect(await ui.find({ type: 'Text', text: /lo más nuevo/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /lectura vieja/ })).toBeUndefined()
})

test('al cortarse el flujo vuelve a preguntar la ruta del binario: una actualización pudo moverlo', async ($, on) => {
  const r = { llamadas: [] as string[][], lanzados: [] as string[][], binario: '/viejo/qdd' }
  const clock = preparar(on, r)
  on('process.spawn', async function* ($, e) {
    r.lanzados.push([...e.argv])
    yield { stream: 'stdout', text: panel('primero') + '\n' }
    if (r.lanzados.length > 1) await new Promise(() => {})
    return { value: { code: 0, signal: null } }
  })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await new Promise((res) => setTimeout(res, 20))
  r.binario = '/nuevo/qdd'
  await clock.advance(2000)
  await new Promise((res) => setTimeout(res, 20))
  expect(r.lanzados.at(-1)?.[0]).toBe('/nuevo/qdd')
})

// MEJ-020: lo que queda en temporizadores viaja en la línea del panel, sin un proceso por lectura.
function panelCon(extras: object) {
  return JSON.stringify({ ...JSON.parse(panel('con datos')), ...extras })
}

function abrirFlujoCon(on, linea: string) {
  on('process.spawn', async function* () {
    yield { stream: 'stdout', text: linea + '\n' }
    await new Promise(() => {})
    return { value: { code: 0, signal: null } }
  })
}

const conteo = (r, palabra: string) => r.llamadas.filter((a) => a.includes(palabra)).length

test('si la línea trae las herramientas, no se lanza «herramienta listar» (MEJ-020)', async ($, on) => {
  const r = { llamadas: [] as string[][], lectura: { value: { exitCode: 0, stdout: panelCon({ herramientas: [] }), stderr: '' } } }
  const clock = preparar(on, r)
  abrirFlujoCon(on, panelCon({ herramientas: [] }))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await clock.advance(60000)
  expect(conteo(r, 'listar')).toBe(0)
})

test('un qdd cuya línea no trae herramientas sigue con «herramienta listar» (MEJ-020)', async ($, on) => {
  const r = { llamadas: [] as string[][] }
  const clock = preparar(on, r)
  abrirFlujoCon(on, panel('anterior'))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await clock.advance(60000)
  expect(conteo(r, 'listar')).toBeGreaterThan(1)
})

test('una herramienta que quitan se cierra aunque llegue por la línea (MEJ-020)', async ($, on) => {
  const r = { llamadas: [] as string[][], lectura: { value: { exitCode: 0, stdout: panelCon({ herramientas: [{ id: 'bot', nombre: 'Bot', comando: ['bot'], acciones: [] }] }), stderr: '' } } }
  const clock = preparar(on, r)
  const cerradas: string[] = []
  on('ui.close', ($, e) => {
    cerradas.push(e.id)
    return { value: undefined }
  })
  abrirFlujoCon(on, panelCon({ herramientas: [] }))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await clock.advance(2000)
  await new Promise((res) => setTimeout(res, 20))
  expect(cerradas.some((id) => id.endsWith('bot'))).toBe(true)
})

test('el ciclo de mejoras solo se lanza si la línea dice que toca (MEJ-020)', async ($, on) => {
  const r = { llamadas: [] as string[][] }
  const clock = preparar(on, r)
  abrirFlujoCon(on, panelCon({ mejoras_toca: false }))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await clock.advance(2000)
  await new Promise((res) => setTimeout(res, 20))
  await clock.advance(120000)
  expect(conteo(r, 'mejoras')).toBe(0)
})

test('si toca, el ciclo de mejoras corre (MEJ-020)', async ($, on) => {
  const r = { llamadas: [] as string[][] }
  const clock = preparar(on, r)
  abrirFlujoCon(on, panelCon({ mejoras_toca: true }))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await clock.advance(2000)
  await new Promise((res) => setTimeout(res, 20))
  await clock.advance(60000)
  expect(conteo(r, 'mejoras')).toBeGreaterThan(0)
})

test('un qdd cuya línea no dice si toca sigue pidiendo el ciclo cada minuto (MEJ-020)', async ($, on) => {
  const r = { llamadas: [] as string[][] }
  const clock = preparar(on, r)
  abrirFlujoCon(on, panel('anterior'))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await clock.advance(120000)
  expect(conteo(r, 'mejoras')).toBeGreaterThan(1)
})
