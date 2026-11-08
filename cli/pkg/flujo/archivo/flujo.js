(function () {
  'use strict';
  // Rule: Prohibido usar 'else'
  // Flujo para conversar (RFC 0025, fase 1): dibuja la foto por niveles N1 · N2 · N3 y muestra cada nodo al tocarlo.
  // Todo el texto entra con textContent: nada de la foto se interpreta como HTML.

  var SVG = 'http://www.w3.org/2000/svg';
  var ANCHO = 196, ALTO = 58, SEP_X = 36, SEP_Y = 62, MARGEN = 28, POR_LINEA = 24;

  var foto = JSON.parse(document.getElementById('foto').textContent);
  var tipos = {};
  foto.plantilla.tipos.forEach(function (t) { tipos[t.id] = t; });
  var base = {};
  foto.nodos.forEach(function (n) { base[n.id] = n; });
  var hayGrupos = foto.nodos.some(function (n) { return n.grupo; });
  var grupos = {};
  (foto.grupos || []).forEach(function (g) { grupos[g.nombre] = g; });

  function etiquetaDeGrupo(nombre) {
    var g = grupos[nombre];
    if (!g || !g.clase) return nombre;
    return '[' + g.clase + '] ' + nombre;
  }
  var estado = { nivel: hayGrupos || foto.nodos.length >= 15 ? 1 : 3, elegido: '' };

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
    $('pie').textContent = 'Generado con QDD el ' + new Date(foto.generado).toLocaleString() + '.';
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
    foto.plantilla.tipos.forEach(function (t) {
      var li = el('li');
      li.appendChild(el('span', '', t.icono));
      li.appendChild(el('strong', '', t.nombre));
      li.appendChild(el('small', '', t.explica));
      ul.appendChild(li);
    });
    var niveles = foto.plantilla.niveles;
    var li = el('li');
    li.appendChild(el('span', '', '⇕'));
    li.appendChild(el('strong', '', 'N1 · N2 · N3'));
    li.appendChild(el('small', '', 'N1 muestra ' + niveles[0].toLowerCase() + ', N2 suma ' + niveles[1].toLowerCase() +
      ' y N3 suma ' + niveles[2].toLowerCase() + '. Una caja azul junta varios pasos: tócala para abrirla.'));
    ul.appendChild(li);
    var usadas = {};
    foto.nodos.forEach(function (n) { if (n.grupo) usadas[(grupos[n.grupo] || {}).clase || 'grupo'] = true; });
    if (Object.keys(usadas).length) {
      var marco = el('li');
      marco.appendChild(el('span', '', '[ ]'));
      marco.appendChild(el('strong', '', Object.keys(usadas).join(', ')));
      marco.appendChild(el('small', '', 'Un marco junta los pasos de una misma ' + Object.keys(usadas).join(' o ') + '.'));
      ul.appendChild(marco);
    }
    var vuelve = el('li');
    vuelve.appendChild(el('span', '', '⤺'));
    vuelve.appendChild(el('strong', '', 'línea ámbar'));
    vuelve.appendChild(el('small', '', 'El camino vuelve a un punto anterior.'));
    ul.appendChild(vuelve);
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
      var abajo = a.y + ALTO + 20 + 4 * k, arriba = b.y - 26 - 4 * k;
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
    var radio = tipo === 'resultado' ? ALTO / 2 : 10;
    g.appendChild(svg('rect', { 'class': 'forma', width: ANCHO, height: ALTO, rx: radio, ry: radio }));
    var icono = n.grupo ? '▦' : (tipos[tipo] || {}).icono || '▭';
    var ic = svg('text', { x: 14, y: ALTO / 2 + 5, 'class': 'icono' });
    ic.textContent = icono;
    g.appendChild(ic);
    var nombre = n.grupo ? etiquetaDeGrupo(n.nombre) : n.nombre;
    var renglones = lineas(nombre, n.resumen ? 1 : 2);
    var y0 = ALTO / 2 + 4 - (renglones.length - 1) * 8 - (n.resumen ? 6 : 0);
    renglones.forEach(function (r, i) {
      var t = svg('text', { x: 36, y: y0 + i * 16 });
      t.textContent = r;
      g.appendChild(t);
    });
    if (n.resumen) {
      var s = svg('text', { x: 36, y: y0 + 17, 'class': 'resumen' });
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
    estado.elegido = id;
    var visible = foto.vistas[estado.nivel - 1].nodos.some(function (n) { return n.id === id; });
    if (!visible) estado.nivel = Math.max(estado.nivel, base[id] ? base[id].nivel : 3);
    botonesDeNivel();
    dibujar();
    mostrarDetalle();
    anotarEnlace();
    var g = document.querySelector('.nodo.elegido');
    if (g && g.scrollIntoView) g.scrollIntoView({ block: 'nearest', inline: 'nearest', behavior: 'smooth' });
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
    var t = tipos[n.tipo] || { icono: '▭', nombre: n.tipo, explica: '' };
    panel.appendChild(el('p', 'tipo', t.icono + ' ' + t.nombre + ' · N' + n.nivel + ' ' + foto.plantilla.niveles[n.nivel - 1]));
    panel.appendChild(el('h2', '', n.nombre));
    if (n.grupo) {
      var g = grupos[n.grupo] || {};
      panel.appendChild(el('p', 'en-grupo', 'En ' + etiquetaDeGrupo(n.grupo) + (g.descripcion ? ': ' + g.descripcion : '')));
    }
    if (n.descripcion) panel.appendChild(el('p', 'texto', n.descripcion));
    if (!n.descripcion) panel.appendChild(el('p', 'texto vacio', t.explica || 'Sin descripción todavía.'));
    var vienen = foto.nodos.filter(function (o) { return (o.siguen || []).indexOf(n.id) >= 0; }).map(function (o) { return o.id; });
    panel.appendChild(fichas('Viene de', vienen));
    panel.appendChild(fichas('Sigue', n.siguen || []));
  }

  // ---- recorrido de la primera vez ----

  var pasos = [
    ['Esto es un flujo', 'Cada caja es un paso y cada círculo, un inicio o un final. Las flechas dicen qué viene después.'],
    ['Tres niveles de detalle', 'N1 muestra lo principal. N2 y N3 suman detalle. Una caja azul junta varios pasos: tócala para abrirla.'],
    ['Toca un paso', 'Al tocarlo ves qué es, de dónde viene y qué sigue. Abajo está la leyenda con cada tipo de paso.']
  ];
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
    if (e.target && /input|textarea/i.test(e.target.tagName)) return;
    if (e.key === '1' || e.key === '2' || e.key === '3') cambiarNivel(Number(e.key));
  });

  var conEnlace = enlace();
  cabecera();
  botonesDeNivel();
  leyenda();
  dibujar();
  mostrarDetalle();
  if (!conEnlace && !guardado('qdd-flujo-recorrido')) abrirRecorrido();
})();
