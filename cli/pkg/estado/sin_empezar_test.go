package estado

import (
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// RFC 0006, blueprint estado-por-tareas: «·» es una etapa sin empezar y «—» una que no aplica (FND-085). El resumen
// de la barra y el del panel cuentan lo mismo: cada ⏳ y cada ❌ de las tareas pendientes (FND-086).

// FND-085: una tarea nueva con solo «Código ✅» llenaba las otras etapas con «—» (no aplica), quedaba «terminada» y
// salía del panel en el acto, aunque no tuviera pruebas ni producción.
func TestSet_TareaNuevaQuedaSinEmpezarYSigueEnElPanel_FND085(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir, []string{"Código", "Pruebas", "Producción"}, ahora); err != nil {
		t.Fatal(err)
	}
	if err := Set(dir, Cambio{Frente: "login", Nombre: "Login", Etapa: "Código", Estado: "ok", Evidencia: "commit abc123"}, ahora); err != nil {
		t.Fatalf("set: %v", err)
	}
	e, _ := Load(dir)
	f := e.Frentes[0]
	if f.Celdas["Pruebas"].Estado != SinEmpezar || f.Celdas["Producción"].Estado != SinEmpezar {
		t.Fatalf("lo que no se tocó queda sin empezar: %+v", f.Celdas)
	}
	if f.Terminado(e.Etapas) {
		t.Fatal("una tarea con etapas sin empezar no está terminada")
	}
	d := DatosPendientes(e, ahora, 4)
	if len(d.Frentes) != 1 || strings.Join(d.Frentes[0].Simbolos, "") != "✅··" {
		t.Fatalf("sigue en el panel con «·» en lo que falta: %+v", d.Frentes)
	}
	if vs := Check(e, nil); len(vs) != 0 {
		t.Fatalf("«·» es un estado válido: %v", vs)
	}
	if err := Set(dir, Cambio{Frente: "login", Etapa: "Pruebas", Estado: "na"}, ahora); err != nil {
		t.Fatal(err)
	}
	if err := Set(dir, Cambio{Frente: "login", Etapa: "Producción", Estado: "sin-empezar"}, ahora); err != nil {
		t.Fatalf("«sin-empezar» es el alias escribible de «·»: %v", err)
	}
	e, _ = Load(dir)
	if e.Frentes[0].Celdas["Pruebas"].Estado != NoAplica || e.Frentes[0].Terminado(e.Etapas) {
		t.Fatalf("«—» sigue diciendo no aplica y «·» no termina: %+v", e.Frentes[0].Celdas)
	}
	md := Markdown(e, ahora, AlcanceAbiertos)
	if vs := CheckRespuesta(e, md, ahora); len(vs) != 0 || !strings.Contains(md, "| "+SinEmpezar+" |") {
		t.Fatalf("la tabla muestra «·» y coincide con el archivo: %v\n%s", vs, md)
	}
	if AnchoVisible(SinEmpezar) != 1 {
		t.Fatal("«·» ocupa una columna, como «—»")
	}
}

// FND-086: la barra contaba tareas con algún ⏳ (◧ 1 tarea · ⏳1) y el panel contaba cada ⏳ de las tareas visibles
// (⏳ 2 esperan): la persona veía dos números distintos para lo mismo.
func TestResumen_LaBarraYElPanelCuentanLoMismo_FND086(t *testing.T) {
	yaml := "formato: 1\netapas: [Código, Pruebas]\nfrentes:\n" +
		"  - {id: a, nombre: A, estado: abierto, celdas: {Código: {estado: ⏳, bloqueo: revisión, de_quien: usuario}, Pruebas: {estado: ⏳, bloqueo: CI, de_quien: CI}}}\n" +
		"  - {id: b, nombre: B, estado: abierto, celdas: {Código: {estado: ❌, falla: no compila}, Pruebas: {estado: ·}}}\n" +
		"  - {id: c, nombre: C, estado: abierto, celdas: {Código: {estado: ✅, evidencia: abc}, Pruebas: {estado: —}}}\n"
	e := mustParse(t, yaml)
	if got := Statusline(e, ahora); got != "◧ 2 tareas · ⏳2 ❌1" {
		t.Fatalf("la barra cuenta cada ⏳ y cada ❌ de las tareas pendientes (la terminada no): %q", got)
	}
	d := DatosPendientes(e, ahora, 1)
	if d.Esperan != 2 || d.Fallan != 1 || d.Tareas != 2 {
		t.Fatalf("el panel trae los mismos totales, aunque una tarea no entre: esperan=%d fallan=%d tareas=%d", d.Esperan, d.Fallan, d.Tareas)
	}
	if len(d.Frentes)+d.Ocultos != d.Tareas {
		t.Fatalf("las tareas del panel son las visibles más las ocultas: %d + %d", len(d.Frentes), d.Ocultos)
	}
	terminadas := mustParse(t, "formato: 1\netapas: [Código]\nfrentes:\n  - {id: c, nombre: C, estado: abierto, celdas: {Código: {estado: ✅, evidencia: abc}}}\n")
	if got := Statusline(terminadas, ahora); got != "◧ sin pendientes · ✅" {
		t.Fatalf("con todo terminado la barra lo dice: %q", got)
	}
}

// Blueprint estado-por-tareas: la palabra es «tarea» (antes «frente»). La tabla y el panel la usan, y una respuesta con
// la tabla vieja («| Frente |») se sigue leyendo.
func TestTextos_DicenTarea(t *testing.T) {
	e := mustParse(t, valido)
	md := Markdown(e, ahora, AlcanceAbiertos)
	if !strings.HasPrefix(md, "| Tarea |") {
		t.Fatalf("la tabla dice «Tarea»: %s", md)
	}
	if vs := CheckRespuesta(e, strings.Replace(md, "| Tarea |", "| Frente |", 1), ahora); len(vs) != 0 {
		t.Fatalf("una tabla con el encabezado viejo se sigue leyendo: %v", vs)
	}
	todo := strings.Join(Panel(e, "p", 80, ahora), "\n")
	if strings.Contains(strings.ToLower(todo), "frente") || !strings.Contains(todo, "Estado por tarea") {
		t.Fatalf("el panel de watch dice «tarea»: %s", todo)
	}
	dir := t.TempDir()
	if err := Init(dir, []string{"Código"}, ahora); err != nil {
		t.Fatal(err)
	}
	if err := Cerrar(dir, "x", "", false, ahora); err == nil || strings.Contains(err.Error(), "frente") {
		t.Fatalf("los errores dicen «tarea»: %v", err)
	}
	if err := Cerrar(dir, "x", "m", false, ahora); err == nil || strings.Contains(err.Error(), "frente") {
		t.Fatalf("los errores dicen «tarea»: %v", err)
	}
}
