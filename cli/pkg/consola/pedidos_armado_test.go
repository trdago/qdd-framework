package consola

import (
	"bytes"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// Pedido del usuario (2026-11-01): «al ya tener mi intención no debería tener que apretar yo de nuevo; si es solo
// esperar, ya está mi intención» y «aprovechando código determinista podemos quedar escuchando para que cuando esté
// listo se ejecute mi acción». Un pedido confirmado que dice «todavía no» (75) o que espera a otro queda armado: el
// mismo proceso de qdd que lo corre lo reintenta solo hasta que sale bien o falla de verdad.

// cadasCortos acorta los intervalos del armado para la prueba.
func cadasCortos(t *testing.T) {
	t.Helper()
	antes, antesEspera, antesTope := CadaArmado, CadaEspera, TopeArmado
	CadaArmado, CadaEspera, TopeArmado = 30*time.Millisecond, 20*time.Millisecond, time.Minute
	t.Cleanup(func() { CadaArmado, CadaEspera, TopeArmado = antes, antesEspera, antesTope })
}

// correrAparte corre el pedido ya aceptado en este proceso, como lo haría «qdd consola correr», y devuelve su código.
func correrAparte(root, id string) chan int {
	fin := make(chan int, 1)
	go func() { fin <- CorrerLanzado(root, id, &bytes.Buffer{}) }()
	return fin
}

func TestArmado_TodaviaNoSeReintentaSoloHastaQueSaleBien(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("comando de prueba en sh")
	}
	cadasCortos(t)
	root := proyectoConCola(t)
	listo := t.TempDir() + "/listo"
	p := mustPedir(t, root, "test -f '"+listo+"' || { echo 'npm preview tiene 2.8.0-rc.12'; exit 75; }; echo instalado", "Instalar rc.13", "")
	if _, err := aceptar(root, p.ID, "", time.Now()); err != nil {
		t.Fatal(err)
	}
	fin := correrAparte(root, p.ID)
	armado := esperarEstado(t, root, p.ID, EstadoArmado)
	if !strings.Contains(armado.Aviso, "rc.12") || armado.Proximo == nil || armado.Intentos < 1 {
		t.Fatalf("queda armado con lo que dijo, cuándo reintenta y cuántas veces lo intentó: %+v", armado)
	}
	a, _ := BuscarAccion(Acciones(root, false), "pedido:"+p.ID)
	if a.Estado != EstadoArmado || a.Tecla != "" || a.Proximo == nil {
		t.Fatalf("el panel lo muestra armado, sin tecla: tu intención ya está tomada: %+v", a)
	}
	if err := os.WriteFile(listo, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if codigo := <-fin; codigo != 0 {
		t.Fatalf("cuando la condición se cumple corre y termina bien: %d", codigo)
	}
	hecho := esperarEstado(t, root, p.ID, EstadoHecho)
	if !strings.Contains(hecho.Salida, "instalado") || hecho.Aviso != "" {
		t.Fatalf("hecho, sin el aviso viejo: %+v", hecho)
	}
}

func TestArmado_UnaFallaDeVerdadNoSeReintenta(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("comando de prueba en sh")
	}
	cadasCortos(t)
	root := proyectoConCola(t)
	p := mustPedir(t, root, "echo roto; exit 3", "Romper", "")
	if _, err := aceptar(root, p.ID, "", time.Now()); err != nil {
		t.Fatal(err)
	}
	if codigo := <-correrAparte(root, p.ID); codigo != 3 {
		t.Fatalf("una falla termina en el acto: %d", codigo)
	}
	if f := esperarEstado(t, root, p.ID, EstadoFallo); f.Intentos > 1 {
		t.Fatalf("y no se reintenta: %+v", f)
	}
}

func TestArmado_UnPedidoQueEsperaSeConfirmaAntesYCorreAlTerminarElOtro(t *testing.T) {
	cadasCortos(t)
	root := proyectoConCola(t)
	primero := mustPedirEnOrden(t, root, "exit 0 # publicar", Orden{})
	segundo := mustPedirEnOrden(t, root, "echo instalado", Orden{Despues: []string{"a"}})
	a, _ := BuscarAccion(Acciones(root, false), "pedido:"+segundo.ID)
	if a.Estado != EstadoEspera || a.Tecla != "b" {
		t.Fatalf("el que espera a [a] ya se puede confirmar, con su tecla: %+v", a)
	}
	estado, err := aceptar(root, segundo.ID, "", time.Now())
	if err != nil || estado != EstadoArmado {
		t.Fatalf("confirmarlo lo arma, no lo corre: %q %v", estado, err)
	}
	fin := correrAparte(root, segundo.ID)
	time.Sleep(100 * time.Millisecond)
	if b, _ := BuscarAccion(Acciones(root, false), "pedido:"+segundo.ID); b.Estado != EstadoArmado || len(b.Espera) != 1 || b.Espera[0] != "a" {
		t.Fatalf("mientras [a] no termina, sigue armado y dice qué espera: %+v", b)
	}
	marcarHecho(t, root, primero.ID)
	if codigo := <-fin; codigo != 0 {
		t.Fatalf("al terminar [a] corre solo: %d", codigo)
	}
	esperarEstado(t, root, segundo.ID, EstadoHecho)
}

