#!/bin/bash
# Crea proyectos realistas "de primera vez" en $1 (uno por carpeta).
set -e
B="$1"
rm -rf "$B" && mkdir -p "$B"

# 1. API Go con módulo y prueba
d="$B/go-api"; mkdir -p "$d/internal/usuarios" "$d/cmd/api"
cat > "$d/go.mod" <<'EOF'
module ejemplo.cl/api

go 1.24
EOF
cat > "$d/cmd/api/main.go" <<'EOF'
package main

import (
	"fmt"

	"ejemplo.cl/api/internal/usuarios"
)

func main() {
	fmt.Println(usuarios.Saludo("mundo"))
}
EOF
cat > "$d/internal/usuarios/usuarios.go" <<'EOF'
package usuarios

// Saludo arma un saludo.
func Saludo(nombre string) string {
	if nombre == "" {
		return "hola"
	}
	return "hola " + nombre
}
EOF
cat > "$d/internal/usuarios/usuarios_test.go" <<'EOF'
package usuarios

import "testing"

func TestSaludo(t *testing.T) {
	if Saludo("ana") != "hola ana" {
		t.Fatal("saludo incorrecto")
	}
}
EOF

# 2. Frontend Vue + Vite (sin node_modules: así llega un repo recién clonado)
d="$B/vue-front"; mkdir -p "$d/src/components"
cat > "$d/package.json" <<'EOF'
{
  "name": "vue-front",
  "version": "0.1.0",
  "private": true,
  "scripts": { "dev": "vite", "build": "vite build" },
  "dependencies": { "vue": "^3.4.0" },
  "devDependencies": { "vite": "^5.0.0", "@vitejs/plugin-vue": "^5.0.0" }
}
EOF
cat > "$d/src/main.js" <<'EOF'
import { createApp } from 'vue'
import App from './App.vue'

createApp(App).mount('#app')
EOF
cat > "$d/src/App.vue" <<'EOF'
<template><h1>{{ titulo }}</h1></template>
<script setup>
const titulo = 'Hola'
</script>
EOF

# 3. Python FastAPI con pytest
d="$B/py-api"; mkdir -p "$d/app" "$d/tests"
cat > "$d/requirements.txt" <<'EOF'
fastapi==0.110.0
pytest==8.0.0
EOF
cat > "$d/app/__init__.py" <<'EOF'
EOF
cat > "$d/app/main.py" <<'EOF'
from fastapi import FastAPI

app = FastAPI()


def suma(a: int, b: int) -> int:
    return a + b


@app.get("/salud")
def salud():
    return {"ok": True}
EOF
cat > "$d/tests/test_main.py" <<'EOF'
from app.main import suma


def test_suma():
    assert suma(2, 3) == 5
EOF

# 4. Flutter
d="$B/flutter-app"; mkdir -p "$d/lib" "$d/test"
cat > "$d/pubspec.yaml" <<'EOF'
name: flutter_app
description: App de ejemplo
version: 1.0.0+1
environment:
  sdk: ">=3.0.0 <4.0.0"
dependencies:
  flutter:
    sdk: flutter
dev_dependencies:
  flutter_test:
    sdk: flutter
EOF
cat > "$d/lib/main.dart" <<'EOF'
import 'package:flutter/material.dart';

void main() => runApp(const MaterialApp(home: Text('Hola')));
EOF

# 5. Monorepo backend Go + frontend Vue
d="$B/monorepo"; mkdir -p "$d/backend" "$d/frontend"
cp -r "$B/go-api/." "$d/backend/"
cp -r "$B/vue-front/." "$d/frontend/"

# 6. Repo vacío (proyecto que recién empieza)
mkdir -p "$B/vacio"

# Todos como repos git con un commit, salvo 'sin-git'
for p in go-api vue-front py-api flutter-app monorepo vacio; do
	git -C "$B/$p" init -q -b main
	git -C "$B/$p" add -A
	git -C "$B/$p" -c user.name=t -c user.email=t@t commit -q --allow-empty -m "inicio"
done

# 7. Carpeta con código pero sin git
d="$B/sin-git"; mkdir -p "$d"
cp -r "$B/go-api/." "$d/" && rm -rf "$d/.git"
