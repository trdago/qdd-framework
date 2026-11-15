const { test } = require('node:test')
const assert = require('node:assert/strict')
const { vista, carpetaDe, motivoDeSalida } = require('./panel.js')

const ESTADO = {
  actualizado: '2026-11-05T20:50:00-03:00',
  abreviaturas: ['Có', 'Pu'],
  frentes: [
    { id: 'a', nombre: 'Panel', simbolos: ['✅', '⏳'], objetivo: 'ag', prefijo: '├─ ', actividad: { running: true } },
    { id: 'b', nombre: 'Ticket', simbolos: ['❌', '—'], objetivo: 'ag', prefijo: '└─ ' },
    { id: 'c', nombre: 'Suelta', simbolos: ['✅', '✅'] },
  ],
  ocultos: 5,
  esperan: 1,
  fallan: 1,
  esperas: [{ frente: 'Panel', etapa: 'Pu', texto: 'correr la suite' }],
  fallas: [{ frente: 'Ticket', etapa: 'Có', texto: 'se imprime el ticket' }],
  objetivos: [{ id: 'ag', nombre: 'Antigravity', simbolo: '⏳', resumen: '1/2 tareas listas', foco: true }],
}

function pedido(campos) {
  return { id: 'pedido:1', tipo: 'pedido', tecla: 'a', etiqueta: 'Publicar', detalle: 'npm publish\notra línea', carpeta: '/x/qdd', ...campos }
}

test('sin datos, sin estado o sin qdd dice el motivo', () => {
  assert.equal(vista(null, 'no se pudo ejecutar qdd').motivo, 'no se pudo ejecutar qdd')
  assert.match(vista(null, '').motivo, /sin datos/)
  const sin = vista({ estado: null, motivo: "créalo con 'qdd estado init'", acciones: [], version: 'v2' }, '')
  assert.match(sin.motivo, /qdd estado init/)
  assert.equal(sin.version, 'v2')
  assert.match(motivoDeSalida(1, ''), /actualiza qdd/)
  assert.equal(motivoDeSalida(1, ' no hay qdd '), 'no hay qdd')
  assert.equal(motivoDeSalida(0, 'x'), '')
})

test('la cabecera dice lo mismo que el panel de Claude Code', () => {
  const v = vista({ version: 'v2.10.0', estado: { ...ESTADO, hace: 'hace 3 h', viejo: true }, acciones: [pedido({ estado: 'pendiente' })] }, '')
  assert.equal(v.titulo, 'Tareas · hace 3 h')
  assert.equal(v.version, 'v2.10.0')
  assert.match(v.aviso, /Sin actualizar hace 3 h/)
  assert.deepEqual(v.resumen.map((r) => [r.texto, r.tono]), [
    ['3 de 8 tareas', 'primario'],
    ['⏳ 1 etapa espera', 'espera'],
    ['❌ 1 etapa falla', 'falla'],
    ['▶ 1 para ti', 'espera'],
  ])
})

test('sin esperas ni fallas lo dice; sin «hace», usa la hora', () => {
  const v = vista({ estado: { ...ESTADO, ocultos: 0, esperan: 0, fallan: 0, esperas: [], fallas: [] }, acciones: [] }, '')
  assert.match(v.titulo, /^Tareas · \d\d:\d\d$/)
  assert.deepEqual(v.resumen.map((r) => r.texto), ['3 tareas', '✓ nada espera ni falla'])
  assert.equal(v.aviso, '')
})

test('un qdd que no trae los totales cuenta las listas', () => {
  const { esperan, fallan, ...viejo } = ESTADO
  assert.deepEqual(vista({ estado: viejo, acciones: [] }, '').resumen.map((r) => r.texto).slice(1), ['⏳ 1 etapa espera', '❌ 1 etapa falla'])
})

test('la tabla agrupa por objetivo y deja lo suelto al final', () => {
  const v = vista({ estado: ESTADO, acciones: [] }, '')
  assert.deepEqual(v.cabecera, ['Có', 'Pu'])
  assert.deepEqual(v.filas.map((f) => [f.nombre, f.objetivo, f.tono]), [
    ['🎯 Antigravity · foco', true, 'primario'],
    ['├─ Panel', false, 'espera'],
    ['└─ Ticket', false, 'falla'],
    ['🎯 sin objetivo', true, 'primario'],
    ['Suelta', false, 'listo'],
  ])
  assert.equal(v.filas[0].resumen, '⏳ 1/2 tareas listas')
  assert.deepEqual(v.filas[1].simbolos, ['✅', '⏳'])
  assert.equal(v.filas[1].running, true)
  assert.equal(v.filas[2].running, false)
})

