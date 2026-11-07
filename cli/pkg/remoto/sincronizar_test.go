package remoto

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/flujo"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// «Sincronizar» (la persona por qdd-movile, 2026-11-07): releer todo el PC, volver a publicarlo y mostrar, sin corregir
// nada, lo que el tablero ya no dice bien.

const estadoInconsistente = `formato: 1
etapas: [Código, PR]
frentes:
  - {id: lista, nombre: Lista, estado: abierto, celdas: {Código: {estado: "✅", evidencia: a}, PR: {estado: "✅", evidencia: b}}}
  - {id: tirada, nombre: Tirada, estado: descartado, motivo_cierre: ya no va, celdas: {Código: {estado: "⏳", bloqueo: y, de_quien: la persona}}}
  - {id: barra, nombre: Barra, estado: abierto, celdas: {Código: {estado: "✅", evidencia: c}, PR: {estado: "⏳", bloqueo: "¿Confirmas [a]?", de_quien: la persona}}}
`

func proyectoInconsistente(t *testing.T) string {
	t.Helper()
	root := proyecto(t, "raro", false)
	escribirEn(t, filepath.Join(root, estado.Ruta), estadoInconsistente)
	escribirEn(t, filepath.Join(root, flujo.Carpeta, "roto.yaml"), "formato: [no es un flujo\n")
	// El pedido [a] de la tarea barra terminó bien hace un rato (lo anota pkg/consola al terminar).
	ruta, err := estado.RutaLocal(root, "pedidos-hechos")
	if err != nil {
		t.Fatal(err)
	}
	hechos := `[{"efecto":"x","motivo":"Mergear","fin":"` + time.Now().Add(-time.Hour).UTC().Format(time.RFC3339) + `","tecla":"a","tarea":"barra"}]`
	escribirEn(t, ruta, hechos)
	return root
}

func escribirEn(t *testing.T, ruta, texto string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte(texto), 0o600); err != nil {
		t.Fatal(err)
	}
}

func tipos(is []Inconsistencia) map[string]Inconsistencia {
	out := map[string]Inconsistencia{}
	for _, i := range is {
		out[i.Tipo] = i
	}
	return out
}

func TestRevisar_DiceLoQueElTableroYaNoDiceBienSinCambiarNada(t *testing.T) {
	homeTemporal(t)
	root := proyectoInconsistente(t)
	antes, _ := os.ReadFile(filepath.Join(root, estado.Ruta))
	got := tipos(Revisar(root, time.Now()))
	if len(got) != 4 {
		t.Fatalf("cuatro clases de inconsistencia: %+v", got)
	}
	if i := got[InconsistenciaEsperaTrasPedido]; i.Tarea != "Barra" || i.Etapa != "PR" || i.Detalle != "¿Confirmas [a]?" {
		t.Fatalf("el ⏳ cuyo pedido ya terminó: %+v", i)
	}
	if i := got[InconsistenciaListaSinCerrar]; i.Tarea != "Lista" {
		t.Fatalf("la tarea con todo listo que sigue abierta, por su nombre: %+v", i)
	}
	if i := got[InconsistenciaTerminadaConEspera]; i.Tarea != "Tirada" {
		t.Fatalf("la descartada que conserva un ⏳: %+v", i)
	}
	if i := got[InconsistenciaFlujoInvalido]; len(i.Detalle) < 6 || i.Detalle[:6] != "roto: " {
		t.Fatalf("el flujo que no se puede usar, con su problema: %+v", i)
	}
	despues, _ := os.ReadFile(filepath.Join(root, estado.Ruta))
	if string(antes) != string(despues) {
		t.Fatal("revisar no escribe nada")
	}
}

func TestRevisar_UnProyectoSanoNoTieneNadaYEsUnaListaVacia(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "sano", true)
	if got := Revisar(root, time.Now()); got == nil || len(got) != 0 {
		t.Fatalf("[] y no null: %+v", got)
	}
}

