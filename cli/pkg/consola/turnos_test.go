package consola

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'

const estadoBaseEncima = `formato: 1
etapas:
  - Código
frentes:
  - {id: base, nombre: Base, estado: abierto, celdas: {Código: {estado: "✅", evidencia: x}}}
  - {id: encima, nombre: Encima, estado: abierto, depende_de: [base], celdas: {Código: {estado: "✅", evidencia: x}}}
`

// conDosEsperas deja un ⏳ de la persona en cada tarea; encima depende de base.
func conDosEsperas(t *testing.T, root, pregunta string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, estado.Ruta)); err != nil {
		escribirArchivo(t, filepath.Join(root, estado.Ruta), estadoBaseEncima)
	}
	for id, bloqueo := range map[string]string{"base": pregunta, "encima": "¿Y esto otro?"} {
		if err := estado.Set(root, estado.Cambio{Frente: id, Etapa: "Código", Estado: "⏳", Bloqueo: bloqueo, DeQuien: "la persona"}, enCola); err != nil {
			t.Fatal(err)
		}
	}
}

func conTurno(acciones []Accion) []Accion {
	var out []Accion
	for _, a := range acciones {
		if a.Turno > 0 {
			out = append(out, a)
		}
	}
	return out
}

func TestTurnos_SeDanUnaVezYNoCambianEntreLecturas(t *testing.T) {
	root := proyectoConCola(t)
	a := mustPedir(t, root, "uno", "Uno", "")
	primera := conTurno(Panel(root, enCola).Acciones)
	if len(primera) != 1 || primera[0].Turno != 1 || primera[0].TurnoDesde == nil || !primera[0].TurnoDesde.Equal(enCola) {
		t.Fatalf("el primero lleva el 1 y su fecha: %+v", primera)
	}
	b := mustPedir(t, root, "dos", "Dos", "")
	segunda := conTurno(Panel(root, enCola.Add(time.Hour)).Acciones)
	if len(segunda) != 2 || segunda[0].pedido != a.ID || segunda[0].Turno != 1 || segunda[1].pedido != b.ID || segunda[1].Turno != 2 {
		t.Fatalf("el que ya tenía número lo conserva: %+v", segunda)
	}
	if !segunda[0].TurnoDesde.Equal(enCola) {
		t.Fatalf("la fecha es la de la primera vez: %v", segunda[0].TurnoDesde)
	}
}

func TestTurnos_UnNumeroNoSeReutiliza(t *testing.T) {
	root := proyectoConCola(t)
	a := mustPedir(t, root, "uno", "Uno", "")
	Panel(root, enCola)
	if err := HechoPorLaPersona(root, a.ID, "", enCola); err != nil {
		t.Fatal(err)
	}
	Panel(root, enCola)
	mustPedir(t, root, "dos", "Dos", "")
	got := conTurno(Panel(root, enCola).Acciones)
	if len(got) != 1 || got[0].Turno != 2 {
		t.Fatalf("el nuevo lleva el 2: %+v", got)
	}
}

func TestTurnos_SiguienteEsLoPrimeroQueSePuedeContestar(t *testing.T) {
	root := proyectoConCola(t)
	mustPedir(t, root, "uno", "Uno", "")
	mustPedir(t, root, "dos", "Dos", "")
	got := conTurno(Panel(root, enCola).Acciones)
	if !got[0].Siguiente || got[1].Siguiente {
		t.Fatalf("solo el primero: %+v", got)
	}
}

func TestTurnos_LoQueSeEsperaVaAntesYQuedaMarcado(t *testing.T) {
	root := proyectoConCola(t)
	tarde := mustPedir(t, root, "merge", "Merge", "")
	if _, err := PedirEnOrden(root, "tag", "Tag", "", Orden{Antes: []string{tarde.ID}}, enCola); err != nil {
		t.Fatal(err)
	}
	got := conTurno(Panel(root, enCola).Acciones)
	if len(got) != 2 || got[0].Etiqueta != "Tag" || got[1].Etiqueta != "Merge" {
		t.Fatalf("lo que espera va después: %+v", got)
	}
	if len(got[1].EsperaTurno) != 1 || got[1].EsperaTurno[0] != got[0].Turno {
		t.Fatalf("dice qué turno espera: %+v", got[1])
	}
	if !got[0].Siguiente || got[1].Siguiente {
		t.Fatalf("sigue lo esperado: %+v", got)
	}
}

func TestTurnos_UnaTareaEsperaLoAbiertoDeSuDependeDe(t *testing.T) {
	root := proyectoConCola(t)
	conDosEsperas(t, root, "¿Sigo?")
	got := conTurno(Panel(root, enCola, TipoPregunta).Acciones)
	if len(got) != 2 || got[0].Etiqueta == got[1].Etiqueta {
		t.Fatalf("dos ⏳ con turno: %+v", got)
	}
	if len(got[1].EsperaTurno) != 1 || got[1].EsperaTurno[0] != got[0].Turno {
		t.Fatalf("encima espera a base: %+v", got)
	}
}

