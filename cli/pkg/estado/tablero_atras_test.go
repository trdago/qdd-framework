package estado

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// RFC 0017 §2 y §3 (MEJ-010, sfrc): una tarea con módulo avisa de los commits nuevos en las rutas de su ficha y pide
// revisar su nombre y su criterio cuando la ficha cambia sus decisiones u objetivo operativo después de su última celda.

var tocada = time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

// gitEn hace un commit con fecha fija: el aviso compara fechas de commits con la de la última celda.
func gitEn(t *testing.T, dir string, en time.Time, args ...string) {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	fecha := en.Format(time.RFC3339)
	c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		"GIT_AUTHOR_DATE="+fecha, "GIT_COMMITTER_DATE="+fecha)
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func escribir(t *testing.T, root, ruta, contenido string) {
	t.Helper()
	p := filepath.Join(root, ruta)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(contenido), 0o644); err != nil {
		t.Fatal(err)
	}
}

func fichaPagos(decision string) string {
	return `id: pagos
nombre: Pagos
estado: aprobado
rutas: [src/pagos]
objetivo_operativo:
    problema: cobrar
    criterio_de_exito: se cobra
objetivo_tecnologico:
    arquitectura: un servicio
    decisiones: ["` + decision + `"]
pruebas:
    comandos: [go test ./...]
`
}

// proyectoConModulo: un repositorio con la ficha «pagos» y su código commiteados antes de la última celda de la tarea
// «cobro» (módulo pagos), y la tarea «otra» sin módulo con la misma fecha.
func proyectoConModulo(t *testing.T) string {
	t.Helper()
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	escribir(t, root, ".qdd/project/modules/pagos.yaml", fichaPagos("Webpay"))
	escribir(t, root, "src/pagos/pago.go", "package pagos\n")
	escribir(t, root, Ruta, `formato: 1
etapas: [Código]
frentes:
  - {id: cobro, nombre: Cobro con Webpay, estado: abierto, modulo: pagos, tocada_el: "`+tocada.Format(time.RFC3339)+`", celdas: {Código: {estado: ❌, falla: falta}}}
  - {id: otra, nombre: Otra, estado: abierto, tocada_el: "`+tocada.Format(time.RFC3339)+`", celdas: {Código: {estado: ❌, falla: falta}}}
`)
	gitEn(t, root, tocada.Add(-48*time.Hour), "init", "-q")
	gitEn(t, root, tocada.Add(-48*time.Hour), "add", "-A")
	gitEn(t, root, tocada.Add(-48*time.Hour), "commit", "-qm", "base")
	return root
}

func commitEn(t *testing.T, root string, en time.Time, ruta, contenido string) {
	t.Helper()
	escribir(t, root, ruta, contenido)
	gitEn(t, root, en, "add", "-A")
	gitEn(t, root, en, "commit", "-qm", "cambio "+ruta)
}

