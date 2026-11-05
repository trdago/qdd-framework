import { expect, mock, test } from 'claude-code/testing'

// Herramientas del proyecto en el panel (RFC 0011): ⚙ Config en el panel de tareas y una pestaña por herramienta.

const TAREAS = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Tareas', isFocused: true, bodyColumns: 60, placement: 'dock', scroll: { offset: 0, bodyRows: 30 }, view: {} },
} as const

const pestaña = (id: string, title: string) => ({ ...TAREAS, requestId: 'qdd-herr-' + id, props: { ...TAREAS.props, title } })

const PANEL = {
  formato: 1,
  carpeta: '/work/bot',
  estado: { formato: 1, resumen: '', etapas: ['Código'], abreviaturas: ['Có'], frentes: [{ id: 'a', nombre: 'Algo', estado: 'abierto', simbolos: ['⏳'] }], esperas: [], fallas: [] },
  acciones: [],
}

const SIM = {
  id: 'sim',
  nombre: 'Simulador WS',
  descripcion: 'Simula lo que manda un usuario al bot.',
  comando: ['./bin/sim'],
  archivo: '.qdd/herramientas/sim.yaml',
  acciones: [
    {
      id: 'enviar',
      nombre: 'Enviar como usuario',
      campos: [
        { id: 'mensaje', tipo: 'texto', requerido: true },
        { id: 'chat', tipo: 'opcion', opciones: ['soporte', 'ventas'], por_defecto: 'soporte' },
      ],
    },
    { id: 'logs', nombre: 'Tras bambalinas', modo: 'en-vivo' },
    { id: 'reiniciar', nombre: 'Reiniciar el estado', confirmar: true },
  ],
}

const LOGS = { id: 'logs', nombre: 'Visor de logs', comando: ['make', 'logs'], archivo: '.qdd/herramientas/logs.yaml', acciones: [{ id: 'ver', nombre: 'Ver', modo: 'en-vivo' }] }

type Salida = { exitCode: number; stdout: string; stderr: string }
const ok = (d: unknown): Salida => ({ exitCode: 0, stdout: typeof d === 'string' ? d : JSON.stringify(d), stderr: '' })

// preparar: cada 'qdd …' responde lo de respuestas (por su subcomando); lo demás, el panel.
function preparar(on, estado: { herramientas: unknown[]; respuestas?: Record<string, Salida> }, registro: { llamadas: string[][]; aperturas: any[]; cierres: any[]; enviados: string[] }) {
  const clock = mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work/bot' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', ($, e) => {
    registro.aperturas.push(e)
    return { value: { isPlaced: true } }
  })
  on('ui.close', ($, e) => {
    registro.cierres.push(e)
    return { value: undefined }
  })
  on('prompt.submit', ($, e) => {
    registro.enviados.push(e.text)
    return { text: e.text }
  })
  on('process.run', ($, e) => {
    const argv = [...e.argv]
    registro.llamadas.push(argv)
    if (argv[1] === 'herramienta' && argv[2] === 'listar') return { value: ok({ herramientas: estado.herramientas, problemas: null }) }
    if (argv[1] === 'herramienta') return { value: { ...(estado.respuestas?.[argv[2]] ?? ok('')) } }
    return { value: ok(PANEL) }
  })
  return clock
}

const nuevo = () => ({ llamadas: [] as string[][], aperturas: [] as any[], cierres: [] as any[], enviados: [] as string[] })

test('cada herramienta guardada es su propio panel, con su nombre como pestaña (HER-C07)', async ($, on) => {
  const r = nuevo()
  preparar(on, { herramientas: [SIM, LOGS] }, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work/bot' })
  expect(r.llamadas).toContainEqual(['qdd', 'herramienta', 'listar', '--json'])
  const abiertas = r.aperturas.map((a) => [a.id, a.title])
  expect(abiertas).toContainEqual(['qdd-herr-sim', 'Simulador WS'])
  expect(abiertas).toContainEqual(['qdd-herr-logs', 'Visor de logs'])
  // Una pestaña no le quita el foco a la conversación.
  expect(r.aperturas.filter((a) => a.id.startsWith('qdd-herr-')).every((a) => !a.focus)).toBe(true)

  const ui = await $.ui.mount({ ...pestaña('sim', 'Simulador WS'), surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: 'Simula lo que manda un usuario al bot.' })).toBeDefined()
  expect(await ui.find({ key: 'h-accion-enviar' })).toMatchObject({ props: { label: '1  Enviar como usuario' } })
  expect(await ui.find({ key: 'h-accion-logs' })).toMatchObject({ props: { label: '2  Tras bambalinas' } })
})

