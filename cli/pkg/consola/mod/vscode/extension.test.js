const { test } = require('node:test')
const assert = require('node:assert/strict')
const Module = require('node:module')
const { EventEmitter } = require('node:events')
const fs = require('node:fs')
const os = require('node:os')
const path = require('node:path')

// vscode simulado: el mínimo que la extensión usa. Se carga con el mismo nombre que el real.
function simular(carpetas) {
  const s = { comandos: {}, proveedores: {}, elecciones: [], avisos: [] }
  s.vscode = {
    Uri: { file: (p) => ({ fsPath: p }), joinPath: (u, ...r) => ({ fsPath: path.join(u.fsPath, ...r) }) },
    workspace: {
      workspaceFolders: carpetas.map((c) => ({ uri: { fsPath: c } })),
      onDidChangeWorkspaceFolders: () => ({ dispose() {} }),
    },
    window: {
      registerWebviewViewProvider: (id, p) => {
        s.proveedores[id] = p
        return { dispose() {} }
      },
      showQuickPick: async (opciones) => s.elecciones.shift() ?? opciones[0],
      showInformationMessage: (m) => s.avisos.push(m),
    },
    commands: {
      registerCommand: (id, fn) => {
        s.comandos[id] = fn
        return { dispose() {} }
      },
    },
  }
  return s
}

function vistaSimulada() {
  const v = new EventEmitter()
  v.visible = true
  v.onDidChangeVisibility = (fn) => {
    v.on('visible', fn)
    return { dispose() {} }
  }
  v.onDidDispose = (fn) => {
    v.on('dispose', fn)
    return { dispose() {} }
  }
  v.enviados = []
  v.webview = {
    cspSource: 'vscode-resource:',
    options: {},
    html: '',
    asWebviewUri: (u) => 'vscode-resource:' + u.fsPath,
    postMessage: async (m) => v.enviados.push(m),
    onDidReceiveMessage: (fn) => {
      v.alMensaje = fn
      return { dispose() {} }
    },
  }
  return v
}

// cargar: la extensión con el vscode simulado y un qdd simulado (un script en QDD_BIN).
function cargar(carpetas) {
  const s = simular(carpetas)
  const original = Module._load
  Module._load = function (pedido, ...resto) {
    if (pedido === 'vscode') return s.vscode
    return original.call(this, pedido, ...resto)
  }
  delete require.cache[require.resolve('./extension.js')]
  const ext = require('./extension.js')
  Module._load = original
  s.ext = ext
  s.contexto = { subscriptions: [], extensionUri: { fsPath: __dirname } }
  ext.activate(s.contexto)
  return s
}

function proyecto(nombre = 'p') {
  const raiz = fs.mkdtempSync(path.join(os.tmpdir(), 'qdd-vscode-'))
  const dir = path.join(raiz, nombre)
  fs.mkdirSync(path.join(dir, '.qdd'), { recursive: true })
  return dir
}

// qddFalso: un binario que imprime una línea por 'seguir' y se queda vivo; anota sus argumentos y su carpeta.
function qddFalso() {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'qdd-bin-'))
  const log = path.join(dir, 'log')
  const bin = path.join(dir, 'qdd')
  const linea = JSON.stringify({ version: 'v9', carpeta: 'x', estado: null, motivo: 'sin estado', acciones: [] })
  fs.writeFileSync(bin, `#!/bin/sh\necho "$PWD $*" >> ${log}\nprintf '%s\\n' '${linea}'\nexec sleep 30\n`, { mode: 0o755 })
  return { bin, log }
}

async function hasta(cond, ms = 3000) {
  const fin = Date.now() + ms
  while (Date.now() < fin) {
    if (cond()) return
    await new Promise((r) => setTimeout(r, 10))
  }
  assert.fail('no se cumplió a tiempo: ' + cond)
}

test('el manifiesto declara la vista que la extensión registra', () => {
  const m = JSON.parse(fs.readFileSync(path.join(__dirname, 'package.json'), 'utf8'))
  assert.equal(m.publisher + '.' + m.name, 'qdd.qdd-panel')
  assert.equal(m.contributes.views.qdd[0].id, 'qdd.panel')
  assert.equal(m.contributes.views.qdd[0].type, 'webview')
  assert.equal(m.contributes.viewsContainers.activitybar[0].id, 'qdd')
  assert.ok(fs.existsSync(path.join(__dirname, m.contributes.viewsContainers.activitybar[0].icon)))
  assert.ok(fs.existsSync(path.join(__dirname, m.main)))
  const s = cargar([])
  assert.ok(s.proveedores['qdd.panel'])
})

