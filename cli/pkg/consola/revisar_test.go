package consola

import (
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// Pedido de la persona (2026-11-26): «actualizar, la l del menú, debería volver a revisar todo lo que está en el panel,
// porque preguntas van quedando en el camino como residuo y no aportan valor muchas veces y ya están contestadas; se
// debería volver a evaluar todo y dejar solo lo que aporta valor». qdd arma la lista completa de lo que espera a la
// persona (también lo que el tope del panel esconde) y la IA la evalúa con evidencia: decidir si algo ya se respondió
// necesita leer git, los PR y la conversación, cosa que un lector determinista no puede.

const estadoConResiduo = `formato: 1
etapas: [Código, PR]
frentes:
  - {id: t1, nombre: Uno, estado: abierto, celdas: {Código: {estado: "⏳", bloqueo: "¿Mergeamos el #8?", de_quien: la persona}, PR: {estado: "·"}}}
  - {id: t2, nombre: Dos, estado: abierto, celdas: {Código: {estado: "✅", evidencia: c1}, PR: {estado: "⏳", bloqueo: "Falta que la persona suba la imagen", de_quien: persona}}}
  - {id: t3, nombre: Tres, estado: abierto, celdas: {Código: {estado: "⏳", bloqueo: "¿Sigue en pie?", de_quien: la persona}, PR: {estado: "·"}}}
  - {id: t4, nombre: Cuatro, estado: abierto, celdas: {Código: {estado: "⏳", bloqueo: "¿Postgres o SQLite?", de_quien: la persona}, PR: {estado: "·"}}}
  - {id: t5, nombre: Cinco, estado: abierto, celdas: {Código: {estado: "⏳", bloqueo: "¿Lo subimos a latest?", de_quien: el usuario}, PR: {estado: "·"}}}
  - {id: de-la-ia, nombre: De la IA, estado: abierto, celdas: {Código: {estado: "⏳", bloqueo: "espera el CI", de_quien: la IA}, PR: {estado: "·"}}}
  - {id: cerrada, nombre: Cerrada, estado: cerrado, cerrado_el: "2026-10-29", motivo_cierre: hecho, celdas: {Código: {estado: "⏳", bloqueo: "¿Residuo viejo?", de_quien: la persona}, PR: {estado: "✅", evidencia: c2}}}
`

func proyectoConResiduo(t *testing.T) string {
	t.Helper()
	root := proyectoConCola(t)
	escribirArchivo(t, filepath.Join(root, estado.Ruta), estadoConResiduo)
	return root
}

func TestRevision_SinNadaQueEspereALaPersonaNoHayNadaQueRevisar(t *testing.T) {
	root := proyectoConTareas(t)
	cambiarTarea(t, root, "barra", "commit a2")
	if err := estado.Set(root, estado.Cambio{Frente: "barra", Etapa: "PR", Estado: "✅", Evidencia: "#8 integrado"}, enCola); err != nil {
		t.Fatal(err)
	}
	if r := Revision(root, enCola); r.Texto != "" || r.Total != 0 {
		t.Fatalf("nada espera a la persona: la l no le manda nada a la IA, y vino %+v", r)
	}
}

func TestRevision_TodoLoQueEsperaALaPersonaAunqueElPanelLoEsconda(t *testing.T) {
	root := proyectoConResiduo(t)
	r := Revision(root, enCola)
	for _, debe := range []string{
		"t1 · Código: ¿Mergeamos el #8?",
		"t2 · PR: Falta que la persona suba la imagen", // una nota sin ¿…? también es residuo posible
		"t3 · Código: ¿Sigue en pie?",
		"t4 · Código: ¿Postgres o SQLite?",
		"t5 · Código: ¿Lo subimos a latest?", // la quinta: el panel muestra 4 y esconde el resto
	} {
		if !strings.Contains(r.Texto, debe) {
			t.Fatalf("falta «%s» en la revisión:\n%s", debe, r.Texto)
		}
	}
	for _, no := range []string{"espera el CI", "¿Residuo viejo?"} {
		if strings.Contains(r.Texto, no) {
			t.Fatalf("«%s» no espera a la persona (es de la IA o de una tarea cerrada):\n%s", no, r.Texto)
		}
	}
	if r.Total != 5 {
		t.Fatalf("5 ⏳ de la persona, vino %d", r.Total)
	}
}

func TestRevision_LosPedidosSinConfirmarConSuTecla(t *testing.T) {
	root := proyectoConResiduo(t)
	p, err := Pedir(root, "echo publicar", "Publicar rc.9", "", enCola)
	if err != nil {
		t.Fatal(err)
	}
	q, err := Pedir(root, "echo viejo", "Pedido que ya no va", "", enCola)
	if err != nil {
		t.Fatal(err)
	}
	if err := Descartar(root, q.ID); err != nil {
		t.Fatal(err)
	}
	r := Revision(root, enCola)
	if !strings.Contains(r.Texto, "["+p.Tecla+"] Publicar rc.9") {
		t.Fatalf("el pedido abierto va con su tecla, para descartarlo si ya no va:\n%s", r.Texto)
	}
	if strings.Contains(r.Texto, "Pedido que ya no va") {
		t.Fatalf("un pedido descartado ya no espera a nadie:\n%s", r.Texto)
	}
	if r.Total != 6 {
		t.Fatalf("5 ⏳ y 1 pedido, vino %d", r.Total)
	}
}

func TestRevision_LeDiceALaIACómoCerrarYQueNuncaRespondaPorLaPersona(t *testing.T) {
	r := Revision(proyectoConResiduo(t), enCola)
	for _, debe := range []string{
		"qdd estado set",        // ya está respondido o hecho: ✅ con su evidencia
		"qdd estado cerrar",     // ya no va
		"qdd consola descartar", // un pedido que sobra
		"--verifica",            // lo que se responde solo deja de preguntarse
		"opción recomendada",    // lo que queda, como pregunta que se entiende sola
		"Nunca respondas por mí",
	} {
		if !strings.Contains(r.Texto, debe) {
			t.Fatalf("la revisión no dice «%s»:\n%s", debe, r.Texto)
		}
	}
}

func TestRevision_UnBloqueoLargoSeCortaParaNoGastarDeMas(t *testing.T) {
	root := proyectoConResiduo(t)
	largo := "¿" + strings.Repeat("palabra ", 200) + "?"
	if err := estado.Set(root, estado.Cambio{Frente: "t1", Etapa: "Código", Estado: "⏳", Bloqueo: largo, DeQuien: "la persona"}, enCola); err != nil {
		t.Fatal(err)
	}
	r := Revision(root, enCola)
	for _, linea := range strings.Split(r.Texto, "\n") {
		if strings.HasPrefix(linea, "- t1 ·") && (utf8.RuneCountInString(linea) > MaxTextoRevision+40 || !strings.HasSuffix(linea, "…")) {
			t.Fatalf("el bloqueo largo se corta en %d caracteres con «…»: %d", MaxTextoRevision, utf8.RuneCountInString(linea))
		}
	}
}
