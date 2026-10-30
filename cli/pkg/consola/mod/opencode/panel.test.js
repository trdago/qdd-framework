import { test } from 'node:test'
import assert from 'node:assert/strict'
import { vista, motivoDeSalida, celdas } from './panel.js'

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
