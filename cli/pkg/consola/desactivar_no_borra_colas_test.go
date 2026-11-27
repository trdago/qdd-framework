package consola

import (
	"os"
	"path/filepath"
	"testing"
)

// Rule: Prohibido usar 'else'
// FND-332: Desactivar borraba toda la carpeta <QDD_HOME>/claude. Ahí, además del marketplace del panel, vive el estado
// local de cada proyecto (pkg/estado.RutaLocal): pedidos, preguntas, goal, turnos, teléfono… Desactivar el panel no
// puede borrar eso: quita solo lo que Activar escribió.

func TestDesactivar_QuitaElPanelYNoLasColasDeLosProyectos_FND332(t *testing.T) {
	c := nuevaConsola(t, "2.2.0", &claudeFalso{})
	if _, err := c.Activar(false); err != nil {
		t.Fatal(err)
	}
	colas := map[string]string{}
	for _, carpeta := range []string{"pedidos", "preguntas", "goal", "turnos"} {
		ruta := filepath.Join(c.DirMarketplace(), carpeta, "proyecto-a-0123456789ab.json")
		escribirArchivo(t, ruta, `[{"id":"1"}]`)
		colas[carpeta] = ruta
	}
	escribirArchivo(t, filepath.Join(c.DirMarketplace(), "limites.json"), `{}`)

	if _, err := c.Desactivar(); err != nil {
		t.Fatal(err)
	}
	for carpeta, ruta := range colas {
		if _, err := os.Stat(ruta); err != nil {
			t.Fatalf("desactivar borró la cola «%s» de un proyecto: %v", carpeta, err)
		}
	}
	if _, err := os.Stat(filepath.Join(c.DirMarketplace(), "limites.json")); err != nil {
		t.Fatal("desactivar borró limites.json")
	}
	for _, delPanel := range []string{"plugins", ".claude-plugin", "activo.json"} {
		if _, err := os.Stat(filepath.Join(c.DirMarketplace(), delPanel)); !os.IsNotExist(err) {
			t.Fatalf("desactivar deja «%s» del panel: %v", delPanel, err)
		}
	}
}
