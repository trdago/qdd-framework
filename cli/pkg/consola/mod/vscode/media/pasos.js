// Pasos de la confirmación en el webview (RFC 0014, fase B): tocar un pedido o una ficha (o la tecla del pedido)
// pregunta «s  Sí / n  No» y solo el Sí manda su id a la extensión. Lógica pura: el webview la carga como script y
// node --test como módulo. Cada función devuelve el estado nuevo y lo que hay que mandar (o null).
(function (global) {
  const inicial = Object.freeze({ confirmando: '' })
  const quieto = (estado) => ({ estado, enviar: null })

  function ofrecidos(vista) {
    return [...(vista?.pedidos || []), ...(vista?.fichas || [])].filter((a) => a.ejecutar)
  }

  const ofrece = (vista, id) => ofrecidos(vista).some((a) => a.id === id)

  function tocar(estado, vista, id) {
    if (!ofrece(vista, id)) return quieto(estado)
    return quieto({ confirmando: id })
  }

  function si(estado, vista) {
    if (!estado.confirmando || !ofrece(vista, estado.confirmando)) return quieto(inicial)
    return { estado: inicial, enviar: { tipo: 'accion', id: estado.confirmando } }
  }

  const no = () => quieto(inicial)

  function tecla(estado, vista, k) {
    if (estado.confirmando && k === 's') return si(estado, vista)
    if (estado.confirmando && (k === 'n' || k === 'Escape')) return no()
    if (estado.confirmando) return quieto(estado)
    const pedido = ofrecidos(vista).find((a) => a.tecla && a.tecla === k)
    if (!pedido) return quieto(estado)
    return tocar(estado, vista, pedido.id)
  }

  // alLlegar: con cada vista nueva, una pregunta por algo que ya no se ofrece se cierra.
  function alLlegar(estado, vista) {
    if (!estado.confirmando || ofrece(vista, estado.confirmando)) return estado
    return inicial
  }

  const pasos = { inicial, tocar, si, no, tecla, alLlegar }
  if (typeof module === 'object' && module.exports) module.exports = pasos
  global.qddPasos = pasos
})(typeof window === 'undefined' ? globalThis : window)
