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

// 1. Detección de binario local para desarrollo (solo desde el repo, nunca dentro de node_modules)
// En una instalación de npm, `..` es la carpeta node_modules: `../qdd` puede ser la carpeta de otro
// paquete llamado `qdd` (p. ej. bengl/qdd), y copiar una carpeta falla con EISDIR y aborta la instalación.
function esArchivo(ruta) {
  try {
    return fs.statSync(ruta).isFile();
  } catch (_) {
    return false;
  }
}

function estaDentroDeNodeModules(dir) {
  return path.resolve(dir).split(path.sep).includes('node_modules');
}

function resolverBinarioLocal(pkgDir, nombreBinario, log = console.log) {
  if (estaDentroDeNodeModules(pkgDir)) {
    return null;
  }
  const candidatos = [
    path.join(pkgDir, '..', nombreBinario),
    path.join(pkgDir, '..', 'cli', nombreBinario)
  ];
  for (const candidato of candidatos) {
    if (esArchivo(candidato)) {
      return candidato;
    }
    if (fs.existsSync(candidato)) {
      log(`[QDD] Se ignora ${candidato}: existe pero no es un archivo binario.`);
    }
  }
  return null;
}

function main() {
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

  const binarioLocal = resolverBinarioLocal(__dirname, binaryName);
  if (binarioLocal) {
    console.log(`[QDD] Binario local detectado en ${binarioLocal}. Usando copia local para desarrollo...`);
    fs.copyFileSync(binarioLocal, path.join(binDir, binaryName));
    fs.chmodSync(path.join(binDir, binaryName), 0o755);
    console.log('[QDD] ¡Instalación completada desde binario local!');
    process.exit(0);
  }

  descargarDesdeRelease({ os, arch, ext, binaryName, binDir });
}

// 2. Descarga desde GitHub Releases
function descargarDesdeRelease({ os, arch, ext, binaryName, binDir }) {
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
      actualizarConsola(path.join(binDir, binaryName));
    } catch (err) {
      console.error('[QDD] Error al extraer el binario:', err.message);
    }
  }
}

// actualizarConsola deja el panel de Claude Code en la versión recién instalada si el usuario lo activó
// ('qdd consola activar', RFC 0006 §3.6). Sin activar no hace nada; un error no rompe la instalación.
function actualizarConsola(bin) {
  try {
    execSync(`"${bin}" consola actualizar --si-activo`, { stdio: 'inherit' });
  } catch (err) {
    console.error('[QDD] No se pudo actualizar el panel de Claude Code:', err.message);
  }
}

module.exports = { esArchivo, estaDentroDeNodeModules, resolverBinarioLocal };

if (require.main === module) {
  main();
}
