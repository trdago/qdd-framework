// Asistente: un formulario de una pregunta por pantalla para el panel (la persona eligió la opción B, 2026-11-20, tras
// «cuando escribo encima y termino de escribir aprieto enter no se entiende qué pasó»). El panel es angosto: un
// nombre al lado del campo se partía en dos líneas y Enter no daba ninguna señal. Aquí cada pregunta va sola, en una
// línea propia sobre su campo, y Enter la responde y pasa a la siguiente; lo ya respondido queda arriba como «✓ Sitio:
// Cloudflare», y un clic ahí vuelve a esa pregunta. El último paso ('fin') lo dibuja quien usa el asistente.
//
// Es puro: el estado vive en quien lo usa, en un objeto con una propiedad por pregunta (su respuesta), más `paso` (la
// pregunta a la vista), `vistos` (las ya respondidas) y `aviso` (por qué Enter no avanzó).
//
// Una pregunta: { id, etiqueta (para el resumen), pregunta (texto o función del estado), tipo: 'texto' | 'opciones',
// opciones (función del estado), ejemplo, ayuda, valida (función de valor y estado: '' si vale), visible (función del
// estado), valor (lo que muestra antes de responderla, si no es la respuesta guardada), aparte (no está en el
// recorrido: se abre con un clic desde el fin, con filasAparte) }. El 'fin' va último en la lista: { id: 'fin', tipo:
// 'fin' }.
//
// Un campo nunca queda abierto después de Enter: Claude Code lo vacía al enviarlo y, si el panel lo vuelve a dibujar
// con el mismo texto, no lo repone (visto en un panel real, 2026-11-20). Por eso Enter siempre lleva a otro paso.

const texto = (x, s) => (typeof x === 'function' ? x(s) : x || '')

export const visibles = (pasos, s) => pasos.filter((p) => !p.visible || p.visible(s))
const recorrido = (pasos, s) => visibles(pasos, s).filter((p) => !p.aparte)

// siguiente: la primera pregunta del recorrido después de `id` que falta responder; si no queda ninguna (o `id` es
// de las aparte), el fin.
export function siguiente(pasos, s, id) {
  const lista = recorrido(pasos, s)
  const desde = lista.findIndex((p) => p.id === id)
  const falta = desde < 0 ? null : lista.slice(desde + 1).find((p) => p.tipo === 'fin' || !(s.vistos || []).includes(p.id))
  return (falta || lista[lista.length - 1]).id
}

// responder guarda la respuesta y avanza; si no vale, se queda en la pregunta con el aviso.
export function responder(pasos, s, id, valor) {
  const paso = pasos.find((p) => p.id === id)
  const v = String(valor ?? '').trim()
  const malo = paso?.valida ? paso.valida(v, { ...s, [id]: v }) : ''
  if (malo) return { ...s, [id]: v, aviso: malo }
  const nuevo = { ...s, [id]: v, aviso: '', vistos: [...new Set([...(s.vistos || []), id])] }
  return { ...nuevo, paso: siguiente(pasos, nuevo, id) }
}

export const irA = (s, id) => ({ ...s, paso: id, aviso: '' })

// atras: la pregunta anterior del recorrido; desde una aparte, el fin.
export function atras(pasos, s) {
  const lista = recorrido(pasos, s)
  const i = lista.findIndex((p) => p.id === s.paso)
  if (i < 0) return irA(s, 'fin')
  if (i === 0) return s
  return irA(s, lista[i - 1].id)
}

const valorDe = (p, s) => (p.valor ? p.valor(s) : s[p.id] || '')

// resumen: el valor de una respuesta como se lee en «✓ Sitio: Cloudflare» (la etiqueta de la opción, no su valor).
function resumen(p, s) {
  const v = valorDe(p, s)
  if (p.tipo !== 'opciones') return v || '—'
  const o = texto(p.opciones, s).find((x) => x.value === v)
  return o ? o.label : v || '—'
}

const corto = (t) => (t.length > 40 ? t.slice(0, 39) + '…' : t)

