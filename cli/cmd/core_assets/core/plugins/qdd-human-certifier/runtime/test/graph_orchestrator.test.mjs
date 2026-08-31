import test from 'node:test';
import assert from 'node:assert/strict';
import { GraphOrchestrator } from '../core/graph_orchestrator.mjs';

test('GraphOrchestrator resolves linear dependencies in topological order', () => {
  const orchestrator = new GraphOrchestrator();

  orchestrator.registerNodes([
    { id: 'auth:login', dependencies: [] },
    { id: 'dashboard:view', dependencies: ['auth:login'] },
    { id: 'dashboard:export', dependencies: ['dashboard:view'] }
  ]);

  const plan = orchestrator.resolveExecutionPlan('dashboard:export');
  const planIds = plan.map((n) => n.id);

  assert.deepEqual(planIds, ['auth:login', 'dashboard:view', 'dashboard:export']);
});

test('GraphOrchestrator resolves diamonds and branches without duplicates', () => {
  const orchestrator = new GraphOrchestrator();

  orchestrator.registerNodes([
    { id: 'root', dependencies: [] },
    { id: 'branchA', dependencies: ['root'] },
    { id: 'branchB', dependencies: ['root'] },
    { id: 'leaf', dependencies: ['branchA', 'branchB'] }
  ]);

  const plan = orchestrator.resolveExecutionPlan('leaf');
  const planIds = plan.map((n) => n.id);

  assert.equal(planIds[0], 'root');
  assert.equal(planIds[3], 'leaf');
  assert.equal(planIds.length, 4);
});

test('GraphOrchestrator detects and throws on circular dependencies', () => {
  const orchestrator = new GraphOrchestrator();

  orchestrator.registerNodes([
    { id: 'nodeA', dependencies: ['nodeB'] },
    { id: 'nodeB', dependencies: ['nodeA'] }
  ]);

  assert.throws(() => {
    orchestrator.resolveExecutionPlan('nodeA');
  }, /Circular dependency detected/);
});

test('GraphOrchestrator resolves by category filter', () => {
  const orchestrator = new GraphOrchestrator();

  orchestrator.registerNodes([
    { id: 'auth:root', category: 'auth', dependencies: [] },
    { id: 'usability:clicks', category: 'usability', dependencies: ['auth:root'] },
    { id: 'usability:tta', category: 'usability', dependencies: ['auth:root'] }
  ]);

  const plan = orchestrator.resolveExecutionPlan('usability');
  const planIds = plan.map((n) => n.id);

  assert.equal(planIds[0], 'auth:root');
  assert.ok(planIds.includes('usability:clicks'));
  assert.ok(planIds.includes('usability:tta'));
});
