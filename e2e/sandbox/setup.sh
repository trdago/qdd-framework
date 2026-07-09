#!/bin/bash
echo "Iniciando custom script para QDD Sandbox..."

# Configurar variables de entorno necesarias para Go
export HOME=/config
export GOPATH=/config/go
export PATH=$PATH:/usr/local/go/bin

echo "🚀 Compilando QDD..."
cd /source/qdd-cli && go build -o /usr/local/bin/qdd .
chmod +x /usr/local/bin/qdd

echo "📦 Preparando Actas-Back-End limpio..."
rm -rf /config/Desktop/workspace/actas-back-end
rsync -a --exclude='.git' --exclude='node_modules' --exclude='.qdd' /source/actas-back-end/ /config/Desktop/workspace/actas-back-end/
chown -R abc:abc /config/Desktop/workspace/actas-back-end

echo '[Desktop Entry]
Version=1.0
Type=Link
Name=Actas Project
Comment=Open Project Folder
URL=file:///config/Desktop/workspace/actas-back-end
Icon=folder' > /config/Desktop/Actas-Project.desktop

chown abc:abc /config/Desktop/Actas-Project.desktop
chmod +x /config/Desktop/Actas-Project.desktop

echo "✅ QDD Sandbox Init Script completado."
