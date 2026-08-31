/**
 * QDD Data Backup Guard - State Isolation and Idempotency Guard
 * Strictly follows Zero-Else and Early Return.
 */

import fs from 'node:fs';
import path from 'node:path';

export class DataBackupGuard {
  constructor(options = {}) {
    this.backupDir = path.resolve(options.backupDir || '.qdd/cache/qa_backup');
    this.snapshots = new Map();
  }

  ensureBackupDir() {
    if (fs.existsSync(this.backupDir)) {
      return;
    }
    fs.mkdirSync(this.backupDir, { recursive: true });
  }

  captureFileSnapshot(key, targetFilePath) {
    const fullPath = path.resolve(targetFilePath);
    if (!fs.existsSync(fullPath)) {
      return false;
    }

    this.ensureBackupDir();
    const content = fs.readFileSync(fullPath);
    const backupFile = path.join(this.backupDir, `${key}_${Date.now()}.bak`);
    fs.writeFileSync(backupFile, content);

    this.snapshots.set(key, {
      originalPath: fullPath,
      backupFile: backupFile
    });
    return true;
  }

  restoreFileSnapshot(key) {
    const record = this.snapshots.get(key);
    if (!record) {
      return false;
    }

    if (!fs.existsSync(record.backupFile)) {
      return false;
    }

    const content = fs.readFileSync(record.backupFile);
    fs.writeFileSync(record.originalPath, content);
    return true;
  }

  cleanupAllSnapshots() {
    for (const [key, record] of this.snapshots.entries()) {
      if (fs.existsSync(record.backupFile)) {
        fs.unlinkSync(record.backupFile);
      }
    }
    this.snapshots.clear();
  }
}
