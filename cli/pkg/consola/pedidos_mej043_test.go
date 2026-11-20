package consola

import (
	"runtime"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// MEJ-043: «qdd consola pedir -- git -c credential.helper="!gh auth git-credential" push» guardaba los argumentos
// unidos por espacios y la shell los volvía a partir (FND-212). Y descartar un pedido liberaba a los que lo esperaban:
// un despliegue podía correr antes del push que lo debía preceder (FND-213).

func TestComandoDeArgs_CitaLosArgumentosConEspacios_FND212(t *testing.T) {
	casos := []struct {
		args       []string
		posix, win string
	}{
		{[]string{"git", "push", "origin", "v2.5.1"}, "git push origin v2.5.1", "git push origin v2.5.1"},
		{[]string{"git", "-c", "credential.helper=!gh auth git-credential", "push"}, "git -c 'credential.helper=!gh auth git-credential' push", `git -c "credential.helper=!gh auth git-credential" push`},
		{[]string{"echo", "it's"}, `echo 'it'\''s'`, `echo "it's"`},
		{[]string{"echo", ""}, "echo ''", `echo ""`},
		{[]string{"echo", `di "hola"`}, `echo 'di "hola"'`, `echo "di \"hola\""`},
		// Un solo argumento es un texto de shell, como por la tool MCP: va tal cual.
		{[]string{"gh pr merge 14 && git push"}, "gh pr merge 14 && git push", "gh pr merge 14 && git push"},
	}
	for _, c := range casos {
		if got := ComandoDeArgs(c.args, "linux"); got != c.posix {
			t.Errorf("posix %q: got %q, want %q", c.args, got, c.posix)
		}
		if got := ComandoDeArgs(c.args, "windows"); got != c.win {
			t.Errorf("windows %q: got %q, want %q", c.args, got, c.win)
		}
	}
}

func TestPedido_CorreConSusArgumentosTalCual_FND212(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("la shell de esta prueba es sh")
	}
	root := proyectoConCola(t)
	p, err := PedirEnOrden(root, ComandoDeArgs([]string{"printf", "%s|", "credential.helper=!gh auth git-credential", "it's"}, runtime.GOOS), "argumentos", "", Orden{}, enCola)
	if err != nil {
		t.Fatal(err)
	}
	correrPedido(t, root, p.ID)
	for _, q := range PedidosA(root, time.Now()) {
		if q.ID == p.ID && !strings.Contains(q.Salida, "credential.helper=!gh auth git-credential|it's|") {
			t.Fatalf("cada argumento llega entero al comando: %q", q.Salida)
		}
	}
}

func TestDescartar_LoQueLoEsperabaQuedaBloqueado_FND213(t *testing.T) {
	root := proyectoConCola(t)
	push := mustPedirEnOrden(t, root, "exit 0 # push", Orden{})
	deploy := mustPedirEnOrden(t, root, "exit 0 # deploy", Orden{Despues: []string{push.ID}})
	if err := Descartar(root, push.Tecla); err != nil {
		t.Fatal(err)
	}
	a, err := BuscarAccion(Acciones(root, false), "pedido:"+deploy.ID)
	if err != nil {
		t.Fatal(err)
	}
	if a.Tecla != "" || a.Estado != EstadoEspera {
		t.Fatalf("el que esperaba al descartado no se puede confirmar: %+v", a)
	}
	if !strings.Contains(a.Bloqueo, "se descartó") || !strings.Contains(a.Bloqueo, "m exit 0 # push") {
		t.Fatalf("el panel dice por qué está bloqueado y qué se descartó: %q", a.Bloqueo)
	}
	if _, err := aceptar(root, deploy.ID, "", enCola); err == nil {
		t.Fatal("ni con «s  Sí» se arma: esperaría algo que ya no va a correr")
	}
	if err := marcarEjecutando(root, deploy.ID, enCola); err == nil {
		t.Fatal("ni se ejecuta directo")
	}
}

func TestDescartar_UnArmadoQueLoEsperabaVuelveAPendiente_FND213(t *testing.T) {
	root := proyectoConCola(t)
	push := mustPedirEnOrden(t, root, "exit 0 # push", Orden{})
	deploy := mustPedirEnOrden(t, root, "exit 0 # deploy", Orden{Despues: []string{push.ID}})
	if estado, err := aceptar(root, deploy.ID, "", enCola); err != nil || estado != EstadoArmado {
		t.Fatalf("confirmado mientras espera, queda armado: %q %v", estado, err)
	}
	if err := Descartar(root, push.ID); err != nil {
		t.Fatal(err)
	}
	for _, p := range Pedidos(root) {
		if p.ID == deploy.ID && (p.Estado != EstadoPendiente || !strings.Contains(p.Aviso, "se descartó")) {
			t.Fatalf("el armado deja de esperar y dice por qué: %+v", p)
		}
	}
}

