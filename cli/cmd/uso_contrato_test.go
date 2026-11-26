package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/uso"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Rule: Prohibido usar 'else'

// Registrar el uso es lo primero de cada comando, herramienta MCP, pulso y ciclo: con el registro tomado por otro
// proceso (o por uno que murió con el bloqueo puesto), con QDD_HOME inservible o con el registro ilegible, el comando
// sigue en el acto y sin error.
// contrato: uso#444f9bea
func TestRegistrarUso_NuncaFallaNiDemoraElComando(t *testing.T) {
	proyecto := t.TempDir()
	os.MkdirAll(filepath.Join(proyecto, ".qdd"), 0o755)
	t0 := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)

	t.Setenv("QDD_HOME", t.TempDir())
	dir, _ := uso.Carpeta()
	os.MkdirAll(dir, 0o700)
	if err := os.WriteFile(filepath.Join(dir, "2.8.0.json.lock"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	inicio := time.Now()
	registrarUsoEn("v2.8.0", proyecto, t0)
	if demora := time.Since(inicio); demora > time.Second {
		t.Fatalf("con el registro tomado por otro, el comando esperó %v", demora)
	}

	os.WriteFile(filepath.Join(dir, "2.9.0.json"), []byte("{roto"), 0o600)
	registrarUsoEn("v2.9.0", proyecto, t0)
	if b, _ := os.ReadFile(filepath.Join(dir, "2.9.0.json")); string(b) != "{roto" {
		t.Fatalf("un registro ilegible no se pisa: %q", b)
	}

	archivo := filepath.Join(t.TempDir(), "no-es-carpeta")
	os.WriteFile(archivo, []byte("x"), 0o600)
	t.Setenv("QDD_HOME", archivo)
	inicio = time.Now()
	registrarUsoEn("v2.8.0", proyecto, t0)
	if demora := time.Since(inicio); demora > time.Second {
		t.Fatalf("sin dónde anotar, el comando esperó %v", demora)
	}
}

// El uso solo sale de eventos reales: qdd no tiene un comando ni una bandera para anotar horas a mano, y la evidencia
// que pide el gate es lo medido, o nada.
// contrato: uso#b1ea0968
func TestUso_NoSeCompletaAManoDesdeQDD(t *testing.T) {
	var subcomandos []string
	for _, c := range usoCmd.Commands() {
		subcomandos = append(subcomandos, c.Name())
	}
	if strings.Join(subcomandos, ",") != "evidencia" {
		t.Fatalf("qdd uso solo muestra y deja la evidencia de lo medido: %v", subcomandos)
	}
	for _, c := range []*cobra.Command{usoCmd, usoEvidenciaCmd} {
		c.LocalFlags().VisitAll(func(f *pflag.Flag) {
			if f.Name != "help" {
				t.Errorf("%s no recibe horas ni tiempos: --%s", c.Name(), f.Name)
			}
		})
	}

	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	if _, _, err := uso.Evidencia(root, "2.8.0", time.Now()); err == nil {
		t.Fatal("sin uso medido no hay evidencia")
	}
	t0 := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	for _, m := range []int{0, 3, 6, 60} {
		if err := uso.Registrar("2.8.0-rc.1", "qdd", t0.Add(time.Duration(m)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	_, ev, err := uso.Evidencia(root, "2.8.0", t0.Add(2*time.Hour))
	if err != nil || ev.Horas != 0.1 {
		t.Fatalf("la evidencia es lo medido entre eventos seguidos (6 min), sin estimar el hueco: %+v %v", ev, err)
	}
}