func TestTurnos_ReescribirElTextoDeUnaEsperaNoLeCambiaElNumero(t *testing.T) {
	root := proyectoConCola(t)
	conDosEsperas(t, root, "¿Sigo?")
	antes := conTurno(Panel(root, enCola, TipoPregunta).Acciones)
	conDosEsperas(t, root, "¿Sigo con esto?")
	despues := conTurno(Panel(root, enCola, TipoPregunta).Acciones)
	if len(antes) != 2 || len(despues) != 2 || despues[0].Turno != antes[0].Turno || despues[0].ID == antes[0].ID {
		t.Fatalf("otra pregunta, el mismo turno: %+v → %+v", antes[0], despues[0])
	}
}

func TestTurnos_UnPedidoConfirmadoConservaElNumeroYDejaDeSerSiguiente(t *testing.T) {
	root := proyectoConCola(t)
	a := mustPedir(t, root, "uno", "Uno", "")
	mustPedir(t, root, "dos", "Dos", "")
	Panel(root, enCola)
	if _, err := aceptar(root, a.ID, "", time.Now()); err != nil {
		t.Fatal(err)
	}
	got := conTurno(Panel(root, enCola).Acciones)
	if got[0].pedido != a.ID || got[0].Turno != 1 || got[0].Siguiente || !got[1].Siguiente {
		t.Fatalf("armado no se contesta otra vez: %+v", got)
	}
}

func TestTurnos_UnaLecturaSinCambiosNoTocaElRegistro(t *testing.T) {
	root := proyectoConCola(t)
	mustPedir(t, root, "uno", "Uno", "")
	Panel(root, enCola)
	ruta, err := RutaLocal(root, "turnos")
	if err != nil {
		t.Fatal(err)
	}
	antes, err := os.Stat(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta+".lock", nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := conTurno(Panel(root, enCola).Acciones); len(got) != 1 {
		t.Fatalf("con el candado tomado, lee sin esperarlo: %+v", got)
	}
	despues, err := os.Stat(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if !despues.ModTime().Equal(antes.ModTime()) {
		t.Fatal("no escribe si nada cambió")
	}
}

func TestTurnos_LoQueNoEsperaRespuestaNoLleva(t *testing.T) {
	root := proyectoConCola(t)
	escribirArchivo(t, filepath.Join(root, estado.Ruta), estadoBaseEncima)
	for _, a := range Panel(root, enCola, TipoAtender, TipoPregunta, ConTelefono).Acciones {
		if a.Turno != 0 {
			t.Fatalf("%s no espera a la persona: %+v", a.ID, a)
		}
	}
}

func TestTurnos_UnPanelQueNoPidePreguntasVeLosMismosNumeros(t *testing.T) {
	root := proyectoConCola(t)
	conDosEsperas(t, root, "¿Sigo?")
	mustPedir(t, root, "uno", "Uno", "")
	sin := conTurno(Panel(root, enCola).Acciones)
	con := conTurno(Panel(root, enCola, TipoPregunta).Acciones)
	if len(sin) != 1 || len(con) != 3 {
		t.Fatalf("sin: %+v con: %+v", sin, con)
	}
	for _, a := range con {
		if a.pedido != "" && a.Turno != sin[0].Turno {
			t.Fatalf("el pedido lleva el mismo número: %d y %d", a.Turno, sin[0].Turno)
		}
	}
}

func TestTurnos_UnRegistroIlegibleSeApartaYSeSigue(t *testing.T) {
	root := proyectoConCola(t)
	ruta, err := RutaLocal(root, "turnos")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte("{roto"), 0o600); err != nil {
		t.Fatal(err)
	}
	mustPedir(t, root, "uno", "Uno", "")
	if got := conTurno(Panel(root, enCola).Acciones); len(got) != 1 || got[0].Turno != 1 {
		t.Fatalf("empieza de nuevo: %+v", got)
	}
	if got := ColasApartadas(root); len(got) != 1 {
		t.Fatalf("doctor lo nombra: %v", got)
	}
}

func TestTurnos_ConElCandadoOcupadoElPanelSaleSinTurnos(t *testing.T) {
	root := proyectoConCola(t)
	mustPedir(t, root, "uno", "Uno", "")
	ruta, err := RutaLocal(root, "turnos")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta+".lock", nil, 0o600); err != nil {
		t.Fatal(err)
	}
	got := Panel(root, enCola).Acciones
	if len(got) == 0 || len(conTurno(got)) != 0 {
		t.Fatalf("el panel se dibuja igual: %+v", got)
	}
}
