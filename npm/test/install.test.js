const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('fs');
const os = require('os');
const path = require('path');

const { resolverBinarioLocal, estaDentroDeNodeModules } = require('../install.js');

function crearRaiz(t) {
  const raiz = fs.mkdtempSync(path.join(os.tmpdir(), 'qdd-install-'));
  t.after(() => fs.rmSync(raiz, { recursive: true, force: true }));
  return raiz;
}

function crearPaquete(raiz, ...segmentos) {
  const pkgDir = path.join(raiz, ...segmentos);
  fs.mkdirSync(pkgDir, { recursive: true });
  return pkgDir;
}

test('instalación global: una carpeta node_modules/qdd (otro paquete) no se toma como binario', (t) => {
  const raiz = crearRaiz(t);
  const pkgDir = crearPaquete(raiz, 'lib', 'node_modules', 'qdd-framework');
  fs.mkdirSync(path.join(raiz, 'lib', 'node_modules', 'qdd'));

  assert.equal(resolverBinarioLocal(pkgDir, 'qdd', () => {}), null);
});

test('instalación global: ni siquiera un archivo ../qdd se usa dentro de node_modules', (t) => {
  const raiz = crearRaiz(t);
  const pkgDir = crearPaquete(raiz, 'node_modules', 'qdd-framework');
  fs.writeFileSync(path.join(raiz, 'node_modules', 'qdd'), 'binario');

  assert.equal(resolverBinarioLocal(pkgDir, 'qdd', () => {}), null);
});

test('repo de desarrollo: un archivo ../qdd se usa como binario local', (t) => {
  const raiz = crearRaiz(t);
  const pkgDir = crearPaquete(raiz, 'npm');
  const binario = path.join(raiz, 'qdd');
  fs.writeFileSync(binario, 'binario');

  assert.equal(resolverBinarioLocal(pkgDir, 'qdd', () => {}), binario);
});

test('repo de desarrollo: un archivo ../cli/qdd se usa como binario local', (t) => {
  const raiz = crearRaiz(t);
  const pkgDir = crearPaquete(raiz, 'npm');
  fs.mkdirSync(path.join(raiz, 'cli'));
  const binario = path.join(raiz, 'cli', 'qdd');
  fs.writeFileSync(binario, 'binario');

  assert.equal(resolverBinarioLocal(pkgDir, 'qdd', () => {}), binario);
});

test('repo de desarrollo: una carpeta ../qdd se ignora con aviso y sigue con ../cli/qdd', (t) => {
  const raiz = crearRaiz(t);
  const pkgDir = crearPaquete(raiz, 'npm');
  fs.mkdirSync(path.join(raiz, 'qdd'));
  fs.mkdirSync(path.join(raiz, 'cli'));
  const binario = path.join(raiz, 'cli', 'qdd');
  fs.writeFileSync(binario, 'binario');
  const avisos = [];

  assert.equal(resolverBinarioLocal(pkgDir, 'qdd', (m) => avisos.push(m)), binario);
  assert.equal(avisos.length, 1);
  assert.match(avisos[0], /no es un archivo/);
});

test('sin binario local devuelve null (se descarga desde GitHub Releases)', (t) => {
  const raiz = crearRaiz(t);
  const pkgDir = crearPaquete(raiz, 'npm');

  assert.equal(resolverBinarioLocal(pkgDir, 'qdd', () => {}), null);
});

test('estaDentroDeNodeModules reconoce rutas de instalación de npm', () => {
  assert.equal(estaDentroDeNodeModules(path.join(path.sep, 'usr', 'lib', 'node_modules', 'qdd-framework')), true);
  assert.equal(estaDentroDeNodeModules(path.join(path.sep, 'home', 'dev', 'qdd', 'npm')), false);
});

// MEJ-013 (issue #106): en Windows con Git Bash, `tar` es GNU tar, que no abre un zip. La instalación terminaba con
// exit 0 y sin bin/qdd.exe, y desde ahí todo `qdd` fallaba en todas las sesiones abiertas.
const { execFileSync } = require('node:child_process');
const { comandoExtraer, instalarDescarga, codigoDeFalla } = require('../install.js');

