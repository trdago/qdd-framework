package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/flujo"
)

// Rule: Prohibido usar 'else'
// RFC 0025: las historias de usuario se escriben, se listan y se vuelven tarea del tablero desde la CLI, y la persona
// decide si una nota técnica la ve el cliente.

func limpiarFlagsDeHistoria(t *testing.T) {
	t.Cleanup(func() {
		historiaComo, historiaQuiero, historiaPara, historiaCriterios, historiaNodos = "", "", "", nil, nil
		itemInterna, flujoDe = false, flujo.DeIA
	})
}

func TestFlujoHistoria_SeEscribeSeListaYSeVuelveTarea(t *testing.T) {
	root := proyectoConFlujo(t)
	limpiarFlagsDeHistoria(t)
	historiaComo, historiaQuiero, historiaPara = "vendedor", "validar el RUT antes de crear el cliente", "no duplicar clientes"
	historiaCriterios, historiaNodos, flujoDe = []string{"un RUT inválido no deja seguir"}, []string{"a"}, flujo.DePersona
	var salida bytes.Buffer
	flujoHistoriaCmd.SetOut(&salida)
	flujoHistoriaCmd.Run(flujoHistoriaCmd, []string{"pago"})
	if !strings.Contains(salida.String(), "h1") {
		t.Fatalf("escribe la historia: %q", salida.String())
	}
	salida.Reset()
	flujoHistoriaTareaCmd.SetOut(&salida)
	flujoHistoriaTareaCmd.Run(flujoHistoriaTareaCmd, []string{"pago", "h1"})
	f, err := flujo.Cargar(root, "pago")
	if err != nil || f.Historias[0].Tarea == "" {
		t.Fatalf("la historia recuerda su tarea: %+v %v", f.Historias, err)
	}
	e, err := estado.Load(root)
	if err != nil || e.ValidarTarea(f.Historias[0].Tarea) != nil {
		t.Fatalf("la tarea está en el tablero: %v", err)
	}
	if !strings.Contains(salida.String(), "un RUT inválido no deja seguir") {
		t.Fatalf("la IA lee los criterios al crearla: %q", salida.String())
	}
	salida.Reset()
	flujoHistoriasCmd.SetOut(&salida)
	flujoJSON = true
	flujoHistoriasCmd.Run(flujoHistoriasCmd, []string{"pago"})
	var lista []flujo.Historia
	if err := json.Unmarshal(salida.Bytes(), &lista); err != nil || len(lista) != 1 || lista[0].Origen != flujo.DePersona {
		t.Fatalf("lista con su origen: %v %q", err, salida.String())
	}
}

func TestFlujoItem_CompartirConElCliente(t *testing.T) {
	root := proyectoConFlujo(t)
	limpiarFlagsDeHistoria(t)
	if _, _, err := flujo.Editar(root, "pago", flujo.DeIA, ahoraDePruebaFlujo(), flujo.AgregarItem("a", flujo.ItemNota, "usar la API del SII", ahoraDePruebaFlujo())); err != nil {
		t.Fatal(err)
	}
	var salida bytes.Buffer
	flujoItemCompartirCmd.SetOut(&salida)
	flujoItemCompartirCmd.Run(flujoItemCompartirCmd, []string{"pago", "a", "i1"})
	f, _ := flujo.Cargar(root, "pago")
	if f.Nodos[0].Items[0].Visible != flujo.VisibleCompartida {
		t.Fatalf("compartida: %+v", f.Nodos[0].Items)
	}
	itemInterna = true
	flujoItemCompartirCmd.Run(flujoItemCompartirCmd, []string{"pago", "a", "i1"})
	f, _ = flujo.Cargar(root, "pago")
	if f.Nodos[0].Items[0].Visible != "" {
		t.Fatalf("vuelve a interna: %+v", f.Nodos[0].Items)
	}
}
