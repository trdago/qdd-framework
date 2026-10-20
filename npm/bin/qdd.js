#!/usr/bin/env node

const { spawnSync } = require('child_process');
const path = require('path');
const os = require('os');

const binaryExt = os.platform() === 'win32' ? '.exe' : '';
const binaryPath = path.join(__dirname, `qdd${binaryExt}`);

const result = spawnSync(binaryPath, process.argv.slice(2), {
  stdio: 'inherit'
});

if (result.error) {
  console.error(`[QDD] No se pudo ejecutar el binario nativo ${binaryPath}: ${result.error.message}`);
  console.error('[QDD] Reinstala el paquete con: npm install -g qdd-framework');
  process.exit(1);
}

process.exit(result.status === null ? 1 : result.status);