test('en Windows usa el tar del sistema con su ruta completa, no el primero del PATH', () => {
  const c = comandoExtraer('windows', 'C:\\t\\qdd.zip', 'C:\\t', { SystemRoot: 'D:\\WIN' });
  assert.strictEqual(c.programa, 'D:\\WIN\\System32\\tar.exe');
  assert.deepStrictEqual(c.args, ['-xf', 'C:\\t\\qdd.zip', '-C', 'C:\\t']);
});

test('en Windows sin SystemRoot usa la carpeta de siempre', () => {
  assert.strictEqual(comandoExtraer('windows', 'a.zip', 'd', {}).programa, 'C:\\Windows\\System32\\tar.exe');
});

test('fuera de Windows usa tar del PATH', () => {
  const c = comandoExtraer('linux', '/t/qdd.tar.gz', '/t', {});
  assert.strictEqual(c.programa, 'tar');
  assert.deepStrictEqual(c.args, ['-xzf', '/t/qdd.tar.gz', '-C', '/t']);
});

test('sin binario la instalación falla, salvo que se tolere a propósito', () => {
  assert.strictEqual(codigoDeFalla({}), 1);
  assert.strictEqual(codigoDeFalla({ QDD_INSTALL_TOLERANTE: '1' }), 0);
});

function carpetaDescarga(t, contenido) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'qdd-descarga-'));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
  const fuente = path.join(dir, 'fuente');
  fs.mkdirSync(fuente);
  for (const [nombre, texto] of Object.entries(contenido)) {
    fs.writeFileSync(path.join(fuente, nombre), texto, { mode: 0o755 });
  }
  const paquete = path.join(dir, 'qdd-download.tar.gz');
  execFileSync('tar', ['-czf', paquete, '-C', fuente, '.']);
  const binDir = path.join(dir, 'bin');
  fs.mkdirSync(binDir);
  return { dir, paquete, binDir, destino: path.join(binDir, 'qdd') };
}

const instalar = (d) => instalarDescarga({ os: 'linux', paquete: d.paquete, carpeta: d.dir, binaryName: 'qdd', binDir: d.binDir, env: {} });

test('instala un binario que responde y borra la descarga', { skip: os.platform() === 'win32' }, (t) => {
  const d = carpetaDescarga(t, { qdd: '#!/bin/sh\necho qdd version v9\n' });
  assert.strictEqual(instalar(d), d.destino);
  assert.ok(fs.existsSync(d.destino));
  assert.ok(!fs.existsSync(d.paquete));
});

test('un paquete sin binario falla y no toca el binario anterior', { skip: os.platform() === 'win32' }, (t) => {
  const d = carpetaDescarga(t, { otro: 'x' });
  fs.writeFileSync(d.destino, 'anterior');
  assert.throws(() => instalar(d), /no trae/);
  assert.strictEqual(fs.readFileSync(d.destino, 'utf8'), 'anterior');
});

test('un binario que no responde falla y no reemplaza al anterior', { skip: os.platform() === 'win32' }, (t) => {
  const d = carpetaDescarga(t, { qdd: '#!/bin/sh\nexit 3\n' });
  fs.writeFileSync(d.destino, 'anterior');
  assert.throws(() => instalar(d));
  assert.strictEqual(fs.readFileSync(d.destino, 'utf8'), 'anterior');
});

test('una descarga que tar no puede abrir falla', { skip: os.platform() === 'win32' }, (t) => {
  const d = carpetaDescarga(t, { qdd: 'x' });
  fs.writeFileSync(d.paquete, 'no es un tar');
  assert.throws(() => instalar(d));
  assert.ok(!fs.existsSync(d.destino));
});

// ── FND-142: en Windows, qdd.exe vive fuera del paquete, en una carpeta por versión ───────────────────────────────────

