package estado

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// FND-190: el panel relee el estado varias veces por segundo, y cada relectura lo cargaba unas cinco veces: con un
// estado.yaml de 115 KB, ~30 ms de CPU por relectura solo en el YAML.
const estadoRecordado = `formato: 3
etapas: [Pruebas, Publicación]
ciclos:
  movil: [Diseño, Tienda]
objetivos:
  - id: OBJ-1
    nombre: Panel liviano
    criterio: menos de 1 % de CPU
    estado: abierto
frentes:
  - id: T-1
    nombre: Seguir sin despertar
    estado: abierto
    objetivo: OBJ-1
    depende_de: [T-0]
    celdas:
      Pruebas: {estado: "⏳", bloqueo: "¿Mides?", de_quien: persona}
`

func escribirEstado(t *testing.T, root, contenido string) {
	t.Helper()
	ruta := filepath.Join(root, Ruta)
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte(contenido), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_ParseaUnaVezMientrasElArchivoNoCambia(t *testing.T) {
	root := t.TempDir()
	escribirEstado(t, root, estadoRecordado)
	antes := parseos.Load()
	for range 5 {
		if _, err := Load(root); err != nil {
			t.Fatal(err)
		}
	}
	if n := parseos.Load() - antes; n != 1 {
		t.Fatalf("5 cargas del mismo archivo parsearon %d veces, quiere 1", n)
	}
	escribirEstado(t, root, estadoRecordado+"actualizado: hoy\n")
	e, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if e.Actualizado != "hoy" {
		t.Fatalf("después de cambiar el archivo, Load devolvió el estado viejo: %q", e.Actualizado)
	}
}

// Quien carga puede modificar lo que recibe (set, cerrar) y no guardarlo si Check lo rechaza: eso no puede llegar a la
// próxima carga.
func TestLoad_ModificarLoCargadoNoCambiaLaProximaCarga(t *testing.T) {
	root := t.TempDir()
	escribirEstado(t, root, estadoRecordado)
	primera, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	primera.Etapas[0] = "x"
	primera.Ciclos["movil"][0] = "x"
	primera.Objetivos[0].Nombre = "x"
	primera.Frentes[0].DependeDe[0] = "x"
	primera.Frentes[0].Celdas["Pruebas"] = Celda{Estado: Falla}
	primera.Frentes = append(primera.Frentes, Frente{ID: "T-2"})
	segunda, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	quiere, err := Parse([]byte(estadoRecordado))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(segunda, quiere) {
		t.Fatalf("la segunda carga vio lo que se modificó en la primera:\n%+v", segunda)
	}
}

// Un campo nuevo con un slice o un mapa que clonar olvide compartiría memoria entre cargas: esta prueba lo detecta sin
// tener que acordarse de ella.
func TestClonar_NoComparteNingunSliceNiMapa(t *testing.T) {
	e, err := Parse([]byte(estadoRecordado))
	if err != nil {
		t.Fatal(err)
	}
	compartidos := []string{}
	buscarCompartidos(reflect.ValueOf(e), reflect.ValueOf(e.clonar()), "Estado", &compartidos)
	if len(compartidos) > 0 {
		t.Fatalf("clonar comparte memoria en: %v", compartidos)
	}
}

func buscarCompartidos(a, b reflect.Value, ruta string, compartidos *[]string) {
	switch a.Kind() {
	case reflect.Struct:
		for i := range a.NumField() {
			buscarCompartidos(a.Field(i), b.Field(i), ruta+"."+a.Type().Field(i).Name, compartidos)
		}
	case reflect.Slice:
		if a.Len() > 0 && a.Pointer() == b.Pointer() {
			*compartidos = append(*compartidos, ruta)
		}
		for i := range a.Len() {
			buscarCompartidos(a.Index(i), b.Index(i), ruta+"[]", compartidos)
		}
	case reflect.Map:
		if a.Len() > 0 && a.Pointer() == b.Pointer() {
			*compartidos = append(*compartidos, ruta)
		}
		for _, k := range a.MapKeys() {
			buscarCompartidos(a.MapIndex(k), b.MapIndex(k), ruta+"[k]", compartidos)
		}
	case reflect.Pointer:
		if !a.IsNil() && a.Pointer() == b.Pointer() {
			*compartidos = append(*compartidos, ruta)
		}
	}
}
