#!/bin/bash
set -e

echo "=> Compilando QDD framework localmente para el entorno interactivo..."
cd "$(dirname "$0")/../cli"
go build -o ../e2e/qdd-bin .

cd ../e2e

echo "=> Construyendo imagen Docker limpia..."
docker build -t qdd-e2e-test .

echo "=> Copiando proyecto local (/home/dpailahueque/Documents/proyectos/actas-back-end)..."
echo "=> Limpiando directorio de pruebas previo..."
docker run --rm -v $(pwd):/workdir -w /workdir ubuntu:22.04 rm -rf test_project
mkdir -p test_project
rsync -a --exclude="node_modules" --exclude=".git" --exclude="dist" --exclude="build" --exclude=".qdd" /home/dpailahueque/Documents/proyectos/actas-back-end/ test_project/

echo "============================================================"
echo " Bienvenido al entorno aislado de pruebas QDD (Docker) "
echo "============================================================"
echo " Estás dentro de un contenedor Ubuntu con Node y Go instalados."
echo " El framework 'qdd' está disponible globalmente."
echo " Tu directorio de trabajo es una copia limpia de 'actas-back-end'."
echo " "
echo " Puedes probar comandos como:"
echo "   $ qdd init"
echo "   $ qdd doctor"
echo "   $ qdd learn"
echo "   $ qdd certify"
echo " "
echo " Cuando termines, escribe 'exit' para destruir el entorno."
echo "============================================================"

# Ejecutar un shell interactivo de bash dentro del contenedor
docker run -it --rm \
    -v $(pwd)/test_project:/test_env/test_project \
    -w /test_env/test_project \
    qdd-e2e-test bash
