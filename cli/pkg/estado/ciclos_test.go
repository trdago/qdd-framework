package estado

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// Ciclos con nombre (decisión de la persona, 2026-11-01, por ciclo-de-vida-agentico en llmlocal): las etapas del
// archivo son el ciclo del proyecto; un objetivo puede seguir otro ciclo y sus tareas toman sus etapas.

var etapasAgentico = []string{"Concepto", "Artefacto", "Medición", "Puerta"}

func celdasDe(etapas []string, estado string) map[string]Celda {
	c := map[string]Celda{}
	for _, e := range etapas {
		c[e] = Celda{Estado: estado}
	}
	return c
}

func conCiclos() Estado {
	return Estado{Formato: FormatoConCiclos, Etapas: etapasSoftware,
		Ciclos: map[string][]string{"agentico": etapasAgentico},
		Objetivos: []Objetivo{
			{ID: "tutor", Nombre: "Tutor agéntico", Criterio: "las 14 fases con su puerta", Estado: ObjetivoAbierto, Ciclo: "agentico", Foco: true},
			{ID: "release", Nombre: "Publicar", Criterio: "uso medido", Estado: ObjetivoAbierto},
		},
		Frentes: []Frente{
			tarea("f01", "tutor", celdasDe(etapasAgentico, SinEmpezar)),
			tarea("panel", "release", vacia()),
			tarea("suelta", "", vacia()),
		}}
}

