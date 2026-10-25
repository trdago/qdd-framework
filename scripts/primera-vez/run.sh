#!/bin/bash
# Recorrido "primera vez": crea 7 proyectos típicos (Go, Vue, FastAPI, Flutter,
# monorepo, vacío y sin git) y recorre en cada uno lo que hace una persona nueva
# con QDD. Termina con código 1 si algún paso falla fuera de lo esperado.
# Uso: scripts/primera-vez/run.sh <binario-qdd> [carpeta-salida]
set -u
QDD="$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"
AQUI="$(cd "$(dirname "$0")" && pwd)"
SALIDA="${2:-$(mktemp -d)}"

bash "$AQUI/fixtures.sh" "$SALIDA/proyectos" || exit 2
bash "$AQUI/journey.sh" "$QDD" "$SALIDA/proyectos" "$SALIDA/resultado" > /dev/null 2>&1

python3 -I "$AQUI/evaluar.py" "$SALIDA/resultado/resumen.tsv" "$AQUI/esperado.tsv"
codigo=$?
echo "Logs de cada paso: $SALIDA/resultado/<proyecto>/<paso>.log"
exit $codigo
