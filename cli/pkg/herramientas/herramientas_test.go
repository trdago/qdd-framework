package herramientas

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func escribir(t *testing.T, path, contenido string, modo os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contenido), modo); err != nil {
		t.Fatal(err)
	}
}

// conShell salta en Windows: las herramientas de prueba son scripts de sh.
func conShell(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("las herramientas de prueba son scripts de sh")
	}
}

// simulador escribe en bin/simulador un script que imprime cada argumento en su línea.
func simulador(t *testing.T, root string) {
	t.Helper()
	escribir(t, filepath.Join(root, "bin", "simulador"), "#!/bin/sh\n[ \"$1\" = --help ] && { echo 'uso: simulador send <mensaje>'; exit 0; }\nprintf '%s\\n' \"$@\"\n", 0755)
}

const definicion = `id: simulador-ws
nombre: Simulador WS
descripcion: Simula lo que manda un usuario al bot.
comando: [./bin/simulador]
acciones:
  - id: enviar
    nombre: Enviar como usuario
    args: [send, "{{mensaje}}", [--chat, "{{chat}}"], [--seco, "{{seco}}"]]
    campos:
      - {id: mensaje, tipo: texto, requerido: true}
      - {id: chat, tipo: opcion, opciones: [soporte, ventas]}
      - {id: seco, tipo: si-no}
    modo: una-vez
  - id: reiniciar
    nombre: Reiniciar el estado
    args: [reset]
    modo: una-vez
    confirmar: true
prueba: {args: [--help]}
`