test('un objetivo con ciclo propio lleva sus abreviaturas', () => {
  const estado = { ...ESTADO, ciclos: [{ nombre: 'agentico', abreviaturas: ['Di', 'Ev'] }], objetivos: [{ ...ESTADO.objetivos[0], ciclo: 'agentico' }] }
  assert.deepEqual(vista({ estado, acciones: [] }, '').filas[1].abreviaturas, ['Di', 'Ev'])
})

test('sin objetivos, las tareas van tal cual', () => {
  const v = vista({ estado: { ...ESTADO, objetivos: [] }, acciones: [] }, '')
  assert.deepEqual(v.filas.map((f) => f.nombre), ['├─ Panel', '└─ Ticket', 'Suelta'])
})

// tarjeta: un pedido como lo arma la vista, desde un proyecto en /x/qdd.
function tarjeta(campos) {
  return vista({ carpeta: '/x/qdd', estado: ESTADO, acciones: [pedido(campos)] }, '').pedidos[0]
}

test('un pedido pendiente lleva su tecla, y sin estado es pendiente (qdd anterior a 2.6.2)', () => {
  const p = tarjeta({ estado: 'pendiente' })
  assert.deepEqual([p.tecla, p.marca, p.tono, p.meta], ['a', '', 'normal', 'en qdd'])
  assert.equal(tarjeta({}).tecla, 'a')
})

test('un pedido que espera se ve distinto si ya se puede confirmar', () => {
  const sinTecla = tarjeta({ estado: 'espera', tecla: '', espera: ['b', 'd'] })
  assert.deepEqual([sinTecla.marca, sinTecla.tecla, sinTecla.tono, sinTecla.meta], ['◌', '', 'tenue', 'espera a [b] [d] · en qdd'])
  const conTecla = tarjeta({ estado: 'espera', espera: ['b'] })
  assert.deepEqual([conTecla.marca, conTecla.tecla, conTecla.meta], ['', 'a', 'espera a [b] · confírmalo y corre solo · en qdd'])
})

test('un pedido armado dice cuándo corre y lo último que dijo', () => {
  const p = tarjeta({ estado: 'armado', tecla: '', espera: ['b'], intentos: 2, aviso: 'línea vieja\nsin tag todavía' })
  assert.deepEqual([p.marca, p.tono], ['◷', 'espera'])
  assert.deepEqual(p.notas, [
    { texto: 'Armado · corre solo cuando [b] termine · intento 2', tono: 'espera' },
    { texto: 'sin tag todavía', tono: 'tenue' },
  ])
  assert.match(tarjeta({ estado: 'armado', proximo: '2026-11-05T20:50:00-03:00' }).notas[0].texto, /cuando esté listo · reintenta \d\d:\d\d/)
})

test('ejecutando, hecho y con falla se ven como en Claude Code', () => {
  const corre = tarjeta({ estado: 'ejecutando', inicio: '2026-11-05T20:50:00-03:00' })
  assert.deepEqual([corre.marca, corre.tono], ['⟳', 'tenue'])
  assert.match(corre.meta, /^ejecutando desde \d\d:\d\d · en qdd$/)
  assert.match(tarjeta({ estado: 'ejecutando' }).meta, /^ejecutando… · en qdd$/)

  const hecho = tarjeta({ estado: 'hecho', fin: '2026-11-05T20:50:00-03:00', resultado: 'compilando\npublicado' })
  assert.deepEqual([hecho.marca, hecho.tono, hecho.comando], ['✓', 'tenue', ''])
  assert.match(hecho.meta, /^hecho \d\d:\d\d$/)
  assert.deepEqual(hecho.notas, [{ texto: 'publicado', tono: 'tenue' }])

  const falla = tarjeta({ estado: 'fallo', resultado: 'a\nb\nexit 1' })
  assert.deepEqual([falla.marca, falla.tono, falla.tecla], ['✗', 'falla', 'a'])
  assert.deepEqual(falla.notas, [{ texto: 'b\nexit 1', tono: 'falla' }])
  assert.equal(falla.meta, 'falló · en qdd')
})

