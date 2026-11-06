package flujo

import (
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

var ahoraDePrueba = time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)

func flujoGuardado(t *testing.T, nodos ...Nodo) string {
	t.Helper()
	root := t.TempDir()
	if err := Guardar(root, deNodos(nodos...)); err != nil {
		t.Fatal(err)
	}
	return root
}

func cadena() []Nodo {
	return []Nodo{paso("a", "Llega", "b"), paso("b", "Valida", "c"), resultado("c", "Fin")}
}

func editar(t *testing.T, root, de string, cambio Cambio) (Flujo, string) {
	t.Helper()
	f, linea, err := Editar(root, "pago", de, ahoraDePrueba, cambio)
	if err != nil {
		t.Fatal(err)
	}
	return f, linea
}

func nodoDe(t *testing.T, f Flujo, id string) Nodo {
	t.Helper()
	for _, n := range f.Nodos {
		if n.ID == id {
			return n
		}
	}
	t.Fatalf("no está el nodo %q en %+v", id, f.Nodos)
	return Nodo{}
}

func TestEditar_AgregarDespuesQuedaEntreElNodoYLoQueSeguia(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	f, linea := editar(t, root, DePersona, Agregar("b", "Cobra el pago", TipoPaso, "con la tarjeta", false))
	if got := nodoDe(t, f, "b").Siguen; len(got) != 1 || got[0] != "cobra-el-pago" {
		t.Fatalf("b sigue en %v", got)
	}
	nuevo := nodoDe(t, f, "cobra-el-pago")
	if nuevo.Descripcion != "con la tarjeta" || len(nuevo.Siguen) != 1 || nuevo.Siguen[0] != "c" {
		t.Fatalf("nuevo: %+v", nuevo)
	}
	if !strings.HasPrefix(linea, "+ ") || !strings.Contains(linea, "Cobra el pago") || !strings.Contains(linea, "b") {
		t.Fatalf("línea: %q", linea)
	}
	guardado, err := Cargar(root, "pago")
	if err != nil || len(guardado.Nodos) != 4 {
		t.Fatalf("guardado: %+v, %v", guardado, err)
	}
}

func TestEditar_AgregarRamaNoTocaLoQueYaSeguia(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	f, _ := editar(t, root, DePersona, Agregar("b", "Rechazo", TipoResultado, "", true))
	if got := nodoDe(t, f, "b").Siguen; len(got) != 2 || got[0] != "c" || got[1] != "rechazo" {
		t.Fatalf("b sigue en %v", got)
	}
	if n := nodoDe(t, f, "rechazo"); n.Tipo != TipoResultado || len(n.Siguen) != 0 {
		t.Fatalf("rama: %+v", n)
	}
}

func TestEditar_UnIDNuevoNoChocaYNoTraeAcentos(t *testing.T) {
	root := flujoGuardado(t, paso("a", "Inicio", "valida"), paso("valida", "Valida"))
	f, _ := editar(t, root, DePersona, Agregar("a", "Válida", TipoPaso, "", false))
	nodoDe(t, f, "valida-2")
	f, _ = editar(t, root, DePersona, Agregar("a", "¿?", TipoPaso, "", false))
	nodoDe(t, f, "nodo")
}

func TestEditar_ModificarDiceQueCambio(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	nombre, descripcion, tipo := "Revisa", "con la firma", TipoResultado
	f, linea := editar(t, root, DePersona, Modificar("b", &nombre, &descripcion, &tipo))
	n := nodoDe(t, f, "b")
	if n.Nombre != nombre || n.Descripcion != descripcion || n.Tipo != tipo {
		t.Fatalf("nodo: %+v", n)
	}
	if !strings.HasPrefix(linea, "~ b") || !strings.Contains(linea, "«Valida» → «Revisa»") {
		t.Fatalf("línea: %q", linea)
	}
}

func TestEditar_ModificarSinCambiosNoEscribe(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	nombre := "Valida"
	if _, _, err := Editar(root, "pago", DePersona, ahoraDePrueba, Modificar("b", &nombre, nil, nil)); err == nil {
		t.Fatal("sin cambios tiene que avisar")
	}
	if p := PorEnviar(root, "pago"); len(p) != 0 {
		t.Fatalf("anotó %v", p)
	}
}

func TestEditar_QuitarUneLoDeAntesConLoDeDespues(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	f, linea := editar(t, root, DePersona, Quitar("b"))
	if got := nodoDe(t, f, "a").Siguen; len(got) != 1 || got[0] != "c" {
		t.Fatalf("a sigue en %v", got)
	}
	if !strings.HasPrefix(linea, "− b") {
		t.Fatalf("línea: %q", linea)
	}
}

