#!/bin/bash
set -e

echo "🚀 Iniciando el QDD Visual Sandbox..."
cd "$(dirname "$0")"

docker-compose -f docker-compose.sandbox.yml up -d --build

echo ""
echo "================================================================"
echo "✅ SANDBOX LEVANTADO EXITOSAMENTE"
echo "================================================================"
echo "Abre tu navegador en: http://localhost:3000"
echo "Verás un escritorio Ubuntu completo."
echo ""
echo "Tu proyecto está en el Escritorio (Actas-Project)."
echo "Puedes abrir la terminal, ejecutar 'qdd init' y abrir VS Code"
echo "escribiendo 'code /config/Desktop/workspace/actas-back-end'"
echo "================================================================"
echo ""
echo "Para detener el sandbox, ejecuta:"
echo "  docker-compose -f e2e/sandbox/docker-compose.sandbox.yml down"
