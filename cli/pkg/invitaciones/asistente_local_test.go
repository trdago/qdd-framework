package invitaciones

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'

// Lo que el asistente propone desde una conversación (transcripcion@mm:ss) queda en la cola de este PC, en QDD_HOME:
// ni en el repositorio (no va a git) ni en el relay, aunque haya un invitado mirando el flujo.
// contrato: flujo#8b2805ee
func TestAsistente_LoDeUnaConversacionQuedaEnQDDHomeYNuncaEnGitNiEnElRelay(t *testing.T) {
	e, r, _ := entornoDePrueba(t)
	const dicho = "validar el RUT del pagador"
	if _, err := ProponerDeAsistente(e.Root, json.RawMessage(`{"op":"historia","flujo":"pago","nodo":"b","quiero":"`+dicho+`"}`), "12:40", ahoraDePrueba); err != nil {
		t.Fatal(err)
	}
	if _, err := Ciclo(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if len(r.resueltas) != 0 || r.fotos != 0 {
		t.Fatalf("al relay no sale nada: %v %d", r.resueltas, r.fotos)
	}
	if !contiene(t, os.Getenv("QDD_HOME"), dicho) {
		t.Fatal("la propuesta queda en la cola de QDD_HOME")
	}
	if contiene(t, e.Root, dicho) {
		t.Fatal("la propuesta no queda en el repositorio")
	}
}

func contiene(t *testing.T, carpeta, texto string) bool {
	t.Helper()
	hay := false
	filepath.WalkDir(carpeta, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if b, _ := os.ReadFile(p); strings.Contains(string(b), texto) {
			hay = true
		}
		return nil
	})
	return hay
}
