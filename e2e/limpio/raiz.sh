#!/usr/bin/env bash
# Prepara la raíz del ambiente limpio: node:22-bookworm (Debian 12, Node 22, git) fijada por su digest de amd64.
# La baja del registro sin Docker, verifica el sha256 de cada capa y la deja en la caché; la segunda vez no baja nada.
# Imprime la ruta de la raíz lista.
set -euo pipefail

IMAGEN="library/node"
DIGEST="sha256:69c6e263db01e9946c7743bbbe9f7b1ba0df384d27ec09e08e5832d6acd3daf3" # node:22-bookworm, linux/amd64
CACHE="${QDD_LIMPIO_CACHE:-${XDG_CACHE_HOME:-$HOME/.cache}/qdd/limpio}"
RAIZ="$CACHE/${DIGEST#sha256:}"

if [ -f "$RAIZ/.lista" ]; then
  echo "$RAIZ"
  exit 0
fi
if [ "$(uname -m)" != "x86_64" ]; then
  echo "raiz: la imagen fijada es de amd64 y esta máquina es $(uname -m)" >&2
  exit 1
fi

registro="https://registry-1.docker.io/v2/$IMAGEN"
token=$(curl -fsS "https://auth.docker.io/token?service=registry.docker.io&scope=repository:$IMAGEN:pull" | jq -r .token)
manifiesto=$(curl -fsS -H "Authorization: Bearer $token" \
  -H "Accept: application/vnd.oci.image.manifest.v1+json,application/vnd.docker.distribution.manifest.v2+json" \
  "$registro/manifests/$DIGEST")
if [ "$(printf '%s' "$manifiesto" | sha256sum | cut -d' ' -f1)" != "${DIGEST#sha256:}" ]; then
  echo "raiz: el manifiesto no coincide con $DIGEST" >&2
  exit 1
fi

armando="$RAIZ.armando"
rm -rf "$armando"
mkdir -p "$armando/capas" "$armando/raiz"
for capa in $(printf '%s' "$manifiesto" | jq -r '.layers[].digest'); do
  archivo="$armando/capas/${capa#sha256:}"
  curl -fsSL -H "Authorization: Bearer $token" -o "$archivo" "$registro/blobs/$capa"
  if [ "$(sha256sum "$archivo" | cut -d' ' -f1)" != "${capa#sha256:}" ]; then
    echo "raiz: la capa $capa no coincide con su sha256" >&2
    exit 1
  fi
  # Las marcas .wh. de una capa borran lo que dejaron las anteriores (formato OCI).
  for marca in $(tar -tzf "$archivo" | grep -E '(^|/)\.wh\.' || true); do
    dir=$(dirname "$marca")
    nombre=$(basename "$marca")
    if [ "$nombre" = ".wh..wh..opq" ]; then
      find "$armando/raiz/$dir" -mindepth 1 -delete 2>/dev/null || true
      continue
    fi
    rm -rf "${armando:?}/raiz/$dir/${nombre#.wh.}"
  done
  tar -xzf "$archivo" -C "$armando/raiz" --no-same-owner --exclude='dev/*' --exclude='*.wh.*'
  rm -f "$archivo"
done
rmdir "$armando/capas"
mv "$armando/raiz" "$RAIZ"
rm -rf "$armando"
touch "$RAIZ/.lista"
echo "$RAIZ"
