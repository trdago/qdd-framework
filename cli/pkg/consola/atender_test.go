package consola

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// Pedido del usuario (2026-10-31): al pulsar una etapa ⏳ (o ❌) del panel, la IA se pronuncia sobre ella: intenta
// completarla y, si falta algo, deja un pedido en el panel o pregunta. qdd ofrece una acción por celda con el mensaje
// exacto; el panel solo la pide por id y se lo envía a la IA.

const estadoParaAtender = `formato: 1
etapas: [Código, PR, Publicación]
frentes:
  - {id: barra, nombre: Barra ordenada, estado: abierto, celdas: {Código: {estado: "✅", evidencia: commit a1}, PR: {estado: "❌", falla: el CI falla en Windows}, Publicación: {estado: "⏳", bloqueo: 5 h de uso medido, de_quien: uso real}}}
  - {id: vieja, nombre: Vieja, estado: cerrado, cerrado_el: "2026-10-30", motivo_cierre: hecha, celdas: {Código: {estado: "✅", evidencia: c}, PR: {estado: "✅", evidencia: p}, Publicación: {estado: "✅", evidencia: v}}}
`

func TestAtender_UnaAccionPorEtapaQueEsperaOFalla(t *testing.T) {
	root := proyectoConCola(t)
	escribirArchivo(t, filepath.Join(root, estado.Ruta), estadoParaAtender)
	var ids []string
	for _, a := range Acciones(root, true) {
		if a.Tipo == TipoAtender {
			ids = append(ids, a.ID)
		}
	}
	if strings.Join(ids, " ") != "atender:barra:1 atender:barra:2" {
		t.Fatalf("solo las etapas ⏳ y ❌ de tareas abiertas, por índice de etapa: %v", ids)
	}
}

func TestAtender_ElMensajeDiceQueHacerYQuePedir(t *testing.T) {
	root := proyectoConCola(t)
	escribirArchivo(t, filepath.Join(root, estado.Ruta), estadoParaAtender)
	espera, err := BuscarAccion(Acciones(root, true), "atender:barra:2")
	if err != nil {
		t.Fatal(err)
	}
	if espera.Confirmar || espera.Etiqueta != "Atender «Publicación» de «Barra ordenada»" {
		t.Fatalf("no escribe nada por sí misma: no pide «s  Sí»; y dice qué atiende: %+v", espera)
	}
	for _, debe := range []string{"«Publicación»", "«Barra ordenada»", "5 h de uso medido", "uso real", "qdd_consola_pedir", "tarea: barra", "pregúntame", "qdd estado set"} {
		if !strings.Contains(espera.Detalle, debe) {
			t.Fatalf("el mensaje a la IA dice %q: %s", debe, espera.Detalle)
		}
	}
	falla, _ := BuscarAccion(Acciones(root, true), "atender:barra:1")
	if !strings.Contains(falla.Detalle, "❌") || !strings.Contains(falla.Detalle, "el CI falla en Windows") {
		t.Fatalf("una etapa con falla pide encontrar la causa: %s", falla.Detalle)
	}
}

func TestAtender_EjecutarlaImprimeElMensajeParaLaIA(t *testing.T) {
	root := proyectoConCola(t)
	escribirArchivo(t, filepath.Join(root, estado.Ruta), estadoParaAtender)
	a, _ := BuscarAccion(Acciones(root, true), "atender:barra:2")
	var salida bytes.Buffer
	if codigo := a.Ejecutar(root, "qdd", &salida); codigo != 0 || strings.TrimSpace(salida.String()) != a.Detalle {
		t.Fatalf("la acción entrega el mensaje tal cual, y nada más: %d %q", codigo, salida.String())
	}
	escribirArchivo(t, filepath.Join(root, estado.Ruta), strings.Replace(estadoParaAtender, `Publicación: {estado: "⏳", bloqueo: 5 h de uso medido, de_quien: uso real}`, `Publicación: {estado: "✅", evidencia: v2.8.0}`, 1))
	if _, err := BuscarAccion(Acciones(root, true), "atender:barra:2"); err == nil {
		t.Fatal("una etapa que ya no espera no se ofrece: qdd lo vuelve a calcular")
	}
}
