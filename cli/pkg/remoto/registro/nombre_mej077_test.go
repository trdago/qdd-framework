package registro

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// MEJ-077 (FND-227): en QDD Móvil 130 la Bandeja mostraba el proyecto de un PC Windows como «Sistema%20de%20gesti%C3%B3n…»:
// la ruta llegó al registro codificada como URL (como en un file:// del editor) y el nombre salía de filepath.Base. Una
// ruta codificada cuya carpeta no existe así se guarda decodificada, y el nombre se muestra decodificado.

func codificada(ruta string) string {
	partes := strings.Split(filepath.ToSlash(ruta), "/")
	for i, p := range partes {
		partes[i] = url.PathEscape(p)
	}
	return filepath.FromSlash(strings.Join(partes, "/"))
}

func TestAnotar_UnaRutaCodificadaComoURLSeGuardaDecodificada_MEJ077(t *testing.T) {
	homeTemporal(t)
	real := carpeta(t, "Sistema de gestión de procesos")
	if codificada(real) == real {
		t.Fatal("la prueba necesita una ruta que cambie al codificarla")
	}
	if err := Anotar(codificada(real), time.Now()); err != nil {
		t.Fatal(err)
	}
	reg, err := Leer()
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.Proyectos) != 1 || reg.Proyectos[0].Ruta != real {
		t.Fatalf("se guarda la carpeta real, decodificada: %+v", reg.Proyectos)
	}
}

func TestAnotar_LaRutaRealReemplazaUnaCodificadaQueYaEstaba_MEJ077(t *testing.T) {
	home := homeTemporal(t)
	real := carpeta(t, "Sistema de gestión")
	antes := Registro{Formato: Formato, Proyectos: []Proyecto{{Ruta: codificada(real), Clave: "sistema-x", Visto: time.Now().Add(-2 * time.Hour)}}}
	data, _ := json.Marshal(antes)
	if err := os.WriteFile(filepath.Join(home, "proyectos.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Anotar(real, time.Now()); err != nil {
		t.Fatal(err)
	}
	reg, _ := Leer()
	if len(reg.Proyectos) != 1 || reg.Proyectos[0].Ruta != real {
		t.Fatalf("el teléfono no ve el mismo proyecto dos veces: %+v", reg.Proyectos)
	}
}

func TestNombreDe_DecodificaSoloSiLaCarpetaNoExisteAsi_MEJ077(t *testing.T) {
	real := carpeta(t, "Sistema de gestión")
	if got := NombreDe(codificada(real)); got != "Sistema de gestión" {
		t.Fatalf("una ruta codificada que no existe así se nombra decodificada: %q", got)
	}
	conPorcentaje := carpeta(t, "100%20listo")
	if got := NombreDe(conPorcentaje); got != "100%20listo" {
		t.Fatalf("una carpeta real con %% en el nombre se nombra tal cual: %q", got)
	}
	if got := NombreDe(filepath.Join(t.TempDir(), "roto%zz")); got != "roto%zz" {
		t.Fatalf("lo que no se puede decodificar queda tal cual: %q", got)
	}
}
