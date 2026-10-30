import { expect, mock, test } from 'claude-code/testing'

const PANE = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 160, rows: 40 },
  props: {
    title: 'Tareas',
    isFocused: true,
    bodyColumns: 60,
    placement: 'dock',
    scroll: { offset: 0, bodyRows: 30 },
    view: {},
  },
} as const

const ESTADO = {
  formato: 1,
  actualizado: '2026-10-29T15:13:01-03:00',
  resumen: '◧ 2 tareas · ⏳1 ❌1',
  etapas: ['Código', 'Pruebas', 'Producción', 'Validación real'],
  abreviaturas: ['Có', 'Pu', 'Po', 'Va'],
  frentes: [
    { id: 'gemini', nombre: 'Gemini retirado', estado: 'abierto', simbolos: ['✅', '✅', '✅', '⏳'] },
    { id: 'ticket', nombre: 'Ticket de MP en logs', estado: 'abierto', simbolos: ['❌', '—', '—', '—'] },
  ],
  esperas: [{ frente: 'Gemini retirado', etapa: 'Validación real', texto: 'ver un análisis con imagen', de_quien: 'uso real' }],
  fallas: [{ frente: 'Ticket de MP en logs', etapa: 'Código', texto: 'la URL con el ticket se imprime' }],
}

const ACCIONES = [
  { id: 'aprobar:pagos', tipo: 'aprobar', etiqueta: 'Aprobar «Pagos»', detalle: 'qdd modules approve pagos', confirmar: true },
  { id: 'revisar-estado', tipo: 'comando', etiqueta: 'Revisar estado', detalle: 'qdd estado check', confirmar: false },
  { id: 'auditar', tipo: 'comando', etiqueta: 'Auditar', detalle: 'qdd audit', confirmar: true },
]

const PANEL = { formato: 1, estado: ESTADO, acciones: ACCIONES }

type Salida = { exitCode: number; stdout: string; stderr: string }

// Prepara los stubs comunes: 'panel' es lo que responde 'qdd consola panel --json' y 'accion' lo de 'qdd consola accion'.
function preparar(on, panel: Salida, llamadas: string[][] = [], accion: Salida = { exitCode: 0, stdout: '', stderr: '' }, aperturas: unknown[] = [], env: Record<string, string> = {}) {
  const clock = mock.clock(on)
  mock.env(on, env)
  on('session.start', () => ({ cwd: '/work' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', ($, e) => {
    aperturas.push(e)
    return { value: { isPlaced: true } }
  })
  on('process.run', ($, e) => {
    llamadas.push([...e.argv])
    if (e.argv[2] === 'accion') return { value: { ...accion } }
    return { value: { ...panel } }
  })
  return clock
}

const ok = (d: unknown): Salida => ({ exitCode: 0, stdout: JSON.stringify(d), stderr: '' })

test('dibuja el panel liviano: título con versión, resumen en una línea, tabla y listas cortas (CCC-C13)', async ($, on) => {
  const llamadas: string[][] = []
  preparar(on, ok({ ...PANEL, version: 'v2.5.0' }), llamadas)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  expect(llamadas[0]).toEqual(['qdd', 'consola', 'panel', '--json'])

  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: /^Tareas · \d\d:\d\d$/ })).toMatchObject({ props: { color: 'cyan', weight: 'bold' } })
  expect(await ui.find({ type: 'Text', text: 'v2.5.0' })).toMatchObject({ props: { dimColor: true } })
  expect(await ui.find({ type: 'Text', text: '2 tareas' })).toMatchObject({ props: { color: 'cyan' } })
  expect(await ui.find({ type: 'Text', text: '⏳ 1 esperan' })).toMatchObject({ props: { color: 'yellow' } })
  expect(await ui.find({ type: 'Text', text: '❌ 1 fallan' })).toMatchObject({ props: { color: 'red' } })
  // Sin números grandes ni recuadro en la cabecera.
  expect(await ui.find({ type: 'Text', text: '▀▀█' })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: /Có Pu Po Va/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: 'Gemini retirado' })).toMatchObject({ props: { color: 'yellow' } })
  expect(await ui.find({ type: 'Text', text: 'Ticket de MP en logs' })).toMatchObject({ props: { color: 'red' } })
  expect(await ui.find({ type: 'Text', text: '• Gemini retirado › Validación real' })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: '  → ver un análisis con imagen · de uso real' })).toMatchObject({ props: { dimColor: true } })
  expect(await ui.find({ type: 'Text', text: /la URL con el ticket se imprime/ })).toBeDefined()
  expect(await ui.find({ key: 'b-aprobar:pagos' })).toMatchObject({ props: { hotkey: '1' } })
  expect(await ui.find({ key: 'b-revisar-estado' })).toMatchObject({ props: { hotkey: 'c' } })
  expect(await ui.find({ key: 'b-auditar' })).toMatchObject({ props: { hotkey: 'u' } })
})

