#!/bin/bash
set -e

# ==============================================================================
# QDD Real-World E2E Testing Script (Ephemeral Clone Strategy)
# ==============================================================================
# Este script toma la ruta de un proyecto real (ej. actas-back-end), crea un clon
# temporal en /tmp, ejecuta QDD (init, learn, audit) para verificar heurísticas
# en escenarios de la vida real, y luego destruye el clon sin afectar el original.

TARGET_DIR=$1

if [ -z "$TARGET_DIR" ]; then
    echo "[!] Uso: $0 <ruta-absoluta-del-proyecto-real>"
    echo "Ejemplo: $0 /home/dpailahueque/Documents/proyectos/actas-back-end/"
    exit 1
fi

if [ ! -d "$TARGET_DIR" ]; then
    echo "[!] Error: El directorio '$TARGET_DIR' no existe."
    exit 1
fi

TMP_CLONE="/tmp/qdd-test-real-world-$(date +%s)"

echo "[+] Clonando proyecto de forma efímera a $TMP_CLONE ..."
mkdir -p "$TMP_CLONE"

# Usamos cp -R pero excluimos .git y dependencias pesadas si es posible para mayor velocidad
# (Aunque para una simulación real exacta, copiaremos todo, pero excluyendo node_modules por velocidad)
rsync -a --exclude="node_modules" "$TARGET_DIR/" "$TMP_CLONE/"

echo "[+] Compilando QDD local para inyectarlo en el clon..."
cd cli
go build -o qdd-bin main.go
QDD_BINARY=$(pwd)/qdd-bin
cd ..

echo "================================================================="
echo "[⚙️] INICIANDO SIMULACIÓN QDD EN EL PROYECTO CLONADO"
echo "================================================================="

cd "$TMP_CLONE"

echo -e "\n--- Ejecutando: qdd init ---"
$QDD_BINARY init

echo -e "\n--- Ejecutando: qdd learn ---"
# Simulamos paso de argumentos vacíos al comando interactivo
echo "" | $QDD_BINARY learn

echo -e "\n--- Ejecutando: qdd audit ---"
$QDD_BINARY audit || true

echo -e "\n--- Validando estado interno (.qdd/config.yaml) ---"
cat .qdd/config.yaml

echo -e "\n================================================================="
echo "[+] SIMULACIÓN COMPLETADA EXITOSAMENTE."
echo "================================================================="

# Cleanup (Rollback / Destrucción del clon)
echo "[-] Limpiando entorno temporal ($TMP_CLONE)..."
rm -rf "$TMP_CLONE"
echo "[✔] Repositorio original ($TARGET_DIR) completamente intacto."
