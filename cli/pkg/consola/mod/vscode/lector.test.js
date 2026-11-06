const { test } = require('node:test')
const assert = require('node:assert/strict')
const { EventEmitter } = require('node:events')
const { crearLector, ARGS_SEGUIR, ARGS_PANEL, INTERVALO_MS } = require('./lector.js')

function proceso() {
  const p = new EventEmitter()
  p.stdout = new EventEmitter()
  p.stderr = new EventEmitter()
  p.matado = false
  p.kill = () => {
    p.matado = true
  }
  return p
}

// mesa: un lector con el proceso, la lectura y el reloj simulados; nada se ejecuta de verdad.
function mesa(respuesta) {
  const m = { abiertos: [], lecturas: [], datos: [], fallas: [], tiempos: [] }
  m.lector = crearLector({
    abrir: (args) => {
      assert.deepEqual(args, ARGS_SEGUIR)
      const p = proceso()
      m.abiertos.push(p)
      return p
    },
    leer: (args, listo) => {
      assert.deepEqual(args, ARGS_PANEL)
      m.lecturas.push(listo)
      if (respuesta) listo(...respuesta())
    },
    alDatos: (d) => m.datos.push(d),
    alFallar: (f) => m.fallas.push(f),
    programar: (fn, ms) => {
      const t = { fn, ms, cancelado: false }
      m.tiempos.push(t)
      return t
    },
    cancelar: (t) => {
      t.cancelado = true
    },
  })
  m.correrTiempo = () => {
    const t = m.tiempos.shift()
    assert.ok(t, 'debía haber un tiempo programado')
    if (!t.cancelado) t.fn()
    return t
  }
  return m
}

const linea = (o) => Buffer.from(JSON.stringify(o) + '\n')

test('sigue a qdd: una línea es un dibujo y el latido vacío no redibuja', () => {
  const m = mesa()
  m.lector.iniciar()
  assert.equal(m.abiertos.length, 1)
  m.abiertos[0].stdout.emit('data', linea({ n: 1 }))
  m.abiertos[0].stdout.emit('data', Buffer.from('\n'))
  m.abiertos[0].stdout.emit('data', linea({ n: 2 }))
  assert.deepEqual(m.datos, [{ n: 1 }, { n: 2 }])
  assert.equal(m.lecturas.length, 0, 'con seguir no se lanza una lectura por vez (MEJ-020)')
})

test('arma una línea que llega en pedazos, también si corta un carácter', () => {
  const m = mesa()
  m.lector.iniciar()
  const todo = linea({ t: 'ñandú ✅' })
  m.abiertos[0].stdout.emit('data', todo.subarray(0, 8))
  assert.deepEqual(m.datos, [])
  m.abiertos[0].stdout.emit('data', todo.subarray(8))
  assert.deepEqual(m.datos, [{ t: 'ñandú ✅' }])
})

test('una línea igual a la anterior no redibuja', () => {
  const m = mesa()
  m.lector.iniciar()
  m.abiertos[0].stdout.emit('data', linea({ n: 1 }))
  m.abiertos[0].stdout.emit('data', linea({ n: 1 }))
  assert.equal(m.datos.length, 1)
})

test('una línea que no es JSON no rompe la vista y lo dice', () => {
  const m = mesa()
  m.lector.iniciar()
  m.abiertos[0].stdout.emit('data', linea({ n: 1 }))
  m.abiertos[0].stdout.emit('data', Buffer.from('no soy json\n'))
  assert.equal(m.fallas.length, 1)
  m.abiertos[0].stdout.emit('data', linea({ n: 2 }))
  assert.deepEqual(m.datos, [{ n: 1 }, { n: 2 }])
})

// Visto con el binario de origin/main el 2026-11-05: 'qdd consola seguir' sin ese comando imprime la ayuda y sale con 0.
test('un qdd sin seguir imprime la ayuda y sale con 0: lee panel --json, no reabre el flujo', () => {
  const m = mesa(() => [null, JSON.stringify({ n: 1 }), ''])
  m.lector.iniciar()
  m.abiertos[0].stdout.emit('data', Buffer.from('Un solo interruptor para la consola de Claude Code:\n\n  qdd consola activar\n'))
  m.abiertos[0].emit('close', 0)
  assert.deepEqual(m.fallas, [], 'la ayuda no es una falla que la persona deba ver')
  assert.deepEqual(m.datos, [{ n: 1 }])
  assert.equal(m.lecturas.length, 1)
  m.correrTiempo()
  assert.equal(m.abiertos.length, 1)
})

