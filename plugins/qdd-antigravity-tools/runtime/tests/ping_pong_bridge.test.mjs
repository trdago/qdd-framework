import test from 'node:test';
import assert from 'node:assert/strict';
import { AntigravityPingPongBridge } from '../node/ping_pong_bridge.mjs';

test('AntigravityPingPongBridge completes handshake and emits events', async () => {
  const bridge = new AntigravityPingPongBridge({ systemInstructions: 'Test instructions' });
  const receivedTokens = [];
  let pingStartedEvent = null;

  bridge.on('ping_started', (ev) => {
    pingStartedEvent = ev;
  });

  bridge.on('token', (tok) => {
    receivedTokens.push(tok);
  });

  const pong = await bridge.ping('Audit codebase', { id: 'sess_test_node_1' });

  assert.ok(pingStartedEvent, 'ping_started event should be fired');
  assert.equal(pingStartedEvent.sessionId, 'sess_test_node_1');
  assert.equal(pong.status, 'SUCCESS');
  assert.equal(pong.turnId, 1);
  assert.ok(receivedTokens.length > 0, 'Tokens should be emitted through event listener');
  assert.ok(pong.content.includes('Antigravity Node Bridge Active'));
});

test('AntigravityPingPongBridge tool registration validation', () => {
  const bridge = new AntigravityPingPongBridge();

  bridge.registerTool('custom_audit', 'Runs audit', {}, () => ({ result: 'OK' }));
  assert.equal(bridge.tools.size, 1);

  assert.throws(() => {
    bridge.registerTool('', 'Invalid tool', {}, null);
  }, /Tool must have a valid name and handler function/);
});
