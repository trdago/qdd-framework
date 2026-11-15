// Lector del panel (RFC 0014, RFC 0016): mientras la vista se mira, sigue 'qdd consola seguir', un proceso que entrega
// una línea JSON por cada cambio. Un qdd sin ese comando imprime la ayuda y sale con 0, así que solo una línea JSON
// cuenta como datos: si el flujo termina sin ninguna, lee 'qdd consola panel --json' cada 2 s. Sin dependencias de
// vscode: abrir, leer y el reloj se inyectan para probarlo con node --test.

const { motivoDeSalida } = require('./panel.js')

const ARGS_SEGUIR = ['consola', 'seguir']
const ARGS_PANEL = ['consola', 'panel', '--json']
const INTERVALO_MS = 2000
const SALTO = 10

const parsear = (texto) => {
  try {
    return { ok: true, valor: JSON.parse(texto) }
  } catch {
    return { ok: false }
  }
}

function motivoDeError(err, stderr) {
  if (err.code === 'ENOENT') return 'No se encontró qdd: instálalo o déjalo en el PATH'
  if (typeof err.code === 'number') return motivoDeSalida(err.code, stderr)
  return (stderr || '').trim() || err.message
}

function crearLector({ abrir, leer, alDatos, alFallar, programar = setTimeout, cancelar = clearTimeout }) {
  let activo = false
  let ronda = 0 // cada iniciar y detener la cambia: lo que llega de una ronda anterior no dibuja
  let proceso = null
  let tiempo = null
  let ultima = ''
  let ultimaFalla = ''

  const fallar = (motivo) => {
    if (motivo === ultimaFalla) return
    ultimaFalla = motivo
    alFallar(motivo)
  }

  // entregar dibuja un JSON válido si cambió; dice si era válido.
  const entregar = (texto) => {
    const leido = parsear(texto)
    if (!leido.ok) return false
    if (texto === ultima) return true
    ultima = texto
    ultimaFalla = ''
    alDatos(leido.valor)
    return true
  }

  const sondear = () => {
    const mia = ronda
    leer(ARGS_PANEL, (err, salida, stderr) => {
      if (mia !== ronda) return
      tiempo = programar(sondear, INTERVALO_MS)
      if (err) return fallar(motivoDeError(err, stderr))
      if (!entregar(String(salida).trim())) fallar('qdd entregó algo que no es JSON')
    })
  }

  const seguir = () => {
    const mia = ronda
    let pendiente = Buffer.alloc(0)
    let conDatos = false
    let cerrado = false
    const p = abrir(ARGS_SEGUIR)
    proceso = p
    p.stdout.on('data', (trozo) => {
      if (mia !== ronda) return
      pendiente = Buffer.concat([pendiente, trozo])
      for (let i = pendiente.indexOf(SALTO); i >= 0; i = pendiente.indexOf(SALTO)) {
        const linea = pendiente.subarray(0, i).toString('utf8').trim()
        pendiente = pendiente.subarray(i + 1)
        if (!linea) continue
        if (entregar(linea)) {
          conDatos = true
          continue
        }
        if (conDatos) fallar('qdd entregó algo que no es JSON')
      }
    })
    const alSalir = () => {
      if (cerrado) return
      cerrado = true
      if (mia !== ronda) return
      proceso = null
      if (!conDatos) return sondear()
      tiempo = programar(seguir, INTERVALO_MS)
    }
    p.on('error', alSalir)
    p.on('close', alSalir)
  }

  return {
    iniciar() {
      if (activo) return
      activo = true
      ronda++
      ultima = ''
      ultimaFalla = ''
      seguir()
    },
    detener() {
      if (!activo) return
      activo = false
      ronda++
      if (tiempo) cancelar(tiempo)
      tiempo = null
      if (proceso) proceso.kill()
      proceso = null
    },
  }
}

module.exports = { crearLector, ARGS_SEGUIR, ARGS_PANEL, INTERVALO_MS }
