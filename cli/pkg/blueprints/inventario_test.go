package blueprints

import (
	"path/filepath"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// Pedido del usuario (2026-11-01): un acceso rápido a los blueprints en el panel, determinista y sin tokens. qdd dice
// cuáles hay, cuáles están instalados y si una versión nueva se puede tomar sin la IA (la receta es la misma) o cambia
// lo que hay que construir.

func entradaDe(t *testing.T, c Inventario, id string) EntradaInventario {
	t.Helper()
	for _, e := range c.Entradas {
		if e.ID == id {
			return e
		}
	}
	t.Fatalf("%s no está en el catálogo: %+v", id, c)
	return EntradaInventario{}
}

func proyectoConAuth(t *testing.T) (root, repo string) {
	t.Helper()
	freshHome(t)
	repo = orgRepo(t, nil)
	root = t.TempDir()
	if _, problems := Add(AddInput{Root: root, Origen: repo, Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	return root, repo
}

func TestInventario_InstaladosAlDiaYLosDemasDisponibles(t *testing.T) {
	root, _ := proyectoConAuth(t)
	c, err := InventarioDe(root)
	if err != nil {
		t.Fatal(err)
	}
	if e := entradaDe(t, c, "auth"); e.Estado != CatAlDia || e.Instalada != "1.0.0" || e.Publicada != "1.0.0" || e.Nombre != "Usuarios y acceso" {
		t.Fatalf("auth está instalado y al día: %+v", e)
	}
	if e := entradaDe(t, c, "pagos"); e.Estado != CatDisponible || e.Instalada != "" || e.Publicada != "1.0.0" {
		t.Fatalf("pagos se puede instalar: %+v", e)
	}
	entradaDe(t, c, "auditoria")
	if c.Instalados != 1 || c.Disponibles != 2 {
		t.Fatalf("cuenta lo que hay: %+v", c)
	}
}

func TestInventario_LaMismaRecetaConOtroTagSeActualizaSinIA(t *testing.T) {
	root, repo := proyectoConAuth(t)
	gitIn(t, repo, "commit", "-q", "--allow-empty", "-m", "otro blueprint")
	gitIn(t, repo, "tag", "v1.1.0")
	if _, err := Sincronizar(root); err != nil {
		t.Fatal(err)
	}
	c, err := InventarioDe(root)
	if err != nil {
		t.Fatal(err)
	}
	e := entradaDe(t, c, "auth")
	if e.Estado != CatSoloLock || e.Ref != "v1.1.0" {
		t.Fatalf("misma receta en un tag nuevo: solo cambia el lock: %+v", e)
	}
	if c.SinIA != 1 || c.ConIA != 0 {
		t.Fatalf("una actualización sin tokens: %+v", c)
	}
}

func TestInventario_UnaRecetaQueCambiaPideLaIAParaAdaptarElModulo(t *testing.T) {
	root, repo := proyectoConAuth(t)
	cambiada := strings.Replace(authBlueprint, `invariantes: ["Autorización por permiso, nunca por nombre de rol"]`, `invariantes: ["Autorización por permiso, nunca por nombre de rol", "Toda sesión expira"]`, 1)
	writeFile(t, filepath.Join(repo, "auth", ManifestName), strings.Replace(cambiada, "version: 1.0.0", "version: 1.1.0", 1))
	gitIn(t, repo, "commit", "-q", "-am", "1.1.0")
	gitIn(t, repo, "tag", "v1.1.0")
	if _, err := Sincronizar(root); err != nil {
		t.Fatal(err)
	}
	c, _ := InventarioDe(root)
	e := entradaDe(t, c, "auth")
	if e.Estado != CatConCambios || e.Publicada != "1.1.0" || e.Cambio != CambioMinor || len(e.Razones) == 0 {
		t.Fatalf("la receta cambió: actualizar es determinista, adaptar el módulo es de la IA: %+v", e)
	}
	if c.ConIA != 1 {
		t.Fatalf("una que pide la IA: %+v", c)
	}
}

func TestInventario_SinEspejoDiceComoTraerlo(t *testing.T) {
	root, _ := proyectoConAuth(t)
	freshHome(t)
	c, err := InventarioDe(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Problemas) == 0 || !strings.Contains(c.Problemas[0], "qdd blueprint sync") {
		t.Fatalf("sin la copia central no inventa nada, dice cómo traerla: %+v", c)
	}
	if e := entradaDe(t, c, "auth"); e.Estado != CatSinMedir || e.Instalada != "1.0.0" {
		t.Fatalf("lo instalado se sigue viendo, sin medir: %+v", e)
	}
}

// Instalado por https y la organización registrada por ssh (así está qdd mismo): es el mismo repositorio.
func TestInventario_SSHYHTTPSSonElMismoRepositorio(t *testing.T) {
	lock := Lock{Blueprints: []LockEntry{{ID: "estado-por-tareas", Origen: "https://github.com/org/bp.git", Version: "2.0.0"}}}
	if l := instaladoDe(lock, "git@github.com:org/bp.git", "estado-por-tareas"); l == nil || l.Version != "2.0.0" {
		t.Fatalf("instalado desde la otra forma de la URL: %+v", l)
	}
	if l := instaladoDe(lock, "git@github.com:otra/bp.git", "estado-por-tareas"); l != nil {
		t.Fatalf("otro repositorio con el mismo id no cuenta: %+v", l)
	}
}
