#!/bin/sh
# Muestra de un registro de go test qué prueba falló y por qué, y después el final. Cortarlo a las últimas líneas escondía
# la falla cuando el paquete no era de los últimos (FND-322). Uso: fallas.sh <registro>
set -eu
echo "qué falló:"
grep -E '^[[:space:]]*--- FAIL|^(FAIL|panic:)|^[[:space:]]+[A-Za-z0-9_]+_test\.go:[0-9]+:|\[build failed\]' "$1" | head -60 || true
echo "final del registro:"
tail -40 "$1"