func TestEditar_QuitarNoRepiteUnaSalidaYNoDejaUnCicloSobreSiMismo(t *testing.T) {
	root := flujoGuardado(t, paso("a", "A", "b", "c"), paso("b", "B", "c", "a"), resultado("c", "C"))
	f, _ := editar(t, root, DePersona, Quitar("b"))
	if got := nodoDe(t, f, "a").Siguen; len(got) != 1 || got[0] != "c" {
		t.Fatalf("a sigue en %v", got)
	}
}

func TestEditar_QuitarElPrimeroDejaAlSiguienteDeInicio(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	f, _ := editar(t, root, DePersona, Quitar("a"))
	if f.Nodos[0].ID != "b" {
		t.Fatalf("empieza en %q", f.Nodos[0].ID)
	}
}

func TestEditar_QuitarLoQueDejaOtroNodoSinLlegarSeRechazaYNoEscribe(t *testing.T) {
	root := flujoGuardado(t, paso("a", "A", "b", "c"), resultado("b", "B"), resultado("c", "C"))
	antes, _ := os.ReadFile(ruta(root, "pago"))
	if _, _, err := Editar(root, "pago", DePersona, ahoraDePrueba, Quitar("a")); err == nil || !strings.Contains(err.Error(), "dos salidas") {
		t.Fatalf("err = %v", err)
	}
	unico := flujoGuardado(t, resultado("a", "A"))
	if _, _, err := Editar(unico, "pago", DePersona, ahoraDePrueba, Quitar("a")); err == nil {
		t.Fatal("quitar el único nodo tiene que rechazarse")
	}
	despues, _ := os.ReadFile(ruta(root, "pago"))
	if string(antes) != string(despues) {
		t.Fatal("una edición rechazada cambió el archivo")
	}
	if p := PorEnviar(root, "pago"); len(p) != 0 {
		t.Fatalf("una edición rechazada no le llega a la IA: %v", p)
	}
}

func TestEditar_QuitarNoPierdeLosComentariosDelNodo(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	editar(t, root, DePersona, Comentar("b", "ojo con esto", DePersona, ahoraDePrueba))
	f, _ := editar(t, root, DePersona, Quitar("b"))
	c := nodoDe(t, f, "a").Comentarios
	if len(c) != 1 || c[0].Texto != "ojo con esto" || c[0].DeNodo != "Valida" {
		t.Fatalf("comentarios: %+v", c)
	}
}

func TestEditar_MoverLoCuelgaDespuesDeOtro(t *testing.T) {
	root := flujoGuardado(t, paso("a", "A", "b"), paso("b", "B", "c"), paso("c", "C", "d"), resultado("d", "D"))
	f, linea := editar(t, root, DePersona, Mover("b", "c"))
	if got := nodoDe(t, f, "a").Siguen; got[0] != "c" {
		t.Fatalf("a sigue en %v", got)
	}
	if got := nodoDe(t, f, "c").Siguen; got[0] != "b" {
		t.Fatalf("c sigue en %v", got)
	}
	if got := nodoDe(t, f, "b").Siguen; got[0] != "d" {
		t.Fatalf("b sigue en %v", got)
	}
	if !strings.HasPrefix(linea, "↷ b") {
		t.Fatalf("línea: %q", linea)
	}
}

func TestEditar_MoverNoAceptaElMismoNodoNiElPrimeroConDosSalidas(t *testing.T) {
	root := flujoGuardado(t, paso("a", "A", "b", "c"), resultado("b", "B"), resultado("c", "C"))
	if _, _, err := Editar(root, "pago", DePersona, ahoraDePrueba, Mover("b", "b")); err == nil {
		t.Fatal("moverse a sí mismo tiene que rechazarse")
	}
	if _, _, err := Editar(root, "pago", DePersona, ahoraDePrueba, Mover("a", "b")); err == nil {
		t.Fatal("mover el primero con dos salidas tiene que rechazarse")
	}
}

func TestEditar_ConectarYDesconectar(t *testing.T) {
	root := flujoGuardado(t, paso("a", "A", "b", "c"), paso("b", "B", "c"), resultado("c", "C"))
	f, _ := editar(t, root, DePersona, Desconectar("a", "c"))
	if got := nodoDe(t, f, "a").Siguen; len(got) != 1 || got[0] != "b" {
		t.Fatalf("a sigue en %v", got)
	}
	f, _ = editar(t, root, DePersona, Conectar("c", "a"))
	if got := nodoDe(t, f, "c").Siguen; len(got) != 1 || got[0] != "a" {
		t.Fatalf("c sigue en %v", got)
	}
	if _, _, err := Editar(root, "pago", DePersona, ahoraDePrueba, Conectar("c", "a")); err == nil {
		t.Fatal("una conexión repetida tiene que avisar")
	}
	if _, _, err := Editar(root, "pago", DePersona, ahoraDePrueba, Desconectar("a", "b")); err == nil || !strings.Contains(err.Error(), "no se llega") {
		t.Fatalf("desconectar lo que deja un nodo sin llegar: %v", err)
	}
}

