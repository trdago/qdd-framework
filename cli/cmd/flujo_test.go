package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const flujoDePrueba = `formato: 1
id: pago
nombre: Cómo se cobra
nodos:
  - {id: a, tipo: paso, nombre: Publica, tarea: rc, siguen: [b]}
  - {id: b, tipo: resultado, nombre: Fin}
`

func proyectoConFlujo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := estado.Init(root, []string{"Código"}, ahoraDePruebaFlujo()); err != nil {
		t.Fatal(err)
	}
	estadoDir = root
	t.Cleanup(func() { estadoDir, flujoJSON, flujoAncho = "", false, anchoFlujoPorDefecto })
	archivo := filepath.Join(root, "f.yaml")
	if err := os.WriteFile(archivo, []byte(flujoDePrueba), 0o644); err != nil {
		t.Fatal(err)
	}
	var salida bytes.Buffer
	flujoCrearCmd.SetOut(&salida)
	flujoCrearCmd.Run(flujoCrearCmd, []string{archivo})
	if !strings.Contains(salida.String(), "Flujo 'pago' guardado") {
		t.Fatalf("crear: %q", salida.String())
	}
	return root
}

func TestFlujo_CrearYVerConElAnchoQueSePide(t *testing.T) {
	proyectoConFlujo(t)
	var salida bytes.Buffer
	flujoVerCmd.SetOut(&salida)
	flujoJSON, flujoAncho = true, 40
	flujoVerCmd.Run(flujoVerCmd, []string{"pago"})
	var d flujoDibujado
	if err := json.Unmarshal(salida.Bytes(), &d); err != nil {
		t.Fatalf("%v: %q", err, salida.String())
	}
	dibujo := strings.Join(d.Lineas, "\n")
	if d.ID != "pago" || d.Nombre != "Cómo se cobra" || !strings.Contains(dibujo, "Publica") || !strings.Contains(dibujo, "( Fin )") {
		t.Fatalf("dibujo: %+v", d)
	}
	if !strings.Contains(dibujo, "tarea rc no está") {
		t.Fatalf("una tarea que no existe se dice:\n%s", dibujo)
	}
}

func TestFlujo_UnaTareaDelTableroMuestraSusSimbolos(t *testing.T) {
	root := proyectoConFlujo(t)
	if err := estado.Set(root, estado.Cambio{Frente: "rc", Nombre: "RC", Etapa: "Código", Estado: "ok", Evidencia: "abc", AunqueExista: true}, ahoraDePruebaFlujo()); err != nil {
		t.Fatal(err)
	}
	if got := estadoDeTarea(root)("rc"); got != "✅" {
		t.Fatalf("símbolos: %q", got)
	}
}

func TestFlujo_ListaDiceCuandoNoHay(t *testing.T) {
	estadoDir = t.TempDir()
	t.Cleanup(func() { estadoDir = "" })
	var salida bytes.Buffer
	flujoListaCmd.SetOut(&salida)
	flujoListaCmd.Run(flujoListaCmd, nil)
	if !strings.Contains(salida.String(), "no tiene flujos") {
		t.Fatalf("lista: %q", salida.String())
	}
}

func ahoraDePruebaFlujo() time.Time { return time.Date(2026, 11, 5, 21, 0, 0, 0, time.UTC) }

// correrFlujo corre un subcomando como lo llama el panel, con sus banderas.
func correrFlujo(t *testing.T, c *cobra.Command, banderas map[string]string, args ...string) string {
	t.Helper()
	var salida bytes.Buffer
	c.SetOut(&salida)
	for nombre, valor := range banderas {
		if err := c.Flags().Set(nombre, valor); err != nil {
			t.Fatal(err)
		}
	}
	c.Run(c, args)
	c.Flags().VisitAll(func(b *pflag.Flag) {
		_ = b.Value.Set(b.DefValue)
		b.Changed = false
	})
	return salida.String()
}

