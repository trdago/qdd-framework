package integration

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// FND-126: el bloque de reglas solo se escribía si faltaba el archivo: al actualizar qdd, ningún proyecto recibía las
// reglas nuevas.

const huellaBloque = "bc3b6157b3e69ee7b8514fcb7a571261caf5c070c17f1db3c7d1a49a3aed1570"

func TestBloque_CambiarSuTextoObligaASubirLaRevision_FND126(t *testing.T) {
	suma := sha256.Sum256([]byte(QDDCommandProtocol))
	if h := hex.EncodeToString(suma[:]); h != huellaBloque {
		t.Fatalf("cambió el texto del bloque: sube «QCP block revision» (hoy %d), para que qdd doctor lo renueve en los proyectos, y deja aquí la huella %s", revisionDe(QDDCommandProtocol), h)
	}
}

func TestBloquesViejos_NombraSoloLosDeUnaRevisionAnterior_FND126(t *testing.T) {
	root := t.TempDir()
	if viejos := BloquesViejos(root); len(viejos) != 0 {
		t.Fatalf("sin archivos no hay nada viejo: %v", viejos)
	}
	escribir := func(rel, contenido string) {
		t.Helper()
		ruta := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(ruta, []byte(contenido), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	escribir("AGENTS.md", "Mis notas\n\n"+markerBegin+"\n# sin revisión: de antes de FND-126\n"+markerEnd+"\n")
	escribir(".clauderc", strings.TrimSpace(QDDCommandProtocol)+"\n")
	escribir(".cursorrules", "reglas propias, sin bloque de qdd\n")
	escribir(".agentrules", markerBegin+"\n# QCP block revision: 99999\n"+markerEnd+"\n")
	if viejos := BloquesViejos(root); len(viejos) != 1 || viejos[0] != "AGENTS.md" {
		t.Fatalf("solo el bloque sin revisión: ni el actual, ni el archivo sin bloque, ni el de un qdd más nuevo: %v", viejos)
	}

	escribir(filepath.Join(".qdd", "config.yaml"), "x: 1\n")
	if err := NewIntegrationManager().SyncAll(root); err != nil {
		t.Fatal(err)
	}
	if viejos := BloquesViejos(root); len(viejos) != 0 {
		t.Fatalf("tras sincronizar, ninguno viejo: %v", viejos)
	}
	data, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Mis notas") || !strings.Contains(string(data), "COMMANDS FOR THE HUMAN") {
		t.Fatalf("renueva el bloque y conserva lo demás: %q", data)
	}
	data, err = os.ReadFile(filepath.Join(root, ".agentrules"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "revision: 99999") {
		t.Fatalf("un qdd más viejo no pisa el bloque de uno más nuevo: %q", data)
	}
}
