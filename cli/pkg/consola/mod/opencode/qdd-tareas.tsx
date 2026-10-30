/** @jsxImportSource @opentui/solid */
// Panel de qdd en la barra lateral de OpenCode (RFC 0006 §3.10). Lo instala y lo mantiene al día 'qdd consola activar'
// (y 'actualizar' / 'doctor --fix'); su versión es siempre la de la CLI. Solo lectura: cada 2 s lee el mismo
// 'qdd consola panel --json' que el panel de Claude Code y redibuja solo si algo cambió. Sin botones ni diálogos.
import { createSignal, For, Show } from "solid-js"
import { INTERVALO_MS, TOPE_MS, motivoDeSalida, vista } from "./panel.js"

const VERSION = "__QDD_VERSION__"

async function leer(dir: string): Promise<{ datos: any; falla: string }> {
  try {
    const proc = Bun.spawn([process.env.QDD_BIN || "qdd", "consola", "panel", "--json"], {
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
  let ultimo = ""
  let leyendo = false
  const refrescar = async () => {
    if (leyendo) return
    leyendo = true
    const r = await leer(api.state.path.directory)
    leyendo = false
    const v = vista(r.datos, r.falla)
    const huella = JSON.stringify(v)
    if (huella === ultimo) return
    ultimo = huella
    poner(v)
  }
  refrescar()
  const reloj = setInterval(refrescar, INTERVALO_MS)
  api.lifecycle.onDispose(() => clearInterval(reloj))

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
                    <span style={{ fg: t().text }}>{"  " + f.nombre}</span>
                  </text>
                )}
              </For>
              <Show when={v().ocultas > 0}>
                <text fg={t().textMuted}>{"+" + v().ocultas + " más antiguas · qdd estado ver"}</text>
              </Show>
              <For each={v().esperas}>
                {(p: any) => <text fg={t().warning}>{"⏳ " + p.titulo + ": " + p.texto}</text>}
              </For>
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
