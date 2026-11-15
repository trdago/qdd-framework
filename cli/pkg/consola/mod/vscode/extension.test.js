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
