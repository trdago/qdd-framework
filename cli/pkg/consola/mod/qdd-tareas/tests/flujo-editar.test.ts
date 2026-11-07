import { expect, mock, test } from 'claude-code/testing'

// El flujo es una interfaz de trabajo entre la persona y la IA (RFC 0012, fases 2 y 3; sprint 6): la persona elige un nodo,
// lo edita o lo comenta, y qdd valida. El panel solo arma el comando y muestra lo que qdd entrega.

const PANEL = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Tareas', isFocused: true, bodyColumns: 60, placement: 'dock', scroll: { offset: 0, bodyRows: 30 }, viewportRows: 40 },
} as const

const pestana = (id: string, columnas = 60) => ({ ...PANEL, requestId: 'qdd-flujo-' + id, props: { ...PANEL.props, title: id, bodyColumns: columnas } })
const PAGO = { id: 'pago', nombre: 'Cómo se cobra', nodos: 2, huella: 'h1' }

const dibujoBase = () => ({
  id: 'pago',
  nombre: 'Cómo se cobra',
  lineas: ['( Llega )', '   │', '( Fin )'],
  nodos: [
    { id: 'llega', tipo: 'paso', nombre: 'Llega', descripcion: 'la solicitud', siguen: ['fin'], comentarios: [{ de: 'persona', texto: '¿por qué primero?', cuando: 'x' }, { de: 'ia', texto: 'porque valida', cuando: 'x' }] },
    { id: 'fin', tipo: 'resultado', nombre: 'Fin' },
  ],
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
  on('ui.open', ($, e) => {
    r.abiertos.push(e.id)
    r.aperturas.push(e)
    return { value: { isPlaced: true } }
  })
  on('ui.close', ($, e) => {
    r.cerrados.push(e.id)
    return { value: undefined }
  })
  on('prompt.submit', ($, e) => {
    if (r.rechazo) return { drop: r.rechazo }
    r.enviados.push(e.text)
    return { text: e.text }
  })
  on('process.run', ($, e) => {
    r.llamadas.push([...e.argv])
    if (e.argv.includes('binario')) return { value: { exitCode: 1, stdout: '', stderr: '' } }
    if (e.argv.includes('panel')) return { value: { exitCode: 0, stdout: JSON.stringify({ formato: 1, carpeta: '/work', acciones: [], herramientas: [], flujos: r.flujos }), stderr: '' } }
    if (e.argv[1] !== 'flujo') return { value: { exitCode: 0, stdout: '{}', stderr: '' } }
    if (e.argv[2] === 'ver') return { value: { exitCode: 0, stdout: JSON.stringify(r.dibujo), stderr: '' } }
    if (e.argv[2] === 'enviar') return { value: { exitCode: 0, stdout: JSON.stringify(r.envio), stderr: '' } }
    if (e.argv[2] === 'nuevo') return { value: { exitCode: r.falla ? 1 : 0, stdout: JSON.stringify({ id: 'cobro', nombre: 'Cobro' }), stderr: r.falla || '' } }
    if (r.falla) return { value: { exitCode: 1, stdout: '', stderr: '[!] ' + r.falla } }
    return { value: { exitCode: 0, stdout: '', stderr: '' } }
  })
}

const nuevo = (flujos: object[] = [PAGO]) => ({
  llamadas: [] as string[][],
  abiertos: [] as string[],
  aperturas: [] as any[],
  cerrados: [] as string[],
  enviados: [] as string[],
  flujos,
  dibujo: dibujoBase(),
  envio: { mensaje: '✎ La persona trabajó', n: 1 },
  falla: '',
  rechazo: '',
})
const esperar = () => new Promise((res) => setTimeout(res, 20))
const comandos = (r, sub: string) => r.llamadas.filter((a) => a[1] === 'flujo' && a[2] === sub)

async function abrir($, on, r, columnas = 60) {
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...pestana('pago', columnas), surface: 'terminal' })
  await esperar()
  return ui
}