test('un «todavía no» sin armar lo dice primero y atenúa lo que dijo el comando', () => {
  const p = tarjeta({ estado: 'pendiente', aviso: 'x\nno hay tag' })
  assert.deepEqual(p.notas, [
    { texto: '◷ Todavía no', tono: 'espera' },
    { texto: 'no hay tag', tono: 'tenue' },
  ])
})

test('el comando va en una línea con cuántas faltan, y la carpeta desde el proyecto', () => {
  const p = tarjeta({ estado: 'pendiente', carpeta: '/x/qdd/cli' })
  assert.deepEqual([p.id, p.etiqueta, p.comando, p.meta], ['pedido:1', 'Publicar', 'npm publish  … (+1 líneas)', 'en qdd/cli'])
  assert.equal(tarjeta({ detalle: 'ls', carpeta: '/otro/repo' }).comando, 'ls')
  assert.equal(tarjeta({ detalle: 'ls', carpeta: '/otro/repo' }).meta, 'en repo', 'otro repositorio: solo su nombre (FND-090)')
  assert.equal(tarjeta({ carpeta: '' }).meta, '')
})

test('las fallas del tablero se listan con su tarea y etapa', () => {
  assert.deepEqual(vista({ estado: ESTADO, acciones: [] }, '').fallas, [{ titulo: 'Ticket › Có', texto: 'se imprime el ticket' }])
})

test('▶ para ti cuenta solo lo que se puede confirmar', () => {
  const acciones = [
    pedido({ id: 'pedido:1', estado: 'pendiente' }),
    pedido({ id: 'pedido:2', estado: 'fallo' }),
    pedido({ id: 'pedido:3', estado: 'espera', tecla: 'd' }),
    pedido({ id: 'pedido:4', estado: 'espera', tecla: '' }),
    pedido({ id: 'pedido:5', estado: 'ejecutando' }),
    pedido({ id: 'pedido:6', estado: 'hecho' }),
  ]
  assert.equal(vista({ estado: ESTADO, acciones }, '').resumen.at(-1).texto, '▶ 3 para ti')
})

test('las fichas por aprobar se listan; las demás acciones no se dibujan', () => {
  const acciones = [
    { id: 'aprobar:x', tipo: 'aprobar', etiqueta: 'Aprobar ficha x', detalle: 'qdd modules approve x' },
    { id: 'descartar:a', tipo: 'descartar', etiqueta: 'Descartar' },
    { id: 'auditar', tipo: 'comando', etiqueta: 'Auditar' },
    { id: 'nuevo', tipo: 'algo-que-no-conoce', etiqueta: '?' },
  ]
  const v = vista({ estado: ESTADO, acciones }, '')
  assert.deepEqual(v.fichas, [{ id: 'aprobar:x', etiqueta: 'Aprobar ficha x' }])
  assert.deepEqual(v.pedidos, [])
})

test('sin estado también se ven los pedidos y las fichas', () => {
  const v = vista({ estado: null, motivo: 'sin estado', acciones: [pedido({ estado: 'pendiente' }), { id: 'aprobar:x', tipo: 'aprobar', etiqueta: 'Aprobar x' }] }, '')
  assert.equal(v.motivo, 'sin estado')
  assert.equal(v.pedidos.length, 1)
  assert.equal(v.fichas.length, 1)
})

test('el proyecto es la primera carpeta con .qdd', () => {
  const con = new Set(['/b/.qdd', '/c/.qdd'])
  const existe = (ruta) => con.has(ruta.replaceAll('\\', '/'))
  assert.deepEqual(carpetaDe(['/a', '/b', '/c'], '', existe), { carpeta: '/b', opciones: ['/b', '/c'] })
  assert.deepEqual(carpetaDe(['/a', '/b', '/c'], '/c', existe), { carpeta: '/c', opciones: ['/b', '/c'] }, 'la elegida manda')
  assert.deepEqual(carpetaDe(['/a', '/b'], '/ya-no-esta', existe), { carpeta: '/b', opciones: ['/b'] })
  assert.deepEqual(carpetaDe(['/a'], '', existe), { carpeta: '', opciones: [] })
  assert.deepEqual(carpetaDe([], '', existe), { carpeta: '', opciones: [] })
})