test('aprobar una ficha pide «s  Sí» y solo entonces llama a qdd con --confirmado (CCC-C10)', async ($, on) => {
  const llamadas: string[][] = []
  preparar(on, ok(PANEL), llamadas, { exitCode: 0, stdout: "[+] Ficha 'pagos' aprobada (revisada en abc123).", stderr: '' })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })

  await ui.press({ key: 'b-aprobar:pagos' })
  expect(await ui.find({ type: 'Text', text: '¿Aprobar «Pagos»?' })).toBeDefined()
  expect(llamadas.some((a) => a[2] === 'accion')).toBe(false)

  await ui.press({ key: 'si' })
  expect(llamadas).toContainEqual(['qdd', 'consola', 'accion', 'aprobar:pagos', '--confirmado'])
  expect(await ui.find({ type: 'Text', text: /Ficha 'pagos' aprobada/ })).toBeDefined()
})

test('con «n  No» no se ejecuta nada', async ($, on) => {
  const llamadas: string[][] = []
  preparar(on, ok(PANEL), llamadas)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await ui.press({ key: 'b-auditar' })
  await ui.press({ key: 'no' })
  expect(llamadas.some((a) => a[2] === 'accion')).toBe(false)
  expect(await ui.find({ key: 'b-auditar' })).toBeDefined()
})

test('revisar el estado corre sin confirmación y muestra el resultado (CCC-C12)', async ($, on) => {
  const llamadas: string[][] = []
  preparar(on, ok(PANEL), llamadas, { exitCode: 1, stdout: '', stderr: '[✗] EPF-C03: falta la evidencia' })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await ui.press({ key: 'b-revisar-estado' })
  expect(llamadas).toContainEqual(['qdd', 'consola', 'accion', 'revisar-estado'])
  expect(await ui.find({ type: 'Text', text: '✗ Revisar estado' })).toMatchObject({ props: { color: 'red' } })
  expect(await ui.find({ type: 'Text', text: /EPF-C03: falta la evidencia/ })).toBeDefined()
})

test('sin estado.yaml dice cómo crearlo y sigue ofreciendo las fichas por aprobar', async ($, on) => {
  const sinEstado = { formato: 1, estado: null, motivo: "el proyecto no tiene .qdd/project/estado.yaml: créalo con 'qdd estado init'", acciones: [ACCIONES[0]] }
  preparar(on, ok(sinEstado))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: /qdd estado init/ })).toBeDefined()
  expect(await ui.find({ key: 'b-aprobar:pagos' })).toBeDefined()
})

test('sin qdd o con un qdd antiguo no falla', async ($, on) => {
  preparar(on, { exitCode: 1, stdout: '', stderr: '' })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: /actualiza qdd/ })).toBeDefined()
})

test('el resumen usa los totales de qdd, los mismos que la barra, y «·» es sin empezar (FND-085, FND-086)', async ($, on) => {
  const nuevo = { id: 'login', nombre: 'Login', estado: 'abierto', simbolos: ['✅', '·', '·', '·'] }
  preparar(on, ok({ ...PANEL, estado: { ...ESTADO, frentes: [...ESTADO.frentes, nuevo], tareas: 3, esperan: 2, fallan: 1 } }))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: '⏳ 2 esperan' })).toMatchObject({ props: { color: 'yellow' } })
  expect(await ui.find({ type: 'Text', text: '❌ 1 fallan' })).toMatchObject({ props: { color: 'red' } })
  expect(await ui.find({ type: 'Text', text: /✅ ·  ·  ·/ })).toBeDefined()
})

test('se redibuja solo cuando el panel cambia', async ($, on) => {
  const salida = ok(PANEL)
  const clock = preparar(on, salida)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  salida.stdout = JSON.stringify({ ...PANEL, estado: { ...ESTADO, esperas: [], fallas: [] } })
  await clock.advance(2000)
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: /Esperando/ })).toBeUndefined()
  // 0 ⏳ y 0 ❌: el resumen cambia de 1 a 0.
  expect(await ui.find({ type: 'Text', text: '⏳ 1 esperan' })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: '⏳ 0 esperan' })).toMatchObject({ props: { color: 'yellow' } })
})

