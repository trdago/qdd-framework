package estado

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// Pedido del usuario (2026-11-01, «qdd sprint»): el tablero dice qué se hace y cómo va, pero no para qué. Los objetivos
// agrupan las tareas con su criterio de éxito y el foco actual; las dependencias bloquean y se ven como árbol.

var etapasSoftware = []string{"Código", "Pruebas", "PR", "Publicación"}

func hecha() map[string]Celda {
	return map[string]Celda{"Código": {Estado: Hecho, Evidencia: "abc1234"}, "Pruebas": {Estado: Hecho, Evidencia: "3/3 pruebas"}, "PR": {Estado: Hecho, Evidencia: "v1.0.0"}, "Publicación": {Estado: NoAplica}}
}

func vacia() map[string]Celda {
	c := map[string]Celda{}
	for _, e := range etapasSoftware {
		c[e] = Celda{Estado: SinEmpezar}
	}
	return c
}

func tarea(id, objetivo string, celdas map[string]Celda, depende ...string) Frente {
	return Frente{ID: id, Nombre: id, Estado: EstadoAbierto, Objetivo: objetivo, DependeDe: depende, Celdas: celdas}
}

func conObjetivos() Estado {
	return Estado{Formato: Formato, Etapas: etapasSoftware,
		Objetivos: []Objetivo{
			{ID: "release", Nombre: "Publicar 2.8.0", Criterio: "5 h de uso medido", Estado: ObjetivoAbierto},
			{ID: "arbol", Nombre: "Objetivos y árbol", Criterio: "el panel dibuja el árbol", Estado: ObjetivoAbierto, Foco: true},
		},
		Frentes: []Frente{
			tarea("receta", "release", hecha()),
			tarea("panel", "release", vacia(), "receta"),
			tarea("herramientas", "release", vacia(), "panel"),
			tarea("modelo", "arbol", vacia()),
			tarea("dibujo", "arbol", vacia(), "modelo", "herramientas"),
			tarea("limpiar", "", vacia()),
		}}
}

func idsDe(vs []Violacion) string {
	var out []string
	for _, v := range vs {
		out = append(out, v.ID+" "+v.Frente)
	}
	return strings.Join(out, "; ")
}

func violacionEn(vs []Violacion, id, frente string) bool {
	for _, v := range vs {
		if v.ID == id && v.Frente == frente {
			return true
		}
	}
	return false
}

func TestCheck_ConObjetivosYDependenciasValidos(t *testing.T) {
	if vs := Check(conObjetivos(), nil); len(vs) > 0 {
		t.Fatalf("un tablero con objetivos y dependencias válidas no tiene violaciones: %s", idsDe(vs))
	}
}

func TestEPT_D01_UnaDependenciaExisteYNoHaceCiclo(t *testing.T) {
	e := conObjetivos()
	e.Frentes[0].DependeDe = []string{"no-existe"}
	if vs := Check(e, nil); !violacionEn(vs, "EPT-D01", "receta") {
		t.Fatalf("depender de una tarea que no está no vale: %s", idsDe(vs))
	}
	e = conObjetivos()
	e.Frentes[0].DependeDe = []string{"herramientas"} // receta → herramientas → panel → receta
	if vs := Check(e, nil); !violacionEn(vs, "EPT-D01", "receta") {
		t.Fatalf("un ciclo de dependencias no vale: %s", idsDe(vs))
	}
	e = conObjetivos()
	e.Frentes[0].DependeDe = []string{"receta"}
	if vs := Check(e, nil); !violacionEn(vs, "EPT-D01", "receta") {
		t.Fatalf("depender de sí misma no vale: %s", idsDe(vs))
	}
}

func TestEPT_D02_SinSuDependenciaNoTermina(t *testing.T) {
	e := conObjetivos()
	e.Frentes[2].Celdas = hecha() // herramientas termina y panel, su dependencia, no
	vs := Check(e, nil)
	if !violacionEn(vs, "EPT-D02", "herramientas") {
		t.Fatalf("la última etapa no pasa a ✅ mientras su dependencia no termina: %s", idsDe(vs))
	}
	if !strings.Contains(vs[0].Mensaje, "panel") {
		t.Fatalf("dice a quién espera: %s", vs[0].Mensaje)
	}
	e.Frentes[1].Celdas = hecha() // panel termina: herramientas ya puede
	if vs := Check(e, nil); len(vs) > 0 {
		t.Fatalf("con su dependencia terminada, la tarea termina: %s", idsDe(vs))
	}
}

