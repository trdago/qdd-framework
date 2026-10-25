package audit

import (
	"os"
	"path/filepath"
	"testing"
)

func escribirHistorial(t *testing.T, dir, contenido string) {
	t.Helper()
	path := filepath.Join(dir, ".qdd", "project", "metrics", "certificate_history.json")
	os.MkdirAll(filepath.Dir(path), 0755)
	if err := os.WriteFile(path, []byte(contenido), 0644); err != nil {
		t.Fatal(err)
	}
}

// FND-050: al actualizar QDD, un proyecto certificado con la escala anterior
// (−2 por violación) pasaba de 94 a 70 sin cambiar nada y quedaba "Empeorando";
// evolution lo marcaba CRITICAL y pedía no sumar funciones.
func TestGenerateCertificate_CambioDeEscalaNoEsEmpeorar(t *testing.T) {
	dir := t.TempDir()
	escribirHistorial(t, dir, `[{"timestamp":"2026-10-23T10:00:00Z","score":94,"total_violations":3,"tendency":"Estable"}]`)

	tres := []Violation{{RuleID: "A"}, {RuleID: "B"}, {RuleID: "C"}}
	cert, err := GenerateCertificate(dir, tres)
	if err != nil {
		t.Fatal(err)
	}
	if cert.Score != 70 || cert.Tendency != TendencyStable {
		t.Errorf("score %d, tendencia %q; want 70 y Estable (no se comparan escalas distintas)", cert.Score, cert.Tendency)
	}
	if !cert.EscalaNueva || cert.Escala != EscalaActual {
		t.Errorf("debe registrar la escala nueva: %+v", cert)
	}

	cert, err = GenerateCertificate(dir, tres[:2])
	if err != nil {
		t.Fatal(err)
	}
	if cert.Tendency != TendencyImproving || cert.EscalaNueva {
		t.Errorf("dentro de la misma escala sí se compara: %+v", cert)
	}
}

func TestUltimoPuntaje(t *testing.T) {
	dir := t.TempDir()
	if _, ok := UltimoPuntaje(dir); ok {
		t.Error("sin historial no hay puntaje medido")
	}

	escribirHistorial(t, dir, `[{"score":94,"total_violations":3,"tendency":"Estable"}]`)
	if _, ok := UltimoPuntaje(dir); ok {
		t.Error("un puntaje de la escala anterior no vale como medición actual")
	}

	escribirHistorial(t, dir, `[{"score":94,"tendency":"Estable"},{"score":84,"escala":2,"tendency":"Estable"}]`)
	if score, ok := UltimoPuntaje(dir); !ok || score != 84 {
		t.Errorf("UltimoPuntaje = %d, %v; want 84, true", score, ok)
	}
}
