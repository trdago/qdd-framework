package blueprints

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FND-105: el aporte se abrió con la URL SSH de la organización (config.yaml) y el lockfile, con la HTTPS
// (FND-104); propose buscaba el espacio de trabajo por la URL tal cual y decía «no hay un aporte en curso».
func TestEspacioPath_LaMismaOrganizacionPorSSHOHTTPSEsElMismoAporte_FND105(t *testing.T) {
	t.Setenv(HomeEnv, t.TempDir())
	ssh, err := espacioPath("git@github.com:org/blueprints.git")
	if err != nil {
		t.Fatal(err)
	}
	for _, otra := range []string{"https://github.com/org/blueprints.git", "https://github.com/org/blueprints", "ssh://git@github.com/org/blueprints.git"} {
		got, err := espacioPath(otra)
		if err != nil || got != ssh {
			t.Errorf("espacioPath(%q) = %q, %v; quiere %q", otra, got, err, ssh)
		}
	}
	if otra, _ := espacioPath("https://github.com/org/otros.git"); otra == ssh {
		t.Error("otro repositorio es otro espacio")
	}
}

func TestEspacioPath_UnAporteAbiertoConLaClaveAntiguaSeSigueEncontrando_FND105(t *testing.T) {
	t.Setenv(HomeEnv, t.TempDir())
	dir, err := centralDir("aportes")
	if err != nil {
		t.Fatal(err)
	}
	antiguo := filepath.Join(dir, repoKey("git@github.com:org/blueprints.git"))
	if err := os.MkdirAll(antiguo, 0o700); err != nil {
		t.Fatal(err)
	}
	got, err := espacioPath("https://github.com/org/blueprints.git")
	if err != nil || got != antiguo {
		t.Fatalf("espacioPath = %q, %v; quiere el aporte en curso %q", got, err, antiguo)
	}
}

// FND-105: con el origen ya en HTTPS en el lockfile y el aporte clonado por SSH con una deploy key de solo lectura,
// el push fallaba sin reintentar: solo se reintentaba si el origen era SSH.
func TestProponer_ConElOrigenEnHTTPSReintentaPorEseOrigen_FND105(t *testing.T) {
	sinGH(t)
	repo := gitOrgRepo(t)
	root := t.TempDir()
	mustAddManaged(t, AddInput{Root: root, Origen: repo, Blueprints: []string{"auth"}})
	ap, err := Aportar(root, "auth", "")
	if err != nil {
		t.Fatal(err)
	}
	espacio, err := espacioPath(repo)
	if err != nil {
		t.Fatal(err)
	}
	gitIn(t, espacio, "remote", "set-url", "--push", "origin", filepath.Join(t.TempDir(), "sin-permiso.git"))
	writeFile(t, filepath.Join(ap.Carpeta, ManifestName), strings.Replace(authBlueprint, "¿Con qué inicia sesión el usuario?", "¿Con qué dato inicia sesión?", 1))

	if _, err := Proponer(root, "auth", "redacción", ""); err != nil {
		t.Fatalf("el push reintenta por el origen del lockfile: %v", err)
	}
	if got := versionEn(repo, "aporte/auth", "auth"); got != "1.0.1" {
		t.Fatalf("la rama llegó al repositorio: %q", got)
	}
}
