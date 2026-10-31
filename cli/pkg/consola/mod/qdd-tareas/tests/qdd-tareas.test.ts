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
  expect(await ui.find({ type: 'Text', text: '⏳ 1 etapa espera' })).toMatchObject({ props: { color: 'yellow' } })
  expect(await ui.find({ type: 'Text', text: '❌ 1 etapa falla' })).toMatchObject({ props: { color: 'red' } })
  // Sin números grandes ni recuadro en la cabecera.
  expect(await ui.find({ type: 'Text', text: '▀▀█' })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: /Có Pu Po Va/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: 'Gemini retirado' })).toMatchObject({ props: { color: 'yellow' } })
  expect(await ui.find({ type: 'Text', text: 'Ticket de MP en logs' })).toMatchObject({ props: { color: 'red' } })
  // Sin sección «Esperando» (pedido del usuario, 2026-10-30): el ⏳ ya está en la tabla y en el resumen.
  expect(await ui.find({ type: 'Text', text: /Esperando/ })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: '• Gemini retirado › Validación real' })).toBeUndefined()
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

// RFC 0006 §3.9 (pedidos del usuario, 2026-10-30): cada mensaje de la persona es un ciclo. Lo que ya se vio sale del
// tablero y qdd anota el modo de permisos de la IA (qdd consola ciclo --modo).
test('cada mensaje marca un ciclo con el modo de la IA y lo ya visto sale del tablero', async ($, on) => {
  const llamadas: string[][] = []
  preparar(on, ok(PANEL), llamadas, { exitCode: 0, stdout: '[✓] cumple', stderr: '' })
  on('classic.UserPromptSubmit', () => ({}))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await ui.press({ key: 'b-revisar-estado' })
  expect(await ui.find({ type: 'Text', text: '✓ Revisar estado' })).toBeDefined()
  await $.classic.UserPromptSubmit({ prompt: 'sigue', permission_mode: 'auto' })
  expect(llamadas).toContainEqual(['qdd', 'consola', 'ciclo', '--modo', 'auto'])
  expect(await ui.find({ type: 'Text', text: '✓ Revisar estado' })).toBeUndefined()
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
  expect(await ui.find({ type: 'Text', text: '⏳ 2 etapas esperan' })).toMatchObject({ props: { color: 'yellow' } })
  expect(await ui.find({ type: 'Text', text: '❌ 1 etapa falla' })).toMatchObject({ props: { color: 'red' } })
  // Cada etapa es una celda propia: las tres «·» de Login, alineadas con su abreviatura (2 columnas).
  expect((await ui.findAll({ type: 'Text', text: /^·\s*$/ })).map((n) => n.props.children?.[0] ?? n.text).filter((t) => t === '· ')).toHaveLength(3)
})

// Pedido del usuario (2026-10-31): al pulsar una etapa ⏳ o ❌, la IA se pronuncia: intenta completarla y, si falta
// algo, deja un pedido o pregunta. qdd da el mensaje exacto (consola accion atender:<tarea>:<etapa>) y el panel lo
// envía a la conversación.
test('pulsar una etapa ⏳ le envía a la IA el mensaje que da qdd para atenderla', async ($, on) => {
  const llamadas: string[][] = []
  const enviados: string[] = []
  const atender = { id: 'atender:gemini:3', tipo: 'atender', etiqueta: 'Atender «Validación real» de «Gemini retirado»', detalle: 'Atiende la etapa…', confirmar: false }
  const mensaje = 'Atiende la etapa «Validación real» de la tarea «Gemini retirado» (id gemini). Intenta completarla ahora.'
  preparar(on, ok({ ...PANEL, acciones: [...ACCIONES, atender] }), llamadas, { exitCode: 0, stdout: mensaje + '\n', stderr: '' })
  on('prompt.submit', ($, e) => {
    enviados.push(e.text)
    return { text: e.text }
  })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ key: 'e-gemini-3' })).toMatchObject({ props: { label: '⏳', plain: true } })
  // Una etapa sin acción (✅, o una ⏳ que qdd no ofrece) no es un botón.
  expect(await ui.find({ key: 'e-gemini-0' })).toBeUndefined()
  expect(await ui.find({ key: 'e-ticket-0' })).toBeUndefined()
  await ui.press({ key: 'e-gemini-3' })
  expect(llamadas).toContainEqual(['qdd', 'consola', 'accion', 'atender:gemini:3'])
  expect(enviados).toEqual([mensaje])
  expect(await ui.find({ type: 'Text', text: /Enviado a la IA/ })).toBeDefined()
})

