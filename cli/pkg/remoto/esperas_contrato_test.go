package remoto

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/qdd-framework/qdd/pkg/consola"
)

// Rule: Prohibido usar 'else'
// El contrato de lo que espera a la persona lo comparten la app, el relay y qdd (testdata/esperas.json, copiado de
// qdd-movile). FND-208: las preguntas y permisos no pedían push. FND-210: un pedido sin estado (cola de qdd 2.6.0), en
// espera de otro o que falló y se puede reintentar tampoco, aunque la app los cuenta en «Te espera».

type casoEspera struct {
	Caso       string         `json:"caso"`
	Accion     consola.Accion `json:"accion"`
	SinConsola bool           `json:"sin_consola"`
	Espera     bool           `json:"espera"`
	Aviso      *string        `json:"aviso"`
}

func TestEsperan_CumpleElContratoDeQDDMovil_FND208(t *testing.T) {
	b, err := os.ReadFile("testdata/esperas.json")
	if err != nil {
		t.Fatal(err)
	}
	var contrato struct {
		Casos []casoEspera `json:"casos"`
	}
	if err := json.Unmarshal(b, &contrato); err != nil {
		t.Fatal(err)
	}
	if len(contrato.Casos) < 23 {
		t.Fatalf("el contrato trae %d casos", len(contrato.Casos))
	}
	for _, c := range contrato.Casos {
		panel := &consola.DatosConsola{Acciones: []consola.Accion{c.Accion}}
		if c.SinConsola {
			panel.Ejecutar = &consola.OpcionesEjecutar{QDD: true}
		}
		aviso, avisa := esperan(ProyectoEstado{Clave: "p", Panel: panel})[c.Accion.ID]
		quiere := ""
		if c.Aviso != nil {
			quiere = *c.Aviso
		}
		if avisa != c.Espera || aviso != quiere {
			t.Errorf("%s: avisa=%v %q, el contrato dice %v %q", c.Caso, avisa, aviso, c.Espera, quiere)
		}
	}
}
