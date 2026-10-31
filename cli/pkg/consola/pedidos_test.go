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
	if got := Pedidos(root); len(got) != 1 || got[0].ID != p.ID || got[0].Estado != EstadoPendiente || got[0].Tecla != "a" {
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

func TestEjecutar_UnPedidoCorreEnSuCarpetaYQuedaHecho_CCC_C18(t *testing.T) {
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
	cola := Pedidos(root)
	if len(cola) != 1 || cola[0].Estado != EstadoHecho || cola[0].Fin == nil || cola[0].Abierto() {
		t.Fatalf("un pedido que terminó bien queda a la vista como hecho, ya cerrado: %+v", cola)
	}
	hecho, _ := BuscarAccion(Acciones(root, false), "pedido:"+p.ID)
	if hecho.Estado != EstadoHecho || hecho.Tecla != "" {
		t.Fatalf("el panel lo muestra hecho y sin tecla, para no clickearlo de nuevo: %+v", hecho)
	}
}

// FND-070: un pedido aceptado se podía volver a ejecutar desde el panel (o con dos clics seguidos).
func TestEjecutar_UnPedidoAceptadoNoCorreDosVeces_FND070(t *testing.T) {
	root := proyectoConCola(t)
	marca := filepath.Join(root, "veces.txt")
	comando := "echo x >> " + marca
	p := mustPedir(t, root, comando, "Una sola vez", "")
	a, _ := BuscarAccion(Acciones(root, false), "pedido:"+p.ID)
	var salida bytes.Buffer
	if codigo := a.Ejecutar(root, "qdd", &salida); codigo != 0 {
		t.Fatalf("código %d: %s", codigo, salida.String())
	}
	salida.Reset()
	if codigo := a.Ejecutar(root, "qdd", &salida); codigo != 1 || !strings.Contains(salida.String(), "ya se ejecutó") {
		t.Fatalf("la segunda vez se rechaza y se dice por qué: %d %q", codigo, salida.String())
	}
	if b, _ := os.ReadFile(marca); strings.Count(string(b), "x") != 1 {
		t.Fatalf("el comando corrió una sola vez: %q", b)
	}
	if again := mustPedir(t, root, comando, "otra", ""); again.ID == p.ID || again.Estado != EstadoPendiente {
		t.Fatalf("pedirlo de nuevo, ya hecho, es un pedido nuevo: %+v", again)
	}
}

func TestEjecutar_UnPedidoEjecutandoseNoSeAceptaNiSeDescarta_FND070(t *testing.T) {
	root := proyectoConCola(t)
	p := mustPedir(t, root, "exit 0", "m", "")
	inicio := time.Now()
	if err := conCola(root, inicio, func(cola []Pedido) ([]Pedido, error) {
		cola[0].Estado, cola[0].Inicio = EstadoEjecutando, &inicio
		return cola, nil
	}); err != nil {
		t.Fatal(err)
	}
	a, _ := BuscarAccion(Acciones(root, false), "pedido:"+p.ID)
	if a.Estado != EstadoEjecutando || a.Tecla != "" {
		t.Fatalf("mientras corre el panel no ofrece tecla: %+v", a)
	}
	var salida bytes.Buffer
	if codigo := a.Ejecutar(root, "qdd", &salida); codigo != 1 || !strings.Contains(salida.String(), "ya se está ejecutando") {
		t.Fatalf("no se acepta mientras corre: %d %q", codigo, salida.String())
	}
	if err := Descartar(root, p.ID); err == nil {
		t.Fatal("no se descarta mientras corre")
	}
}

func TestPedidos_UnoQueQuedoEjecutandoseSeDaPorCortado(t *testing.T) {
	root := proyectoConCola(t)
	mustPedir(t, root, "exit 0", "m", "")
	inicio := enCola
	if err := conCola(root, enCola, func(cola []Pedido) ([]Pedido, error) {
		cola[0].Estado, cola[0].Inicio = EstadoEjecutando, &inicio
		return cola, nil
	}); err != nil {
		t.Fatal(err)
	}
	cola := PedidosA(root, enCola.Add(TopeEjecucion+time.Minute))
	if cola[0].Estado != EstadoFallo || !strings.Contains(cola[0].Salida, "se cortó") || !cola[0].Ejecutable() {
		t.Fatalf("un pedido que no terminó queda como falla, para reintentarlo: %+v", cola[0])
	}
}

func TestPedidos_LosHechosSeVanDespuesDeMediaHora(t *testing.T) {
	root := proyectoConCola(t)
	mustPedir(t, root, "exit 0", "m", "")
	fin := enCola
	if err := conCola(root, enCola, func(cola []Pedido) ([]Pedido, error) {
		cola[0].Estado, cola[0].Fin = EstadoHecho, &fin
		return cola, nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(PedidosA(root, enCola.Add(VidaHecho-time.Minute))) != 1 {
		t.Fatal("recién hecho se sigue viendo")
	}
	if len(PedidosA(root, enCola.Add(VidaHecho+time.Minute))) != 0 {
		t.Fatal("pasada media hora ya no aparece")
	}
}

// FND-071: las teclas se recorrían cuando un pedido salía de la cola; la tecla que la persona veía cambiaba.
func TestPedidos_CadaPedidoConservaSuTecla_FND071(t *testing.T) {
	root := proyectoConCola(t)
	a := mustPedir(t, root, "exit 0", "primero", "")
	b := mustPedir(t, root, "echo dos", "segundo", "")
	if a.Tecla != "a" || b.Tecla != "b" {
		t.Fatalf("teclas en orden: %q %q", a.Tecla, b.Tecla)
	}
	accion, _ := BuscarAccion(Acciones(root, false), "pedido:"+a.ID)
	var salida bytes.Buffer
	accion.Ejecutar(root, "qdd", &salida)
	if TeclaDe(root, b.ID) != "b" {
		t.Fatalf("el segundo sigue con la b después de hacer el primero: %q", TeclaDe(root, b.ID))
	}
	c := mustPedir(t, root, "echo tres", "tercero", "")
	if c.Tecla != "a" {
		t.Fatalf("la tecla de un hecho queda libre para el siguiente: %q", c.Tecla)
	}
}

func TestPedidos_UnaColaDe260SinTeclaNiEstadoSeLee(t *testing.T) {
	root := proyectoConCola(t)
	ruta, _ := colaPath(root)
	os.MkdirAll(filepath.Dir(ruta), 0700)
	vieja := `[{"id":"1","comando":"exit 0","motivo":"m","carpeta":"` + filepath.ToSlash(root) + `","creado":"2026-10-30T12:00:00Z"},{"id":"2","comando":"echo","motivo":"n","carpeta":"` + filepath.ToSlash(root) + `","creado":"2026-10-30T12:00:00Z"}]`
	if err := os.WriteFile(ruta, []byte(vieja), 0600); err != nil {
		t.Fatal(err)
	}
	cola := Pedidos(root)
	if len(cola) != 2 || cola[0].Estado != EstadoPendiente || cola[0].Tecla != "a" || cola[1].Tecla != "b" {
		t.Fatalf("la cola de la versión anterior queda pendiente y con teclas: %+v", cola)
	}
}

func TestEjecutar_UnPedidoQueFallaQuedaParaReintentar_CCC_C18(t *testing.T) {
	root := proyectoConCola(t)
	p := mustPedir(t, root, "echo no hay permiso; exit 3", "Falla", "")
	a, _ := BuscarAccion(Acciones(root, false), "pedido:"+p.ID)
	var salida bytes.Buffer
	if codigo := a.Ejecutar(root, "qdd", &salida); codigo != 3 {
		t.Fatalf("se devuelve el código del comando: %d", codigo)
	}
	f, _ := BuscarAccion(Acciones(root, false), "pedido:"+p.ID)
	if f.Estado != EstadoFallo || f.Tecla != "a" || !strings.Contains(f.Resultado, "no hay permiso") {
		t.Fatalf("un pedido que falló muestra su error y conserva su tecla para reintentarlo: %+v", f)
	}
	if codigo := f.Ejecutar(root, "qdd", &salida); codigo != 3 {
		t.Fatalf("reintentar lo vuelve a correr: %d", codigo)
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

// Pedido del usuario (2026-10-31): lo que se lee en el panel es un resumen, estilo chat; un motivo largo se rechaza.
func TestPedir_ElMotivoEsUnResumenCorto(t *testing.T) {
	root := proyectoConCola(t)
	largo := strings.Repeat("x", MaxMotivo+1)
	if _, err := Pedir(root, "git push origin v1", largo, "", enCola); err == nil || !strings.Contains(err.Error(), "resum") {
		t.Fatalf("un motivo de más de %d caracteres se rechaza pidiendo un resumen: %v", MaxMotivo, err)
	}
	if _, err := Pedir(root, "git push origin v1", "Publicar v1 → npm preview", "", enCola); err != nil {
		t.Fatal(err)
	}
}
