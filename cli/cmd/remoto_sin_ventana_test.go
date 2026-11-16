package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/remoto"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// Sprint 9 (FND-238): en Windows la tarea de QDD Móvil abría una ventana de consola. doctor ve una tarea que todavía
// lanza qdd.exe (o a la que le falta qddw.exe) y --fix la pasa al lanzador sin ventana; sin consola, qdd remoto escribe
// su registro en <QDD_HOME>/remoto/correr.log.

func tareaConVentana(t *testing.T) (remoto.Servicio, string, *[][]string) {
	t.Helper()
	t.Setenv("QDD_HOME", t.TempDir())
	if err := registro.Anotar(t.TempDir(), time.Now()); err != nil {
		t.Fatal(err)
	}
	antesLanzador := remoto.LanzadorEmbebido
	remoto.LanzadorEmbebido = []byte("MZ lanzador de prueba")
	dir := t.TempDir()
	anterior := remoto.Servicio{SO: "windows", Bin: `C:\L\qdd\bin\2.10.1\qdd.exe`, Usuario: dir, DirQDD: dir}
	archivo, err := anterior.Instalar(func(string, ...string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	s := anterior
	s.Lanzador = filepath.Join(dir, "bin", remoto.NombreLanzador)
	antesServicio, antesEjecutor, antesConsultar := remotoServicio, remotoEjecutor, remotoConsultar
	t.Cleanup(func() {
		remotoServicio, remotoEjecutor, remotoConsultar = antesServicio, antesEjecutor, antesConsultar
		remoto.LanzadorEmbebido = antesLanzador
	})
	remotoConsultar = func(string, ...string) (string, error) { return "", errors.New("no existe") }
	remotoServicio = func() (remoto.Servicio, error) { return s, nil }
	corridos := &[][]string{}
	remotoEjecutor = func(n string, a ...string) error {
		*corridos = append(*corridos, append([]string{n}, a...))
		return nil
	}
	return s, archivo, corridos
}

func TestDoctor_VeYReparaLaTareaQueAbreUnaVentana_FND238(t *testing.T) {
	s, archivo, corridos := tareaConVentana(t)
	g := buildRemotoChecks()
	item := itemDe(g, itemRemotoVentana)
	if item == nil || item.Success || !strings.Contains(item.Error, "ventana") || !strings.Contains(item.Error, "qdd doctor --fix") {
		t.Fatalf("doctor dice que abre una ventana y cómo se arregla: %+v", item)
	}
	if v := itemDe(g, itemRemotoVersion); v != nil {
		t.Fatalf("es la versión instalada: no se confunde con otra versión: %+v", v)
	}
	repairRemoto(g)
	if !strings.Contains(leerTarea(t, archivo), "<Command>"+s.Lanzador+"</Command>") || s.TareaConVentana() {
		t.Fatalf("la tarea pasa a qddw.exe: %s", leerTarea(t, archivo))
	}
	if len(*corridos) != 2 || (*corridos)[1][1] != "/Run" {
		t.Fatalf("la registra y la arranca: %v", *corridos)
	}
	var bueno *CheckItem
	for _, i := range buildRemotoChecks().Items {
		if strings.HasPrefix(i.Name, itemRemotoVentana) {
			bueno = i
		}
	}
	if bueno == nil || !bueno.Success || !strings.Contains(bueno.Name, "correr.log") {
		t.Fatalf("al día, doctor lo da por bueno y dice dónde está el registro: %+v", bueno)
	}
}

func TestDoctor_VeYReparaQddwQueFalta_FND238(t *testing.T) {
	s, _, _ := tareaConVentana(t)
	if err := s.RenovarTarea(func(string, ...string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	os.Remove(s.Lanzador)
	g := buildRemotoChecks()
	item := itemDe(g, itemRemotoVentana)
	if item == nil || item.Success || !strings.Contains(item.Error, "qddw.exe") {
		t.Fatalf("doctor ve que falta qddw.exe (antivirus o instalación a medias): %+v", item)
	}
	repairRemoto(g)
	if _, err := os.Stat(s.Lanzador); err != nil {
		t.Fatal("--fix lo vuelve a dejar")
	}
}

func TestRegistroDeCorrer_SinConsolaVaAlArchivo_FND238(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	t.Setenv(remoto.EnvSinConsola, "")
	if w, cerrar := registroDeCorrer(); w != os.Stderr {
		cerrar()
		t.Fatal("con consola (o systemd y launchd, que la recogen) sigue en stderr")
	}
	t.Setenv(remoto.EnvSinConsola, "1")
	w, cerrar := registroDeCorrer()
	fmt.Fprintln(w, "conectado al relay")
	cerrar()
	ruta, _ := remoto.RutaRegistroCorrer()
	if b, _ := os.ReadFile(ruta); string(b) != "conectado al relay\n" {
		t.Fatalf("sin consola el registro va a %s: %q", ruta, b)
	}
}
