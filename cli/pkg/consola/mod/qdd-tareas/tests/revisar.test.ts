import { expect, mock, test } from 'claude-code/testing'

// Pedido de la persona (2026-11-26): «actualizar, la l del menú, debería volver a revisar todo lo que está en el panel,
// porque preguntas van quedando en el camino como residuo […] y ya están contestadas». La l relee el tablero y le manda
// a la IA lo que qdd consola revisar arma con todo lo que espera a la persona; la IA lo evalúa con evidencia y deja solo
// lo que aporta valor. Dos l seguidas mandan una sola revisión.

const PANEL = {
  formato: 1,
  carpeta: '/work/bot',
  estado: { formato: 1, resumen: '', etapas: ['Código'], abreviaturas: ['Có'], frentes: [{ id: 'a', nombre: 'Algo', estado: 'abierto', simbolos: ['⏳'] }], esperas: [], fallas: [] },
  acciones: [],
}
const REVISION = { texto: 'Revisa todo lo que el panel espera de mí…\n\n⏳ que me esperan (2):\n- a · Código: ¿Sigue en pie?', total: 3 }

type Salida = { exitCode: number; stdout: string; stderr: string }
const ok = (d: unknown): Salida => ({ exitCode: 0, stdout: JSON.stringify(d), stderr: '' })

const tareas = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-tareas',
  viewport: { columns: 120, rows: 40 },
  props: { title: 'Tareas', isFocused: true, bodyColumns: 120, placement: 'dock', scroll: { offset: 0, bodyRows: 30 }, view: {} },
} as const

function preparar(on, r) {
  // Reloj propio y sin ciclos: la prueba decide cuánto pasa entre una l y otra.
  on('clock.every', () => ({ value: undefined }))
  on('clock.now', () => ({ value: r.ahora }))
  on('clock.sleep', () => ({ value: undefined }))
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work/bot' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('ui.log', () => ({ value: undefined }))
  on('prompt.submit', ($, e) => {
    r.enviados.push(e.text)
    return { text: e.text }
  })
  on('process.run', ($, e) => {
    const argv = [...e.argv]
    if (argv[1] === 'consola' && argv[2] === 'revisar') {
      r.revisiones += 1
      return { value: r.revision }
    }
    if (argv[1] === 'herramienta') return { value: ok({ herramientas: [], problemas: null }) }
    return { value: ok(PANEL) }
  })
}

const nuevo = (revision: Salida) => ({ ahora: 1_000_000, enviados: [] as string[], revisiones: 0, revision })

async function montar($, on, r) {
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work/bot' })
  return await $.ui.mount({ ...tareas, surface: 'terminal' })
}

test('l le manda a la IA la revisión de todo lo que te espera', async ($, on) => {
  const r = nuevo(ok(REVISION))
  const ui = await montar($, on, r)
  await ui.press({ key: 'leer' })
  expect(r.enviados).toEqual([REVISION.texto])
  expect(await ui.find({ type: 'Text', text: /la IA revisa 3/ })).toBeDefined()
})

test('sin nada que te espere, l solo relee y no le escribe a la IA', async ($, on) => {
  const r = nuevo(ok({ texto: '', total: 0 }))
  const ui = await montar($, on, r)
  await ui.press({ key: 'leer' })
  expect(r.revisiones).toBe(1)
  expect(r.enviados).toEqual([])
  expect(await ui.find({ type: 'Text', text: /Nada te espera/ })).toBeDefined()
})

test('dos l seguidas mandan una sola revisión; pasado el minuto, otra', async ($, on) => {
  const r = nuevo(ok(REVISION))
  const ui = await montar($, on, r)
  await ui.press({ key: 'leer' })
  r.ahora += 5_000
  await ui.press({ key: 'leer' })
  expect(r.enviados.length).toBe(1)
  r.ahora += 61_000
  await ui.press({ key: 'leer' })
  expect(r.enviados.length).toBe(2)
})

test('un qdd sin «consola revisar» deja la l como antes: relee y no escribe', async ($, on) => {
  const r = nuevo({ exitCode: 1, stdout: '', stderr: 'unknown command "revisar" for "qdd consola"' })
  const ui = await montar($, on, r)
  await ui.press({ key: 'leer' })
  expect(r.enviados).toEqual([])
})