func TestArmado_DejarDeEsperarLoVuelveAPendienteYElProcesoTermina(t *testing.T) {
	cadasCortos(t)
	root := proyectoConCola(t)
	primero := mustPedirEnOrden(t, root, "exit 0 # publicar", Orden{})
	segundo := mustPedirEnOrden(t, root, "echo instalado", Orden{Despues: []string{primero.ID}})
	if _, err := aceptar(root, segundo.ID, "", time.Now()); err != nil {
		t.Fatal(err)
	}
	fin := correrAparte(root, segundo.ID)
	for tope := time.Now().Add(5 * time.Second); esperarEstado(t, root, segundo.ID, EstadoArmado).PID != os.Getpid(); {
		if time.Now().After(tope) {
			t.Fatal("el proceso que espera no tomó el pedido")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := Desarmar(root, "b"); err != nil {
		t.Fatal(err)
	}
	if codigo := <-fin; codigo != 0 {
		t.Fatalf("el proceso que esperaba termina sin correr nada: %d", codigo)
	}
	p := esperarEstado(t, root, segundo.ID, EstadoPendiente)
	if p.PID != 0 || p.Proximo != nil || !strings.Contains(p.Aviso, "Dejaste de esperar") {
		t.Fatalf("vuelve a pendiente, con su tecla y diciendo por qué: %+v", p)
	}
	if err := Desarmar(root, "b"); err == nil {
		t.Fatal("uno que no está armado no se desarma")
	}
}

func TestArmado_PasadoElTopeVuelveAPendienteConElMotivo(t *testing.T) {
	cadasCortos(t)
	TopeArmado = 50 * time.Millisecond
	root := proyectoConCola(t)
	mustPedirEnOrden(t, root, "exit 0 # nunca", Orden{})
	segundo := mustPedirEnOrden(t, root, "echo instalado", Orden{Despues: []string{"a"}})
	if _, err := aceptar(root, segundo.ID, "", time.Now()); err != nil {
		t.Fatal(err)
	}
	if codigo := <-correrAparte(root, segundo.ID); codigo != 0 {
		t.Fatalf("termina sin correr: %d", codigo)
	}
	if p := esperarEstado(t, root, segundo.ID, EstadoPendiente); !strings.Contains(p.Aviso, "Dejó de esperar") {
		t.Fatalf("dice que dejó de esperar y por qué: %+v", p)
	}
}

func TestArmado_SiMuereElProcesoQueEsperabaVuelveAPendiente(t *testing.T) {
	root := proyectoConCola(t)
	p := mustPedir(t, root, "exit 0", "m", "")
	conCola(root, time.Now(), func(cola []Pedido) ([]Pedido, error) {
		ahora := time.Now()
		cola[0].Estado, cola[0].ArmadoEn, cola[0].Proximo, cola[0].PID = EstadoArmado, &ahora, &ahora, 999999
		return cola, nil
	})
	q := Pedidos(root)[0]
	if q.ID != p.ID || q.Estado != EstadoPendiente || q.PID != 0 || !strings.Contains(q.Aviso, "Dejó de esperar") {
		t.Fatalf("sin nadie que espere, el panel lo ofrece de nuevo y dice por qué: %+v", q)
	}
}

func TestArmado_YaArmadoNoSeAceptaDosVeces(t *testing.T) {
	root := proyectoConCola(t)
	mustPedirEnOrden(t, root, "exit 0 # primero", Orden{})
	segundo := mustPedirEnOrden(t, root, "exit 0 # segundo", Orden{Despues: []string{"a"}})
	if _, err := aceptar(root, segundo.ID, "", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := aceptar(root, segundo.ID, "", time.Now()); err == nil || !strings.Contains(err.Error(), "armado") {
		t.Fatalf("uno armado ya tiene tu intención (FND-070): %v", err)
	}
}
