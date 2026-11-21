const fs = require('fs');
const path = require('path');
const https = require('https');
const { execSync, execFileSync } = require('child_process');

// Configuración del Release (Automático según tu GitHub)
const pkg = require('./package.json');
const VERSION = 'v' + pkg.version;
const REPO = 'trdago/qdd-framework';
// TOPE_VERSION_MS: el antivirus puede demorar la primera ejecución de un binario recién descargado.
const TOPE_VERSION_MS = 60000;
// TOPE_TURNO_MS: un turno más viejo que esto quedó abandonado; bajar el binario tarda mucho menos.
const TOPE_TURNO_MS = 5 * 60 * 1000;

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

  const carpeta = carpetaDeVersiones({ os, env: process.env });
  const destino = destinoDelBinario({ carpeta, version: pkg.version, binDir, binaryName });

  if (!fs.existsSync(binDir)) {
    fs.mkdirSync(binDir, { recursive: true });
  }

  if (process.argv.includes('--primer-uso')) {
    pedirTurno(path.join(binDir, '.bajando'));
  }

  const binarioLocal = resolverBinarioLocal(__dirname, binaryName);
  if (binarioLocal) {
    console.log(`[QDD] Binario local detectado en ${binarioLocal}. Usando copia local para desarrollo...`);
    fs.copyFileSync(binarioLocal, path.join(binDir, binaryName));
    fs.chmodSync(path.join(binDir, binaryName), 0o755);
    console.log('[QDD] ¡Instalación completada desde binario local!');
    process.exit(0);
  }

  // Reinstalar la misma versión: su binario puede estar corriendo y no hace falta tocarlo.
  if (carpeta && respondeBinario(destino)) {
    console.log(`[QDD] La versión ${pkg.version} ya está en ${destino}.`);
    terminarInstalacion({ bin: destino, carpeta });
    return;
  }

  if (process.env.QDD_INSTALL_BINARIO) {
    instalarBinarioIndicado({ origen: process.env.QDD_INSTALL_BINARIO, destino, carpeta });
    return;
  }

  descargarDesdeRelease({ os, arch, ext, binaryName, binDir, destino, carpeta });
}

// QDD_INSTALL_BINARIO instala ese binario en vez de descargarlo: es como el CI prueba el paquete antes de publicarlo.
function instalarBinarioIndicado({ origen, destino, carpeta }) {
  try {
    terminarInstalacion({ bin: instalarCopia(origen, destino), carpeta });
  } catch (err) {
    fallar(`no se pudo instalar ${origen}: ${err.message}`);
  }
}

// tomarTurno: varios qdd sin binario a la vez (un panel, por ejemplo) no bajan cada uno el suyo.
function tomarTurno(ruta) {
  try {
    fs.closeSync(fs.openSync(ruta, 'wx'));
    return true;
  } catch (err) {
    if (err.code !== 'EEXIST') {
      throw err;
    }
  }
  if (Date.now() - fs.statSync(ruta).mtimeMs < TOPE_TURNO_MS) {
    return false;
  }
  fs.rmSync(ruta, { force: true });
  return tomarTurno(ruta);
}

// pedirTurno sale con 75, el «todavía no» de los pedidos, si otro qdd ya lo está bajando.
function pedirTurno(ruta) {
  try {
    if (!tomarTurno(ruta)) {
      console.error('[QDD] Otro proceso ya está bajando el binario.');
      process.exit(75);
    }
  } catch (err) {
    fallar(`no se pudo escribir en la carpeta del paquete: ${err.message}`);
  }
  process.on('exit', () => borrarSinFallar(ruta));
}

// carpetaDeDescarga es propia de este proceso: un postinstall y un primer uso pueden correr a la vez.
function carpetaDeDescarga() {
  try {
    const carpeta = fs.mkdtempSync(path.join(__dirname, 'descarga-'));
    process.on('exit', () => borrarSinFallar(carpeta));
    return carpeta;
  } catch (err) {
    return fallar(`no se pudo escribir en la carpeta del paquete: ${err.message}`);
  }
}

