package mejoras

import (
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// MEJ-087 (REP-503): un reporte del relay no se cierra con la palabra de quien lo arregla. Su tarea terminada lo deja
// «arreglada»; se cierra cuando llega, admitida por la persona, la verificación de quien lo reportó. Si sigue
// fallando, vuelve a la cola el mismo reporte con la evidencia nueva.

func reporteArreglado(t *testing.T) (string, Fuentes, *[]ReporteRelay) {
	t.Helper()
	root := proyecto(t)
	lote := &[]ReporteRelay{{ID: "REP-9", Tipo: "bug", Titulo: "Se cae", Detalle: "al abrir", Version: "v", SO: "s", Llego: "2026-11-03T14:00:00Z"}}
	f := sinGitHub()
	f.Relay = func(root, desde string) ([]ReporteRelay, error) {
		var nuevos []ReporteRelay
		for _, r := range *lote {
			if numeroRelay(r.ID) > numeroRelay(desde) {
				nuevos = append(nuevos, r)
			}
		}
		return nuevos, nil
	}
	ciclo(t, root, f, t0)
	if err := Admitir(root, "MEJ-001", t0); err != nil {
		t.Fatal(err)
	}
	ciclo(t, root, f, t0)
	if err := estado.Cerrar(root, mejora(t, root, "MEJ-001").Tarea, "arreglado en rc.96", false, t0.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	r := ciclo(t, root, f, t0.Add(2*time.Minute))
	if len(r.Hechas) != 0 {
		t.Fatalf("no cuenta como hecha: %+v", r)
	}
	return root, f, lote
}

func llega(t *testing.T, root string, f Fuentes, lote *[]ReporteRelay, titulo, detalle string) string {
	t.Helper()
	*lote = append(*lote, ReporteRelay{ID: "REP-10", Tipo: "mejora", Titulo: titulo, Detalle: detalle, Version: "v", SO: "s", Llego: "2026-11-03T15:00:00Z"})
	ciclo(t, root, f, t0.Add(3*time.Minute))
	return "MEJ-002"
}

func TestVerificacion_UnReporteDelRelayArregladoEsperaAQuienLoReporto_MEJ087(t *testing.T) {
	root, f, _ := reporteArreglado(t)
	m := mejora(t, root, "MEJ-001")
	if m.Estado != Arreglada || m.Motivo != "arreglado en rc.96" {
		t.Fatalf("su tarea terminada la deja arreglada, no hecha: %+v", m)
	}
	ciclo(t, root, f, t0.Add(5*time.Minute))
	if m := mejora(t, root, "MEJ-001"); m.Estado != Arreglada {
		t.Fatalf("sigue esperando la verificación: %+v", m)
	}
}

func TestVerificacion_LaVerificacionAdmitidaLaCierra_MEJ087(t *testing.T) {
	root, f, lote := reporteArreglado(t)
	v := llega(t, root, f, lote, "Verificado REP-9 en 2.10.0-rc.96", "abrí el panel 3 veces: no se cae")
	if m := mejora(t, root, "MEJ-001"); m.Estado != Arreglada {
		t.Fatalf("sin admitir no cierra nada: %+v", m)
	}
	if err := Admitir(root, v, t0); err != nil {
		t.Fatal(err)
	}
	m := mejora(t, root, "MEJ-001")
	if m.Estado != Hecha || !strings.Contains(m.Motivo, "verificado por quien lo reportó en 2.10.0-rc.96") || !strings.Contains(m.Motivo, "abrí el panel 3 veces") {
		t.Fatalf("verificada por quien reportó, con su evidencia: %+v", m)
	}
	if v := mejora(t, root, v); v.Estado != Hecha || !strings.Contains(v.Motivo, "MEJ-001") {
		t.Fatalf("la verificación no es otra mejora: %+v", v)
	}
}

func TestVerificacion_SiSigueFallandoVuelveElMismoReporte_MEJ087(t *testing.T) {
	root, f, lote := reporteArreglado(t)
	v := llega(t, root, f, lote, "Sigue fallando REP-9 en 2.10.0-rc.96", "se cae al abrir el segundo proyecto")
	if err := Admitir(root, v, t0); err != nil {
		t.Fatal(err)
	}
	m := mejora(t, root, "MEJ-001")
	if m.Estado != EnCola || m.Tarea != "" || !strings.Contains(m.Detalle, "al abrir") || !strings.Contains(m.Detalle, "Sigue fallando en 2.10.0-rc.96: se cae al abrir el segundo proyecto") {
		t.Fatalf("se reabre el mismo, con la evidencia nueva: %+v", m)
	}
	if len(mustLista(t, root)) != 2 {
		t.Fatal("no se abre uno duplicado")
	}
}

func TestVerificacion_DeUnReporteQueNoEstaNoSeAdmite_MEJ087(t *testing.T) {
	root, f, lote := reporteArreglado(t)
	*lote = append(*lote, ReporteRelay{ID: "REP-11", Tipo: "mejora", Titulo: "Verificado REP-77 en v", Detalle: "x", Version: "v", SO: "s", Llego: "2026-11-03T15:00:00Z"})
	ciclo(t, root, f, t0.Add(3*time.Minute))
	if err := Admitir(root, "MEJ-002", t0); err == nil || !strings.Contains(err.Error(), "REP-77") {
		t.Fatalf("dice qué reporte no encontró: %v", err)
	}
	if m := mejora(t, root, "MEJ-002"); m.Estado != PorAdmitir {
		t.Fatalf("queda esperando: %+v", m)
	}
}

func TestArreglar_UnArregloHechoFueraDelCicloTambienEspera_MEJ087(t *testing.T) {
	root := proyecto(t)
	var desdes []string
	ciclo(t, root, conRelay(&desdes, ReporteRelay{ID: "REP-9", Tipo: "bug", Titulo: "Se cae", Version: "v", SO: "s", Llego: "2026-11-03T14:00:00Z"}), t0)
	if err := Arreglar(root, "MEJ-001", "2.10.0-rc.91", "la sesión en un diálogo ya no recibe", t0); err == nil {
		t.Fatal("una por admitir no se arregla: primero la admite la persona")
	}
	Admitir(root, "MEJ-001", t0)
	if err := Arreglar(root, "MEJ-001", "2.10.0-rc.91", "la sesión en un diálogo ya no recibe", t0); err != nil {
		t.Fatal(err)
	}
	if m := mejora(t, root, "MEJ-001"); m.Estado != Arreglada || m.Motivo != "arreglada en 2.10.0-rc.91: la sesión en un diálogo ya no recibe" {
		t.Fatalf("queda arreglada con la versión: %+v", m)
	}
}

func TestVerificacionDe_LeeElTituloQueArmaQddReportar_MEJ087(t *testing.T) {
	casos := map[string]struct {
		rep   string
		sigue bool
		ok    bool
	}{
		"Verificado REP-500 en 2.10.0-rc.96":      {"REP-500", false, true},
		"Sigue fallando REP-12 en 2.10.0":         {"REP-12", true, true},
		"Verificado el panel":                     {"", false, false},
		"Se cae al verificar REP-500 en el panel": {"", false, false},
	}
	for titulo, c := range casos {
		rep, sigue, version, ok := VerificacionDe(titulo)
		if rep != c.rep || sigue != c.sigue || ok != c.ok || (ok && version == "") {
			t.Fatalf("%q: %q %v %q %v", titulo, rep, sigue, version, ok)
		}
	}
}
