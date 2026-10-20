#!/usr/bin/env node
/**
 * Ejemplo Completo de Activación Ping-Pong en Node.js
 * Ejecución: node examples/node_ping_pong_example.mjs
 */

import { AntigravityPingPongBridge } from '../runtime/node/ping_pong_bridge.mjs';

async function main() {
  console.log('==================================================================');
  console.log('🤖 QDD Antigravity Tools - Node.js Ping-Pong Bridge');
  console.log('==================================================================\n');

  const bridge = new AntigravityPingPongBridge({
    systemInstructions: 'Eres un agente de arquitectura QDD.'
  });

  // Escuchar eventos reactivos
  bridge.on('ping_started', (ev) => {
    console.log(`[+] PING iniciado en sesión '${ev.sessionId}'`);
  });

  bridge.on('token', (tok) => {
    process.stdout.write(tok);
  });

  // Enviar PING
  const pong = await bridge.ping('Valida la integridad de los contratos MCP.', {
    id: 'node_sess_demo_01'
  });

  console.log(`\n\n[🏓 PONG RECIBIDO]`);
  console.log(` - Status:   ${pong.status}`);
  console.log(` - Turno:    ${pong.turnId}`);
  console.log(` - Duración: ${pong.durationMs.toFixed(2)}ms`);

  console.log('\n==================================================================');
  console.log('✅ Handshake reactivo completado con éxito.');
  console.log('==================================================================');
}

main().catch(console.error);