async function elegir(ui, id = 'llega') {
  await ui.press({ key: 'fl-n-' + id })
  await esperar()
}

test('mirar sigue igual, y elegir un nodo lo resalta y muestra su hilo y sus acciones', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, r)
  expect(await ui.find({ type: 'Text', text: '( Llega )' })).toBeDefined()
  expect(await ui.find({ key: 'fl-comentar' })).toBeUndefined()

  await elegir(ui)
  expect(comandos(r, 'ver').at(-1)).toEqual(['qdd', 'flujo', 'ver', 'pago', '--ancho', '60', '--json', '--elegido', 'llega'])
  expect(await ui.find({ type: 'Text', text: 'la solicitud' })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: 'Tú: ¿por qué primero?' })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: 'IA: porque valida' })).toBeDefined()
  for (const k of ['fl-editar', 'fl-despues', 'fl-rama', 'fl-mover', 'fl-flechas', 'fl-comentar', 'fl-detalle', 'fl-quitar']) expect(await ui.find({ key: k })).toBeDefined()

  await elegir(ui)
  expect(await ui.find({ key: 'fl-comentar' })).toBeUndefined()
})

test('agregar pasa por qdd como edición de la persona y vuelve a dibujar', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, r)
  await elegir(ui)
  const dibujosAntes = comandos(r, 'ver').length
  await ui.press({ key: 'fl-despues' })
  await ui.select({ key: 'fl-tipo', value: 'resultado' })
  await ui.input({ key: 'fl-nombre', text: 'Cobra' })
  await esperar()
  expect(comandos(r, 'agregar')[0]).toEqual(['qdd', 'flujo', 'agregar', '--de', 'persona', '--nombre', 'Cobra', '--tipo', 'resultado', '--', 'pago', 'llega'])
  expect(comandos(r, 'ver').length).toBeGreaterThan(dibujosAntes)
  expect(await ui.find({ key: 'fl-nombre' })).toBeUndefined()
  expect(r.enviados).toEqual([])
})

test('rama agrega una salida más', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, r)
  await elegir(ui)
  await ui.press({ key: 'fl-rama' })
  await ui.input({ key: 'fl-nombre', text: 'Rechazo' })
  await esperar()
  expect(comandos(r, 'agregar')[0]).toEqual(['qdd', 'flujo', 'agregar', '--de', 'persona', '--nombre', 'Rechazo', '--tipo', 'paso', '--rama', '--', 'pago', 'llega'])
})

test('editar trae lo que el nodo tiene y guarda sus campos', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, r)
  await elegir(ui)
  await ui.press({ key: 'fl-editar' })
  await ui.input({ key: 'fl-nombre', text: 'Recibe', kind: 'change' })
  await ui.press({ key: 'fl-guardar' })
  await esperar()
  expect(comandos(r, 'modificar')[0]).toEqual(['qdd', 'flujo', 'modificar', '--de', 'persona', '--nombre', 'Recibe', '--descripcion', 'la solicitud', '--tipo', 'paso', '--', 'pago', 'llega'])
})

test('un comentario queda en el flujo y la IA lo recibe en el acto, aunque empiece con guion', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, r)
  await elegir(ui)
  await ui.press({ key: 'fl-comentar' })
  await ui.input({ key: 'fl-texto', text: '--no-verify es lo que me preocupa' })
  await esperar()
  expect(comandos(r, 'comentar')[0]).toEqual(['qdd', 'flujo', 'comentar', '--de', 'persona', '--', 'pago', 'llega', '--no-verify es lo que me preocupa'])
  expect(comandos(r, 'enviar')[0]).toEqual(['qdd', 'flujo', 'enviar', '--json', '--', 'pago'])
  expect(r.enviados).toEqual(['✎ La persona trabajó'])
})