func cargar(t *testing.T, root string) Estado {
	t.Helper()
	e, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestSet_AnotaCuandoSeTocoLaTarea(t *testing.T) {
	root := proyectoConTareas(t, "a")
	now := time.Date(2026, 11, 15, 9, 30, 0, 0, time.UTC)
	if err := Set(root, Cambio{Frente: "a", Etapa: "Código", Estado: "ok", Evidencia: "commit abc1234"}, now); err != nil {
		t.Fatal(err)
	}
	if got := cargar(t, root).Frentes[0].TocadaEl; got != now.Format(time.RFC3339) {
		t.Fatalf("set anota tocada_el con la hora de la celda: %q", got)
	}
}

func TestTarea_DeclaraSuModuloSoloSiLaFichaExiste_EPT_R03(t *testing.T) {
	root := proyectoConModulo(t)
	if err := GuardarTarea(root, CambioTarea{ID: "otra", CambiaModulo: true, Modulo: "pagos"}, tocada); err != nil {
		t.Fatal(err)
	}
	if got := cargar(t, root).Frentes[1].Modulo; got != "pagos" {
		t.Fatalf("la tarea guarda su módulo: %q", got)
	}
	antes, _ := os.ReadFile(filepath.Join(root, Ruta))
	err := GuardarTarea(root, CambioTarea{ID: "otra", CambiaModulo: true, Modulo: "pago"}, tocada)
	if err == nil || !strings.Contains(err.Error(), "pago") {
		t.Fatalf("un módulo sin ficha es un error que lo nombra: %v", err)
	}
	despues, _ := os.ReadFile(filepath.Join(root, Ruta))
	if string(antes) != string(despues) {
		t.Fatal("una referencia equivocada no cambia el archivo")
	}
	if err := GuardarTarea(root, CambioTarea{ID: "otra", CambiaModulo: true}, tocada); err != nil {
		t.Fatal(err)
	}
	if got := cargar(t, root).Frentes[1].Modulo; got != "" {
		t.Fatalf("un módulo vacío lo quita: %q", got)
	}
}

func TestCommitsDesdeSuCelda_CuentaSoloLosDeSusRutasDespuesDeLaCelda_EPT_R01(t *testing.T) {
	root := proyectoConModulo(t)
	commitEn(t, root, tocada.Add(-time.Hour), "src/pagos/viejo.go", "package pagos\n")
	commitEn(t, root, tocada.Add(time.Hour), "src/pagos/pago.go", "package pagos // 1\n")
	commitEn(t, root, tocada.Add(2*time.Hour), "src/pagos/pago.go", "package pagos // 2\n")
	commitEn(t, root, tocada.Add(3*time.Hour), "src/otro/x.go", "package otro\n")
	c := CommitsDesdeSuCelda(root, cargar(t, root))
	if len(c.Tareas) != 1 || c.Tareas[0].Tarea != "cobro" || c.Tareas[0].Commits != 2 || c.Tareas[0].Modulo != "pagos" {
		t.Fatalf("solo la tarea con módulo, con los 2 commits de sus rutas después de su celda: %+v", c)
	}
	if c.SoloFoco {
		t.Fatal("dentro del presupuesto revisa todas")
	}
	if l := c.Tareas[0].Linea(); !strings.Contains(l, "cobro") || !strings.Contains(l, "2 commits") || !strings.Contains(l, "pagos") {
		t.Fatalf("la línea nombra la tarea, los commits y la ficha: %q", l)
	}
	if l := (TareaConCommits{Tarea: "x", Modulo: "m", Commits: 1}).Linea(); !strings.Contains(l, "1 commit en") {
		t.Fatalf("un commit va en singular: %q", l)
	}
}

func TestCommitsDesdeSuCelda_SinCommitsNuevosNoOpina(t *testing.T) {
	root := proyectoConModulo(t)
	if c := CommitsDesdeSuCelda(root, cargar(t, root)); len(c.Tareas) != 0 {
		t.Fatalf("sin commits después de la celda no hay aviso: %+v", c)
	}
}

func TestCommitsDesdeSuCelda_SinGitNoOpina(t *testing.T) {
	root := proyectoConModulo(t)
	if err := os.RemoveAll(filepath.Join(root, ".git")); err != nil {
		t.Fatal(err)
	}
	if c := CommitsDesdeSuCelda(root, cargar(t, root)); len(c.Tareas) != 0 {
		t.Fatalf("sin git no opina: %+v", c)
	}
}

func TestCommitsDesdeSuCelda_FueraDelPresupuestoSoloElFoco(t *testing.T) {
	root := proyectoConModulo(t)
	commitEn(t, root, tocada.Add(time.Hour), "src/pagos/pago.go", "package pagos // 1\n")
	e := cargar(t, root)
	e.Objetivos = []Objetivo{{ID: "caja", Nombre: "Caja", Criterio: "cobra", Estado: ObjetivoAbierto, Foco: true}}
	e.Frentes = append(e.Frentes, Frente{ID: "foco", Nombre: "Foco", Estado: EstadoAbierto, Objetivo: "caja", Modulo: "pagos",
		TocadaEl: tocada.Format(time.RFC3339), Celdas: map[string]Celda{"Código": {Estado: SinEmpezar}}})
	viejo := presupuestoCommits
	presupuestoCommits = -1
	t.Cleanup(func() { presupuestoCommits = viejo })
	c := CommitsDesdeSuCelda(root, e)
	if len(c.Tareas) != 1 || c.Tareas[0].Tarea != "foco" || !c.SoloFoco {
		t.Fatalf("pasado el presupuesto, solo las tareas del objetivo en foco y lo dice: %+v", c)
	}
}

func TestTareasPorRevisar_LaFichaCambioSusDecisionesDespuesDeLaCelda_EPT_R02(t *testing.T) {
	root := proyectoConModulo(t)
	commitEn(t, root, tocada.Add(time.Hour), ".qdd/project/modules/pagos.yaml", fichaPagos("Mercado Pago"))
	rs := TareasPorRevisar(root, cargar(t, root)).Tareas
	if len(rs) != 1 || rs[0].Tarea != "cobro" || strings.Join(rs[0].Cambios, ",") != "decisiones" {
		t.Fatalf("la tarea del módulo se revisa por sus decisiones: %+v", rs)
	}
	l := rs[0].Linea()
	for _, want := range []string{"cobro", "Cobro con Webpay", "pagos", "decisiones", "nombre", "criterio"} {
		if !strings.Contains(l, want) {
			t.Fatalf("la línea dice %q: %q", want, l)
		}
	}
}

func TestTareasPorRevisar_FueraDelPresupuestoSoloElFoco(t *testing.T) {
	root := proyectoConModulo(t)
	commitEn(t, root, tocada.Add(time.Hour), ".qdd/project/modules/pagos.yaml", fichaPagos("Mercado Pago"))
	viejo := presupuestoCommits
	presupuestoCommits = -1
	t.Cleanup(func() { presupuestoCommits = viejo })
	r := TareasPorRevisar(root, cargar(t, root))
	if len(r.Tareas) != 0 || !r.SoloFoco {
		t.Fatalf("pasado el presupuesto y sin foco, no revisa y lo dice: %+v", r)
	}
}

func TestTareasPorRevisar_SinCommitTambienVeLaFichaEditada(t *testing.T) {
	root := proyectoConModulo(t)
	escribir(t, root, ".qdd/project/modules/pagos.yaml", strings.Replace(fichaPagos("Webpay"), "problema: cobrar", "problema: cobrar en cuotas", 1))
	rs := TareasPorRevisar(root, cargar(t, root)).Tareas
	if len(rs) != 1 || strings.Join(rs[0].Cambios, ",") != "objetivo operativo" {
		t.Fatalf("la ficha de la carpeta se compara con la de la celda: %+v", rs)
	}
}

func TestTareasPorRevisar_OtrosCambiosDeLaFichaNoPidenRevision(t *testing.T) {
	root := proyectoConModulo(t)
	commitEn(t, root, tocada.Add(time.Hour), ".qdd/project/modules/pagos.yaml", strings.Replace(fichaPagos("Webpay"), "go test ./...", "go test ./src/...", 1))
	if rs := TareasPorRevisar(root, cargar(t, root)).Tareas; len(rs) != 0 {
		t.Fatalf("cambiar las pruebas de la ficha no cambia qué falta: %+v", rs)
	}
}

func TestTareasPorRevisar_FichaPosteriorALaCeldaNoOpina(t *testing.T) {
	root := proyectoConModulo(t)
	e := cargar(t, root)
	e.Frentes[0].TocadaEl = tocada.Add(-72 * time.Hour).Format(time.RFC3339)
	if rs := TareasPorRevisar(root, e).Tareas; len(rs) != 0 {
		t.Fatalf("sin la ficha de cuando se tocó la tarea no hay con qué comparar: %+v", rs)
	}
}

// Compatibilidad: un estado.yaml sin módulo ni tocada_el se lee y se escribe igual; no aparecen campos vacíos.
func TestEstadoSinModulo_NoCambiaAlGuardarse(t *testing.T) {
	root := proyectoConTareas(t, "a")
	if err := GuardarTarea(root, CambioTarea{ID: "a", CambiaObjetivo: true}, tocada); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(root, Ruta))
	if strings.Contains(string(data), "modulo") || strings.Contains(string(data), "tocada_el") {
		t.Fatalf("sin módulo ni celda nueva, el archivo no gana esos campos:\n%s", data)
	}
}
