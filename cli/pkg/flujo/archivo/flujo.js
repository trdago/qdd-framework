(function () {
  'use strict';
  // Rule: Prohibido usar 'else'
  // Flujo para conversar (RFC 0025): dibuja la foto por niveles N1 · N2 · N3 y muestra cada nodo al tocarlo. Con una
  // invitación, además propone cambios por el relay. Todo el texto entra con textContent: nada de la foto ni del relay
  // se interpreta como HTML.

  var SVG = 'http://www.w3.org/2000/svg';
  var ANCHO = 216, ALTO = 64, SEP_X = 36, SEP_Y = 62, MARGEN = 28, POR_LINEA = 22;

  var foto, tipos, base, hayGrupos, grupos;

  // usarFoto deja lista la foto: la del archivo al abrirlo y, con una invitación, la nueva que llega del relay.
  function usarFoto(f) {
    foto = f;
    tipos = {};
    base = {};
    grupos = {};
    foto.plantilla.tipos.forEach(function (t) { tipos[t.id] = t; });
    foto.nodos.forEach(function (n) { base[n.id] = n; });
    hayGrupos = foto.nodos.some(function (n) { return n.grupo; });
    (foto.grupos || []).forEach(function (g) { grupos[g.nombre] = g; });
  }
  usarFoto(JSON.parse(document.getElementById('foto').textContent));

  function etiquetaDeGrupo(nombre) {
    var g = grupos[nombre];
    if (!g || !g.clase) return nombre;
    return '[' + g.clase + '] ' + nombre;
  }
  var estado = { nivel: hayGrupos || foto.nodos.length >= 15 ? 1 : 3, elegido: '' };

  // Íconos de la marca: trazos de 1,5 en una grilla de 16, del color del texto que los rodea. Uno por tipo de nodo de
  // cada plantilla, y «[ ]» para un grupo.
  var ICONOS = {
    'paso': 'M2 4h12v8H2z',
    'resultado': 'M8 2a6 6 0 1 0 0.01 0z',
    'pagina': 'M3 1.5h6.5L13 5v9.5H3z M9.5 1.5V5H13',
    'seccion': 'M2 2h12v12H2z M2 6h12 M2 10h12',
    'componente': 'M2 4h12v8H2z M5 8h6',
    'enlace': 'M4 12L12 4 M6.5 4H12v5.5',
    'fuente': 'M3 4c0-3 10-3 10 0v8c0 3-10 3-10 0z M3 4c0 3 10 3 10 0',
    'transformacion': 'M13 8a5 5 0 1 1-1.5-3.5 M12 1.5v3h-3',
    'regla': 'M8 2l6 6-6 6-6-6z',
    'destino': 'M2 9h3.5l1 2h3l1-2H14v5H2z M8 2v6 M5.5 5.5L8 8l2.5-2.5',
    'agente': 'M3 5h10v8H3z M6 9h.01 M10 9h.01 M8 5V2.5',
    'herramienta': 'M10.5 2a3.5 3.5 0 0 0-3.2 5L2.5 11.8l1.7 1.7L9 8.7a3.5 3.5 0 0 0 5-3.2l-2 1-2-2z',
    'memoria': 'M3 3h10v10H3z M6 6h4v4H6z M6 1v2 M10 1v2 M6 13v2 M10 13v2',
    'persona': 'M8 2.5a2.5 2.5 0 1 0 0.01 0z M3 14c0-3.5 10-3.5 10 0',
    grupo: 'M5.5 2H2v12h3.5 M10.5 2H14v12h-3.5',
    vuelta: 'M5 3L2 6l3 3 M2 6h8a4 4 0 0 1 0 8H7',
    niveles: 'M2 3h12 M4 8h8 M6 13h4'
  };

  function icono(clave, x, y, tam) {
    var g = svg('svg', { 'class': 'icono', viewBox: '0 0 16 16', width: tam, height: tam, x: x, y: y, 'aria-hidden': 'true' });
    g.appendChild(svg('path', { d: ICONOS[clave] || ICONOS.paso }));
    return g;
  }

  function $(id) { return document.getElementById(id); }

  function el(nombre, clase, texto) {
    var e = document.createElement(nombre);
    if (clase) e.className = clase;
    if (texto !== undefined) e.textContent = texto;
    return e;
  }

  function svg(nombre, atributos) {
    var e = document.createElementNS(SVG, nombre);
    Object.keys(atributos || {}).forEach(function (k) { e.setAttribute(k, atributos[k]); });
    return e;
  }

  function guardado(clave) {
    try { return window.localStorage.getItem(clave); } catch (e) { return null; }
  }

  function guardar(clave, valor) {
    try { window.localStorage.setItem(clave, valor); } catch (e) { /* sin almacenamiento: no pasa nada */ }
  }

  // ---- cabecera, niveles y leyenda ----

  function cabecera() {
    $('nombre').textContent = foto.nombre;
    $('descripcion').textContent = foto.descripcion || '';
    var marca = 'r' + foto.revision;
    if (foto.version) marca = 'v' + foto.version.n + ' «' + foto.version.nombre + '» · ' + marca;
    $('version').textContent = marca;
    pintarModo();
    var cuando = new Date(foto.generado).toLocaleString('es-CL', { dateStyle: 'long', timeStyle: 'short' });
    $('pie').textContent = 'Generado con QDD el ' + cuando + '.';
  }

  function botonesDeNivel() {
    var nav = $('niveles');
    nav.textContent = '';
    foto.plantilla.niveles.forEach(function (nombre, i) {
      var b = el('button', 'nivel');
      b.type = 'button';
      b.setAttribute('aria-pressed', String(estado.nivel === i + 1));
      b.appendChild(el('b', '', 'N' + (i + 1)));
      b.appendChild(document.createTextNode(nombre));
      b.appendChild(el('small', '', String(foto.vistas[i].nodos.length)));
      b.title = 'Tecla ' + (i + 1);
      b.addEventListener('click', function () { cambiarNivel(i + 1); });
      nav.appendChild(b);
    });
  }

  function leyenda() {
    var ul = $('leyenda');
    ul.textContent = '';
    function entrada(clave, titulo, explica) {
      var li = el('li');
      li.appendChild(icono(clave, 0, 0, 18));
      li.appendChild(el('strong', '', titulo));
      li.appendChild(el('small', '', explica));
      ul.appendChild(li);
    }
    foto.plantilla.tipos.forEach(function (t) { entrada(t.id, t.nombre, t.explica); });
    var niveles = foto.plantilla.niveles;
    entrada('niveles', 'N1 · N2 · N3', 'N1 muestra ' + niveles[0].toLowerCase() + ', N2 suma ' + niveles[1].toLowerCase() +
      ' y N3 suma ' + niveles[2].toLowerCase() + '.');
    var usadas = {};
    foto.nodos.forEach(function (n) { if (n.grupo) usadas[(grupos[n.grupo] || {}).clase || 'grupo'] = true; });
    if (Object.keys(usadas).length) {
      entrada('grupo', Object.keys(usadas).join(', '), 'Un marco junta los pasos de una misma ' + Object.keys(usadas).join(' o ') +
        '. En N1 es una caja: tócala para abrirla.');
    }
    entrada('vuelta', 'línea punteada', 'El camino vuelve a un punto anterior.');
  }

  // ---- disposición: capas por el camino más largo, sin contar las vueltas atrás ----

  function disponer(vista) {
    var porID = {}, orden = {};
    vista.nodos.forEach(function (n, i) { porID[n.id] = n; orden[n.id] = i; });
    var vuelta = {}, visitado = {}, enPila = {};
    function recorrer(id) {
      visitado[id] = true;
      enPila[id] = true;
      (porID[id].siguen || []).forEach(function (s) {
        if (!porID[s]) return;
        if (enPila[s]) { vuelta[id + '>' + s] = true; return; }
        if (!visitado[s]) recorrer(s);
      });
      enPila[id] = false;
    }
    vista.nodos.forEach(function (n) { if (!visitado[n.id]) recorrer(n.id); });

    var entran = {}, capa = {};
    vista.nodos.forEach(function (n) { entran[n.id] = 0; capa[n.id] = 0; });
    vista.nodos.forEach(function (n) {
      (n.siguen || []).forEach(function (s) { if (porID[s] && !vuelta[n.id + '>' + s]) entran[s]++; });
    });
    var cola = vista.nodos.filter(function (n) { return entran[n.id] === 0; }).map(function (n) { return n.id; });
    while (cola.length) {
      var id = cola.shift();
      (porID[id].siguen || []).forEach(function (s) {
        if (!porID[s] || vuelta[id + '>' + s]) return;
        capa[s] = Math.max(capa[s], capa[id] + 1);
        entran[s]--;
        if (entran[s] === 0) cola.push(s);
      });
    }

    var capas = [];
    vista.nodos.forEach(function (n) {
      (capas[capa[n.id]] = capas[capa[n.id]] || []).push(n.id);
    });
    var columna = {};
    capas.forEach(function (ids, c) {
      if (c > 0) {
        ids.sort(function (a, b) { return centro(a) - centro(b) || ordenDeGrupo(a) - ordenDeGrupo(b) || orden[a] - orden[b]; });
      }
      ids.forEach(function (id, i) { columna[id] = i - (ids.length - 1) / 2; });
    });
    // los de un mismo grupo quedan juntos en su fila, para que su marco no encierre a otros.
    function ordenDeGrupo(id) {
      var g = base[id] && base[id].grupo;
      if (!g) return 0;
      return foto.nodos.findIndex(function (n) { return n.grupo === g; }) + 1;
    }
    function centro(id) {
      var padres = vista.nodos.filter(function (p) {
        return (p.siguen || []).indexOf(id) >= 0 && columna[p.id] !== undefined && !vuelta[p.id + '>' + id];
      });
      if (!padres.length) return 0;
      return padres.reduce(function (s, p) { return s + columna[p.id]; }, 0) / padres.length;
    }
    // las vueltas atrás suben por un carril a la izquierda, uno por vuelta, sin cruzar nodos.
    var claves = Object.keys(vuelta);
    claves.forEach(function (k, i) { vuelta[k] = i + 1; });
    var carril = claves.length ? 14 + 10 * claves.length : 0;
    var anchoMax = Math.max.apply(null, capas.map(function (c) { return c ? c.length : 0; }));
    var ancho = carril + 2 * MARGEN + anchoMax * ANCHO + (anchoMax - 1) * SEP_X;
    var pos = {};
    vista.nodos.forEach(function (n) {
      pos[n.id] = {
        x: carril + MARGEN + (columna[n.id] + (anchoMax - 1) / 2) * (ANCHO + SEP_X),
        y: MARGEN + 16 + capa[n.id] * (ALTO + SEP_Y)
      };
    });
    var alto = MARGEN * 2 + 16 + capas.length * (ALTO + SEP_Y) - SEP_Y;
    return { pos: pos, capa: capa, vuelta: vuelta, ancho: ancho, alto: alto, porID: porID };
  }

  function lineas(texto, maximo) {
    var palabras = String(texto).split(/\s+/), salida = [''], i = 0;
    palabras.forEach(function (p) {
      var actual = salida[i];
      if (!actual) { salida[i] = p; return; }
      if ((actual + ' ' + p).length <= POR_LINEA) { salida[i] = actual + ' ' + p; return; }
      i++;
      salida[i] = p;
    });
    if (salida.length <= maximo) return salida;
    var cortadas = salida.slice(0, maximo);
    cortadas[maximo - 1] = cortadas[maximo - 1].slice(0, POR_LINEA - 1) + '…';
    return cortadas;
  }

  // ---- dibujo ----

  function dibujar() {
    var vista = foto.vistas[estado.nivel - 1];
    var d = disponer(vista);
    var lienzo = $('lienzo');
    lienzo.textContent = '';
    var raiz = svg('svg', { width: d.ancho, height: d.alto, viewBox: '0 0 ' + d.ancho + ' ' + d.alto, role: 'group' });
    raiz.setAttribute('aria-label', foto.nombre + ', nivel ' + estado.nivel);
    var defs = svg('defs');
    var punta = svg('marker', { id: 'punta', viewBox: '0 0 10 10', refX: 9, refY: 5, markerWidth: 7, markerHeight: 7, orient: 'auto-start-reverse' });
    punta.appendChild(svg('path', { d: 'M0,0 L10,5 L0,10 z', 'class': 'punta' }));
    defs.appendChild(punta);
    var puntaVuelta = svg('marker', { id: 'punta-vuelta', viewBox: '0 0 10 10', refX: 9, refY: 5, markerWidth: 7, markerHeight: 7, orient: 'auto-start-reverse' });
    puntaVuelta.appendChild(svg('path', { d: 'M0,0 L10,5 L0,10 z', 'class': 'punta-vuelta' }));
    defs.appendChild(puntaVuelta);
    raiz.appendChild(defs);
    if (estado.nivel > 1) marcos(raiz, vista, d);
    vista.nodos.forEach(function (n) {
      (n.siguen || []).forEach(function (s) {
        if (!d.pos[s]) return;
        raiz.appendChild(arista(d.pos[n.id], d.pos[s], d.vuelta[n.id + '>' + s] || 0));
      });
    });
    vista.nodos.forEach(function (n) { raiz.appendChild(nodo(n, d.pos[n.id])); });
    lienzo.appendChild(raiz);
  }

  // marcos: en N2 y N3, cada grupo es un marco con su clase que sigue a sus nodos fila por fila, así no encierra a otros.
  function marcos(raiz, vista, d) {
    var filas = {}, margen = 12;
    vista.nodos.forEach(function (n) {
      var propio = base[n.id];
      if (!propio || !propio.grupo) return;
      var p = d.pos[n.id], porCapa = filas[propio.grupo] = filas[propio.grupo] || {};
      var f = porCapa[d.capa[n.id]];
      if (!f) { porCapa[d.capa[n.id]] = { x1: p.x, x2: p.x + ANCHO, y1: p.y, y2: p.y + ALTO }; return; }
      f.x1 = Math.min(f.x1, p.x);
      f.x2 = Math.max(f.x2, p.x + ANCHO);
    });
    Object.keys(filas).forEach(function (nombre) {
      var rs = Object.keys(filas[nombre]).map(Number).sort(function (a, b) { return a - b; }).map(function (c) {
        var f = filas[nombre][c];
        return { x1: f.x1 - margen, x2: f.x2 + margen, y1: f.y1 - margen, y2: f.y2 + margen };
      });
      rs[0].y1 -= 16;
      var puntos = [[rs[0].x1, rs[0].y1], [rs[0].x2, rs[0].y1]];
      rs.forEach(function (r, i) {
        if (i === rs.length - 1) { puntos.push([r.x2, r.y2]); return; }
        var medio = (r.y2 + rs[i + 1].y1) / 2;
        puntos.push([r.x2, medio], [rs[i + 1].x2, medio]);
      });
      for (var i = rs.length - 1; i >= 0; i--) {
        if (i === rs.length - 1) puntos.push([rs[i].x1, rs[i].y2]);
        if (i === 0) continue;
        var medio = (rs[i - 1].y2 + rs[i].y1) / 2;
        puntos.push([rs[i].x1, medio], [rs[i - 1].x1, medio]);
      }
      var c = { x1: rs[0].x1, y1: rs[0].y1 };
      var g = svg('g', { 'class': 'marco' });
      g.appendChild(svg('path', { d: 'M' + puntos.map(function (p) { return p[0] + ',' + p[1]; }).join(' L') + ' Z' }));
      var t = svg('text', { x: c.x1 + 10, y: c.y1 + 13 });
      t.textContent = etiquetaDeGrupo(nombre);
      g.appendChild(t);
      var titulo = svg('title');
      titulo.textContent = etiquetaDeGrupo(nombre) + ((grupos[nombre] || {}).descripcion ? ' — ' + grupos[nombre].descripcion : '');
      g.appendChild(titulo);
      raiz.appendChild(g);
    });
  }

  // arista: hacia abajo, una curva; una vuelta atrás (k > 0) sale por abajo, sube por su carril y entra por arriba.
  function arista(a, b, k) {
    if (k) {
      var x = 4 + 10 * k, sale = a.x + 30 + 6 * k, entra = b.x + ANCHO - 30 - 6 * k;
      var abajo = a.y + ALTO + 20 + 4 * k, arriba = b.y - 40 - 4 * k;
      var p = 'M' + sale + ',' + (a.y + ALTO) + ' L' + sale + ',' + abajo + ' L' + x + ',' + abajo + ' L' + x + ',' + arriba +
        ' L' + entra + ',' + arriba + ' L' + entra + ',' + (b.y - 2);
      var camino = svg('path', { d: p, 'class': 'arista vuelve', 'marker-end': 'url(#punta-vuelta)' });
      var titulo = svg('title');
      titulo.textContent = 'Vuelve atrás';
      camino.appendChild(titulo);
      return camino;
    }
    var x1 = a.x + ANCHO / 2, y1 = a.y + ALTO, x2 = b.x + ANCHO / 2, y2 = b.y, medio = (y1 + y2) / 2;
    var camino = 'M' + x1 + ',' + y1 + ' C' + x1 + ',' + medio + ' ' + x2 + ',' + medio + ' ' + x2 + ',' + (y2 - 2);
    return svg('path', { d: camino, 'class': 'arista', 'marker-end': 'url(#punta)' });
  }

  function nodo(n, p) {
    var propio = base[n.id];
    var tipo = propio ? propio.tipo : '';
    var clases = ['nodo'];
    if (n.grupo) clases.push('grupo');
    if (tipo === 'resultado') clases.push('resultado');
    if (propio && propio.nivel === 3) clases.push('desvio');
    if (estado.elegido === n.id) clases.push('elegido');
    var g = svg('g', { 'class': clases.join(' '), transform: 'translate(' + p.x + ',' + p.y + ')', tabindex: 0, role: 'button' });
    g.setAttribute('aria-label', n.nombre);
    var radio = tipo === 'resultado' ? ALTO / 2 : 2; // la firma de la marca: esquinas de 2 px
    g.appendChild(svg('rect', { 'class': 'forma', width: ANCHO, height: ALTO, rx: radio, ry: radio }));
    g.appendChild(icono(n.grupo ? 'grupo' : tipo, 12, ALTO / 2 - 8, 16));
    var nombre = n.grupo ? etiquetaDeGrupo(n.nombre) : n.nombre;
    var renglones = lineas(nombre, 2);
    var total = renglones.length + (n.resumen ? 1 : 0);
    var y0 = ALTO / 2 + 4 - (total - 1) * 7.5;
    renglones.forEach(function (r, i) {
      var t = svg('text', { x: 36, y: y0 + i * 15 });
      t.textContent = r;
      g.appendChild(t);
    });
    if (n.resumen) {
      var s = svg('text', { x: 36, y: y0 + renglones.length * 15, 'class': 'resumen' });
      s.textContent = lineas(n.resumen, 1)[0];
      g.appendChild(s);
    }
    var titulo = svg('title');
    titulo.textContent = nombre + (n.resumen ? ' — ' + n.resumen : '');
    g.appendChild(titulo);
    g.addEventListener('click', function () { tocar(n); });
    g.addEventListener('keydown', function (e) {
      if (e.key !== 'Enter' && e.key !== ' ') return;
      e.preventDefault();
      tocar(n);
    });
    return g;
  }

  // ---- interacción ----

  function tocar(n) {
    if (n.grupo) { abrirGrupo(n); return; }
    elegir(n.id);
  }

  function abrirGrupo(n) {
    var primero = foto.nodos.filter(function (b) { return b.grupo === n.nombre && b.nivel <= 2; })[0];
    cambiarNivel(2);
    if (primero) elegir(primero.id);
  }

  // enlace: #n2 o #n2/<nodo> abre ese nivel y ese paso, para decir «mira este paso».
  function enlace() {
    var m = /^#n([123])(?:\/(.+))?$/.exec(window.location.hash || '');
    if (!m) return false;
    estado.nivel = Number(m[1]);
    var id = m[2] ? decodeURIComponent(m[2]) : '';
    if (base[id]) estado.elegido = id;
    return true;
  }

  function anotarEnlace() {
    var hash = '#n' + estado.nivel + (estado.elegido ? '/' + encodeURIComponent(estado.elegido) : '');
    try { window.history.replaceState(null, '', hash); } catch (e) { /* un archivo local puede no dejar: no pasa nada */ }
  }

  function cambiarNivel(nivel) {
    estado.nivel = nivel;
    var visible = foto.vistas[nivel - 1].nodos.some(function (n) { return n.id === estado.elegido; });
    if (!visible) estado.elegido = '';
    botonesDeNivel();
    dibujar();
    mostrarDetalle();
    anotarEnlace();
  }

  function elegir(id) {
    if (id !== estado.elegido) propio.formulario = propio.aviso = '';
    estado.elegido = id;
    var visible = foto.vistas[estado.nivel - 1].nodos.some(function (n) { return n.id === id; });
    if (!visible) estado.nivel = Math.max(estado.nivel, base[id] ? base[id].nivel : 3);
    botonesDeNivel();
    dibujar();
    mostrarDetalle();
    anotarEnlace();
    var g = document.querySelector('.nodo.elegido');
    var quieto = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    if (g && g.scrollIntoView) g.scrollIntoView({ block: 'nearest', inline: 'nearest', behavior: quieto ? 'auto' : 'smooth' });
  }

  function fichas(titulo, ids) {
    var caja = document.createDocumentFragment();
    if (!ids.length) return caja;
    caja.appendChild(el('h3', '', titulo));
    var ul = el('ul', 'fichas');
    ids.forEach(function (id) {
      if (!base[id]) return;
      var li = el('li');
      var b = el('button', 'ficha', base[id].nombre);
      b.type = 'button';
      b.addEventListener('click', function () { elegir(id); });
      li.appendChild(b);
      ul.appendChild(li);
    });
    caja.appendChild(ul);
    return caja;
  }

  function mostrarDetalle() {
    var panel = $('detalle');
    var n = base[estado.elegido];
    document.querySelector('.cuerpo').classList.toggle('sin-detalle', !n);
    panel.hidden = !n;
    panel.textContent = '';
    if (!n) return;
    var cerrar = el('button', 'cerrar', '×');
    cerrar.type = 'button';
    cerrar.setAttribute('aria-label', 'Cerrar');
    cerrar.addEventListener('click', function () { estado.elegido = ''; dibujar(); mostrarDetalle(); anotarEnlace(); });
    panel.appendChild(cerrar);
    var t = tipos[n.tipo] || { nombre: n.tipo, explica: '' };
    var tipo = el('p', 'tipo');
    var dibujo = svg('svg', { 'class': 'icono', viewBox: '0 0 16 16', 'aria-hidden': 'true' });
    dibujo.appendChild(svg('path', { d: ICONOS[n.tipo] || ICONOS.paso }));
    tipo.appendChild(dibujo);
    tipo.appendChild(document.createTextNode(t.nombre + ' · N' + n.nivel + ' ' + foto.plantilla.niveles[n.nivel - 1]));
    panel.appendChild(tipo);
    panel.appendChild(el('h2', '', n.nombre));
    if (n.grupo) {
      var g = grupos[n.grupo] || {};
      panel.appendChild(el('p', 'en-grupo', 'En ' + etiquetaDeGrupo(n.grupo) + (g.descripcion ? ': ' + g.descripcion : '')));
    }
    if (n.descripcion) panel.appendChild(el('p', 'texto', n.descripcion));
    if (!n.descripcion) panel.appendChild(el('p', 'texto vacio', t.explica || 'Sin descripción todavía.'));
    panel.appendChild(notasDe(n));
    panel.appendChild(historiasDe(historiasDelNodo(n.id), 'Historias de usuario'));
    var vienen = foto.nodos.filter(function (o) { return (o.siguen || []).indexOf(n.id) >= 0; }).map(function (o) { return o.id; });
    panel.appendChild(fichas('Viene de', vienen));
    panel.appendChild(fichas('Sigue', n.siguen || []));
    if (invitacion) panel.appendChild(accionesDeNodo(n));
  }

  // Notas técnicas que el equipo compartió: las internas nunca vienen en la foto.
  function notasDe(n) {
    var caja = document.createDocumentFragment();
    if (!(n.notas || []).length) return caja;
    caja.appendChild(el('h3', '', 'Notas técnicas'));
    var ul = el('ul', 'notas-tecnicas');
    n.notas.forEach(function (nota) { ul.appendChild(el('li', '', nota.texto)); });
    caja.appendChild(ul);
    return caja;
  }

  function historiasDelNodo(id) {
    return (foto.historias || []).filter(function (h) { return (h.nodos || []).indexOf(id) >= 0; });
  }

  function fraseDe(h) {
    var partes = ['quiero ' + h.quiero];
    if (h.como) partes.unshift('como ' + h.como);
    if (h.para) partes.push('para ' + h.para);
    var frase = partes.join(', ');
    return frase.charAt(0).toUpperCase() + frase.slice(1);
  }

  function historiasDe(lista, titulo) {
    var caja = document.createDocumentFragment();
    if (!lista.length) return caja;
    caja.appendChild(el('h3', '', titulo));
    var ul = el('ul', 'historias');
    lista.forEach(function (h) {
      var li = el('li', 'historia');
      li.appendChild(el('span', 'frase', fraseDe(h)));
      if ((h.criterios || []).length) {
        var criterios = el('ul', 'criterios');
        h.criterios.forEach(function (c) { criterios.appendChild(el('li', '', c)); });
        li.appendChild(criterios);
      }
      ul.appendChild(li);
    });
    caja.appendChild(ul);
    return caja;
  }

  // Las historias sin nodo no tienen dónde verse en el detalle: van en su propia sección, con las demás.
  function pintarHistorias() {
    var lista = foto.historias || [];
    $('historias').hidden = !lista.length;
    var cuerpo = $('lista-historias');
    cuerpo.textContent = '';
    lista.forEach(function (h) {
      var li = el('li', 'historia');
      li.appendChild(el('span', 'frase', fraseDe(h)));
      (h.criterios || []).forEach(function (c) { li.appendChild(el('small', '', '· ' + c)); });
      if ((h.nodos || []).length) li.appendChild(fichas('En', h.nodos));
      cuerpo.appendChild(li);
    });
  }

  // ---- proponer (RFC 0025, fase 3) ----
  // Con una invitación, las acciones van en el mismo detalle del nodo. Lo propuesto le llega al equipo y entra al flujo
  // solo cuando lo aprueban. El archivo habla solo con el relay (la CSP no deja otro destino) y, sin conexión, guarda lo
  // propuesto en el navegador y lo envía al volver.

  var invitacion = conToken(JSON.parse($('enlace').textContent));

  // conToken: el archivo trae su token; la página de un enlace (fase 3b) no, porque el token viaja en el fragmento
  // (#t=…), que el navegador no le manda al Worker. Se guarda en este navegador para volver a abrir el enlace sin él, y se
  // saca de la barra para que no quede a la vista ni en una captura. Sin token, la página es de solo lectura.
  function conToken(inv) {
    if (!inv || inv.token) return inv;
    var clave = 'qdd-flujo-token-' + inv.invitacion;
    var m = /(?:^#|&)t=([^&]+)/.exec(window.location.hash || '');
    if (m) {
      guardar(clave, decodeURIComponent(m[1]));
      try { window.history.replaceState(null, '', window.location.pathname + window.location.search); } catch (e) { /* sin historial, queda en la barra */ }
    }
    inv.token = (m && decodeURIComponent(m[1])) || guardado(clave) || '';
    return inv.token ? inv : null;
  }
  var SONDEO_MS = 15000;
  var propio = { lista: [], vencida: false, sinRed: false, formulario: '', aviso: '', novedad: '' };

  // Cada acción: id, nombre del botón y campos [nombre, etiqueta, control, opcional].
  var ACCIONES = [
    { id: 'comentar', nombre: 'Comentar', campos: [['texto', 'Tu comentario o pregunta', 'area']] },
    { id: 'agregar', nombre: 'Agregar después', campos: [['nombre', 'Nombre del paso nuevo', 'linea'], ['descripcion', 'Qué pasa en él', 'area', true], ['tipo', 'Tipo', 'tipo']] },
    { id: 'renombrar', nombre: 'Renombrar', campos: [['nombre', 'Nombre nuevo', 'linea']] },
    { id: 'describir', nombre: 'Describir', campos: [['descripcion', 'Qué pasa en este paso', 'area']] },
    { id: 'conectar', nombre: 'Conectar con', campos: [['con', 'Con qué paso', 'nodo']] },
    { id: 'mover', nombre: 'Mover después de', campos: [['despues_de', 'Después de qué paso', 'nodo']] },
    { id: 'quitar', nombre: 'Quitar', campos: [] },
    { id: 'historia', nombre: 'Historia de usuario', campos: [['como', 'Como (quién lo necesita)', 'linea', true], ['quiero', 'Quiero (qué quiere hacer)', 'area'], ['para', 'Para (para qué le sirve)', 'linea', true], ['criterios', 'Criterios de aceptación, uno por línea', 'area', true]] },
    { id: 'nota', nombre: 'Nota técnica', campos: [['texto', 'Nota técnica o idea de implementación', 'area'], ['compartida', 'Pedir que el cliente también la vea', 'si', true]], experto: true }
  ];
  var MAX_CRITERIOS = 10;

  var ESTADOS = {
    'por-enviar': ['…', 'Por enviar'],
    pendiente: ['●', 'Esperando al equipo'],
    aprobada: ['✓', 'Integrada'],
    rechazada: ['✕', 'No se integró']
  };

  function claveLocal(que) { return 'qdd-flujo-' + que + '-' + invitacion.invitacion; }

  function leerLocal(que, defecto) {
    try { return JSON.parse(guardado(claveLocal(que))) || defecto; } catch (e) { return defecto; }
  }

  function guardarLocal(que, valor) { guardar(claveLocal(que), JSON.stringify(valor)); }

  function autor() { return guardado('qdd-flujo-autor') || ''; }

  function nombreDe(id) { return '«' + (base[id] ? base[id].nombre : id) + '»'; }

  function operacionDe(accion, n, v) {
    var o = { op: accion, flujo: foto.flujo, nodo: n.id };
    if (accion === 'renombrar') return { op: 'modificar', flujo: foto.flujo, nodo: n.id, nombre: v.nombre };
    if (accion === 'describir') return { op: 'modificar', flujo: foto.flujo, nodo: n.id, descripcion: v.descripcion };
    if (accion === 'agregar') { o.nombre = v.nombre; o.tipo = v.tipo; if (v.descripcion) o.descripcion = v.descripcion; }
    if (accion === 'comentar' || accion === 'nota') o.texto = v.texto;
    if (accion === 'nota' && v.compartida) o.compartida = true;
    if (accion === 'historia') return historiaDe(o, v);
    if (accion === 'conectar') o.con = v.con;
    if (accion === 'mover') o.despues_de = v.despues_de;
    return o;
  }

  function historiaDe(o, v) {
    o.quiero = v.quiero;
    if (v.como) o.como = v.como;
    if (v.para) o.para = v.para;
    var criterios = criteriosDe(v.criterios);
    if (criterios.length) o.criterios = criterios;
    return o;
  }

  function criteriosDe(texto) {
    return String(texto || '').split('\n').map(function (c) { return c.trim(); }).filter(Boolean);
  }

  function textoDe(accion, n, v) {
    var textos = {
      comentar: 'Comentas ' + nombreDe(n.id) + ': ' + v.texto,
      agregar: 'Agregar «' + v.nombre + '» después de ' + nombreDe(n.id),
      renombrar: 'Renombrar ' + nombreDe(n.id) + ' a «' + v.nombre + '»',
      describir: 'Describir ' + nombreDe(n.id) + ': ' + v.descripcion,
      conectar: 'Flecha de ' + nombreDe(n.id) + ' a ' + nombreDe(v.con),
      mover: 'Mover ' + nombreDe(n.id) + ' después de ' + nombreDe(v.despues_de),
      quitar: 'Quitar ' + nombreDe(n.id),
      historia: 'Historia en ' + nombreDe(n.id) + ': ' + fraseDe(v).replace(/^./, function (c) { return c.toLowerCase(); }),
      nota: 'Nota técnica ' + (v.compartida ? 'para compartir ' : '') + 'en ' + nombreDe(n.id) + ': ' + v.texto
    };
    return textos[accion];
  }

  function accionesDeNodo(n) {
    var caja = el('section', 'proponer');
    caja.appendChild(el('h3', '', 'Proponer un cambio'));
    if (propio.vencida) {
      caja.appendChild(el('p', 'texto vacio', 'Esta invitación ya no está vigente: pídele una nueva a quien te la mandó.'));
      return caja;
    }
    var fila = el('div', 'acciones');
    ACCIONES.forEach(function (a) {
      if (a.experto && invitacion.rol !== 'experto') return;
      var b = el('button', 'accion', a.nombre);
      b.type = 'button';
      b.setAttribute('aria-pressed', String(propio.formulario === a.id));
      b.addEventListener('click', function () {
        propio.formulario = propio.formulario === a.id ? '' : a.id;
        propio.aviso = '';
        mostrarDetalle();
        var primero = document.querySelector('.formulario input, .formulario textarea, .formulario select, .formulario .boton');
        if (primero) primero.focus();
      });
      fila.appendChild(b);
    });
    caja.appendChild(fila);
    var elegida = ACCIONES.filter(function (a) { return a.id === propio.formulario; })[0];
    if (elegida) caja.appendChild(formulario(elegida, n));
    if (propio.aviso) caja.appendChild(el('p', 'aviso', propio.aviso));
    return caja;
  }

  function control(campo, n) {
    var tipo = campo[2];
    if (tipo === 'area') {
      var area = el('textarea');
      area.maxLength = 500;
      area.rows = 3;
      return area;
    }
    if (tipo === 'si') {
      var casilla = el('input');
      casilla.type = 'checkbox';
      return casilla;
    }
    if (tipo === 'tipo' || tipo === 'nodo') {
      var lista = el('select');
      var opciones = tipo === 'tipo' ? foto.plantilla.tipos : foto.nodos.filter(function (o) { return o.id !== n.id; });
      opciones.forEach(function (o) {
        var op = el('option', '', o.nombre);
        op.value = o.id;
        lista.appendChild(op);
      });
      return lista;
    }
    var linea = el('input');
    linea.type = 'text';
    linea.maxLength = campo[0] === 'autor' ? 80 : 120;
    return linea;
  }

  function formulario(a, n) {
    var form = el('form', 'formulario');
    var campos = {};
    var lista = a.campos.slice();
    if (!autor()) lista.unshift(['autor', 'Tu nombre (el equipo lo ve en cada propuesta)', 'linea']);
    lista.forEach(function (c) {
      var casilla = c[2] === 'si';
      var etiqueta = el('label', casilla ? 'casilla' : '', c[1] + (c[3] && !casilla ? ' (opcional)' : ''));
      campos[c[0]] = control(c, n);
      if (casilla) etiqueta.insertBefore(campos[c[0]], etiqueta.firstChild);
      if (!casilla) etiqueta.appendChild(campos[c[0]]);
      form.appendChild(etiqueta);
    });
    if (a.id === 'quitar') form.appendChild(el('p', 'texto', 'Propones quitar ' + nombreDe(n.id) + ' del flujo.'));
    var error = el('p', 'aviso error');
    error.hidden = true;
    form.appendChild(error);
    var enviar = el('button', 'boton', 'Enviar propuesta');
    enviar.type = 'submit';
    form.appendChild(enviar);
    form.appendChild(el('p', 'nota', 'Le llega al equipo; la verás integrada cuando la aprueben.'));
    form.addEventListener('submit', function (e) {
      e.preventDefault();
      var v = {};
      Object.keys(campos).forEach(function (k) { v[k] = campos[k].type === 'checkbox' ? campos[k].checked : campos[k].value.trim(); });
      if (v.autor) guardar('qdd-flujo-autor', v.autor);
      var falta = lista.filter(function (c) { return !c[3] && !v[c[0]] && !(c[0] === 'autor' && autor()); })[0];
      if (falta) {
        error.textContent = 'Falta: ' + falta[1].toLowerCase() + '.';
        error.hidden = false;
        campos[falta[0]].focus();
        return;
      }
      if (a.id === 'historia' && criteriosDe(v.criterios).length > MAX_CRITERIOS) {
        error.textContent = 'Son ' + criteriosDe(v.criterios).length + ' criterios: el tope es ' + MAX_CRITERIOS + '.';
        error.hidden = false;
        campos.criterios.focus();
        return;
      }
      proponer(operacionDe(a.id, n, v), textoDe(a.id, n, v));
      propio.formulario = '';
      propio.aviso = 'Enviada: le llega al equipo. La verás integrada cuando la aprueben.';
      mostrarDetalle();
    });
    return form;
  }

  function llamarRelay(metodo, cuerpo) {
    var cabeceras = { apikey: invitacion.clave };
    var opciones = { method: metodo, headers: cabeceras, cache: 'no-store' };
    if (metodo === 'GET') cabeceras['x-qdd-token'] = invitacion.token;
    if (cuerpo) {
      cabeceras['content-type'] = 'application/json';
      opciones.body = JSON.stringify(cuerpo);
    }
    return fetch(invitacion.relay + 'flujo-proponer', opciones).then(function (r) {
      return r.json().catch(function () { return {}; }).then(function (datos) { return { codigo: r.status, datos: datos }; });
    });
  }

  function proponer(operacion, texto) {
    var cola = leerLocal('cola', []);
    cola.push({ autor: autor(), base: 'r' + foto.revision, operacion: operacion, texto: texto });
    guardarLocal('cola', cola);
    pintarPropuestas();
    vaciarCola();
  }

  var enviando = false;

  // vaciarCola envía lo propuesto, de a uno y en orden. Un error del relay lo descarta con su motivo; sin red, espera.
  function vaciarCola() {
    var cola = leerLocal('cola', []);
    if (enviando || !cola.length || propio.vencida) return;
    enviando = true;
    var envio = cola[0];
    llamarRelay('POST', { token: invitacion.token, autor: envio.autor, base: envio.base, operacion: envio.operacion }).then(function (r) {
      enviando = false;
      propio.sinRed = r.codigo >= 500;
      if (propio.sinRed) { pintarPropuestas(); return; }
      guardarLocal('cola', leerLocal('cola', []).slice(1));
      if (r.codigo === 201 && r.datos.id) anotarEnviada(r.datos.id, envio);
      if (r.codigo !== 201) propio.novedad = motivoDeError(r.codigo, envio.texto);
      if (r.codigo === 401) propio.vencida = true;
      pintarPropuestas();
      vaciarCola();
    }, function () {
      enviando = false;
      propio.sinRed = true;
      pintarPropuestas();
    });
  }

  function anotarEnviada(id, envio) {
    var textos = leerLocal('textos', {});
    textos[id] = envio.texto;
    guardarLocal('textos', textos);
    propio.lista.push({ id: id, autor: envio.autor, estado: 'pendiente' });
  }

  function motivoDeError(codigo, texto) {
    var motivos = {
      401: 'Esta invitación ya no está vigente: pídele una nueva a quien te la mandó.',
      403: 'Tu invitación no permite esa propuesta.',
      429: 'Llegaste al tope de propuestas de esta invitación.'
    };
    return (motivos[codigo] || 'El equipo no pudo recibir la propuesta') + ' («' + texto + '»)';
  }

  // sondear trae el estado de lo propuesto y, si el equipo aprobó algo, la foto nueva del flujo.
  function sondear() {
    if (propio.vencida) return;
    llamarRelay('GET').then(function (r) {
      if (r.codigo === 401) propio.vencida = true;
      if (r.codigo !== 200) { pintarPropuestas(); mostrarDetalle(); return; }
      propio.sinRed = false;
      propio.lista = r.datos.propuestas || [];
      var nueva = r.datos.foto;
      if (nueva && nueva.plantilla && nueva.nodos && nueva.vistas && nueva.revision > foto.revision) fotoNueva(nueva);
      pintarPropuestas();
      vaciarCola();
    }, function () {
      propio.sinRed = true;
      pintarPropuestas();
    });
  }

  function fotoNueva(f) {
    usarFoto(f);
    if (!base[estado.elegido]) estado.elegido = '';
    propio.novedad = 'Llegó la versión r' + f.revision + ' del flujo, con lo que aprobó el equipo.';
    cabecera();
    botonesDeNivel();
    leyenda();
    pintarHistorias();
    dibujar();
    mostrarDetalle();
  }

  function aviso() {
    if (propio.vencida) return 'Esta invitación ya no está vigente: pídele una nueva a quien te la mandó.';
    if (propio.sinRed && leerLocal('cola', []).length) return 'Sin conexión: tus propuestas se envían solas al volver.';
    if (propio.sinRed) return 'Sin conexión: el flujo se lee igual.';
    return propio.novedad;
  }

  // pintarModo dice en la cabecera si el archivo propone; una invitación cerrada vuelve a ser de solo lectura.
  function pintarModo() {
    var modo = 'Solo lectura';
    if (invitacion) modo = 'Puedes proponer · ' + invitacion.rol;
    if (invitacion && propio.vencida) modo = 'Invitación cerrada · solo lectura';
    $('modo').textContent = modo;
  }

  function pintarPropuestas() {
    pintarModo();
    $('propuestas').hidden = false;
    var texto = aviso();
    $('propuestas-aviso').textContent = texto;
    $('propuestas-aviso').hidden = !texto;
    var ul = $('lista-propuestas');
    ul.textContent = '';
    var textos = leerLocal('textos', {});
    leerLocal('cola', []).forEach(function (e) { ul.appendChild(filaDePropuesta('por-enviar', e.texto, e.autor, '')); });
    propio.lista.slice().sort(function (a, b) { return numero(b.id) - numero(a.id); }).forEach(function (p) {
      ul.appendChild(filaDePropuesta(p.estado, textos[p.id] || descripcionDe(p.operacion) || 'Propuesta ' + p.id, p.autor, p.motivo));
    });
    if (!ul.children.length) ul.appendChild(el('li', 'vacio', 'Todavía no hay propuestas. Toca un paso y elige qué proponer.'));
  }

  // descripcionDe cuenta la operación que guardó el relay: así un navegador sin lo enviado desde él (otro equipo, otro
  // navegador) también sabe qué se propuso. Va siempre como texto, nunca como HTML.
  function descripcionDe(o) {
    if (!o || typeof o !== 'object') return '';
    var en = nombreDe(o.nodo);
    var textos = {
      comentar: 'Comentario en ' + en + ': ' + o.texto,
      agregar: 'Agregar «' + o.nombre + '» ' + (o.rama ? 'como rama de ' : 'después de ') + en,
      conectar: 'Flecha de ' + en + ' a ' + nombreDe(o.con),
      desconectar: 'Quitar la flecha de ' + en + ' a ' + nombreDe(o.con),
      mover: 'Mover ' + en + ' después de ' + nombreDe(o.despues_de),
      quitar: 'Quitar ' + en,
      nota: 'Nota técnica ' + (o.compartida ? 'para compartir ' : '') + 'en ' + en + ': ' + o.texto,
      historia: 'Historia' + (o.nodo ? ' en ' + en : '') + ': ' + (o.quiero ? fraseDe(o).replace(/^./, function (c) { return c.toLowerCase(); }) : '')
    };
    if (o.op === 'modificar' && typeof o.nombre === 'string') return 'Renombrar ' + en + ' a «' + o.nombre + '»';
    if (o.op === 'modificar' && typeof o.descripcion === 'string') return 'Describir ' + en + ': ' + o.descripcion;
    if (o.op === 'modificar') return 'Cambiar ' + en;
    return textos[o.op] || '';
  }

  function numero(id) { return Number(String(id).replace(/^PRO-/, '')) || 0; }

  function filaDePropuesta(cual, texto, quien, motivo) {
    var e = ESTADOS[cual] || ESTADOS.pendiente;
    var li = el('li', 'propuesta ' + (ESTADOS[cual] ? cual : 'pendiente'));
    li.appendChild(el('span', 'marca', e[0]));
    var cuerpo = el('div');
    cuerpo.appendChild(el('span', '', texto));
    cuerpo.appendChild(el('small', '', e[1] + (quien ? ' · ' + quien : '') + (motivo ? ' · ' + motivo : '')));
    li.appendChild(cuerpo);
    return li;
  }

  // ---- recorrido de la primera vez ----

  var pasos = [
    ['Esto es un flujo', 'Cada caja es un paso y cada círculo, un inicio o un final. Las flechas dicen qué viene después.'],
    ['Tres niveles de detalle', 'N1 muestra lo principal. N2 y N3 suman detalle. Una caja punteada junta varios pasos: tócala para abrirla.'],
    ['Toca un paso', 'Al tocarlo ves qué es, de dónde viene y qué sigue. Abajo está la leyenda con cada tipo de paso.']
  ];
  if (invitacion) pasos.push(['Propón cambios', 'Toca un paso y elige qué proponer: comentar, agregar, renombrar, conectar, quitar o una historia de usuario. Le llega al equipo y lo verás integrado cuando lo aprueben.']);
  var paso = 0;

  function mostrarPaso() {
    $('recorrido-paso').textContent = (paso + 1) + ' de ' + pasos.length;
    $('recorrido-titulo').textContent = pasos[paso][0];
    $('recorrido-texto').textContent = pasos[paso][1];
    $('recorrido-seguir').textContent = paso === pasos.length - 1 ? 'Empezar' : 'Siguiente';
  }

  function abrirRecorrido() {
    paso = 0;
    mostrarPaso();
    $('recorrido').hidden = false;
    $('recorrido-seguir').focus();
  }

  function cerrarRecorrido() {
    $('recorrido').hidden = true;
    guardar('qdd-flujo-recorrido', 'visto');
  }

  $('recorrido-seguir').addEventListener('click', function () {
    if (paso === pasos.length - 1) { cerrarRecorrido(); return; }
    paso++;
    mostrarPaso();
  });
  $('recorrido-saltar').addEventListener('click', cerrarRecorrido);
  $('ver-recorrido').addEventListener('click', abrirRecorrido);
  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape' && !$('recorrido').hidden) { cerrarRecorrido(); return; }
    if (e.target && /input|textarea|select/i.test(e.target.tagName)) return;
    if (e.key === '1' || e.key === '2' || e.key === '3') cambiarNivel(Number(e.key));
  });

  var conEnlace = enlace();
  cabecera();
  botonesDeNivel();
  leyenda();
  pintarHistorias();
  dibujar();
  mostrarDetalle();
  if (!conEnlace && !guardado('qdd-flujo-recorrido')) abrirRecorrido();
  if (invitacion) {
    pintarPropuestas();
    sondear();
    window.setInterval(sondear, SONDEO_MS);
    window.addEventListener('online', sondear);
  }
})();
