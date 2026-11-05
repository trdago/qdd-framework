const { test } = require('node:test');
const assert = require('node:assert');
const { spawnSync } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

// FND-135: mientras npm reinstala qdd el binario falta unos segundos. Un pedido que llama a qdd en ese momento debe
// salir con 75 («todavía no»), para que el panel lo reintente, en vez de fallar pidiendo que se reinstale.

const envoltorio = path.join(__dirname, '..', 'bin', 'qdd.js');
const nombreBinario = os.platform() === 'win32' ? 'qdd.exe' : 'qdd';

function carpetaConEnvoltorio(t) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'qdd-envoltorio-'));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
  fs.copyFileSync(envoltorio, path.join(dir, 'qdd.js'));
  return dir;
}

test('sin binario sale con 75 y dice que qdd se está actualizando', (t) => {
  const dir = carpetaConEnvoltorio(t);
  const r = spawnSync(process.execPath, [path.join(dir, 'qdd.js'), '--version'], { encoding: 'utf8' });
  assert.strictEqual(r.status, 75, r.stderr);
  assert.match(r.stderr, /actualizando/);
  assert.match(r.stderr, /npm install -g qdd-framework/);
});

test('con binario entrega su salida tal cual', { skip: os.platform() === 'win32' }, (t) => {
  const dir = carpetaConEnvoltorio(t);
  const binario = path.join(dir, nombreBinario);
  fs.writeFileSync(binario, '#!/bin/sh\nexit 7\n', { mode: 0o755 });
  const r = spawnSync(process.execPath, [path.join(dir, 'qdd.js')], { encoding: 'utf8' });
  assert.strictEqual(r.status, 7);
});

test('un binario que no se puede ejecutar sigue siendo un fallo', { skip: os.platform() === 'win32' }, (t) => {
  const dir = carpetaConEnvoltorio(t);
  fs.writeFileSync(path.join(dir, nombreBinario), 'nada', { mode: 0o644 });
  const r = spawnSync(process.execPath, [path.join(dir, 'qdd.js')], { encoding: 'utf8' });
  assert.strictEqual(r.status, 1, r.stderr);
  assert.match(r.stderr, /Reinstala/);
});
