package consola

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'

var enCola = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

// proyectoConCola es un proyecto QDD con su propia carpeta central (QDD_HOME).
func proyectoConCola(t *testing.T) string {
	t.Helper()
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".qdd"), 0755); err != nil {
		t.Fatal(err)
	}
	return root
}

func mustPedir(t *testing.T, root, comando, motivo, carpeta string) Pedido {
	t.Helper()
	p, err := Pedir(root, comando, motivo, carpeta, enCola)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPedir_QuedaEnLaColaFueraDelProyecto_CCC_C17(t *testing.T) {
	root := proyectoConCola(t)
	p := mustPedir(t, root, "git push origin v2.5.1", "Publicar v2.5.1: el push del tag es tuyo", "")
	if p.ID == "" || p.Carpeta != root || !p.Creado.Equal(enCola) {
		t.Fatalf("el pedido lleva id, la carpeta del proyecto por defecto y su hora: %+v", p)
	}
	if got := Pedidos(root); len(got) != 1 || got[0] != p {
		t.Fatalf("el pedido queda en la cola del proyecto: %+v", got)
	}
	entradas, _ := os.ReadDir(filepath.Join(root, ".qdd"))
	if len(entradas) != 0 {
		t.Fatalf("la cola no se escribe en el proyecto (no se versiona): %v", entradas)
	}
	if otro := Pedidos(t.TempDir()); len(otro) != 0 {
		t.Fatalf("cada proyecto tiene su cola: %+v", otro)
	}
}

func TestPedir_ValidaYNoDuplica(t *testing.T) {
	root := proyectoConCola(t)
	if _, err := Pedir(root, "  ", "motivo", "", enCola); err == nil {
		t.Fatal("un pedido sin comando se rechaza")
	}
	if _, err := Pedir(root, "git push", " ", "", enCola); err == nil || !strings.Contains(err.Error(), "motivo") {
		t.Fatalf("un pedido sin motivo se rechaza: %v", err)
	}
	if _, err := Pedir(root, "git push", "m", filepath.Join(root, "no-existe"), enCola); err == nil {
		t.Fatal("una carpeta que no existe se rechaza")
	}
	a := mustPedir(t, root, "git push", "uno", "")
	b := mustPedir(t, root, "git push", "otra vez", "")
	if a.ID != b.ID || len(Pedidos(root)) != 1 {
		t.Fatalf("el mismo comando en la misma carpeta no se duplica: %+v %+v", a, b)
	}
}

func TestPedir_TopeDeNuevePedidos(t *testing.T) {
	root := proyectoConCola(t)
	for i := 0; i < MaxPedidos; i++ {
		mustPedir(t, root, "echo "+string(rune('a'+i)), "m", "")
	}
	if _, err := Pedir(root, "echo otro", "m", "", enCola); err == nil || !strings.Contains(err.Error(), "9") {
		t.Fatalf("con la cola llena se rechaza y se dice el tope: %v", err)
	}
}

func TestAcciones_OfreceLosPedidosConSuComandoYCarpeta_CCC_C17(t *testing.T) {
	root := proyectoConCola(t)
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	p := mustPedir(t, root, "git push origin v2.5.1", "Publicar v2.5.1", sub)
	a, err := BuscarAccion(Acciones(root, false), "pedido:"+p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if a.Tipo != TipoPedido || !a.Confirmar || a.Etiqueta != "Publicar v2.5.1" || a.Detalle != "git push origin v2.5.1" || a.Carpeta != sub || a.Tecla != "a" {
		t.Fatalf("el botón muestra el motivo, el comando exacto y la carpeta, y pide confirmación: %+v", a)
	}
	if err := Autorizar(a, false); err == nil {
		t.Fatal("un pedido nunca corre sin --confirmado")
	}
}

func comandoQueEscribe(archivo string) string {
	if runtime.GOOS == "windows" {
		return "echo hecho> " + archivo
	}
	return "echo hecho > " + archivo
}

func TestEjecutar_UnPedidoCorreEnSuCarpetaYSaleDeLaCola_CCC_C18(t *testing.T) {
	root := proyectoConCola(t)
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	p := mustPedir(t, root, comandoQueEscribe("marca.txt"), "Probar", sub)
	a, _ := BuscarAccion(Acciones(root, false), "pedido:"+p.ID)
	var salida bytes.Buffer
	if codigo := a.Ejecutar(root, "qdd", &salida); codigo != 0 {
		t.Fatalf("código %d: %s", codigo, salida.String())
	}
	if _, err := os.Stat(filepath.Join(sub, "marca.txt")); err != nil {
		t.Fatal("el comando corre en la carpeta del pedido")
	}
	if len(Pedidos(root)) != 0 {
		t.Fatal("un pedido que terminó bien sale de la cola")
	}
}

func TestEjecutar_UnPedidoQueFallaSigueEnLaCola_CCC_C18(t *testing.T) {
	root := proyectoConCola(t)
	p := mustPedir(t, root, "exit 3", "Falla", "")
	a, _ := BuscarAccion(Acciones(root, false), "pedido:"+p.ID)
	var salida bytes.Buffer
	if codigo := a.Ejecutar(root, "qdd", &salida); codigo != 3 {
		t.Fatalf("se devuelve el código del comando: %d", codigo)
	}
	if len(Pedidos(root)) != 1 {
		t.Fatal("un pedido que falló queda para reintentarlo o descartarlo")
	}
}

func TestDescartar_QuitaElPedidoSinEjecutarlo_CCC_C19(t *testing.T) {
	root := proyectoConCola(t)
	p := mustPedir(t, root, comandoQueEscribe("no.txt"), "No lo quiero", "")
	if err := Descartar(root, p.ID); err != nil {
		t.Fatal(err)
	}
	if len(Pedidos(root)) != 0 {
		t.Fatal("el pedido descartado sale de la cola")
	}
	if _, err := os.Stat(filepath.Join(root, "no.txt")); err == nil {
		t.Fatal("descartar no ejecuta nada")
	}
	if err := Descartar(root, p.ID); err == nil {
		t.Fatal("descartar un id que no está es un error")
	}
}
