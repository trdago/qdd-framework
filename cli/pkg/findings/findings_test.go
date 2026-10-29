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

// FND-061: nada pedía corregir un finding sin estado. QDD lo informa para que
// una persona lo decida y nunca inventa el estado.
func TestPorDecidir(t *testing.T) {
	dir := t.TempDir()
	escribir(t, dir, "FND-001.yaml", "title: a\nstatus: open\n")
	escribir(t, dir, "FND-002.yaml", "title: b\nstatus: RESOLVED\n")
	escribir(t, dir, "FND-003.yaml", "title: c\n")
	escribir(t, dir, "FND-004.yaml", "title: d\nstatus: quizás\n")

	got := PorDecidir(dir)
	if len(got) != 2 || got[0].Archivo != "FND-003.yaml" || got[1].Archivo != "FND-004.yaml" {
		t.Errorf("PorDecidir = %+v; want FND-003 y FND-004", got)
	}
}

func TestNoCanonicos(t *testing.T) {
	dir := t.TempDir()
	escribir(t, dir, "FND-001.yaml", "title: a\nstatus: open\n")
	escribir(t, dir, "FND-002.yaml", "title: b\nstatus: \"RESOLVED\"\n")
	escribir(t, dir, "FND-003.yaml", "title: c\n")

	got := NoCanonicos(dir)
	if len(got) != 1 || got[0].Archivo != "FND-002.yaml" {
		t.Errorf("NoCanonicos = %+v; want solo FND-002", got)
	}
}

// doctor --fix escribe la forma canónica de un estado reconocido y no toca el
// resto del archivo ni los findings cuyo estado tiene que decidir una persona.
func TestEscribirCanonicos(t *testing.T) {
	dir := t.TempDir()
	escribir(t, dir, "FND-001.yaml", "title: a\nstatus: \"RESOLVED\"\naction_taken: |\n    status: closed dentro del texto\n")
	escribir(t, dir, "FND-002.yaml", "title: b\nstatus: closed # cerrado en el sprint 3\n")
	escribir(t, dir, "FND-003.yaml", "title: c\nstatus: quizás\n")
	escribir(t, dir, "FND-004.yaml", "title: d\n")

	cambiados, err := EscribirCanonicos(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cambiados) != 2 {
		t.Errorf("EscribirCanonicos cambió %v; want FND-001 y FND-002", cambiados)
	}
	want := map[string]string{
		"FND-001.yaml": "title: a\nstatus: resolved\naction_taken: |\n    status: closed dentro del texto\n",
		"FND-002.yaml": "title: b\nstatus: resolved\n",
		"FND-003.yaml": "title: c\nstatus: quizás\n",
		"FND-004.yaml": "title: d\n",
	}
	for archivo, contenido := range want {
		data, _ := os.ReadFile(filepath.Join(Dir(dir), archivo))
		if string(data) != contenido {
			t.Errorf("%s =\n%q\nwant\n%q", archivo, data, contenido)
		}
	}
}
