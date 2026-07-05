.PHONY: all build-ui build-cli build install run test audit

all: build

build-ui:
	@echo "==> Construyendo Frontend Vue (Dashboard)..."
	cd cli/ui && npm install && npm run build

build-cli:
	@echo "==> Compilando Backend Go (CLI)..."
	cd cli && go build -o ../qdd main.go

build: build-ui build-cli
	@echo "==> Binario 'qdd' generado en la raíz del proyecto."

install: build
	@echo "==> Instalando 'qdd' en ~/.local/bin/ ..."
	mkdir -p ~/.local/bin
	rm -f ~/.local/bin/qdd
	cp qdd ~/.local/bin/qdd
	@echo "==> ¡Instalación completada! Ahora puedes usar el comando 'qdd' en cualquier lugar."

run: install
	@echo "==> Iniciando QDD Dashboard..."
	qdd dashboard

test:
	@echo "==> Ejecutando pruebas unitarias y de arquitectura (Go)..."
	cd cli && go test ./...
	@echo "==> Ejecutando pruebas de componentes Frontend (Vitest)..."
	cd cli/ui && npm run test

audit: install
	@echo "==> Ejecutando auditoría de reglas QDD..."
	qdd audit
