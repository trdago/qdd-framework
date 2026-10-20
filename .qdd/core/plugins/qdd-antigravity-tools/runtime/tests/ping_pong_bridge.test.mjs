import test from 'node:test';
import assert from 'node:assert/strict';
import { AntigravityPingPongBridge, extractJsonBlock } from '../node/ping_pong_bridge.mjs';

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

test('AntigravityPingPongBridge tool registration and execution', async () => {
  const bridge = new AntigravityPingPongBridge();

  bridge.registerTool('custom_audit', 'Runs audit', {}, (args) => ({ result: 'OK', target: args.target }));
  assert.equal(bridge.tools.size, 1);

  const res = await bridge.executeTool('custom_audit', { target: 'backend' });
  assert.equal(res.status, 'SUCCESS');
  assert.equal(res.result.target, 'backend');

  const errRes = await bridge.executeTool('non_existent');
  assert.equal(errRes.status, 'ERROR');

  assert.throws(() => {
    bridge.registerTool('', 'Invalid tool', {}, null);
  }, /Tool must have a valid name and handler function/);
});

test('extractJsonBlock parses delimited and markdown blocks', () => {
  const delimited = 'Thought\n[VERDICT_START]\n{"passed":true,"score":99}\n[VERDICT_END]\nEnd';
  const parsed1 = extractJsonBlock(delimited);
  assert.equal(parsed1.passed, true);
  assert.equal(parsed1.score, 99);

  const md = '```json\n{"status":"OK"}\n```';
  const parsed2 = extractJsonBlock(md);
  assert.equal(parsed2.status, 'OK');

  const empty = extractJsonBlock('');
  assert.deepEqual(empty, {});
});
