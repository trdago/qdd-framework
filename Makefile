.PHONY: all build-ui build-cli build install run test audit core-assets

all: build

build-ui:
	@echo "==> Construyendo Frontend Vue (Dashboard)..."
	cd cli/ui && npm install && npm run build

# cli/cmd/core_assets/ is a build artifact (not tracked in git — see .gitignore):
# it's embedded into the Go binary via //go:embed, so it must exist before
# `go build` runs. Keep this in sync with package.json's build:cli script.
core-assets:
	@echo "==> Empaquetando Core Assets (.qdd/core -> cli/cmd/core_assets)..."
	rm -rf cli/cmd/core_assets
	mkdir -p cli/cmd/core_assets/core
	cp -r .qdd/core/* cli/cmd/core_assets/core/
	find cli/cmd/core_assets -type d -empty -exec touch {}/.keep \;

build-cli: core-assets
	@echo "==> Compilando Backend Go (CLI)..."
	cd cli && go build -ldflags="-X 'github.com/qdd-framework/qdd/cmd.Version=v$$(cat ../package.json | grep version | head -1 | awk -F: '{ print $$2 }' | sed 's/[\", ]//g')'" -o ../qdd main.go

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

test: core-assets
	@echo "==> Ejecutando pruebas unitarias y de arquitectura (Go)..."
	cd cli && go test ./...
	@echo "==> Ejecutando pruebas de componentes Frontend (Vitest)..."
	cd cli/ui && npm run test

audit: install
	@echo "==> Ejecutando auditoría de reglas QDD..."
	qdd audit