func parsear(t *testing.T, texto string) Herramienta {
	t.Helper()
	h, err := Parse([]byte(texto))
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func correr(t *testing.T, root string, h Herramienta, accion string, valores map[string]string, o Opciones) (string, error) {
	t.Helper()
	var salida bytes.Buffer
	o.Salida, o.Errores = &salida, &salida
	_, err := Correr(context.Background(), root, h, accion, valores, o)
	return salida.String(), err
}

func TestCandidatos_LosEjecutablesDelProyecto_HER_C01(t *testing.T) {
	conShell(t)
	root := t.TempDir()
	simulador(t, root)
	escribir(t, filepath.Join(root, "scripts", "sembrar.sh"), "#!/bin/sh\n", 0755)
	escribir(t, filepath.Join(root, "scripts", "notas.txt"), "no es un ejecutable\n", 0644)
	escribir(t, filepath.Join(root, "cmd", "worker", "main.go"), "package main\n", 0644)
	escribir(t, filepath.Join(root, "target", "release", "bot"), "binario", 0755)
	escribir(t, filepath.Join(root, "package.json"), `{"scripts": {"dev": "vite", "seed": "node seed.js"}}`, 0644)
	escribir(t, filepath.Join(root, "Makefile"), ".PHONY: build\nbuild:\n\tgo build\nmigrar: build\n\t./migrar\nVAR := x\n", 0644)
	escribir(t, filepath.Join(root, ".qdd", "herramientas", "sembrar.yaml"), strings.NewReplacer("./bin/simulador", "./scripts/sembrar.sh", "id: simulador-ws", "id: sembrar").Replace(definicion), 0644)

	var got []string
	for _, c := range Candidatos(root) {
		linea := strings.Join(c.Comando, " ")
		if c.Agregada {
			linea += " (agregada)"
		}
		got = append(got, linea)
	}
	want := []string{"./bin/simulador", "./scripts/sembrar.sh (agregada)", "go run ./cmd/worker", "./target/release/bot", "npm run dev", "npm run seed", "make build", "make migrar"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("los ejecutables del proyecto, en orden y sin lo que no se ejecuta:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestGuardar_SinDefinicionValidaNoHayPestaña_HER_C02(t *testing.T) {
	conShell(t)
	root := t.TempDir()
	simulador(t, root)
	escribir(t, filepath.Join(root, "bin", "roto"), "#!/bin/sh\necho 'falta la base de datos' >&2\nexit 3\n", 0755)
	casos := map[string]string{
		"un campo no declarado":   strings.Replace(definicion, `"{{mensaje}}"`, `"{{texto}}"`, 1),
		"un secreto":              strings.Replace(definicion, "args: [reset]", "args: [reset, --token=ghp_0123456789abcdefghijABCDEFGHIJ012345]", 1),
		"un campo a medias":       strings.Replace(definicion, `"{{mensaje}}"`, `"--m={{mensaje}}"`, 1),
		"un modo que no existe":   strings.Replace(definicion, "modo: una-vez\n  - id: reiniciar", "modo: siempre\n  - id: reiniciar", 1),
		"su prueba de humo falla": strings.Replace(definicion, "./bin/simulador", "./bin/roto", 1),
		"un comando fuera":        strings.Replace(definicion, "./bin/simulador", "../otro/bin", 1),
	}
	porque := map[string]string{"un campo no declarado": "texto", "un secreto": "secreto", "un campo a medias": "argumento entero", "un modo que no existe": "siempre", "su prueba de humo falla": "falta la base de datos", "un comando fuera": "fuera del proyecto"}
	for caso, texto := range casos {
		_, problemas := Guardar(root, []byte(texto))
		if len(problemas) == 0 || !strings.Contains(strings.Join(problemas, "\n"), porque[caso]) {
			t.Errorf("%s: no se guarda y se dice por qué (%q): %v", caso, porque[caso], problemas)
		}
	}
	if hs, _ := Cargar(root); len(hs) != 0 {
		t.Fatalf("nada inválido quedó guardado: %+v", hs)
	}
	path, problemas := Guardar(root, []byte(definicion))
	if len(problemas) != 0 || filepath.ToSlash(path) != ".qdd/herramientas/simulador-ws.yaml" {
		t.Fatalf("una definición válida, con su prueba de humo en 0, se guarda versionada: %q %v", path, problemas)
	}
	if hs, errs := Cargar(root); len(hs) != 1 || len(errs) != 0 || hs[0].Nombre != "Simulador WS" {
		t.Fatalf("y se vuelve a leer igual: %+v %v", hs, errs)
	}
}

func TestCorrer_UnCampoEsUnArgumento_HER_C03(t *testing.T) {
	conShell(t)
	root := t.TempDir()
	simulador(t, root)
	h := parsear(t, definicion)
	salida, err := correr(t, root, h, "enviar", map[string]string{"mensaje": "hola; rm -rf / && touch pwned"}, Opciones{})
	if err != nil || salida != "send\nhola; rm -rf / && touch pwned\n" {
		t.Fatalf("lo que escribe la persona llega como un solo argumento, sin shell; un grupo sin valor no va: %q %v", salida, err)
	}
	if _, err := os.Stat(filepath.Join(root, "pwned")); !os.IsNotExist(err) {
		t.Fatal("no corrió nada más")
	}
	salida, err = correr(t, root, h, "enviar", map[string]string{"mensaje": "hola", "chat": "ventas", "seco": "si"}, Opciones{})
	if err != nil || salida != "send\nhola\n--chat\nventas\n--seco\n" {
		t.Fatalf("un grupo con su valor va entero; un si-no solo decide si su grupo va: %q %v", salida, err)
	}
	for _, valores := range []map[string]string{{}, {"mensaje": "hola", "chat": "otro"}, {"mensaje": "hola", "nadie": "x"}} {
		if _, err := correr(t, root, h, "enviar", valores, Opciones{}); err == nil {
			t.Errorf("%v: falta lo requerido, una opción que no existe o un campo que la acción no tiene: no corre", valores)
		}
	}
}

func TestCorrer_FaltaUnaVariableNoCorreNiMuestraValores_HER_C04(t *testing.T) {
	conShell(t)
	root := t.TempDir()
	simulador(t, root)
	h := parsear(t, strings.Replace(definicion, "comando: [./bin/simulador]", "comando: [./bin/simulador]\nentorno: [BOT_URL, BOT_TOKEN]", 1))
	t.Setenv("BOT_URL", "ws://bot.local/socket")
	os.Unsetenv("BOT_TOKEN")
	salida, err := correr(t, root, h, "enviar", map[string]string{"mensaje": "hola"}, Opciones{})
	if err == nil || !strings.Contains(err.Error(), "BOT_TOKEN") || strings.Contains(err.Error(), "BOT_URL") || strings.Contains(err.Error()+salida, "ws://") || salida != "" {
		t.Fatalf("dice cuál falta, ninguno de sus valores, y no corre: %q %v", salida, err)
	}
	t.Setenv("BOT_TOKEN", "abc")
	if salida, err = correr(t, root, h, "enviar", map[string]string{"mensaje": "hola"}, Opciones{}); err != nil || !strings.HasPrefix(salida, "send") {
		t.Fatalf("con el entorno completo corre: %q %v", salida, err)
	}
}

func TestCorrer_LoQueConfirmaNoCorreSinElSi_HER_C06(t *testing.T) {
	conShell(t)
	root := t.TempDir()
	simulador(t, root)
	h := parsear(t, definicion)
	if salida, err := correr(t, root, h, "reiniciar", nil, Opciones{}); err == nil || salida != "" || !strings.Contains(err.Error(), "confirm") {
		t.Fatalf("sin confirmación no corre: %q %v", salida, err)
	}
	if salida, err := correr(t, root, h, "reiniciar", nil, Opciones{Confirmado: true}); err != nil || salida != "reset\n" {
		t.Fatalf("confirmada, corre: %q %v", salida, err)
	}
}

func TestQuitar_BorraSuDefinicion_HER_C08(t *testing.T) {
	conShell(t)
	root := t.TempDir()
	simulador(t, root)
	if _, problemas := Guardar(root, []byte(definicion)); len(problemas) != 0 {
		t.Fatal(problemas)
	}
	if err := Quitar(root, "simulador-ws"); err != nil {
		t.Fatal(err)
	}
	if hs, _ := Cargar(root); len(hs) != 0 {
		t.Fatalf("quitada, no queda su definición: %+v", hs)
	}
	if err := Quitar(root, "simulador-ws"); err == nil {
		t.Fatal("quitar una que no existe lo dice")
	}
	if err := Quitar(root, "../config"); err == nil {
		t.Fatal("un id inválido no borra nada fuera de .qdd/herramientas")
	}
}

func TestAyuda_LeeLaAyudaSinEntrada(t *testing.T) {
	conShell(t)
	root := t.TempDir()
	simulador(t, root)
	escribir(t, filepath.Join(root, "bin", "espera"), "#!/bin/sh\nread linea\necho \"leyó: $linea\"\n", 0755)
	if texto, err := Ayuda(context.Background(), root, []string{"./bin/simulador"}); err != nil || !strings.Contains(texto, "uso: simulador") {
		t.Fatalf("la ayuda de --help: %q %v", texto, err)
	}
	if texto, _ := Ayuda(context.Background(), root, []string{"./bin/espera"}); strings.Contains(texto, "leyó: x") {
		t.Fatalf("sin entrada: nada le escribe a su stdin: %q", texto)
	}
}

func TestDiagnosticar_CadaHerramientaDiceQueLeFaltaYComoSeArregla(t *testing.T) {
	conShell(t)
	root := t.TempDir()
	if ds := Diagnosticar(root); len(ds) != 0 {
		t.Fatalf("sin herramientas, nada que diagnosticar: %+v", ds)
	}
	simulador(t, root)
	if _, problemas := Guardar(root, []byte(definicion)); len(problemas) != 0 {
		t.Fatal(problemas)
	}
	if ds := Diagnosticar(root); len(ds) != 1 || !ds[0].OK || !strings.Contains(ds[0].Nombre, "Simulador WS") {
		t.Fatalf("lista, con su prueba de humo en 0: %+v", ds)
	}
	escribir(t, filepath.Join(root, "bin", "simulador"), "#!/bin/sh\nexit 2\n", 0755)
	if ds := Diagnosticar(root); ds[0].OK || !strings.Contains(ds[0].Detalle, "salió con 2") {
		t.Fatalf("su prueba de humo falla: %+v", ds)
	}
	os.Remove(filepath.Join(root, "bin", "simulador"))
	ds := Diagnosticar(root)
	if ds[0].OK || !strings.Contains(ds[0].Detalle, "no existe") || !strings.Contains(ds[0].Detalle, "qdd herramienta quitar simulador-ws --si") {
		t.Fatalf("sin su ejecutable: dice que lo compiles o la quites, con el comando exacto: %+v", ds)
	}
	escribir(t, filepath.Join(root, Dir, "rota.yaml"), "id: [\n", 0644)
	if ds = Diagnosticar(root); len(ds) != 2 || ds[1].OK || !strings.Contains(ds[1].Nombre, "rota.yaml") {
		t.Fatalf("una definición que no se lee también se ve: %+v", ds)
	}
}