test('si la IA no pudo recibirlo, queda pendiente y se dice', async ($, on) => {
  const r = nuevo()
  r.rechazo = 'la sesión está ocupada'
  const ui = await abrir($, on, r)
  await elegir(ui)
  await ui.press({ key: 'fl-comentar' })
  await ui.input({ key: 'fl-texto', text: 'hola' })
  await esperar()
  expect(comandos(r, 'devolver')[0]).toEqual(['qdd', 'flujo', 'devolver', '--n', '1', '--', 'pago'])
  expect(await ui.find({ type: 'Text', text: /la sesión está ocupada/ })).toBeDefined()
})

test('pedir detalle le pregunta a la IA por ese nodo', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, r)
  await elegir(ui)
  await ui.press({ key: 'fl-detalle' })
  await esperar()
  expect(comandos(r, 'pedir-detalle')[0]).toEqual(['qdd', 'flujo', 'pedir-detalle', '--', 'pago', 'llega'])
  expect(r.enviados.length).toBe(1)
})

test('mover y las flechas se eligen con un clic en otro nodo', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, r)
  await elegir(ui)
  await ui.press({ key: 'fl-mover' })
  await elegir(ui, 'fin')
  await esperar()
  expect(comandos(r, 'mover')[0]).toEqual(['qdd', 'flujo', 'mover', '--de', 'persona', '--despues-de', 'fin', '--', 'pago', 'llega'])

  await ui.press({ key: 'fl-flechas' })
  await elegir(ui, 'fin')
  await esperar()
  expect(comandos(r, 'desconectar')[0]).toEqual(['qdd', 'flujo', 'desconectar', '--de', 'persona', '--con', 'fin', '--', 'pago', 'llega'])
})

test('lo que qdd rechaza se ve con su porqué, sin marca, y el formulario sigue ahí', async ($, on) => {
  const r = nuevo()
  r.falla = 'al nodo "fin" no se llega'
  const ui = await abrir($, on, r)
  await elegir(ui)
  await ui.press({ key: 'fl-rama' })
  await ui.input({ key: 'fl-nombre', text: 'X' })
  await esperar()
  expect(await ui.find({ type: 'Text', text: '⚠ al nodo "fin" no se llega' })).toBeDefined()
  expect(await ui.find({ key: 'fl-nombre' })).toBeDefined()
  expect(r.enviados).toEqual([])
})

test('lo pendiente se ve, se envía o se deshace', async ($, on) => {
  const r = nuevo()
  r.dibujo.por_enviar = ['+ cobra «Cobra» después de llega']
  r.dibujo.puede_deshacer = true
  const ui = await abrir($, on, r)
  expect(await ui.find({ type: 'Text', text: '+ cobra «Cobra» después de llega' })).toBeDefined()
  await ui.press({ key: 'fl-deshacer' })
  await esperar()
  expect(comandos(r, 'deshacer')[0]).toEqual(['qdd', 'flujo', 'deshacer', '--', 'pago'])
  await ui.press({ key: 'fl-enviar' })
  await esperar()
  expect(comandos(r, 'enviar').length).toBe(1)
  expect(r.enviados.length).toBe(1)
})

test('sin pendientes no hay botones de enviar ni deshacer', async ($, on) => {
  const ui = await abrir($, on, nuevo())
  expect(await ui.find({ key: 'fl-enviar' })).toBeUndefined()
  expect(await ui.find({ key: 'fl-deshacer' })).toBeUndefined()
})

test('un mensaje en el chat se lleva lo pendiente: la pestaña lo vuelve a leer', async ($, on) => {
  const r = nuevo()
  r.dibujo.por_enviar = ['+ cobra']
  const ui = await abrir($, on, r)
  const antes = comandos(r, 'ver').length
  r.dibujo.por_enviar = []
  await $.classic.UserPromptSubmit({ prompt: 'hola', permission_mode: 'auto' })
  await esperar()
  await ui.find({ key: 'fl-enviar' })
  await esperar()
  expect(comandos(r, 'ver').length).toBeGreaterThan(antes)
  expect(await ui.find({ key: 'fl-enviar' })).toBeUndefined()
})

