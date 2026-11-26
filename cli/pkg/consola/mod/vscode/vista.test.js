const { test } = require('node:test')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const vm = require('node:vm')
const { vista } = require('./panel.js')

// Contrato de la ficha estado (RFC 0014): todo texto que llega de qdd (una etiqueta, un comando que dejó una IA) entra
// al webview con textContent, nunca como HTML. Los scripts del webview corren aquí sobre un DOM mínimo que anota cada
// intento de escribir HTML.

function domFalso() {
  const html = []
  const nodo = (etiqueta) => {
    const n = {
      etiqueta,
      hijos: [],
      textContent: '',
      className: '',
      classList: { add() {} },
      style: { setProperty() {} },
      append(...hs) {
        n.hijos.push(...hs)
      },
      replaceChildren(...hs) {
        n.hijos = hs
      },
      addEventListener() {},
    }
    for (const prop of ['innerHTML', 'outerHTML']) {
      Object.defineProperty(n, prop, { set: (v) => html.push(prop + '=' + v), get: () => '' })
    }
    n.insertAdjacentHTML = (_, v) => html.push('insertAdjacentHTML=' + v)
    return n
  }
  const raiz = nodo('main')
  const oyentes = {}
  const documento = {
    createElement: nodo,
    createTextNode: (texto) => ({ etiqueta: '#text', textContent: texto, hijos: [] }),
    getElementById: () => raiz,
    documentElement: nodo('html'),
    write: (v) => html.push('document.write=' + v),
  }
  const ventana = { addEventListener: (tipo, fn) => (oyentes[tipo] = fn) }
  const enviados = []
  const contexto = vm.createContext({ document: documento, window: ventana, acquireVsCodeApi: () => ({ postMessage: (m) => enviados.push(m) }) })
  ventana.window = ventana
  return { contexto, raiz, oyentes, html, ventana }
}

function textos(n, out = []) {
  if (n.textContent) out.push(n.textContent)
  for (const h of n.hijos || []) textos(h, out)
  return out
}

const MALO = '<img src=x onerror="alert(1)">'

// contrato: estado#7188db80
test('el texto de qdd entra al webview con textContent: una etiqueta con HTML se ve tal cual y no se interpreta', () => {
  const d = domFalso()
  for (const archivo of ['pasos.js', 'materializar.js', 'vista.js']) {
    const fuente = fs.readFileSync(path.join(__dirname, 'media', archivo), 'utf8')
    vm.runInContext(fuente, d.contexto)
  }
  const datos = {
    version: 'v9',
    carpeta: '/x',
    motivo: '',
    estado: { abreviaturas: ['Có'], frentes: [{ id: 'a', nombre: MALO, simbolos: ['⏳'] }], esperas: [], fallas: [{ frente: MALO, etapa: 'Có', texto: MALO }] },
    acciones: [
      { id: 'pedido:1', tipo: 'pedido', tecla: 'a', etiqueta: MALO, detalle: 'echo "' + MALO + '"', estado: 'pendiente', confirmar: true, carpeta: '/x' },
      { id: 'aprobar:x', tipo: 'aprobar', etiqueta: MALO, detalle: MALO, confirmar: true },
    ],
  }
  d.oyentes.message({ data: { tipo: 'vista', vista: vista(datos, '') } })
  assert.deepEqual(d.html, [], 'ningún texto entra como HTML')
  const vistos = textos(d.raiz)
  assert.ok(vistos.filter((t) => t.includes(MALO)).length >= 4, 'la etiqueta, el comando, la fila y la falla se ven tal cual: ' + vistos.join(' | '))
})
