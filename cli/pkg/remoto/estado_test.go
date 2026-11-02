package remoto

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// RFC 0013 (rc.1): qdd remoto estado --json es lo que el PC publicaría, sin red. El panel de cada proyecto es el de
// qdd consola panel --json; un proyecto que falla trae error y nunca un panel inventado.

func homeTemporal(t *testing.T) string {
	t.Helper()
	h := t.TempDir()
	t.Setenv("QDD_HOME", h)
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(h, "claude-code"))
	return h
}

func proyecto(t *testing.T, nombre string, conEstado bool) string {
	t.Helper()
	base, _ := filepath.EvalSymlinks(t.TempDir())
	root := filepath.Join(base, nombre)
	if err := os.MkdirAll(filepath.Join(root, ".qdd", "project"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !conEstado {
		return root
	}
	if err := estado.Init(root, []string{"Diseño", "Pruebas"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestArmar_CadaProyectoConSuPanelOSuError(t *testing.T) {
	homeTemporal(t)
	con, sin, borrado := proyecto(t, "con", true), proyecto(t, "sin", false), proyecto(t, "borrado", true)
	t0 := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	registro.Anotar(con, t0.Add(2*time.Hour))
	registro.Anotar(sin, t0.Add(time.Hour))
	registro.Anotar(borrado, t0)
	os.RemoveAll(borrado)
	reg, _ := registro.Leer()
	e := Armar("v2.9.0-rc.1", reg, t0.Add(3*time.Hour), 0)
	if e.Version != 1 || e.QDD != "v2.9.0-rc.1" || e.Maquina.Nombre == "" || e.Maquina.SO == "" {
		t.Fatalf("cabecera: %+v", e)
	}
	if len(e.Proyectos) != 3 {
		t.Fatalf("tres proyectos: %+v", e.Proyectos)
	}
	nombres := []string{e.Proyectos[0].Nombre, e.Proyectos[1].Nombre, e.Proyectos[2].Nombre}
	if strings.Join(nombres, ",") != "con,sin,borrado" {
		t.Fatalf("por visto, el más reciente primero: %v", nombres)
	}
	if p := e.Proyectos[0]; p.Panel == nil || p.Panel.Estado == nil || p.Error != "" {
		t.Fatalf("con estado: panel con estado: %+v", p)
	}
	if p := e.Proyectos[1]; p.Panel == nil || p.Panel.Estado != nil || p.Panel.Motivo == "" {
		t.Fatalf("sin estado: el panel dice el motivo, como qdd consola panel: %+v", p)
	}
	if p := e.Proyectos[2]; p.Panel != nil || p.Error != "la carpeta ya no existe" {
		t.Fatalf("borrado: error y sin panel: %+v", p)
	}
	b, _ := json.Marshal(e.Proyectos[2])
	if strings.Contains(string(b), `"panel"`) {
		t.Fatalf("un proyecto que falla no trae panel: %s", b)
	}
}

func TestArmar_ElPanelEsElDeQddConsolaPanel(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "p", true)
	registro.Anotar(root, time.Now())
	reg, _ := registro.Leer()
	now := time.Now()
	e := Armar("v1", reg, now, 0)
	esperado := consola.Panel(root, now)
	esperado.Version = "v1"
	a, _ := json.Marshal(e.Proyectos[0].Panel)
	b, _ := json.Marshal(esperado)
	if string(a) != string(b) {
		t.Fatalf("mismo JSON que qdd consola panel --json:\n%s\n%s", a, b)
	}
}

func TestArmar_ElLimiteDelPlanDejaLosMasRecientesYCuentaLosDeFuera(t *testing.T) {
	homeTemporal(t)
	t0 := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	for i, n := range []string{"a", "b", "c", "d", "e"} {
		registro.Anotar(proyecto(t, n, false), t0.Add(time.Duration(i)*time.Hour))
	}
	reg, _ := registro.Leer()
	e := Armar("v1", reg, t0, 3)
	if len(e.Proyectos) != 3 || e.FueraDelPlan != 2 {
		t.Fatalf("3 dentro, 2 fuera: %d %d", len(e.Proyectos), e.FueraDelPlan)
	}
	if e.Proyectos[0].Nombre != "e" || e.Proyectos[2].Nombre != "c" {
		t.Fatalf("los más recientes: %+v", e.Proyectos)
	}
	if sin := Armar("v1", reg, t0, 0); len(sin.Proyectos) != 5 || sin.FueraDelPlan != 0 {
		t.Fatal("sin límite van todos")
	}
}

func TestLinea_DiceNombreFocoTareasEsperanYFallan(t *testing.T) {
	p := ProyectoEstado{Nombre: "qdd", Panel: &consola.DatosConsola{Estado: &estado.DatosPanel{Tareas: 3, Esperan: 2, Fallan: 1,
		Objetivos: []estado.ObjetivoPanel{{Nombre: "Otro"}, {Nombre: "Móvil", Foco: true}}}}}
	if l := Linea(p); l != "qdd · foco: Móvil · 3 tareas · 2 esperan · 1 fallan" {
		t.Fatalf("línea: %q", l)
	}
	if l := Linea(ProyectoEstado{Nombre: "viejo", Error: "la carpeta ya no existe"}); l != "viejo · ✕ la carpeta ya no existe" {
		t.Fatalf("con error: %q", l)
	}
	if l := Linea(ProyectoEstado{Nombre: "x", Panel: &consola.DatosConsola{Motivo: "no hay estado"}}); l != "x · sin estado" {
		t.Fatalf("sin estado: %q", l)
	}
	if l := Linea(ProyectoEstado{Nombre: "y", Panel: &consola.DatosConsola{Estado: &estado.DatosPanel{}}}); l != "y · sin foco · 0 tareas · 0 esperan · 0 fallan" {
		t.Fatalf("sin foco: %q", l)
	}
}
