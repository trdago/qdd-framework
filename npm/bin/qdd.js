#!/usr/bin/env node

const { spawnSync } = require('child_process');
const path = require('path');
const os = require('os');

const binaryExt = os.platform() === 'win32' ? '.exe' : '';
const binaryPath = path.join(__dirname, `qdd${binaryExt}`);

const result = spawnSync(binaryPath, process.argv.slice(2), {
  stdio: 'inherit'
});

process.exit(result.status || 0);
