// Materializar en el webview (RFC 0014, fase D): como el panel de Claude Code (hypr-materialize), la vista anima solo lo
// que cambió entre una lectura y la siguiente, con los efectos que dice qdd ('qdd consola efectos'). Lógica pura: el
// webview la carga como script y node --test como módulo.
(function (global) {
  const TOPE_MS = 2000
  const HEX = /^#[0-9a-f]{3,8}$/i

  // claves: lo que se puede animar por separado (pedido, ficha, fila), con la firma de lo que muestra.
  function claves(vista) {
    const out = new Map()
    for (const p of vista?.pedidos || []) out.set('pedido:' + p.id, JSON.stringify(p))
    for (const f of vista?.fichas || []) out.set('ficha:' + f.id, JSON.stringify(f))
    for (const f of vista?.filas || []) {
      if (f.id) out.set('fila:' + f.id, JSON.stringify(f))
    }
    return out
  }

  // cambiados: lo nuevo o distinto respecto de la vista anterior. La primera no anima: todo sería nuevo.
  function cambiados(anterior, nueva) {
    const out = new Set()
    if (!anterior) return out
    const antes = claves(anterior)
    for (const [clave, firma] of claves(nueva)) {
      if (antes.get(clave) !== firma) out.add(clave)
    }
    return out
  }

  // estilo: si anima y con qué; el acento entra al CSS, así que solo pasa un color #hex.
  function estilo(efectos) {
    if (!efectos?.activo) return { animar: false, acento: '', ms: 0 }
    const acento = HEX.test(String(efectos.acento || '')) ? efectos.acento : ''
    const ms = Math.min(Math.max(Number(efectos.abrir_ms) || 0, 0), TOPE_MS)
    return { animar: ms > 0, acento, ms }
  }

  const materializar = { claves, cambiados, estilo }
  if (typeof module === 'object' && module.exports) module.exports = materializar
  global.qddMaterializar = materializar
})(typeof window === 'undefined' ? globalThis : window)
