package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/incidentes"
)

// Rule: Prohibido usar 'else'
// FND-189 (MEJ-050): una vista del panel que lanza ya no lo deja en negro, y la falla llega a QDD sin que la persona
// tenga que notarla: el panel llama 'qdd incidentes panel <vista> <mensaje> --traza <función>…'.

func TestIncidentesPanel_UnaVistaQueFallaQuedaAnonimizada_FND189(t *testing.T) {
	raiz, dir := conProyectoPrivado(t)
	var salida bytes.Buffer
	codigo := incidenteDelPanel([]string{"blueprints", "null is not an object (evaluating 'inv.entradas.filter') en " + raiz + "/x"}, []string{"vistaBlueprints", "dibujarVista"}, &salida)
	if codigo != 0 {
		t.Fatalf("salida %d: %s", codigo, salida.String())
	}
	i := soloIncidente(t, dir)
	if i.Tipo != incidentes.TipoPanel || i.Origen != "panel blueprints" || !strings.HasPrefix(i.Mensaje, "null is not an object") {
		t.Fatalf("incidente = %+v", i)
	}
	if strings.Join(i.Traza, " ") != "vistaBlueprints dibujarVista" {
		t.Fatalf("la traza son los nombres de las funciones del panel: %v", i.Traza)
	}
	sinRastrosDelProyecto(t, dir, raiz)
}

func TestIncidentesPanel_SoloNombresDeVistaYFuncion(t *testing.T) {
	_, dir := conProyectoPrivado(t)
	var salida bytes.Buffer
	if incidenteDelPanel([]string{"../x", "m"}, nil, &salida) == 0 {
		t.Fatal("una vista que no es un nombre no se registra")
	}
	if incidenteDelPanel([]string{"flujos", "m"}, []string{"rm -rf /"}, &salida) == 0 {
		t.Fatal("una traza que no es un nombre de función no se registra")
	}
	if incidenteDelPanel([]string{"flujos", strings.Repeat("x", 400)}, nil, &salida) != 0 {
		t.Fatal("un mensaje largo se recorta, no se rechaza")
	}
	if i := soloIncidente(t, dir); len([]rune(i.Mensaje)) > maxMensajeDelPanel {
		t.Fatalf("recortado a %d: %d", maxMensajeDelPanel, len([]rune(i.Mensaje)))
	}
}
