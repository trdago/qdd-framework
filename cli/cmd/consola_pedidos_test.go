package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/qdd-framework/qdd/pkg/consola"
)

// Rule: Prohibido usar 'else'

var ahoraPrueba = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

func TestConsolaPedir_DejaElComandoEnElPanel_CCC_C17(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	var salida, errores bytes.Buffer
	if codigo := consolaPedir(root, "Publicar v2.5.1", "", []string{"git", "push", "origin", "v2.5.1"}, &salida, &errores); codigo != 0 {
		t.Fatalf("código %d: %s", codigo, errores.String())
	}
	cola := consola.Pedidos(root)
	if len(cola) != 1 || cola[0].Comando != "git push origin v2.5.1" {
		t.Fatalf("los argumentos forman el comando: %+v", cola)
	}
	if !strings.Contains(salida.String(), "panel") || !strings.Contains(salida.String(), "tecla a") {
		t.Fatalf("la salida dice que quedó en el panel y con qué tecla: %q", salida.String())
	}
	if codigo := consolaPedir(root, "", "", []string{"ls"}, &salida, &errores); codigo != 1 || !strings.Contains(errores.String(), "motivo") {
		t.Fatalf("sin motivo se rechaza: %d %q", codigo, errores.String())
	}
}

func TestConsolaAccion_UnPedidoSinConfirmarNoCorre_CCC_C18(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	p, err := consola.Pedir(root, "exit 0", "m", "", ahoraPrueba)
	if err != nil {
		t.Fatal(err)
	}
	var salida, errores bytes.Buffer
	if codigo := consolaAccion(root, "pedido:"+p.ID, false, &salida, &errores); codigo != 1 || !strings.Contains(errores.String(), "--confirmado") {
		t.Fatalf("un pedido no corre sin --confirmado: %d %q", codigo, errores.String())
	}
	if len(consola.Pedidos(root)) != 1 {
		t.Fatal("el pedido sigue en la cola")
	}
	if codigo := consolaAccion(root, "pedido:"+p.ID, true, &salida, &errores); codigo != 0 || consola.Pedidos(root)[0].Estado != consola.EstadoHecho {
		t.Fatalf("confirmado corre y queda hecho: %d %q", codigo, salida.String())
	}
	salida.Reset()
	errores.Reset()
	if codigo := consolaAccion(root, "pedido:"+p.ID, true, &salida, &errores); codigo == 0 {
		t.Fatalf("FND-070: un pedido hecho no se acepta dos veces: %q %q", salida.String(), errores.String())
	}
}

func TestConsolaPedirTool_NuncaEjecutaYDiceQueEsperaALaPersona_CCC_C17(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".qdd"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	root, _ = os.Getwd()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"comando": "echo hola > marca.txt", "motivo": "Probar"}
	res, err := handleConsolaPedirTool(context.Background(), req)
	if err != nil || res.IsError {
		t.Fatalf("%v %+v", err, res)
	}
	texto := res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(texto, "persona") || !strings.Contains(texto, "No lo ejecutes") {
		t.Fatalf("la respuesta le dice a la IA que espere a la persona: %q", texto)
	}
	if cola := consola.Pedidos(root); len(cola) != 1 || cola[0].Carpeta != root {
		t.Fatalf("queda en la cola del proyecto, sin ejecutarse: %+v", cola)
	}
}