// filasAparte: las preguntas aparte como filas «Vence el: —»; un clic abre esa pregunta y Enter vuelve al fin.
export function filasAparte({ Box, Button }, s, pasos, a) {
  const filas = visibles(pasos, s)
    .filter((p) => p.aparte)
    .map((p) => Button({ key: a.prefijo + 'ir-' + p.id, label: corto(p.etiqueta + ': ' + resumen(p, s)), plain: true, onPress: () => a.onIr(p.id) }))
  return Box({ key: a.prefijo + 'aparte', flexDirection: 'column', children: filas })
}

// dibujar: el título con el avance, lo ya respondido, la pregunta a la vista (o el fin) y Atrás / Cancelar.
// a = { prefijo, titulo, conAvance, volverAlFin (el fin es el inicio, como al modificar: ahí no hay Atrás), colores: { titulo, pregunta, secundario, falla }, fin: [elementos],
//       onCambio(id, v), onResponder(id, v), onIr(id), onAtras(), onCancelar() }
export function dibujar(c, s, pasos, a) {
  const { Box, Text, Button, Input, Select } = c
  const lista = recorrido(pasos, s)
  const paso = visibles(pasos, s).find((p) => p.id === s.paso) || lista[0]
  const i = lista.indexOf(paso)
  const titulo = a.titulo + (a.conAvance && i >= 0 ? ' · ' + (i + 1) + ' de ' + lista.length : '')
  const hechos = lista.filter((p) => p.tipo !== 'fin' && p.id !== paso.id && (s.vistos || []).includes(p.id))
  const filas = hechos.map((p) => Button({ key: a.prefijo + 'ir-' + p.id, label: corto('✓ ' + p.etiqueta + ': ' + resumen(p, s)), plain: true, onPress: () => a.onIr(p.id) }))
  const cuerpo = paso.tipo === 'fin' ? a.fin : preguntaDe(c, s, paso, a, i === lista.length - 2 || paso.aparte)
  const botones = [
    ...(i !== 0 && !(a.volverAlFin && paso.tipo === 'fin') ? [Button({ key: a.prefijo + 'atras', label: '← Atrás', plain: true, dimColor: true, onPress: a.onAtras })] : []),
    Button({ key: a.prefijo + 'cancelar', label: 'Cancelar', plain: true, dimColor: true, onPress: a.onCancelar }),
  ]
  return [
    Text({ key: a.prefijo + 'titulo', weight: 'bold', color: a.colores.titulo, wrap: 'wrap', children: [titulo] }),
    ...(filas.length ? [Box({ key: a.prefijo + 'hechos', flexDirection: 'column', children: filas })] : []),
    ...cuerpo,
    Box({ key: a.prefijo + 'botones', flexDirection: 'row', flexWrap: 'wrap', columnGap: 2, children: botones }),
  ]
}

function preguntaDe({ Text, Input, Select }, s, p, a, ultima) {
  const key = a.prefijo + p.id
  const campo =
    p.tipo === 'opciones'
      ? Select({ key, options: texto(p.opciones, s), value: valorDe(p, s) || undefined, onSelect: (v) => a.onResponder(p.id, v) })
      : Input({ key, placeholder: texto(p.ejemplo, s), value: valorDe(p, s), submitLabel: ultima ? 'listo' : 'siguiente', onInput: (v) => a.onCambio(p.id, v), onSubmit: (v) => a.onResponder(p.id, v) })
  const ayuda = texto(p.ayuda, s)
  return [
    Text({ key: key + '-pregunta', weight: 'bold', color: a.colores.pregunta, wrap: 'wrap', children: [texto(p.pregunta, s)] }),
    campo,
    ...(ayuda ? [Text({ key: key + '-ayuda', color: a.colores.secundario, wrap: 'wrap', children: [ayuda] })] : []),
    ...(s.aviso ? [Text({ key: key + '-aviso', color: a.colores.falla, wrap: 'wrap', children: ['⚠ ' + s.aviso] })] : []),
  ]
}
