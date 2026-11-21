package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/remoto"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'

func TestDoctor_RenuevaElServicioQueMataLoQueLanzo_FND128(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	if err := registro.Anotar(t.TempDir(), time.Now()); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	s := remoto.Servicio{SO: "linux", Bin: "/opt/qdd", Usuario: dir, DirQDD: dir}
	antesServicio, antesEjecutor := remotoServicio, remotoEjecutor
	t.Cleanup(func() { remotoServicio, remotoEjecutor = antesServicio, antesEjecutor })
	remotoServicio = func() (remoto.Servicio, error) { return s, nil }
	var corridos []string
	remotoEjecutor = func(n string, a ...string) error {
		corridos = append(corridos, n)
		return nil
	}
	archivo := filepath.Join(dir, ".config", "systemd", "user", "qdd-remoto.service")
	if err := os.MkdirAll(filepath.Dir(archivo), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archivo, []byte("[Service]\nExecStart=/opt/qdd remoto correr\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	g := buildRemotoChecks()
	var item *CheckItem
	for _, i := range g.Items {
		if i.Name == itemRemotoServicio {
			item = i
		}
	}
	if item == nil || item.Success {
		t.Fatalf("doctor lo ve: %+v", g.Items)
	}
	repairRemoto(g)
	if len(corridos) != 1 {
		t.Fatalf("solo recarga la configuración: %v", corridos)
	}
	b, err := os.ReadFile(archivo)
	if err != nil || !strings.Contains(string(b), "ExecStart=/opt/qdd remoto correr") {
		t.Fatalf("conserva el binario: %s %v", b, err)
	}
	if _, deja, err := s.DejaVivoLoLanzado(); err != nil || !deja {
		t.Fatalf("quedó al día: %v %v", deja, err)
	}
}

// Una corrida de las pruebas con doctor --fix reescribió el servicio real de la máquina con el binario de prueba
// (2026-11-04). El TestMain de este paquete aísla de la carpeta del usuario y de systemctl a lo que corre en este
// proceso; lo que corre en otro lo aísla binarioDePrueba.
func TestLasPruebasNoTocanElServicioReal(t *testing.T) {
	real, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	usuario, err := remotoUsuario()
	if err != nil || usuario == real {
		t.Fatalf("las pruebas usan otra carpeta de usuario: %q %v", usuario, err)
	}
	if err := remotoEjecutor("false"); err != nil {
		t.Fatalf("las pruebas no corren comandos del sistema: %v", err)
	}
}
