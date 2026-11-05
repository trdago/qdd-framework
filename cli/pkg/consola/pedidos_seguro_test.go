package consola

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// FND-154 (pedido de la persona, 2026-11-05): tras una tanda en modo goal quedan avisos, y contestarlos no puede lanzar
// dos veces lo mismo. «Cerrar issue #112» corrió dos veces: 16:48 y 16:49, con dos textos distintos.

func correrPedido(t *testing.T, root, id string) {
	t.Helper()
	if err := marcarEjecutando(root, id, time.Now()); err != nil {
		t.Fatal(err)
	}
	var salida strings.Builder
	if codigo := CorrerLanzado(root, id, &salida); codigo != 0 {
		t.Fatalf("corre: %d %s", codigo, salida.String())
	}
}

func esYaHecho(t *testing.T, err error) *YaHecho {
	t.Helper()
	var ya *YaHecho
	if !errors.As(err, &ya) {
		t.Fatalf("dice que ya se hizo: %v", err)
	}
	return ya
}

func TestSeguro_LoQueYaCorrioNoSePideDeNuevo_FND154(t *testing.T) {
	root := proyectoConCola(t)
	p := mustPedir(t, root, "exit 0", "Publicar", "")
	correrPedido(t, root, p.ID)
	_, err := Pedir(root, "exit 0", "Publicar otra vez", "", time.Now())
	ya := esYaHecho(t, err)
	if ya.Motivo != "Publicar" || !strings.Contains(ya.Error(), "de_nuevo") {
		t.Fatalf("dice qué se hizo y cómo repetirlo: %+v %v", ya, ya)
	}
	if n := len(abiertos(Pedidos(root))); n != 0 {
		t.Fatalf("no queda otro aviso en el panel: %d", n)
	}
}

func TestSeguro_ElMismoEfectoConOtroTextoNoSeDuplica_FND154(t *testing.T) {
	root := proyectoConCola(t)
	primero := mustPedir(t, root, "gh issue close 112 -R a/b --comment 'en rc.6'", "Cerrar issue #112 en rc.6", "")
	segundo := mustPedir(t, root, "gh issue close 112 -R a/b --comment 'hecha'", "Cerrar issue #112 hecha", "")
	if segundo.ID != primero.ID || len(abiertos(Pedidos(root))) != 1 {
		t.Fatalf("cerrar el mismo issue es un solo pedido: %+v %+v", primero, segundo)
	}
}

func TestSeguro_ElMismoEfectoYaHechoConOtroTextoNoSePide_FND154(t *testing.T) {
	root := proyectoConCola(t)
	anotarHecho(root, Pedido{Comando: "gh pr merge 7 --merge", Motivo: "Mergear #7"}, time.Now())
	_, err := Pedir(root, "qdd consola mergear-pr 7", "Mergear #7 de nuevo", "", time.Now())
	esYaHecho(t, err)
}

func TestSeguro_RepetirloDeVerdadHayQueDecirlo_FND154(t *testing.T) {
	root := proyectoConCola(t)
	p := mustPedir(t, root, "exit 0", "Publicar", "")
	correrPedido(t, root, p.ID)
	otro, err := PedirEnOrden(root, "exit 0", "Publicar otra vez", "", Orden{DeNuevo: true}, time.Now())
	if err != nil || otro.ID == "" {
		t.Fatalf("con de_nuevo se pide: %+v %v", otro, err)
	}
}

func TestSeguro_ElRecuerdoVenceALas24Horas_FND154(t *testing.T) {
	root := proyectoConCola(t)
	p := mustPedir(t, root, "exit 0", "Publicar", "")
	correrPedido(t, root, p.ID)
	if _, err := Pedir(root, "exit 0", "Publicar mañana", "", time.Now().Add(VidaHechos+time.Minute)); err != nil {
		t.Fatalf("pasado el plazo se puede pedir: %v", err)
	}
}

func TestSeguro_UnPedidoQueFallaSePuedePedirDeNuevo_FND154(t *testing.T) {
	root := proyectoConCola(t)
	p := mustPedir(t, root, "exit 1", "Publicar", "")
	if err := marcarEjecutando(root, p.ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	CorrerLanzado(root, p.ID, &strings.Builder{})
	if _, err := Pedir(root, "exit 1", "Publicar", "", time.Now()); err != nil {
		t.Fatalf("lo que falló no se hizo: %v", err)
	}
}

func TestSeguro_EfectoSoloReconoceComandosSimples_FND154(t *testing.T) {
	iguales := [][2]string{
		{"gh issue close 5 -R a/b --comment 'uno'", "gh issue close 5 -R a/b --comment 'dos; tres && cuatro'"},
		{"gh pr merge 5 --merge", "qdd consola mergear-pr 5"},
		{"echo  hola", "echo hola"},
	}
	for _, c := range iguales {
		if efectoDe(c[0], "/w") != efectoDe(c[1], "/w") {
			t.Errorf("mismo efecto: %q y %q", c[0], c[1])
		}
	}
	distintos := [][2]string{
		{"gh issue close 5 -R a/b", "gh issue close 5 -R c/d"},
		{"gh issue close 5", "gh issue close 6"},
		{"gh pr merge 5 --merge", "gh pr merge 5 --merge && git tag v1 && git push origin v1"},
		{"gh issue close 5", "gh pr merge 5"},
	}
	for _, c := range distintos {
		if efectoDe(c[0], "/w") == efectoDe(c[1], "/w") {
			t.Errorf("efectos distintos: %q y %q", c[0], c[1])
		}
	}
	if efectoDe("make test", "/a") == efectoDe("make test", "/b") {
		t.Error("el mismo comando en otra carpeta es otra cosa")
	}
}
