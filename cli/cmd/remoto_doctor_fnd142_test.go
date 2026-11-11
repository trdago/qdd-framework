package cmd

import (
	"os"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/qdd-framework/qdd/pkg/remoto"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'

func leerTarea(t *testing.T, archivo string) string {
	t.Helper()
	b, err := os.ReadFile(archivo)
	if err != nil {
		t.Fatal(err)
	}
	u := make([]uint16, 0, len(b)/2)
	for i := 2; i+1 < len(b); i += 2 {
		u = append(u, uint16(b[i])|uint16(b[i+1])<<8)
	}
	return string(utf16.Decode(u))
}

func tareaEnOtraVersion(t *testing.T) (remoto.Servicio, string, *[][]string) {
	t.Helper()
	dir := t.TempDir()
	vieja := remoto.Servicio{SO: "windows", Bin: `C:\L\qdd\bin\2.10.0\qdd.exe`, Usuario: dir, DirQDD: dir}
	archivo, err := vieja.Instalar(func(string, ...string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	nueva := vieja
	nueva.Bin = `C:\L\qdd\bin\2.10.1\qdd.exe`
	antesServicio, antesEjecutor := remotoServicio, remotoEjecutor
	t.Cleanup(func() { remotoServicio, remotoEjecutor = antesServicio, antesEjecutor })
	remotoServicio = func() (remoto.Servicio, error) { return nueva, nil }
	corridos := &[][]string{}
	remotoEjecutor = func(n string, a ...string) error {
		*corridos = append(*corridos, append([]string{n}, a...))
		return nil
	}
	return nueva, archivo, corridos
}

// FND-142: en Windows cada versión de qdd vive en su carpeta: instalar otra mueve también la tarea de qdd remoto.
func TestActualizar_LaTareaSigueALaVersionInstalada_FND142(t *testing.T) {
	_, archivo, corridos := tareaEnOtraVersion(t)
	seguirVersionEnLaTarea()
	if !strings.Contains(leerTarea(t, archivo), `2.10.1\qdd.exe</Command>`) || len(*corridos) != 1 {
		t.Fatalf("la tarea lanza la versión nueva: %s %v", leerTarea(t, archivo), *corridos)
	}
}

func TestDoctor_VeYReparaLaTareaEnOtraVersion_FND142(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	if err := registro.Anotar(t.TempDir(), time.Now()); err != nil {
		t.Fatal(err)
	}
	s, archivo, corridos := tareaEnOtraVersion(t)
	g := buildRemotoChecks()
	var item *CheckItem
	for _, i := range g.Items {
		if i.Name == itemRemotoVersion {
			item = i
		}
	}
	if item == nil || item.Success {
		t.Fatalf("doctor lo ve: %+v", g.Items)
	}
	repairRemoto(g)
	if len(*corridos) != 1 || (*corridos)[0][0] != "schtasks" {
		t.Fatalf("solo vuelve a registrar la tarea: %v", *corridos)
	}
	if !strings.Contains(leerTarea(t, archivo), `2.10.1\qdd.exe</Command>`) || s.TareaSinEstaVersion() {
		t.Fatalf("quedó al día: %s", leerTarea(t, archivo))
	}
}

// FND-195 (MEJ-060): una tarea sin usuario no se pudo registrar sin administrador, y nada lo decía: doctor la ve y --fix
// la registra limitada al usuario.
func TestDoctor_VeYReparaLaTareaSinUsuario_FND195(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	if err := registro.Anotar(t.TempDir(), time.Now()); err != nil {
		t.Fatal(err)
	}
	s, archivo, corridos := tareaEnOtraVersion(t)
	s.Bin = `C:\L\qdd\bin\2.10.0\qdd.exe`
	s.Cuenta = `PC\ana`
	remotoServicio = func() (remoto.Servicio, error) { return s, nil }
	g := buildRemotoChecks()
	var item *CheckItem
	for _, i := range g.Items {
		if i.Name == itemRemotoCuenta {
			item = i
		}
	}
	if item == nil || item.Success || !strings.Contains(item.Error, "administrador") {
		t.Fatalf("doctor lo ve: %+v", g.Items)
	}
	repairRemoto(g)
	if len(*corridos) != 1 || (*corridos)[0][0] != "schtasks" || (*corridos)[0][1] != "/Create" {
		t.Fatalf("vuelve a registrar la tarea: %v", *corridos)
	}
	if !strings.Contains(leerTarea(t, archivo), `<UserId>PC\ana</UserId>`) || s.TareaSinCuenta() {
		t.Fatalf("quedó con el usuario: %s", leerTarea(t, archivo))
	}
}

// FND-197 (MEJ-058): REM-06 «arrancaba» una tarea escrita sin usuario: schtasks /Create la rechazaba sin administrador.
// Antes de arrancarla, doctor la pone al día.
func TestDoctor_ArrancarQDDRemotoPoneAlDiaLaTarea_FND197(t *testing.T) {
	s, archivo, corridos := tareaEnOtraVersion(t)
	s.Bin = `C:\L\qdd\bin\2.10.0\qdd.exe`
	s.Cuenta = `PC\ana`
	remotoServicio = func() (remoto.Servicio, error) { return s, nil }
	if err := arrancarServicioMovil(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(leerTarea(t, archivo), `<UserId>PC\ana</UserId>`) {
		t.Fatalf("la tarea quedó con el usuario antes de registrarla: %s", leerTarea(t, archivo))
	}
	ultimo := (*corridos)[len(*corridos)-1]
	if ultimo[0] != "schtasks" || ultimo[1] != "/Run" {
		t.Fatalf("y después la arranca: %v", *corridos)
	}
}
