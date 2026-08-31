#!/usr/bin/env node

/**
 * QDD Antigravity Tools CLI Runner
 * Strictly follows Zero-Else and Early Return.
 */

import { AntigravityPingPongBridge } from './ping_pong_bridge.mjs';

async function main() {
  const prompt = process.argv.slice(2).join(' ') || 'Audita la gobernanza de arquitectura del proyecto.';

  console.log('⚡ [QDD Antigravity Tools] Starting Ping-Pong Handshake...');
  const bridge = new AntigravityPingPongBridge();

  bridge.on('token', (tok) => process.stdout.write(tok));

  const pong = await bridge.ping(prompt);
  console.log(`\n\n[PONG ACK] Status: ${pong.status} | Duration: ${pong.durationMs}ms`);
}

main().catch((err) => {
  console.error(`💥 Fatal error: ${err.message}`);
  process.exit(1);
});
