/**
 * QDD Zero-Latency Local Frontend Server (<10ms)
 * Serves SPA distribution directories without external server dependencies.
 * Strictly follows Zero-Else and Early Return.
 */

import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';

const MIME_TYPES = {
  '.html': 'text/html; charset=utf-8',
  '.js': 'application/javascript; charset=utf-8',
  '.mjs': 'application/javascript; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.svg': 'image/svg+xml',
  '.ico': 'image/x-icon',
  '.woff2': 'font/woff2',
  '.ttf': 'font/ttf'
};

export class LocalFrontendServer {
  constructor(options = {}) {
    this.distPath = path.resolve(options.distPath || 'dist');
    this.port = options.port || 5173;
    this.server = null;
  }

  async start() {
    if (!fs.existsSync(this.distPath)) {
      throw new Error(`Frontend distribution directory not found: ${this.distPath}. Please build the project first.`);
    }

    return new Promise((resolve, reject) => {
      this.server = http.createServer((req, res) => {
        this.handleRequest(req, res);
      });

      this.server.listen(this.port, () => {
        resolve({
          url: `http://localhost:${this.port}`,
          port: this.port,
          distPath: this.distPath
        });
      });

      this.server.on('error', (err) => {
        reject(err);
      });
    });
  }

  handleRequest(req, res) {
    const parsedUrl = new URL(req.url, `http://localhost:${this.port}`);
    let filePath = path.join(this.distPath, parsedUrl.pathname);

    // If directory, check index.html
    if (fs.existsSync(filePath) && fs.statSync(filePath).isDirectory()) {
      filePath = path.join(filePath, 'index.html');
    }

    // Direct file hit
    if (fs.existsSync(filePath) && fs.statSync(filePath).isFile()) {
      const ext = path.extname(filePath).toLowerCase();
      const contentType = MIME_TYPES[ext] || 'application/octet-stream';
      res.writeHead(200, { 'Content-Type': contentType });
      fs.createReadStream(filePath).pipe(res);
      return;
    }

    // SPA Fallback: serve root index.html for virtual routes
    const spaFallbackPath = path.join(this.distPath, 'index.html');
    if (fs.existsSync(spaFallbackPath)) {
      res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
      fs.createReadStream(spaFallbackPath).pipe(res);
      return;
    }

    // Not found
    res.writeHead(404, { 'Content-Type': 'text/plain' });
    res.end('404 Not Found');
  }

  async stop() {
    if (!this.server) {
      return;
    }
    return new Promise((resolve) => {
      this.server.close(() => {
        this.server = null;
        resolve();
      });
    });
  }
}
