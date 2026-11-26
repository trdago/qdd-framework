package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
)

// Rule: Prohibido usar 'else'

// FND-236: el panel dice de qué versión es; si no es la de qdd, el panel avisa que hay que reiniciar la consola.
func TestConLoQuePide_UnPanelDeOtraVersionAvisaReiniciar_FND236(t *testing.T) {
	antes := Version
	Version = "v2.10.0-rc.92"
	defer func() { Version = antes }()
	var datos consola.DatosConsola
	conLoQuePide(&datos, t.TempDir(), time.Now(), []string{"atender", "panel:2.10.0-rc.87"})
	if !strings.Contains(datos.AvisoVersion, "claude --continue") {
		t.Fatalf("avisa reiniciar: %q", datos.AvisoVersion)
	}
	datos = consola.DatosConsola{AvisoVersion: "⬆ qdd 2.11.0 disponible"}
	conLoQuePide(&datos, t.TempDir(), time.Now(), []string{"panel:2.10.0-rc.92"})
	if datos.AvisoVersion != "⬆ qdd 2.11.0 disponible" {
		t.Fatalf("al día, deja el aviso que había: %q", datos.AvisoVersion)
	}
}
