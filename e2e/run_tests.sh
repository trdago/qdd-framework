#!/bin/bash
set -e

echo "=> Compilando QDD framework localmente para E2E..."
cd "$(dirname "$0")/../cli"
go build -o ../e2e/qdd-bin .

cd ../e2e

echo "=> Construyendo imagen Docker para pruebas..."
docker build -t qdd-e2e-test .

echo "=> Copiando proyecto local (/home/dpailahueque/Documents/proyectos/actas-back-end)..."
echo "=> Limpiando directorio de pruebas (usando docker para evitar problemas de permisos)..."
docker run --rm -v $(pwd):/workdir -w /workdir ubuntu:22.04 rm -rf test_project
mkdir -p test_project
rsync -a --exclude="node_modules" --exclude=".git" --exclude="dist" --exclude="build" --exclude=".qdd" /home/dpailahueque/Documents/proyectos/actas-back-end/ test_project/

echo "=> Ejecutando pruebas dentro del contenedor..."
docker run --rm \
    -v $(pwd)/test_project:/test_env/test_project \
    -v $(pwd)/inner_test.sh:/test_env/inner_test.sh \
    -w /test_env/test_project \
    qdd-e2e-test bash /test_env/inner_test.sh
