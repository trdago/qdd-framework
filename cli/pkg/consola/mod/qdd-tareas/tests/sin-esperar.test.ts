import { expect, mock, test } from 'claude-code/testing'

// MEJ-015 (issue 113): un mensaje, el inicio de sesión y el final de una herramienta no quedan a merced de lo que tarde la
// CLI. En un Windows cargado, cada llamada cuesta cientos de milisegundos.

const PRESUPUESTO_MS = 400
const LINEA = '[qdd · panel] Espera a la persona: pedido a «Merge #7».'
const PANEL = JSON.stringify({ formato: 1, estado: null, motivo: '', acciones: [{ id: 'aprobar:x', tipo: 'aprobar', etiqueta: 'Aprobar x' }] })
const INICIO = { surface: 'terminal', isInteractive: true, cwd: '/work' }

const ok = (stdout: string) => ({ exitCode: 0, stdout, stderr: '' })
const pausa = () => new Promise((res) => setTimeout(res, 20))

function compuerta() {
  let abrir: () => void = () => {}
  const abierta = new Promise<void>((res) => (abrir = res))
  return { abierta, abrir }
}

// preparar: r.retener deja esperando al subcomando de 'qdd consola' que se nombre; r.ciclo es lo que responde el ciclo.
function preparar(on, r) {
  const clock = mock.clock(on)
  mock.env(on, {})
  on('session.start', () => ({ cwd: '/work' }))
  on('session.id', () => ({ value: 'sesion-1' }))
  on('command.register', () => ({ value: undefined }))
  on('ui.open', () => {
    r.abiertos += 1
    return { value: { isPlaced: true } }
  })
  on('classic.UserPromptSubmit', () => ({}))
  on('tool.call', () => ({ result: 'hecho' }))
  on('process.run', async ($, e) => {
    const sub = e.argv[2]
    r.llamadas.push(sub)
    if (r.retener[sub]) await r.retener[sub]
    if (sub === 'ciclo') return { value: ok(r.ciclo) }
    return { value: ok(PANEL) }
  })
  return clock
}

function nuevo(): any {
  return { abiertos: 0, llamadas: [] as string[], retener: {}, ciclo: '' }
}

const lecturas = (r) => r.llamadas.filter((sub: string) => sub === 'panel').length

test('un mensaje espera a un qdd lento solo su presupuesto, y lo que llega tarde viaja con el siguiente', async ($, on) => {
  const r = nuevo()
  const clock = preparar(on, r)
  await $.session.start(INICIO)
  const lento = compuerta()
  r.retener.ciclo = lento.abierta
  r.ciclo = LINEA

  let listo = false
  const primero = $.classic.UserPromptSubmit({ prompt: 'a', permission_mode: 'auto' }).then((x) => {
    listo = true
    return x
  })
  await clock.advance(PRESUPUESTO_MS - 1)
  await pausa()
  expect(listo).toBe(false)
  await clock.advance(1)
  await pausa()
  expect(listo).toBe(true)
  expect((await primero).additionalContext).toBeUndefined()

  lento.abrir()
  await pausa()
  delete r.retener.ciclo
  r.ciclo = ''
  const segundo = await $.classic.UserPromptSubmit({ prompt: 'b', permission_mode: 'auto' })
  expect(segundo.additionalContext).toEqual([LINEA])
  const tercero = await $.classic.UserPromptSubmit({ prompt: 'c', permission_mode: 'auto' })
  expect(tercero.additionalContext).toBeUndefined()
})

test('lo que llegó tarde y lo nuevo viajan juntos, primero lo atrasado', async ($, on) => {
  const r = nuevo()
  const clock = preparar(on, r)
  await $.session.start(INICIO)
  const lento = compuerta()
  r.retener.ciclo = lento.abierta
  r.ciclo = 'atrasado'
  const primero = $.classic.UserPromptSubmit({ prompt: 'a', permission_mode: 'auto' })
  await clock.advance(PRESUPUESTO_MS)
  await primero
  lento.abrir()
  await pausa()
  delete r.retener.ciclo
  r.ciclo = 'nuevo'
  const segundo = await $.classic.UserPromptSubmit({ prompt: 'b', permission_mode: 'auto' })
  expect(segundo.additionalContext).toEqual(['atrasado', 'nuevo'])
})

test('la sesión no espera a un qdd lento más que su presupuesto, y el panel se abre cuando llega la lectura', async ($, on) => {
  const r = nuevo()
  const clock = preparar(on, r)
  const lento = compuerta()
  r.retener.panel = lento.abierta

  let listo = false
  const inicio = $.session.start(INICIO).then(() => {
    listo = true
  })
  await pausa()
  await clock.advance(PRESUPUESTO_MS)
  await pausa()
  expect(listo).toBe(true)
  expect(r.abiertos).toBe(0)

  lento.abrir()
  await pausa()
  await inicio
  expect(r.abiertos).toBeGreaterThan(0)
})

test('el final de una tool de qdd no espera a la lectura del panel', async ($, on) => {
  const r = nuevo()
  preparar(on, r)
  await $.session.start(INICIO)
  const lento = compuerta()
  r.retener.panel = lento.abierta
  const fin = await Promise.race([$.tool.call({ tool: 'mcp__qdd__qdd_estado' }), pausa().then(() => 'esperó a la lectura')])
  expect(fin).toEqual({ result: 'hecho' })
  lento.abrir()
})

test('una ráfaga de tools de qdd lanza una lectura en curso y una más, no una por tool', async ($, on) => {
  const r = nuevo()
  preparar(on, r)
  await $.session.start(INICIO)
  const lento = compuerta()
  r.retener.panel = lento.abierta
  const antes = lecturas(r)
  for (let i = 0; i < 5; i += 1) {
    await Promise.race([$.tool.call({ tool: 'mcp__qdd__qdd_estado' }), pausa()])
  }
  await pausa()
  expect(lecturas(r) - antes).toBe(1)

  lento.abrir()
  await pausa()
  expect(lecturas(r) - antes).toBe(2)
})