func TestSincronizar_ReenviaElEstadoYDespuesDiceQueEncontroEnCadaProyectoConConsola(t *testing.T) {
	homeTemporal(t)
	raro, quieto := proyectoInconsistente(t), proyecto(t, "quieto", true)
	registro.Anotar(raro, time.Now())
	registro.Anotar(quieto, time.Now().Add(-time.Minute))
	if err := consola.LatirSesion(raro, "s1", time.Now()); err != nil {
		t.Fatal(err)
	}
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefono("d1", "")
	estadoBase(t, tr)
	estados := tr.contar("estado")
	tr.eventos <- Evento{Tipo: TipoBroadcast, Evento: "sincronizar"}
	e := tr.esperar(t, 3*time.Second, "sincronizado")
	if tr.contar("estado") != estados+1 {
		t.Fatalf("antes de responder manda el estado completo: %d → %d", estados, tr.contar("estado"))
	}
	enviados := tr.Enviados()
	if enviados[len(enviados)-1].Evento != "sincronizado" {
		t.Fatalf("sincronizado es lo último: %+v", enviados[len(enviados)-1])
	}
	var s Sincronizado
	if err := json.Unmarshal(e.Payload, &s); err != nil {
		t.Fatal(err)
	}
	if s.Version != VersionProtocolo || !s.EstadoEnviado || len(s.Proyectos) != 2 {
		t.Fatalf("%+v", s)
	}
	porNombre := map[string]ProyectoSincronizado{}
	for _, p := range s.Proyectos {
		porNombre[p.Nombre] = p
	}
	if p := porNombre["raro"]; !p.Consola || len(p.Inconsistencias) != 4 || p.Clave == "" {
		t.Fatalf("con una consola abierta se revisa: %+v", p)
	}
	if p := porNombre["quieto"]; p.Consola || p.Inconsistencias == nil || len(p.Inconsistencias) != 0 {
		t.Fatalf("sin consola abierta no se revisa, y la lista es []: %+v", p)
	}
}

// Pedido de qdd-movile (2026-11-07): el teléfono muestra qué proyecto tiene las reglas de las IAs atrasadas. Sin el campo
// no se midió (la carpeta no está); [] es al día.
func TestArmar_CadaProyectoDiceSusReglasAtrasadas(t *testing.T) {
	homeTemporal(t)
	viejo, alDia, borrado := proyecto(t, "viejo", true), proyecto(t, "al-dia", true), proyecto(t, "borrado", true)
	escribirEn(t, filepath.Join(viejo, "AGENTS.md"), "### QDD AUTOGENERATED COMMANDS BEGIN ###\n# QCP block revision: 1\n### QDD AUTOGENERATED COMMANDS END ###\n")
	registro.Anotar(viejo, time.Now())
	registro.Anotar(alDia, time.Now().Add(-time.Minute))
	registro.Anotar(borrado, time.Now().Add(-2*time.Minute))
	os.RemoveAll(borrado)
	reg, _ := registro.Leer()
	b, _ := json.Marshal(Armar("v", reg, time.Now(), 0).Proyectos)
	var ps []map[string]json.RawMessage
	json.Unmarshal(b, &ps)
	if got := string(ps[0]["reglas_atrasadas"]); got != `["AGENTS.md"]` {
		t.Fatalf("viejo: %s", got)
	}
	if got := string(ps[1]["reglas_atrasadas"]); got != `[]` {
		t.Fatalf("al día es [], no falta el campo: %s", got)
	}
	if _, hay := ps[2]["reglas_atrasadas"]; hay {
		t.Fatalf("sin carpeta no se midió: %s", ps[2]["reglas_atrasadas"])
	}
}

func TestEstructura_UnCambioDeReglasPideElEstadoCompleto(t *testing.T) {
	antes := Estado{Proyectos: []ProyectoEstado{{Clave: "a", Nombre: "a", ReglasAtrasadas: &[]string{"AGENTS.md"}}}}
	ahora := Estado{Proyectos: []ProyectoEstado{{Clave: "a", Nombre: "a", ReglasAtrasadas: &[]string{}}}}
	if !pideEstado(estructuraDe(antes), estructuraDe(ahora)) {
		t.Fatal("un delta no lleva las reglas: renovarlas manda el estado completo")
	}
}
