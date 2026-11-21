package flujo

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// Un id que trae una ruta («../fuera») no lee ni escribe fuera de la carpeta de flujos, por ninguna puerta exportada.

// archivosFuera lista lo que hay en root fuera de las carpetas de flujos, con su contenido.
func archivosFuera(t *testing.T, root string) map[string]string {
	t.Helper()
	vistos := map[string]string{}
	propias := []string{filepath.Join(root, Carpeta), filepath.Join(root, CarpetaLocal)}
	err := filepath.WalkDir(root, func(camino string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		for _, propia := range propias {
			if strings.HasPrefix(camino, propia+string(filepath.Separator)) {
				return nil
			}
		}
		datos, _ := os.ReadFile(camino)
		vistos[camino] = string(datos)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return vistos
}

// contrato: flujo#d7ab8543
func TestID_ConRutaNoSaleDeLaCarpeta(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	const id = "../fuera"
	// Lo que leería un id con ruta: <flujos>/../fuera.* queda en .qdd/project, fuera de la carpeta de flujos.
	afuera := filepath.Join(root, ".qdd", "project")
	plantados := map[string]string{
		"fuera.historial.jsonl": `{"rev":9,"en":"x","autor":"intruso","resumen":"secreto de afuera"}` + "\n",
		"fuera.yaml":            "id: fuera\nnombre: Afuera\nnodos:\n- {id: a, tipo: paso, nombre: A}\n",
	}
	for nombre, contenido := range plantados {
		if err := os.WriteFile(filepath.Join(afuera, nombre), []byte(contenido), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	antes := archivosFuera(t, root)

	if h := Historial(root, id); len(h) != 0 {
		t.Errorf("Historial leyó fuera de la carpeta de flujos: %+v", h)
	}
	if _, err := Cargar(root, id); err == nil {
		t.Error("Cargar aceptó un id con ruta")
	}
	if _, err := CargarVersion(root, id, 1); err == nil {
		t.Error("CargarVersion aceptó un id con ruta")
	}
	if l := PorEnviar(root, id); len(l) != 0 {
		t.Errorf("PorEnviar leyó fuera: %v", l)
	}
	if PuedeDeshacer(root, id) {
		t.Error("PuedeDeshacer leyó fuera")
	}
	if m, n := Enviar(root, id); n != 0 || m != "" {
		t.Errorf("Enviar leyó fuera: %q", m)
	}
	escrituras := map[string]error{}
	_, _, escrituras["Editar"] = Editar(root, id, DePersona, ahoraDePrueba, Quitar("a"))
	escrituras["Devolver"] = Devolver(root, id, 1)
	escrituras["Deshacer"] = Deshacer(root, id)
	_, escrituras["CerrarVersion"] = CerrarVersion(root, id, "v", ahoraDePrueba)
	_, escrituras["Volver"] = Volver(root, id, 1, DePersona, ahoraDePrueba)
	escrituras["PedirDetalle"] = PedirDetalle(root, id, "a", ahoraDePrueba)
	escrituras["RepararHistorial"] = RepararHistorial(root, id, ahoraDePrueba)
	for nombre, err := range escrituras {
		if err == nil {
			t.Errorf("%s aceptó un id con ruta", nombre)
		}
	}
	if p := RevisarHistorial(root, id); p != "" {
		t.Errorf("RevisarHistorial leyó fuera: %s", p)
	}

	despues := archivosFuera(t, root)
	if len(despues) != len(antes) {
		t.Fatalf("se creó o borró algo fuera de la carpeta de flujos:\nantes %v\ndespués %v", antes, despues)
	}
	for camino, contenido := range antes {
		if despues[camino] != contenido {
			t.Errorf("%s cambió: %q", camino, despues[camino])
		}
	}
}
