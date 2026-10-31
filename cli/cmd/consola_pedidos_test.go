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
	if codigo := consolaPedir(root, "Publicar v2.5.1", "", consola.Orden{}, []string{"git", "push", "origin", "v2.5.1"}, &salida, &errores); codigo != 0 {
		t.Fatalf("código %d: %s", codigo, errores.String())
	}
	cola := consola.Pedidos(root)
	if len(cola) != 1 || cola[0].Comando != "git push origin v2.5.1" {
		t.Fatalf("los argumentos forman el comando: %+v", cola)
	}
	if !strings.Contains(salida.String(), "panel") || !strings.Contains(salida.String(), "tecla a") {
		t.Fatalf("la salida dice que quedó en el panel y con qué tecla: %q", salida.String())
	}
	if codigo := consolaPedir(root, "", "", consola.Orden{}, []string{"ls"}, &salida, &errores); codigo != 1 || !strings.Contains(errores.String(), "motivo") {
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
	if codigo := consolaAccion(root, "pedido:"+p.ID, true, &salida, &errores); codigo != 0 || esperarFin(t, root, p.ID).Estado != consola.EstadoHecho {
		t.Fatalf("confirmado corre (aparte, FND-089) y queda hecho: %d %q", codigo, salida.String())
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

func TestConsolaPedidos_ListaCadaPedidoConSuEstado(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	hecho, _ := consola.Pedir(root, "exit 0", "Ya hecho", "", ahoraPrueba)
	consola.Pedir(root, "exit 0; echo", "Por hacer", "", ahoraPrueba)
	var salida, errores bytes.Buffer
	consolaAccion(root, "pedido:"+hecho.ID, true, &salida, &errores)
	esperarFin(t, root, hecho.ID)
	salida.Reset()
	listarPedidos(root, &salida)
	texto := salida.String()
	if !strings.Contains(texto, "[✓] Ya hecho · hecho") || !strings.Contains(texto, "[b] Por hacer · pendiente") {
		t.Fatalf("cada pedido dice su tecla (o ✓) y su estado: %q", texto)
	}
	salida.Reset()
	listarPedidos(t.TempDir(), &salida)
	if !strings.Contains(salida.String(), "No hay comandos") {
		t.Fatalf("sin pedidos lo dice: %q", salida.String())
	}
}

// FND-080: el orden se da al pedir (--despues/--antes, o despues/antes en la tool) y se ve en la lista.
func TestConsolaPedir_ConOrdenDiceQueEsperaYLaListaLoMuestra_FND080(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	var salida, errores bytes.Buffer
	consolaPedir(root, "Tag v2.7.0", "", consola.Orden{}, []string{"exit", "0"}, &salida, &errores)
	salida.Reset()
	if codigo := consolaPedir(root, "Merge #14", "", consola.Orden{Despues: []string{"a"}}, []string{"exit", "0;", "echo"}, &salida, &errores); codigo != 0 {
		t.Fatalf("código %d: %s", codigo, errores.String())
	}
	if !strings.Contains(salida.String(), "tecla b") || !strings.Contains(salida.String(), "espera a [a]") {
		t.Fatalf("la salida dice su tecla y qué espera: %q", salida.String())
	}
	salida.Reset()
	listarPedidos(root, &salida)
	if !strings.Contains(salida.String(), "[b] Merge #14 · espera a [a]") {
		t.Fatalf("la lista dice qué espera: %q", salida.String())
	}
	if codigo := consolaPedir(root, "Otro", "", consola.Orden{Despues: []string{"z"}}, []string{"ls"}, &salida, &errores); codigo != 1 || !strings.Contains(errores.String(), "'z'") {
		t.Fatalf("una referencia que no está se rechaza: %d %q", codigo, errores.String())
	}
}

func TestConsolaPedirTool_AceptaDespuesYAntes_FND080(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	t.Chdir(root)
	root, _ = os.Getwd()
	consola.Pedir(root, "exit 0", "Merge #14", "", ahoraPrueba)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"comando": "exit 0; echo", "motivo": "Tag v2.6.1", "antes": "a"}
	res, err := handleConsolaPedirTool(context.Background(), req)
	if err != nil || res.IsError {
		t.Fatalf("%v %+v", err, res)
	}
	cola := consola.Pedidos(root)
	if len(cola) != 2 || cola[0].Motivo != "Tag v2.6.1" {
		t.Fatalf("lo que va antes queda primero en la cola: %+v", cola)
	}
	req.Params.Arguments = map[string]any{"comando": "ls", "motivo": "m", "despues": "a, z"}
	if res, _ := handleConsolaPedirTool(context.Background(), req); !res.IsError {
		t.Fatal("una referencia que no está se rechaza")
	}
}

// FND-081: si el comando corre en otro proyecto, la salida dice en qué panel quedó.
func TestConsolaPedir_DiceEnQuePanelQuedo_FND081(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root, otro := t.TempDir(), t.TempDir()
	os.MkdirAll(filepath.Join(root, ".qdd"), 0755)
	os.MkdirAll(filepath.Join(otro, ".qdd"), 0755)
	var salida, errores bytes.Buffer
	if codigo := consolaPedir(root, "Mergear", otro, consola.Orden{}, []string{"exit", "0"}, &salida, &errores); codigo != 0 {
		t.Fatalf("código %d: %s", codigo, errores.String())
	}
	if !strings.Contains(salida.String(), "tecla a") || !strings.Contains(salida.String(), "panel de "+filepath.Base(otro)) {
		t.Fatalf("la salida dice su tecla y el panel del otro proyecto: %q", salida.String())
	}
}

// esperarFin espera a que un pedido lanzado aparte (FND-089) termine: hecho o con falla.
func esperarFin(t *testing.T, root, id string) consola.Pedido {
	t.Helper()
	tope := time.Now().Add(10 * time.Second)
	for time.Now().Before(tope) {
		for _, p := range consola.Pedidos(root) {
			if p.ID == id && (p.Estado == consola.EstadoHecho || p.Estado == consola.EstadoFallo) {
				return p
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("el pedido %s no terminó: %+v", id, consola.Pedidos(root))
	return consola.Pedido{}
}