func TestEditar_UnNodoQueNoEstaLoDice(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	if _, _, err := Editar(root, "pago", DePersona, ahoraDePrueba, Quitar("z")); err == nil || !strings.Contains(err.Error(), `"z"`) {
		t.Fatalf("err = %v", err)
	}
}

func TestComentar_QuedaEnElArchivoSinControlesYConTope(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	editar(t, root, DePersona, Comentar("b", "uno\ndos\x1b[31m\t tres", DePersona, ahoraDePrueba))
	f, err := Cargar(root, "pago")
	if err != nil {
		t.Fatal(err)
	}
	c := nodoDe(t, f, "b").Comentarios
	if len(c) != 1 || c[0].De != DePersona || c[0].Texto != "uno dos[31m tres" || c[0].Cuando != "2026-11-06T10:00:00Z" {
		t.Fatalf("comentario: %+v", c)
	}
	if _, _, err := Editar(root, "pago", DePersona, ahoraDePrueba, Comentar("b", "  \n ", DePersona, ahoraDePrueba)); err == nil {
		t.Fatal("un comentario vacío tiene que rechazarse")
	}
	largo := strings.Repeat("x", maxComentario+1)
	if _, _, err := Editar(root, "pago", DePersona, ahoraDePrueba, Comentar("b", largo, DePersona, ahoraDePrueba)); err == nil {
		t.Fatal("un comentario demasiado largo tiene que rechazarse")
	}
}

func TestComentar_NoSeDibujaNiCambiaLasLineas(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	antes, _ := Cargar(root, "pago")
	f, _ := editar(t, root, DePersona, Comentar("b", "ojo", DePersona, ahoraDePrueba))
	if strings.Join(Dibujar(antes, 40, sinEstado), "\n") != strings.Join(Dibujar(f, 40, sinEstado), "\n") {
		t.Fatal("un comentario cambió el dibujo")
	}
}

func TestValidar_UnComentarioRaroSeNombra(t *testing.T) {
	f := deNodos(paso("a", "A"))
	f.Nodos[0].Comentarios = []Comentario{{De: "otro", Texto: "x"}, {De: DeIA, Texto: " "}}
	problemas := strings.Join(Validar(f), "\n")
	if !strings.Contains(problemas, `"otro"`) || !strings.Contains(problemas, "vacío") {
		t.Fatalf("problemas: %q", problemas)
	}
}

func TestEnviar_LaIARecibeCadaEdicionUnaVezEnUnaLinea(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	editar(t, root, DePersona, Agregar("b", "Cobra", TipoPaso, "", false))
	editar(t, root, DePersona, Quitar("a"))
	if p := PorEnviar(root, "pago"); len(p) != 2 {
		t.Fatalf("por enviar: %v", p)
	}
	mensaje, n := Enviar(root, "pago")
	if n != 2 || !strings.Contains(mensaje, "Pago") || !strings.Contains(mensaje, "+ ") || !strings.Contains(mensaje, "− a") {
		t.Fatalf("mensaje %q, n %d", mensaje, n)
	}
	if !strings.Contains(mensaje, "qdd flujo ver pago") {
		t.Fatalf("el mensaje no dice cómo verlo: %q", mensaje)
	}
	if otra, n := Enviar(root, "pago"); otra != "" || n != 0 {
		t.Fatalf("se envió dos veces: %q %d", otra, n)
	}
}

func TestEnviar_UnaLineaLargaSeCorta(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	_, linea := editar(t, root, DePersona, Agregar("b", strings.Repeat("palabra ", 60), TipoPaso, "", false))
	if len([]rune(linea)) > maxLineaEdicion {
		t.Fatalf("línea de %d", len([]rune(linea)))
	}
}

func TestEnviar_ElComentarioVaEntero(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	texto := strings.Repeat("palabra ", 60) + "fin"
	editar(t, root, DePersona, Comentar("b", texto, DePersona, ahoraDePrueba))
	mensaje, _ := Enviar(root, "pago")
	if !strings.Contains(mensaje, texto) || !strings.Contains(mensaje, "qdd flujo comentar pago b") {
		t.Fatalf("mensaje: %q", mensaje)
	}
}

