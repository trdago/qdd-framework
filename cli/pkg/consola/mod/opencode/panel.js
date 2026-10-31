// Lógica pura del panel de qdd en OpenCode (RFC 0006 §3.10): convierte lo que entrega 'qdd consola panel --json' en lo
// que dibuja la barra lateral. Solo lectura: no ofrece acciones. Sin dependencias, para probarla con node --test.

export const INTERVALO_MS = 2000
export const TOPE_MS = 3000
// Tarea en curso (RFC 0006 §3.13, blueprint estado-por-tareas): el ● late con un tic local y destella con cada pulso.
export const LATIDO_MS = 500
export const PULSO_MIN_MS = 5000

// celdas alinea los símbolos con las abreviaturas: ✅ ⏳ ❌ ocupan 2 columnas; — (no aplica), · (sin empezar) y ? una.
const ANGOSTOS = new Set(['—', '·', '?'])
export function celdas(simbolos) {
  return simbolos.map((s) => (ANGOSTOS.has(s) ? s + ' ' : s)).join(' ').trimEnd()
}

// total: el de qdd (el mismo que la barra, FND-086); un qdd anterior no lo trae y se cuenta la lista.
export function total(n, lista) {
  return typeof n === 'number' ? n : lista.length
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

// AVISO_VIEJO: qdd marca viejo un tablero con tareas pendientes que nadie tocó en horas (FND-088).
const AVISO_VIEJO = (hace) => '⚠ Sin actualizar ' + hace + ': lo que dice ⏳ puede haber cambiado'

// vista arma lo que se dibuja; con datos nulos, el motivo (sin qdd, qdd antiguo o sin estado.yaml).
export function vista(datos, falla) {
  if (!datos) return { motivo: falla || 'sin datos de qdd todavía' }
  const fichas = (datos.acciones || []).filter((a) => a.tipo === 'aprobar').length
  const e = datos.estado
  if (!e) return { motivo: datos.motivo || 'el proyecto no tiene estado por frente', fichas, version: datos.version || '' }
  const cuando = e.hace || hora(e.actualizado)
  return {
    titulo: 'Tareas' + (cuando ? ' · ' + cuando : ''),
    aviso: e.viejo ? AVISO_VIEJO(e.hace) : '',
    version: datos.version || '',
    resumen: [
      { texto: cuantas(e), tono: 'primary' },
      { texto: '⏳ ' + total(e.esperan, e.esperas), tono: 'warning' },
      { texto: '❌ ' + total(e.fallan, e.fallas), tono: 'error' },
      ...(paraTi(datos.acciones) > 0 ? [{ texto: '▶ ' + paraTi(datos.acciones) + ' para ti', tono: 'warning' }] : []),
    ],
    cabecera: e.abreviaturas.join(' '),
    filas: e.frentes.map((f) => ({
      id: f.id,
      nombre: f.nombre,
      celdas: celdas(f.simbolos),
      tono: tono(f.simbolos),
      running: !!f.actividad?.running,
      pulso: f.actividad?.pulso || '',
    })),
    actual: e.actual || '',
    ocultas: e.ocultos || 0,
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

// destellosNuevos devuelve los ids cuyo pulso cambió respecto de la lectura anterior (vistos: id → pulso; se actualiza).
export function destellosNuevos(vistos, filas) {
  const nuevos = []
  for (const f of filas || []) {
    if (!f.pulso) continue
    if (vistos[f.id] && vistos[f.id] !== f.pulso) nuevos.push(f.id)
    vistos[f.id] = f.pulso
  }
  return nuevos
}

// brillo del ● en un tic: «acento» si destella, si no alterna «brillante» y «tenue» cada 2 tics (1 s).
export function brillo(tic, destella) {
  if (destella) return 'acento'
  return Math.floor(tic / 2) % 2 === 0 ? 'brillante' : 'tenue'
}

// fuenteDePulso dice si un evento de OpenCode es una herramienta del agente que empieza a correr: «subagente» para
// la herramienta task, «agente» para las demás y '' si no es una herramienta en curso.
export function fuenteDePulso(evento) {
  const parte = evento?.properties?.part
  if (!parte || parte.type !== 'tool' || parte.state?.status !== 'running') return ''
  return parte.tool === 'task' ? 'subagente' : 'agente'
}

// tocaEnviar aplica el tope de un pulso a la CLI cada PULSO_MIN_MS.
export function tocaEnviar(ahora, ultimo) {
  return ahora - ultimo >= PULSO_MIN_MS
}

// paraTi cuenta los comandos que la IA dejó para la persona y que todavía se pueden ejecutar (pendientes o con falla;
// un qdd anterior a 2.6.2 no trae estado: cuenta como pendiente; uno en espera de otro no cuenta, FND-080). Se
// ejecutan desde Claude Code.
export function paraTi(acciones) {
  return (acciones || []).filter((a) => a.tipo === 'pedido' && (!a.estado || a.estado === 'pendiente' || a.estado === 'fallo')).length
}
