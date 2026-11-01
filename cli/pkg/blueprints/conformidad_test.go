package blueprints

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/modules"
)

func TestConformidad_CadaEscenarioDeLaVarianteTieneSuPrueba(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "src", "panel", "panel_test.go"), "func TestPanel_SeVenLasMasNuevas_P_01(t *testing.T) {}\n")
	writeFile(t, filepath.Join(root, "src", "panel", "panel.go"), "// P-02: el aviso (un comentario del código no es una prueba)\n")
	writeFile(t, filepath.Join(root, "src", "panel", "viejo_test.go"), "// P-020 es otro escenario\n")
	b, rutas := mustParse(t, recetaConLimites), []string{"src/panel"}

	c := ConformidadDe(root, "panel", b, modules.Implementacion{Blueprint: "panel", Respuestas: map[string]string{"avisos": "no"}}, rutas)
	if strings.Join(c.Escenarios, ",") != "P-01" || len(c.Faltan) != 0 {
		t.Fatalf("con sus respuestas, solo los escenarios de su variante; el id con «_» en el nombre de la prueba vale: %+v", c)
	}
	c = ConformidadDe(root, "panel", b, modules.Implementacion{Blueprint: "panel"}, rutas)
	if strings.Join(c.Escenarios, ",") != "P-01,P-02" || strings.Join(c.Faltan, ",") != "P-02" {
		t.Fatalf("sin respuestas (la implementación de origen), todas las opciones; ni el código ni P-020 cuentan como prueba de P-02: %+v", c)
	}
	writeFile(t, filepath.Join(root, "src", "panel", "tests", "aviso.test.ts"), "test('P-02 el aviso no quita el foco', () => {})\n")
	if c = ConformidadDe(root, "panel", b, modules.Implementacion{Blueprint: "panel"}, rutas); len(c.Faltan) != 0 || !strings.Contains(LineaConformidad(c), "2/2") {
		t.Fatalf("con su prueba, conforme: %+v %q", c, LineaConformidad(c))
	}
}

func TestConformar_LoAplicadoYLoQueUnaFichaImplementa(t *testing.T) {
	root := t.TempDir()
	if _, problems := AddManaged(AddInput{Root: root, Origen: gitOrgRepo(t), Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	for _, id := range []string{"origen", "cliente"} {
		writeFile(t, filepath.Join(root, "internal", id, "auth_test.go"), "func TestLogin_AUTH_01(t *testing.T) {}\n")
		m := modules.Module{ID: id, Nombre: id, Rutas: []string{"internal/" + id},
			ObjetivoOperativo:   modules.ObjetivoOperativo{Problema: "Entrar.", CriterioDeExito: "Entra."},
			ObjetivoTecnologico: modules.ObjetivoTecnologico{Arquitectura: "Un login."},
			Pruebas:             modules.Pruebas{Comandos: []string{"go test ./internal/..."}}}
		if id == "origen" {
			m.Implementa = []modules.Implementacion{{Blueprint: "auth"}}
		}
		if _, problems := modules.Save(root, m); len(problems) > 0 {
			t.Fatal(problems)
		}
	}
	if _, err := saveApplication(root, Aplicacion{Blueprint: "auth", Version: "1.0.0", Modulo: "cliente", Respuestas: map[string]string{"identificador": "correo", "recuperacion": "correo"}}); err != nil {
		t.Fatal(err)
	}
	cs, err := Conformar(root)
	if err != nil || len(cs) != 2 {
		t.Fatalf("una conformidad por módulo que aplica o implementa un blueprint: %+v %v", cs, err)
	}
	porModulo := map[string]Conformidad{}
	for _, c := range cs {
		porModulo[c.Modulo] = c
	}
	if c := porModulo["cliente"]; len(c.Faltan) != 0 || c.Blueprint != "auth@1.0.0" {
		t.Fatalf("lo aplicado se mide con sus respuestas (correo): no le toca RUT-01: %+v", c)
	}
	if c := porModulo["origen"]; !strings.Contains(strings.Join(c.Faltan, ","), "RUT-01") {
		t.Fatalf("la implementación de origen cubre todas las opciones: le falta RUT-01: %+v", c)
	}
}

func TestConformidad_UnEscenarioFueraDelCodigoSeDiceConSuMotivo(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "src", "panel", "panel_test.go"), "func TestPanel_P_01(t *testing.T) {}\n")
	b, rutas := mustParse(t, recetaConLimites), []string{"src/panel"}
	impl := modules.Implementacion{Blueprint: "panel", Fuera: map[string]string{"P-02": "el aviso lo da el sistema operativo"}}
	c := ConformidadDe(root, "panel", b, impl, rutas)
	linea := LineaConformidad(c)
	if len(c.Faltan) != 0 || !strings.HasPrefix(linea, "✅") || !strings.Contains(linea, "1/1") || !strings.Contains(linea, "fuera P-02 (el aviso lo da el sistema operativo)") {
		t.Fatalf("lo que no hace este código no falta, pero siempre se ve con su motivo: %+v %q", c, linea)
	}
	impl.Fuera = map[string]string{"P-02": " ", "X-99": "no existe"}
	c = ConformidadDe(root, "panel", b, impl, rutas)
	if strings.Join(c.Faltan, ",") != "P-02,X-99" {
		t.Fatalf("sin motivo, o de un escenario que no existe, no vale: %+v", c)
	}
}
