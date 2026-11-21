package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// FND-226: 'qdd version' caía en el intérprete de intenciones y mostraba el aviso del motor cognitivo deprecado en vez
// de la versión; solo 'qdd --version' la daba. Se usa tras cada instalación para confirmar qué quedó.
func TestVersion_SubcomandoDaLaVersion_FND226(t *testing.T) {
	var out bytes.Buffer
	versionCmd.SetOut(&out)
	t.Cleanup(func() { versionCmd.SetOut(nil) })
	versionCmd.Run(versionCmd, nil)
	if got := strings.TrimSpace(out.String()); got != "qdd version "+rootCmd.Version {
		t.Fatalf("'qdd version' dice lo mismo que 'qdd --version': %q", got)
	}
}

func TestVersion_NoPideContextoDelProyecto_FND226(t *testing.T) {
	c, _, err := rootCmd.Find([]string{"version"})
	if err != nil || c != versionCmd {
		t.Fatalf("'version' es un subcomando, no una intención: %v %v", c, err)
	}
	if c.PersistentPreRunE == nil {
		t.Fatal("no pasa por el gatekeeper ni por la revisión de versión del proyecto: muestra la versión en cualquier carpeta")
	}
}
