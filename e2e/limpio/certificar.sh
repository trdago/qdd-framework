#!/usr/bin/env bash
# Certifica QDD desde cero: instala el paquete en un Debian 12 fijo, aislado con bubblewrap (sin Docker ni daemon),
# y lo usa como una persona nueva (init, doctor, estado, pedido, audit). Nada de esta máquina entra: ni HOME, ni
# credenciales, ni la red (salvo --version, que baja de npm). Lo que escribe muere con el ambiente.
#
#   e2e/limpio/certificar.sh                       empaqueta esta rama (binario de cli/) y la certifica sin red
#   e2e/limpio/certificar.sh --binario <qdd>       con ese binario ya compilado
#   e2e/limpio/certificar.sh --version 2.10.0-rc.96   la versión publicada en npm (con red)
set -euo pipefail

aqui=$(cd "$(dirname "$0")" && pwd)
repo=$(cd "$aqui/../.." && pwd)
binario=""
version=""
while [ $# -gt 0 ]; do
  case "$1" in
    --binario) binario="$2"; shift 2 ;;
    --version) version="$2"; shift 2 ;;
    *) echo "uso: certificar.sh [--binario <qdd> | --version <x.y.z>]" >&2; exit 2 ;;
  esac
done

if ! command -v bwrap >/dev/null; then
  echo "certificar: falta bubblewrap (bwrap). Arch: pacman -S bubblewrap · Debian/Ubuntu: apt install bubblewrap" >&2
  exit 1
fi

raiz=$("$aqui/raiz.sh")
entrada=$(mktemp -d "${TMPDIR:-/tmp}/qdd-limpio.XXXXXX")
trap 'rm -rf "$entrada"' EXIT
red=(--unshare-net)

if [ -n "$version" ]; then
  red=(--share-net --ro-bind /etc/resolv.conf /etc/resolv.conf)
fi
if [ -z "$version" ] && [ -z "$binario" ]; then
  mkdir -p "$repo/cli/ui/dist"
  [ -f "$repo/cli/ui/dist/index.html" ] || echo '' >"$repo/cli/ui/dist/index.html"
  cp -r "$repo/.qdd/core/." "$repo/cli/cmd/core_assets/core/"
  (cd "$repo/cli" && CGO_ENABLED=0 go build -buildvcs=false -o "$entrada/qdd" .)
  binario="$entrada/qdd"
fi
if [ -z "$version" ]; then
  [ "$binario" = "$entrada/qdd" ] || cp "$binario" "$entrada/qdd"
  (cd "$repo/npm" && npm pack --silent --pack-destination "$entrada" >/dev/null)
  mv "$entrada"/qdd-framework-*.tgz "$entrada/qdd-framework.tgz"
fi

cp "$aqui/dentro.sh" "$entrada/dentro.sh"
inicio=$(date +%s%N)
codigo=0
bwrap --ro-bind "$raiz" / \
  --proc /proc --dev /dev --tmpfs /tmp --tmpfs /home \
  --ro-bind "$entrada" /mnt \
  "${red[@]}" --unshare-user --unshare-pid --unshare-ipc --unshare-uts --unshare-cgroup-try \
  --uid 1000 --gid 1000 --hostname limpio --die-with-parent --new-session --clearenv \
  --setenv HOME /home/persona --setenv USER persona --setenv LANG C.UTF-8 --setenv TERM xterm-256color \
  --setenv PATH /usr/local/bin:/usr/bin:/bin \
  --setenv QDD_LIMPIO_VERSION "$version" \
  --dir /home/persona --chdir /home/persona \
  /bin/bash /mnt/dentro.sh || codigo=$?
echo "Ambiente limpio: $(( ($(date +%s%N) - inicio) / 1000000 )) ms en total."
exit "$codigo"