test('a 40 columnas nada se sale y las acciones siguen', async ($, on) => {
  const r = nuevo()
  r.dibujo.nodos[0].nombre = 'Un nombre tan largo que no cabe en una fila de cuarenta columnas'
  const ui = await abrir($, on, r, 40)
  await elegir(ui)
  expect(await ui.find({ key: 'fl-comentar' })).toBeDefined()
  const fila = await ui.find({ key: 'fl-n-llega' })
  expect(String(fila.props.label).length).toBeLessThanOrEqual(38)
})

test('sin flujos, Flujos abre ahí mismo el campo para crear el primero', async ($, on) => {
  const r = nuevo([])
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANEL, surface: 'terminal' })
  await esperar()
  await ui.press({ key: 'flujos' })
  await esperar()
  await ui.input({ key: 'fl-nuevo-nombre', text: 'Cobro' })
  await esperar()
  expect(comandos(r, 'nuevo')[0]).toEqual(['qdd', 'flujo', 'nuevo', '--json', '--', 'Cobro'])
  expect(await ui.find({ key: 'fl-nuevo-nombre' })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: '✓ «Cobro» ya es una pestaña, arriba' })).toBeDefined()
  await ui.press({ key: 'fl-nuevo-cerrar' })
  expect(await ui.find({ key: 'fl-nuevo-listo' })).toBeUndefined()
})

test('crear un flujo que qdd rechaza deja el campo con el porqué', async ($, on) => {
  const r = nuevo([])
  r.falla = '[!] el flujo no tiene nombre'
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANEL, surface: 'terminal' })
  await esperar()
  await ui.press({ key: 'flujos' })
  await esperar()
  await ui.input({ key: 'fl-nuevo-nombre', text: ' ' })
  await esperar()
  expect(await ui.find({ type: 'Text', text: '⚠ el flujo no tiene nombre' })).toBeDefined()
  expect(await ui.find({ key: 'fl-nuevo-nombre' })).toBeDefined()
})

test('una pestaña abierta ofrece crear otro flujo, en su mismo lugar', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, r)
  await ui.press({ key: 'fl-nuevo' })
  await esperar()
  expect(await ui.find({ key: 'fl-nuevo-nombre' })).toBeDefined()
  await ui.press({ key: 'fl-nuevo-cancelar' })
  expect(await ui.find({ key: 'fl-nuevo-nombre' })).toBeUndefined()
})

test('abrir un formulario le da las teclas al campo: lo escrito no se va al chat', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, r)
  await elegir(ui)
  await ui.press({ key: 'fl-comentar' })
  await esperar()
  expect(r.aperturas.at(-1)).toMatchObject({ id: 'qdd-flujo-pago', focus: true })
})

test('si Claude Code no suelta las teclas (hay texto en el chat), se le dice a la persona', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, r)
  await elegir(ui)
  await ui.press({ key: 'fl-comentar' })
  await esperar()
  expect(await ui.find({ type: 'Text', text: 'Haz clic en el campo para escribir aquí.' })).toBeDefined()
})

// Pedido de la persona (2026-11-07): preguntarle a un nodo, o al flujo completo, y ver la respuesta ahí mismo.
test('preguntar a un nodo llega a la IA en el acto', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, r)
  await elegir(ui)
  await ui.press({ key: 'fl-preguntar' })
  await ui.input({ key: 'fl-pregunta', text: '¿Qué significa «solicitud» aquí?' })
  await esperar()
  expect(comandos(r, 'preguntar')[0]).toEqual(['qdd', 'flujo', 'preguntar', '--', 'pago', 'llega', '¿Qué significa «solicitud» aquí?'])
  expect(r.enviados).toEqual(['✎ La persona trabajó'])
})

test('preguntar al flujo completo, sin elegir un nodo', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, r)
  await ui.press({ key: 'fl-preguntar-flujo' })
  await ui.input({ key: 'fl-pregunta-flujo', text: '¿Para qué sirve este flujo?' })
  await esperar()
  expect(comandos(r, 'preguntar')[0]).toEqual(['qdd', 'flujo', 'preguntar', '--', 'pago', '¿Para qué sirve este flujo?'])
  expect(r.enviados.length).toBe(1)
})

