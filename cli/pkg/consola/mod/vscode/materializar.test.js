const { test } = require('node:test')
const assert = require('node:assert/strict')
const { claves, cambiados, estilo } = require('./media/materializar.js')

// Fase D (RFC 0014): la vista anima solo lo que cambió, como el panel de Claude Code (hypr-materialize), con los
// efectos que dice qdd (qdd consola efectos). Lógica pura: el webview la carga como script y node --test como módulo.

const VISTA = {
  pedidos: [{ id: 'pedido:1', etiqueta: 'Publicar', notas: [] }],
  fichas: [{ id: 'aprobar:x', etiqueta: 'Aprobar x' }],
  filas: [{ id: 'a', nombre: 'Panel', simbolos: ['✅', '⏳'] }, { id: 'b', nombre: 'Ticket', simbolos: ['✅'] }],
}

test('claves: una por pedido, ficha y fila, con lo que se ve', () => {
  assert.deepEqual([...claves(VISTA).keys()], ['pedido:pedido:1', 'ficha:aprobar:x', 'fila:a', 'fila:b'])
  assert.deepEqual([...claves({ motivo: 'sin datos' }).keys()], [])
})

test('la primera vista no anima nada; después, solo lo nuevo o lo que cambió', () => {
  assert.deepEqual(cambiados(null, VISTA), new Set())
  const nueva = {
    ...VISTA,
    pedidos: [{ id: 'pedido:1', etiqueta: 'Publicar', notas: [{ texto: '◷ Todavía no' }] }, { id: 'pedido:2', etiqueta: 'Instalar', notas: [] }],
    filas: [VISTA.filas[0], { id: 'b', nombre: 'Ticket', simbolos: ['❌'] }],
  }
  assert.deepEqual(cambiados(VISTA, nueva), new Set(['pedido:pedido:1', 'pedido:pedido:2', 'fila:b']))
  assert.deepEqual(cambiados(VISTA, VISTA), new Set(), 'la misma vista otra vez no anima')
})

test('estilo: con efectos activos da el acento y la duración; apagados, sin datos o un color raro, no anima', () => {
  assert.deepEqual(estilo({ activo: true, acento: '#7daea3', abrir_ms: 800 }), { animar: true, acento: '#7daea3', ms: 800 })
  assert.equal(estilo({ activo: false, acento: '#7daea3', abrir_ms: 800 }).animar, false)
  assert.equal(estilo(undefined).animar, false, 'un qdd sin efectos no anima')
  assert.deepEqual(estilo({ activo: true, acento: 'red; background:url(x)', abrir_ms: 99999 }), { animar: true, acento: '', ms: 2000 }, 'el acento solo es un color #hex y la duración tiene tope')
})
