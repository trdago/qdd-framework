const fs = require('fs');
const path = require('path');
const https = require('https');
const { execSync } = require('child_process');

// Configuración del Release (Automático según tu GitHub)
const VERSION = 'v0.1.1';
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
  console.error(`Plataforma no soportada: ${process.platform} ${process.arch}`);
  process.exit(1);
}

const ext = os === 'windows' ? 'zip' : 'tar.gz';
const binaryExt = os === 'windows' ? '.exe' : '';
const binaryName = `qdd${binaryExt}`;
const url = `https://github.com/${REPO}/releases/download/${VERSION}/qdd_${os}_${arch}.${ext}`;

const binDir = path.join(__dirname, 'bin');
const downloadDest = path.join(__dirname, `qdd-download.${ext}`);

console.log(`[QDD] Descargando binario nativo para ${os}-${arch}...`);
console.log(`[QDD] URL: ${url}`);

// (Para efectos de desarrollo y prueba, comentamos la descarga real 
// hasta que el repo de GitHub tenga sus primeros Releases configurados con GoReleaser)
/*
const file = fs.createWriteStream(downloadDest);
https.get(url, (response) => {
  if (response.statusCode === 301 || response.statusCode === 302) {
    return https.get(response.headers.location, (res) => {
      res.pipe(file);
      file.on('finish', extractBinary);
    });
  }
  response.pipe(file);
  file.on('finish', extractBinary);
}).on('error', (err) => {
  console.error('[QDD] Error descargando el binario:', err);
  process.exit(1);
});

function extractBinary() {
  console.log('[QDD] Extrayendo...');
  try {
    if (os === 'windows') {
      execSync(`tar -xf ${downloadDest} -C ${__dirname}`);
    } else {
      execSync(`tar -xzf ${downloadDest} -C ${__dirname}`);
    }
    fs.renameSync(path.join(__dirname, binaryName), path.join(binDir, binaryName));
    fs.chmodSync(path.join(binDir, binaryName), 0o755);
    fs.unlinkSync(downloadDest);
    console.log('[QDD] ¡Instalación nativa completada exitosamente!');
  } catch (err) {
    console.error('[QDD] Error al extraer el binario:', err);
  }
}
*/
console.log('[QDD] NOTA: Script preparado para producción. Descomenta la descarga cuando GoReleaser publique los binarios en GitHub Releases.');
