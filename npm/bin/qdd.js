#!/usr/bin/env node

const { spawnSync } = require('child_process');
const path = require('path');
const os = require('os');

const binaryExt = os.platform() === 'win32' ? '.exe' : '';
const binaryPath = path.join(__dirname, `qdd${binaryExt}`);

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
