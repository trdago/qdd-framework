package uso

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// Uso medido de cada versión de qdd (pedido del usuario, 2026-10-30): una versión que no es un fix se publica después
// de horas reales de uso, medidas, nunca estimadas.

var t0 = time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)

func TestRegistrar_CuentaSoloElTiempoEntreEventosSeguidos(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	for _, m := range []int{0, 2, 4, 30, 31} {
		if err := Registrar("v2.8.0-rc.1", "qdd", t0.Add(time.Duration(m)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	u, err := Leer("2.8.0-rc.1")
	if err != nil {
		t.Fatal(err)
	}
	if u.Segundos != 5*60 {
		t.Fatalf("0→2→4 son 4 min, el salto de 26 min no cuenta y 30→31 es 1 min: %d s", u.Segundos)
	}
	if u.Proyectos["qdd"] != 5*60 || u.Version != "2.8.0-rc.1" {
		t.Fatalf("por proyecto y con su versión: %+v", u)
	}
	if err := Registrar("v2.8.0-rc.1", "otro", t0.Add(32*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if u, _ := Leer("2.8.0-rc.1"); u.Proyectos["otro"] != 60 || u.Segundos != 6*60 {
		t.Fatalf("el tiempo va al proyecto donde ocurrió el evento: %+v", u)
	}
}

func TestRegistrar_NoInventaTiempo(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	Registrar("2.8.0", "qdd", t0)
	if u, _ := Leer("2.8.0"); u.Segundos != 0 {
		t.Fatalf("un evento aislado no suma: %d", u.Segundos)
	}
	Registrar("2.8.0", "qdd", t0.Add(-time.Minute))
	if u, _ := Leer("2.8.0"); u.Segundos != 0 || u.Ultimo != t0.Format(time.RFC3339) {
		t.Fatalf("un reloj que vuelve atrás no suma ni retrocede: %+v", u)
	}
	if err := Registrar("dev", "qdd", t0); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("QDD_HOME"), "uso", "dev.json")); !os.IsNotExist(err) {
		t.Fatal("una compilación sin versión no se mide")
	}
	if err := Registrar("../x", "qdd", t0); err == nil {
		t.Fatal("una versión que no es SemVer no se escribe")
	}
}

func TestTotal_JuntaLasCandidatasDeUnaVersion(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	for _, v := range []string{"2.8.0-rc.1", "2.8.0-rc.2", "2.9.0-rc.1"} {
		Registrar(v, "qdd", t0)
		Registrar(v, "qdd", t0.Add(3*time.Minute))
	}
	total, versiones := Total("v2.8.0")
	if total != 6*time.Minute || len(versiones) != 2 {
		t.Fatalf("2.8.0 suma sus dos candidatas y no la 2.9.0: %v %+v", total, versiones)
	}
}

func TestEvidencia_DejaLasHorasMedidasEnElProyecto(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	for m := 0; m <= 90; m += 3 {
		Registrar("2.8.0-rc.1", "qdd", t0.Add(time.Duration(m)*time.Minute))
	}
	ruta, ev, err := Evidencia(root, "v2.8.0", t0.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if ruta != filepath.Join(root, RutaEvidencia, "v2.8.0.yaml") || ev.Horas != 1.5 || ev.Version != "2.8.0" {
		t.Fatalf("evidencia: %s %+v", ruta, ev)
	}
	leida, err := LeerEvidencia(root, "2.8.0")
	if err != nil || leida.Horas != 1.5 || leida.Proyectos["qdd"] != 1.5 || !strings.HasPrefix(leida.Versiones[0], "2.8.0-rc.1") {
		t.Fatalf("se lee igual: %+v %v", leida, err)
	}
	if _, _, err := Evidencia(root, "2.10.0", t0); err == nil {
		t.Fatal("sin uso medido no hay evidencia que dejar")
	}
}
