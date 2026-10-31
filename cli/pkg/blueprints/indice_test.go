package blueprints

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestIndice_ResumeSinPerderLoQueSeDecide(t *testing.T) {
	i := IndiceDe(mustParse(t, authBlueprint))
	if i.Ref != "auth@1.0.0" || i.Nombre != "Usuarios y acceso" {
		t.Fatalf("identidad: %+v", i)
	}
	var ids []string
	for _, p := range i.Preguntas {
		ids = append(ids, p.ID)
	}
	if !reflect.DeepEqual(ids, []string{"identificador", "recuperacion", "proveedor_whatsapp"}) {
		t.Fatalf("preguntas en orden: %v", ids)
	}
	if !reflect.DeepEqual(i.Preguntas[0].Opciones, []string{"correo", "rut"}) || i.Preguntas[2].Si != "recuperacion = whatsapp" {
		t.Fatalf("opciones y condición de cada pregunta: %+v", i.Preguntas)
	}
	if i.Invariantes != 3 || !reflect.DeepEqual(i.Escenarios, []string{"AUTH-01", "RUT-01"}) {
		t.Fatalf("cuenta las invariantes de base y de opciones, y nombra cada escenario: %d %v", i.Invariantes, i.Escenarios)
	}
	b, _ := json.Marshal(i)
	if len(b) > 600 {
		t.Fatalf("el índice es un resumen, no el blueprint: %d bytes", len(b))
	}
}

func TestLineaIndice_UnaLineaCorta(t *testing.T) {
	i := IndiceDe(mustParse(t, authBlueprint))
	i.Modulos = []string{"auth"}
	l := LineaIndice(i)
	if strings.Contains(l, "\n") || utf8.RuneCountInString(l) > MaxLineaIndice {
		t.Fatalf("una línea de %d caracteres como mucho: %q", MaxLineaIndice, l)
	}
	for _, want := range []string{"auth@1.0.0", "Usuarios y acceso", "❓3", "🔒3", "✓2", "auth"} {
		if !strings.Contains(l, want) {
			t.Fatalf("falta %q en %q", want, l)
		}
	}
}

func TestSeccion_DetalleAPedido(t *testing.T) {
	b := mustParse(t, authBlueprint)
	inv, err := Seccion(b, "invariantes")
	if err != nil || !strings.Contains(inv, "nunca por nombre de rol") || !strings.Contains(inv, "rut → El RUT se valida") {
		t.Fatalf("invariantes de base y de cada opción: %q %v", inv, err)
	}
	esc, _ := Seccion(b, "escenarios")
	if !strings.Contains(esc, "AUTH-01") || !strings.Contains(esc, "RUT-01") || strings.Contains(esc, "¿Con qué") {
		t.Fatalf("solo los escenarios: %q", esc)
	}
	pre, _ := Seccion(b, "preguntas")
	if !strings.Contains(pre, "proveedor_whatsapp") || strings.Contains(pre, "AUTH-01") {
		t.Fatalf("solo las preguntas: %q", pre)
	}
	if _, err := Seccion(b, "otra"); err == nil || !strings.Contains(err.Error(), "preguntas") {
		t.Fatalf("una sección desconocida dice cuáles hay: %v", err)
	}
}

func TestIndice_LineaDeCadaSeccionEnElManifiesto(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ManifestName), authBlueprint)
	b, err := ParseDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	lineas := strings.Split(authBlueprint, "\n")
	i := IndiceDe(b)
	for seccion, clave := range map[string]string{"vocabulario": "vocabulario:", "preguntas": "preguntas:", "invariantes": "invariantes: [\"Autorización", "escenarios": "conformidad:"} {
		n := i.Secciones[seccion]
		if n < 1 || !strings.HasPrefix(strings.TrimSpace(lineas[n-1]), clave) {
			t.Fatalf("la sección %s apunta a su línea del YAML (%d): %v", seccion, n, i.Secciones)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "index.json")); err == nil {
		t.Fatal("el índice se calcula; no deja archivos en la carpeta del blueprint (cambiaría su sha256)")
	}
}

func TestMapaDeSecciones_DondeEditarSinLeerTodo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ManifestName), authBlueprint)
	m := MapaDeSecciones(dir)
	if !strings.HasPrefix(m, "blueprint.yaml: vocabulario L5 · preguntas L8") || !strings.Contains(m, "escenarios L") || strings.Contains(m, "\n") {
		t.Fatalf("una línea con cada sección y su línea, en orden: %q", m)
	}
	if MapaDeSecciones(t.TempDir()) != "" {
		t.Fatal("un blueprint nuevo, sin manifiesto, no tiene mapa")
	}
}
