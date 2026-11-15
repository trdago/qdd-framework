// Dibuja en el webview lo que arma panel.js. Todo el texto entra con textContent: una etiqueta o un comando que dejó
// una IA nunca se interpreta como HTML. Solo manda mensajes con ids; no ejecuta nada. Fase B: lo que se ejecuta pasa
// por «s  Sí / n  No» (media/pasos.js) y el resultado que devuelve la extensión se muestra arriba de «Para ti».
(function () {
  const vscode = acquireVsCodeApi()
  const raiz = document.getElementById('raiz')
  const pasos = window.qddPasos
  let estado = pasos.inicial
  let actual = null
  let resultado = null

  function el(etiqueta, clase, texto) {
    const nodo = document.createElement(etiqueta)
    if (clase) nodo.className = clase
    if (texto) nodo.textContent = texto
    return nodo
  }

  function con(padre, ...hijos) {
    padre.append(...hijos.filter(Boolean))
    return padre
  }

  function titulo(texto, marca) {
    return con(el('h2', 'seccion'), marca ? el('span', 'tono-espera', marca + ' ') : null, document.createTextNode(texto))
  }

  function cabecera(v) {
    const fila = con(el('div', 'cabeza'), el('h1', 'titulo', v.titulo || 'QDD'), el('span', 'version', v.version))
    const proyecto = v.proyecto ? el('div', 'proyecto', v.proyecto) : null
    const resumen = (v.resumen || []).map((r) => el('span', 'tono-' + r.tono, r.texto))
    const aviso = v.aviso ? el('p', 'aviso tono-espera', v.aviso) : null
    return con(el('header'), fila, proyecto, resumen.length ? con(el('div', 'resumen'), ...resumen) : null, aviso)
  }

  function boton(texto, clase, alTocar) {
    const nodo = el('button', 'boton ' + clase, texto)
    nodo.type = 'button'
    nodo.addEventListener('click', alTocar)
    return nodo
  }

  function aplicar(r) {
    estado = r.estado
    if (r.enviar) vscode.postMessage(r.enviar)
    if (actual) dibujar(actual)
  }

  // acciones: el botón de ejecutar, o la pregunta de confirmación si es lo que se está confirmando.
  function acciones(a, verbo) {
    if (!a.ejecutar) return null
    if (estado.confirmando !== a.id) return con(el('div', 'acciones'), boton(verbo, 'primario', () => aplicar(pasos.tocar(estado, actual, a.id))))
    return con(
      el('div', 'acciones confirmando'),
      el('span', 'pregunta', '¿' + verbo + '?'),
      boton('s  Sí', 'primario', () => aplicar(pasos.si(estado, actual))),
      boton('n  No', 'secundario', () => aplicar(pasos.no())),
    )
  }

  function notas(lista) {
    return lista.map((n) => el('p', 'nota tono-' + n.tono, n.texto))
  }

  function pedido(p) {
    const marca = p.marca ? el('span', 'marca', p.marca) : null
    const tecla = p.tecla ? el('kbd', 'tecla', p.tecla) : null
    const cabeza = con(el('div', 'pedido-cabeza'), marca, tecla, el('span', 'etiqueta', p.etiqueta))
    cabeza.title = p.etiqueta
    const comando = p.comando ? el('code', 'comando', '$ ' + p.comando) : null
    if (comando) comando.title = p.comando
    return con(el('li', 'pedido tono-' + p.tono), cabeza, ...notas(p.notas), comando, p.meta ? el('p', 'meta', p.meta) : null, acciones(p, 'Ejecutar'))
  }

  function ficha(f) {
    const detalle = f.detalle ? el('code', 'comando', '$ ' + f.detalle) : null
    return con(el('li', 'ficha'), el('div', 'pedido-cabeza', f.etiqueta), detalle, acciones(f, 'Aprobar'))
  }

  function tonoDe(r) {
    if (r.ejecutando) return 'tenue'
    if (r.todaviaNo) return 'espera'
    if (r.ok) return 'listo'
    return 'falla'
  }

  function recuadro() {
    if (!resultado) return null
    const marca = { tenue: '⟳ ', espera: '◷ Todavía no · ', listo: '✓ ', falla: '✗ ' }[tonoDe(resultado)]
    return con(el('div', 'resultado tono-' + tonoDe(resultado)), el('strong', '', marca + resultado.titulo), el('p', 'nota', resultado.texto))
  }

  function paraTi(v) {
    if (v.pedidos.length === 0 && v.fichas.length === 0 && !resultado) return null
    return con(el('section', 'para-ti'), titulo('Para ti', '▶'), recuadro(), con(el('ul', 'lista'), ...v.pedidos.map(pedido), ...v.fichas.map(ficha)))
  }

  function celdas(lista) {
    return con(el('span', 'celdas'), ...lista.map((s) => el('span', 'celda', s)))
  }

  function fila(f) {
    if (f.abreviaturas) return con(el('div', 'fila fila-ciclo'), el('span', 'nombre'), celdas(f.abreviaturas))
    if (f.objetivo) {
      const nodo = con(el('div', 'fila-objetivo'), el('span', 'nombre', f.nombre), f.resumen ? el('span', 'meta', f.resumen) : null)
      nodo.title = f.nombre
      return nodo
    }
    const nombre = el('span', 'nombre tono-' + f.tono, f.nombre)
    const nodo = con(el('div', 'fila' + (f.running ? ' en-curso' : '')), f.running ? el('span', 'latido', '●') : null, nombre, celdas(f.simbolos))
    nodo.title = f.nombre
    return nodo
  }

  function tabla(v) {
    if (!v.filas) return null
    const titulos = con(el('div', 'fila fila-titulos'), el('span', 'nombre'), celdas(v.cabecera))
    return con(el('section', 'tabla'), titulos, ...v.filas.map(fila))
  }

  function fallas(v) {
    if (!v.fallas?.length) return null
    const lista = v.fallas.map((f) => con(el('li', 'falla'), el('strong', 'tono-falla', f.titulo), el('p', 'nota', f.texto)))
    return con(el('section'), titulo('❌ Fallas'), con(el('ul', 'lista'), ...lista))
  }

  function dibujar(v) {
    const motivo = v.motivo ? el('p', 'motivo', v.motivo) : null
    raiz.replaceChildren(...[cabecera(v), motivo, paraTi(v), tabla(v), fallas(v)].filter(Boolean))
  }

  window.addEventListener('message', (evento) => {
    const m = evento.data
    if (m?.tipo === 'resultado') resultado = m
    if (m?.tipo === 'vista') {
      actual = m.vista
      estado = pasos.alLlegar(estado, actual)
    }
    if (actual) dibujar(actual)
  })
  // Las mismas teclas que el panel de Claude Code mientras la vista tiene el foco: la del pedido pregunta, «s» confirma.
  window.addEventListener('keydown', (e) => {
    if (!actual || e.ctrlKey || e.metaKey || e.altKey) return
    aplicar(pasos.tecla(estado, actual, e.key))
  })
  vscode.postMessage({ tipo: 'listo' })
})()
