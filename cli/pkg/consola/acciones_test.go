package consola

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/modules"
)

// Las listas exactas de acciones no dependen de si la máquina tiene xdg-open: sin abridor salvo en los tests de §3.9.
func init() {
	buscarAbridor = func() (Abridor, bool) { return Abridor{}, false }
}

// proyectoConFichas arma un proyecto QDD (con .qdd/) con una ficha por id; las de borradores quedan en borrador y el resto aprobadas.
func proyectoConFichas(t *testing.T, aprobadas []string, borradores []string) string {
	t.Helper()
	root := t.TempDir()
	for _, d := range []string{"src", ".qdd"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0755); err != nil {
			t.Fatal(err)
		}
	}
	escribir := func(id, estadoFicha string) {
		m := modules.Module{
			ID: id, Nombre: "Módulo " + id, Estado: estadoFicha, Rutas: []string{"src"},
			ObjetivoOperativo:   modules.ObjetivoOperativo{Problema: "p", CriterioDeExito: "c"},
			ObjetivoTecnologico: modules.ObjetivoTecnologico{Arquitectura: "a"},
			Pruebas:             modules.Pruebas{Comandos: []string{"go test ./..."}},
		}
		if _, problemas := modules.Save(root, m); len(problemas) > 0 {
			t.Fatalf("ficha %s: %v", id, problemas)
		}
		if estadoFicha == modules.EstadoAprobado {
			if _, problemas := modules.Approve(root, id); len(problemas) > 0 {
				t.Fatalf("aprobar %s: %v", id, problemas)
			}
		}
	}
	for _, id := range aprobadas {
		escribir(id, modules.EstadoAprobado)
	}
	for _, id := range borradores {
		escribir(id, modules.EstadoBorrador)
	}
	return root
}

func ids(acciones []Accion) []string {
	var out []string
	for _, a := range acciones {
		out = append(out, a.ID)
	}
	return out
}

func TestAcciones_OfreceFichasEnBorradorYComandos_CCC_C10(t *testing.T) {
	root := proyectoConFichas(t, []string{"auth"}, []string{"pagos", "catalogo"})
	got := ids(Acciones(root, true))
	want := []string{"aprobar:catalogo", "aprobar:pagos", "revisar-estado", "auditar"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("acciones:\n got %v\nwant %v", got, want)
	}
	a, _ := BuscarAccion(Acciones(root, true), "aprobar:pagos")
	if a.Tipo != TipoAprobar || !a.Confirmar || a.Detalle != "qdd modules approve pagos" {
		t.Fatalf("aprobar pagos mal armada: %+v", a)
	}
	r, _ := BuscarAccion(Acciones(root, true), "revisar-estado")
	if r.Confirmar || r.Detalle != "qdd estado check" {
		t.Fatalf("revisar el estado es de solo lectura y no pide confirmación: %+v", r)
	}
}

func TestAcciones_SinEstadoNoOfreceRevisar(t *testing.T) {
	root := proyectoConFichas(t, nil, nil)
	if got := ids(Acciones(root, false)); strings.Join(got, ",") != "auditar" {
		t.Fatalf("sin estado.yaml solo queda auditar: %v", got)
	}
}

func TestAcciones_FueraDeUnProyectoNoOfreceNada(t *testing.T) {
	if got := Acciones(t.TempDir(), false); len(got) != 0 {
		t.Fatalf("una carpeta sin .qdd/ no tiene acciones: %v", ids(got))
	}
}

func TestAcciones_TopeDeNueveFichas(t *testing.T) {
	var borradores []string
	for _, c := range "abcdefghijk" {
		borradores = append(borradores, "f"+string(c))
	}
	root := proyectoConFichas(t, nil, borradores)
	n := 0
	for _, a := range Acciones(root, false) {
		if a.Tipo == TipoAprobar {
			n++
		}
	}
	if n != MaxFichas {
		t.Fatalf("se ofrecen %d fichas; el tope es %d (una tecla 1…9 por ficha)", n, MaxFichas)
	}
}

func TestAutorizar_RechazaIdNoOfrecidoYEscrituraSinConfirmar_CCC_C11(t *testing.T) {
	root := proyectoConFichas(t, []string{"auth"}, []string{"pagos"})
	acciones := Acciones(root, true)
	if _, err := BuscarAccion(acciones, "aprobar:auth"); err == nil {
		t.Fatal("una ficha ya aprobada no se ofrece; pedirla debe fallar")
	}
	if _, err := BuscarAccion(acciones, "rm -rf /"); err == nil {
		t.Fatal("un id que qdd no ofrece debe fallar")
	}
	a, err := BuscarAccion(acciones, "aprobar:pagos")
	if err != nil {
		t.Fatal(err)
	}
	if err := Autorizar(a, false); err == nil {
		t.Fatal("aprobar sin --confirmado debe fallar")
	}
	if err := Autorizar(a, true); err != nil {
		t.Fatalf("aprobar con --confirmado: %v", err)
	}
	r, _ := BuscarAccion(acciones, "revisar-estado")
	if err := Autorizar(r, false); err != nil {
		t.Fatalf("revisar el estado no pide confirmación: %v", err)
	}
}

