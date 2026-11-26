package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/mejoras"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

func proyectoMejoras(t *testing.T, nombre string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), nombre)
	if err := os.MkdirAll(filepath.Join(root, ".qdd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := estado.Init(root, []string{"Código", "PR"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestMejoras_UnaSesionReportaAOtroProyectoYSuCicloLoEntrega(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	viejas := mejorasFuentes
	mejorasFuentes = func() mejoras.Fuentes { return mejoras.Fuentes{} }
	t.Cleanup(func() { mejorasFuentes = viejas })
	qdd, sfrc := proyectoMejoras(t, "qdd"), proyectoMejoras(t, "sfrc")
	if err := registro.Anotar(qdd, time.Now()); err != nil {
		t.Fatal(err)
	}
	var salida, errores bytes.Buffer
	if c := mejorasReportar(sfrc, "qdd", mejoras.Reporte{Titulo: "Las consultas no salen en el tablero", Autor: "sfrc-82"}, &salida, &errores); c != 0 {
		t.Fatalf("%d %s", c, errores.String())
	}
	if c := mejorasReportar(sfrc, "no-existe", mejoras.Reporte{Titulo: "x"}, &salida, &errores); c != 1 || !strings.Contains(errores.String(), "qdd") {
		t.Fatalf("un proyecto desconocido dice cuáles hay: %d %s", c, errores.String())
	}
	salida.Reset()
	if c := mejorasCiclo(qdd, true, true, time.Now(), &salida, &errores); c != 0 {
		t.Fatalf("%d %s", c, errores.String())
	}
	var r mejoras.Resumen
	if err := json.Unmarshal(salida.Bytes(), &r); err != nil || len(r.Tareas) != 1 {
		t.Fatalf("el ciclo del proyecto que recibe lo entrega: %v %s", err, salida.String())
	}
	salida.Reset()
	if mejorasCiclo(qdd, true, false, time.Now(), &salida, &errores); salida.Len() != 0 {
		t.Fatalf("con --si-toca, antes de 10 min no hace nada: %q", salida.String())
	}
	salida.Reset()
	if c := mejorasLista(qdd, false, &salida, &errores); c != 0 || !strings.Contains(salida.String(), "MEJ-001") || !strings.Contains(salida.String(), mejoras.EnTarea) {
		t.Fatalf("%d %s", c, salida.String())
	}
	if c := mejorasLista(sfrc, false, &salida, &errores); c != 0 {
		t.Fatal("la cola del que reporta sigue vacía")
	}
}

func TestDoctor_MEJ03UnaMejoraSinSuTareaLaDevuelveElCiclo(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := proyectoMejoras(t, "qdd")
	if g := buildMejorasChecks(root); len(g.Items) != 0 {
		t.Fatal("sin cola, el grupo no aparece")
	}
	if err := mejoras.Reportar(root, mejoras.Reporte{Titulo: "Una mejora"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := mejoras.Ciclo(root, mejoras.Fuentes{}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if g := buildMejorasChecks(root); groupHasFailures(g) || len(g.Items) != 3 {
		t.Fatalf("cola sana: %+v", g.Items)
	}
	e, _ := estado.Load(root)
	e.Frentes = nil
	if err := estado.Save(root, e); err != nil {
		t.Fatal(err)
	}
	g := buildMejorasChecks(root)
	if !groupHasFailures(g) || !strings.Contains(g.Items[2].Error, "MEJ-001") {
		t.Fatalf("MEJ-03 la señala: %+v", g.Items)
	}
	repairMejoras(root, g)
	if g := buildMejorasChecks(root); groupHasFailures(g) {
		t.Fatalf("--fix corre el ciclo, que la vuelve a entregar: %+v", g.Items)
	}
}

// RFC 0024, fase 3: un reporte del relay lo admite la persona, con el pedido 📮 que confirmó o en su terminal; la IA no.
func TestMejorasAdmitir_SoloLaPersona(t *testing.T) {
	nunca := func(string, string) bool { return false }
	if err := autorizarAdmision("MEJ-001", "", nil, nunca); err == nil {
		t.Fatal("sin pedido ni terminal no se admite")
	}
	if err := autorizarAdmision("MEJ-001", "4", nil, nunca); err == nil {
		t.Fatal("un pedido que la persona no confirmó no admite")
	}
	confirmado := func(p, c string) bool { return p == "4" && c == "qdd mejoras admitir MEJ-001" }
	if err := autorizarAdmision("MEJ-001", "4", nil, confirmado); err != nil {
		t.Fatalf("el pedido confirmado admite: %v", err)
	}
	if err := autorizarAdmision("MEJ-001", "", func(string) bool { return true }, nunca); err != nil {
		t.Fatalf("en su terminal, la persona dice sí: %v", err)
	}
}

// Solo el repositorio de QDD lee el relay de reportes: los demás proyectos no le preguntan nada.
func TestMejorasRelay_SoloElRepositorioDeQDD(t *testing.T) {
	if relayAplica("trdago/otro", true) || relayAplica("", false) {
		t.Fatal("otro repositorio no lee los reportes de QDD")
	}
	if !relayAplica("trdago/qdd-framework", true) {
		t.Fatal("el repositorio de QDD los lee")
	}
	t.Setenv("QDD_INCIDENTES_REPO", "yo/mi-fork")
	if !relayAplica("yo/mi-fork", true) || relayAplica("trdago/qdd-framework", true) {
		t.Fatal("QDD_INCIDENTES_REPO cambia el repositorio de QDD, como en los incidentes")
	}
}
