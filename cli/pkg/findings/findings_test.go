package findings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizar(t *testing.T) {
	cases := map[string]Estado{
		"open": Abierto, "OPEN": Abierto, " in-progress ": Abierto, "pendiente": Abierto,
		"resolved": Resuelto, "RESOLVED": Resuelto, "closed": Resuelto, "fixed": Resuelto, "resuelto": Resuelto,
		"wontfix": Descartado, "descartado": Descartado,
		"": SinEstado, "quizás": SinEstado,
	}
	for in, want := range cases {
		if got := Normalizar(in); got != want {
			t.Errorf("Normalizar(%q) = %q; want %q", in, got, want)
		}
	}
}

func escribir(t *testing.T, dir, nombre, contenido string) {
	t.Helper()
	path := filepath.Join(dir, ".qdd", "project", "findings", nombre)
	os.MkdirAll(filepath.Dir(path), 0755)
	if err := os.WriteFile(path, []byte(contenido), 0644); err != nil {
		t.Fatal(err)
	}
}

// FND-056: cada lector interpretaba el estado a su manera. Un finding sin
// status no es abierto ni resuelto: se informa aparte para que alguien lo decida.
func TestContarEn(t *testing.T) {
	dir := t.TempDir()
	escribir(t, dir, "FND-001.yaml", "title: a\nstatus: open\n")
	escribir(t, dir, "FND-002.yaml", "title: b\nstatus: RESOLVED\n")
	escribir(t, dir, "FND-003.yaml", "title: c\nstatus: \"resolved\"\n")
	escribir(t, dir, "FND-004.yaml", "title: d\naction_taken: se corrigió\n")
	escribir(t, dir, "FND-005.yaml", "title: e\nstatus: wontfix\n")
	escribir(t, dir, "notas.md", "status: open\n")

	got := ContarEn(dir)
	want := Conteo{Abiertos: 1, Resueltos: 2, Descartados: 1, SinEstado: 1}
	if got != want {
		t.Errorf("ContarEn = %+v; want %+v", got, want)
	}
}

func TestLeer_SinCarpeta(t *testing.T) {
	fs, err := Leer(t.TempDir())
	if err != nil || len(fs) != 0 {
		t.Errorf("Leer sin carpeta = %v, %v; want vacío sin error", fs, err)
	}
}

func TestLeer_Campos(t *testing.T) {
	dir := t.TempDir()
	escribir(t, dir, "FND-007-X.yaml", "id: FND-007\ntitle: Título real\ndescription: Qué pasó\nstatus: open\nseverity: HIGH\n")
	fs, _ := Leer(dir)
	if len(fs) != 1 {
		t.Fatalf("Leer = %v", fs)
	}
	f := fs[0]
	if f.Archivo != "FND-007-X.yaml" || f.ID != "FND-007" || f.Titulo != "Título real" || f.Descripcion != "Qué pasó" || f.Estado != Abierto || f.Severidad != "HIGH" {
		t.Errorf("finding = %+v", f)
	}
}
