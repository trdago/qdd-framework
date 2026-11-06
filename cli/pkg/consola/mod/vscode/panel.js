// Lógica pura del panel de qdd en Antigravity y VS Code (RFC 0014): convierte lo que entrega 'qdd consola panel --json'
// en lo que dibuja la vista. La extensión dibuja y qdd decide: aquí no hay reglas de pedidos, fichas ni PRs. Sin
// dependencias, para probarla con node --test.

const path = require('node:path')

const hora = (iso) => {
  const t = new Date(iso)
  if (!iso || Number.isNaN(t.getTime())) return ''
  const dos = (n) => String(n).padStart(2, '0')
  return dos(t.getHours()) + ':' + dos(t.getMinutes())
}

// nombre: el último segmento de una ruta (con / o \): del proyecto se ve solo el nombre de la carpeta (FND-090).
const nombre = (ruta) => String(ruta || '').replace(/[\\/]+$/, '').split(/[\\/]/).pop()

function enProyecto(raiz, ruta) {
  if (!ruta) return ''
  if (raiz && (ruta + '/').startsWith(raiz + '/')) return nombre(raiz) + ruta.slice(raiz.length).replace(/\\/g, '/')
  return nombre(ruta)
}

// primeraLinea: un comando de varias líneas se ve en una, con cuántas faltan.
function primeraLinea(comando) {
  const lineas = String(comando || '').trim().split('\n')
  if (lineas.length === 1) return lineas[0]
  return lineas[0] + '  … (+' + (lineas.length - 1) + ' líneas)'
}

const ultimaLinea = (texto) => String(texto || '').trim().split('\n').pop()
const ultimasDos = (texto) => String(texto || '').trim().split('\n').slice(-2).join('\n')
const esperaDe = (a) => (a.espera || []).map((t) => '[' + t + ']').join(' ') || 'otro pedido'
const unir = (...partes) => partes.filter(Boolean).join(' · ')
const conTexto = (texto, tono) => (texto ? [{ texto, tono }] : [])

// ejecutable: se puede confirmar. Uno que espera a otro también (con su tecla): queda armado y corre cuando el otro
// termine. Un qdd anterior a 2.6.2 no trae estado: es pendiente.
function ejecutable(a) {
  const estado = a.estado || 'pendiente'
  return estado === 'pendiente' || estado === 'fallo' || (estado === 'espera' && Boolean(a.tecla))
}

// notasDeArmado: ya tiene la intención de la persona. Primero cuándo corre; lo que dijo el comando, atenuado debajo.
function notasDeArmado(a) {
  const cuando = a.espera?.length ? 'cuando ' + esperaDe(a) + ' termine' : 'cuando esté listo'
  const intento = a.intentos ? 'intento ' + a.intentos : ''
  const proximo = !a.espera?.length && hora(a.proximo) ? 'reintenta ' + hora(a.proximo) : ''
  return [{ texto: unir('Armado', 'corre solo ' + cuando, intento, proximo), tono: 'espera' }, ...conTexto(ultimaLinea(a.aviso), 'tenue')]
}

// tarjeta arma un pedido: cada estado resuelve lo suyo y sale; lo que queda es un pendiente.
function tarjeta(a, raiz) {
  const estado = a.estado || 'pendiente'
  const donde = enProyecto(raiz, a.carpeta)
  const lugar = donde ? 'en ' + donde : ''
  const base = { id: a.id, etiqueta: a.etiqueta, comando: primeraLinea(a.detalle), tecla: '', marca: '', tono: 'normal', meta: lugar, notas: [] }
  if (estado === 'espera' && !a.tecla) return { ...base, marca: '◌', tono: 'tenue', meta: unir('espera a ' + esperaDe(a), lugar) }
  if (estado === 'espera') return { ...base, tecla: a.tecla, meta: unir('espera a ' + esperaDe(a), 'confírmalo y corre solo', lugar) }
  if (estado === 'armado') return { ...base, marca: '◷', tono: 'espera', notas: notasDeArmado(a) }
  if (estado === 'ejecutando') {
    const desde = hora(a.inicio) ? 'ejecutando desde ' + hora(a.inicio) : 'ejecutando…'
    return { ...base, marca: '⟳', tono: 'tenue', meta: unir(desde, lugar) }
  }
  if (estado === 'hecho') {
    const cuando = hora(a.fin) ? 'hecho ' + hora(a.fin) : 'hecho'
    return { ...base, marca: '✓', tono: 'tenue', comando: '', meta: cuando, notas: conTexto(ultimaLinea(a.resultado), 'tenue') }
  }
  if (estado === 'fallo') {
    const cuando = hora(a.fin) ? 'falló ' + hora(a.fin) : 'falló'
    return { ...base, tecla: a.tecla || '', marca: '✗', tono: 'falla', meta: unir(cuando, lugar), notas: conTexto(ultimasDos(a.resultado), 'falla') }
  }
  const todaviaNo = a.aviso ? [{ texto: unir('◷ Todavía no', hora(a.aviso_en)), tono: 'espera' }, ...conTexto(ultimaLinea(a.aviso), 'tenue')] : []
  return { ...base, tecla: a.tecla || '', notas: todaviaNo }
}

// tonoDeTarea: la tarea toma el color de su peor etapa.
function tonoDeTarea(simbolos) {
  if (simbolos.includes('❌')) return 'falla'
  if (simbolos.includes('⏳')) return 'espera'
  return 'listo'
}

