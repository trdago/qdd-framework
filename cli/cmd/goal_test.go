package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
)

// Rule: Prohibido usar 'else'

const estadoConTareaGoal = `formato: 1
actualizado: "2026-11-04"
etapas: [Código, PR]
frentes:
  - {id: barra, nombre: Barra, estado: abierto, celdas: {Código: {estado: "⏳", bloqueo: falta, de_quien: c}, PR: {estado: "—"}}}
`

func proyectoGoal(t *testing.T) string {
	t.Helper()
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	ruta := filepath.Join(root, ".qdd", "project", "estado.yaml")
	if err := os.MkdirAll(filepath.Dir(ruta), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte(estadoConTareaGoal), 0644); err != nil {
		t.Fatal(err)
	}
	return root
}

// FND-146: esta es la barrera que queda si la regla del panel no ve el comando (el issue #112 no la había comprobado): la
// acción del panel relanza qdd con el entorno de quien la llamó, así que una IA tampoco lo activa por aquí.
func TestGoal_UnaIANoLoActivaNiPorLaAccionDelPanel_FND146(t *testing.T) {
	root := proyectoGoal(t)
	var salida, errores bytes.Buffer
	t.Setenv("CLAUDECODE", "1")
	if codigo := consolaAccion(root, consola.AccionGoalActivar, true, "", &salida, &errores); codigo == 0 || !strings.Contains(salida.String(), "solo la persona") {
		t.Fatalf("con CLAUDECODE es una IA: %d %s %s", codigo, salida.String(), errores.String())
	}
	if consola.VistaGoal(root, time.Now()).Activo {
		t.Fatal("no quedó activo")
	}
	t.Setenv("CLAUDECODE", "")
	if codigo := consolaAccion(root, consola.AccionGoalActivar, true, "", &salida, &errores); codigo != 0 {
		t.Fatalf("el panel limpia CLAUDECODE y la persona sí: %d %s %s", codigo, salida.String(), errores.String())
	}
	if !consola.VistaGoal(root, time.Now()).Activo {
		t.Fatal("quedó activo")
	}
}

func TestGoal_UnaIANoLoActivaYLaPersonaSi(t *testing.T) {
	root := proyectoGoal(t)
	now := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	var salida, errores bytes.Buffer
	if codigo := goalActivar(root, "panel", "1", now, &salida, &errores); codigo == 0 || !strings.Contains(errores.String(), "solo la persona") {
		t.Fatalf("con CLAUDECODE es una IA: %d %s", codigo, errores.String())
	}
	if consola.VistaGoal(root, now).Activo {
		t.Fatal("no quedó activo")
	}
	if codigo := goalActivar(root, "panel", "", now, &salida, &errores); codigo != 0 {
		t.Fatalf("la persona sí: %d %s", codigo, errores.String())
	}
	if !consola.VistaGoal(root, now).Activo {
		t.Fatal("quedó activo")
	}
	if codigo := goalDesactivar(root, now, &salida, &errores); codigo != 0 || consola.VistaGoal(root, now).Activo {
		t.Fatalf("se apaga: %d", codigo)
	}
}

func TestGoal_EstadoEnJSONSigueElContrato(t *testing.T) {
	root := proyectoGoal(t)
	var salida bytes.Buffer
	if codigo := goalEstado(root, true, time.Now(), &salida); codigo != 0 {
		t.Fatal(codigo)
	}
	var g map[string]any
	if err := json.Unmarshal(salida.Bytes(), &g); err != nil {
		t.Fatal(err)
	}
	if g["activo"] != false || g["elegidas"] == nil {
		t.Fatalf("apagado dice activo:false y elegidas vacía: %s", salida.String())
	}
}

func TestGoal_DoctorVeElArchivoIlegibleYLoAparta(t *testing.T) {
	root := proyectoGoal(t)
	if g := buildGoalChecks(root); len(g.Items) != 0 {
		t.Fatalf("sin modo no opina: %+v", g.Items)
	}
	if codigo := goalActivar(root, "panel", "", time.Now(), &bytes.Buffer{}, &bytes.Buffer{}); codigo != 0 {
		t.Fatal(codigo)
	}
	if g := buildGoalChecks(root); len(g.Items) != 1 || !g.Items[0].Success {
		t.Fatalf("activo y legible: %+v", g.Items)
	}
	ruta, err := consola.RutaLocal(root, "goal")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte("{roto"), 0644); err != nil {
		t.Fatal(err)
	}
	g := buildGoalChecks(root)
	if len(g.Items) != 1 || g.Items[0].Success {
		t.Fatalf("doctor lo ve: %+v", g.Items)
	}
	repairGoal(root, g)
	if g := buildGoalChecks(root); len(g.Items) != 0 {
		t.Fatalf("apartado queda apagado: %+v", g.Items)
	}
}

func TestGoal_NoPasaPorElGatekeeperNiElRegistroDeTrabajo(t *testing.T) {
	if goalCmd.PersistentPreRunE == nil || goalCmd.PersistentPostRun == nil {
		t.Fatal("corre cada 5 s y en proyectos sin config: sin los hooks de la raíz")
	}
}

