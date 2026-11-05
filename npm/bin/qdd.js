#!/usr/bin/env node

const { spawnSync } = require('child_process');
const path = require('path');
const os = require('os');
const fs = require('fs');

const binaryExt = os.platform() === 'win32' ? '.exe' : '';
const binaryPath = path.join(__dirname, `qdd${binaryExt}`);
const instalador = path.join(__dirname, '..', 'install.js');

// Con npm 12, quien instala sin --allow-scripts=qdd-framework no corrió el postinstall (FND-143): el primer qdd baja el
// binario con el mismo instalador. Lo que escribe va a stderr: otro programa puede estar leyendo la salida de qdd.
function bajarAlPrimerUso() {
  if (fs.existsSync(binaryPath) || !fs.existsSync(instalador)) {
    return;
  }
  console.error('[QDD] qdd no tiene su binario: lo bajo ahora, porque npm no corrió su instalación.');
  spawnSync(process.execPath, [instalador, '--primer-uso'], { stdio: ['ignore', 2, 'inherit'] });
}

bajarAlPrimerUso();

const result = spawnSync(binaryPath, process.argv.slice(2), {
  stdio: 'inherit'
});

// Sin binario, lo más probable es que npm esté reinstalando qdd: 75 es «todavía no» para los pedidos del panel (FND-135).
if (result.error && result.error.code === 'ENOENT') {
  console.error('[QDD] qdd no tiene su binario: se está actualizando, o npm bloqueó su instalación (npm 12 lo hace por defecto).');
  console.error('[QDD] Si sigue así, reinstala el paquete con: npm install -g qdd-framework --allow-scripts=qdd-framework');
  process.exit(75);
}

if (result.error) {
  console.error(`[QDD] No se pudo ejecutar el binario nativo ${binaryPath}: ${result.error.message}`);
  console.error('[QDD] Reinstala el paquete con: npm install -g qdd-framework --allow-scripts=qdd-framework');
  process.exit(1);
}

process.exit(result.status === null ? 1 : result.status);
