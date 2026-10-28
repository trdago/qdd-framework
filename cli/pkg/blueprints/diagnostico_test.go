package blueprints

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/modules"
)

func diagnostico(t *testing.T, ds []Diagnostico, nombre string) Diagnostico {
	t.Helper()
	for _, d := range ds {
		if d.Nombre == nombre {
			return d
		}
	}
	t.Fatalf("falta el diagnóstico %q en %+v", nombre, ds)
	return Diagnostico{}
}

func todosOK(t *testing.T, ds []Diagnostico) {
	t.Helper()
	for _, d := range ds {
		if !d.OK {
			t.Fatalf("%s: %s", d.Nombre, d.Detalle)
		}
	}
}

func TestDiagnosticar_ProjectWithoutBlueprintsIsHealthy(t *testing.T) {
	ds := Diagnosticar(t.TempDir())
	if len(ds) != 5 {
		t.Fatalf("doctor siempre muestra los 5 diagnósticos: %+v", ds)
	}
	todosOK(t, ds)
	if !strings.Contains(diagnostico(t, ds, DiagAcceso).Detalle, "sin blueprints") {
		t.Fatalf("sin blueprints no se intenta acceder a ningún repositorio: %+v", ds)
	}
}

func TestReparar_MigratesLegacyCopies(t *testing.T) {
	root, _ := legacyProject(t)
	if diagnostico(t, Diagnosticar(root), DiagLegado).OK {
		t.Fatal("las copias del esquema anterior se reportan")
	}
	acciones := Reparar(root)
	if !strings.Contains(strings.Join(acciones, "\n"), "auth") {
		t.Fatalf("la reparación dice qué migró: %v", acciones)
	}
	todosOK(t, Diagnosticar(root))
}

