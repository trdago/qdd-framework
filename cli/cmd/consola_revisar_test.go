package cmd

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// La tecla l del panel lee el JSON de qdd consola revisar: sin proyecto ni estado, total 0 y nada que mandar a la IA.
func TestConsolaRevisar_SinEstadoNoHayNadaQueMandar(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir()) // la cola de pedidos se lee de aquí: nunca la de la persona
	var out bytes.Buffer
	if code := revisarPanel(t.TempDir(), time.Now(), &out); code != 0 {
		t.Fatalf("exit %d", code)
	}
	var r struct {
		Texto string `json:"texto"`
		Total int    `json:"total"`
	}
	if err := json.Unmarshal(out.Bytes(), &r); err != nil || r.Texto != "" || r.Total != 0 {
		t.Fatalf("JSON vacío esperado, vino %q (%v)", out.String(), err)
	}
}