const { carpetaDeVersiones, destinoDelBinario, respondeBinario, moverBinario, instalarCopia, registrarVersion } = require('../install.js');

test('en Windows el binario va a una carpeta por versión, fuera del paquete', () => {
  const carpeta = carpetaDeVersiones({ os: 'windows', env: { LOCALAPPDATA: 'L' } });
  assert.strictEqual(carpeta, path.join('L', 'qdd', 'bin'));
  assert.strictEqual(destinoDelBinario({ carpeta, version: '2.10.0', binDir: 'paquete', binaryName: 'qdd.exe' }), path.join('L', 'qdd', 'bin', '2.10.0', 'qdd.exe'));
});

test('en Linux, en macOS y en Windows sin LOCALAPPDATA sigue dentro del paquete', () => {
  assert.strictEqual(carpetaDeVersiones({ os: 'linux', env: { LOCALAPPDATA: 'L' } }), '');
  assert.strictEqual(carpetaDeVersiones({ os: 'darwin', env: {} }), '');
  assert.strictEqual(carpetaDeVersiones({ os: 'windows', env: {} }), '');
  assert.strictEqual(destinoDelBinario({ carpeta: '', version: '2.10.0', binDir: 'paquete', binaryName: 'qdd' }), path.join('paquete', 'qdd'));
});

test('el binario de la descarga llega a su carpeta de versión, que se crea', { skip: os.platform() === 'win32' }, (t) => {
  const d = carpetaDescarga(t, { qdd: '#!/bin/sh\necho qdd version v9\n' });
  const destino = path.join(d.dir, 'local', 'qdd', 'bin', '9.0.0', 'qdd');
  assert.strictEqual(instalarDescarga({ os: 'linux', paquete: d.paquete, carpeta: d.dir, binaryName: 'qdd', binDir: d.binDir, env: {}, destino }), destino);
  assert.ok(fs.existsSync(destino));
  assert.ok(!fs.existsSync(d.destino), 'dentro del paquete no queda nada que npm no pueda reemplazar');
});

test('respondeBinario: solo un binario que corre', { skip: os.platform() === 'win32' }, (t) => {
  const raiz = crearRaiz(t);
  const bueno = path.join(raiz, 'bueno');
  const malo = path.join(raiz, 'malo');
  fs.writeFileSync(bueno, '#!/bin/sh\nexit 0\n', { mode: 0o755 });
  fs.writeFileSync(malo, '#!/bin/sh\nexit 3\n', { mode: 0o755 });
  assert.strictEqual(respondeBinario(bueno), true);
  assert.strictEqual(respondeBinario(malo), false);
  assert.strictEqual(respondeBinario(path.join(raiz, 'no-existe')), false);
});

test('moverBinario copia si el destino está en otro volumen', (t) => {
  const raiz = crearRaiz(t);
  const origen = path.join(raiz, 'origen');
  const destino = path.join(raiz, 'destino');
  fs.writeFileSync(origen, 'binario');
  const otroVolumen = () => { throw Object.assign(new Error('cross-device'), { code: 'EXDEV' }); };
  moverBinario(origen, destino, otroVolumen);
  assert.strictEqual(fs.readFileSync(destino, 'utf8'), 'binario');
  assert.ok(!fs.existsSync(origen));
  assert.ok(!fs.existsSync(destino + '.nuevo'));
});

test('moverBinario no esconde otro error', (t) => {
  const raiz = crearRaiz(t);
  const origen = path.join(raiz, 'origen');
  fs.writeFileSync(origen, 'binario');
  const ocupado = () => { throw Object.assign(new Error('busy'), { code: 'EBUSY' }); };
  assert.throws(() => moverBinario(origen, path.join(raiz, 'destino'), ocupado), /busy/);
  assert.ok(fs.existsSync(origen));
});

