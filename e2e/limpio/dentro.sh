#!/usr/bin/env bash
# Corre dentro del ambiente limpio: una persona que instala qdd por primera vez en un proyecto nuevo.
# init, doctor, estado, pedido, learn, certify y audit. Cada paso dice cuánto tardó; el primero que falla corta con su salida. /mnt trae el paquete (y el binario).
set -uo pipefail

export NPM_CONFIG_PREFIX="$HOME/.npm-global" NPM_CONFIG_UPDATE_NOTIFIER=false NPM_CONFIG_FUND=false NPM_CONFIG_AUDIT=false
export PATH="$HOME/.npm-global/bin:$PATH"
REGISTRO="$HOME/limpio.log"
FALLAS=0

paso() {
  local nombre="$1"
  shift
  local inicio salida codigo
  inicio=$(date +%s%N)
  salida=$("$@" 2>&1)
  codigo=$?
  printf '%s\n--- %s (exit %d)\n' "$salida" "$nombre" "$codigo" >>"$REGISTRO"
  local ms=$(( ($(date +%s%N) - inicio) / 1000000 ))
  if [ "$codigo" -eq 0 ]; then
    printf '  ✓ %-46s %6d ms\n' "$nombre" "$ms"
    return 0
  fi
  printf '  ✗ %-46s %6d ms (exit %d)\n' "$nombre" "$ms" "$codigo"
  printf '%s\n' "$salida" | tail -25 | sed 's/^/      /'
  FALLAS=$((FALLAS + 1))
  return 1
}

# contiene <texto> <comando…>: el comando sale bien y su salida dice el texto.
contiene() {
  local texto="$1"
  shift
  local salida
  salida=$("$@" 2>&1) || { printf '%s\n' "$salida"; return 1; }
  printf '%s\n' "$salida"
  grep -qF -- "$texto" <<<"$salida" || { echo "falta «$texto»"; return 1; }
}

# aislamiento: nada de la máquina entra salvo /mnt (paquete, binario y dentro.sh) y la raíz, todo en solo lectura;
# HOME vacío y sin credenciales; el entorno trae solo lo que fija certificar.sh (contrato ambiente-limpio#378ef7ec).
aislamiento() {
  local nombre fallas=0
  while IFS='=' read -r nombre _; do
    case "$nombre" in
      HOME | USER | LANG | TERM | PATH | QDD_LIMPIO_VERSION | PWD | OLDPWD | SHLVL | _ | NPM_CONFIG_*) ;;
      *) echo "el entorno trae $nombre: certificar.sh corre con --clearenv"; fallas=1 ;;
    esac
  done < <(env)
  [ "$HOME" = /home/persona ] || { echo "HOME es $HOME, no /home/persona"; fallas=1; }
  [ -z "$(ls -A "$HOME")" ] || { echo "HOME no está vacío: $(ls -A "$HOME" | tr '\n' ' ')"; fallas=1; }
  [ "$(ls -A /home)" = persona ] || { echo "/home trae más que persona: $(ls -A /home | tr '\n' ' ')"; fallas=1; }
  for archivo in $(ls -A /mnt); do
    case "$archivo" in
      dentro.sh | qdd | qdd-framework.tgz) ;;
      *) echo "/mnt trae $archivo: solo entran el paquete, el binario y dentro.sh"; fallas=1 ;;
    esac
  done
  for dir in / /mnt /usr /etc /root; do
    if touch "$dir/.qdd-escritura" 2>/dev/null; then
      echo "se puede escribir en $dir: la raíz y /mnt van en solo lectura"
      rm -f "$dir/.qdd-escritura"
      fallas=1
    fi
  done
  return "$fallas"
}

instalar() {
  if [ -n "${QDD_LIMPIO_VERSION:-}" ]; then
    npm install -g "qdd-framework@$QDD_LIMPIO_VERSION" --allow-scripts=qdd-framework
    return
  fi
  QDD_INSTALL_BINARIO=/mnt/qdd npm install -g /mnt/qdd-framework.tgz --allow-scripts=qdd-framework
}

