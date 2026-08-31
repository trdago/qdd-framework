#!/usr/bin/env node

/**
 * QDD Composable Human QA & Usability Graph Engine
 * Master CLI Runner
 * Strictly follows Zero-Else and Early Return.
 */

import path from 'node:path';
import fs from 'node:fs';
import { GraphOrchestrator } from './core/graph_orchestrator.mjs';
import { LocalFrontendServer } from './core/local_frontend_server.mjs';
import { BrowserHarness } from './core/browser_context.mjs';

const ANSI = {
  reset: '\x1b[0m',
  bold: '\x1b[1m',
  green: '\x1b[32m',
  red: '\x1b[31m',
  yellow: '\x1b[33m',
  cyan: '\x1b[36m',
  dim: '\x1b[2m'
};

async function loadConfig() {
  const configPath = path.resolve('.qdd/human_qa.config.json');
  if (fs.existsSync(configPath)) {
    try {
      const raw = fs.readFileSync(configPath, 'utf8');
      return JSON.parse(raw);
    } catch (e) {
      console.warn(`${ANSI.yellow}⚠️ Warning: failed to parse ${configPath}, using defaults${ANSI.reset}`);
    }
  }

  // Auto-detect dist path
  const candidates = ['frontend/dist', 'dist', 'build', 'out', 'public'];
  let detectedDist = 'dist';
  for (const c of candidates) {
    if (fs.existsSync(path.resolve(c))) {
      detectedDist = c;
      break;
    }
  }

  return {
    distPath: detectedDist,
    port: 5173,
    baseUrl: 'http://localhost:5173',
    unitsDir: 'scripts/human_qa/units',
    headless: true
  };
}

async function discoverUnits(unitsDir) {
  const units = [];
  const fullDir = path.resolve(unitsDir);

  if (!fs.existsSync(fullDir)) {
    return units;
  }

  function scanDir(dir) {
    const entries = fs.readdirSync(dir, { withFileTypes: true });
    for (const ent of entries) {
      const entPath = path.join(dir, ent.name);
      if (ent.isDirectory()) {
        scanDir(entPath);
        continue;
      }
      if (ent.name.endsWith('.mjs') || ent.name.endsWith('.js')) {
        units.push(entPath);
      }
    }
  }

  scanDir(fullDir);
  return units;
}

async function main() {
  const args = process.argv.slice(2);
  const targetPattern = args[0] || 'all';

  console.log(`\n${ANSI.cyan}${ANSI.bold}╔══════════════════════════════════════════════════════════════════╗${ANSI.reset}`);
  console.log(`${ANSI.cyan}${ANSI.bold}║    🎭 QDD Composable Human QA & Usability Graph Certifier        ║${ANSI.reset}`);
  console.log(`${ANSI.cyan}${ANSI.bold}╚══════════════════════════════════════════════════════════════════╝${ANSI.reset}\n`);

  const config = await loadConfig();
  const orchestrator = new GraphOrchestrator();

  const unitFiles = await discoverUnits(config.unitsDir);
  if (unitFiles.length === 0) {
    console.log(`${ANSI.yellow}ℹ️ No custom units found in '${config.unitsDir}'.${ANSI.reset}`);
    console.log(`${ANSI.dim}To add custom units, place them in '${config.unitsDir}/<category>/<unit_name>.mjs'.${ANSI.reset}\n`);
    process.exit(0);
  }

  for (const file of unitFiles) {
    try {
      const module = await import(file);
      const unit = module.default || module;
      if (unit && unit.id) {
        orchestrator.registerNode(unit);
      }
    } catch (err) {
      console.error(`${ANSI.red}❌ Error loading unit from ${file}: ${err.message}${ANSI.reset}`);
    }
  }

  let executionPlan;
  try {
    executionPlan = orchestrator.resolveExecutionPlan(targetPattern);
  } catch (err) {
    console.error(`\n${ANSI.red}❌ DAG Resolution Error: ${err.message}${ANSI.reset}\n`);
    process.exit(1);
  }

  console.log(`${ANSI.bold}🎯 Target Pattern:${ANSI.reset} ${targetPattern}`);
  console.log(`${ANSI.bold}📊 Execution Plan (${executionPlan.length} units):${ANSI.reset}`);
  executionPlan.forEach((node, idx) => {
    const deps = node.dependencies && node.dependencies.length > 0 ? ` [deps: ${node.dependencies.join(', ')}]` : '';
    console.log(`  ${ANSI.dim}${idx + 1}.${ANSI.reset} ${ANSI.cyan}${node.id}${ANSI.reset}${ANSI.dim}${deps}${ANSI.reset}`);
  });
  console.log('');

  // Start local server if dist exists
  let server = null;
  if (fs.existsSync(path.resolve(config.distPath))) {
    try {
      server = new LocalFrontendServer({ distPath: config.distPath, port: config.port });
      const info = await server.start();
      console.log(`${ANSI.green}⚡ Local Frontend Server running at ${info.url} (${config.distPath})${ANSI.reset}\n`);
    } catch (e) {
      console.warn(`${ANSI.yellow}⚠️ Could not start local frontend server: ${e.message}${ANSI.reset}`);
    }
  }

  let chromium;
  try {
    const pw = await import('playwright');
    chromium = pw.chromium;
  } catch (e) {
    console.warn(`${ANSI.yellow}⚠️ Playwright not installed. Running in dry-run DAG verification mode.${ANSI.reset}\n`);
  }

  let harness = null;
  if (chromium) {
    harness = new BrowserHarness({ baseUrl: config.baseUrl, headless: config.headless });
    await harness.initialize(chromium);
  }

  let passed = 0;
  let failed = 0;
  const startTime = Date.now();

  for (const node of executionPlan) {
    process.stdout.write(`  ⏳ Running ${ANSI.bold}${node.id}${ANSI.reset}... `);
    const nodeStart = Date.now();

    try {
      if (typeof node.run === 'function') {
        await node.run({
          page: harness ? harness.desktopPage : null,
          mobilePage: harness ? harness.mobilePage : null,
          harness,
          config
        });
      }
      const duration = Date.now() - nodeStart;
      console.log(`${ANSI.green}✅ PASS${ANSI.reset} ${ANSI.dim}(${duration}ms)${ANSI.reset}`);
      passed++;
    } catch (err) {
      const duration = Date.now() - nodeStart;
      console.log(`${ANSI.red}❌ FAIL${ANSI.reset} ${ANSI.dim}(${duration}ms)${ANSI.reset}`);
      console.error(`     ${ANSI.red}${err.message}${ANSI.reset}`);
      failed++;
      break;
    }
  }

  if (harness) {
    await harness.close();
  }
  if (server) {
    await server.stop();
  }

  const totalTime = ((Date.now() - startTime) / 1000).toFixed(2);
  console.log(`\n${ANSI.bold}══════════════════════════════════════════════════════════════════${ANSI.reset}`);
  if (failed > 0) {
    console.log(`${ANSI.red}${ANSI.bold}🔴 CERTIFICATION FAILED:${ANSI.reset} ${passed} passed, ${failed} failed (${totalTime}s)\n`);
    process.exit(1);
  }

  console.log(`${ANSI.green}${ANSI.bold}🏆 100% HUMAN CERTIFICATION PASSED:${ANSI.reset} ${passed} units verified (${totalTime}s)\n`);
  process.exit(0);
}

main().catch((err) => {
  console.error(`\n${ANSI.red}💥 Unexpected Fatal Error: ${err.message}${ANSI.reset}\n`);
  process.exit(1);
});