// contrato: estado#7188db80
test('el HTML tiene CSP estricta: nada remoto, nada en línea, sin eval', () => {
  const s = cargar([])
  const v = vistaSimulada()
  s.proveedores['qdd.panel'].resolveWebviewView(v)
  const csp = /Content-Security-Policy" content="([^"]+)"/.exec(v.webview.html)[1]
  assert.match(csp, /default-src 'none'/)
  assert.match(csp, /script-src vscode-resource:;/)
  assert.match(csp, /style-src vscode-resource:;/)
  assert.doesNotMatch(csp, /unsafe|https?:/)
  assert.doesNotMatch(v.webview.html, /<script>|<style>|onclick=/, 'sin código ni estilos en línea')
  assert.ok(v.webview.html.indexOf('pasos.js') < v.webview.html.indexOf('vista.js'), 'la confirmación en dos pasos se carga antes que la vista')
  assert.equal(v.webview.options.enableScripts, true)
  assert.equal(v.webview.options.localResourceRoots[0].fsPath, path.join(__dirname, 'media'))
  v.emit('dispose')
})

test('sin una carpeta con .qdd, lo dice y no ejecuta nada', async () => {
  const { bin, log } = qddFalso()
  process.env.QDD_BIN = bin
  const s = cargar([fs.mkdtempSync(path.join(os.tmpdir(), 'sin-qdd-'))])
  const v = vistaSimulada()
  s.proveedores['qdd.panel'].resolveWebviewView(v)
  v.alMensaje({ tipo: 'listo' })
  await hasta(() => v.enviados.length > 0)
  assert.match(v.enviados[0].vista.motivo, /carpeta con \.qdd/)
  assert.equal(fs.existsSync(log), false)
  v.emit('dispose')
})

test('lee a qdd en la carpeta del proyecto y le manda la vista al webview', async () => {
  const { bin, log } = qddFalso()
  process.env.QDD_BIN = bin
  const dir = proyecto()
  const s = cargar([dir])
  const v = vistaSimulada()
  s.proveedores['qdd.panel'].resolveWebviewView(v)
  v.alMensaje({ tipo: 'listo' })
  await hasta(() => v.enviados.some((m) => m.vista?.version === 'v9'))
  assert.equal(fs.readFileSync(log, 'utf8').trim(), fs.realpathSync(dir) + ' consola seguir')
  v.emit('dispose')
})

test('un webview que vuelve a nacer recibe la última vista sin esperar otra línea', async () => {
  const { bin } = qddFalso()
  process.env.QDD_BIN = bin
  const s = cargar([proyecto()])
  const v = vistaSimulada()
  s.proveedores['qdd.panel'].resolveWebviewView(v)
  v.alMensaje({ tipo: 'listo' })
  await hasta(() => v.enviados.some((m) => m.vista?.version === 'v9'))
  v.enviados.length = 0
  v.alMensaje({ tipo: 'listo' })
  assert.equal(v.enviados[0].vista.version, 'v9')
  v.emit('dispose')
})

test('oculta no lee; visible vuelve a leer (solo se gasta si se mira)', async () => {
  const { bin, log } = qddFalso()
  process.env.QDD_BIN = bin
  const s = cargar([proyecto()])
  const v = vistaSimulada()
  v.visible = false
  s.proveedores['qdd.panel'].resolveWebviewView(v)
  await new Promise((r) => setTimeout(r, 50))
  assert.equal(fs.existsSync(log), false, 'nacida oculta no lanza nada')
  v.visible = true
  v.emit('visible')
  await hasta(() => fs.existsSync(log))
  v.visible = false
  v.emit('visible')
  v.visible = true
  v.emit('visible')
  await hasta(() => fs.readFileSync(log, 'utf8').trim().split('\n').length === 2)
  v.emit('dispose')
})

test('con varias carpetas con .qdd, el selector cambia el proyecto', async () => {
  const { bin, log } = qddFalso()
  process.env.QDD_BIN = bin
  const uno = proyecto('uno')
  const dos = proyecto('dos')
  const s = cargar([uno, dos])
  const v = vistaSimulada()
  s.proveedores['qdd.panel'].resolveWebviewView(v)
  await hasta(() => fs.existsSync(log))
  s.elecciones.push(dos)
  await s.comandos['qdd.elegirCarpeta']()
  await hasta(() => fs.readFileSync(log, 'utf8').includes(fs.realpathSync(dos)))
  v.emit('dispose')
})

test('el selector con una sola carpeta avisa y no cambia nada', async () => {
  const { bin } = qddFalso()
  process.env.QDD_BIN = bin
  const s = cargar([proyecto()])
  await s.comandos['qdd.elegirCarpeta']()
  assert.equal(s.avisos.length, 1)
})

test('deactivate deja la vista sin procesos', async () => {
  const { bin, log } = qddFalso()
  process.env.QDD_BIN = bin
  const s = cargar([proyecto()])
  const v = vistaSimulada()
  s.proveedores['qdd.panel'].resolveWebviewView(v)
  await hasta(() => fs.existsSync(log))
  s.ext.deactivate()
  const antes = fs.readFileSync(log, 'utf8')
  await new Promise((r) => setTimeout(r, 50))
  assert.equal(fs.readFileSync(log, 'utf8'), antes)
})

// Fase B (RFC 0014): «s  Sí» en un pedido y aprobar una ficha. El webview solo manda el id; la extensión ejecuta
// 'qdd consola accion <id> --confirmado' (qdd vuelve a calcular la acción y la autoriza) y devuelve el resultado.
const PEDIDO = { id: 'pedido:1', tipo: 'pedido', tecla: 'a', etiqueta: 'Publicar', detalle: 'npm publish', estado: 'pendiente', confirmar: true }
const HECHO = { ...PEDIDO, id: 'pedido:2', tecla: 'b', estado: 'hecho' }
const FICHA = { id: 'aprobar:pagos', tipo: 'aprobar', etiqueta: 'Aprobar «Pagos»', detalle: 'qdd modules approve pagos', confirmar: true }

// qddConAcciones: 'seguir' entrega el panel con esas acciones; 'accion' anota y sale con el código pedido.
function qddConAcciones(acciones, codigo = 0, demora = 0) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'qdd-bin-'))
  const log = path.join(dir, 'log')
  const bin = path.join(dir, 'qdd')
  const linea = JSON.stringify({ version: 'v9', carpeta: 'x', estado: null, motivo: 'sin estado', acciones })
  fs.writeFileSync(bin, `#!/bin/sh
echo "$*" >> ${log}
if [ "$2" = "accion" ]; then sleep ${demora}; echo "lanzado $3"; exit ${codigo}; fi
printf '%s\\n' '${linea}'
exec sleep 30
`, { mode: 0o755 })
  return { bin, log, acciones: () => fs.readFileSync(log, 'utf8').trim().split('\n').filter((l) => l.startsWith('consola accion')) }
}

