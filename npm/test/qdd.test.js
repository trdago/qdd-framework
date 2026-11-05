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
  // FND-143: npm 12 bloquea el postinstall por defecto: reinstalar sin esta bandera deja a qdd igual, sin binario.
  assert.match(r.stderr, /npm install -g qdd-framework --allow-scripts=qdd-framework/);
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

// FND-142: en Windows, qdd.exe vive fuera del paquete para que npm pueda actualizar con sesiones abiertas.
const { rutaDelBinario } = require('../bin/qdd.js');

test('en Windows usa el binario de la versión del paquete', () => {
  const ruta = rutaDelBinario({ platform: 'win32', env: { LOCALAPPDATA: 'L' }, dir: 'paquete', version: '2.10.0', existe: () => true });
  assert.strictEqual(ruta, path.join('L', 'qdd', 'bin', '2.10.0', 'qdd.exe'));
});

test('en Windows sin esa carpeta usa el del paquete: una instalación anterior sigue andando', () => {
  const dentro = path.join('paquete', 'qdd.exe');
  assert.strictEqual(rutaDelBinario({ platform: 'win32', env: { LOCALAPPDATA: 'L' }, dir: 'paquete', version: '2.10.0', existe: () => false }), dentro);
  assert.strictEqual(rutaDelBinario({ platform: 'win32', env: {}, dir: 'paquete', version: '2.10.0', existe: () => true }), dentro);
  assert.strictEqual(rutaDelBinario({ platform: 'win32', env: { LOCALAPPDATA: 'L' }, dir: 'paquete', version: '', existe: () => true }), dentro);
});

test('fuera de Windows nada cambia', () => {
  const ruta = rutaDelBinario({ platform: 'linux', env: { LOCALAPPDATA: 'L' }, dir: 'paquete', version: '2.10.0', existe: () => true });
  assert.strictEqual(ruta, path.join('paquete', 'qdd'));
});

// FND-143: quien instala sin la bandera de npm 12 queda sin binario: el primer qdd lo baja con el instalador del paquete.
function paqueteConInstalador(t, instalador) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'qdd-primer-uso-'));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
  fs.mkdirSync(path.join(dir, 'bin'));
  fs.copyFileSync(envoltorio, path.join(dir, 'bin', 'qdd.js'));
  fs.writeFileSync(path.join(dir, 'install.js'), instalador);
  return dir;
}

const correr = (dir, ...args) => spawnSync(process.execPath, [path.join(dir, 'bin', 'qdd.js'), ...args], { encoding: 'utf8' });

test('sin binario lo baja con el instalador y corre lo pedido', { skip: os.platform() === 'win32' }, (t) => {
  const dir = paqueteConInstalador(t, `
    const fs = require('fs');
    const path = require('path');
    if (!process.argv.includes('--primer-uso')) process.exit(9);
    console.log('ruido del instalador');
    fs.writeFileSync(path.join(__dirname, 'bin', 'qdd'), '#!/bin/sh\\necho salida-de-qdd\\nexit 7\\n', { mode: 0o755 });
  `);
  const r = correr(dir, 'consola', 'panel');
  assert.strictEqual(r.status, 7, r.stderr);
  assert.strictEqual(r.stdout, 'salida-de-qdd\n', 'lo del instalador no ensucia una salida que otro programa lee');
  assert.match(r.stderr, /ruido del instalador/);
});

test('con binario no llama al instalador', { skip: os.platform() === 'win32' }, (t) => {
  const dir = paqueteConInstalador(t, 'process.exit(1);');
  fs.writeFileSync(path.join(dir, 'bin', nombreBinario), '#!/bin/sh\nexit 0\n', { mode: 0o755 });
  const r = correr(dir);
  assert.strictEqual(r.status, 0);
  assert.strictEqual(r.stderr, '');
});

test('si el instalador no puede bajarlo sale con 75 y da el comando', (t) => {
  const dir = paqueteConInstalador(t, 'process.exit(1);');
  const r = correr(dir);
  assert.strictEqual(r.status, 75, r.stderr);
  assert.match(r.stderr, /--allow-scripts=qdd-framework/);
});
