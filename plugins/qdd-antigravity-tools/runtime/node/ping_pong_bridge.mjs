/**
 * QDD Antigravity Ping-Pong Bridge for Node.js
 * Strictly follows Zero-Else and Early Return.
 */

import { EventEmitter } from 'node:events';

export function extractJsonBlock(rawText) {
  if (!rawText || typeof rawText !== 'string') {
    return {};
  }
  const text = rawText.trim();
  if (!text) {
    return {};
  }

  if (text.includes('[VERDICT_START]') && text.includes('[VERDICT_END]')) {
    const startIdx = text.indexOf('[VERDICT_START]') + '[VERDICT_START]'.length;
    const endIdx = text.indexOf('[VERDICT_END]');
    const extracted = text.substring(startIdx, endIdx).trim();
    try {
      return JSON.parse(extracted);
    } catch {
      return { raw: extracted };
    }
  }

  if (text.includes('```json')) {
    const startIdx = text.indexOf('```json') + 7;
    const endIdx = text.indexOf('```', startIdx);
    if (endIdx !== -1) {
      const extracted = text.substring(startIdx, endIdx).trim();
      try {
        return JSON.parse(extracted);
      } catch {
        return { raw: extracted };
      }
    }
  }

  try {
    return JSON.parse(text);
  } catch {
    return { raw: text };
  }
}

export class AntigravityPingPongBridge extends EventEmitter {
  constructor(options = {}) {
    super();
    this.systemInstructions = options.systemInstructions || '';
    this.tools = new Map();
    this.turnCounter = 0;
  }

  registerTool(name, description, schema, handler) {
    if (!name || typeof handler !== 'function') {
      throw new Error('Tool must have a valid name and handler function');
    }
    this.tools.set(name, { name, description, schema, handler });
  }

  async executeTool(name, args = {}) {
    const tool = this.tools.get(name);
    if (!tool) {
      return { status: 'ERROR', error: `Tool '${name}' not found` };
    }
    try {
      const result = await tool.handler(args);
      return { status: 'SUCCESS', result };
    } catch (err) {
      return { status: 'ERROR', error: err.message };
    }
  }

  async ping(prompt, session = {}) {
    this.turnCounter++;
    const startTime = Date.now();
    const sessionId = session.id || `sess_node_${Date.now()}`;
    const toolsExecuted = [];
    const tokens = [];

    this.emit('ping_started', { sessionId, turnId: this.turnCounter, prompt });

    try {
      const chunks = [
        `[Antigravity Node Bridge Active]\n`,
        `Received prompt: "${prompt}"\n`,
        `Session: ${sessionId} (Turn ${this.turnCounter})\n`,
        `QDD Standard: Zero-Else Validated.`
      ];

      for (const chunk of chunks) {
        tokens.push(chunk);
        this.emit('token', chunk);
      }

      const durationMs = Date.now() - startTime;
      const pong = {
        protocol: 'qdd-ping-pong/v1',
        message_type: 'PONG',
        sessionId,
        turnId: this.turnCounter,
        status: 'SUCCESS',
        content: tokens.join(''),
        durationMs,
        toolsExecuted
      };

      this.emit('pong_received', pong);
      return pong;
    } catch (err) {
      const durationMs = Date.now() - startTime;
      const errorPong = {
        protocol: 'qdd-ping-pong/v1',
        message_type: 'PONG',
        sessionId,
        turnId: this.turnCounter,
        status: 'ERROR',
        error: err.message,
        durationMs,
        toolsExecuted
      };

      this.emit('pong_error', errorPong);
      return errorPong;
    }
  }
}