test('muestra la carpeta del proyecto a la derecha y la abre con «o» sin confirmar (CCC-C21)', async ($, on) => {
  const llamadas: string[][] = []
  const abrir = { id: 'abrir-carpeta', tipo: 'abrir', etiqueta: 'Abrir carpeta', detalle: 'xdg-open /home/ana/proyectos/tienda', confirmar: false }
  const conCarpeta = { ...PANEL, carpeta: '/home/ana/proyectos/tienda', acciones: [...ACCIONES, abrir] }
  preparar(on, ok(conCarpeta), llamadas, { exitCode: 0, stdout: '[+] Carpeta abierta: xdg-open /home/ana/proyectos/tienda', stderr: '' }, [], { HOME: '/home/ana' })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/home/ana/proyectos/tienda' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })

  expect(await ui.find({ type: 'Text', text: '📂 ~/proyectos/tienda' })).toBeDefined()
  expect(await ui.find({ key: 'b-abrir-carpeta' })).toMatchObject({ props: { hotkey: 'o' } })
  // No se repite en la lista de acciones de abajo.
  expect(await ui.find({ key: 'fila-abrir-carpeta' })).toBeUndefined()

  await ui.press({ key: 'b-abrir-carpeta' })
  expect(llamadas).toContainEqual(['qdd', 'consola', 'accion', 'abrir-carpeta'])
  expect(await ui.find({ type: 'Text', text: /Carpeta abierta/ })).toBeDefined()
})

test('sin abridor en el sistema muestra la carpeta pero no el botón', async ($, on) => {
  preparar(on, ok({ ...PANEL, carpeta: '/srv/app' }))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/srv/app' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: '📂 /srv/app' })).toBeDefined()
  expect(await ui.find({ key: 'b-abrir-carpeta' })).toBeUndefined()
})

test('cada tarea tiene su ✕: pide «s  Sí» y la descarta; las más antiguas se cuentan (CCC-C22)', async ($, on) => {
  const llamadas: string[][] = []
  const descartar = { id: 'descartar:ticket', tipo: 'descartar', etiqueta: 'Descartar «Ticket de MP en logs»', detalle: 'qdd estado cerrar ticket --descartado', confirmar: true }
  const conX = { ...PANEL, estado: { ...ESTADO, ocultos: 5 }, acciones: [...ACCIONES, descartar] }
  preparar(on, ok(conX), llamadas, { exitCode: 0, stdout: "[+] Tarea 'ticket' descartada: sale del panel y queda en el archivo con su motivo.", stderr: '' })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })

  expect(await ui.find({ type: 'Text', text: '2 de 7 tareas' })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: '+5 más antiguas · qdd estado ver' })).toBeDefined()
  expect(await ui.find({ key: 'x-ticket' })).toMatchObject({ props: { label: '✕' } })
  // Sin acción ofrecida por qdd no hay ✕, y la ✕ no se repite en la lista de acciones.
  expect(await ui.find({ key: 'x-gemini' })).toBeUndefined()
  expect(await ui.find({ key: 'fila-descartar:ticket' })).toBeUndefined()

  await ui.press({ key: 'x-ticket' })
  expect(await ui.find({ type: 'Text', text: '¿Descartar «Ticket de MP en logs»?' })).toBeDefined()
  expect(llamadas.some((a) => a[2] === 'accion')).toBe(false)
  await ui.press({ key: 'si' })
  expect(llamadas).toContainEqual(['qdd', 'consola', 'accion', 'descartar:ticket', '--confirmado'])
  expect(await ui.find({ type: 'Text', text: /Tarea 'ticket' descartada/ })).toBeDefined()
})

const PEDIDO = {
  id: 'pedido:1',
  tipo: 'pedido',
  etiqueta: 'Publicar v2.5.1 (dispara npm y el binario)',
  detalle: 'git push origin v2.5.1',
  carpeta: '/work/qdd',
  tecla: 'a',
  confirmar: true,
}