// 2. Descarga desde GitHub Releases
function descargarDesdeRelease({ os, arch, ext, binaryName, binDir, destino, carpeta }) {
  const url = `https://github.com/${REPO}/releases/download/${VERSION}/qdd_${os}_${arch}.${ext}`;
  const descarga = carpetaDeDescarga();
  const downloadDest = path.join(descarga, `qdd-download.${ext}`);

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
        console.error('[QDD] Si estás instalando desde una versión no publicada aún, compila localmente con "make build"');
        fallar(`GitHub Release devolvió HTTP ${response.statusCode} para ${targetUrl}`);
      }

      response.pipe(file);
      file.on('finish', () => {
        file.close(extractBinary);
      });
    }).on('error', (err) => {
      fallar(`error de red descargando el binario: ${err.message}`);
    });
  }

  function extractBinary() {
    console.log('[QDD] Extrayendo...');
    try {
      const bin = instalarDescarga({ os, paquete: downloadDest, carpeta: descarga, binaryName, binDir, env: process.env, destino });
      console.log('[QDD] ¡Instalación nativa completada exitosamente!');
      terminarInstalacion({ bin, carpeta });
    } catch (err) {
      fallar(`no se pudo instalar el binario: ${err.message}`);
    }
  }
}

// comandoExtraer: en Windows, el tar del sistema con su ruta completa. En Git Bash, el primero del PATH es GNU tar, que no
// abre un zip y toma «C:» por un host remoto (MEJ-013).
function comandoExtraer(os, paquete, destino, env) {
  if (os === 'windows') {
    const programa = path.win32.join(env.SystemRoot || 'C:\\Windows', 'System32', 'tar.exe');
    return { programa, args: ['-xf', paquete, '-C', destino] };
  }
  return { programa: 'tar', args: ['-xzf', paquete, '-C', destino] };
}

// En Windows, npm no puede reemplazar un qdd.exe en uso, y siempre hay uno: el panel y los pedidos corren desde él (issue
// #107, FND-142). Por eso ahí el binario vive fuera del paquete, en %LOCALAPPDATA%\qdd\bin\<versión>\qdd.exe: lo que
// corre se queda con su versión y npm no se topa con nada. bin\actual.txt dice cuál es la instalada: lo que corre lo
// mira para saber que tiene que cerrarse. Linux y macOS pueden reemplazar un binario en uso: siguen con él en el paquete.
function carpetaDeVersiones({ os, env }) {
  if (os !== 'windows' || !env.LOCALAPPDATA) {
    return '';
  }
  return path.join(env.LOCALAPPDATA, 'qdd', 'bin');
}

function destinoDelBinario({ carpeta, version, binDir, binaryName }) {
  if (!carpeta) {
    return path.join(binDir, binaryName);
  }
  return path.join(carpeta, version, binaryName);
}

function respondeBinario(ruta) {
  try {
    execFileSync(ruta, ['--version'], { stdio: 'pipe', timeout: TOPE_VERSION_MS });
    return true;
  } catch (_) {
    return false;
  }
}

// En Windows, un antivirus puede retener unos segundos el qdd.exe que se acaba de ejecutar (#175).
const OCUPADO = new Set(['EBUSY', 'EPERM']);
const ESPERA_OCUPADO_MS = 1000;
const INTENTOS_OCUPADO = 30;

function dormirSincrono(ms) {
  Atomics.wait(new Int32Array(new SharedArrayBuffer(4)), 0, 0, ms);
}

// moverBinario copia si el destino está en otro volumen (el prefijo de npm y %LOCALAPPDATA% pueden estarlo) o si el
// binario sigue ocupado tras unos 30 s: copiar solo necesita leerlo.
function moverBinario(origen, destino, renombrar = fs.renameSync, { dormir = dormirSincrono, intentos = INTENTOS_OCUPADO } = {}) {
  for (let i = 0; ; i++) {
    try {
      renombrar(origen, destino);
      return;
    } catch (err) {
      if (err.code === 'EXDEV') {
        break;
      }
      if (!OCUPADO.has(err.code)) {
        throw err;
      }
      if (i + 1 >= intentos) {
        break;
      }
      dormir(ESPERA_OCUPADO_MS);
    }
  }
  copiarBinario(origen, destino);
  borrarSinFallar(origen);
}

// quitarSiVacia no deja una carpeta de versión sin binario cuando la instalación falla (#175).
function quitarSiVacia(carpeta) {
  try {
    fs.rmdirSync(carpeta);
  } catch (_) {
    // tiene archivos (otra instalación) o ya no está: se deja.
  }
}