test('si qdd ya no ofrece la etapa, no se envía nada y se ve por qué', async ($, on) => {
  const enviados: string[] = []
  const atender = { id: 'atender:gemini:3', tipo: 'atender', etiqueta: 'Atender «Validación real» de «Gemini retirado»', detalle: '…', confirmar: false }
  preparar(on, ok({ ...PANEL, acciones: [atender] }), [], { exitCode: 1, stdout: '', stderr: "[!] la acción 'atender:gemini:3' no está entre las que ofrece" })
  on('prompt.submit', ($, e) => {
    enviados.push(e.text)
    return { text: e.text }
  })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await ui.press({ key: 'e-gemini-3' })
  expect(enviados).toEqual([])
  expect(await ui.find({ type: 'Text', text: /no está entre las que ofrece/ })).toBeDefined()
})

test('la cabecera dice hace cuánto se actualizó y avisa si el tablero quedó viejo (FND-088)', async ($, on) => {
  preparar(on, ok({ ...PANEL, estado: { ...ESTADO, hace: 'hace 9 h', viejo: true } }))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: 'Tareas · hace 9 h' })).toMatchObject({ props: { color: 'cyan' } })
  expect(await ui.find({ type: 'Text', text: /^⚠ Sin actualizar hace 9 h/ })).toMatchObject({ props: { color: 'yellow' } })
})

test('un tablero al día no avisa nada (FND-088)', async ($, on) => {
  preparar(on, ok({ ...PANEL, estado: { ...ESTADO, hace: 'hace 3 min' } }))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: 'Tareas · hace 3 min' })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /Sin actualizar/ })).toBeUndefined()
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
  // FND-090: un conteo en cero no se muestra; sin esperas ni fallas, el resumen lo dice en verde.
  expect(await ui.find({ type: 'Text', text: /etapas? esperan?$/ })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: /etapas? fallan?$/ })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: '✓ nada espera ni falla' })).toMatchObject({ props: { color: 'green' } })
})

// FND-090 (captura del usuario, 2026-10-31): el panel mostraba «~/Documents/proyectos/qdd o: o Abrir carpeta». Del
// proyecto se ve solo el nombre de la carpeta (blueprint consola-claude-code), y el botón sin la tecla repetida.
test('muestra solo el nombre de la carpeta del proyecto y la abre con «o» sin confirmar (CCC-C21, FND-090)', async ($, on) => {
  const llamadas: string[][] = []
  const abrir = { id: 'abrir-carpeta', tipo: 'abrir', etiqueta: 'Abrir carpeta', detalle: 'xdg-open /home/ana/proyectos/tienda', confirmar: false }
  const conCarpeta = { ...PANEL, carpeta: '/home/ana/proyectos/tienda', acciones: [...ACCIONES, abrir] }
  preparar(on, ok(conCarpeta), llamadas, { exitCode: 0, stdout: '[+] Carpeta abierta: xdg-open /home/ana/proyectos/tienda', stderr: '' }, [], { HOME: '/home/ana' })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/home/ana/proyectos/tienda' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })

  expect(await ui.find({ type: 'Text', text: '📂 tienda' })).toBeDefined()
  expect((await ui.findAll({ type: 'Text' })).some((n) => /\/home|~\//.test(n.text))).toBe(false)
  expect(await ui.find({ key: 'b-abrir-carpeta' })).toMatchObject({ props: { hotkey: 'o', plain: true, label: 'Abrir carpeta' } })
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
  expect(await ui.find({ type: 'Text', text: '📂 app' })).toBeDefined()
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
  preparar(on, ok({ ...PANEL, acciones: [PEDIDO, ...ACCIONES] }), llamadas, { exitCode: 0, stdout: '[+] En marcha: Publicar v2.5.1', stderr: '' })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: /Para ti/ })).toBeDefined()
  expect(await ui.find({ key: 'b-pedido:1' })).toMatchObject({ props: { hotkey: 'a' } })
  expect(await ui.find({ type: 'Text', text: /git push origin v2\.5\.1/ })).toBeDefined()

  await ui.press({ key: 'b-pedido:1' })
  expect(await ui.find({ type: 'Text', text: '¿Publicar v2.5.1 (dispara npm y el binario)?' })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: 'Ejecutará: git push origin v2.5.1' })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: 'En: qdd' })).toBeDefined()
  expect(llamadas.some((a) => a[2] === 'accion')).toBe(false)

  await ui.press({ key: 'si' })
  expect(llamadas).toContainEqual(['qdd', 'consola', 'accion', 'pedido:1', '--confirmado'])
  // FND-089: qdd lo lanza aparte y vuelve en el acto; la tarjeta dice que corre, sin recuadro de resultado.
  expect(await ui.find({ type: 'Text', text: /⟳ .*Publicar v2\.5\.1/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /^\s*ejecutando desde/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /En marcha/ })).toBeUndefined()
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

