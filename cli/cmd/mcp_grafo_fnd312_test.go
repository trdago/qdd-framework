package cmd

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/server"
)

// Rule: Prohibido usar 'else'

// contrato: mcp#8a991eb6
// FND-312: la consulta al grafo aceptaba cualquier SQL, también el que borra.
func TestGrafo_ConsultaSoloLee_FND312(t *testing.T) {
	for nombre, registrar := range map[string]func(*server.MCPServer){"qdd_query_graph": registerQueryGraphTool, "qdd_graph_query": registerGraphQueryTool} {
		dir := enProyecto(t)
		ruta := grafoDePrueba(t, dir)
		for _, q := range []string{
			"DELETE FROM nodes", "DROP TABLE nodes", "INSERT INTO nodes(id) VALUES ('b')",
			"PRAGMA query_only=0; DELETE FROM nodes", "ATTACH DATABASE 'otra.db' AS o; CREATE TABLE o.x(a)", "ATTACH DATABASE 'otra.db' AS o",
			"SELECT 1; DELETE FROM nodes", "WITH x AS (SELECT 1) DELETE FROM nodes", "VACUUM INTO 'otra.db'",
		} {
			if texto, falla := llamarTool(t, registrar, nombre, map[string]interface{}{"query": q}); !falla {
				t.Errorf("%s aceptó %q: %s", nombre, q, texto)
			}
		}
		if _, err := os.Stat(filepath.Join(dir, "otra.db")); err == nil {
			t.Errorf("%s creó otra base", nombre)
		}
		texto, falla := llamarTool(t, registrar, nombre, map[string]interface{}{"query": " select id FROM nodes; "})
		if falla || !strings.Contains(texto, `"a"`) {
			t.Errorf("%s no devolvió lo que hay en %s: %s", nombre, ruta, texto)
		}
	}
}

func TestGrafo_SinBaseLoDice_FND312(t *testing.T) {
	dir := enProyecto(t)
	texto, falla := llamarTool(t, registerQueryGraphTool, "qdd_query_graph", map[string]interface{}{"query": "SELECT 1"})
	if !falla || !strings.Contains(texto, "qdd_sync_graph") {
		t.Errorf("sin base debía decir cómo crearla: %q", texto)
	}
	if _, err := os.Stat(filepath.Join(dir, ".qdd", "knowledge.db")); err == nil {
		t.Error("una lectura creó la base")
	}
}

func TestMCP_ElMapaNoDiceQueSoloLee_FND312(t *testing.T) {
	if soloLeen["qdd_map"] {
		t.Error("qdd_map escribe topology.json: no puede llevar readOnlyHint")
	}
}

func TestPostgresTuner_NoDiceQueAnalizaLaConsulta_FND312(t *testing.T) {
	enProyecto(t)
	texto, falla := llamarTool(t, registerPostgresTunerTool, "qdd_postgres_tuner", map[string]interface{}{"query": "SELECT 1"})
	if falla || strings.Contains(texto, "Análisis") || !strings.Contains(texto, "no analiza") {
		t.Errorf("son consejos generales y tiene que decirlo: %s", texto)
	}
}

func grafoDePrueba(t *testing.T, dir string) string {
	t.Helper()
	ruta := filepath.Join(dir, ".qdd", "knowledge.db")
	db, err := sql.Open("sqlite", ruta)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(t.Context(), "CREATE TABLE nodes(id TEXT); INSERT INTO nodes VALUES ('a')"); err != nil {
		t.Fatal(err)
	}
	return ruta
}
