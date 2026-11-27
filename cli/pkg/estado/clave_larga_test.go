package estado

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// FND-334 (desajuste 4 de docs/qdd-movil-comunicacion.md): la clave de un proyecto (<carpeta>-<hash de 12>) no tenía
// tope, y el relay exige ^[a-z0-9-]{1,80}$ en acciones, respuestas, ejecutar y push: una carpeta de más de 67
// caracteres dejaba ese proyecto sin acciones ni push desde el teléfono. La clave se acorta a 80 conservando el hash, y
// qdd doctor --fix renombra lo local que quedó con la clave larga (opción recomendada, de acuerdo con qdd-movile).

var claveRelay = regexp.MustCompile(`^[a-z0-9-]{1,80}$`)

func proyectoLargo(t *testing.T) string {
	t.Helper()
	base, _ := filepath.EvalSymlinks(t.TempDir())
	root := filepath.Join(base, strings.Repeat("Carpeta-Larga ", 7))
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func claveDeRuta(ruta string) string {
	return strings.TrimSuffix(filepath.Base(ruta), ".json")
}

func TestClaveLocal_UnaCarpetaLargaNoPasaDe80YConservaElHash_FND334(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := proyectoLargo(t)
	ruta, err := RutaLocal(root, "pedidos")
	if err != nil {
		t.Fatal(err)
	}
	clave := claveDeRuta(ruta)
	if !claveRelay.MatchString(clave) || strings.Contains(clave, "--") {
		t.Fatalf("la clave cabe en el relay (%d caracteres): %s", len(clave), clave)
	}
	larga := claveDeRuta(rutaLargaDe(t, root))
	if clave[len(clave)-13:] != larga[len(larga)-13:] || !strings.HasPrefix(larga, strings.TrimSuffix(clave[:len(clave)-13], "-")) {
		t.Fatalf("se acorta la carpeta y queda el mismo hash: %s ← %s", clave, larga)
	}
}

func TestClaveLocal_UnaCarpetaCortaQuedaIgual_FND334(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	base, _ := filepath.EvalSymlinks(t.TempDir())
	root := filepath.Join(base, "Mi Proyecto")
	os.MkdirAll(root, 0o755)
	ruta, _ := RutaLocal(root, "pedidos")
	if !strings.HasPrefix(claveDeRuta(ruta), "mi-proyecto-") || ruta != rutaLargaDe(t, root) {
		t.Fatalf("una clave que ya cabía no cambia, y sus colas siguen donde estaban: %s", ruta)
	}
	if n := ClavesLargas(root); len(n) != 0 {
		t.Fatalf("nada que renombrar: %v", n)
	}
}

func TestClaveLocal_DoctorRenombraLoQueQuedoConLaClaveLarga_FND334(t *testing.T) {
	home := t.TempDir()
	t.Setenv("QDD_HOME", home)
	root := proyectoLargo(t)
	larga := claveDeRuta(rutaLargaDe(t, root))
	nueva, _ := RutaLocal(root, "pedidos")
	escribir := func(rel, contenido string) {
		p := filepath.Join(home, "claude", rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(contenido), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	escribir(filepath.Join("pedidos", larga+".json"), "cola")
	escribir(filepath.Join("verificaciones", larga+".json.json"), "verificaciones")
	escribir(filepath.Join("goal", larga+".json"), "goal viejo")
	escribir(filepath.Join("goal", claveDeRuta(nueva)+".json"), "goal nuevo")
	escribir(filepath.Join("pedidos", "otro-proyecto-0123456789ab.json"), "ajeno")

	if n := ClavesLargas(root); len(n) != 3 {
		t.Fatalf("doctor ve los 3 archivos con la clave larga: %v", n)
	}
	hecho, err := MigrarClavesLargas(root)
	if err != nil {
		t.Fatal(err)
	}
	leer := func(rel string) string {
		b, _ := os.ReadFile(filepath.Join(home, "claude", rel))
		return string(b)
	}
	if leer(filepath.Join("pedidos", filepath.Base(nueva))) != "cola" || leer(filepath.Join("verificaciones", claveDeRuta(nueva)+".json.json")) != "verificaciones" {
		t.Fatalf("la cola y las verificaciones pasan a la clave nueva: %v", hecho)
	}
	if leer(filepath.Join("goal", claveDeRuta(nueva)+".json")) != "goal nuevo" || leer(filepath.Join("goal", larga+".json")) != "goal viejo" {
		t.Fatal("si ya hay un archivo con la clave nueva, no se pisa: el viejo queda donde estaba")
	}
	if leer(filepath.Join("pedidos", "otro-proyecto-0123456789ab.json")) != "ajeno" {
		t.Fatal("lo de otro proyecto no se toca")
	}
	if n := ClavesLargas(root); len(n) != 1 {
		t.Fatalf("queda solo el que choca, a la vista de doctor: %v", n)
	}
}

// rutaLargaDe es la ruta con la clave de antes del arreglo, sin tope: el nombre de la carpeta entero.
func rutaLargaDe(t *testing.T, root string) string {
	t.Helper()
	ruta, err := rutaLocalLarga(os.Getenv("QDD_HOME"), ClaveLocal(root), "pedidos")
	if err != nil {
		t.Fatal(err)
	}
	return ruta
}