// FND-090: primero lo que es de la persona (▶ Para ti), después lo que falla y al final las herramientas.
test('las secciones van en orden: tareas, para ti, fallas y acciones, separadas por una línea (FND-090)', async ($, on) => {
  preparar(on, ok({ ...PANEL, acciones: [PEDIDO, ...ACCIONES] }))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  const textos = (await ui.findAll({ type: 'Text' })).map((n) => n.text)
  const donde = (re: RegExp) => textos.findIndex((t) => re.test(t))
  const orden = [/^Tareas · /, /^Tarea$/, /Para ti/, /Fallas/, /^Acciones$/].map(donde)
  expect(orden.every((i) => i >= 0)).toBe(true)
  expect([...orden].sort((a, b) => a - b)).toEqual(orden)
  expect(textos.filter((t) => /^─+$/.test(t)).length).toBe(3)
})

test('las acciones van en una sola fila de botones, sin repetir el comando (FND-090)', async ($, on) => {
  preparar(on, ok(PANEL))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ key: 'botones' })).toMatchObject({ props: { flexDirection: 'row', flexWrap: 'wrap' } })
  expect(await ui.find({ type: 'Text', text: /solo lectura/ })).toBeUndefined()
  expect(await ui.find({ key: 'b-auditar' })).toMatchObject({ props: { label: 'u  Auditar' } })
})

test('la confirmación y el resultado van junto al botón pulsado (FND-090)', async ($, on) => {
  const descartar = { id: 'descartar:ticket', tipo: 'descartar', etiqueta: 'Descartar «Ticket de MP en logs»', detalle: 'qdd estado cerrar ticket --descartado', confirmar: true }
  preparar(on, ok({ ...PANEL, acciones: [PEDIDO, ...ACCIONES, descartar] }), [], { exitCode: 1, stdout: '', stderr: '[!] el pedido 1 espera a [b]' })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  const donde = async (re: RegExp) => (await ui.findAll({ type: 'Text' })).map((n) => n.text).findIndex((t) => re.test(t))
  await ui.press({ key: 'x-ticket' })
  expect(await donde(/^¿Descartar «Ticket/)).toBeLessThan(await donde(/Para ti/))
  await ui.press({ key: 'no' })
  await ui.press({ key: 'b-pedido:1' })
  await ui.press({ key: 'si' })
  expect(await donde(/espera a \[b\]/)).toBeLessThan(await donde(/Fallas/))
})

test('un pedido en una subcarpeta la nombra desde el proyecto, sin la ruta completa (FND-090)', async ($, on) => {
  const enNpm = { ...PEDIDO, carpeta: '/work/qdd/npm' }
  preparar(on, ok({ ...PANEL, carpeta: '/work/qdd', acciones: [enNpm, { ...PEDIDO, id: 'pedido:2', tecla: 'b', carpeta: '/work/qdd' }] }))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work/qdd' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: /en qdd\/npm/ })).toBeDefined()
  expect((await ui.findAll({ type: 'Text' })).some((n) => n.text.includes('/work'))).toBe(false)
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
  expect(await ui.find({ type: 'Text', text: /⟳ .*Publicar v2\.5\.1/ })).toMatchObject({ props: { dimColor: true } })
  expect(llamadas.filter((a) => a[2] === 'accion').length).toBe(1)
})

