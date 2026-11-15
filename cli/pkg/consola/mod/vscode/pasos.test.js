const { test } = require('node:test')
const assert = require('node:assert/strict')
const { inicial, tocar, tecla, si, no, alLlegar } = require('./media/pasos.js')

// Fase B (RFC 0014): ejecutar es en dos pasos, como en el panel de Claude Code: tocar (o la tecla del pedido) pregunta
// «s  Sí / n  No» y solo el Sí manda el id. Lo que ya no se ofrece no se confirma.

const VISTA = {
  pedidos: [
    { id: 'pedido:1', tecla: 'a', ejecutar: true },
    { id: 'pedido:2', tecla: 'b', ejecutar: false },
  ],
  fichas: [{ id: 'aprobar:pagos', ejecutar: true }],
}

test('tocar pregunta y no manda nada; Sí manda el id una vez', () => {
  let r = tocar(inicial, VISTA, 'pedido:1')
  assert.deepEqual(r, { estado: { confirmando: 'pedido:1' }, enviar: null })
  r = si(r.estado, VISTA)
  assert.deepEqual(r, { estado: inicial, enviar: { tipo: 'accion', id: 'pedido:1' } })
  assert.equal(si(r.estado, VISTA).enviar, null, 'un segundo Sí no manda otra vez')
})

test('No o Escape cancelan sin mandar nada', () => {
  const pidiendo = tocar(inicial, VISTA, 'aprobar:pagos').estado
  assert.deepEqual(no(pidiendo), { estado: inicial, enviar: null })
  assert.deepEqual(tecla(pidiendo, VISTA, 'Escape'), { estado: inicial, enviar: null })
  assert.deepEqual(tecla(pidiendo, VISTA, 'n'), { estado: inicial, enviar: null })
})

test('con el teclado: la tecla del pedido pregunta y «s» confirma', () => {
  const r = tecla(inicial, VISTA, 'a')
  assert.equal(r.estado.confirmando, 'pedido:1')
  assert.deepEqual(tecla(r.estado, VISTA, 's').enviar, { tipo: 'accion', id: 'pedido:1' })
  assert.deepEqual(tecla(inicial, VISTA, 's'), { estado: inicial, enviar: null }, '«s» sin nada que confirmar no hace nada')
})

test('lo que no se puede ejecutar ni se pregunta', () => {
  assert.deepEqual(tocar(inicial, VISTA, 'pedido:2'), { estado: inicial, enviar: null })
  assert.deepEqual(tecla(inicial, VISTA, 'b'), { estado: inicial, enviar: null })
  assert.deepEqual(tocar(inicial, VISTA, 'inventado'), { estado: inicial, enviar: null })
})

test('si lo que se confirmaba desaparece o ya no se ofrece, la pregunta se cierra', () => {
  const pidiendo = tocar(inicial, VISTA, 'pedido:1').estado
  assert.deepEqual(alLlegar(pidiendo, VISTA), pidiendo)
  const hecho = { ...VISTA, pedidos: [{ id: 'pedido:1', tecla: 'a', ejecutar: false }] }
  assert.deepEqual(alLlegar(pidiendo, hecho), inicial)
  assert.equal(si(pidiendo, hecho).enviar, null)
  assert.deepEqual(alLlegar(pidiendo, { motivo: 'sin datos' }), inicial)
})