async function conVista(qdd) {
  process.env.QDD_BIN = qdd.bin
  const s = cargar([proyecto()])
  const v = vistaSimulada()
  s.proveedores['qdd.panel'].resolveWebviewView(v)
  await hasta(() => v.enviados.some((m) => m.vista?.version === 'v9'))
  return v
}

const resultados = (v) => v.enviados.filter((m) => m.tipo === 'resultado')

test('«s  Sí» en un pedido lo ejecuta una vez con --confirmado y devuelve el resultado', async () => {
  const qdd = qddConAcciones([PEDIDO])
  const v = await conVista(qdd)
  v.alMensaje({ tipo: 'accion', id: PEDIDO.id })
  await hasta(() => resultados(v).some((m) => !m.ejecutando))
  assert.deepEqual(qdd.acciones(), ['consola accion pedido:1 --confirmado'])
  const r = resultados(v).at(-1)
  assert.deepEqual([r.id, r.titulo, r.ok, r.todaviaNo, r.texto], ['pedido:1', 'Publicar', true, false, 'lanzado pedido:1'])
  v.emit('dispose')
})

test('aprobar una ficha la aprueba con qdd', async () => {
  const qdd = qddConAcciones([FICHA])
  const v = await conVista(qdd)
  v.alMensaje({ tipo: 'accion', id: FICHA.id })
  await hasta(() => qdd.acciones().length === 1)
  assert.deepEqual(qdd.acciones(), ['consola accion aprobar:pagos --confirmado'])
  v.emit('dispose')
})