func guardarEn(t *testing.T, e Estado) string {
	t.Helper()
	root := t.TempDir()
	if err := Save(root, e); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestEPT_K01_UnCicloTieneEtapasYUnObjetivoSigueUnoQueExiste(t *testing.T) {
	if vs := Check(conCiclos(), nil); len(vs) != 0 {
		t.Fatalf("un estado con ciclos válido no debería tener violaciones: %s", idsDe(vs))
	}
	casos := map[string]func(*Estado){
		"sin etapas":         func(e *Estado) { e.Ciclos["vacio"] = nil },
		"etapa repetida":     func(e *Estado) { e.Ciclos["doble"] = []string{"A", "A"} },
		"se llama proyecto":  func(e *Estado) { e.Ciclos[CicloProyecto] = []string{"A"} },
		"objetivo sin ciclo": func(e *Estado) { e.Objetivos[1].Ciclo = "no-existe" },
	}
	for nombre, romper := range casos {
		e := conCiclos()
		e.Ciclos = map[string][]string{"agentico": etapasAgentico}
		romper(&e)
		if !tieneID(Check(e, nil), "EPT-K01") {
			t.Errorf("%s: se esperaba EPT-K01, hubo %s", nombre, idsDe(Check(e, nil)))
		}
	}
	// «proyecto» nombra el ciclo de las etapas del archivo: un objetivo puede pedirlo explícito.
	e := conCiclos()
	e.Objetivos[1].Ciclo = CicloProyecto
	if vs := Check(e, nil); len(vs) != 0 {
		t.Fatalf("ciclo «proyecto» explícito: %s", idsDe(vs))
	}
	root := guardarEn(t, conCiclos())
	if err := DeclararCiclo(root, "doble", []string{"A", "A"}, time.Now()); err == nil {
		t.Fatal("declarar un ciclo con una etapa repetida debería fallar")
	}
	if err := GuardarObjetivo(root, CambioObjetivo{ID: "nuevo", Nombre: "N", Criterio: "c", Ciclo: "no-existe"}, time.Now()); err == nil {
		t.Fatal("un objetivo con un ciclo que no existe no se escribe")
	}
}

func TestEPT_K02_LaTareaTomaLasEtapasDeSuCiclo(t *testing.T) {
	root := guardarEn(t, conCiclos())
	if err := GuardarTarea(root, CambioTarea{ID: "f02", Nombre: "Explicar el tutor", Objetivo: "tutor", CambiaObjetivo: true}, time.Now()); err != nil {
		t.Fatal(err)
	}
	e, _ := Load(root)
	f := e.Frentes[e.buscar("f02")]
	if strings.Join(e.EtapasDe(f), ",") != strings.Join(etapasAgentico, ",") || len(f.Celdas) != 4 {
		t.Fatalf("la tarea de un objetivo con ciclo toma sus etapas: %v", f.Celdas)
	}
	if err := Set(root, Cambio{Frente: "f02", Etapa: "Código", Estado: "⏳", Bloqueo: "x", DeQuien: "la persona"}, time.Now()); err == nil {
		t.Fatal("una etapa fuera del ciclo de la tarea no se escribe")
	}
	if err := Set(root, Cambio{Frente: "f02", Etapa: "Concepto", Estado: "⏳", Bloqueo: "explicar", DeQuien: "el tutor"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	// Una celda de otra etapa o una que falta no valen.
	e = conCiclos()
	e.Frentes[0].Celdas["Código"] = Celda{Estado: SinEmpezar}
	delete(e.Frentes[0].Celdas, "Puerta")
	vs := Check(e, nil)
	if !violacionEn(vs, "EPT-K02", "f01") || !violacionEn(vs, "EPF-C01", "f01") {
		t.Fatalf("celda ajena (EPT-K02) y celda que falta (EPF-C01): %s", idsDe(vs))
	}
	// Terminada y totales usan las etapas de su ciclo, no las del proyecto.
	e = conCiclos()
	e.Frentes[0].Celdas = celdasDe(etapasAgentico, NoAplica)
	if !e.TareaTerminada(e.Frentes[0]) {
		t.Fatal("con sus 4 etapas en — la tarea está terminada")
	}
}

func TestEPT_K03_CambiarDeCicloNoPierdeAvance(t *testing.T) {
	e := conCiclos()
	e.Ciclos["corto"] = []string{"Concepto", "Puerta"}
	e.Objetivos = append(e.Objetivos, Objetivo{ID: "corto", Nombre: "Corto", Criterio: "c", Estado: ObjetivoAbierto, Ciclo: "corto"})
	e.Frentes[0].Celdas["Concepto"] = Celda{Estado: Hecho, Evidencia: "corrida c1"}
	root := guardarEn(t, e)
	// Las etapas en común conservan su celda; las nuevas empiezan en «·»; las que sobran sin avance se van.
	if err := GuardarTarea(root, CambioTarea{ID: "f01", Objetivo: "corto", CambiaObjetivo: true}, time.Now()); err != nil {
		t.Fatal(err)
	}
	e, _ = Load(root)
	f := e.Frentes[e.buscar("f01")]
	if len(f.Celdas) != 2 || f.Celdas["Concepto"].Evidencia != "corrida c1" {
		t.Fatalf("conserva Concepto y deja solo las etapas del ciclo nuevo: %v", f.Celdas)
	}
	// Con avance en una etapa que el ciclo nuevo no tiene, no se escribe.
	if err := Set(root, Cambio{Frente: "f01", Etapa: "Puerta", Estado: "⏳", Bloqueo: "aprobar", DeQuien: "la persona"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	err := GuardarTarea(root, CambioTarea{ID: "f01", Objetivo: "release", CambiaObjetivo: true}, time.Now())
	if err == nil || !strings.Contains(err.Error(), "Puerta") {
		t.Fatalf("pasar a un ciclo sin «Puerta» perdería su ⏳: %v", err)
	}
	// Lo mismo al cambiar el ciclo de un objetivo o las etapas de un ciclo.
	if err := GuardarObjetivo(root, CambioObjetivo{ID: "corto", Ciclo: CicloProyecto, CambiaCiclo: true}, time.Now()); err == nil {
		t.Fatal("cambiar el ciclo del objetivo perdería el avance de sus tareas")
	}
	if err := DeclararCiclo(root, "corto", []string{"Concepto"}, time.Now()); err == nil {
		t.Fatal("quitar «Puerta» del ciclo perdería el avance de sus tareas")
	}
	if err := DeclararCiclo(root, "corto", []string{"Concepto", "Medición", "Puerta"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	e, _ = Load(root)
	if c := e.Frentes[e.buscar("f01")].Celdas["Medición"]; c.Estado != SinEmpezar {
		t.Fatalf("una etapa nueva del ciclo empieza en «·»: %v", c)
	}
}

func TestEPT_K04_UnaTablaPorCiclo(t *testing.T) {
	e := conCiclos()
	md := Markdown(e, time.Now(), AlcanceAbiertos)
	tablas := strings.Split(strings.TrimSpace(md), "\n\n")
	if len(tablas) != 2 {
		t.Fatalf("una tabla por ciclo, la del proyecto primero:\n%s", md)
	}
	if !strings.HasPrefix(tablas[0], "| Tarea | Código |") || !strings.HasPrefix(tablas[1], "| Tarea | Concepto |") {
		t.Fatalf("cada tabla con sus columnas, la del proyecto primero:\n%s", md)
	}
	if strings.Contains(tablas[0], "f01") || !strings.Contains(tablas[1], "Tutor agéntico") {
		t.Fatalf("las tareas del ciclo agéntico van en su tabla, bajo su objetivo:\n%s", md)
	}
	if vs := CheckRespuesta(e, md, time.Now()); len(vs) != 0 {
		t.Fatalf("la respuesta con las dos tablas coincide con el archivo: %s", idsDe(vs))
	}
	sinSegunda := tablas[0]
	if !violacionEn(CheckRespuesta(e, sinSegunda, time.Now()), "EPF-A01", "f01") {
		t.Fatal("sin la tabla del ciclo agéntico, f01 falta en la respuesta")
	}
	if vs := CheckRespuesta(e, md+"\nf01 · Concepto: ⏳\n", time.Now()); !violacionEn(vs, "EPF-C08", "f01") {
		t.Fatalf("una afirmación sobre una etapa del ciclo de la tarea se compara con su celda: %s", idsDe(vs))
	}
	if strings.Contains(Mensajeria(e, time.Now(), AlcanceAbiertos), "f01*\n  Código") {
		t.Fatal("en mensajería, cada tarea con las etapas de su ciclo")
	}
}

func TestEPT_K05_ElPanelLlevaLosCiclos(t *testing.T) {
	e := conCiclos()
	e.Frentes[0].Celdas["Concepto"] = Celda{Estado: Espera, Bloqueo: "explicar", DeQuien: "el tutor"}
	d := DatosPendientes(e, time.Now(), 10)
	b, _ := json.Marshal(d)
	if len(d.Ciclos) != 1 || d.Ciclos[0].Nombre != "agentico" || strings.Join(d.Ciclos[0].Abreviaturas, " ") != strings.Join(abreviaturas(etapasAgentico), " ") {
		t.Fatalf("el JSON trae cada ciclo que usa un objetivo, con sus etapas y abreviaturas: %s", b)
	}
	var tutor ObjetivoPanel
	for _, o := range d.Objetivos {
		if o.ID == "tutor" {
			tutor = o
		}
	}
	if tutor.Ciclo != "agentico" {
		t.Fatalf("el objetivo dice su ciclo: %s", b)
	}
	for _, f := range d.Frentes {
		if f.ID == "f01" && len(f.Simbolos) != 4 {
			t.Fatalf("la fila trae un símbolo por etapa de su ciclo: %v", f.Simbolos)
		}
		if f.ID == "panel" && len(f.Simbolos) != len(etapasSoftware) {
			t.Fatalf("la del proyecto, por las del proyecto: %v", f.Simbolos)
		}
	}
	if len(d.Esperas) != 1 || d.Esperas[0].Etapa != "Concepto" || d.Esperan != 1 {
		t.Fatalf("los ⏳ de un ciclo propio cuentan con su etapa: %s", b)
	}
	if lineas := strings.Join(Panel(e, "p", 80, time.Now()), "\n"); !strings.Contains(lineas, "Concepto") {
		t.Fatalf("la vista de terminal nombra las etapas de cada ciclo:\n%s", lineas)
	}
}

func TestCiclos_FormatoTresSoloConCiclosYUnQddViejoNoLoPierde(t *testing.T) {
	root := guardarEn(t, conObjetivos())
	if e, _ := Load(root); e.Formato != FormatoConObjetivos {
		t.Fatalf("sin ciclos queda en formato %d, se esperaba %d", e.Formato, FormatoConObjetivos)
	}
	root = guardarEn(t, conCiclos())
	data, _ := os.ReadFile(filepath.Join(root, Ruta))
	if e, _ := Parse(data); e.Formato != FormatoConCiclos || len(e.Ciclos["agentico"]) != 4 || e.Objetivos[0].Ciclo != "agentico" {
		t.Fatalf("con ciclos se guarda en formato %d y se relee entero:\n%s", FormatoConCiclos, data)
	}
}

func tieneID(vs []Violacion, id string) bool {
	for _, v := range vs {
		if v.ID == id {
			return true
		}
	}
	return false
}
