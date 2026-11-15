// El panel de qdd en Antigravity y VS Code (RFC 0014): una vista en la barra de actividad. La extensión dibuja y qdd
// decide; solo lee mientras la vista está visible. Sin shell: el binario se ejecuta con argumentos fijos. Fase B: el
// webview manda el id de lo que la persona confirmó con «s  Sí» y la extensión llama 'qdd consola accion <id>
// --confirmado' (qdd vuelve a calcular la acción y la autoriza; un pedido se lanza aparte, FND-089).

const vscode = require('vscode')
const { spawn, execFile } = require('node:child_process')
const fs = require('node:fs')
const { vista, carpetaDe, accionDe } = require('./panel.js')
const { crearLector } = require('./lector.js')

const TOPE_MS = 3000
// TOPE_ACCION_MS: aprobar una ficha o lanzar un pedido es corto (el pedido corre aparte), pero no tanto como leer.
const TOPE_ACCION_MS = 30000
const CODIGO_TODAVIA_NO = 75
const LINEAS_RESULTADO = 4
const SIN_PROYECTO = 'Abre una carpeta con .qdd para ver su panel'

const binario = () => process.env.QDD_BIN || 'qdd'
const ultimasLineas = (texto) => String(texto || '').trim().split('\n').slice(-LINEAS_RESULTADO).join('\n')

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
<script src="${recurso('pasos.js')}"></script>
<script src="${recurso('vista.js')}"></script>
</body>
</html>`
}

function crearPanel(extensionUri) {
  let vistaWeb = null
  let lector = null
  let elegida = ''
  let ultima = vista(null, '')
  let datos = null
  let ocupado = false

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
      alDatos: (leidos) => {
        datos = leidos
        dibujar(vista(leidos, ''))
      },
      alFallar: (motivo) => {
        datos = null
        dibujar(vista(null, motivo))
      },
    })
    if (vistaWeb?.visible) lector.iniciar()
  }

  const avisar = (resultado) => {
    if (vistaWeb) vistaWeb.webview.postMessage({ tipo: 'resultado', ...resultado })
  }

  // ejecutar corre lo que la persona confirmó, una cosa a la vez; lo que la vista no ofrece no llega a qdd.
  const ejecutar = (id) => {
    const a = accionDe(datos, id)
    const { carpeta } = proyecto()
    if (!a || ocupado || !carpeta) return
    ocupado = true
    const base = { id: a.id, titulo: a.etiqueta }
    avisar({ ...base, ejecutando: true, ok: true, todaviaNo: false, texto: 'ejecutando…' })
    const args = ['consola', 'accion', a.id, ...(a.confirmar ? ['--confirmado'] : [])]
    execFile(binario(), args, { cwd: carpeta, timeout: TOPE_ACCION_MS, windowsHide: true }, (err, salida, stderr) => {
      ocupado = false
      const codigo = typeof err?.code === 'number' ? err.code : err ? -1 : 0
      const texto = ultimasLineas(String(salida || '') + String(stderr || '')) || (err && !salida ? String(err.message) : '') || 'sin salida'
      avisar({ ...base, ejecutando: false, ok: codigo === 0, todaviaNo: codigo === CODIGO_TODAVIA_NO, texto })
    })
  }

  return {
    resolveWebviewView(view) {
      vistaWeb = view
      view.webview.options = { enableScripts: true, localResourceRoots: [vscode.Uri.joinPath(extensionUri, 'media')] }
      view.webview.html = htmlDe(view.webview, extensionUri)
      view.webview.onDidReceiveMessage((m) => {
        if (m?.tipo === 'listo') enviar()
        if (m?.tipo === 'accion') ejecutar(m.id)
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
