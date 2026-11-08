import { expect, mock, test } from 'claude-code/testing'

// RFC 0025, fase 3b (pedido de la persona, 2026-11-08): «cuando el flujo está armado debería bastar con un click en una
// esquina que diga compartir para que genere el enlace». El panel corre qdd flujo compartir --enlace con su sesión
// (para el permiso del baúl), copia el enlace y lo muestra solo en el panel: nunca en el chat.

const PANEL = {
  plugin: 'qdd-tareas',
  component: 'Pane',
  requestId: 'qdd-flujos',
  viewport: { columns: 160, rows: 40 },
  props: { title: 'Flujos', isFocused: true, bodyColumns: 60, placement: 'dock', scroll: { offset: 0, bodyRows: 30 }, viewportRows: 40 },
} as const

const DIBUJO = { id: 'pago', nombre: 'Cómo se cobra', lineas: ['( Llega )'], nodos: [{ id: 'llega', tipo: 'paso', nombre: 'Llega' }], por_enviar: [], puede_deshacer: false }
const ENLACE = 'https://qdd-flujos.ana.workers.dev/f/abcdefghijklmnopqrstuvwxyz#t=tok'

function preparar(on, r) {
  mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('classic.UserPromptSubmit', () => ({}))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('ui.copy', ($, e) => {
    r.copiado.push(e.text)
    return { value: { copied: true } }
  })
  on('prompt.submit', ($, e) => {
    r.enviados.push(e.text)
    return { text: e.text }
  })
  on('process.run', ($, e) => {
    r.llamadas.push([...e.argv])
    if (e.argv.includes('binario')) return { value: { exitCode: 1, stdout: '', stderr: '' } }
    if (e.argv.includes('panel')) return { value: { exitCode: 0, stdout: JSON.stringify({ formato: 1, carpeta: '/work', acciones: [], herramientas: [], flujos: [{ id: 'pago', nombre: 'Cómo se cobra', nodos: 1, huella: 'h1' }] }), stderr: '' } }
    if (e.argv[2] === 'ver') return { value: { exitCode: 0, stdout: JSON.stringify(DIBUJO), stderr: '' } }
    if (e.argv[2] === 'compartir') return { value: r.respuesta }
    return { value: { exitCode: 0, stdout: '{}', stderr: '' } }
  })
}

const nuevo = (respuesta) => ({ llamadas: [] as string[][], copiado: [] as string[], enviados: [] as string[], respuesta })
const esperar = () => new Promise((res) => setTimeout(res, 20))

async function abrir($, on, r) {
  preparar(on, r)
  await $.session.start({ surface: 'terminal', isInteractive: true, cwd: '/work' })
  const ui = await $.ui.mount({ ...PANEL, surface: 'terminal' })
  await esperar()
  return ui
}

test('un clic en Compartir genera el enlace, lo copia y lo muestra solo en el panel', async ($, on) => {
  const r = nuevo({ exitCode: 0, stdout: JSON.stringify({ enlace: ENLACE, invitacion: 'inv-1', mensaje: 'Enlace para quien invitas' }) + '\n', stderr: '' })
  const ui = await abrir($, on, r)
  await ui.press({ key: 'fl-compartir' })
  await esperar()
  const llamada = r.llamadas.find((a) => a[2] === 'compartir')
  expect(llamada).toEqual(['qdd', 'flujo', 'compartir', '--enlace', '--rol', 'cliente', '--json', '--sesion', 'sesion-1', '--', 'pago'])
  expect(r.copiado).toEqual([ENLACE])
  expect(await ui.find({ type: 'Text', text: /Enlace copiado/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: ENLACE })).toBeDefined()
  expect(r.enviados).toEqual([])
})

test('sin permiso del baúl dice dónde darlo; sin Cloudflare, dónde quedó el archivo', async ($, on) => {
  const r = nuevo({ exitCode: 75, stdout: JSON.stringify({ permiso: 'pendiente', mensaje: 'Falta tu permiso: confírmalo en «Para ti» (🔑)' }) + '\n', stderr: '' })
  const ui = await abrir($, on, r)
  await ui.press({ key: 'fl-compartir' })
  await esperar()
  expect(await ui.find({ type: 'Text', text: /Falta tu permiso/ })).toBeDefined()
  expect(r.copiado).toEqual([])
  r.respuesta = { exitCode: 0, stdout: JSON.stringify({ archivo: '/work/tmp/flujo-pago.html', mensaje: 'Sin Cloudflare en el baúl, quedó el archivo' }) + '\n', stderr: '' }
  await ui.press({ key: 'fl-compartir' })
  await esperar()
  expect(await ui.find({ type: 'Text', text: /\/work\/tmp\/flujo-pago\.html/ })).toBeDefined()
  r.respuesta = { exitCode: 1, stdout: '', stderr: '[!] Cloudflare respondió 403' }
  await ui.press({ key: 'fl-compartir' })
  await esperar()
  expect(await ui.find({ type: 'Text', text: /Cloudflare respondió 403/ })).toBeDefined()
})
