package remoto

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// Sprint 9 (FND-238): en Windows la tarea «QDD remoto» lanza qddw.exe, que qdd.exe lleva dentro y deja a su lado: corre
// qdd remoto sin ventana. Una tarea anterior (que lanza qdd.exe) pasa a qddw.exe con RenovarTarea, como al cambiar de
// versión (FND-142), y un qddw.exe que falta se vuelve a dejar.

const lanzadorDePrueba = "MZ lanzador de prueba"

func conLanzador(t *testing.T) {
	t.Helper()
	antes := LanzadorEmbebido
	LanzadorEmbebido = []byte(lanzadorDePrueba)
	t.Cleanup(func() { LanzadorEmbebido = antes })
}

func servicioConLanzador(t *testing.T) (Servicio, string, *[][]string) {
	t.Helper()
	conLanzador(t)
	dir := t.TempDir()
	s := Servicio{SO: "windows", Bin: `C:\L\qdd\bin\2.10.1\qdd.exe`, Usuario: dir, DirQDD: dir, Lanzador: filepath.Join(dir, "bin", NombreLanzador)}
	corridos := &[][]string{}
	return s, s.Lanzador, corridos
}

func anotar(corridos *[][]string) Ejecutor {
	return func(n string, a ...string) error {
		*corridos = append(*corridos, append([]string{n}, a...))
		return nil
	}
}

func tareaDe(t *testing.T, s Servicio) string {
	t.Helper()
	xml, ok := s.tareaInstalada()
	if !ok {
		t.Fatal("no hay tarea")
	}
	return xml
}

func TestLanzador_ActivarDejaQddwYLaTareaLoLanza_FND238(t *testing.T) {
	s, lanzador, corridos := servicioConLanzador(t)
	if _, err := s.Instalar(anotar(corridos)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tareaDe(t, s), "<Command>"+lanzador+"</Command><Arguments>remoto correr</Arguments>") {
		t.Fatalf("la tarea lanza qddw.exe: %s", tareaDe(t, s))
	}
	if b, _ := os.ReadFile(lanzador); string(b) != lanzadorDePrueba {
		t.Fatalf("qddw.exe queda junto a qdd.exe: %q", b)
	}
	if s.TareaConVentana() || s.LanzadorFalta() || s.TareaSinEstaVersion() {
		t.Fatal("al día: nada que reparar")
	}
}

func TestLanzador_LaTareaAnteriorPasaAQddwAlRenovar_FND238(t *testing.T) {
	s, lanzador, corridos := servicioConLanzador(t)
	anterior := s
	anterior.Lanzador = ""
	if _, err := anterior.Instalar(anotar(&[][]string{})); err != nil {
		t.Fatal(err)
	}
	if !s.TareaConVentana() || s.TareaSinEstaVersion() {
		t.Fatalf("la tarea de esta versión abre una ventana: no es otra versión (%v, %v)", s.TareaConVentana(), s.TareaSinEstaVersion())
	}
	if err := s.RenovarTarea(anotar(corridos)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tareaDe(t, s), "<Command>"+lanzador+"</Command>") || s.TareaConVentana() {
		t.Fatalf("pasa a qddw.exe: %s", tareaDe(t, s))
	}
	if b, _ := os.ReadFile(lanzador); string(b) != lanzadorDePrueba {
		t.Fatal("y lo deja junto a qdd.exe")
	}
	if len(*corridos) != 2 || (*corridos)[0][1] != "/Create" || (*corridos)[1][1] != "/Run" {
		t.Fatalf("la registra de nuevo y la arranca: %v", *corridos)
	}
}

func TestLanzador_SiFaltaQddwSeVuelveADejar_FND238(t *testing.T) {
	s, lanzador, corridos := servicioConLanzador(t)
	if _, err := s.Instalar(anotar(&[][]string{})); err != nil {
		t.Fatal(err)
	}
	os.Remove(lanzador)
	if !s.LanzadorFalta() {
		t.Fatal("sin qddw.exe la tarea no arrancaría: se ve")
	}
	if err := s.RenovarTarea(anotar(corridos)); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(lanzador); string(b) != lanzadorDePrueba || s.LanzadorFalta() {
		t.Fatal("se vuelve a dejar desde el qdd.exe instalado")
	}
}

func TestLanzador_UnoIgualNoSeReescribe_FND238(t *testing.T) {
	s, lanzador, _ := servicioConLanzador(t)
	os.MkdirAll(filepath.Dir(lanzador), 0o755)
	os.WriteFile(lanzador, []byte(lanzadorDePrueba), 0o755)
	antes := time.Now().Add(-time.Hour).Truncate(time.Second)
	os.Chtimes(lanzador, antes, antes)
	if _, err := s.Instalar(anotar(&[][]string{})); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(lanzador); !info.ModTime().Equal(antes) {
		t.Fatal("un qddw.exe igual (quizás corriendo) no se toca")
	}
}

func TestLanzador_SinLanzadorEnLaCompilacionLaTareaQuedaComoAntes_FND238(t *testing.T) {
	antes := LanzadorEmbebido
	LanzadorEmbebido = nil
	t.Cleanup(func() { LanzadorEmbebido = antes })
	if RutaLanzador(`/x/bin/2.10.1/qdd.exe`) != "" {
		t.Fatal("una compilación sin lanzador no lo ofrece")
	}
	dir := t.TempDir()
	s := Servicio{SO: "windows", Bin: `C:\L\qdd\bin\2.10.1\qdd.exe`, Usuario: dir, DirQDD: dir}
	if _, err := s.Instalar(anotar(&[][]string{})); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tareaDe(t, s), `<Command>C:\L\qdd\bin\2.10.1\qdd.exe</Command>`) || s.TareaConVentana() || s.LanzadorFalta() {
		t.Fatal("lanza qdd.exe como antes y no avisa nada")
	}
}

func TestLanzador_VaJuntoAQDD_FND238(t *testing.T) {
	conLanzador(t)
	if got := RutaLanzador(filepath.Join("/x", "bin", "2.10.1", "qdd.exe")); got != filepath.Join("/x", "bin", "2.10.1", NombreLanzador) {
		t.Fatalf("en la carpeta de su versión: %q", got)
	}
}

func TestLanzador_LinuxYMacNoCambian_FND238(t *testing.T) {
	conLanzador(t)
	for _, so := range []string{"linux", "darwin"} {
		s := Servicio{SO: so, Bin: "/usr/bin/qdd", Usuario: t.TempDir(), Lanzador: "/usr/bin/qddw.exe"}
		b, err := s.Contenido()
		if err != nil || strings.Contains(string(b), NombreLanzador) || !strings.Contains(string(b), "/usr/bin/qdd") {
			t.Fatalf("%s sigue lanzando qdd remoto correr: %s %v", so, b, err)
		}
	}
}