test('una lectura que no entrega JSON lo dice', () => {
  const m = mesa(() => [null, 'no soy json', ''])
  m.lector.iniciar()
  m.abiertos[0].emit('close', 1)
  assert.match(m.fallas[0], /no es JSON/)
})

test('un qdd sin seguir pasa a leer panel --json cada 2 s', () => {
  const m = mesa(() => [null, JSON.stringify({ n: 1 }), ''])
  m.lector.iniciar()
  m.abiertos[0].emit('close', 1)
  assert.equal(m.lecturas.length, 1)
  assert.deepEqual(m.datos, [{ n: 1 }])
  assert.equal(m.tiempos.at(-1).ms, INTERVALO_MS)
  m.correrTiempo()
  assert.equal(m.lecturas.length, 2)
  assert.equal(m.datos.length, 1, 'lo mismo no redibuja')
  assert.equal(m.abiertos.length, 1, 'no vuelve a intentar seguir')
})

test('sin el binario, seguir falla con error y la lectura dice el motivo', () => {
  const sinBinario = Object.assign(new Error('spawn qdd ENOENT'), { code: 'ENOENT' })
  const m = mesa(() => [sinBinario, '', ''])
  m.lector.iniciar()
  m.abiertos[0].emit('error', sinBinario)
  m.abiertos[0].emit('close', -2)
  assert.equal(m.lecturas.length, 1, 'error y close son una sola salida')
  assert.match(m.fallas[0], /no se encontró qdd/i)
})

test('un qdd que sale con error y sin stderr pide actualizarlo', () => {
  const m = mesa(() => [Object.assign(new Error('Command failed'), { code: 1 }), '', ''])
  m.lector.iniciar()
  m.abiertos[0].emit('close', 1)
  assert.match(m.fallas[0], /actualiza qdd/)
})

test('al volver el qdd tras un fallo, el dibujo vuelve', () => {
  let respuesta = [new Error('x'), '', 'qdd no responde']
  const m = mesa(() => respuesta)
  m.lector.iniciar()
  m.abiertos[0].emit('close', 1)
  assert.deepEqual(m.fallas, ['qdd no responde'])
  respuesta = [null, JSON.stringify({ n: 1 }), '']
  m.correrTiempo()
  assert.deepEqual(m.datos, [{ n: 1 }])
})

test('la misma falla no se repite', () => {
  const m = mesa(() => [new Error('x'), '', 'qdd no responde'])
  m.lector.iniciar()
  m.abiertos[0].emit('close', 1)
  m.correrTiempo()
  assert.equal(m.fallas.length, 1)
})

test('un flujo que termina después de entregar datos se vuelve a abrir (qdd se actualizó)', () => {
  const m = mesa()
  m.lector.iniciar()
  m.abiertos[0].stdout.emit('data', linea({ n: 1 }))
  m.abiertos[0].emit('close', 0)
  assert.equal(m.lecturas.length, 0)
  m.correrTiempo()
  assert.equal(m.abiertos.length, 2)
})

test('detener mata el proceso y no deja nada programado', () => {
  const m = mesa()
  m.lector.iniciar()
  m.lector.detener()
  assert.equal(m.abiertos[0].matado, true)
  m.abiertos[0].emit('close', null)
  assert.equal(m.lecturas.length, 0, 'su cierre no pasa a leer')
  assert.equal(m.tiempos.length, 0)
})

test('detener cancela la próxima lectura y una respuesta atrasada no dibuja', () => {
  const m = mesa(() => [null, JSON.stringify({ n: 1 }), ''])
  m.lector.iniciar()
  m.abiertos[0].emit('close', 1)
  m.lector.detener()
  assert.equal(m.correrTiempo().cancelado, true)
  assert.equal(m.lecturas.length, 1)
  const tarde = mesa()
  tarde.lector.iniciar()
  tarde.abiertos[0].emit('close', 1)
  tarde.lector.detener()
  tarde.lecturas[0](null, JSON.stringify({ n: 9 }), '')
  assert.deepEqual(tarde.datos, [])
})

test('iniciar dos veces no abre dos procesos, y al volver dibuja aunque nada haya cambiado', () => {
  const m = mesa()
  m.lector.iniciar()
  m.lector.iniciar()
  assert.equal(m.abiertos.length, 1)
  m.abiertos[0].stdout.emit('data', linea({ n: 1 }))
  m.lector.detener()
  m.lector.iniciar()
  m.abiertos[1].stdout.emit('data', linea({ n: 1 }))
  assert.equal(m.datos.length, 2)
})