test('un comando pedido por la IA muestra su motivo, el comando exacto y la carpeta, y corre solo con «s  Sí» (CCC-C17, CCC-C18)', async ($, on) => {
  const llamadas: string[][] = []
  preparar(on, ok({ ...PANEL, acciones: [PEDIDO, ...ACCIONES] }), llamadas, { exitCode: 0, stdout: 'To github.com:trdago/qdd-framework.git\n * [new tag] v2.5.1 -> v2.5.1', stderr: '' })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: /Para ti/ })).toBeDefined()
  expect(await ui.find({ key: 'b-pedido:1' })).toMatchObject({ props: { hotkey: 'a' } })
  expect(await ui.find({ type: 'Text', text: /git push origin v2\.5\.1/ })).toBeDefined()

  await ui.press({ key: 'b-pedido:1' })
  expect(await ui.find({ type: 'Text', text: '¿Publicar v2.5.1 (dispara npm y el binario)?' })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: 'Ejecutará: git push origin v2.5.1' })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: 'En: /work/qdd' })).toBeDefined()
  expect(llamadas.some((a) => a[2] === 'accion')).toBe(false)

  await ui.press({ key: 'si' })
  expect(llamadas).toContainEqual(['qdd', 'consola', 'accion', 'pedido:1', '--confirmado'])
  expect(await ui.find({ type: 'Text', text: /new tag/ })).toBeDefined()
})

test('un comando pedido se puede descartar sin ejecutarlo (CCC-C19)', async ($, on) => {
  const llamadas: string[][] = []
  preparar(on, ok({ ...PANEL, acciones: [PEDIDO] }), llamadas)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await ui.press({ key: 'b-pedido:1' })
  expect(await ui.find({ key: 'descartar' })).toMatchObject({ props: { hotkey: 'x' } })
  await ui.press({ key: 'descartar' })
  expect(llamadas).toContainEqual(['qdd', 'consola', 'descartar', '1'])
  expect(llamadas.some((a) => a[2] === 'accion')).toBe(false)
})

test('el panel se abre solo cuando llega un comando pedido, aunque no haya frentes (CCC-C20)', async ($, on) => {
  const aperturas: unknown[] = []
  const vacio = { formato: 1, estado: null, motivo: 'sin estado', acciones: [ACCIONES[2]] }
  const salida = ok(vacio)
  const clock = preparar(on, salida, [], undefined, aperturas)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  expect(aperturas.length).toBe(0)
  salida.stdout = JSON.stringify({ ...vacio, acciones: [PEDIDO, ACCIONES[2]] })
  await clock.advance(2000)
  expect(aperturas.length).toBe(1)
  await clock.advance(2000)
  expect(aperturas.length).toBe(1)
})

test('las secciones van en orden: tareas, acciones, para ti, esperando y fallas', async ($, on) => {
  preparar(on, ok({ ...PANEL, acciones: [PEDIDO, ...ACCIONES] }))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  const textos = (await ui.findAll({ type: 'Text' })).map((n) => n.text)
  const donde = (re: RegExp) => textos.findIndex((t) => re.test(t))
  const orden = [/^Tareas · /, /^Tarea$/, /^Acciones$/, /Para ti/, /Esperando/, /Fallas/].map(donde)
  expect(orden.every((i) => i >= 0)).toBe(true)
  expect([...orden].sort((a, b) => a - b)).toEqual(orden)
})

// FND-070: un pedido aceptado no puede quedar para aceptarlo de nuevo: queda en gris y sin botón.
test('un pedido aceptado queda en gris y sin botón aunque qdd todavía no lo marque (FND-070)', async ($, on) => {
  const llamadas: string[][] = []
  preparar(on, ok({ ...PANEL, acciones: [PEDIDO] }), llamadas, { exitCode: 0, stdout: 'listo', stderr: '' })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await ui.press({ key: 'b-pedido:1' })
  await ui.press({ key: 'si' })
  expect(await ui.find({ key: 'b-pedido:1' })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: /✓ .*Publicar v2\.5\.1/ })).toMatchObject({ props: { dimColor: true } })
  expect(llamadas.filter((a) => a[2] === 'accion').length).toBe(1)
})

test('un pedido ejecutándose o hecho se ve en gris y no se puede clickear (FND-070)', async ($, on) => {
  const ejecutando = { ...PEDIDO, tecla: undefined, estado: 'ejecutando', inicio: '2026-10-30T12:00:00-03:00' }
  const hecho = { ...PEDIDO, id: 'pedido:2', etiqueta: 'Mergear #31', tecla: undefined, estado: 'hecho', fin: '2026-10-30T12:05:00-03:00' }
  preparar(on, ok({ ...PANEL, acciones: [ejecutando, hecho] }))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ key: 'b-pedido:1' })).toBeUndefined()
  expect(await ui.find({ key: 'b-pedido:2' })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: /ejecutando/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /✓ .*Mergear #31/ })).toMatchObject({ props: { dimColor: true } })
  expect(await ui.find({ type: 'Text', text: /hecho 12:05|hecho \d\d:05/ })).toBeDefined()
})

