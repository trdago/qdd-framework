package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Pedido de la persona (2026-11-07): Docker en desarrollo solo para la base de datos. Quien igual lo quiera para la app
// puede usarlo, pero QDD avisa que es más lento y por qué.

func proyectoCon(t *testing.T, archivos map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for ruta, contenido := range archivos {
		completa := filepath.Join(dir, ruta)
		if err := os.MkdirAll(filepath.Dir(completa), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(completa, []byte(contenido), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestDockerDesarrollo_SoloBaseDeDatosNoAvisa(t *testing.T) {
	casos := map[string]string{
		"imagen":             "services:\n  db:\n    image: postgres:17-alpine\n",
		"registro con ruta":  "services:\n  base:\n    image: ghcr.io/org/postgres:17\n    build: ./base\n",
		"llamado db":         "services:\n  db:\n    build: ./base\n",
		"montaje de inicio":  "services:\n  db:\n    image: postgres:17\n    volumes:\n      - ./init.sql:/docker-entrypoint-initdb.d/init.sql\n",
		"construida":         "services:\n  db:\n    image: mariadb:11\n    build: ./db\n",
		"sin imagen":         "services:\n  redis:\n    build: ./cache\n",
		"herramienta aparte": "services:\n  correo:\n    image: mailpit:latest\n    volumes:\n      - datos:/data\n",
		"montaje absoluto":   "services:\n  visor:\n    image: visor:1\n    volumes:\n      - /var/run/docker.sock:/var/run/docker.sock\n",
	}
	for nombre, compose := range casos {
		t.Run(nombre, func(t *testing.T) {
			if v := RunDockerDesarrolloCheck(proyectoCon(t, map[string]string{"docker-compose.yml": compose})); len(v) != 0 {
				t.Fatalf("no debía avisar: %+v", v)
			}
		})
	}
}

func TestDockerDesarrollo_LaAppEnDockerAvisaSinBloquear(t *testing.T) {
	casos := map[string]struct{ archivo, compose, servicio string }{
		"construida":        {"docker-compose.yml", "services:\n  db:\n    image: postgres:17\n  api:\n    build: ./backend\n", "api"},
		"código montado":    {"compose.yaml", "services:\n  front:\n    image: node:22\n    volumes:\n      - ./frontend:/app\n", "front"},
		"raíz montada":      {"compose.yml", "services:\n  web:\n    image: node:22\n    volumes:\n      - .:/app\n", "web"},
		"con variable":      {"docker-compose.dev.yml", "services:\n  web:\n    image: node:22\n    volumes:\n      - ${PWD}:/app\n", "web"},
		"con PWD":           {"docker-compose.dev.yml", "services:\n  web:\n    image: node:22\n    volumes:\n      - $PWD:/app\n", "web"},
		"formato largo":     {"docker-compose.yaml", "services:\n  web:\n    image: node:22\n    volumes:\n      - type: bind\n        source: ./src\n        target: /app\n", "web"},
		"en una subcarpeta": {"infra/docker-compose.yml", "services:\n  api:\n    build: ../backend\n", "api"},
	}
	for nombre, c := range casos {
		t.Run(nombre, func(t *testing.T) {
			v := RunDockerDesarrolloCheck(proyectoCon(t, map[string]string{c.archivo: c.compose}))
			if len(v) != 1 {
				t.Fatalf("debía dar un aviso: %+v", v)
			}
			if v[0].RuleID != ReglaAppEnDocker || !v[0].Recomendacion {
				t.Fatalf("debía ser una recomendación %s: %+v", ReglaAppEnDocker, v[0])
			}
			for _, debe := range []string{c.archivo, c.servicio, "más lento", "base de datos", "docker-en-desarrollo.md"} {
				if !strings.Contains(v[0].Description, debe) {
					t.Errorf("el aviso no dice %q: %s", debe, v[0].Description)
				}
			}
		})
	}
}

// Un proyecto real traía cinco composes: un aviso por archivo le restaba 10 puntos por una sola decisión.
func TestDockerDesarrollo_UnSoloAvisoPorProyectoConTodosSusServicios(t *testing.T) {
	v := RunDockerDesarrolloCheck(proyectoCon(t, map[string]string{
		"docker-compose.yml":       "services:\n  front:\n    build: ./frontend\n  api:\n    build: ./backend\n  db:\n    image: postgres:17\n",
		"docker-compose.local.yml": "services:\n  worker:\n    build: ./worker\n",
		".devcontainer.json":       "{}",
	}))
	if len(v) != 1 {
		t.Fatalf("debía ser un solo aviso: %+v", v)
	}
	for _, debe := range []string{"docker-compose.yml (api, front)", "docker-compose.local.yml (worker)", ".devcontainer.json"} {
		if !strings.Contains(v[0].Description, debe) {
			t.Errorf("el aviso no dice %q: %s", debe, v[0].Description)
		}
	}
	if strings.Contains(v[0].Description, "db") {
		t.Fatalf("la base de datos no es la app: %s", v[0].Description)
	}
}

func TestDockerDesarrollo_EnCarpetaDeDespliegueSoloCuentaElDeDesarrollo(t *testing.T) {
	const app = "services:\n  api:\n    build: ../backend\n"
	for _, archivo := range []string{"deploy/docker-compose.yml", "deploy/docker/compose.yml", "deployment/docker-compose.yaml"} {
		if v := RunDockerDesarrolloCheck(proyectoCon(t, map[string]string{archivo: app})); len(v) != 0 {
			t.Errorf("%s no debía avisar: %+v", archivo, v)
		}
	}
	for _, archivo := range []string{"deploy/docker/docker-compose.local.yml", "deploy/docker-compose.dev.yml"} {
		if v := RunDockerDesarrolloCheck(proyectoCon(t, map[string]string{archivo: app})); len(v) != 1 {
			t.Errorf("%s debía avisar: %+v", archivo, v)
		}
	}
}

func TestDockerDesarrollo_ContenedorDeDesarrolloAvisa(t *testing.T) {
	for _, archivo := range []string{".devcontainer/devcontainer.json", ".devcontainer.json"} {
		t.Run(archivo, func(t *testing.T) {
			v := RunDockerDesarrolloCheck(proyectoCon(t, map[string]string{archivo: "{}"}))
			if len(v) != 1 || !v[0].Recomendacion || !strings.Contains(v[0].Description, archivo) {
				t.Fatalf("debía avisar por %s: %+v", archivo, v)
			}
		})
	}
}

func TestDockerDesarrollo_LoQueNoEsDesarrolloNoAvisa(t *testing.T) {
	const app = "services:\n  api:\n    build: ./backend\n"
	for _, archivo := range []string{
		"docker-compose.prod.yml",
		"docker-compose.production.yaml",
		"compose.staging.yml",
		"e2e/sandbox/docker-compose.sandbox.yml",
		"test/docker-compose.yml",
		"tests/compose.yml",
		"node_modules/x/docker-compose.yml",
		"vendor/x/compose.yaml",
		".claude/worktrees/x/docker-compose.yml",
		"Dockerfile",
		"docs/ejemplo.yml",
	} {
		t.Run(archivo, func(t *testing.T) {
			if v := RunDockerDesarrolloCheck(proyectoCon(t, map[string]string{archivo: app})); len(v) != 0 {
				t.Fatalf("no debía avisar: %+v", v)
			}
		})
	}
}

func TestDockerDesarrollo_UnComposeIlegibleNoSeInventa(t *testing.T) {
	for nombre, compose := range map[string]string{
		"roto":          "services: [sin cerrar\n",
		"sin servicios": "version: '3'\n",
		"vacío":         "",
		"servicio raro": "services:\n  api:\n    build: ./backend\n  otro: 3\n",
	} {
		t.Run(nombre, func(t *testing.T) {
			if v := RunDockerDesarrolloCheck(proyectoCon(t, map[string]string{"docker-compose.yml": compose})); len(v) != 0 {
				t.Fatalf("no debía avisar: %+v", v)
			}
		})
	}
}

func TestDockerDesarrollo_ElGateLoInformaYNoBloquea(t *testing.T) {
	dir := proyectoCon(t, map[string]string{"docker-compose.yml": "services:\n  api:\n    build: ./backend\n"})
	var violaciones []Violation
	NewEngine(dir).runCoreChecks(QDDPolicies{}, &violaciones)
	var aviso []Violation
	for _, v := range violaciones {
		if v.RuleID == ReglaAppEnDocker {
			aviso = append(aviso, v)
		}
	}
	if len(aviso) != 1 {
		t.Fatalf("el gate debía correr la regla: %+v", violaciones)
	}
	if len(Bloqueantes(aviso)) != 0 {
		t.Fatalf("no debe bloquear, tampoco en adopción estricta: %+v", aviso)
	}
}
