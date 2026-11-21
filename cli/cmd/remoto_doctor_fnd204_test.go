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
// FND-204 (MEJ-064): tras actualizar, la tarea de Windows quedaba registrada y sin arrancar, y una tarea «qdd-remoto»
// registrada a mano seguía apuntando a una versión desinstalada. Doctor ve las dos cosas y --fix las arregla.

func itemDe(g *CheckGroup, nombre string) *CheckItem {
	for _, i := range g.Items {
		if i.Name == nombre {
			return i
		}
	}
	return nil
}

func escribirTarea(t *testing.T, archivo, xml string) {
	t.Helper()
	out := []byte{0xFF, 0xFE}
	for _, r := range utf16.Encode([]rune(xml)) {
		out = append(out, byte(r), byte(r>>8))
	}
	if err := os.WriteFile(archivo, out, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestActualizar_ArrancaLaTareaConLaVersionNueva_FND204(t *testing.T) {
	_, _, corridos := tareaEnOtraVersion(t)
	seguirVersionEnLaTarea()
	if len(*corridos) != 2 || strings.Join((*corridos)[1], " ") != "schtasks /Run /TN QDD remoto" {
		t.Fatalf("al actualizar, qdd remoto vuelve a correr ya, no en el próximo inicio de sesión: %v", *corridos)
	}
}

func TestDoctor_VeYReparaLaTareaSinReintento_FND204(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	if err := registro.Anotar(t.TempDir(), time.Now()); err != nil {
		t.Fatal(err)
	}
	s, archivo, corridos := tareaEnOtraVersion(t)
	s.Bin = `C:\L\qdd\bin\2.10.0\qdd.exe`
	remotoServicio = func() (remoto.Servicio, error) { return s, nil }
	xml := leerTarea(t, archivo)
	inicio, fin := strings.Index(xml, "<TimeTrigger>"), strings.Index(xml, "</TimeTrigger>")
	escribirTarea(t, archivo, xml[:inicio]+xml[fin+len("</TimeTrigger>"):])
	g := buildRemotoChecks()
	item := itemDe(g, itemRemotoReintento)
	if item == nil || item.Success || !strings.Contains(item.Error, "inicio de sesión") {
		t.Fatalf("doctor ve que si se cae no vuelve: %+v", g.Items)
	}
	repairRemoto(g)
	if len(*corridos) != 2 || (*corridos)[1][1] != "/Run" || s.TareaSinReintento() {
		t.Fatalf("--fix la registra con el reintento y la arranca: %v", *corridos)
	}
}

func TestDoctor_VeYQuitaLaTareaSobrante_FND204(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	if err := registro.Anotar(t.TempDir(), time.Now()); err != nil {
		t.Fatal(err)
	}
	s, _, corridos := tareaEnOtraVersion(t)
	s.Bin = `C:\L\qdd\bin\2.10.0\qdd.exe`
	remotoServicio = func() (remoto.Servicio, error) { return s, nil }
	remotoConsultar = func(string, ...string) (string, error) {
		return `<Command>C:\L\qdd\bin\2.10.0-rc.59\qdd.exe</Command><Arguments>remoto correr</Arguments>`, nil
	}
	g := buildRemotoChecks()
	item := itemDe(g, itemRemotoSobrante)
	if item == nil || item.Success || !strings.Contains(item.Error, "qdd-remoto") {
		t.Fatalf("doctor nombra la tarea que sobra: %+v", g.Items)
	}
	repairRemoto(g)
	if len(*corridos) != 2 || strings.Join((*corridos)[1], " ") != "schtasks /Delete /TN qdd-remoto /F" {
		t.Fatalf("--fix la apaga y la borra, y no toca la de qdd: %v", *corridos)
	}
}

func TestDoctor_SinTareaSobranteNoDiceNada_FND204(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	if err := registro.Anotar(t.TempDir(), time.Now()); err != nil {
		t.Fatal(err)
	}
	tareaEnOtraVersion(t)
	if item := itemDe(buildRemotoChecks(), itemRemotoSobrante); item != nil {
		t.Fatalf("sin tarea vieja no hay nada que decir: %+v", item)
	}
}

func TestRemoto_UnPCQuitadoDesactivaLaTarea_FND204(t *testing.T) {
	s, _, corridos := tareaEnOtraVersion(t)
	remotoServicio = func() (remoto.Servicio, error) { return s, nil }
	pausarAlQuitarse()
	if len(*corridos) != 1 || strings.Join((*corridos)[0], " ") != "schtasks /Change /TN QDD remoto /DISABLE" {
		t.Fatalf("la tarea se desactiva para no relanzar cada 5 minutos un PC que el teléfono quitó: %v", *corridos)
	}
}