test('un pedido que falló muestra su error y se puede reintentar con su misma tecla', async ($, on) => {
  const fallo = { ...PEDIDO, estado: 'fallo', resultado: 'Permission denied (publickey)', fin: '2026-10-30T12:05:00-03:00' }
  preparar(on, ok({ ...PANEL, acciones: [fallo] }))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: /✗ .*Publicar v2\.5\.1/ })).toMatchObject({ props: { color: 'red' } })
  expect(await ui.find({ type: 'Text', text: /Permission denied/ })).toBeDefined()
  const reintentar = await ui.find({ key: 'b-pedido:1' })
  expect(reintentar).toMatchObject({ props: { hotkey: 'a' } })
  expect(reintentar?.text).toContain('↻ Reintentar')
})

test('un pedido hecho no vuelve a abrir el panel', async ($, on) => {
  const aperturas: unknown[] = []
  const vacio = { formato: 1, estado: null, motivo: 'sin estado', acciones: [] }
  const salida = ok(vacio)
  const clock = preparar(on, salida, [], undefined, aperturas)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  salida.stdout = JSON.stringify({ ...vacio, acciones: [{ ...PEDIDO, tecla: undefined, estado: 'hecho' }] })
  await clock.advance(2000)
  expect(aperturas.length).toBe(0)
})

// RFC 0006 §3.13, blueprint estado-por-tareas: la tarea en curso dice «● running», el ● late y destella con cada pulso.
const EN_CURSO = (pulso: string) => ({
  ...PANEL,
  estado: {
    ...ESTADO,
    actual: 'gemini',
    frentes: ESTADO.frentes.map((f) => (f.id === 'gemini' ? { ...f, actividad: { running: true, pulso, fuente: 'agente' } } : f)),
  },
})

test('la tarea en curso dice «● running» y el ● alterna tenue y brillante cada 1 s (EPT-C08)', async ($, on) => {
  const clock = preparar(on, ok(EN_CURSO('2026-10-30T10:00:00-03:00')))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: 'running' })).toMatchObject({ props: { dimColor: true } })
  expect(await ui.findAll({ type: 'Text', text: '●' })).toHaveLength(1)
  const brillo = async () => !!(await ui.find({ type: 'Text', text: '●' }))?.props?.dimColor
  const antes = await brillo()

  await clock.advance(1000)
  expect(await brillo()).toBe(!antes)
  await clock.advance(1000)
  expect(await brillo()).toBe(antes)
})

test('un pulso nuevo en el JSON hace destellar el ● en acento (EPT-C09)', async ($, on) => {
  const salida = ok(EN_CURSO('2026-10-30T10:00:00-03:00'))
  const clock = preparar(on, salida)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: '●' })).not.toMatchObject({ props: { color: 'cyan' } })
  salida.stdout = JSON.stringify(EN_CURSO('2026-10-30T10:00:07-03:00'))
  await clock.advance(2000)
  expect(await ui.find({ type: 'Text', text: '●' })).toMatchObject({ props: { color: 'cyan' } })
  await clock.advance(1000)
  expect(await ui.find({ type: 'Text', text: '●' })).not.toMatchObject({ props: { color: 'cyan' } })
})

test('cada herramienta del agente late la tarea sin demorarla, con un pulso a la CLI cada 5 s como mucho (EPT-C10)', async ($, on) => {
  const llamadas: string[][] = []
  const clock = preparar(on, ok(EN_CURSO('2026-10-30T10:00:00-03:00')), llamadas)
  on('tool.call', () => ({ result: 'hecho' }))
  on('agent.spawn', () => ({ model: 'sonnet' }))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  const pulsos = () => llamadas.filter((a) => a[1] === 'estado' && a[2] === 'pulso')

  expect(await $.tool.call({ tool: 'Bash', command: 'ls' })).toMatchObject({ result: 'hecho' })
  expect(await ui.find({ type: 'Text', text: '●' })).toMatchObject({ props: { color: 'cyan' } })
  await $.tool.call({ tool: 'Read', file_path: '/work/a' })
  expect(pulsos()).toEqual([['qdd', 'estado', 'pulso', '--fuente', 'agente']])

  await clock.advance(5000)
  await $.agent.spawn({ prompt: 'revisa' })
  await clock.advance(1) // el pulso a la CLI va sin esperar: se completa en el tic siguiente
  expect(pulsos()).toContainEqual(['qdd', 'estado', 'pulso', '--fuente', 'subagente'])
  expect(pulsos().length).toBe(2)
})

