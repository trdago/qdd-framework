#!/usr/bin/env node

const { spawnSync } = require('child_process');
const path = require('path');
const fs = require('fs');

// rutaDelBinario: en Windows, qdd.exe vive fuera del paquete, en la carpeta de su versión (install.js, FND-142). Una
// instalación anterior a ese cambio, o sin LOCALAPPDATA, lo tiene dentro del paquete.
function rutaDelBinario({ platform, env, dir, version, existe }) {
  const nombre = platform === 'win32' ? 'qdd.exe' : 'qdd';
  const enElPaquete = path.join(dir, nombre);
  if (platform !== 'win32' || !env.LOCALAPPDATA || !version) {
    return enElPaquete;
  }
  const porVersion = path.join(env.LOCALAPPDATA, 'qdd', 'bin', version, nombre);
  return existe(porVersion) ? porVersion : enElPaquete;
}

function versionDelPaquete() {
  try {
    return require('../package.json').version;
  } catch (_) {
    return '';
  }
}

function main() {
  const binaryPath = rutaDelBinario({ platform: process.platform, env: process.env, dir: __dirname, version: versionDelPaquete(), existe: fs.existsSync });

  const result = spawnSync(binaryPath, process.argv.slice(2), {
    stdio: 'inherit'
  });

  // Sin binario, lo más probable es que npm esté reinstalando qdd: 75 es «todavía no» para los pedidos del panel (FND-135).
  if (result.error && result.error.code === 'ENOENT') {
    console.error('[QDD] qdd se está actualizando: inténtalo de nuevo en un momento.');
    console.error('[QDD] Si sigue así, reinstala el paquete con: npm install -g qdd-framework');
    process.exit(75);
  }

  if (result.error) {
    console.error(`[QDD] No se pudo ejecutar el binario nativo ${binaryPath}: ${result.error.message}`);
    console.error('[QDD] Reinstala el paquete con: npm install -g qdd-framework');
    process.exit(1);
  }

  process.exit(result.status === null ? 1 : result.status);
}

module.exports = { rutaDelBinario };

if (require.main === module) {
  main();
}