func TestPedirDeNuevo_ElDescartadoReengancha_FND213(t *testing.T) {
	root := proyectoConCola(t)
	push := mustPedirEnOrden(t, root, "exit 0 # push", Orden{})
	deploy := mustPedirEnOrden(t, root, "exit 0 # deploy", Orden{Despues: []string{push.ID}})
	if err := Descartar(root, push.ID); err != nil {
		t.Fatal(err)
	}
	nuevo := mustPedirEnOrden(t, root, "exit 0 # push", Orden{})
	for _, p := range Pedidos(root) {
		if p.ID == deploy.ID && (len(p.Descartados) != 0 || len(p.Despues) != 1 || p.Despues[0] != nuevo.ID) {
			t.Fatalf("pedir de nuevo el mismo comando lo vuelve a esperar: %+v", p)
		}
	}
	a, _ := BuscarAccion(Acciones(root, false), "pedido:"+deploy.ID)
	if a.Bloqueo != "" || a.Estado != EstadoEspera || a.Tecla == "" {
		t.Fatalf("vuelve a esperar al nuevo y se puede confirmar para que corra después: %+v", a)
	}
}

func TestPedirDeNuevo_ElBloqueadoQuedaLibre_FND213(t *testing.T) {
	root := proyectoConCola(t)
	push := mustPedirEnOrden(t, root, "exit 0 # push", Orden{})
	deploy := mustPedirEnOrden(t, root, "exit 0 # deploy", Orden{Despues: []string{push.ID}})
	if err := Descartar(root, push.ID); err != nil {
		t.Fatal(err)
	}
	mustPedirEnOrden(t, root, "exit 0 # deploy", Orden{})
	a, _ := BuscarAccion(Acciones(root, false), "pedido:"+deploy.ID)
	if a.Bloqueo != "" || a.Tecla == "" || a.Estado == EstadoEspera {
		t.Fatalf("la IA lo pidió de nuevo sabiendo que el otro se descartó: se puede ejecutar: %+v", a)
	}
}

func TestHechoPorLaPersona_SigueLiberandoALosQueEsperaban(t *testing.T) {
	root := proyectoConCola(t)
	push := mustPedirEnOrden(t, root, "exit 0 # push", Orden{})
	deploy := mustPedirEnOrden(t, root, "exit 0 # deploy", Orden{Despues: []string{push.ID}})
	if err := HechoPorLaPersona(root, push.ID, "", enCola); err != nil {
		t.Fatal(err)
	}
	a, _ := BuscarAccion(Acciones(root, false), "pedido:"+deploy.ID)
	if a.Bloqueo != "" || a.Tecla == "" || a.Estado == EstadoEspera {
		t.Fatalf("lo que la persona hizo cumple la espera: %+v", a)
	}
}

// FND-242: un pedido corre aparte, sin la sesión de Claude Code en su entorno. Uno que usa el baúl (qdd ambiente
// levantar) falló con «sin sesión de Claude Code ni terminal no hay a quién pedirle permiso» (2026-11-20). El comando
// corre con la sesión que lo pidió: el permiso 🔑 llega al panel de esa sesión.
func TestPedido_CorreConLaSesionQueLoPidio_FND242(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("la shell de esta prueba es sh")
	}
	root := proyectoConCola(t)
	const sesion = "2a4886f1-7ef9-470c-85ff-48fdb864f404"
	t.Setenv("CLAUDE_CODE_SESSION_ID", sesion)
	p := mustPedirEnOrden(t, root, `printf 'sesion=%s|' "$CLAUDE_CODE_SESSION_ID"`, Orden{})
	t.Setenv("CLAUDE_CODE_SESSION_ID", "")
	correrPedido(t, root, p.ID)
	for _, q := range PedidosA(root, time.Now()) {
		if q.ID == p.ID && !strings.Contains(q.Salida, "sesion="+sesion+"|") {
			t.Fatalf("el comando no recibió la sesión que lo pidió: %q", q.Salida)
		}
	}
}
