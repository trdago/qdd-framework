package modules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Contrato: cada invariante de una ficha se demuestra con pruebas que lo citan por su id. El id sale del texto: si la
// promesa cambia, la prueba que la citaba deja de contar hasta que alguien la revise y la cite de nuevo.

func escribir(t *testing.T, root, rel, contenido string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(contenido), 0o644); err != nil {
		t.Fatal(err)
	}
}

// contrato: fichas#d9301055
func TestIDInvariante_SaleDelTextoYCambiaSiCambiaLaPromesa(t *testing.T) {
	a := IDInvariante("Nunca borra   un dato.\n")
	if a != IDInvariante("Nunca borra un dato.") {
		t.Fatalf("los espacios no cambian la promesa: %s", a)
	}
	if len(a) != 8 || strings.Trim(a, "0123456789abcdef") != "" {
		t.Fatalf("8 hexadecimales: %q", a)
	}
	if a == IDInvariante("Nunca borra un dato salvo con respaldo.") {
		t.Fatal("otra promesa, otro id")
	}
}

func TestContrato_CadaInvarianteConLasPruebasQueLoCitan(t *testing.T) {
	root := t.TempDir()
	m := Module{ID: "pagos", ObjetivoTecnologico: ObjetivoTecnologico{Invariantes: []string{"Cobra una sola vez.", "Nunca guarda la tarjeta."}}}
	una := IDInvariante("Cobra una sola vez.")
	escribir(t, root, "pkg/pagos/cobro_test.go", "package pagos\n\n// "+CitaInvariante("pagos", una)+"\nfunc TestCobro(t *testing.T) {}\n")
	escribir(t, root, "web/tests/cobro.test.ts", "// "+CitaInvariante("pagos", una)+"\n")
	escribir(t, root, "pkg/pagos/cobro.go", "// "+CitaInvariante("pagos", IDInvariante("Nunca guarda la tarjeta."))+" (no es una prueba)\n")
	escribir(t, root, "node_modules/x/a.test.js", "// "+CitaInvariante("pagos", IDInvariante("Nunca guarda la tarjeta."))+"\n")
	c := Contrato(m, CitasDePruebas(root))
	if len(c) != 2 || c[0].ID != una || len(c[0].Pruebas) != 2 {
		t.Fatalf("la primera, con sus dos pruebas (Go y TS): %+v", c)
	}
	if !strings.HasPrefix(c[0].Pruebas[0], "pkg/pagos/cobro_test.go:") {
		t.Fatalf("ruta relativa con línea: %v", c[0].Pruebas)
	}
	if len(c[1].Pruebas) != 0 {
		t.Fatalf("un archivo que no es prueba, o uno de node_modules, no cuenta: %v", c[1].Pruebas)
	}
	if sin := SinPrueba(c); len(sin) != 1 || sin[0].Texto != "Nunca guarda la tarjeta." {
		t.Fatalf("sin prueba, la segunda: %+v", sin)
	}
}

func TestContexto_CadaInvarianteMuestraSuIdYSiTienePrueba(t *testing.T) {
	root := t.TempDir()
	escribir(t, root, Dir+"/pagos.yaml", `id: pagos
nombre: Pagos
estado: aprobado
rutas: [pkg/pagos]
objetivo_operativo: {problema: p, criterio_de_exito: c}
objetivo_tecnologico:
    arquitectura: a
    invariantes: ["Cobra una sola vez.", "Nunca guarda la tarjeta."]
pruebas: {comandos: [go test ./pkg/pagos]}
`)
	escribir(t, root, "pkg/pagos/cobro_test.go", "// "+CitaInvariante("pagos", IDInvariante("Cobra una sola vez."))+"\n")
	out := BuildContext(ContextInput{Root: root, Paths: []string{"pkg/pagos/cobro.go"}})
	if !strings.Contains(out, "[pagos#"+IDInvariante("Cobra una sola vez.")+"] Cobra una sola vez. (probado en pkg/pagos/cobro_test.go") {
		t.Fatalf("el invariante con su id y su prueba:\n%s", out)
	}
	if !strings.Contains(out, "[pagos#"+IDInvariante("Nunca guarda la tarjeta.")+"] Nunca guarda la tarjeta. (⚠️ sin prueba que lo cite") {
		t.Fatalf("el que no tiene prueba lo dice:\n%s", out)
	}
}