test('las preguntas y sus respuestas se ven en el nodo y en el flujo', async ($, on) => {
  const r = nuevo()
  r.dibujo.nodos[0].comentarios = [{ de: 'persona', texto: '¿Qué es la solicitud?', cuando: 'x', pregunta: true }, { de: 'ia', texto: 'El pedido de cobro.', cuando: 'x' }]
  r.dibujo.comentarios = [{ de: 'persona', texto: '¿Para qué sirve?', cuando: 'x', pregunta: true }, { de: 'ia', texto: 'Cobra un pago.', cuando: 'x' }]
  const ui = await abrir($, on, r)
  expect(await ui.find({ type: 'Text', text: /❓ Tú: ¿Para qué sirve\?/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /IA: Cobra un pago\./ })).toBeDefined()
  await elegir(ui)
  expect(await ui.find({ type: 'Text', text: /❓ Tú: ¿Qué es la solicitud\?/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /IA: El pedido de cobro\./ })).toBeDefined()
})

// Ítems de un nodo (pedido de la persona, 2026-11-07): las pruebas de qdd-human con su estado, y notas o características.
const conItems = (r) => {
  r.dibujo.nodos[0].items = [
    { id: 'i1', tipo: 'prueba', texto: 'rechaza una firma vencida', estado: 'falla', evidencia: '.qdd/evidence/run_1' },
    { id: 'i2', tipo: 'prueba', texto: 'acepta una firma buena', estado: 'pasa' },
    { id: 'i3', tipo: 'prueba', texto: 'avisa por correo', estado: 'pendiente' },
    { id: 'i4', tipo: 'nota', texto: 'ojo con la hora del servidor' },
  ]
  return r
}

test('la fila del nodo resume sus pruebas y la ficha lista cada ítem', async ($, on) => {
  const r = conItems(nuevo())
  const ui = await abrir($, on, r)
  expect((await ui.find({ key: 'fl-n-llega' })).props.label).toMatch(/✓1 ✗1 ☐1/)
  await elegir(ui)
  expect(await ui.find({ type: 'Text', text: /✗ rechaza una firma vencida/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /run_1/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /✓ acepta una firma buena/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /☐ avisa por correo/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /nota: ojo con la hora/ })).toBeDefined()
  expect(await ui.find({ key: 'fl-item-pasa-i4' })).toBeUndefined()
})

test('una prueba se marca desde el nodo', async ($, on) => {
  const r = conItems(nuevo())
  const ui = await abrir($, on, r)
  await elegir(ui)
  await ui.press({ key: 'fl-item-pasa-i3' })
  await esperar()
  expect(comandos(r, 'item')[0]).toEqual(['qdd', 'flujo', 'item', 'marcar', '--de', 'persona', '--evidencia', 'a mano', '--', 'pago', 'llega', 'i3', 'pasa'])
  await ui.press({ key: 'fl-item-quitar-i4' })
  await esperar()
  expect(comandos(r, 'item')[1]).toEqual(['qdd', 'flujo', 'item', 'quitar', '--de', 'persona', '--', 'pago', 'llega', 'i4'])
})

test('agregar un ítem elige su tipo y no llama a la IA en el acto', async ($, on) => {
  const r = nuevo()
  const ui = await abrir($, on, r)
  await elegir(ui)
  await ui.press({ key: 'fl-item' })
  await ui.select({ key: 'fl-item-tipo', value: 'prueba' })
  await ui.input({ key: 'fl-item-texto', text: 'rechaza una firma vencida' })
  await esperar()
  expect(comandos(r, 'item')[0]).toEqual(['qdd', 'flujo', 'item', 'agregar', '--de', 'persona', '--', 'pago', 'llega', 'prueba', 'rechaza una firma vencida'])
  expect(comandos(r, 'enviar')).toEqual([])
})
