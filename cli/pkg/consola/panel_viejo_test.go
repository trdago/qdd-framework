package consola

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// FND-236 (2026-11-15): dos consolas abiertas antes de instalar rc.88 seguían mezclando los proyectos en «Para ti»:
// Claude Code lee el mod una vez, al abrir la consola, y nada decía que su panel era de una versión anterior. El
// instalador deja la versión en el mod; el panel la manda («panel:<versión>» en --con) y qdd avisa si no es la suya.

func TestActivar_ElModLlevaLaVersionQueLoInstalo_FND236(t *testing.T) {
	f := &claudeFalso{}
	c := nuevaConsola(t, "2.2.0", f)
	if _, err := c.Activar(false); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(c.DirMarketplace(), "plugins/qdd-tareas/hooks/register.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `const VERSION_DEL_PANEL = "2.2.0"`) || strings.Contains(string(b), marcaVersionPanel) {
		t.Fatal("el mod instalado dice de qué versión es")
	}
	if d := c.Diagnosticar(); len(d) != 1 || !d[0].OK {
		t.Fatalf("doctor lo sigue viendo al día: %+v", d)
	}
}

func TestAvisoDeReinicio_SoloSiElPanelEsDeOtraVersion_FND236(t *testing.T) {
	aviso := AvisoDeReinicio("2.10.0-rc.87", "v2.10.0-rc.92")
	for _, debe := range []string{"2.10.0-rc.87", "2.10.0-rc.92", "claude --continue"} {
		if !strings.Contains(aviso, debe) {
			t.Fatalf("dice de qué versión es el panel, cuál está instalada y cómo reiniciar (%q): %q", debe, aviso)
		}
	}
	for _, c := range [][2]string{{"2.10.0-rc.92", "v2.10.0-rc.92"}, {"v2.10.0-rc.92", "2.10.0-rc.92"}, {marcaVersionPanel, "v2.10.0-rc.92"}, {"", "v2.10.0-rc.92"}, {"2.10.0-rc.87", "dev"}, {"2.10.0-rc.87", ""}} {
		if aviso := AvisoDeReinicio(c[0], c[1]); aviso != "" {
			t.Fatalf("misma versión, un mod sin instalar o un qdd sin versión no avisan %v: %q", c, aviso)
		}
	}
}
