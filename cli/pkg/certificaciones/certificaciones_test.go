package certificaciones

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizar(t *testing.T) {
	cases := map[string]Estado{
		"certified": Cumple, "PASSED": Cumple, " pass ": Cumple, "aprobada": Cumple,
		"failed": NoCumple, "FAIL": NoCumple, "rechazada": NoCumple,
		"pending": Pendiente, "draft": Pendiente, "borrador": Pendiente,
		"": SinEstado, "quizás": SinEstado,
	}
	for in, want := range cases {
		if got := Normalizar(in); got != want {
			t.Errorf("Normalizar(%q) = %q; want %q", in, got, want)
		}
	}
}

func escribir(t *testing.T, dir, carpeta, nombre, contenido string) {
	t.Helper()
	path := filepath.Join(dir, ".qdd", carpeta, "certification", nombre)
	os.MkdirAll(filepath.Dir(path), 0755)
	if err := os.WriteFile(path, []byte(contenido), 0644); err != nil {
		t.Fatal(err)
	}
}

// FND-060: el dashboard mezclaba el catálogo con lo que el proyecto adoptó y
// daba por cumplida toda certificación que no decía su estado.
func TestLeer_DistingueCatalogoYProyecto(t *testing.T) {
	dir := t.TempDir()
	escribir(t, dir, "core", "CERT-A.yaml", "id: CERT-A\ntitle: Estándar del catálogo\nstatus: certified\n")
	escribir(t, dir, "core", "CERT-B.yaml", "id: CERT-B\ntitle: Versión del catálogo\nstatus: certified\n")
	escribir(t, dir, "project", "CERT-B.yaml", "id: CERT-B\nname: Adoptada por el proyecto\nstatus: PASSED\nversion: \"1.2.0\"\n")
	escribir(t, dir, "project", "CERT-C.yaml", "title: Sin estado declarado\n")
	escribir(t, dir, "project", "notas.md", "status: certified\n")

	cs, err := Leer(dir)
	if err != nil {
		t.Fatal(err)
	}
	por := map[string]Certificacion{}
	for _, c := range cs {
		por[c.ID] = c
	}
	if len(cs) != 3 {
		t.Fatalf("Leer = %d certificaciones %+v; want 3 (CERT-B una sola vez)", len(cs), cs)
	}
	if a := por["CERT-A"]; a.Origen != Catalogo || a.Nombre != "Estándar del catálogo" {
		t.Errorf("CERT-A = %+v; want del catálogo con su título", a)
	}
	if b := por["CERT-B"]; b.Origen != Proyecto || b.Estado != Cumple || b.Nombre != "Adoptada por el proyecto" || b.Version != "1.2.0" {
		t.Errorf("CERT-B = %+v; want la del proyecto, cumple, con su nombre y versión", b)
	}
	if c := por["CERT-C"]; c.Origen != Proyecto || c.Estado != SinEstado || c.Nombre != "Sin estado declarado" {
		t.Errorf("CERT-C = %+v; want del proyecto, sin estado, id desde el archivo", c)
	}
}

func TestContar(t *testing.T) {
	cs := []Certificacion{
		{Origen: Proyecto, Estado: Cumple},
		{Origen: Proyecto, Estado: NoCumple},
		{Origen: Proyecto, Estado: Pendiente},
		{Origen: Proyecto, Estado: SinEstado},
		{Origen: Catalogo, Estado: Cumple},
	}
	want := Conteo{Cumplen: 1, NoCumplen: 1, Pendientes: 1, SinEstado: 1, Disponibles: 1}
	if got := Contar(cs); got != want {
		t.Errorf("Contar = %+v; want %+v (el catálogo no cuenta como cumplida)", got, want)
	}
}

func TestLeer_SinCarpetas(t *testing.T) {
	cs, err := Leer(t.TempDir())
	if err != nil || len(cs) != 0 {
		t.Errorf("Leer sin carpetas = %v, %v; want vacío sin error", cs, err)
	}
}

func TestPorDecidir(t *testing.T) {
	dir := t.TempDir()
	escribir(t, dir, "core", "CERT-A.yaml", "id: CERT-A\n")
	escribir(t, dir, "project", "CERT-B.yaml", "id: CERT-B\nstatus: certified\n")
	escribir(t, dir, "project", "CERT-C.yaml", "id: CERT-C\nstatus: quizás\n")

	got := PorDecidir(dir)
	if len(got) != 1 || got[0].ID != "CERT-C" {
		t.Errorf("PorDecidir = %+v; want solo CERT-C (el catálogo no declara estado del proyecto)", got)
	}
}