func TestEnviar_LaIANoSeLlamaASiMisma(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	editar(t, root, DeIA, Comentar("b", "respuesta", DeIA, ahoraDePrueba))
	editar(t, root, DeIA, Quitar("a"))
	if m, n := Enviar(root, "pago"); m != "" || n != 0 {
		t.Fatalf("mensaje a la propia IA: %q", m)
	}
}

func TestDevolver_LoQueNoSeEntregoVuelveAQuedarPorEnviar(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	editar(t, root, DePersona, Quitar("a"))
	editar(t, root, DePersona, Quitar("b"))
	_, n := Enviar(root, "pago")
	Devolver(root, "pago", n)
	if p := PorEnviar(root, "pago"); len(p) != 2 {
		t.Fatalf("por enviar: %v", p)
	}
	if err := Deshacer(root, "pago"); err != nil {
		t.Fatalf("lo devuelto se puede deshacer: %v", err)
	}
}

func TestDetalle_PedirloNoCambiaElFlujoYLeDiceALaIAComoResponder(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	antes, _ := os.ReadFile(ruta(root, "pago"))
	if err := PedirDetalle(root, "pago", "b", ahoraDePrueba); err != nil {
		t.Fatal(err)
	}
	despues, _ := os.ReadFile(ruta(root, "pago"))
	if string(antes) != string(despues) {
		t.Fatal("pedir detalle cambió el archivo")
	}
	mensaje, _ := Enviar(root, "pago")
	if !strings.Contains(mensaje, "🔎") || !strings.Contains(mensaje, "qdd flujo detalle pago b") {
		t.Fatalf("mensaje: %q", mensaje)
	}
	if err := PedirDetalle(root, "pago", "z", ahoraDePrueba); err == nil {
		t.Fatal("pedir detalle de un nodo que no está tiene que avisar")
	}
}

func TestDeshacer_VuelveAtrasDeUnaEnUna(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	original, _ := os.ReadFile(ruta(root, "pago"))
	editar(t, root, DePersona, Agregar("b", "Cobra", TipoPaso, "", false))
	editar(t, root, DePersona, Quitar("a"))
	if err := Deshacer(root, "pago"); err != nil {
		t.Fatal(err)
	}
	f, _ := Cargar(root, "pago")
	if f.Nodos[0].ID != "a" {
		t.Fatalf("empieza en %q", f.Nodos[0].ID)
	}
	if err := Deshacer(root, "pago"); err != nil {
		t.Fatal(err)
	}
	final, _ := os.ReadFile(ruta(root, "pago"))
	if string(final) != string(original) {
		t.Fatalf("no volvió al original:\n%s", final)
	}
	if p := PorEnviar(root, "pago"); len(p) != 0 {
		t.Fatalf("lo deshecho no le llega a la IA: %v", p)
	}
	if err := Deshacer(root, "pago"); err == nil {
		t.Fatal("sin nada que deshacer tiene que avisar")
	}
}

func TestDeshacer_LoEnviadoYaNoSeDeshace(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	editar(t, root, DePersona, Quitar("a"))
	Enviar(root, "pago")
	if err := Deshacer(root, "pago"); err == nil || !strings.Contains(err.Error(), "ya la recibió") {
		t.Fatalf("err = %v", err)
	}
}

func TestDeshacer_NoPisaLoQueLaIACambioDespues(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	editar(t, root, DePersona, Quitar("a"))
	editar(t, root, DeIA, Comentar("b", "lo vi", DeIA, ahoraDePrueba))
	if err := Deshacer(root, "pago"); err == nil || !strings.Contains(err.Error(), "cambió") {
		t.Fatalf("err = %v", err)
	}
	f, _ := Cargar(root, "pago")
	if len(nodoDe(t, f, "b").Comentarios) != 1 {
		t.Fatal("deshacer pisó el comentario de la IA")
	}
}

func TestEditar_DosEscriturasALaVezNoPierdenNinguna(t *testing.T) {
	root := flujoGuardado(t, paso("a", "A"))
	var g sync.WaitGroup
	for i := 0; i < 20; i++ {
		g.Add(1)
		go func() {
			defer g.Done()
			if _, _, err := Editar(root, "pago", DePersona, ahoraDePrueba, Comentar("a", "hola", DePersona, ahoraDePrueba)); err != nil {
				t.Error(err)
			}
		}()
	}
	g.Wait()
	f, _ := Cargar(root, "pago")
	if len(f.Nodos[0].Comentarios) != 20 {
		t.Fatalf("quedaron %d de 20", len(f.Nodos[0].Comentarios))
	}
	if p := PorEnviar(root, "pago"); len(p) != 20 {
		t.Fatalf("por enviar %d de 20", len(p))
	}
}

