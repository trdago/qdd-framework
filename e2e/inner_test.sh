#!/bin/bash
set -e

# --- LIBRERÍA DE ASERCIONES ---
assert_dir_exists() {
    if [ ! -d "$1" ]; then echo "❌ FALLO: Directorio $1 no existe"; exit 1; fi
    echo "✅ [OK] Directorio $1 verificado"
}
assert_file_exists() {
    if [ ! -f "$1" ]; then echo "❌ FALLO: Archivo $1 no existe"; exit 1; fi
    echo "✅ [OK] Archivo $1 verificado"
}
assert_file_contains() {
    if ! grep -q "$2" "$1"; then echo "❌ FALLO: '$2' no encontrado en $1"; exit 1; fi
    echo "✅ [OK] '$2' encontrado en $1"
}
assert_file_not_contains() {
    if grep -q "$2" "$1"; then echo "❌ FALLO: '$2' encontrado en $1 (y no debería estar)"; exit 1; fi
    echo "✅ [OK] Ausencia de '$2' confirmada en $1"
}

echo "=========================================="
echo "FASE 1: INICIALIZACIÓN (QDD INIT)"
echo "=========================================="
qdd init

echo "-> Validando Estructura Core..."
for d in core/runtime core/specification core/templates core/plugins core/certification core/wisdom project/findings project/sprints project/certification project/evidence project/metrics project/adr dashboard; do
    assert_dir_exists ".qdd/$d"
done

echo "-> Validando Configuraciones y Estado..."
assert_file_exists ".qdd/config.yaml"
assert_file_exists ".qdd/state.json"

echo "-> Validando Inyección de Skills (qdd.md)..."
assert_file_exists ".agents/workflows/qdd.md"
assert_file_contains ".agents/workflows/qdd.md" "description:"
assert_file_not_contains ".agents/workflows/qdd.md" "<description>"

echo "-> Validando Servidores MCP e Integraciones..."
assert_file_exists ".cursor/mcp.json"
assert_file_contains ".cursor/mcp.json" "qdd"
assert_file_exists ".clauderc"
assert_file_exists ".antigravityrules"
assert_file_contains ".clauderc" "/qdd"
assert_file_contains ".antigravityrules" "/qdd"

echo "=========================================="
echo "FASE 2: RESILIENCIA (QDD DOCTOR)"
echo "=========================================="
echo "-> Simulando daño crítico..."
rm -rf .qdd/core
rm -f .qdd/config.yaml

echo "-> Validando detección de daño..."
if qdd doctor; then
    echo "❌ FALLO: qdd doctor no detectó el daño (devolvió 0)"
    exit 1
fi
echo "✅ [OK] qdd doctor detectó el daño correctamente"

echo "-> Ejecutando auto-reparación..."
qdd doctor --fix

echo "-> Validando reconstrucción..."
assert_dir_exists ".qdd/core/certification"
assert_file_exists ".qdd/config.yaml"
assert_file_exists ".qdd/state.json"

echo "=========================================="
echo "FASE 3: COGNICIÓN (QDD LEARN)"
echo "=========================================="
echo "-> Saltando qdd learn temporalmente, la arquitectura ahora es detectada automáticamente por qdd init."

echo "=========================================="
echo "FASE 4: MOTOR DE AUDITORÍA (QDD CERTIFY)"
echo "=========================================="
qdd certify

echo "-> Validando reportes de certificación..."
if [ $(ls .qdd/project/evidence/doctor/report_*.md 2>/dev/null | wc -l) -eq 0 ]; then
    echo "❌ FALLO: No se generó reporte en evidence/doctor/"
    exit 1
fi
echo "✅ [OK] Reporte markdown generado"
assert_file_exists ".qdd/project/metrics/certificate_history.json"

echo "=========================================="
echo "FASE 5: VALIDACIÓN DASHBOARD UI/API"
echo "=========================================="
echo "-> Levantando QDD Dashboard en background..."
# Se redirige el output para no ensuciar los logs y se captura el PID
qdd dashboard > dashboard_test.log 2>&1 &
DASH_PID=$!

echo "-> Esperando a que el servidor del dashboard esté disponible (max 10s)..."
for i in {1..10}; do
    if curl -s http://localhost:8099/api/state > api_state.json; then
        echo "✅ [OK] Dashboard respondió"
        break
    fi
    echo "   ...esperando (intento $i)"
    sleep 1
done

echo "-> Validando que el API refleje la arquitectura detectada..."
if ! grep -q '"architecture":"Serverless Cloud Functions"' api_state.json; then
    echo "❌ FALLO: La arquitectura 'Serverless Cloud Functions' no se encuentra en el JSON servido por el Dashboard."
    cat api_state.json
    kill $DASH_PID
    exit 1
fi
echo "✅ [OK] El dashboard expuso correctamente la arquitectura al frontend"

echo "-> Teardown: apagando dashboard..."
kill $DASH_PID || true

echo "====================================================="
echo "✅ SUCCESS: TODAS LAS ASERCIONES HAN PASADO CON ÉXITO"
echo "====================================================="
