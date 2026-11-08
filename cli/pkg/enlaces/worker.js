// Worker de QDD para compartir flujos (RFC 0025, fase 3b). Sirve solo /f/<id>: la página que qdd guardó en KV bajo
// ese id. No guarda nada, no lleva secretos y el token de una invitación nunca le llega: viaja en el fragmento (#t=…),
// que el navegador no manda. Lo instala y lo actualiza «qdd flujo compartir --enlace».
const SEGURIDAD = {
  'cache-control': 'no-store',
  'referrer-policy': 'no-referrer',
  'x-content-type-options': 'nosniff',
  'x-frame-options': 'DENY',
  'x-robots-tag': 'noindex, nofollow',
}

function texto(estado, mensaje) {
  return new Response(mensaje, { status: estado, headers: { ...SEGURIDAD, 'content-type': 'text/plain; charset=utf-8' } })
}

export default {
  async fetch(request, env) {
    if (request.method !== 'GET' && request.method !== 'HEAD') return texto(405, 'Solo se lee.')
    const m = /^\/f\/([a-z2-7]{26})$/.exec(new URL(request.url).pathname)
    if (!m) return texto(404, 'No hay nada aquí.')
    const pagina = await env.FLUJOS.get('f/' + m[1])
    if (pagina === null) return texto(404, 'Este enlace ya no está disponible: pídele uno nuevo a quien te lo mandó.')
    const cuerpo = request.method === 'HEAD' ? null : pagina
    return new Response(cuerpo, { headers: { ...SEGURIDAD, 'content-type': 'text/html; charset=utf-8' } })
  },
}
