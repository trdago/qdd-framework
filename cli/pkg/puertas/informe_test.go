package puertas

import (
	"math"
	"testing"
	"time"
)

func TestResumir_PorPuerta(t *testing.T) {
	t0 := time.Date(2026, 11, 20, 21, 0, 0, 0, time.UTC)
	en := func(min int) time.Time { return t0.Add(time.Duration(min) * time.Minute) }
	regs := []Registro{
		{ID: "a1", Fecha: en(0), Puerta: PuertaAudit, Resultado: Rechaza, MS: 20, Sesion: "s1", Largo: 120},
		{ID: "a2", Fecha: en(1), Puerta: PuertaAudit, Resultado: Pasa, MS: 10, Sesion: "s1"},
		{ID: "a3", Fecha: en(2), Puerta: PuertaAudit, Resultado: Rechaza, MS: 30, Sesion: "s2", Largo: 80},
		{ID: "a4", Fecha: en(3), Puerta: PuertaAudit, Resultado: Rechaza, MS: 40, Sesion: "s2", Largo: 80},
		{ID: "a5", Fecha: en(4), Puerta: PuertaAudit, Resultado: Error, MS: 5, Sesion: "s2"},
		{ID: "p1", Fecha: en(5), Puerta: PuertaPublicar, Resultado: Rechaza, MS: 2, Sesion: "s1", Largo: 160},
	}
	filas := Resumir(regs, map[string]string{"a4": "era un archivo generado"})
	if len(filas) != 2 || filas[0].Puerta != PuertaAudit || filas[1].Puerta != PuertaPublicar {
		t.Fatalf("una fila por puerta, en orden: %+v", filas)
	}
	a := filas[0]
	if a.Decisiones != 5 || a.Rechazos != 3 || a.Errores != 1 || a.Falsos != 1 {
		t.Errorf("conteos: %+v", a)
	}
	// a1 se corrigió (a2 pasa), a3 no (a4 vuelve a rechazar), a4 tuvo reintento que no pasó (a5 error).
	if a.ConReintento != 3 || a.Corregidos != 1 {
		t.Errorf("reintentos: %+v", a)
	}
	if a.MSMax != 40 || a.MSPromedio != 21 {
		t.Errorf("ms: %+v", a)
	}
	if math.Abs(a.TokensPorRechazo-(120+80+80)/4.0/3) > 0.01 {
		t.Errorf("tokens por rechazo = caracteres/4 promedio: %v", a.TokensPorRechazo)
	}
	if p := filas[1]; p.ConReintento != 0 || p.Rechazos != 1 {
		t.Errorf("sin reintento no cuenta como corregido ni como fallado: %+v", p)
	}
}

func TestLeerRegistro_ConMarcasDeFalso(t *testing.T) {
	home := t.TempDir()
	for _, r := range []Registro{{Puerta: PuertaAudit, Resultado: Rechaza, Sesion: "s1", Fecha: time.Unix(1, 0)}, {Puerta: PuertaFicha, Resultado: Pasa, Fecha: time.Unix(2, 0)}} {
		if err := Anotar(home, "qdd", r); err != nil {
			t.Fatal(err)
		}
	}
	regs, _, err := LeerRegistro(home, "qdd")
	if err != nil || len(regs) != 2 || regs[0].ID == "" || regs[0].ID == regs[1].ID {
		t.Fatalf("cada decisión con su id: %+v %v", regs, err)
	}
	if err := MarcarFalso(home, "qdd", regs[0].ID, "no era"); err != nil {
		t.Fatal(err)
	}
	regs, falsos, _ := LeerRegistro(home, "qdd")
	if len(regs) != 2 || falsos[regs[0].ID] != "no era" {
		t.Fatalf("la marca no es una decisión y queda con su motivo: %d %v", len(regs), falsos)
	}
	if _, ok := UltimoRechazo(regs, "s1"); !ok {
		t.Fatal("el último rechazo de la sesión se encuentra")
	}
	if _, ok := UltimoRechazo(regs, "s9"); ok {
		t.Fatal("otra sesión no tiene rechazos")
	}
}
