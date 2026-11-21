package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/qdd-framework/qdd/pkg/consola"
)

// Rule: Prohibido usar 'else'
// MEJ-024 (issue #148): con el modo goal activo, un pedido de la IA corre sin el «s  Sí» de la persona. Un push que solo la
// persona puede autorizar no tenía cómo quedar en el panel esperándola. Ahora la IA lo marca de_persona (--de-persona en
// la CLI) y el modo goal no lo lanza (TestPedidoDePersona_ElModoGoalNoLoLanza).
// MEJ-026 (issue #150): qdd consola fin-de-turno dice qué le falta dejar en el panel a la IA al terminar un turno.

func TestConsolaPedirTool_DePersonaEsperaALaPersonaAunqueGoal_MEJ024(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".qdd"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	root, _ = os.Getwd()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"comando": "git push origin main", "motivo": "Subir main: la persona autoriza cada push", "de_persona": true}
	res, err := handleConsolaPedirTool(context.Background(), req)
	if err != nil || res.IsError {
		t.Fatalf("%v %+v", err, res)
	}
	if cola := consola.Pedidos(root); len(cola) != 1 || !cola[0].DePersona {
		t.Fatalf("queda marcado de la persona: %+v", cola)
	}
	if texto := res.Content[0].(mcp.TextContent).Text; !strings.Contains(texto, "modo goal") {
		t.Fatalf("la respuesta dice que espera a la persona aunque el modo goal esté activo: %q", texto)
	}
}

func TestConsolaPedir_ConDePersonaLoMarca_MEJ024(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	if consolaPedirCmd.Flags().Lookup("de-persona") == nil {
		t.Fatal("qdd consola pedir acepta --de-persona")
	}
	var salida, errores bytes.Buffer
	if codigo := consolaPedir(root, "Subir main", "", consola.Orden{DePersona: true}, []string{"git", "push"}, &salida, &errores); codigo != 0 {
		t.Fatalf("código %d: %s", codigo, errores.String())
	}
	if cola := consola.Pedidos(root); len(cola) != 1 || !cola[0].DePersona {
		t.Fatalf("queda marcado de la persona: %+v", cola)
	}
	if !strings.Contains(salida.String(), "modo goal") {
		t.Fatalf("la salida lo dice: %q", salida.String())
	}
}

func TestConsolaFinDeTurno_DevuelveLoQueFalta_MEJ026(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".qdd"), 0755); err != nil {
		t.Fatal(err)
	}
	entrada, _ := json.Marshal(consola.TurnoTerminado{Mensaje: "¿Parto con esa?"})
	var salida bytes.Buffer
	if codigo := finDeTurno(root, bytes.NewReader(entrada), &salida); codigo != 0 {
		t.Fatalf("código %d", codigo)
	}
	var r struct {
		Devolver string `json:"devolver"`
	}
	if err := json.Unmarshal(salida.Bytes(), &r); err != nil || !strings.Contains(r.Devolver, "¿Parto con esa?") {
		t.Fatalf("devuelve el turno con lo que falta: %q %v", salida.String(), err)
	}
}

func TestConsolaFinDeTurno_FueraDeUnProyectoQddNoDevuelve_MEJ026(t *testing.T) {
	entrada, _ := json.Marshal(consola.TurnoTerminado{Mensaje: "¿Parto con esa?"})
	var salida bytes.Buffer
	if codigo := finDeTurno(t.TempDir(), bytes.NewReader(entrada), &salida); codigo != 0 || strings.Contains(salida.String(), "devolver\":\"¿") {
		t.Fatalf("sin .qdd no hay panel donde dejarlo: %d %q", codigo, salida.String())
	}
	if !strings.Contains(salida.String(), `"devolver":""`) {
		t.Fatalf("igual responde JSON: %q", salida.String())
	}
	if codigo := finDeTurno(t.TempDir(), strings.NewReader("no es json"), &salida); codigo == 0 {
		t.Fatal("una entrada rota es un error del panel")
	}
}
