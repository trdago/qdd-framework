import { test } from 'node:test'
import assert from 'node:assert/strict'
import { vista, motivoDeSalida, celdas, destellosNuevos, brillo, fuenteDePulso, tocaEnviar } from './panel.js'

const ESTADO = {
  actualizado: '2026-10-30T02:40:00-03:00',
  abreviaturas: ['Có', 'Pu'],
  frentes: [
    { id: 'a', nombre: 'Panel', simbolos: ['✅', '⏳'] },
    { id: 'b', nombre: 'Ticket', simbolos: ['❌', '—'] },
  ],
  ocultos: 5,
  esperas: [{ frente: 'Panel', etapa: 'Pruebas', texto: 'correr la suite', de_quien: 'CI' }],
  fallas: [{ frente: 'Ticket', etapa: 'Código', texto: 'se imprime el ticket' }],
}
const ACCIONES = [
  { id: 'aprobar:x', tipo: 'aprobar' },
  { id: 'aprobar:y', tipo: 'aprobar' },
  { id: 'auditar', tipo: 'comando' },
]

test('dibuja lo mismo que Claude Code, en lectura (CCC-C23)', () => {
  const v = vista({ version: 'v2.5.0', estado: ESTADO, acciones: ACCIONES }, '')
  assert.match(v.titulo, /^Tareas · \d\d:\d\d$/)
  assert.equal(v.version, 'v2.5.0')
  assert.deepEqual(v.resumen.map((r) => r.texto), ['2 de 7 tareas', '⏳ 1', '❌ 1'])
  assert.deepEqual(v.filas.map((f) => [f.nombre, f.celdas, f.tono]), [['Panel', '✅ ⏳', 'warning'], ['Ticket', '❌ —', 'error']])
  assert.equal(v.ocultas, 5)
  assert.equal(v.esperas[0].texto, 'correr la suite · de CI')
  assert.equal(v.fichas, 2)
  assert.equal(v.acciones, undefined, 'en OpenCode el panel no ofrece acciones')
})

test('sin estado.yaml, sin qdd o con un qdd antiguo dice el motivo', () => {
  assert.match(vista({ estado: null, motivo: "créalo con 'qdd estado init'", acciones: [] }, '').motivo, /qdd estado init/)
  assert.equal(vista(null, 'no se pudo ejecutar qdd').motivo, 'no se pudo ejecutar qdd')
  assert.match(motivoDeSalida(1, ''), /actualiza qdd/)
  assert.equal(motivoDeSalida(0, 'x'), '')
})

test('alinea ✅ ⏳ ❌ (2 columnas) con — (1 columna)', () => {
  assert.equal(celdas(['✅', '—', '⏳']), '✅ —  ⏳')
})

test('la tarea en curso: running, destello con un pulso nuevo y latido de 1 s (EPT-C11)', () => {
  const conPulso = (pulso) => ({ ...ESTADO, actual: 'a', frentes: [{ ...ESTADO.frentes[0], actividad: { running: true, pulso, fuente: 'agente' } }, ESTADO.frentes[1]] })
  const v = vista({ estado: conPulso('10:00:00'), acciones: [] }, '')
  assert.deepEqual(v.filas.map((f) => f.running), [true, false])
  assert.equal(v.actual, 'a')
  const vistos = {}
  assert.deepEqual(destellosNuevos(vistos, v.filas), [], 'la primera lectura no destella')
  assert.deepEqual(destellosNuevos(vistos, vista({ estado: conPulso('10:00:07'), acciones: [] }, '').filas), ['a'])
  assert.deepEqual([0, 1, 2, 3].map((t) => brillo(t, false)), ['brillante', 'brillante', 'tenue', 'tenue'])
  assert.equal(brillo(3, true), 'acento')
})

test('las herramientas del agente de OpenCode laten, con un pulso a la CLI cada 5 s como mucho (EPT-C12)', () => {
  const ev = (tool, status) => ({ type: 'message.part.updated', properties: { part: { type: 'tool', tool, state: { status } } } })
  assert.equal(fuenteDePulso(ev('bash', 'running')), 'agente')
  assert.equal(fuenteDePulso(ev('task', 'running')), 'subagente')
  assert.equal(fuenteDePulso(ev('bash', 'completed')), '')
  assert.equal(fuenteDePulso({ properties: { part: { type: 'text' } } }), '')
  assert.equal(tocaEnviar(10000, 6000), false)
  assert.equal(tocaEnviar(11000, 6000), true)
})

test('el resumen cuenta «▶ N para ti» con los pedidos que se pueden ejecutar', () => {
  const pedidos = [{ id: 'pedido:1', tipo: 'pedido', estado: 'pendiente' }, { id: 'pedido:2', tipo: 'pedido', estado: 'hecho' }]
  const v = vista({ estado: ESTADO, acciones: [...pedidos, ...ACCIONES] }, '')
  assert.equal(v.resumen.at(-1).texto, '▶ 1 para ti')
  assert.equal(vista({ estado: ESTADO, acciones: [pedidos[1]] }, '').resumen.length, 3)
})
