package estado

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// Pedido de la persona (2026-11-08): «cuando son preguntas que puedes contestarte tú mismo, deja programada una tarea que
// pregunte cada x tiempo si eso se cumplió o no, y deja una señal de que se autocontestará; ojalá de forma determinista».
// Un ⏳ con verificación tipada (npm:, pr:, tag:, pedido:, archivo:) se revisa cada CadaVerificacion y pasa solo a ✅.

var tv = time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)

func tablero(t *testing.T) string {
	t.Helper()
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".qdd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Init(root, []string{"Código", "Publicación"}, tv); err != nil {
		t.Fatal(err)
	}
	if err := Set(root, Cambio{Frente: "rel", Nombre: "Release", Etapa: "Código", Estado: "ok", Evidencia: "PR #1"}, tv); err != nil {
		t.Fatal(err)
	}
	return root
}

func esperaVerificada(t *testing.T, root, verifica string) {
	t.Helper()
	c := Cambio{Frente: "rel", Etapa: "Publicación", Estado: "espera", Bloqueo: "¿Ya está en npm?", DeQuien: "persona", Verifica: verifica}
	if err := Set(root, c, tv); err != nil {
		t.Fatal(err)
	}
}

func celdaDe(t *testing.T, root string) Celda {
	t.Helper()
	e, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return e.Frentes[e.buscar("rel")].Celdas["Publicación"]
}

type llamadas struct{ args []string }

func verificadores(l *llamadas, cumplido bool, err error) Verificadores {
	return Verificadores{"npm": func(root, arg string) (bool, string, error) {
		l.args = append(l.args, arg)
		return cumplido, "npm tiene " + arg, err
	}}
}

func TestSet_VerificaSoloEnUnaEsperaYConTipoConocido(t *testing.T) {
	root := tablero(t)
	if err := Set(root, Cambio{Frente: "rel", Etapa: "Publicación", Estado: "ok", Verifica: "npm:qdd@1"}, tv); err == nil {
		t.Fatal("solo un ⏳ se responde solo")
	}
	for _, v := range []string{"rm -rf /", "npm:", "curl:http://x", "npm"} {
		if err := Set(root, Cambio{Frente: "rel", Etapa: "Publicación", Estado: "espera", Bloqueo: "x", DeQuien: "qdd", Verifica: v}, tv); err == nil {
			t.Fatalf("%q no es una verificación tipada: nunca se corre un comando suelto del estado", v)
		}
	}
	esperaVerificada(t, root, "npm:qdd-framework@2.10.0-rc.43")
	c := celdaDe(t, root)
	if c.Verifica != "npm:qdd-framework@2.10.0-rc.43" || !strings.Contains(c.Bloqueo, SenalVerifica) {
		t.Fatalf("queda la verificación y la señal de que se responde sola: %+v", c)
	}
}

func TestVerificar_CumplidaPasaAOkConEvidencia(t *testing.T) {
	root := tablero(t)
	esperaVerificada(t, root, "npm:qdd-framework@2.10.0-rc.43")
	var l llamadas
	r, err := Verificar(root, verificadores(&l, true, nil), false, tv)
	if err != nil || len(r.Cumplidas) != 1 {
		t.Fatalf("debió cumplirse: %+v %v", r, err)
	}
	c := celdaDe(t, root)
	if NormalizarEstado(c.Estado) != Hecho || !strings.Contains(c.Evidencia, "npm tiene qdd-framework@2.10.0-rc.43") || c.Verifica != "" {
		t.Fatalf("la celda pasa a ✅ con la evidencia de la verificación: %+v", c)
	}
}

func TestVerificar_TodaviaNoEsperaSuTurno(t *testing.T) {
	root := tablero(t)
	esperaVerificada(t, root, "npm:qdd@9")
	var l llamadas
	if _, err := Verificar(root, verificadores(&l, false, nil), true, tv); err != nil {
		t.Fatal(err)
	}
	if _, err := Verificar(root, verificadores(&l, false, nil), true, tv.Add(CadaVerificacion-time.Second)); err != nil {
		t.Fatal(err)
	}
	if len(l.args) != 1 {
		t.Fatalf("con --si-toca se revisa una vez cada %v: %v", CadaVerificacion, l.args)
	}
	if _, err := Verificar(root, verificadores(&l, false, nil), true, tv.Add(CadaVerificacion)); err != nil {
		t.Fatal(err)
	}
	if len(l.args) != 2 || NormalizarEstado(celdaDe(t, root).Estado) != Espera {
		t.Fatalf("pasado el plazo se revisa de nuevo y sigue ⏳: %v", l.args)
	}
}

func TestVerificar_UnErrorSeAvisaYNoCierra(t *testing.T) {
	root := tablero(t)
	esperaVerificada(t, root, "npm:qdd@9")
	var l llamadas
	r, err := Verificar(root, verificadores(&l, false, errors.New("sin red")), false, tv)
	if err != nil || len(r.Avisos) != 1 || !strings.Contains(r.Avisos[0], "sin red") {
		t.Fatalf("la falla de una verificación se avisa: %+v %v", r, err)
	}
	if NormalizarEstado(celdaDe(t, root).Estado) != Espera {
		t.Fatal("un error nunca da por cumplida la espera")
	}
}

func TestVerificar_SinVerificadorDelTipoSeAvisa(t *testing.T) {
	root := tablero(t)
	esperaVerificada(t, root, "tag:v1")
	r, err := Verificar(root, Verificadores{}, false, tv)
	if err != nil || len(r.Avisos) != 1 {
		t.Fatalf("un tipo sin verificador en esta máquina se avisa: %+v %v", r, err)
	}
}

// doctor (VRF-01, VRF-02): una verificación que falla o que este qdd no entiende se ve, sin cerrar la espera.
func TestRevisarVerificaciones_DiceCualesFallanYCualesNoSeEntienden(t *testing.T) {
	root := tablero(t)
	if r := RevisarVerificaciones(root); r.Hay {
		t.Fatalf("sin esperas verificadas no hay nada que revisar: %+v", r)
	}
	esperaVerificada(t, root, "npm:qdd@9")
	var l llamadas
	if _, err := Verificar(root, verificadores(&l, false, errors.New("sin red")), false, tv); err != nil {
		t.Fatal(err)
	}
	r := RevisarVerificaciones(root)
	if !r.Hay || len(r.Fallan) != 1 || !strings.Contains(r.Fallan[0], "sin red") || len(r.Invalidas) != 0 {
		t.Fatalf("la que falló se nombra con su error: %+v", r)
	}
	e, _ := Load(root)
	c := e.Frentes[e.buscar("rel")].Celdas["Publicación"]
	c.Verifica = "curl:http://x"
	e.Frentes[e.buscar("rel")].Celdas["Publicación"] = c
	if err := Save(root, e); err != nil {
		t.Fatal(err)
	}
	if r := RevisarVerificaciones(root); len(r.Invalidas) != 1 {
		t.Fatalf("una editada a mano con un tipo que no existe se nombra: %+v", r)
	}
}
