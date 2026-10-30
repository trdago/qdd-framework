package blueprints

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'

// FND-083: propose solo agregaba la carpeta del blueprint; un cambio al catálogo o al README del repositorio (sacar
// un blueprint reemplazado del paquete base) nunca llegaba al PR, y el commit vacío fallaba con «exit status 1».
func TestProponer_IncluyeElCatalogoYElReadmeYSinNadaNuevoNoFallaACiegas_FND083(t *testing.T) {
	pedidos := sinGH(t)
	repo := gitOrgRepo(t)
	root := t.TempDir()
	if _, problems := AddManaged(AddInput{Root: root, Origen: repo, Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	ap, err := Aportar(root, "auth", "")
	if err != nil {
		t.Fatal(err)
	}
	editado := strings.Replace(authBlueprint, `invariantes: ["Autorización por permiso, nunca por nombre de rol"]`, `invariantes: ["Autorización por permiso, nunca por nombre de rol", "Toda sesión expira"]`, 1)
	writeFile(t, filepath.Join(ap.Carpeta, ManifestName), editado)
	if _, err := Proponer(root, "auth", "las sesiones expiran", ""); err != nil {
		t.Fatal(err)
	}
	espacio := filepath.Dir(ap.Carpeta)
	writeFile(t, filepath.Join(espacio, "README.md"), "# Blueprints\n\n`auth` reemplaza al viejo.\n")
	writeFile(t, filepath.Join(espacio, CatalogName), "blueprints:\n  auth: auth\npaquetes:\n  base: [auth]\n")
	p, err := Proponer(root, "auth", "el paquete base solo trae auth", "")
	if err != nil {
		t.Fatalf("un cambio al catálogo y al README se propone: %v", err)
	}
	for _, archivo := range []string{"README.md", CatalogName} {
		got, err := git(repo, "show", "aporte/auth:"+archivo)
		if err != nil || !strings.Contains(got, "auth") || (archivo == CatalogName && !strings.Contains(got, "base: [auth]")) {
			t.Fatalf("%s llega a la rama publicada: %q %v", archivo, got, err)
		}
	}
	if p.Version != "1.1.0" {
		t.Fatalf("la versión sigue la del blueprint, no cambia por el catálogo: %+v", p)
	}
	if _, err := Proponer(root, "auth", "otra vez", ""); err != nil {
		t.Fatalf("proponer sin nada nuevo vuelve a publicar la rama, sin un commit vacío: %v", err)
	}
	if len(*pedidos) != 3 {
		t.Fatalf("cada propose deja el PR al día: %v", len(*pedidos))
	}
	if _, err := os.Stat(filepath.Join(espacio, CatalogName)); err != nil {
		t.Fatal(err)
	}
}

func TestGit_UnErrorDiceLoQueGitEscribio_FND083(t *testing.T) {
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q")
	gitIn(t, dir, "commit", "-q", "--allow-empty", "-m", "x")
	_, err := git(dir, "commit", "-m", "vacío")
	if err == nil || err.Error() == "exit status 1" || !strings.Contains(err.Error(), "nothing") {
		t.Fatalf("el error de git trae su mensaje, aunque lo escriba en la salida: %v", err)
	}
}