func TestReparar_RebuildsOnANewMachine(t *testing.T) {
	repo := gitOrgRepo(t)
	root := t.TempDir()
	if _, problems := AddManaged(AddInput{Root: root, Origen: repo, Blueprints: []string{"auth", "pagos"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	freshHome(t)
	d := diagnostico(t, Diagnosticar(root), DiagCentral)
	if d.OK || !strings.Contains(d.Detalle, "auth") {
		t.Fatalf("en un PC nuevo faltan los blueprints en la carpeta central: %+v", d)
	}
	Reparar(root)
	todosOK(t, Diagnosticar(root))
}

func TestDiagnosticar_WithoutAccessExplainsTheCredential(t *testing.T) {
	repo := gitOrgRepo(t)
	root := t.TempDir()
	if _, problems := AddManaged(AddInput{Root: root, Origen: repo, Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	os.RemoveAll(repo) // sin acceso (o sin red)
	if d := diagnostico(t, Diagnosticar(root), DiagAcceso); !d.OK || !strings.Contains(d.Detalle, "sin conexión") {
		t.Fatalf("sin red, pero con la carpeta central completa, no es un problema: %+v", d)
	}

	freshHome(t) // y además en un PC nuevo: ahora sí falta algo
	Reparar(root)
	ds := Diagnosticar(root)
	acceso := diagnostico(t, ds, DiagAcceso)
	if acceso.OK || !strings.Contains(acceso.Detalle, "solo lectura") {
		t.Fatalf("sin acceso y con blueprints faltantes, se explica qué credencial falta: %+v", acceso)
	}
	if diagnostico(t, ds, DiagCentral).OK {
		t.Fatal("lo que no se pudo reconstruir sigue fallando")
	}
}

func TestReparar_SyncsButNeverUpdatesOnItsOwn(t *testing.T) {
	repo := gitOrgRepo(t)
	root := t.TempDir()
	if _, problems := AddManaged(AddInput{Root: root, Origen: repo, Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	publicar(t, repo, "1.1.0", "v1.1.0")
	Reparar(root)
	d := diagnostico(t, Diagnosticar(root), DiagVigencia)
	if d.OK || !strings.Contains(d.Detalle, "qdd blueprint add auth") {
		t.Fatalf("doctor trae lo publicado y dice cómo actualizar: %+v", d)
	}
	if lock, _ := LoadLock(root); lock.Blueprints[0].Version != "1.0.0" {
		t.Fatal("doctor nunca actualiza un blueprint por su cuenta: puede traer preguntas del humano")
	}
}

// moduloAplicado importa auth del repositorio y lo aplica al módulo crm, con
// rutas propias del proyecto.
func moduloAplicado(t *testing.T) (string, string) {
	t.Helper()
	repo := gitOrgRepo(t)
	root := t.TempDir()
	gitIn(t, root, "init", "-q")
	if _, problems := AddManaged(AddInput{Root: root, Origen: repo, Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	writeFile(t, filepath.Join(root, "internal", "crm", "main.go"), "package crm\n")
	in := ApplyInput{Root: root, Blueprint: "auth", Modulo: "crm", Rutas: []string{"internal/crm"}, Respuestas: map[string]string{"identificador": "correo", "recuperacion": "correo"}}
	if _, problems := Apply(in); len(problems) > 0 {
		t.Fatal(problems)
	}
	return root, repo
}

func TestApply_ReapplyKeepsTheModuleRoutesAndCitesExistingSources(t *testing.T) {
	root, _ := moduloAplicado(t)
	if _, problems := Apply(ApplyInput{Root: root, Blueprint: "auth", Modulo: "crm"}); len(problems) > 0 {
		t.Fatal(problems)
	}
	m, _ := modules.Find(root, "crm")
	if strings.Join(m.Rutas, ",") != "internal/crm" {
		t.Fatalf("re-aplicar sin --ruta conserva las rutas del módulo: %v", m.Rutas)
	}
	if _, err := os.Stat(filepath.Join(root, "internal", "auth")); !os.IsNotExist(err) {
		t.Fatal("re-aplicar no crea esqueletos en las rutas sugeridas por el blueprint")
	}
	for _, f := range m.Fuentes {
		if strings.HasPrefix(f, Dir+"/") {
			t.Fatalf("la ficha no cita archivos que ya no existen en el proyecto: %v", m.Fuentes)
		}
	}
	if !strings.Contains(strings.Join(m.Fuentes, ","), LockFile) {
		t.Fatalf("la ficha cita el lockfile como fuente del blueprint: %v", m.Fuentes)
	}
}

func TestReparar_ReappliesOutdatedModules(t *testing.T) {
	root, repo := moduloAplicado(t)
	publicar(t, repo, "1.1.0", "v1.1.0")
	if _, problems := AddManaged(AddInput{Root: root, Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	if diagnostico(t, Diagnosticar(root), DiagModulos).OK {
		t.Fatal("un módulo generado con la versión anterior se reporta")
	}
	acciones := strings.Join(Reparar(root), "\n")
	todosOK(t, Diagnosticar(root))
	m, _ := modules.Find(root, "crm")
	if m.Blueprint != "auth@1.1.0" || m.Estado != modules.EstadoBorrador || !strings.Contains(acciones, "crm") {
		t.Fatalf("se re-aplica con las respuestas registradas y la ficha vuelve a borrador para el humano: %+v %s", m, acciones)
	}
}

func TestReparar_NewQuestionsAreTheHumansDecision(t *testing.T) {
	root, repo := moduloAplicado(t)
	nueva := strings.Replace(authBlueprint, "ficha:\n", "  - id: mfa\n    pregunta: ¿Se exige un segundo factor?\n    opciones: [{valor: si}, {valor: no}]\nficha:\n", 1)
	writeFile(t, filepath.Join(repo, "auth", ManifestName), strings.Replace(nueva, "version: 1.0.0", "version: 1.1.0", 1))
	gitIn(t, repo, "commit", "-q", "-am", "mfa")
	gitIn(t, repo, "tag", "v1.1.0")
	if _, problems := AddManaged(AddInput{Root: root, Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	acciones := strings.Join(Reparar(root), "\n")
	d := diagnostico(t, Diagnosticar(root), DiagModulos)
	if d.OK || !strings.Contains(acciones, "mfa") || !strings.Contains(acciones, "qdd blueprint apply auth --modulo crm") {
		t.Fatalf("una pregunta nueva no se responde sola: doctor dice exactamente qué decidir: %s | %+v", acciones, d)
	}
}

func TestReparar_ReimportsTheBlueprintOfAModule(t *testing.T) {
	root, _ := moduloAplicado(t)
	if err := saveLock(root, Lock{}); err != nil {
		t.Fatal(err)
	}
	if diagnostico(t, Diagnosticar(root), DiagModulos).OK {
		t.Fatal("un módulo cuyo blueprint ya no está importado se reporta")
	}
	Reparar(root)
	todosOK(t, Diagnosticar(root))
}

func TestReparar_HandEditedLockfile(t *testing.T) {
	root, _ := moduloAplicado(t)
	lock, _ := LoadLock(root)
	lock.Blueprints[0].SHA256 = strings.Repeat("0", 64)
	if err := saveLock(root, lock); err != nil {
		t.Fatal(err)
	}
	Reparar(root)
	if problems := Check(root); len(problems) != 0 {
		t.Fatalf("un lockfile editado a mano se rehace desde la versión publicada que fija: %v", problems)
	}
}

func TestReparar_UnreadableLockfileComesBackFromGit(t *testing.T) {
	root, _ := moduloAplicado(t)
	gitIn(t, root, "add", ".")
	gitIn(t, root, "commit", "-q", "-m", "proyecto")
	writeFile(t, filepath.Join(root, LockFile), "blueprints: [roto\n")
	if diagnostico(t, Diagnosticar(root), DiagCentral).OK {
		t.Fatal("un lockfile ilegible se reporta")
	}
	Reparar(root)
	todosOK(t, Diagnosticar(root))
}