proyecto_nuevo() {
  mkdir -p "$HOME/proyecto" && cd "$HOME/proyecto" || return 1
  git init -q -b main
  git config user.email limpio@qdd.invalid
  git config user.name "Ambiente limpio"
  cat >package.json <<'JSON'
{ "name": "suma", "version": "1.0.0", "scripts": { "test": "node --test" } }
JSON
  printf 'exports.suma = (a, b) => a + b\n' >suma.js
  cat >suma.test.js <<'JS'
const test = require('node:test')
const assert = require('node:assert')
const { suma } = require('./suma')
test('suma', () => assert.strictEqual(suma(2, 3), 5))
JS
  git add -A && git commit -qm "proyecto nuevo"
}

tarea_en_el_estado() {
  local salida id
  qdd estado init --etapas "Código,Pruebas,Producción" || return 1
  salida=$(qdd estado nueva "Primera tarea" 2>&1)
  printf '%s\n' "$salida"
  id=$(grep -oE 'tarea [a-z0-9-]+' <<<"$salida" | head -1 | cut -d' ' -f2)
  [ -n "$id" ] || { echo "qdd estado nueva no dio el id"; return 1; }
  qdd estado set "$id" Código ok --evidencia "suma.js" || return 1
  qdd estado --json | grep -qF "\"$id\"" || { echo "la tarea $id no está en el estado"; return 1; }
}

# presupuesto <tope> <qué> <comando…>: lo que una IA recibe en cada sesión cabe en su presupuesto de caracteres
# (~4 por token). Un proyecto nuevo es el piso: si ya se pasa acá, se pasa en todos.
presupuesto() {
  local tope="$1" que="$2"
  shift 2
  local n
  n=$("$@" 2>/dev/null | wc -c)
  echo "$que: $n caracteres (tope $tope)"
  [ "$n" -gt 0 ] && [ "$n" -le "$tope" ]
}

tools_list() {
  printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"limpio","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' |
    timeout 20 qdd mcp-server 2>/dev/null | grep '"id":2' | node -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>process.stdout.write(JSON.stringify(JSON.parse(s).result.tools)))'
}

esperado="${QDD_LIMPIO_VERSION:-$(tar -xzOf /mnt/qdd-framework.tgz package/package.json | node -pe 'JSON.parse(require("fs").readFileSync(0)).version')}"

echo "Ambiente limpio: $(. /etc/os-release && echo "$PRETTY_NAME"), node $(node --version), git $(git --version | cut -d' ' -f3)"
paso "aislado: sin HOME, credenciales ni escritura afuera" aislamiento || exit 1
paso "instala qdd con npm" instalar || exit 1
paso "qdd version es $esperado" contiene "v$esperado" qdd version || exit 1
paso "proyecto nuevo con git y una prueba" proyecto_nuevo || exit 1
cd "$HOME/proyecto" || exit 1
paso "qdd init detecta Node y deja .qdd" bash -c 'qdd init && test -d .qdd'
paso "qdd doctor sin ✗ ni ❌" bash -c 's=$(qdd doctor 2>&1); c=$?; printf "%s\n" "$s"; [ $c -eq 0 ] && ! grep -qE "❌|✗" <<<"$s"'
paso "qdd estado: init, una tarea y su celda" tarea_en_el_estado
paso "qdd consola pedir deja el pedido" contiene "echo hola" bash -c "qdd consola pedir -m 'Probar el ambiente' 'echo hola' && qdd consola pedidos"
paso "tokens: tools/list del MCP ≤ 18.000" presupuesto 18000 "tools/list" tools_list
paso "tokens: qdd_project_context ≤ 8.000" presupuesto 8000 "qdd_project_context" qdd modules context suma.js
paso "qdd learn" qdd learn
paso "qdd certify" qdd certify
paso "qdd audit aprueba el proyecto nuevo" qdd audit

if [ "$FALLAS" -gt 0 ]; then
  echo "$FALLAS paso(s) fallaron; registro completo en $REGISTRO"
  exit 1
fi
echo "Certificado: qdd $esperado se instala y trabaja desde cero."
