package flujo

import (
	"os"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// RFC 0025 (decisión de la persona, 2026-11-08): «los flujos se tienen que versionar». Cada cambio guardado sube una
// revisión con su autor en el historial; una versión con nombre guarda la foto entera; volver a una es una revisión nueva.

func TestRevision_CadaEdicionSubeYQuedaEnElHistorialConSuAutor(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	editar(t, root, DePersona, Agregar("b", "Cobra", TipoPaso, "", false))
	f, _ := editar(t, root, DeIA, Quitar("a"))
	if f.Revision != 2 {
		t.Fatalf("dos ediciones: revisión 2, no %d", f.Revision)
	}
	h := Historial(root, "pago")
	if len(h) != 2 || h[0].Rev != 1 || h[0].Autor != DePersona || h[1].Autor != DeIA || !strings.Contains(h[1].Resumen, "a") {
		t.Fatalf("el historial dice quién cambió qué: %+v", h)
	}
}

func TestRevision_UnaEdicionRechazadaNoSube(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	if _, _, err := Editar(root, "pago", DePersona, ahoraDePrueba, Quitar("zzz")); err == nil {
		t.Fatal("quitar un nodo que no existe falla")
	}
	if f, _ := Cargar(root, "pago"); f.Revision != 0 || len(Historial(root, "pago")) != 0 {
		t.Fatalf("nada cambió: %+v", f.Revision)
	}
}

func TestRevision_DeshacerBorraLaEdicionComoSiNoHubieraPasado(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	editar(t, root, DePersona, Agregar("b", "Cobra", TipoPaso, "", false))
	editar(t, root, DePersona, Quitar("a"))
	if err := Deshacer(root, "pago"); err != nil {
		t.Fatal(err)
	}
	f, _ := Cargar(root, "pago")
	if h := Historial(root, "pago"); f.Revision != 1 || len(h) != 1 || h[0].Rev != 1 {
		t.Fatalf("lo deshecho no queda: revisión %d, historial %+v", f.Revision, h)
	}
}

func TestVersion_GuardaLaFotoYSePuedeCompararYVolver(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	editar(t, root, DePersona, Agregar("b", "Cobra", TipoPaso, "", false))
	v, err := CerrarVersion(root, "pago", "acordada con Ana", ahoraDePrueba)
	if err != nil || v.N != 1 || v.Revision != 1 || v.Nombre != "acordada con Ana" {
		t.Fatalf("versión 1 sobre la revisión 1: %+v %v", v, err)
	}
	editar(t, root, DePersona, Quitar("cobra"))
	editar(t, root, DePersona, Modificar("b", ptr("Revisa"), nil, nil))
	v1, err := CargarVersion(root, "pago", 1)
	if err != nil {
		t.Fatal(err)
	}
	actual, _ := Cargar(root, "pago")
	d := Comparar(v1, actual)
	if len(d.Quitados) != 1 || d.Quitados[0].ID != "cobra" || len(d.Cambiados) != 1 || d.Cambiados[0].ID != "b" || len(d.Agregados) != 0 {
		t.Fatalf("de v1 a hoy: se quitó cobra y cambió b: %+v", d)
	}
	f, err := Volver(root, "pago", 1, DePersona, ahoraDePrueba)
	if err != nil {
		t.Fatal(err)
	}
	if f.Revision != 4 || len(f.Versiones) != 1 || len(Comparar(v1, f).Cambiados)+len(Comparar(v1, f).Quitados)+len(Comparar(v1, f).Agregados) != 0 {
		t.Fatalf("volver es una revisión nueva con los nodos de v1 y conserva las versiones: %+v", f)
	}
	if h := Historial(root, "pago"); !strings.Contains(h[len(h)-1].Resumen, "v1") {
		t.Fatalf("el historial dice a qué versión volvió: %+v", h[len(h)-1])
	}
}

func TestVersion_SinNombreONumeroQueNoExisteAvisa(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	if _, err := CerrarVersion(root, "pago", "  ", ahoraDePrueba); err == nil {
		t.Fatal("una versión lleva nombre")
	}
	if _, err := CargarVersion(root, "pago", 7); err == nil {
		t.Fatal("la versión 7 no existe")
	}
}

func TestHistorial_NoEsUnFlujoDeLaLista(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	editar(t, root, DePersona, Quitar("a"))
	CerrarVersion(root, "pago", "uno", ahoraDePrueba)
	if l := Listar(root, sinEstado); len(l) != 1 || l[0].Problema != "" {
		t.Fatalf("el historial y las versiones no aparecen como flujos: %+v", l)
	}
	if _, err := os.Stat(rutaHistorial(root, "pago")); err != nil {
		t.Fatalf("el historial vive junto al flujo, versionado: %v", err)
	}
}

func TestRevisarHistorial_UnaEdicionAManoSeVeYSeRepara(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	editar(t, root, DePersona, Quitar("a"))
	if p := RevisarHistorial(root, "pago"); p != "" {
		t.Fatalf("cuadra: %s", p)
	}
	f, _ := Cargar(root, "pago")
	f.Revision = 5
	Guardar(root, f)
	if p := RevisarHistorial(root, "pago"); !strings.Contains(p, "5") {
		t.Fatalf("revisión 5 sin historial: %q", p)
	}
	if err := RepararHistorial(root, "pago", ahoraDePrueba); err != nil {
		t.Fatal(err)
	}
	if p := RevisarHistorial(root, "pago"); p != "" {
		t.Fatalf("reparado: %s", p)
	}
}

func ptr(s string) *string { return &s }