func TestEPT_D02_UnaDependenciaCerradaLibera(t *testing.T) {
	e := conObjetivos()
	e.Frentes[1].Estado, e.Frentes[1].MotivoCierre = EstadoCerrado, "hecho en otro PR"
	e.Frentes[2].Celdas = hecha()
	if vs := Check(e, nil); violacionEn(vs, "EPT-D02", "herramientas") {
		t.Fatalf("una dependencia cerrada está terminada: %s", idsDe(vs))
	}
	e.Frentes[1].Estado = EstadoDescartado
	if vs := Check(e, nil); !violacionEn(vs, "EPT-D02", "herramientas") {
		t.Fatalf("una dependencia descartada no libera: hay que quitarla: %s", idsDe(vs))
	}
}

func TestSet_NoTerminaUnaTareaBloqueada_EPT_D02(t *testing.T) {
	root := t.TempDir()
	if err := Save(root, conObjetivos()); err != nil {
		t.Fatal(err)
	}
	if err := Set(root, Cambio{Frente: "herramientas", Etapa: "Código", Estado: "ok", Evidencia: "abc1234"}, ahora); err != nil {
		t.Fatalf("una etapa intermedia de una tarea bloqueada avanza: %v", err)
	}
	err := Set(root, Cambio{Frente: "herramientas", Etapa: "Publicación", Estado: "ok", Evidencia: "v1.0.0"}, ahora)
	if err == nil || !strings.Contains(err.Error(), "espera a panel") {
		t.Fatalf("la última etapa no pasa a ✅ mientras panel no termina, y lo dice: %v", err)
	}
	for _, etapa := range etapasSoftware {
		if err := Set(root, Cambio{Frente: "panel", Etapa: etapa, Estado: "ok", Evidencia: "abc1234"}, ahora); err != nil {
			t.Fatal(err)
		}
	}
	if err := Set(root, Cambio{Frente: "herramientas", Etapa: "Publicación", Estado: "ok", Evidencia: "v1.0.0"}, ahora); err != nil {
		t.Fatalf("con panel terminada, herramientas termina: %v", err)
	}
}

func TestEPT_O01_ObjetivoConCriterio(t *testing.T) {
	root := t.TempDir()
	if err := Save(root, conObjetivos()); err != nil {
		t.Fatal(err)
	}
	if err := GuardarObjetivo(root, CambioObjetivo{ID: "nuevo", Nombre: "Algo"}, ahora); err == nil {
		t.Fatal("un objetivo sin criterio de éxito no se escribe")
	}
	if err := GuardarObjetivo(root, CambioObjetivo{ID: "nuevo", Nombre: "Algo", Criterio: "medido en 10 usos"}, ahora); err != nil {
		t.Fatalf("con criterio sí: %v", err)
	}
	e, _ := Load(root)
	if o, ok := e.objetivo("nuevo"); !ok || o.Estado != ObjetivoAbierto {
		t.Fatalf("queda abierto: %+v", o)
	}
}

func TestEPT_O02_UnaTareaSirveAUnObjetivoQueExiste(t *testing.T) {
	e := conObjetivos()
	e.Frentes[0].Objetivo = "no-existe"
	if vs := Check(e, nil); !violacionEn(vs, "EPT-O02", "receta") {
		t.Fatalf("apuntar a un objetivo que no está no vale: %s", idsDe(vs))
	}
	root := t.TempDir()
	if err := Save(root, conObjetivos()); err != nil {
		t.Fatal(err)
	}
	if err := GuardarTarea(root, CambioTarea{ID: "limpiar", Objetivo: "no-existe", CambiaObjetivo: true}, ahora); err == nil {
		t.Fatal("qdd estado tarea no escribe un objetivo que no existe")
	}
}

func TestEPT_O03_UnSoloFoco(t *testing.T) {
	root := t.TempDir()
	if err := Save(root, conObjetivos()); err != nil {
		t.Fatal(err)
	}
	if err := GuardarObjetivo(root, CambioObjetivo{ID: "release", Foco: true}, ahora); err != nil {
		t.Fatal(err)
	}
	e, _ := Load(root)
	if o, _ := e.objetivo("arbol"); o.Foco {
		t.Fatal("marcar un foco quita el anterior")
	}
	e.Objetivos[1].Foco = true
	if vs := Check(e, nil); !violacionEn(vs, "EPT-O03", "") {
		t.Fatalf("dos focos abiertos no valen: %s", idsDe(vs))
	}
}

