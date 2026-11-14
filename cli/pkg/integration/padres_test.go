package integration

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// MEJ-062: una sesión en sfrc cargaba el bloque de QDD tres veces (~/AGENTS.md, ~/Documents/proyectos/AGENTS.md y el
// del proyecto): los AGENTS.md de las carpetas padre también llegan a la IA. El del proyecto es el que vale; los de
// arriba sobran.

var ahoraPadres = time.Date(2026, 11, 14, 1, 45, 0, 0, time.UTC)

func bloqueActual() string { return strings.TrimSpace(QDDCommandProtocol) + "\n" }

// arbolDePadres arma, bajo un tope que hace de carpeta personal:
//
//	tope/AGENTS.md                        bloque suelto (tope no es proyecto)
//	tope/proyectos/.qdd, AGENTS.md        un proyecto QDD que contiene otros
//	tope/proyectos/app/.qdd, AGENTS.md    el proyecto
//	tope/proyectos/app/.claude/worktrees/w/.qdd, AGENTS.md
//	tope/grupo/.qdd, AGENTS.md            proyecto QDD que no contiene proyectos directos
//	tope/grupo/sub/otra/.qdd
func arbolDePadres(t *testing.T) string {
	t.Helper()
	tope := t.TempDir()
	viejo := TopePadres
	TopePadres = func() string { return tope }
	t.Cleanup(func() { TopePadres = viejo })
	escribir := func(rel, contenido string) {
		t.Helper()
		ruta := filepath.Join(tope, rel)
		if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(ruta, []byte(contenido), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	escribir("AGENTS.md", "Notas mías\n\n"+markerBegin+"\n# sin revisión\n"+markerEnd+"\n")
	escribir("proyectos/.qdd/config.yaml", "x: 1\n")
	escribir("proyectos/AGENTS.md", bloqueActual())
	escribir("proyectos/app/.qdd/config.yaml", "x: 1\n")
	escribir("proyectos/app/AGENTS.md", bloqueActual())
	escribir("proyectos/app/.claude/worktrees/w/.qdd/config.yaml", "x: 1\n")
	escribir("proyectos/app/.claude/worktrees/w/AGENTS.md", bloqueActual())
	escribir("grupo/.qdd/config.yaml", "x: 1\n")
	escribir("grupo/AGENTS.md", bloqueActual())
	escribir("grupo/sub/otra/.qdd/config.yaml", "x: 1\n")
	return tope
}

func TestBloquesEnPadres_NombraLasCopiasDeArriba_MEJ062(t *testing.T) {
	tope := arbolDePadres(t)
	app := filepath.Join(tope, "proyectos", "app")
	want := []string{filepath.Join(tope, "proyectos", "AGENTS.md"), filepath.Join(tope, "AGENTS.md")}
	if got := BloquesEnPadres(app); !slices.Equal(got, want) {
		t.Fatalf("el de la carpeta de proyectos y el suelto de arriba: %v, want %v", got, want)
	}
	// Un worktree vive dentro de su proyecto (.claude/worktrees): el bloque del proyecto no es una copia de otro.
	if got := BloquesEnPadres(filepath.Join(app, ".claude", "worktrees", "w")); !slices.Equal(got, want) {
		t.Fatalf("desde un worktree, el del proyecto que lo contiene no cuenta: %v", got)
	}
	// grupo es un proyecto QDD que no contiene proyectos directos: su bloque es suyo y su sync lo volvería a escribir.
	if got := BloquesEnPadres(filepath.Join(tope, "grupo", "sub", "otra")); !slices.Equal(got, []string{filepath.Join(tope, "AGENTS.md")}) {
		t.Fatalf("solo el suelto: %v", got)
	}
	// Nada fuera del tope (la carpeta personal) se mira.
	if got := BloquesEnPadres(filepath.Dir(tope)); got != nil {
		t.Fatalf("fuera del tope no se mira nada: %v", got)
	}
}

func TestQuitarBloquesDePadres_DejaRespaldoYConservaLoDemas_MEJ062(t *testing.T) {
	tope := arbolDePadres(t)
	app := filepath.Join(tope, "proyectos", "app")
	quitados, err := QuitarBloquesDePadres(app, ahoraPadres)
	if err != nil || len(quitados) != 2 {
		t.Fatalf("quita los dos: %v %v", quitados, err)
	}
	data, err := os.ReadFile(filepath.Join(tope, "AGENTS.md"))
	if err != nil || strings.Contains(string(data), markerBegin) || !strings.Contains(string(data), "Notas mías") {
		t.Fatalf("saca solo el bloque y conserva lo demás: %q %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(tope, "proyectos", "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("un AGENTS.md que era solo el bloque se borra: %v", err)
	}
	for _, rel := range []string{"AGENTS.md", filepath.Join("proyectos", "AGENTS.md")} {
		respaldo, err := os.ReadFile(filepath.Join(tope, rel+".qdd-respaldo-20261114T014500"))
		if err != nil || !strings.Contains(string(respaldo), markerBegin) {
			t.Fatalf("%s: deja un respaldo con el archivo entero: %q %v", rel, respaldo, err)
		}
	}
	if data, _ := os.ReadFile(filepath.Join(app, "AGENTS.md")); string(data) != bloqueActual() {
		t.Fatalf("el bloque del proyecto no se toca: %q", data)
	}
	if got := BloquesEnPadres(app); got != nil {
		t.Fatalf("después no queda ninguno: %v", got)
	}
	if quitados, err := QuitarBloquesDePadres(app, ahoraPadres.Add(time.Hour)); err != nil || len(quitados) != 0 {
		t.Fatalf("repetirlo no hace nada: %v %v", quitados, err)
	}
}

func TestSync_UnaCarpetaDeProyectosNoLlevaElBloqueEnAGENTS_MEJ062(t *testing.T) {
	tope := arbolDePadres(t)
	proyectos := filepath.Join(tope, "proyectos")
	if err := os.WriteFile(filepath.Join(proyectos, "AGENTS.md"), []byte("Mis notas\n\n"+bloqueActual()), 0o644); err != nil {
		t.Fatal(err)
	}
	if viejos := BloquesViejos(proyectos); !slices.Equal(viejos, []string{"AGENTS.md"}) {
		t.Fatalf("doctor lo nombra aunque sea de la revisión actual: %v", viejos)
	}
	if err := NewIntegrationManager().SyncAll(proyectos); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(proyectos, "AGENTS.md"))
	if err != nil || strings.Contains(string(data), markerBegin) || !strings.Contains(string(data), "Mis notas") {
		t.Fatalf("el sync lo saca de AGENTS.md y conserva lo demás: %q %v", data, err)
	}
	if data, _ := os.ReadFile(filepath.Join(proyectos, ".cursorrules")); !strings.Contains(string(data), markerBegin) {
		t.Fatal("los archivos que no se heredan a las subcarpetas siguen llevando el bloque")
	}
	if viejos := BloquesViejos(proyectos); len(viejos) != 0 {
		t.Fatalf("después, nada que reparar: %v", viejos)
	}
	// Un proyecto normal sigue recibiendo su AGENTS.md.
	app := filepath.Join(proyectos, "app")
	if err := NewIntegrationManager().SyncAll(app); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(app, "AGENTS.md")); !strings.Contains(string(data), markerBegin) {
		t.Fatal("el proyecto conserva su bloque")
	}
}
