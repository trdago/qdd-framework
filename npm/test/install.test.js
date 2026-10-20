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