func TestEditar_ElRegistroTieneTope(t *testing.T) {
	root := flujoGuardado(t, paso("a", "A"))
	for i := 0; i < maxRegistro+5; i++ {
		editar(t, root, DePersona, Comentar("a", "hola", DePersona, ahoraDePrueba))
	}
	if p := PorEnviar(root, "pago"); len(p) != maxRegistro {
		t.Fatalf("registro de %d", len(p))
	}
}

func TestEditar_UnRegistroRotoNoImpideEditar(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	editar(t, root, DePersona, Quitar("a"))
	if err := os.WriteFile(rutaRegistro(root, "pago"), []byte("{no es json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	editar(t, root, DePersona, Quitar("b"))
	if p := PorEnviar(root, "pago"); len(p) != 1 {
		t.Fatalf("por enviar: %v", p)
	}
}

func TestEditar_ElQuePasariaElTopeNoSeGuarda(t *testing.T) {
	root := flujoGuardado(t, paso("a", "A"))
	f, _ := Cargar(root, "pago")
	f.Descripcion = strings.Repeat("x", MaxBytes-2000)
	if err := Guardar(root, f); err != nil {
		t.Fatal(err)
	}
	texto := strings.Repeat("y", maxComentario)
	var err error
	for i := 0; i < 3 && err == nil; i++ {
		_, _, err = Editar(root, "pago", DePersona, ahoraDePrueba, Comentar("a", texto, DePersona, ahoraDePrueba))
	}
	if err == nil || !strings.Contains(err.Error(), "pesa") {
		t.Fatalf("err = %v", err)
	}
}

func TestReemplazar_ConservaLosComentariosDeLaPersona(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	editar(t, root, DePersona, Comentar("b", "de un nodo que sigue", DePersona, ahoraDePrueba))
	editar(t, root, DePersona, Comentar("c", "de un nodo que se va", DePersona, ahoraDePrueba))
	nuevo := deNodos(paso("a", "Llega", "b"), resultado("b", "Valida distinto"))
	movidos, err := Reemplazar(root, nuevo)
	if err != nil {
		t.Fatal(err)
	}
	f, _ := Cargar(root, "pago")
	if c := nodoDe(t, f, "b").Comentarios; len(c) != 1 || c[0].Texto != "de un nodo que sigue" {
		t.Fatalf("b: %+v", c)
	}
	c := nodoDe(t, f, "a").Comentarios
	if movidos != 1 || len(c) != 1 || c[0].DeNodo != "Fin" {
		t.Fatalf("movidos %d, a: %+v", movidos, c)
	}
}

func TestNuevo_CreaUnFlujoQueSeAbreYNoPisaOtro(t *testing.T) {
	root := t.TempDir()
	f, err := Nuevo(root, "Cómo se cobra")
	if err != nil {
		t.Fatal(err)
	}
	if f.ID != "como-se-cobra" || len(f.Nodos) != 1 || f.Nodos[0].Tipo != TipoResultado {
		t.Fatalf("flujo: %+v", f)
	}
	otro, err := Nuevo(root, "Cómo se cobra")
	if err != nil || otro.ID != "como-se-cobra-2" {
		t.Fatalf("otro: %+v, %v", otro, err)
	}
	if _, err := Nuevo(root, "  "); err == nil {
		t.Fatal("sin nombre tiene que rechazarse")
	}
}

func TestDibujarElegido_MarcaElNodoSinCambiarElAncho(t *testing.T) {
	f := deNodos(resultado("a", "Inicio", "b"), paso("b", "Valida", "c"), resultado("c", "Fin"))
	normal := Dibujar(f, 40, sinEstado)
	caja := DibujarElegido(f, 40, sinEstado, "b")
	if !strings.Contains(strings.Join(caja, "\n"), "┏") || strings.Contains(strings.Join(normal, "\n"), "┏") {
		t.Fatalf("caja:\n%s", strings.Join(caja, "\n"))
	}
	circulo := DibujarElegido(f, 40, sinEstado, "c")
	if !strings.Contains(strings.Join(circulo, "\n"), "▶ Fin ◀") {
		t.Fatalf("círculo:\n%s", strings.Join(circulo, "\n"))
	}
	if len(normal) != len(caja) {
		t.Fatalf("filas %d y %d", len(normal), len(caja))
	}
	for i := range normal {
		if len([]rune(normal[i])) != len([]rune(caja[i])) {
			t.Fatalf("fila %d cambió de ancho", i)
		}
	}
}