// filaDeTarea es una tarea: el prefijo del árbol y a quién espera van en el nombre.
function filaDeTarea(f) {
  return {
    id: f.id,
    nombre: (f.prefijo || '') + f.nombre + (f.notas?.length ? ' · ' + f.notas.join(' · ') : ''),
    simbolos: f.simbolos,
    tono: tonoDeTarea(f.simbolos),
    running: Boolean(f.actividad?.running),
    objetivo: false,
  }
}

const filaDeObjetivo = (id, titulo, resumen) => ({ id: 'objetivo:' + id, nombre: titulo, resumen, simbolos: [], tono: 'primario', running: false, objetivo: true })

// filas: con objetivos, cada uno encabeza sus tareas (el foco primero, como los entrega qdd) y las tareas sin objetivo
// van al final. Un objetivo que sigue su propio ciclo lleva bajo su cabecera las abreviaturas de ese ciclo.
function filas(e) {
  const objetivos = e.objetivos || []
  if (objetivos.length === 0) return e.frentes.map(filaDeTarea)
  const salida = []
  for (const o of objetivos) {
    salida.push(filaDeObjetivo(o.id, '🎯 ' + o.nombre + (o.foco ? ' · foco' : ''), o.simbolo + ' ' + o.resumen))
    const ciclo = (e.ciclos || []).find((c) => c.nombre === o.ciclo)
    if (ciclo) salida.push({ ...filaDeObjetivo(o.id, '', ''), id: 'ciclo:' + o.id, abreviaturas: ciclo.abreviaturas, objetivo: false })
    salida.push(...e.frentes.filter((f) => f.objetivo === o.id).map(filaDeTarea))
  }
  const sueltas = e.frentes.filter((f) => !f.objetivo)
  if (sueltas.length) salida.push(filaDeObjetivo('', '🎯 sin objetivo', ''), ...sueltas.map(filaDeTarea))
  return salida
}

const etapas = (n, uno, varios) => n + (n === 1 ? ' etapa ' + uno : ' etapas ' + varios)
const total = (n, lista) => (typeof n === 'number' ? n : (lista || []).length)

function cuantas(e) {
  const n = e.frentes.length
  if (e.ocultos > 0) return n + ' de ' + (n + e.ocultos) + ' tareas'
  return n + (n === 1 ? ' tarea' : ' tareas')
}

function resumen(e, paraTi) {
  const esperan = total(e.esperan, e.esperas)
  const fallan = total(e.fallan, e.fallas)
  const limpio = !esperan && !fallan && e.frentes.length > 0
  return [
    { texto: cuantas(e), tono: 'primario' },
    ...(esperan ? [{ texto: '⏳ ' + etapas(esperan, 'espera', 'esperan'), tono: 'espera' }] : []),
    ...(fallan ? [{ texto: '❌ ' + etapas(fallan, 'falla', 'fallan'), tono: 'falla' }] : []),
    ...(limpio ? [{ texto: '✓ nada espera ni falla', tono: 'listo' }] : []),
    ...(paraTi ? [{ texto: '▶ ' + paraTi + ' para ti', tono: 'espera' }] : []),
  ]
}

// vista arma lo que se dibuja; sin datos o sin estado, el motivo. Lo que no conoce del JSON lo ignora.
function vista(datos, falla) {
  if (!datos) return { motivo: falla || 'sin datos de qdd todavía', version: '', proyecto: '', pedidos: [], fichas: [] }
  const acciones = datos.acciones || []
  const delPedido = acciones.filter((a) => a.tipo === 'pedido')
  const comun = {
    version: datos.version || '',
    proyecto: nombre(datos.carpeta),
    pedidos: delPedido.map((a) => tarjeta(a, datos.carpeta || '')),
    fichas: acciones.filter((a) => a.tipo === 'aprobar').map((a) => ({ id: a.id, etiqueta: a.etiqueta })),
  }
  const e = datos.estado
  if (!e) return { ...comun, motivo: datos.motivo || 'el proyecto no tiene estado por tareas' }
  const cuando = e.hace || hora(e.actualizado)
  return {
    ...comun,
    titulo: 'Tareas' + (cuando ? ' · ' + cuando : ''),
    aviso: e.viejo ? '⚠ Sin actualizar ' + e.hace + ': lo que dice ⏳ puede haber cambiado' : '',
    resumen: resumen(e, delPedido.filter(ejecutable).length),
    cabecera: e.abreviaturas || [],
    filas: filas(e),
    fallas: (e.fallas || []).map((p) => ({ titulo: p.frente + ' › ' + p.etapa, texto: p.texto })),
  }
}

// motivoDeSalida explica por qué qdd no entregó datos.
function motivoDeSalida(codigo, stderr) {
  if (codigo === 0) return ''
  return (stderr || '').trim() || 'esta versión de qdd no trae «qdd consola panel»: actualiza qdd'
}

// carpetaDe: el proyecto es la carpeta del workspace con .qdd; la elegida manda mientras siga abierta, y si no, la primera.
function carpetaDe(carpetas, elegida, existe) {
  const opciones = carpetas.filter((c) => existe(path.join(c, '.qdd')))
  if (opciones.includes(elegida)) return { carpeta: elegida, opciones }
  return { carpeta: opciones[0] || '', opciones }
}

module.exports = { vista, carpetaDe, motivoDeSalida }