test('dos clics seguidos ejecutan el pedido una sola vez', async () => {
  const qdd = qddConAcciones([PEDIDO], 0, 0.2)
  const v = await conVista(qdd)
  v.alMensaje({ tipo: 'accion', id: PEDIDO.id })
  v.alMensaje({ tipo: 'accion', id: PEDIDO.id })
  await hasta(() => resultados(v).some((m) => !m.ejecutando))
  await new Promise((r) => setTimeout(r, 50))
  assert.equal(qdd.acciones().length, 1)
  v.emit('dispose')
})

test('un id que la vista no ofrece (inventado o de un pedido hecho) no ejecuta nada', async () => {
  const qdd = qddConAcciones([PEDIDO, HECHO])
  const v = await conVista(qdd)
  for (const id of [HECHO.id, 'auditar', '--help', 42]) v.alMensaje({ tipo: 'accion', id })
  await new Promise((r) => setTimeout(r, 100))
  assert.deepEqual(qdd.acciones(), [])
  assert.deepEqual(resultados(v), [])
  v.emit('dispose')
})

test('«todavía no» (exit 75) y un rechazo de qdd vuelven como resultado, sin esconderlos', async () => {
  for (const [codigo, ok, todaviaNo] of [[75, false, true], [1, false, false]]) {
    const qdd = qddConAcciones([PEDIDO], codigo)
    const v = await conVista(qdd)
    v.alMensaje({ tipo: 'accion', id: PEDIDO.id })
    await hasta(() => resultados(v).some((m) => !m.ejecutando))
    const r = resultados(v).at(-1)
    assert.deepEqual([r.ok, r.todaviaNo, r.texto], [ok, todaviaNo, 'lanzado pedido:1'], 'exit ' + codigo)
    v.emit('dispose')
  }
})

// qddConPid: como qddFalso, en una carpeta cuyo nombre rompería un shell, y anota su PID (exec lo conserva en sleep).
function qddConPid() {
  const dir = path.join(fs.mkdtempSync(path.join(os.tmpdir(), 'qdd-bin-')), 'con espacio; $(echo roto) & "comillas"')
  fs.mkdirSync(dir)
  const log = path.join(path.dirname(dir), 'log')
  const pid = path.join(path.dirname(dir), 'pid')
  const bin = path.join(dir, 'qdd')
  const linea = JSON.stringify({ version: 'v9', carpeta: 'x', estado: null, motivo: 'sin estado', acciones: [] })
  fs.writeFileSync(bin, `#!/bin/sh\nprintf '%s\\n' "$*" >> '${log}'\necho $$ > '${pid}'\nprintf '%s\\n' '${linea}'\nexec sleep 30\n`, { mode: 0o755 })
  return { bin, log, pid }
}

const vivo = (pid) => {
  try {
    process.kill(pid, 0)
    return true
  } catch {
    return false
  }
}

// contrato: estado#7188db80
test('oculta no deja ningún proceso de qdd; el binario corre sin shell y con argumentos fijos', async () => {
  const qdd = qddConPid()
  process.env.QDD_BIN = qdd.bin
  const s = cargar([proyecto()])
  const v = vistaSimulada()
  s.proveedores['qdd.panel'].resolveWebviewView(v)
  let pid = 0
  try {
    await hasta(() => fs.existsSync(qdd.pid) && fs.readFileSync(qdd.pid, 'utf8').trim() !== '')
    pid = Number(fs.readFileSync(qdd.pid, 'utf8'))
    assert.equal(fs.readFileSync(qdd.log, 'utf8'), 'consola seguir\n', 'con una ruta que un shell partiría, corre igual y con sus argumentos')
    assert.ok(vivo(pid))
    v.visible = false
    v.emit('visible')
    await hasta(() => !vivo(pid))
    await new Promise((r) => setTimeout(r, 50))
    assert.equal(fs.readFileSync(qdd.log, 'utf8'), 'consola seguir\n', 'oculta no lanza otro')
  } finally {
    v.emit('dispose')
    if (pid && vivo(pid)) process.kill(pid)
  }
})
