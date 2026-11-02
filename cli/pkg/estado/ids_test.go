package estado

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// FND-109 (notificador_de_ofertas, 2026-11-02): la IA escribió 'qdd estado set segmento-barra …' y 'set qdd-estado …'
// para tareas que ya existían como barra-estado y estado-por-frente; set las creó en silencio (con el id de nombre) y
// hubo que descartarlas a mano. Regla de QDD: nombrar una tarea u objetivo nunca lo crea. Crear es explícito (qdd estado
// nueva, o --nombre) y qdd da o valida el id; una referencia que no existe es un error que sugiere la más parecida.

const notificador = `formato: 2
etapas: [Código, Pruebas]
objetivos:
  - {id: gemini, nombre: Retirar Gemini, criterio: sin llamadas a Gemini, estado: abierto}
frentes:
  - {id: barra-estado, nombre: Segmento de estado en la barra (consola-claude-code v1.1.0), estado: abierto, celdas: {Código: {estado: ·}, Pruebas: {estado: ·}}}
  - {id: estado-por-frente, nombre: qdd estado (RFC 0006), estado: abierto, celdas: {Código: {estado: ·}, Pruebas: {estado: ·}}}
  - {id: claves-gemini, nombre: Quitar claves de Gemini (GCP y GitHub), estado: abierto, objetivo: gemini, celdas: {Código: {estado: ·}, Pruebas: {estado: ·}}}
`