func TestEjecutar_AprobarDejaLaFichaAprobada_CCC_C10(t *testing.T) {
	root := proyectoConFichas(t, nil, []string{"pagos"})
	a, err := BuscarAccion(Acciones(root, false), "aprobar:pagos")
	if err != nil {
		t.Fatal(err)
	}
	var salida bytes.Buffer
	if codigo := a.Ejecutar(root, "qdd-que-no-se-usa", &salida); codigo != 0 {
		t.Fatalf("código %d: %s", codigo, salida.String())
	}
	m, _ := modules.Find(root, "pagos")
	if m.Estado != modules.EstadoAprobado || m.UltimaRevision == "" {
		t.Fatalf("la ficha debió quedar aprobada: %+v", m)
	}
	if !strings.Contains(salida.String(), "Ficha 'pagos' aprobada") {
		t.Fatalf("la salida debe decir lo mismo que 'qdd modules approve': %q", salida.String())
	}
	if _, err := BuscarAccion(Acciones(root, false), "aprobar:pagos"); err == nil {
		t.Fatal("una vez aprobada, la ficha ya no se ofrece")
	}
}

func TestPanel_SinEstadoDiceElMotivoYMantieneLasAcciones(t *testing.T) {
	root := proyectoConFichas(t, nil, []string{"pagos"})
	d := Panel(root, time.Now())
	if d.Estado != nil || !strings.Contains(d.Motivo, "qdd estado init") {
		t.Fatalf("sin estado.yaml: estado nulo y motivo con cómo crearlo: %+v", d)
	}
	out, _ := json.Marshal(d)
	if !strings.Contains(string(out), `"estado":null`) || !strings.Contains(string(out), `"aprobar:pagos"`) {
		t.Fatalf("JSON del panel: %s", out)
	}
}

func TestPanel_ConEstado(t *testing.T) {
	root := proyectoConFichas(t, nil, nil)
	yaml := "formato: 1\netapas: [Código, Pruebas]\nfrentes:\n  - id: a\n    nombre: Frente A\n    estado: abierto\n    celdas:\n      Código: {estado: ✅, evidencia: abc}\n"
	if err := os.MkdirAll(filepath.Join(root, ".qdd/project"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".qdd/project/estado.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	d := Panel(root, time.Now())
	if d.Estado == nil || len(d.Estado.Frentes) != 1 || d.Motivo != "" {
		t.Fatalf("con estado.yaml el panel trae los frentes: %+v", d)
	}
	if got := ids(d.Acciones); strings.Join(got, ",") != "descartar:a,revisar-estado,auditar" {
		t.Fatalf("acciones con estado y sin borradores: %v", got)
	}
}

// §3.9: la ✕ de una tarea pide confirmación y la deja «descartada» con su motivo; sale del panel.
func TestEjecutar_DescartarTareaConLaX_CCC_C22(t *testing.T) {
	root := proyectoConFichas(t, nil, nil)
	yaml := "formato: 1\netapas: [Código, Pruebas]\nfrentes:\n  - id: a\n    nombre: Frente A\n    estado: abierto\n    celdas:\n      Código: {estado: ✅, evidencia: abc}\n      Pruebas: {estado: ⏳, bloqueo: correr la suite, de_quien: CI}\n"
	if err := os.MkdirAll(filepath.Join(root, ".qdd/project"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".qdd/project/estado.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	a, err := BuscarAccion(Acciones(root, true), "descartar:a")
	if err != nil {
		t.Fatal(err)
	}
	if err := Autorizar(a, false); err == nil {
		t.Fatal("descartar escribe en estado.yaml: exige --confirmado")
	}
	var salida bytes.Buffer
	if codigo := a.Ejecutar(root, "qdd-que-no-se-usa", &salida); codigo != 0 {
		t.Fatalf("código %d: %s", codigo, salida.String())
	}
	d := Panel(root, time.Now())
	if len(d.Estado.Frentes) != 0 {
		t.Fatalf("la tarea descartada sale del panel: %+v", d.Estado.Frentes)
	}
	escrito, _ := os.ReadFile(filepath.Join(root, ".qdd/project/estado.yaml"))
	if !strings.Contains(string(escrito), "descartado") || !strings.Contains(string(escrito), MotivoDescarte) {
		t.Fatalf("queda en el archivo con estado y motivo:\n%s", escrito)
	}
}
