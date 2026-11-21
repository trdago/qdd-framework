package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/materializar"
)

// Rule: Prohibido usar 'else'
// El panel le pide a qdd los cuadros de lo que cambió, una vez por cambio, y solo los reproduce.

func TestConsolaMaterializar_DevuelveLosCuadrosDeLoQueCambio(t *testing.T) {
	entrada := `{"lineas":[[{"texto":"✅","color":"green"}]],"abrir":true,"semilla":3}`
	var out, errs bytes.Buffer
	if code := consolaMaterializar(strings.NewReader(entrada), &out, &errs, materializar.Estilo{Acento: "#7daea3", AbrirMs: 800, CerrarMs: 400}); code != 0 {
		t.Fatalf("exit %d: %s", code, errs.String())
	}
	var r respuestaMaterializar
	if err := json.Unmarshal(out.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.CuadroMs != 40 || len(r.Cuadros) != 20 || r.Cuadros[19][0][0].Texto != "✅" {
		t.Fatalf("20 cuadros de 40 ms que terminan en la celda tal cual: %+v", r)
	}
}

// MEJ-017: abrir el panel lanzaba un proceso por celda (21 con 4 tareas). Un dibujo pide todo de una vez, y cada pedido
// conserva su dirección y su semilla.
func TestConsolaMaterializar_UnLoteDevuelveCadaPedidoEnOrden(t *testing.T) {
	entrada := `{"lote":[{"lineas":[[{"texto":"a"}]],"abrir":true,"semilla":1},{"lineas":[[{"texto":"b"}]],"abrir":false,"semilla":2}]}`
	var out, errs bytes.Buffer
	estilo := materializar.Estilo{AbrirMs: 800, CerrarMs: 400}
	if code := consolaMaterializar(strings.NewReader(entrada), &out, &errs, estilo); code != 0 {
		t.Fatalf("exit %d: %s", code, errs.String())
	}
	var r respuestaLote
	if err := json.Unmarshal(out.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if len(r.Lote) != 2 {
		t.Fatalf("lote = %+v", r)
	}
	if got := r.Lote[0].Cuadros; len(got) != 20 || got[19][0][0].Texto != "a" {
		t.Fatalf("abrir: 20 cuadros que terminan en «a»: %+v", got)
	}
	if got := r.Lote[1].Cuadros; len(got) != 10 {
		t.Fatalf("cerrar dura la mitad: %d cuadros", len(got))
	}
}

func TestConsolaMaterializar_CerrarUsaSuDuracionYUnaEntradaRotaEsError(t *testing.T) {
	var out, errs bytes.Buffer
	estilo := materializar.Estilo{AbrirMs: 800, CerrarMs: 400}
	if code := consolaMaterializar(strings.NewReader(`{"lineas":[[{"texto":"a"}]],"abrir":false}`), &out, &errs, estilo); code != 0 {
		t.Fatal(errs.String())
	}
	var r respuestaMaterializar
	json.Unmarshal(out.Bytes(), &r)
	if len(r.Cuadros) != 10 {
		t.Fatalf("al irse dura 0,4 s: %d cuadros", len(r.Cuadros))
	}
	if code := consolaMaterializar(strings.NewReader(`{no`), &out, &errs, estilo); code == 0 {
		t.Fatal("una entrada que no es JSON es un error, no cuadros inventados")
	}
}

func TestConsolaEfectos_ApagaEnciendeYElPanelLoDice(t *testing.T) {
	home := t.TempDir()
	t.Setenv("QDD_HOME", home)
	t.Setenv(materializar.EnvEfectos, "")
	var out bytes.Buffer
	if code := consolaEfectos([]string{"off"}, &out); code != 0 || !strings.Contains(out.String(), "apagados") {
		t.Fatalf("off: %d %s", code, out.String())
	}
	d := consola.DatosConsola{}
	conEfectos(&d)
	if d.Efectos == nil || d.Efectos.Activo {
		t.Fatalf("el panel los ve apagados: %+v", d.Efectos)
	}
	out.Reset()
	if code := consolaEfectos([]string{"on"}, &out); code != 0 || !strings.Contains(out.String(), "encendidos") {
		t.Fatalf("on: %d %s", code, out.String())
	}
	conEfectos(&d)
	if !d.Efectos.Activo || d.Efectos.CuadroMs != 40 || d.Efectos.AbrirMs == 0 {
		t.Fatalf("encendidos, con su estilo: %+v", d.Efectos)
	}
	if code := consolaEfectos([]string{"quizas"}, &out); code == 0 {
		t.Fatal("solo on u off")
	}
}

func TestDoctorEfectos_DiceComoEstanYReparaUnArchivoRoto(t *testing.T) {
	home := t.TempDir()
	t.Setenv("QDD_HOME", home)
	t.Setenv(materializar.EnvEfectos, "")
	if item := checkEfectos(); !item.Success || !strings.Contains(item.Error, "encendidos") {
		t.Fatalf("encendidos por defecto, con su estilo: %+v", item)
	}
	escribirArchivoCmd(t, materializar.RutaEfectos(home), "{roto")
	item := checkEfectos()
	if item.Success {
		t.Fatalf("un interruptor ilegible es una falla: %+v", item)
	}
	g := &CheckGroup{Name: grupoConsola, Items: []*CheckItem{item}}
	repairConsola(t.TempDir(), g)
	if item := checkEfectos(); !item.Success {
		t.Fatalf("doctor --fix lo deja legible y encendido: %+v", item)
	}
}

func escribirArchivoCmd(t *testing.T, ruta, contenido string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte(contenido), 0o644); err != nil {
		t.Fatal(err)
	}
}