func proyectoNotificador(t *testing.T) string {
	t.Helper()
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	ruta := filepath.Join(root, Ruta)
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte(notificador), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func cantidadDeTareas(t *testing.T, root string) int {
	t.Helper()
	e, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return len(e.Frentes)
}

func TestIDs_UnIDDesconocidoNoCreaLaTareaYSugiereLaQueEra_FND109(t *testing.T) {
	root := proyectoNotificador(t)
	casos := []struct{ escrito, era string }{
		{"segmento-barra", "barra-estado"},
		{"qdd-estado", "estado-por-frente"},
		{"barra-estdo", "barra-estado"},
	}
	for _, c := range casos {
		err := Set(root, Cambio{Frente: c.escrito, Etapa: "Código", Estado: "ok", Evidencia: "commit abc"}, ahora)
		if err == nil || !strings.Contains(err.Error(), "¿quisiste decir "+c.era) || !strings.Contains(err.Error(), "qdd estado nueva") {
			t.Fatalf("%s: se rechaza y sugiere %s y cómo crear una: %v", c.escrito, c.era, err)
		}
	}
	if n := cantidadDeTareas(t, root); n != 3 {
		t.Fatalf("no se creó ninguna tarea fantasma: %d", n)
	}
}

func TestIDs_ADosLetrasDeUnIDImparTambienSeParece_FND109(t *testing.T) {
	// «menos de la mitad del largo del id»: con 5 letras, 2 de distancia es menos que 2,5 (no se divide entero).
	e := mustParse(t, "formato: 1\netapas: [Código]\nfrentes:\n  - {id: barra, nombre: Barra, estado: abierto, celdas: {Código: {estado: ·}}}\n")
	if err := e.ValidarTarea("bxrrx"); err == nil || !strings.Contains(err.Error(), "¿quisiste decir barra") {
		t.Fatalf("a 2 letras de un id de 5 se sugiere: %v", err)
	}
}

func TestIDs_SinNadaParecidoListaLasAbiertas_FND109(t *testing.T) {
	root := proyectoNotificador(t)
	err := Set(root, Cambio{Frente: "xyz", Etapa: "Código", Estado: "ok", Evidencia: "c"}, ahora)
	if err == nil || !strings.Contains(err.Error(), "abiertas: barra-estado, estado-por-frente, claves-gemini") {
		t.Fatalf("sin una parecida, dice cuáles hay: %v", err)
	}
}

func TestIDs_CerrarActivarYDependerTambienSugieren_FND109(t *testing.T) {
	root := proyectoNotificador(t)
	if err := Cerrar(root, "qdd-estado", "listo", false, ahora); err == nil || !strings.Contains(err.Error(), "estado-por-frente") {
		t.Fatalf("cerrar sugiere: %v", err)
	}
	if err := Activar(root, "segmento-barra", "claude-code", ahora); err == nil || !strings.Contains(err.Error(), "barra-estado") {
		t.Fatalf("activar sugiere: %v", err)
	}
	err := GuardarTarea(root, CambioTarea{ID: "barra-estado", DependeDe: []string{"qdd-estado"}, CambiaDependencias: true}, ahora)
	if err == nil || !strings.Contains(err.Error(), "¿quisiste decir estado-por-frente") {
		t.Fatalf("una dependencia desconocida sugiere: %v", err)
	}
}

func TestIDs_NuevaDaElIDDesdeElNombre_FND109(t *testing.T) {
	root := proyectoNotificador(t)
	id, err := Nueva(root, CambioNueva{Nombre: "Ticket de MP en logs de error", Objetivo: "gemini"}, ahora)
	if err != nil || id != "ticket-mp-logs-error" {
		t.Fatalf("el id es el slug del nombre, sin palabras vacías ni tildes: %q %v", id, err)
	}
	e, _ := Load(root)
	f := e.Frentes[e.buscar(id)]
	if f.Nombre != "Ticket de MP en logs de error" || f.Objetivo != "gemini" || f.Estado != EstadoAbierto || len(f.Celdas) != 2 {
		t.Fatalf("queda abierta, con su objetivo y sus etapas en «·»: %+v", f)
	}
	otra, err := Nueva(root, CambioNueva{Nombre: "Ticket de MP en logs de error (otra vez)", AunqueExista: true}, ahora)
	if err != nil || otra != "ticket-mp-logs-error-2" {
		t.Fatalf("un id tomado lleva sufijo: %q %v", otra, err)
	}
	if id, err := Nueva(root, CambioNueva{Nombre: "Área de diseño: Ñandú"}, ahora); err != nil || id != "area-diseno-nandu" {
		t.Fatalf("sin tildes ni eñes: %q %v", id, err)
	}
}

func TestIDs_UnaTareaCasiIgualAUnaAbiertaSePideConfirmar_FND109(t *testing.T) {
	root := proyectoNotificador(t)
	_, err := Nueva(root, CambioNueva{Nombre: "Quitar claves de Gemini"}, ahora)
	if err == nil || !strings.Contains(err.Error(), "claves-gemini") || !strings.Contains(err.Error(), "--aunque-exista") {
		t.Fatalf("se parece a una abierta: se rechaza y la nombra: %v", err)
	}
	if _, err := Nueva(root, CambioNueva{Nombre: "Quitar claves de Gemini", AunqueExista: true}, ahora); err != nil {
		t.Fatalf("con --aunque-exista se crea: %v", err)
	}
}

func TestIDs_SetConNombreSigueCreandoConUnIDValido_FND109(t *testing.T) {
	root := proyectoNotificador(t)
	if err := Set(root, Cambio{Frente: "aviso-staffing", Nombre: "Aviso «Staffing plan parse» engañoso", Etapa: "Código", Estado: "ok", Evidencia: "commit abc"}, ahora); err != nil {
		t.Fatalf("con --nombre se crea, como antes: %v", err)
	}
	for _, malo := range []string{"Aviso Staffing", "aviso_staffing", "-aviso", "aviso--staffing", strings.Repeat("a", 41)} {
		err := Set(root, Cambio{Frente: malo, Nombre: "Otra cosa distinta " + malo, Etapa: "Código", Estado: "·"}, ahora)
		if err == nil || !strings.Contains(err.Error(), "id") {
			t.Fatalf("%q no es un id de QDD (minúsculas, números y guiones; ≤40): %v", malo, err)
		}
	}
	if err := Set(root, Cambio{Frente: "otra-barra", Nombre: "Segmento de estado en la barra", Etapa: "Código", Estado: "·"}, ahora); err == nil || !strings.Contains(err.Error(), "barra-estado") {
		t.Fatalf("set con --nombre también avisa la casi igual: %v", err)
	}
}

func TestIDs_ObjetivoDesconocidoNoSeCreaSinNombreYSugiere_FND109(t *testing.T) {
	root := proyectoNotificador(t)
	if err := GuardarObjetivo(root, CambioObjetivo{ID: "gemni", Foco: true}, ahora); err == nil || !strings.Contains(err.Error(), "¿quisiste decir gemini") {
		t.Fatalf("un objetivo mal escrito no se crea: %v", err)
	}
	if err := CerrarObjetivo(root, "retirar-gemini", "evidencia", false, ahora); err == nil || !strings.Contains(err.Error(), "gemini") {
		t.Fatalf("cerrar un objetivo sugiere por el nombre: %v", err)
	}
	err := GuardarTarea(root, CambioTarea{ID: "barra-estado", Objetivo: "gemni", CambiaObjetivo: true}, ahora)
	if err == nil || !strings.Contains(err.Error(), "¿quisiste decir gemini") {
		t.Fatalf("asignar un objetivo desconocido sugiere: %v", err)
	}
	if err := GuardarObjetivo(root, CambioObjetivo{ID: "Release 2", Nombre: "Release", Criterio: "c"}, ahora); err == nil || !strings.Contains(err.Error(), "id") {
		t.Fatalf("un objetivo nuevo también lleva un id de QDD: %v", err)
	}
	if err := GuardarObjetivo(root, CambioObjetivo{ID: "release", Nombre: "Release", Criterio: "c"}, ahora); err != nil {
		t.Fatalf("con nombre y criterio se crea: %v", err)
	}
}

func TestIDs_CicloDesconocidoSugiere_FND109(t *testing.T) {
	root := proyectoNotificador(t)
	if err := DeclararCiclo(root, "agentico", []string{"Concepto", "Puerta"}, ahora); err != nil {
		t.Fatal(err)
	}
	err := GuardarObjetivo(root, CambioObjetivo{ID: "gemini", Ciclo: "agentic", CambiaCiclo: true}, ahora)
	if err == nil || !strings.Contains(err.Error(), "¿quisiste decir agentico") {
		t.Fatalf("un ciclo mal escrito sugiere: %v", err)
	}
}

func TestIDs_ElCheckNoRechazaIDsViejos_FND109(t *testing.T) {
	e := mustParse(t, "formato: 1\netapas: [Código]\nfrentes:\n  - {id: Viejo_ID, nombre: Viejo, estado: abierto, celdas: {Código: {estado: ·}}}\n")
	for _, v := range Check(e, nil) {
		if strings.Contains(v.Mensaje, "id") && v.Frente == "Viejo_ID" {
			t.Fatalf("un id de un archivo anterior sigue valiendo: el formato se exige al crear: %v", v)
		}
	}
	_ = time.Now
}

func TestIDs_DoctorVeLasTareasQueParecenFantasma_FND109(t *testing.T) {
	e := mustParse(t, notificador+`  - {id: qdd-estado, nombre: qdd-estado, estado: abierto, celdas: {Código: {estado: ✅, evidencia: x}, Pruebas: {estado: ·}}}
  - {id: quitar-claves, nombre: Quitar las claves de Gemini, estado: abierto, celdas: {Código: {estado: ·}, Pruebas: {estado: ·}}}
  - {id: vieja, nombre: vieja, estado: descartado, motivo_cierre: x, cerrado_el: "2026-11-01T00:00:00Z", celdas: {Código: {estado: ·}, Pruebas: {estado: ·}}}
`)
	got := strings.Join(Sospechosas(e), "; ")
	if !strings.Contains(got, "qdd-estado se llama como su id") || !strings.Contains(got, "claves-gemini y quitar-claves casi se llaman igual") || strings.Contains(got, "vieja") {
		t.Fatalf("ve la que se llama como su id y las casi iguales, solo entre las abiertas: %q", got)
	}
	// Una palabra repetida en un nombre largo no lo hace parecido a uno corto (falso positivo visto en qdd: unir).
	repetida := mustParse(t, "formato: 1\netapas: [Código]\nfrentes:\n  - {id: unir, nombre: 'Un solo blueprint: estado-por-tareas (une estado-por-frente)', estado: abierto, celdas: {Código: {estado: ·}}}\n  - {id: bp-estado, nombre: Aplicar el blueprint de estado, estado: abierto, celdas: {Código: {estado: ·}}}\n")
	if s := Sospechosas(repetida); len(s) != 0 {
		t.Fatalf("cada palabra cuenta una vez: %v", s)
	}
	if s := Sospechosas(mustParse(t, notificador)); len(s) != 0 {
		t.Fatalf("un estado sano no tiene sospechosas: %v", s)
	}
}