func TestEPT_O04_CumplidoConEvidencia(t *testing.T) {
	e := conObjetivos()
	e.Frentes[1].Celdas, e.Frentes[2].Celdas = hecha(), hecha()
	o, _ := e.objetivo("release")
	if s, texto := e.EstadoObjetivo(o); s != Espera || texto != "tareas listas, falta verificar el criterio" {
		t.Fatalf("con todas sus tareas terminadas y sin evidencia: %s %q", s, texto)
	}
	root := t.TempDir()
	if err := Save(root, e); err != nil {
		t.Fatal(err)
	}
	if err := CerrarObjetivo(root, "release", "", false, ahora); err == nil {
		t.Fatal("un objetivo no se cumple sin evidencia de su criterio")
	}
	if err := CerrarObjetivo(root, "release", "corrida uso-5h", false, ahora); err != nil {
		t.Fatal(err)
	}
	e, _ = Load(root)
	o, _ = e.objetivo("release")
	if s, texto := e.EstadoObjetivo(o); s != Hecho || texto != "corrida uso-5h" {
		t.Fatalf("cumplido con su evidencia: %s %q", s, texto)
	}
	e.Objetivos[0].Evidencia = ""
	if vs := Check(e, nil); !violacionEn(vs, "EPT-O04", "release") {
		t.Fatalf("cumplido sin evidencia no vale: %s", idsDe(vs))
	}
}

func TestEPT_O05_ElObjetivoNoDesaparece(t *testing.T) {
	prev := conObjetivos()
	e := conObjetivos()
	e.Objetivos = e.Objetivos[1:]
	for i := range e.Frentes {
		if e.Frentes[i].Objetivo == "release" {
			e.Frentes[i].Objetivo = ""
		}
	}
	if vs := Check(e, &prev); !violacionEn(vs, "EPT-O05", "release") {
		t.Fatalf("un objetivo que desaparece sin cerrarse no vale: %s", idsDe(vs))
	}
	root := t.TempDir()
	if err := Save(root, conObjetivos()); err != nil {
		t.Fatal(err)
	}
	if err := CerrarObjetivo(root, "release", "lo absorbió otro", true, ahora); err != nil {
		t.Fatal(err)
	}
	e, _ = Load(root)
	if o, _ := e.objetivo("release"); o.Estado != ObjetivoDescartado || o.Motivo == "" {
		t.Fatalf("se descarta con su motivo, y sigue en el archivo: %+v", o)
	}
}

func TestEPT_O06_LaTablaAgrupaPorObjetivo(t *testing.T) {
	md := Markdown(conObjetivos(), ahora, AlcanceAbiertos)
	lineas := strings.Split(strings.TrimSpace(md), "\n")
	var primeras []string
	for _, l := range lineas[2:] {
		primeras = append(primeras, partirFila(l)[0])
	}
	got := strings.Join(primeras, " / ")
	want := "🎯 Objetivos y árbol · foco / modelo / dibujo (espera a modelo, herramientas) / 🎯 Publicar 2.8.0 / receta / panel / herramientas (espera a panel) / 🎯 sin objetivo / limpiar"
	if got != want {
		t.Fatalf("el foco primero, cada objetivo antes de sus tareas y las sin objetivo al final:\n got %s\nwant %s", got, want)
	}
	fila := partirFila(lineas[2])
	if fila[1] != NoAplica || !strings.HasPrefix(fila[len(fila)-1], Espera) {
		t.Fatalf("la fila del objetivo lleva «—» y su estado en la última etapa: %v", fila)
	}
	if vs := CheckRespuesta(conObjetivos(), md, ahora); len(vs) > 0 {
		t.Fatalf("la tabla agrupada pasa su propio lector: %s", idsDe(vs))
	}
}