func verFlujo(t *testing.T, id string, banderas map[string]string) flujoDibujado {
	t.Helper()
	todas := map[string]string{"json": "true"}
	for k, v := range banderas {
		todas[k] = v
	}
	var d flujoDibujado
	if err := json.Unmarshal([]byte(correrFlujo(t, flujoVerCmd, todas, id)), &d); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestFlujo_LaPersonaEditaYLaIARecibeUnaVez(t *testing.T) {
	proyectoConFlujo(t)
	correrFlujo(t, flujoAgregarCmd, map[string]string{"nombre": "Cobra", "de": "persona"}, "pago", "a")
	correrFlujo(t, flujoModificarCmd, map[string]string{"descripcion": "con tarjeta", "de": "persona"}, "pago", "cobra")
	d := verFlujo(t, "pago", map[string]string{"elegido": "cobra"})
	if len(d.Nodos) != 3 || d.Nodos[1].ID != "cobra" || d.Nodos[1].Descripcion != "con tarjeta" {
		t.Fatalf("nodos: %+v", d.Nodos)
	}
	if len(d.PorEnviar) != 2 || !d.PuedeDeshacer {
		t.Fatalf("por enviar: %+v", d)
	}
	if !strings.Contains(strings.Join(d.Lineas, "\n"), "┏") {
		t.Fatalf("no resalta el elegido:\n%s", strings.Join(d.Lineas, "\n"))
	}
	var envio flujoEnvio
	if err := json.Unmarshal([]byte(correrFlujo(t, flujoEnviarCmd, map[string]string{"json": "true"}, "pago")), &envio); err != nil {
		t.Fatal(err)
	}
	if envio.N != 2 || !strings.Contains(envio.Mensaje, "+ cobra") {
		t.Fatalf("envío: %+v", envio)
	}
	correrFlujo(t, flujoDevolverCmd, map[string]string{"n": "2"}, "pago")
	if d := verFlujo(t, "pago", nil); len(d.PorEnviar) != 2 {
		t.Fatalf("devolver: %+v", d.PorEnviar)
	}
	correrFlujo(t, flujoDeshacerCmd, nil, "pago")
	if d := verFlujo(t, "pago", nil); d.Nodos[1].Descripcion != "" || len(d.PorEnviar) != 1 {
		t.Fatalf("deshacer: %+v", d)
	}
}

func TestFlujo_LaIARespondeEnElMismoNodoSinLlamarseASiMisma(t *testing.T) {
	proyectoConFlujo(t)
	correrFlujo(t, flujoComentarCmd, map[string]string{"de": "persona"}, "pago", "a", "¿por qué publica primero?")
	correrFlujo(t, flujoPedirDetalleCmd, nil, "pago", "b")
	var envio flujoEnvio
	if err := json.Unmarshal([]byte(correrFlujo(t, flujoEnviarCmd, map[string]string{"json": "true"}, "pago")), &envio); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(envio.Mensaje, "💬 a") || !strings.Contains(envio.Mensaje, "🔎 b") || !strings.Contains(envio.Mensaje, "qdd flujo comentar pago a") {
		t.Fatalf("mensaje: %q", envio.Mensaje)
	}
	correrFlujo(t, flujoComentarCmd, nil, "pago", "a", "porque el tag va antes")
	correrFlujo(t, flujoDetalleCmd, nil, "pago", "b", "cierra el pago")
	d := verFlujo(t, "pago", nil)
	if len(d.PorEnviar) != 0 {
		t.Fatalf("la IA se llamó a sí misma: %v", d.PorEnviar)
	}
	c := d.Nodos[0].Comentarios
	if len(c) != 2 || c[0].De != "persona" || c[1].De != "ia" || d.Nodos[1].Descripcion != "cierra el pago" {
		t.Fatalf("nodos: %+v", d.Nodos)
	}
}

func TestFlujo_CrearNoPierdeLosComentariosYLoDice(t *testing.T) {
	root := proyectoConFlujo(t)
	correrFlujo(t, flujoComentarCmd, map[string]string{"de": "persona"}, "pago", "b", "este fin no me convence")
	archivo := filepath.Join(root, "g.yaml")
	if err := os.WriteFile(archivo, []byte("formato: 1\nid: pago\nnombre: Otro\nnodos:\n  - {id: a, tipo: paso, nombre: Solo}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	salida := correrFlujo(t, flujoCrearCmd, nil, archivo)
	if !strings.Contains(salida, "1 comentario") {
		t.Fatalf("salida: %q", salida)
	}
	if d := verFlujo(t, "pago", nil); len(d.Nodos[0].Comentarios) != 1 {
		t.Fatalf("nodos: %+v", d.Nodos)
	}
}

func TestFlujo_NuevoDaElIDParaAbrirlo(t *testing.T) {
	proyectoConFlujo(t)
	var f struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(correrFlujo(t, flujoNuevoCmd, map[string]string{"json": "true"}, "Cómo se paga")), &f); err != nil {
		t.Fatal(err)
	}
	if f.ID != "como-se-paga" {
		t.Fatalf("id: %q", f.ID)
	}
}

// Pedido de la persona (2026-11-07): preguntarle a un nodo o al flujo completo, y que la IA responda ahí mismo.
func TestFlujo_PreguntarAUnNodoYAlFlujoYLaIARespondeAhi(t *testing.T) {
	proyectoConFlujo(t)
	correrFlujo(t, flujoPreguntarCmd, nil, "pago", "a", "¿Qué significa publicar aquí?")
	correrFlujo(t, flujoPreguntarCmd, nil, "pago", "¿Para qué sirve este flujo?")
	var envio flujoEnvio
	if err := json.Unmarshal([]byte(correrFlujo(t, flujoEnviarCmd, map[string]string{"json": "true"}, "pago")), &envio); err != nil {
		t.Fatal(err)
	}
	if envio.N != 2 || !strings.Contains(envio.Mensaje, "qdd flujo responder pago a") || !strings.Contains(envio.Mensaje, "qdd flujo responder pago \"") {
		t.Fatalf("las dos preguntas llegan con cómo responder: %+v", envio)
	}
	correrFlujo(t, flujoResponderCmd, nil, "pago", "a", "Subir la versión a npm.")
	correrFlujo(t, flujoResponderCmd, nil, "pago", "Publica una versión.")
	d := verFlujo(t, "pago", nil)
	if len(d.PorEnviar) != 0 {
		t.Fatalf("la IA no se llama a sí misma: %v", d.PorEnviar)
	}
	if c := d.Nodos[0].Comentarios; len(c) != 2 || !c[0].Pregunta || c[1].De != "ia" {
		t.Fatalf("pregunta y respuesta bajo el nodo: %+v", c)
	}
	if len(d.Comentarios) != 2 || !d.Comentarios[0].Pregunta || d.Comentarios[1].De != "ia" {
		t.Fatalf("y en el flujo: %+v", d.Comentarios)
	}
}

// Ítems de un nodo (pedido de la persona, 2026-11-07): pruebas que qdd-human marca, notas, características y funciones.
func TestFlujo_ItemsDeUnNodo(t *testing.T) {
	proyectoConFlujo(t)
	correrFlujo(t, flujoItemAgregarCmd, map[string]string{"de": "persona"}, "pago", "a", "prueba", "publica con el tag correcto")
	correrFlujo(t, flujoItemAgregarCmd, map[string]string{"de": "persona"}, "pago", "a", "nota", "corre en CI")
	var envio flujoEnvio
	if err := json.Unmarshal([]byte(correrFlujo(t, flujoEnviarCmd, map[string]string{"json": "true"}, "pago")), &envio); err != nil {
		t.Fatal(err)
	}
	if envio.N != 2 || !strings.Contains(envio.Mensaje, "qdd flujo item marcar pago a i1") {
		t.Fatalf("la IA ve los ítems nuevos y cómo marcar la prueba: %+v", envio)
	}
	correrFlujo(t, flujoItemMarcarCmd, map[string]string{"evidencia": ".qdd/evidence/run_1"}, "pago", "a", "i1", "pasa")
	correrFlujo(t, flujoItemQuitarCmd, nil, "pago", "a", "i2")
	d := verFlujo(t, "pago", nil)
	items := d.Nodos[0].Items
	if len(items) != 1 || items[0].Estado != "pasa" || items[0].Evidencia != ".qdd/evidence/run_1" {
		t.Fatalf("ítems: %+v", items)
	}
	if len(d.PorEnviar) != 0 {
		t.Fatalf("lo que hace la IA no la llama: %v", d.PorEnviar)
	}
}
