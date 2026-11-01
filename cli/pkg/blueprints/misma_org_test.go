package blueprints

import (
	"os"
	"path/filepath"
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
