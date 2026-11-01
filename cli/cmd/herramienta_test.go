package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestHerramienta_CampoIgualValorLlegaEntero(t *testing.T) {
	v, err := valoresDeArgs([]string{"mensaje=hola = chao; rm -rf /", "chat=ventas", "vacio="})
	if err != nil || v["mensaje"] != "hola = chao; rm -rf /" || v["chat"] != "ventas" || v["vacio"] != "" || len(v) != 3 {
		t.Fatalf("todo lo que sigue al primer «=» es el valor: %v %v", v, err)
	}
	if _, err := valoresDeArgs([]string{"suelto"}); err == nil {
		t.Fatal("un valor sin campo se rechaza")
	}
}

func TestHerramienta_ElListadoDiceLoQueFaltaEnEstaMaquina(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("la herramienta de prueba es un script de sh")
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".qdd", "herramientas"), 0755); err != nil {
		t.Fatal(err)
	}
	def := "id: sim\nnombre: Sim\ncomando: [./bin/sim]\nentorno: [SIM_CLAVE_DE_PRUEBA]\nacciones:\n  - {id: ver, nombre: Ver, args: [ver]}\nprueba: {args: [--help]}\n"
	if err := os.WriteFile(filepath.Join(root, ".qdd", "herramientas", "sim.yaml"), []byte(def), 0644); err != nil {
		t.Fatal(err)
	}
	vs, errs := listadoDeHerramientas(root)
	if len(vs) != 1 || len(errs) != 0 || !strings.Contains(vs[0].Aviso, "no existe") {
		t.Fatalf("sin el ejecutable lo dice: %+v %v", vs, errs)
	}
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "sim"), []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if vs, _ = listadoDeHerramientas(root); !strings.Contains(vs[0].Aviso, "SIM_CLAVE_DE_PRUEBA") {
		t.Fatalf("sin la variable lo dice: %+v", vs)
	}
	t.Setenv("SIM_CLAVE_DE_PRUEBA", "x")
	if vs, _ = listadoDeHerramientas(root); vs[0].Aviso != "" || vs[0].Archivo != ".qdd/herramientas/sim.yaml" {
		t.Fatalf("lista para usar: %+v", vs)
	}
}

func TestDoctor_LasHerramientasSoloSiElProyectoTiene(t *testing.T) {
	root := t.TempDir()
	if g := buildHerramientasChecks(root); len(g.Items) != 0 {
		t.Fatalf("sin herramientas, doctor no muestra el grupo: %+v", g.Items)
	}
	if err := os.MkdirAll(filepath.Join(root, ".qdd", "herramientas"), 0755); err != nil {
		t.Fatal(err)
	}
	def := "id: sim\nnombre: Sim\ncomando: [./bin/sim]\nacciones:\n  - {id: ver, nombre: Ver, args: [ver]}\nprueba: {args: [--help]}\n"
	if err := os.WriteFile(filepath.Join(root, ".qdd", "herramientas", "sim.yaml"), []byte(def), 0644); err != nil {
		t.Fatal(err)
	}
	g := buildHerramientasChecks(root)
	if len(g.Items) != 1 || g.Items[0].Success || !strings.Contains(g.Items[0].Error, "qdd herramienta quitar sim --si") {
		t.Fatalf("sin su ejecutable, doctor lo dice con el comando para quitarla: %+v", g.Items)
	}
}