// FND-089 (captura del usuario, 2026-10-31): el pedido mostraba su script entero (20 líneas) y «nada por hacer»
// justo arriba de uno que se estaba ejecutando.
test('un comando de varias líneas se ve en una; entero solo al confirmarlo (FND-089)', async ($, on) => {
  const largo = { ...PEDIDO, detalle: 'R=trdago/qdd-framework; T=v2.8.0-rc.2\ngit fetch -q origin main\ngit push origin $T' }
  preparar(on, ok({ ...PANEL, acciones: [largo] }))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: '   $ R=trdago/qdd-framework; T=v2.8.0-rc.2  … (+2 líneas)' })).toMatchObject({ props: { wrap: 'truncate-end' } })
  expect(await ui.find({ type: 'Text', text: /git push origin/ })).toBeUndefined()
  await ui.press({ key: 'b-pedido:1' })
  expect(await ui.find({ type: 'Text', text: /Ejecutará: R=trdago[\s\S]*git push origin \$T/ })).toBeDefined()
})

test('«Para ti» cuenta lo que se está ejecutando: nunca dice «nada por hacer» arriba de un pedido en marcha (FND-089)', async ($, on) => {
  const ejecutando = { ...PEDIDO, tecla: undefined, estado: 'ejecutando', inicio: '2026-10-30T12:00:00-03:00' }
  preparar(on, ok({ ...PANEL, acciones: [ejecutando] }))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: '1 ejecutando' })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /nada por hacer/ })).toBeUndefined()
})

test('mientras un pedido corre, los demás botones responden (FND-089)', async ($, on) => {
  const llamadas: string[][] = []
  preparar(on, ok({ ...PANEL, acciones: [PEDIDO, ...ACCIONES] }), llamadas, { exitCode: 0, stdout: '[+] En marcha', stderr: '' })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await ui.press({ key: 'b-pedido:1' })
  await ui.press({ key: 'si' })
  await ui.press({ key: 'b-revisar-estado' })
  expect(llamadas.filter((a) => a[2] === 'accion').map((a) => a[3])).toEqual(['pedido:1', 'revisar-estado'])
})

test('si qdd rechaza el pedido al lanzarlo, se ve el motivo y el botón sigue (FND-089)', async ($, on) => {
  preparar(on, ok({ ...PANEL, acciones: [PEDIDO] }), [], { exitCode: 1, stdout: '', stderr: '[!] el pedido 1 espera a [b]' })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  await ui.press({ key: 'b-pedido:1' })
  await ui.press({ key: 'si' })
  expect(await ui.find({ type: 'Text', text: /espera a \[b\]/ })).toBeDefined()
  expect(await ui.find({ key: 'b-pedido:1' })).toMatchObject({ props: { hotkey: 'a' } })
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
  // FND-090: un hecho ocupa una línea; su comando ya no hace falta.
  expect((await ui.findAll({ type: 'Text', text: /\$ git push/ })).length).toBe(1)
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

// Uso medido (2026-10-30): sin tarea en curso el pulso igual va, sin destello; qdd no toca la actividad y lo anota
// como uso real.
test('sin tarea en curso, las herramientas igual envían el pulso (uso real), sin destello', async ($, on) => {
  const llamadas: string[][] = []
  const clock = preparar(on, ok(PANEL), llamadas)
  on('tool.call', () => ({ result: 'hecho' }))
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  await $.tool.call({ tool: 'Bash', command: 'ls' })
  await clock.advance(10)
  expect(llamadas).toContainEqual(['qdd', 'estado', 'pulso', '--fuente', 'agente'])
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
  expect(await ui.find({ type: 'Text', text: /◌ .*Mergear #14/ })).toMatchObject({ props: { dimColor: true } })
  expect(await ui.find({ type: 'Text', text: /espera a \[d\]/ })).toMatchObject({ props: { dimColor: true } })
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