test('una acción una-vez corre con sus campos, cada uno un argumento, y muestra su salida al terminar (HER-C05, HER-C03)', async ($, on) => {
  const r = nuevo()
  preparar(on, { herramientas: [SIM], respuestas: { correr: ok('enviado al chat soporte\n') } }, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work/bot' })
  const ui = await $.ui.mount({ ...pestaña('sim', 'Simulador WS'), surface: 'terminal' })
  await ui.press({ key: 'h-accion-enviar' })
  expect(await ui.find({ key: 'h-campo-mensaje' })).toBeDefined()
  expect(await ui.find({ key: 'h-campo-chat' })).toMatchObject({ props: { value: 'soporte' } })
  // Sin lo requerido no corre: dice qué falta.
  await ui.press({ key: 'h-correr' })
  expect(r.llamadas.some((a) => a[2] === 'correr')).toBe(false)
  expect(await ui.find({ type: 'Text', text: /falta «mensaje»/ })).toBeDefined()

  await ui.input({ key: 'h-campo-mensaje', text: 'hola; rm -rf /', kind: 'change' })
  await ui.select({ key: 'h-campo-chat', value: 'ventas' })
  await ui.press({ key: 'h-correr' })
  expect(r.llamadas).toContainEqual(['qdd', 'herramienta', 'correr', 'sim', 'enviar', 'mensaje=hola; rm -rf /', 'chat=ventas'])
  expect(await ui.find({ type: 'Text', text: /enviado al chat soporte/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /✓ Enviar como usuario/ })).toBeDefined()
})

test('una acción en-vivo muestra su salida mientras corre y ■ la detiene (HER-C05)', async ($, on) => {
  const r = nuevo()
  preparar(on, { herramientas: [SIM] }, r)
  let soltar: () => void = () => {}
  const espera = new Promise<void>((res) => (soltar = res))
  let cortada = false
  const lanzados: string[][] = []
  on('process.spawn', async function* ($, e) {
    if (e.argv.includes('seguir')) return { value: { code: 1, signal: null } }
    lanzados.push([...e.argv])
    try {
      yield { stream: 'stdout', text: 'evento 1\n' }
      await espera
      yield { stream: 'stdout', text: 'evento 2\n' }
      yield { stream: 'stdout', text: 'evento 3\n' }
      return { code: 0, signal: null }
    } finally {
      cortada = true
    }
  })
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work/bot' })
  const ui = await $.ui.mount({ ...pestaña('sim', 'Simulador WS'), surface: 'terminal' })
  await ui.press({ key: 'h-accion-logs' })
  await ui.press({ key: 'h-correr' })
  expect(lanzados).toEqual([['qdd', 'herramienta', 'correr', 'sim', 'logs']])
  await new Promise((res) => setTimeout(res, 20))
  expect(await ui.find({ type: 'Text', text: /evento 1/ })).toBeDefined()
  expect(await ui.find({ key: 'h-detener' })).toMatchObject({ props: { label: 'd  ■ Detener' } })

  await ui.press({ key: 'h-detener' })
  soltar()
  await new Promise((res) => setTimeout(res, 20))
  expect(cortada).toBe(true)
  expect(await ui.find({ type: 'Text', text: /evento 3/ })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: /■ Tras bambalinas · detenida/ })).toBeDefined()
  expect(await ui.find({ key: 'h-detener' })).toBeUndefined()
})