func TestEPT_O06_SinObjetivosLaTablaSigueComoAntes(t *testing.T) {
	e := conObjetivos()
	e.Objetivos = nil
	for i := range e.Frentes {
		e.Frentes[i].Objetivo = ""
	}
	if md := Markdown(e, ahora, AlcanceAbiertos); strings.Contains(md, "🎯") {
		t.Fatalf("sin objetivos no hay filas 🎯:\n%s", md)
	}
}

func TestEPT_A02_ElArbolDelPanel(t *testing.T) {
	d := DatosPendientes(conObjetivos(), ahora, 10)
	var filas []string
	for _, f := range d.Frentes {
		filas = append(filas, f.Objetivo+"|"+f.Prefijo+f.ID+"|"+strings.Join(f.Notas, ","))
	}
	// receta terminó: sale del panel de pendientes y panel queda como raíz de su objetivo, sin «espera». dibujo cuelga
	// de modelo, su primera dependencia del mismo objetivo, y dice que también espera a una de otro objetivo.
	want := strings.Join([]string{
		"arbol|└─ modelo|",
		"arbol|   └─ dibujo|espera a modelo,espera a herramientas (otro objetivo)",
		"release|└─ panel|",
		"release|   └─ herramientas|espera a panel",
		"|└─ limpiar|",
	}, "\n")
	if got := strings.Join(filas, "\n"); got != want {
		t.Fatalf("cada tarea cuelga de su primera dependencia del mismo objetivo y dice a quién espera:\n got:\n%s\nwant:\n%s", got, want)
	}
	if len(d.Objetivos) != 2 || d.Objetivos[0].ID != "arbol" || !d.Objetivos[0].Foco {
		t.Fatalf("los objetivos van en orden, el foco primero: %+v", d.Objetivos)
	}
	if d.Objetivos[1].Simbolo != Espera || d.Objetivos[1].Resumen != "1/3 tareas listas" {
		t.Fatalf("cada objetivo trae su estado resuelto: %+v", d.Objetivos[1])
	}
}

func TestEPT_A02_SinObjetivosElArbolEsDeDependencias(t *testing.T) {
	e := conObjetivos()
	e.Objetivos = nil
	for i := range e.Frentes {
		e.Frentes[i].Objetivo = ""
	}
	d := DatosPendientes(e, ahora, 10)
	var filas []string
	for _, f := range d.Frentes {
		filas = append(filas, f.Prefijo+f.ID)
	}
	want := "panel / └─ herramientas / modelo / └─ dibujo / limpiar"
	if got := strings.Join(filas, " / "); got != want || len(d.Objetivos) != 0 {
		t.Fatalf("sin objetivos, las raíces van sin conector y cada tarea bajo su dependencia:\n got %s\nwant %s", got, want)
	}
}

func TestEPT_A03_CadaFilaTraeSuPregunta(t *testing.T) {
	d := DatosPendientes(conObjetivos(), ahora, 10)
	if !strings.Contains(d.Frentes[0].Pregunta, "¿Cómo vamos con «modelo»?") || !strings.Contains(d.Frentes[0].Pregunta, "bloqueos") {
		t.Fatalf("la tarea trae la pregunta que el panel le envía al agente: %q", d.Frentes[0].Pregunta)
	}
	if !strings.Contains(d.Objetivos[0].Pregunta, "«Objetivos y árbol»") || !strings.Contains(d.Objetivos[0].Pregunta, "el panel dibuja el árbol") {
		t.Fatalf("el objetivo trae su pregunta, con su criterio: %q", d.Objetivos[0].Pregunta)
	}
}

func TestFormato_SoloSubeConObjetivosODependencias(t *testing.T) {
	root := t.TempDir()
	plano := conObjetivos()
	plano.Objetivos = nil
	for i := range plano.Frentes {
		plano.Frentes[i].Objetivo, plano.Frentes[i].DependeDe = "", nil
	}
	if err := Save(root, plano); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(root, Ruta))
	if !strings.Contains(string(data), "formato: 1") {
		t.Fatal("un tablero plano sigue en formato 1: un qdd anterior lo lee igual")
	}
	if err := Save(root, conObjetivos()); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(filepath.Join(root, Ruta))
	if !strings.Contains(string(data), "formato: 2") {
		t.Fatal("con objetivos o dependencias es formato 2: un qdd anterior lo rechaza en vez de perderlos")
	}
	if _, err := Load(root); err != nil {
		t.Fatalf("este qdd lee los dos formatos: %v", err)
	}
}
