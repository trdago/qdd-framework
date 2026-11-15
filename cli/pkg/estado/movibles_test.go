package estado

import (
	"testing"
)

// Rule: Prohibido usar 'else'
// MEJ-055 (FND-216): el modo goal empujaba con «quedan N tareas» aunque todas esperaran a la persona o a terceros. Una
// tarea la puede mover la IA solo si su primera etapa sin terminar es suya: sin empezar, fallida, o un ⏳ de la IA.

const tableroMovibles = `formato: 1
etapas: [Código, Pruebas, PR]
frentes:
  - {id: libre, nombre: Libre, estado: abierto, celdas: {Código: {estado: "✅"}, Pruebas: {estado: "·"}}}
  - {id: persona, nombre: Persona, estado: abierto, celdas: {Código: {estado: "⏳", bloqueo: "¿Resend o SES?", de_quien: persona}, Pruebas: {estado: "·"}}}
  - {id: tras-persona, nombre: TrasPersona, estado: abierto, celdas: {Código: {estado: "✅"}, Pruebas: {estado: "⏳", bloqueo: "¿Lo ves?", de_quien: la persona}, PR: {estado: "·"}}}
  - {id: de-ia, nombre: DeIA, estado: abierto, celdas: {Código: {estado: "⏳", bloqueo: fase B, de_quien: la IA}}}
  - {id: nosotros, nombre: Nosotros, estado: abierto, celdas: {Código: {estado: "⏳", bloqueo: medir CER, de_quien: nosotros}}}
  - {id: sesion, nombre: Sesion, estado: abierto, celdas: {Código: {estado: "⏳", bloqueo: sigue, de_quien: qdd-6e}}}
  - {id: otro-proyecto, nombre: OtroProyecto, estado: abierto, celdas: {Código: {estado: "⏳", bloqueo: app, de_quien: qdd-movile-13}}}
  - {id: tercero, nombre: Tercero, estado: abierto, celdas: {Código: {estado: "⏳", bloqueo: plazo, de_quien: INRPAC}}}
  - {id: verifica, nombre: Verifica, estado: abierto, celdas: {Código: {estado: "⏳", bloqueo: CI, de_quien: la IA, verifica: "pr:9"}}}
  - {id: falla, nombre: Falla, estado: abierto, celdas: {Código: {estado: "❌", falla: rojo}}}
  - {id: lista, nombre: Lista, estado: abierto, celdas: {Código: {estado: "✅"}, Pruebas: {estado: "✅"}, PR: {estado: "—"}}}
  - {id: espera-a-otra, nombre: EsperaAOtra, estado: abierto, depende_de: [persona], celdas: {Código: {estado: "·"}}}
  - {id: tras-cerrada, nombre: TrasCerrada, estado: abierto, depende_de: [cerrada], celdas: {Código: {estado: "·"}}}
  - {id: cerrada, nombre: Cerrada, estado: cerrado, celdas: {Código: {estado: "✅"}}}
  - {id: mixta, nombre: Mixta, estado: abierto, celdas: {Código: {estado: "⏳", bloqueo: decidir, de_quien: usuario o qdd-26}}}
  - {id: sin-quien, nombre: SinQuien, estado: abierto, celdas: {Código: {estado: "⏳", bloqueo: algo}}}
`

// sfrc (MEJ-055): con etapas «software», lo que queda en Validación real sin empezar es un uso real, no trabajo de la IA.
const tableroUsoReal = `formato: 1
etapas: [Código, Pruebas, Producción, Validación real]
frentes:
  - {id: validar, nombre: Validar, estado: abierto, celdas: {Código: {estado: "✅"}, Pruebas: {estado: "✅"}, Producción: {estado: "✅"}, Validación real: {estado: "·"}}}
  - {id: fallo-real, nombre: FalloReal, estado: abierto, celdas: {Código: {estado: "✅"}, Pruebas: {estado: "✅"}, Producción: {estado: "✅"}, Validación real: {estado: "❌", falla: el correo no llegó}}}
  - {id: desplegar, nombre: Desplegar, estado: abierto, celdas: {Código: {estado: "✅"}, Pruebas: {estado: "✅"}, Producción: {estado: "·"}, Validación real: {estado: "·"}}}
`

func TestMovibles_SoloLasQueLaIAPuedeMover_FND216(t *testing.T) {
	e, err := Parse([]byte(tableroMovibles))
	if err != nil {
		t.Fatal(err)
	}
	movibles, esperas := e.Movibles("qdd")
	got := map[string]bool{}
	for _, f := range movibles {
		got[f.ID] = true
	}
	want := map[string]bool{"libre": true, "de-ia": true, "nosotros": true, "sesion": true, "falla": true, "lista": true, "tras-cerrada": true}
	for id := range want {
		if !got[id] {
			t.Errorf("%s la puede mover la IA", id)
		}
	}
	for id := range got {
		if !want[id] {
			t.Errorf("%s no la puede mover la IA", id)
		}
	}
	quienes := map[string]string{}
	for _, x := range esperas {
		quienes[x.Frente.ID] = x.DeQuien
	}
	for id, quien := range map[string]string{"persona": "persona", "tras-persona": "la persona", "otro-proyecto": "qdd-movile-13", "tercero": "INRPAC", "verifica": "la IA", "mixta": "usuario o qdd-26", "sin-quien": ""} {
		if q, ok := quienes[id]; !ok || q != quien {
			t.Errorf("%s espera a %q: %q (está: %v)", id, quien, q, ok)
		}
	}
	if _, ok := quienes["espera-a-otra"]; ok {
		t.Error("una tarea que espera a otra no es un ⏳ propio: la lista nombra la que la bloquea")
	}
}

func TestMovibles_LaValidacionRealSinEmpezarEsperaUnUsoReal_FND216(t *testing.T) {
	e, err := Parse([]byte(tableroUsoReal))
	if err != nil {
		t.Fatal(err)
	}
	movibles, esperas := e.Movibles("sfrc")
	if len(movibles) != 2 || movibles[0].ID != "fallo-real" || movibles[1].ID != "desplegar" {
		t.Fatalf("un fallo en el uso real y un despliegue los mueve la IA: %+v", movibles)
	}
	if len(esperas) != 1 || esperas[0].Frente.ID != "validar" || esperas[0].DeQuien != UsoReal {
		t.Fatalf("la Validación real sin empezar espera un uso real: %+v", esperas)
	}
}

func TestMovibles_DeLaIA(t *testing.T) {
	for quien, es := range map[string]bool{
		"ia": true, "IA": true, "la IA": true, "Claude": true, "agente": true, "nosotros": true, "qdd": true, "qdd-6e": true, "qdd-88": true,
		"persona": false, "la persona": false, "usuario": false, "dpailahueque": false, "uso real": false, "CI": false,
		"qdd-movile": false, "qdd-movile-13": false, "notificador": false, "": false, "usuario o qdd-26": false,
	} {
		if DeLaIA(quien, "qdd") != es {
			t.Errorf("%q: de la IA = %v", quien, !es)
		}
	}
}