test('sin tarea en curso, las herramientas no envían pulsos', async ($, on) => {
  const llamadas: string[][] = []
  preparar(on, ok(PANEL), llamadas)
  on('tool.call', () => ({ result: 'hecho' }))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await $.tool.call({ tool: 'Bash', command: 'ls' })
  expect(llamadas.some((a) => a[2] === 'pulso')).toBe(false)
})

test('el resumen cuenta «▶ N para ti» solo con los pedidos que se pueden ejecutar', async ($, on) => {
  const hecho = { ...PEDIDO, id: 'pedido:2', tecla: undefined, estado: 'hecho' }
  const salida = ok({ ...PANEL, acciones: [PEDIDO, hecho, ...ACCIONES] })
  const clock = preparar(on, salida)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: '▶ 1 para ti' })).toMatchObject({ props: { color: 'yellow' } })
  salida.stdout = JSON.stringify({ ...PANEL, acciones: [hecho, ...ACCIONES] })
  await clock.advance(2000)
  expect(await ui.find({ type: 'Text', text: /para ti$/ })).toBeUndefined()
})

// FND-080: un pedido que espera a otro se muestra después de él, en gris, sin botón, y dice qué espera.
test('un pedido en espera dice a quién espera, no tiene botón ni cuenta como por hacer, y va en el orden de qdd (FND-080)', async ($, on) => {
  const tag = { ...PEDIDO, id: 'pedido:3', etiqueta: 'Publicar v2.7.0', detalle: 'git push origin v2.7.0', tecla: 'd' }
  const merge = { ...PEDIDO, id: 'pedido:1', etiqueta: 'Mergear #14', detalle: 'gh pr merge 14', tecla: undefined, estado: 'espera', espera: ['d'] }
  const llamadas: string[][] = []
  preparar(on, ok({ ...PANEL, acciones: [tag, merge, ...ACCIONES] }), llamadas)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ key: 'b-pedido:1' })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: /Mergear #14 .*espera a \[d\]/ })).toMatchObject({ props: { dimColor: true } })
  expect(await ui.find({ type: 'Text', text: '1 por hacer · 1 en espera' })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: '▶ 1 para ti' })).toBeDefined()
  const textos = (await ui.findAll({ type: 'Text' })).map((n) => n.text)
  const comandoDe = (re: RegExp) => textos.findIndex((t) => re.test(t))
  expect(comandoDe(/v2\.7\.0$/)).toBeLessThan(comandoDe(/gh pr merge 14/))
  expect(llamadas.some((a) => a[2] === 'accion')).toBe(false)
})

// FND-082: un comando que todavía no puede correr (falta un tag, un merge) no se ve como una falla.
test('un pedido que dijo «todavía no» se ve pendiente con su aviso en gris, no en rojo (FND-082)', async ($, on) => {
  const todaviaNo = { ...PEDIDO, estado: 'pendiente', aviso: 'Falta el tag v2.7.0', aviso_en: '2026-10-30T12:05:00-03:00' }
  preparar(on, ok({ ...PANEL, acciones: [todaviaNo] }))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: /todavía no .*Falta el tag v2\.7\.0/ })).toMatchObject({ props: { dimColor: true } })
  expect(await ui.find({ type: 'Text', text: /✗/ })).toBeUndefined()
  expect(await ui.find({ key: 'b-pedido:1' })).toMatchObject({ props: { hotkey: 'a' } })
})

test('si al ejecutarlo sale «todavía no» (código 75), el panel no lo marca como falla (FND-082)', async ($, on) => {
  preparar(on, ok({ ...PANEL, acciones: [PEDIDO] }), [], { exitCode: 75, stdout: 'Falta el tag v2.7.0', stderr: '' })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await ui.press({ key: 'b-pedido:1' })
  await ui.press({ key: 'si' })
  expect(await ui.find({ type: 'Text', text: /✗/ })).toBeUndefined()
  expect(await ui.find({ key: 'b-pedido:1' })).toMatchObject({ props: { hotkey: 'a' } })
})