// borrarSinFallar: que no se pueda borrar algo temporal no rompe la instalación ni tapa su mensaje (#175).
function borrarSinFallar(ruta) {
  try {
    fs.rmSync(ruta, { recursive: true, force: true, maxRetries: 5, retryDelay: 200 });
  } catch (err) {
    console.error(`[QDD] No se pudo borrar ${ruta}: ${err.message}`);
  }
}

// copiarBinario nunca deja un binario a medias con el nombre final.
function copiarBinario(origen, destino) {
  const temporal = destino + '.nuevo';
  fs.copyFileSync(origen, temporal);
  fs.renameSync(temporal, destino);
}

function instalarCopia(origen, destino) {
  execFileSync(origen, ['--version'], { stdio: 'pipe', timeout: TOPE_VERSION_MS });
  fs.mkdirSync(path.dirname(destino), { recursive: true });
  copiarBinario(origen, destino);
  return destino;
}

// registrarVersion avisa a lo que corre que hay otra versión y borra las viejas: quedan esta y la anterior. Lo que no se
// puede borrar sigue corriendo: se borrará en otra actualización.
function registrarVersion({ carpeta, version, borrar = fs.rmSync }) {
  const puntero = path.join(carpeta, 'actual.txt');
  const anterior = esArchivo(puntero) ? fs.readFileSync(puntero, 'utf8').trim() : '';
  if (anterior !== version) {
    fs.writeFileSync(puntero, version);
  }
  for (const nombre of fs.readdirSync(carpeta)) {
    if (!/^\d+\.\d+\.\d+/.test(nombre) || nombre === version || nombre === anterior) {
      continue;
    }
    try {
      borrar(path.join(carpeta, nombre), { recursive: true, force: true });
    } catch (_) {
      console.log(`[QDD] La versión ${nombre} sigue en uso: se borrará más adelante.`);
    }
  }
}

// terminarInstalacion: primero el panel, y al final el puntero, porque lo que corre se cierra apenas cambia.
function terminarInstalacion({ bin, carpeta }) {
  actualizarConsola(bin);
  if (!carpeta) {
    return;
  }
  try {
    registrarVersion({ carpeta, version: pkg.version });
  } catch (err) {
    console.error('[QDD] No se pudo registrar la versión instalada:', err.message);
  }
}

// instalarDescarga deja el binario en su destino solo si responde; si algo falla, lanza el error y el binario anterior sigue ahí.
function instalarDescarga({ os, paquete, carpeta, binaryName, binDir, env, destino = path.join(binDir, binaryName) }) {
  const { programa, args } = comandoExtraer(os, paquete, carpeta, env);
  execFileSync(programa, args, { stdio: 'pipe' });
  const extraido = path.join(carpeta, binaryName);
  if (!esArchivo(extraido)) {
    throw new Error(`el paquete descargado no trae ${binaryName}`);
  }
  fs.chmodSync(extraido, 0o755);
  execFileSync(extraido, ['--version'], { stdio: 'pipe', timeout: TOPE_VERSION_MS });
  fs.mkdirSync(path.dirname(destino), { recursive: true });
  try {
    moverBinario(extraido, destino);
  } catch (err) {
    quitarSiVacia(path.dirname(destino));
    throw err;
  }
  fs.rmSync(paquete, { force: true });
  return destino;
}

// codigoDeFalla: sin binario, npm tiene que enterarse (y conservar la versión anterior). Quien necesite instalar sin
// binario a propósito, como un build de CI, lo pide con QDD_INSTALL_TOLERANTE=1.
function codigoDeFalla(env) {
  return env.QDD_INSTALL_TOLERANTE === '1' ? 0 : 1;
}

function fallar(motivo) {
  console.error(`[QDD] La instalación falló: ${motivo}`);
  console.error('[QDD] Para reintentar: npm install -g qdd-framework --allow-scripts=qdd-framework');
  process.exit(codigoDeFalla(process.env));
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

module.exports = { esArchivo, estaDentroDeNodeModules, resolverBinarioLocal, comandoExtraer, instalarDescarga, codigoDeFalla, carpetaDeVersiones, destinoDelBinario, respondeBinario, moverBinario, instalarCopia, registrarVersion, tomarTurno };

if (require.main === module) {
  main();
}
