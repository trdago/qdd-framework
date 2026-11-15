// El panel de qdd en Antigravity y VS Code (RFC 0014): una vista en la barra de actividad. La extensión dibuja y qdd
// decide; solo lee mientras la vista está visible. Sin shell: el binario se ejecuta con argumentos fijos.

const vscode = require('vscode')
const { spawn, execFile } = require('node:child_process')
const fs = require('node:fs')
const { vista, carpetaDe } = require('./panel.js')
const { crearLector } = require('./lector.js')

const TOPE_MS = 3000
const SIN_PROYECTO = 'Abre una carpeta con .qdd para ver su panel'

const binario = () => process.env.QDD_BIN || 'qdd'

function carpetasAbiertas() {
  return (vscode.workspace.workspaceFolders || []).map((c) => c.uri.fsPath)
}

function htmlDe(webview, extensionUri) {
  const recurso = (archivo) => webview.asWebviewUri(vscode.Uri.joinPath(extensionUri, 'media', archivo))
  const csp = `default-src 'none'; style-src ${webview.cspSource}; script-src ${webview.cspSource};`
  return `<!doctype html>
<html lang="es">
<head>
<meta charset="utf-8">
<meta http-equiv="Content-Security-Policy" content="${csp}">
<meta name="viewport" content="width=device-width, initial-scale=1">
<link rel="stylesheet" href="${recurso('vista.css')}">
</head>
<body>
<main id="raiz"></main>
<script src="${recurso('vista.js')}"></script>
</body>
</html>`
}

function crearPanel(extensionUri) {
  let vistaWeb = null
  let lector = null
  let elegida = ''
  let ultima = vista(null, '')

  const enviar = () => {
    if (vistaWeb) vistaWeb.webview.postMessage({ tipo: 'vista', vista: ultima })
  }
  const dibujar = (nueva) => {
    ultima = nueva
    enviar()
  }
  const detener = () => {
    if (lector) lector.detener()
    lector = null
  }
  const proyecto = () => carpetaDe(carpetasAbiertas(), elegida, fs.existsSync)

  // refrescar arma el lector del proyecto; la vista oculta no lanza nada.
  const refrescar = () => {
    detener()
    const { carpeta } = proyecto()
    if (!carpeta) return dibujar(vista(null, SIN_PROYECTO))
    lector = crearLector({
      abrir: (args) => spawn(binario(), args, { cwd: carpeta, stdio: ['ignore', 'pipe', 'ignore'], windowsHide: true }),
      leer: (args, listo) => execFile(binario(), args, { cwd: carpeta, timeout: TOPE_MS, windowsHide: true, maxBuffer: 16 * 1024 * 1024 }, listo),
      alDatos: (datos) => dibujar(vista(datos, '')),
      alFallar: (motivo) => dibujar(vista(null, motivo)),
    })
    if (vistaWeb?.visible) lector.iniciar()
  }

  return {
    resolveWebviewView(view) {
      vistaWeb = view
      view.webview.options = { enableScripts: true, localResourceRoots: [vscode.Uri.joinPath(extensionUri, 'media')] }
      view.webview.html = htmlDe(view.webview, extensionUri)
      view.webview.onDidReceiveMessage((m) => {
        if (m?.tipo === 'listo') enviar()
      })
      view.onDidChangeVisibility(() => {
        if (!lector) return
        if (view.visible) return lector.iniciar()
        lector.detener()
      })
      view.onDidDispose(() => {
        detener()
        vistaWeb = null
      })
      refrescar()
    },
    refrescar,
    detener,
    async elegirCarpeta() {
      const { opciones } = proyecto()
      if (opciones.length < 2) {
        vscode.window.showInformationMessage('QDD: este workspace no tiene otra carpeta con .qdd')
        return
      }
      const eleccion = await vscode.window.showQuickPick(opciones, { placeHolder: 'Proyecto de QDD' })
      if (!eleccion) return
      elegida = eleccion
      refrescar()
    },
  }
}

let panel = null

function activate(context) {
  panel = crearPanel(context.extensionUri)
  context.subscriptions.push(
    vscode.window.registerWebviewViewProvider('qdd.panel', panel),
    vscode.commands.registerCommand('qdd.elegirCarpeta', () => panel.elegirCarpeta()),
    vscode.workspace.onDidChangeWorkspaceFolders(() => panel.refrescar()),
  )
}

function deactivate() {
  if (panel) panel.detener()
}

module.exports = { activate, deactivate }
