// Lógica pura del panel de qdd en OpenCode (RFC 0006 §3.10): convierte lo que entrega 'qdd consola panel --json' en lo
// que dibuja la barra lateral. Solo lectura: no ofrece acciones. Sin dependencias, para probarla con node --test.

export const INTERVALO_MS = 2000
export const TOPE_MS = 3000

// celdas alinea los símbolos con las abreviaturas: ✅ ⏳ ❌ ocupan 2 columnas y — una.
export function celdas(simbolos) {
  return simbolos.map((s) => (s === '—' || s === '?' ? s + ' ' : s)).join(' ').trimEnd()
}

// tono de una tarea: el de su peor etapa.
export function tono(simbolos) {
  if (simbolos.includes('❌')) return 'error'
  if (simbolos.includes('⏳')) return 'warning'
  return 'success'
}

export function hora(iso) {
  const t = new Date(iso)
  if (Number.isNaN(t.getTime())) return ''
  const dos = (n) => String(n).padStart(2, '0')
  return dos(t.getHours()) + ':' + dos(t.getMinutes())
}

export function cuantas(e) {
  const n = e.frentes.length
  const ocultos = e.ocultos || 0
  if (ocultos > 0) return n + ' de ' + (n + ocultos) + ' tareas'
  return n + (n === 1 ? ' tarea' : ' tareas')
}

// vista arma lo que se dibuja; con datos nulos, el motivo (sin qdd, qdd antiguo o sin estado.yaml).
export function vista(datos, falla) {
  if (!datos) return { motivo: falla || 'sin datos de qdd todavía' }
  const fichas = (datos.acciones || []).filter((a) => a.tipo === 'aprobar').length
  const e = datos.estado
  if (!e) return { motivo: datos.motivo || 'el proyecto no tiene estado por frente', fichas, version: datos.version || '' }
  const cuando = hora(e.actualizado)
  return {
    titulo: 'Tareas' + (cuando ? ' · ' + cuando : ''),
    version: datos.version || '',
    resumen: [
      { texto: cuantas(e), tono: 'primary' },
      { texto: '⏳ ' + e.esperas.length, tono: 'warning' },
      { texto: '❌ ' + e.fallas.length, tono: 'error' },
    ],
    cabecera: e.abreviaturas.join(' '),
    filas: e.frentes.map((f) => ({ id: f.id, nombre: f.nombre, celdas: celdas(f.simbolos), tono: tono(f.simbolos) })),
    ocultas: e.ocultos || 0,
    esperas: e.esperas.map((p) => ({ titulo: p.frente + ' › ' + p.etapa, texto: p.texto + ' · de ' + (p.de_quien || '¿?') })),
    fallas: e.fallas.map((p) => ({ titulo: p.frente + ' › ' + p.etapa, texto: p.texto })),
    fichas,
  }
}

// motivoDeSalida explica por qué 'qdd consola panel --json' no entregó datos.
export function motivoDeSalida(codigo, stderr) {
  if (codigo === 0) return ''
  const texto = (stderr || '').trim()
  return texto || 'esta versión de qdd no trae «qdd consola panel»: actualiza qdd'
}
