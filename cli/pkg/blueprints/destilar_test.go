package blueprints

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/modules"
)

// proyectoConPagos es un proyecto con la ficha «pagos» aprobada y el repositorio de la organización registrado.
func proyectoConPagos(t *testing.T, aprobar bool) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, configFile), "blueprints:\n  origen: "+gitOrgRepo(t)+"\n")
	if err := os.MkdirAll(filepath.Join(root, "internal", "pagos"), 0755); err != nil {
		t.Fatal(err)
	}
	m := modules.Module{ID: "pagos", Nombre: "Pagos", Rutas: []string{"internal/pagos"},
		ObjetivoOperativo: modules.ObjetivoOperativo{Problema: "Cobrar sin cobrar dos veces.", CriterioDeExito: "Ningún cobro duplicado.", CasoReal: "Cliente ACME cobrado dos veces en marzo"},
		ObjetivoTecnologico: modules.ObjetivoTecnologico{Arquitectura: "Un cobro idempotente por clave.",
			Decisiones:  []string{"La clave de idempotencia la arma pagos.go con clave()"},
			Invariantes: []string{"Un cobro con la misma clave dentro de 30 min no se repite"}},
		Pruebas: modules.Pruebas{Comandos: []string{"go test ./internal/pagos/..."}}}
	if _, problems := modules.Save(root, m); len(problems) > 0 {
		t.Fatal(problems)
	}
	if !aprobar {
		return root
	}
	if _, problems := modules.Approve(root, "pagos"); len(problems) > 0 {
		t.Fatal(problems)
	}
	return root
}

func TestDestilar_UnBorradorDesdeLaFichaYLaEntrevista(t *testing.T) {
	root := proyectoConPagos(t, true)
	d, err := Destilar(root, "pagos", "cobros", "")
	if err != nil {
		t.Fatal(err)
	}
	b, err := ParseDir(d.Aporte.Carpeta)
	if err != nil {
		t.Fatal(err)
	}
	if b.ID != "cobros" || b.RequiereQDD != RequiereQDDReceta || b.Ficha.ObjetivoTecnologico.Arquitectura != "Un cobro idempotente por clave." || len(b.Ficha.ObjetivoTecnologico.Invariantes) != 1 {
		t.Fatalf("el borrador parte de lo que ya funciona (la ficha aprobada): %+v", b)
	}
	if len(b.Ficha.Rutas) > 0 || b.Ficha.ObjetivoOperativo.CasoReal != "" {
		t.Fatalf("las rutas y el caso real son del proyecto, no de la receta: %+v", b.Ficha)
	}
	entrevista := strings.Join(d.Entrevista, "\n")
	for _, want := range []string{"incluye", "30 min", "pagos.go"} {
		if !strings.Contains(entrevista, want) {
			t.Fatalf("la entrevista pregunta los límites, si cada valor es de negocio o técnico y qué hace cada detalle de la implementación: falta %q en\n%s", want, entrevista)
		}
	}
	if d.Calidad.Defectos == 0 || !strings.Contains(d.Siguiente, "entrevista") {
		t.Fatalf("un borrador recién destilado no está listo, y dice qué sigue: %+v %q", d.Calidad, d.Siguiente)
	}
}

func TestDestilar_NoPisaNiCopiaLoQueNoCorresponde(t *testing.T) {
	root := proyectoConPagos(t, true)
	if _, err := Destilar(root, "pagos", "cobros", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := Destilar(root, "pagos", "cobros", ""); err == nil || !strings.Contains(err.Error(), "contribute") {
		t.Fatalf("un blueprint que ya existe no se pisa: %v", err)
	}
	if _, err := Destilar(proyectoConPagos(t, false), "pagos", "cobros", ""); err == nil || !strings.Contains(err.Error(), "aprob") {
		t.Fatalf("solo se destila lo aprobado: %v", err)
	}
	if _, err := Destilar(root, "no-existe", "x", ""); err == nil {
		t.Fatal("un módulo sin ficha no se destila")
	}
}

func TestEntrevista_TerminaCuandoTodoEstaDelimitado(t *testing.T) {
	if e := Entrevista(CalidadDe(mustParse(t, recetaConLimites))); len(e) != 0 {
		t.Fatalf("con límites y cada valor clasificado no queda nada que preguntar: %v", e)
	}
}

func TestEntrevista_UnValorRepetidoSePreguntaUnaVez(t *testing.T) {
	b := mustParse(t, strings.Replace(recetaConLimites, `invariantes: ["Una tarea nunca desaparece sin cierre"]`, `invariantes: ["Se apaga a los 120 s", "Sin pulsos por 120 s, se apaga"]`, 1))
	if e := Entrevista(CalidadDe(b)); len(e) != 1 {
		t.Fatalf("una pregunta por valor: %v", e)
	}
}
