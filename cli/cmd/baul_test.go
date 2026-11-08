package cmd

import (
	"os"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// RFC 0023: guardar pide el valor en la terminal de la persona; sin terminal (la herramienta Bash de una IA, un pedido,
// el teléfono) se niega y dice dónde correrlo.
func TestBaulGuardar_SinTerminalSeNiega(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "entrada")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("cf-SECRETO\n")
	f.Seek(0, 0)
	defer f.Close()
	_, err = leerValorOculto(f, "cloudflare")
	if err == nil || !strings.Contains(err.Error(), "tu terminal") {
		t.Fatalf("sin terminal no se lee el valor: %v", err)
	}
}

func TestDoctorBaul_SinBaulNoApareceYConTokensSi(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	if g := buildBaulChecks(root); len(g.Items) != 0 {
		t.Fatalf("sin baúl ni .env el grupo no aparece: %+v", g.Items)
	}
	os.WriteFile(root+"/.env", []byte("GITHUB_TOKEN=ghp_valor_de_prueba\n"), 0o600)
	g := buildBaulChecks(root)
	if len(g.Items) != 1 || !g.Items[0].Success || strings.Contains(g.Items[0].Error, "ghp_valor") {
		t.Fatalf("un aviso sin el valor: %+v", g.Items)
	}
}
