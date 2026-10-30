/** @jsxImportSource @opentui/solid */
// Panel de qdd en la barra lateral de OpenCode (RFC 0006 §3.10). Lo instala y lo mantiene al día 'qdd consola activar'
// (y 'actualizar' / 'doctor --fix'); su versión es siempre la de la CLI. Solo lectura: cada 2 s lee el mismo
// 'qdd consola panel --json' que el panel de Claude Code y redibuja solo si algo cambió. Sin botones ni diálogos.
// La tarea en curso (§3.13, blueprint estado-por-tareas) dice «● running»: el ● late con un tic local y destella con
// cada pulso. Su única escritura son los pulsos de su agente ('qdd estado pulso', como mucho cada 5 s).
import { createSignal, For, Show } from "solid-js"
import { INTERVALO_MS, LATIDO_MS, TOPE_MS, brillo, destellosNuevos, fuenteDePulso, motivoDeSalida, tocaEnviar, vista } from "./panel.js"

const VERSION = "__QDD_VERSION__"

const bin = () => process.env.QDD_BIN || "qdd"

// pulsar deja la señal de vida sin esperar: OpenCode sigue igual aunque qdd tarde o falle.
function pulsar(dir: string, fuente: string) {
  try {
    Bun.spawn([bin(), "estado", "pulso", "--fuente", fuente], { cwd: dir, stdout: "ignore", stderr: "ignore", timeout: TOPE_MS })
  } catch {}
}

async function leer(dir: string): Promise<{ datos: any; falla: string }> {
  try {
    const proc = Bun.spawn([bin(), "consola", "panel", "--json"], {
      cwd: dir,
      stdout: "pipe",
      stderr: "pipe",
      timeout: TOPE_MS,
    })
    const [salida, errores, codigo] = await Promise.all([
      new Response(proc.stdout).text(),
      new Response(proc.stderr).text(),
      proc.exited,
    ])
    const falla = motivoDeSalida(codigo, errores)
    if (falla) return { datos: null, falla }
    return { datos: JSON.parse(salida), falla: "" }
  } catch (e) {
    return { datos: null, falla: "no se pudo ejecutar qdd: " + String((e as Error)?.message ?? e) }
  }
}

const tui = async (api: any) => {
  const [actual, poner] = createSignal(vista(null, ""))
  const [tic, setTic] = createSignal(0)
  const [destellos, setDestellos] = createSignal<Record<string, number>>({})
  const vistos: Record<string, string> = {}
  let ultimo = ""
  let leyendo = false
  let ultimoEnvio = -Infinity
  const destellar = (ids: string[]) => {
    if (ids.length === 0) return
    const d = { ...destellos() }
    for (const id of ids) d[id] = 2
    setDestellos(d)
  }
  const refrescar = async () => {
    if (leyendo) return
    leyendo = true
    const r = await leer(api.state.path.directory)
    leyendo = false
    const v = vista(r.datos, r.falla)
    const huella = JSON.stringify(v)
    if (huella === ultimo) return
    ultimo = huella
    destellar(destellosNuevos(vistos, v.filas))
    poner(v)
  }
  refrescar()
  const reloj = setInterval(refrescar, INTERVALO_MS)
  // El latido solo cambia algo si hay una tarea en curso o un destello pendiente.
  const latido = setInterval(() => {
    const d = destellos()
    const habia = Object.values(d).some((n) => n > 0)
    if (habia) setDestellos(Object.fromEntries(Object.entries(d).map(([id, n]) => [id, Math.max(0, n - 1)])))
    if (!habia && !(actual().filas || []).some((f: any) => f.running)) return
    setTic(tic() + 1)
  }, LATIDO_MS)
  const quitarEvento = api.event.on("message.part.updated", (evento: any) => {
    const fuente = fuenteDePulso(evento)
    const enCurso = actual().actual
    if (!fuente || !enCurso) return
    destellar([enCurso])
    const ahora = Date.now()
    if (!tocaEnviar(ahora, ultimoEnvio)) return
    ultimoEnvio = ahora
    pulsar(api.state.path.directory, fuente)
  })
  api.lifecycle.onDispose(() => {
    clearInterval(reloj)
    clearInterval(latido)
    quitarEvento()
  })

  api.slots.register({
    order: 50,
    slots: {
      sidebar_content(ctx: any) {
        const t = () => ctx.theme.current
        const v = actual
        return (
          <box flexDirection="column" paddingBottom={1}>
            <box flexDirection="row" justifyContent="space-between">
              <text fg={t().text}>
                <b>{v().titulo || "Tareas"}</b>
              </text>
              <text fg={t().textMuted}>{v().version || "qdd " + VERSION}</text>
            </box>
            <Show when={v().motivo}>
              <text fg={t().textMuted}>{v().motivo}</text>
            </Show>
            <Show when={v().filas}>
              <text>
                <For each={v().resumen}>
                  {(r: any, i: () => number) => (
                    <span style={{ fg: t()[r.tono] }}>{(i() > 0 ? " · " : "") + r.texto}</span>
                  )}
                </For>
              </text>
              <text fg={t().textMuted}>{v().cabecera + "  Tarea"}</text>
              <For each={v().filas}>
                {(f: any) => (
                  <text>
                    <span style={{ fg: t()[f.tono] }}>{f.celdas}</span>
                    <Show when={f.running}>
                      {(() => {
                        const b = () => brillo(tic(), (destellos()[f.id] || 0) > 0)
                        const color = () => (b() === "acento" ? t().accent : b() === "brillante" ? t()[f.tono] : t().textMuted)
                        return (
                          <>
                            <span style={{ fg: color() }}>{"  ●"}</span>
                            <span style={{ fg: t().textMuted }}>{" running"}</span>
                          </>
                        )
                      })()}
                    </Show>
                    <span style={{ fg: t().text }}>{"  " + f.nombre}</span>
                  </text>
                )}
              </For>
              <Show when={v().ocultas > 0}>
                <text fg={t().textMuted}>{"+" + v().ocultas + " más antiguas · qdd estado ver"}</text>
              </Show>
              <For each={v().fallas}>
                {(p: any) => <text fg={t().error}>{"❌ " + p.titulo + ": " + p.texto}</text>}
              </For>
            </Show>
            <Show when={v().fichas > 0}>
              <text fg={t().accent}>{v().fichas + " ficha(s) por aprobar: hazlo desde Claude Code o con qdd modules approve"}</text>
            </Show>
          </box>
        )
      },
    },
  })
}

export default { id: "qdd.tareas", tui }