test('lo que pide confirmar no corre sin el «¿seguro?» de la persona (HER-C06)', async ($, on) => {
  const r = nuevo()
  preparar(on, { herramientas: [SIM], respuestas: { correr: ok('listo\n') } }, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work/bot' })
  const ui = await $.ui.mount({ ...pestaña('sim', 'Simulador WS'), surface: 'terminal' })
  await ui.press({ key: 'h-accion-reiniciar' })
  await ui.press({ key: 'h-correr' })
  expect(r.llamadas.some((a) => a[2] === 'correr')).toBe(false)
  expect(await ui.find({ type: 'Text', text: '¿Seguro? Reiniciar el estado' })).toBeDefined()
  await ui.press({ key: 'h-no' })
  expect(r.llamadas.some((a) => a[2] === 'correr')).toBe(false)
  await ui.press({ key: 'h-correr' })
  await ui.press({ key: 'h-si' })
  expect(r.llamadas).toContainEqual(['qdd', 'herramienta', 'correr', 'sim', 'reiniciar', '--si'])
})

test('abrir una herramienta y correr una acción no le pide nada a la IA (HER-C09)', async ($, on) => {
  const r = nuevo()
  preparar(on, { herramientas: [SIM], respuestas: { correr: ok('ok\n') } }, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work/bot' })
  const ui = await $.ui.mount({ ...pestaña('sim', 'Simulador WS'), surface: 'terminal' })
  await ui.press({ key: 'h-accion-enviar' })
  await ui.input({ key: 'h-campo-mensaje', text: 'hola' })
  await ui.press({ key: 'h-correr' })
  expect(r.llamadas.some((a) => a[2] === 'correr')).toBe(true)
  expect(r.enviados).toEqual([])
})

test('⚙ Config agrega una herramienta: lista los candidatos o una ruta, la IA la arma y queda como pestaña (HER-C01)', async ($, on) => {
  const r = nuevo()
  const estado = {
    herramientas: [] as unknown[],
    respuestas: {
      candidatos: ok([{ comando: ['./bin/sim'], origen: 'bin', agregada: false }, { comando: ['make', 'logs'], origen: 'make', agregada: true }]),
      configurar: ok('Configura `./bin/sim` como herramienta del panel de QDD…\n'),
    },
  }
  const clock = preparar(on, estado, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work/bot' })
  const ui = await $.ui.mount({ ...TAREAS, surface: 'terminal' })
  expect(await ui.find({ key: 'config' })).toMatchObject({ props: { label: '⚙ Config' } })
  await ui.press({ key: 'config' })
  expect(await ui.find({ type: 'Text', text: /No hay herramientas/ })).toBeDefined()
  await ui.press({ key: 'h-agregar' })
  expect(r.llamadas).toContainEqual(['qdd', 'herramienta', 'candidatos', '--json'])
  expect(await ui.find({ key: 'h-cand-0' })).toMatchObject({ props: { label: './bin/sim · bin' } })
  // Lo ya agregado no se ofrece otra vez; una ruta escrita a mano también vale.
  expect(await ui.find({ key: 'h-cand-1' })).toBeUndefined()
  expect(await ui.find({ key: 'h-ruta' })).toBeDefined()

  await ui.press({ key: 'h-cand-0' })
  expect(r.llamadas).toContainEqual(['qdd', 'herramienta', 'configurar', './bin/sim'])
  expect(r.enviados).toEqual(['Configura `./bin/sim` como herramienta del panel de QDD…'])
  expect(await ui.find({ type: 'Text', text: /2 · La IA lee su ayuda y propone sus acciones/ })).toBeDefined()

  // La IA la guardó con qdd: aparece en el listado, queda como pestaña y el paso 5 se ve hecho.
  estado.herramientas = [SIM]
  await clock.advance(10000)
  expect(r.aperturas.map((a) => a.id)).toContain('qdd-herr-sim')
  expect(await ui.find({ type: 'Text', text: /✓ Simulador WS quedó guardada/ })).toBeDefined()
})

test('una ruta escrita a mano se configura igual, con sus argumentos', async ($, on) => {
  const r = nuevo()
  preparar(on, { herramientas: [], respuestas: { candidatos: ok([]), configurar: ok('Configura…\n') } }, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work/bot' })
  const ui = await $.ui.mount({ ...TAREAS, surface: 'terminal' })
  await ui.press({ key: 'config' })
  await ui.press({ key: 'h-agregar' })
  expect(await ui.find({ type: 'Text', text: /No encontré ejecutables/ })).toBeDefined()
  await ui.input({ key: 'h-ruta', text: '/opt/bot/simular --modo ws' })
  expect(r.llamadas).toContainEqual(['qdd', 'herramienta', 'configurar', '/opt/bot/simular', '--modo', 'ws'])
  expect(r.enviados).toEqual(['Configura…'])
})

test('quitar pide confirmación, borra su definición con qdd y cierra su pestaña (HER-C08)', async ($, on) => {
  const r = nuevo()
  const estado = { herramientas: [SIM, LOGS] as unknown[], respuestas: { quitar: ok('[+] Quitada\n') } }
  preparar(on, estado, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work/bot' })
  const ui = await $.ui.mount({ ...TAREAS, surface: 'terminal' })
  await ui.press({ key: 'config' })
  expect(await ui.find({ type: 'Text', text: 'Simulador WS' })).toBeDefined()
  await ui.press({ key: 'h-quitar-sim' })
  expect(r.llamadas.some((a) => a[2] === 'quitar')).toBe(false)
  expect(await ui.find({ type: 'Text', text: /¿Quitar «Simulador WS»\?/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /\.qdd\/herramientas\/sim\.yaml/ })).toBeDefined()
  estado.herramientas = [LOGS]
  await ui.press({ key: 'h-si' })
  expect(r.llamadas).toContainEqual(['qdd', 'herramienta', 'quitar', 'sim', '--si'])
  expect(r.cierres.map((c) => c.id)).toEqual(['qdd-herr-sim'])
  expect(await ui.find({ type: 'Text', text: 'Simulador WS' })).toBeUndefined()
})

test('lo que le falta a una herramienta en esta máquina se ve en su pestaña y no corre', async ($, on) => {
  const r = nuevo()
  preparar(on, { herramientas: [{ ...SIM, aviso: 'falta la variable de entorno BOT_TOKEN' }] }, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work/bot' })
  const ui = await $.ui.mount({ ...pestaña('sim', 'Simulador WS'), surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: /⚠ falta la variable de entorno BOT_TOKEN/ })).toMatchObject({ props: { color: 'yellow' } })
  await ui.press({ key: 'h-accion-logs' })
  expect(await ui.find({ key: 'h-correr' })).toBeUndefined()
})