test('instalarCopia deja el binario indicado solo si responde', { skip: os.platform() === 'win32' }, (t) => {
  const raiz = crearRaiz(t);
  const bueno = path.join(raiz, 'bueno');
  fs.writeFileSync(bueno, '#!/bin/sh\nexit 0\n', { mode: 0o755 });
  const destino = path.join(raiz, 'qdd', 'bin', '9.0.0', 'qdd');
  assert.strictEqual(instalarCopia(bueno, destino), destino);
  assert.ok(fs.existsSync(destino));
  const malo = path.join(raiz, 'malo');
  fs.writeFileSync(malo, '#!/bin/sh\nexit 3\n', { mode: 0o755 });
  const otro = path.join(raiz, 'qdd', 'bin', '9.0.1', 'qdd');
  assert.throws(() => instalarCopia(malo, otro));
  assert.ok(!fs.existsSync(otro));
});

function carpetaConVersiones(t, ...versiones) {
  const carpeta = path.join(crearRaiz(t), 'bin');
  for (const v of versiones) {
    fs.mkdirSync(path.join(carpeta, v), { recursive: true });
    fs.writeFileSync(path.join(carpeta, v, 'qdd.exe'), 'exe');
  }
  return carpeta;
}

test('registrarVersion escribe el puntero y deja la versión nueva y la anterior', (t) => {
  const carpeta = carpetaConVersiones(t, '2.9.0', '2.9.1', '2.10.0-rc.1', '2.10.0', 'no-es-version');
  fs.writeFileSync(path.join(carpeta, 'actual.txt'), '2.10.0-rc.1');
  registrarVersion({ carpeta, version: '2.10.0' });
  assert.strictEqual(fs.readFileSync(path.join(carpeta, 'actual.txt'), 'utf8'), '2.10.0');
  assert.deepStrictEqual(fs.readdirSync(carpeta).sort(), ['2.10.0', '2.10.0-rc.1', 'actual.txt', 'no-es-version']);
});

test('registrarVersion no para por una versión que sigue corriendo', (t) => {
  const carpeta = carpetaConVersiones(t, '2.9.0', '2.9.1', '2.10.0');
  const borradas = [];
  const enUso = (ruta) => {
    borradas.push(path.basename(ruta));
    throw Object.assign(new Error('en uso'), { code: 'EBUSY' });
  };
  registrarVersion({ carpeta, version: '2.10.0', borrar: enUso });
  assert.deepStrictEqual(borradas.sort(), ['2.9.0', '2.9.1']);
  assert.strictEqual(fs.readFileSync(path.join(carpeta, 'actual.txt'), 'utf8'), '2.10.0');
});

test('reinstalar la misma versión no toca el puntero: nada tiene que reiniciarse', (t) => {
  const carpeta = carpetaConVersiones(t, '2.10.0');
  const puntero = path.join(carpeta, 'actual.txt');
  fs.writeFileSync(puntero, '2.10.0');
  const antes = new Date(Date.now() - 60000);
  fs.utimesSync(puntero, antes, antes);
  registrarVersion({ carpeta, version: '2.10.0' });
  assert.strictEqual(fs.statSync(puntero).mtimeMs, antes.getTime());
});

// FND-143: varios qdd a la vez sin binario (un panel, por ejemplo) no bajan cada uno el suyo.
const { tomarTurno } = require('../install.js');

test('tomarTurno: uno a la vez, y un turno abandonado no bloquea para siempre', (t) => {
  const turno = path.join(crearRaiz(t), '.bajando');
  assert.strictEqual(tomarTurno(turno), true);
  assert.strictEqual(tomarTurno(turno), false);
  const viejo = new Date(Date.now() - 10 * 60 * 1000);
  fs.utimesSync(turno, viejo, viejo);
  assert.strictEqual(tomarTurno(turno), true);
  assert.strictEqual(tomarTurno(turno), false);
});

test('tomarTurno no esconde otro error', (t) => {
  const turno = path.join(crearRaiz(t), 'no-existe', '.bajando');
  assert.throws(() => tomarTurno(turno), /ENOENT/);
});
