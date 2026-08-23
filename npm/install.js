const fs = require('fs');
const path = require('path');
const https = require('https');
const { execSync } = require('child_process');

// Configuración del Release (Automático según tu GitHub)
const pkg = require('./package.json');
const VERSION = 'v' + pkg.version;
const REPO = 'trdago/qdd-framework';

const platformMap = {
  win32: 'windows',
  darwin: 'darwin',
  linux: 'linux'
};

const archMap = {
  x64: 'amd64',
  arm64: 'arm64'
};

const os = platformMap[process.platform];
const arch = archMap[process.arch];

if (!os || !arch) {
  console.error(`[QDD] Plataforma no soportada: ${process.platform} ${process.arch}`);
  process.exit(1);
}

const ext = os === 'windows' ? 'zip' : 'tar.gz';
const binaryExt = os === 'windows' ? '.exe' : '';
const binaryName = `qdd${binaryExt}`;
const binDir = path.join(__dirname, 'bin');

if (!fs.existsSync(binDir)) {
  fs.mkdirSync(binDir, { recursive: true });
}

// 1. Detección de binario local para desarrollo
const localBinRoot = path.join(__dirname, '..', binaryName);
const localBinCli = path.join(__dirname, '..', 'cli', binaryName);

if (fs.existsSync(localBinRoot)) {
  console.log(`[QDD] Binario local detectado en ${localBinRoot}. Usando copia local para desarrollo...`);
  fs.copyFileSync(localBinRoot, path.join(binDir, binaryName));
  fs.chmodSync(path.join(binDir, binaryName), 0o755);
  console.log('[QDD] ¡Instalación completada desde binario local!');
  process.exit(0);
}

if (fs.existsSync(localBinCli)) {
  console.log(`[QDD] Binario local detectado en ${localBinCli}. Usando copia local para desarrollo...`);
  fs.copyFileSync(localBinCli, path.join(binDir, binaryName));
  fs.chmodSync(path.join(binDir, binaryName), 0o755);
  console.log('[QDD] ¡Instalación completada desde binario local!');
  process.exit(0);
}

// 2. Descarga desde GitHub Releases
const url = `https://github.com/${REPO}/releases/download/${VERSION}/qdd_${os}_${arch}.${ext}`;
const downloadDest = path.join(__dirname, `qdd-download.${ext}`);

console.log(`[QDD] Descargando binario nativo para ${os}-${arch}...`);
console.log(`[QDD] URL: ${url}`);

downloadBinary(url);

function downloadBinary(targetUrl) {
  const file = fs.createWriteStream(downloadDest);
  https.get(targetUrl, (response) => {
    if (response.statusCode === 301 || response.statusCode === 302) {
      if (response.headers.location) {
        return downloadBinary(response.headers.location);
      }
    }

    if (response.statusCode !== 200) {
      console.error(`[QDD] Error: GitHub Release devolvió HTTP ${response.statusCode} para ${targetUrl}`);
      console.error('[QDD] Si estás instalando desde una versión no publicada aún, compila localmente con "make build"');
      process.exit(0); // Permitir que la instalación de npm continúe sin romper builds de CI
    }

    response.pipe(file);
    file.on('finish', () => {
      file.close(extractBinary);
    });
  }).on('error', (err) => {
    console.error('[QDD] Error de red descargando el binario:', err.message);
    process.exit(0);
  });
}

function extractBinary() {
  console.log('[QDD] Extrayendo...');
  try {
    if (os === 'windows') {
      execSync(`tar -xf "${downloadDest}" -C "${__dirname}"`);
    }
    if (os !== 'windows') {
      execSync(`tar -xzf "${downloadDest}" -C "${__dirname}"`);
    }
    const extractedBin = path.join(__dirname, binaryName);
    if (fs.existsSync(extractedBin)) {
      fs.renameSync(extractedBin, path.join(binDir, binaryName));
      fs.chmodSync(path.join(binDir, binaryName), 0o755);
    }
    if (fs.existsSync(downloadDest)) {
      fs.unlinkSync(downloadDest);
    }
    console.log('[QDD] ¡Instalación nativa completada exitosamente!');
  } catch (err) {
    console.error('[QDD] Error al extraer el binario:', err.message);
  }
}
