import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import { DataBackupGuard } from '../core/data_backup_guard.mjs';

test('DataBackupGuard captures, modifies, and restores file state correctly', () => {
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'qdd_guard_test_'));
  const testFile = path.join(tmpDir, 'state.json');
  fs.writeFileSync(testFile, JSON.stringify({ version: 1, state: 'initial' }));

  const guard = new DataBackupGuard({ backupDir: path.join(tmpDir, 'backups') });

  const captured = guard.captureFileSnapshot('test_state', testFile);
  assert.ok(captured, 'Snapshot should be captured successfully');

  // Modify file
  fs.writeFileSync(testFile, JSON.stringify({ version: 2, state: 'corrupted' }));
  const corruptedContent = JSON.parse(fs.readFileSync(testFile, 'utf8'));
  assert.equal(corruptedContent.state, 'corrupted');

  // Restore snapshot
  const restored = guard.restoreFileSnapshot('test_state');
  assert.ok(restored, 'Snapshot should be restored successfully');

  const restoredContent = JSON.parse(fs.readFileSync(testFile, 'utf8'));
  assert.equal(restoredContent.state, 'initial');
  assert.equal(restoredContent.version, 1);

  guard.cleanupAllSnapshots();
  fs.rmSync(tmpDir, { recursive: true, force: true });
});
