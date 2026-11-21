package puertas

import (
	"strings"
	"testing"
)

func TestAcciones_UnCommitTraeSuMensajeParaLaPuertaFinding(t *testing.T) {
	casos := map[string]string{
		"git commit -m 'fix(baul): x' -m 'FND-241'":                              "fix(baul): x\n\nFND-241",
		"git commit -qm \"$(cat <<'EOF'\nfix: algo\n\nFND-7 lo cuenta\nEOF\n)\"": "fix: algo\n\nFND-7 lo cuenta",
		"git commit --message=feat:x":                                            "feat:x",
		"git commit -F tmp/msg.txt":                                              "",
	}
	for comando, want := range casos {
		got := Acciones(comando)
		if len(got) != 2 || got[1].Puerta != PuertaFinding {
			t.Fatalf("%q: un commit da audit y finding: %+v", comando, got)
		}
		if got[1].Mensaje != want {
			t.Errorf("%q: mensaje %q, want %q", comando, got[1].Mensaje, want)
		}
	}
	if a := Acciones("git commit -F tmp/msg.txt")[1]; a.ArchivoMensaje != "tmp/msg.txt" {
		t.Errorf("con -F el mensaje lo lee quien llama: %+v", a)
	}
}

func TestDecidir_Finding(t *testing.T) {
	activa := Politica{Activa: true}
	hay := map[string]bool{"FND-241": true, "FND-9": false}
	casos := []struct {
		mensaje   string
		resultado string
		dice      string
	}{
		{"feat: algo nuevo", NoAplica, ""},
		{"fix(baul): un 400 es rechazo (FND-241)\n\nCo-Authored-By: x", Pasa, ""},
		{"fix: arregla algo", Rechaza, "nombra su finding"},
		{"fix!: rompe algo sin finding", Rechaza, "nombra su finding"},
		{"fix(x): algo (FND-9)", Rechaza, "FND-9"},
		{"fix(x): algo (FND-404)", Rechaza, "FND-404"},
	}
	for _, c := range casos {
		d := Decidir(Hechos{Accion: Accion{Puerta: PuertaFinding}, Politica: activa, Mensaje: c.mensaje, Findings: hay})
		if d.Resultado != c.resultado || !strings.Contains(d.Mensaje, c.dice) {
			t.Errorf("%q: %+v", c.mensaje, d)
		}
		if len([]rune(d.Mensaje)) > 280 {
			t.Errorf("%q: más de 280 caracteres", c.mensaje)
		}
	}
}

func TestDecidir_Ficha(t *testing.T) {
	activa := Politica{Activa: true}
	editar := Accion{Puerta: PuertaFicha, Archivo: "cli/pkg/baul/verificar.go"}
	if d := Decidir(Hechos{Accion: editar, Politica: activa}); d.Resultado != NoAplica {
		t.Errorf("una zona sin ficha no bloquea: %+v", d)
	}
	sin := Decidir(Hechos{Accion: editar, Politica: activa, Ficha: "baul"})
	if sin.Resultado != Rechaza || !strings.Contains(sin.Mensaje, "qdd modules context cli/pkg/baul/verificar.go") || !strings.Contains(sin.Mensaje, "«baul»") {
		t.Errorf("sin leer la ficha se rechaza con el comando exacto: %+v", sin)
	}
	if d := Decidir(Hechos{Accion: editar, Politica: activa, Ficha: "baul", Leida: true}); d.Resultado != Pasa {
		t.Errorf("leída la ficha, pasa: %+v", d)
	}
	largo := Accion{Puerta: PuertaFicha, Archivo: strings.Repeat("carpeta/", 40) + "x.go"}
	if d := Decidir(Hechos{Accion: largo, Politica: activa, Ficha: "f"}); len([]rune(d.Mensaje)) > 280 {
		t.Errorf("una ruta larga no pasa de 280: %d", len([]rune(d.Mensaje)))
	}
}
