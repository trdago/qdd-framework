package consola

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// Una versión nueva de qdd, a una tecla (pedido del usuario, 2026-11-05: como la app, que avisa y actualiza con un clic).

const binNPM = "/usr/lib/node_modules/qdd-framework/bin/qdd"

func pedidoDeVersion(root string) (Pedido, bool) {
	for _, p := range PedidosA(root, enCola) {
		if p.Version != "" {
			return p, true
		}
	}
	return Pedido{}, false
}

func TestVersionAOfrecer_SigueElCanalInstalado(t *testing.T) {
	casos := []struct {
		nombre, instalada, latest, preview, quiere string
	}{
		{"estable, hay una estable nueva", "v2.9.6", "2.9.7", "2.10.0-rc.1", "2.9.7"},
		{"estable al día no pasa a preview", "v2.9.6", "2.9.6", "2.10.0-rc.1", ""},
		{"preview, hay un rc nuevo", "v2.10.0-rc.1", "2.9.6", "2.10.0-rc.2", "2.10.0-rc.2"},
		{"preview, salió su estable", "v2.10.0-rc.2", "2.10.0", "2.10.0-rc.2", "2.10.0"},
		{"preview, la etiqueta retrocedió", "v2.10.0-rc.1", "2.9.6", "2.9.7-rc.1", ""},
		{"rc 10 es mayor que rc 9", "v2.10.0-rc.9", "2.9.6", "2.10.0-rc.10", "2.10.0-rc.10"},
		{"una versión sin forma no se ofrece", "v2.9.6", "2.9.7; rm -rf /", "", ""},
		{"un comando en el registro no se ofrece", "v2.9.6", "$(curl evil)", "", ""},
		{"un binario sin versión real no ofrece nada", "dev", "2.9.7", "", ""},
	}
	for _, c := range casos {
		tags := map[string]string{"latest": c.latest, "preview": c.preview}
		if got := VersionAOfrecer(c.instalada, tags); got != c.quiere {
			t.Errorf("%s: VersionAOfrecer = %q; quiere %q", c.nombre, got, c.quiere)
		}
	}
}

func TestInstaladoConNPM(t *testing.T) {
	if !InstaladoConNPM(binNPM) || !InstaladoConNPM(`C:\Users\ana\AppData\Roaming\npm\node_modules\qdd-framework\bin\qdd.exe`) {
		t.Fatal("una instalación de npm se reconoce en Linux y en Windows")
	}
	if InstaladoConNPM("/home/ana/qdd/tmp/qdd-rama") {
		t.Fatal("un binario compilado a mano no se actualiza con npm")
	}
}

func TestVersion_SeOfreceComoPedidoOpcional(t *testing.T) {
	root := proyectoConCola(t)
	if err := OfrecerVersion(root, "2.9.7", enCola); err != nil {
		t.Fatal(err)
	}
	p, ok := pedidoDeVersion(root)
	if !ok || p.Comando != ComandoActualizarQDD("2.9.7") || p.Tecla == "" || p.Estado != EstadoPendiente {
		t.Fatalf("queda como pedido con su tecla y el comando exacto: %+v", p)
	}
	if !strings.HasPrefix(p.Motivo, "⬆ Actualizar qdd a 2.9.7") || len([]rune(p.Motivo)) > MaxMotivo {
		t.Fatalf("el motivo dice a qué versión: %q", p.Motivo)
	}
	// FND-143: con npm 12, sin la bandera el botón dejaría a qdd sin binario.
	if !strings.Contains(p.Comando, "npm i -g qdd-framework@2.9.7 ") || !strings.Contains(p.Comando, "--allow-scripts=qdd-framework") || !strings.Contains(p.Comando, "/reload-plugins") {
		t.Fatalf("el comando instala esa versión y dice cómo cargar el panel: %q", p.Comando)
	}
	if err := OfrecerVersion(root, "2.9.7", enCola); err != nil {
		t.Fatal(err)
	}
	if n := len(PedidosA(root, enCola)); n != 1 {
		t.Fatalf("ofrecerla de nuevo no la duplica: %d pedidos", n)
	}
	if esperas := Esperas(root, enCola); len(esperas) != 0 {
		t.Fatalf("es opcional: no se le dice a la IA que el panel la espera: %v", esperas)
	}
	acciones := accionesDePedidos(root)
	if len(acciones) != 1 || acciones[0].VersionNueva != "2.9.7" {
		t.Fatalf("el panel sabe que es opcional, para no abrirse por ella: %+v", acciones)
	}
}

func TestVersion_UnaMasNuevaReemplazaALaPendiente(t *testing.T) {
	root := proyectoConCola(t)
	for _, v := range []string{"2.9.7", "2.9.8"} {
		if err := OfrecerVersion(root, v, enCola); err != nil {
			t.Fatal(err)
		}
	}
	p, ok := pedidoDeVersion(root)
	if !ok || p.Version != "2.9.8" || len(PedidosA(root, enCola)) != 1 {
		t.Fatalf("queda solo la más nueva: %+v", PedidosA(root, enCola))
	}
}

func TestVersion_AlDiaQuitaLaPendiente(t *testing.T) {
	root := proyectoConCola(t)
	if err := OfrecerVersion(root, "2.9.7", enCola); err != nil {
		t.Fatal(err)
	}
	if err := OfrecerVersion(root, "", enCola); err != nil {
		t.Fatal(err)
	}
	if _, ok := pedidoDeVersion(root); ok {
		t.Fatal("si ya no hay nada que ofrecer, el pedido sale del panel")
	}
}

func TestVersion_DescartadaNoVuelveHastaQueSalgaOtra(t *testing.T) {
	root := proyectoConCola(t)
	if err := OfrecerVersion(root, "2.9.7", enCola); err != nil {
		t.Fatal(err)
	}
	p, _ := pedidoDeVersion(root)
	if err := Descartar(root, p.ID); err != nil {
		t.Fatal(err)
	}
	if err := OfrecerVersion(root, "2.9.7", enCola); err != nil {
		t.Fatal(err)
	}
	if _, ok := pedidoDeVersion(root); ok {
		t.Fatal("la persona la descartó: no vuelve a ofrecerse")
	}
	if err := OfrecerVersion(root, "2.9.8", enCola); err != nil {
		t.Fatal(err)
	}
	if p, ok := pedidoDeVersion(root); !ok || p.Version != "2.9.8" {
		t.Fatalf("una versión más nueva sí se ofrece: %+v", p)
	}
}

func TestVersion_LeCedeSuLugarAUnPedidoDeLaIA(t *testing.T) {
	root := proyectoConCola(t)
	if err := OfrecerVersion(root, "2.9.7", enCola); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < MaxPedidos; i++ {
		if _, err := Pedir(root, "echo "+string(rune('a'+i)), "motivo "+string(rune('a'+i)), root, enCola); err != nil {
			t.Fatalf("el pedido %d de la IA tiene lugar: %v", i, err)
		}
	}
	if _, ok := pedidoDeVersion(root); ok {
		t.Fatal("la actualización es opcional: cede su lugar")
	}
}

func TestVersion_NoLaLanzaElModoGoal(t *testing.T) {
	root := proyectoConCola(t)
	if err := OfrecerVersion(root, "2.9.7", enCola); err != nil {
		t.Fatal(err)
	}
	if elegidas := correrPedidos(root, "/no/existe", "s1", enCola); len(elegidas) != 0 {
		t.Fatalf("actualizar qdd lo decide la persona, también con el modo goal: %+v", elegidas)
	}
}

func TestTraerVersion_ConsultaYOfrece(t *testing.T) {
	root := proyectoConCola(t)
	obtener := func() (map[string]string, error) {
		return map[string]string{"latest": "2.9.7", "preview": "2.9.7-rc.1"}, nil
	}
	if err := traerVersionCon(obtener, "linux", binNPM, root, "v2.9.6", enCola); err != nil {
		t.Fatal(err)
	}
	if p, ok := pedidoDeVersion(root); !ok || p.Version != "2.9.7" {
		t.Fatalf("se ofrece la del canal instalado: %+v", p)
	}
	if aviso := AvisoDeVersion(root, "v2.9.6"); aviso != "" {
		t.Fatalf("con un botón no hace falta el aviso: %q", aviso)
	}
}

func TestTraerVersion_SinRedNoTocaNada(t *testing.T) {
	root := proyectoConCola(t)
	if err := OfrecerVersion(root, "2.9.7", enCola); err != nil {
		t.Fatal(err)
	}
	obtener := func() (map[string]string, error) { return nil, errors.New("sin red") }
	if err := traerVersionCon(obtener, "linux", binNPM, root, "v2.9.6", enCola); err == nil {
		t.Fatal("una falla de red se informa")
	}
	if _, ok := pedidoDeVersion(root); !ok {
		t.Fatal("sin red no se sabe nada nuevo: lo ofrecido sigue")
	}
}

// Issue #107: en Windows, npm no puede reemplazar un qdd.exe dentro del paquete mientras una sesión lo usa, y el pedido
// mismo corre desde qdd.exe. Un botón fallaría siempre: se avisa con lo que hay que hacer.
func TestTraerVersion_EnWindowsAvisaSinBoton(t *testing.T) {
	root := proyectoConCola(t)
	obtener := func() (map[string]string, error) { return map[string]string{"latest": "2.9.7"}, nil }
	if err := traerVersionCon(obtener, "windows", `C:\npm\node_modules\qdd-framework\bin\qdd.exe`, root, "v2.9.6", enCola); err != nil {
		t.Fatal(err)
	}
	if _, ok := pedidoDeVersion(root); ok {
		t.Fatal("en Windows no se ofrece un botón que no puede funcionar")
	}
	aviso := AvisoDeVersion(root, "v2.9.6")
	if !strings.Contains(aviso, "2.9.7") || !strings.Contains(aviso, "cierra las sesiones") || !strings.Contains(aviso, "npm i -g qdd-framework@2.9.7 --allow-scripts=qdd-framework") {
		t.Fatalf("el aviso dice qué versión y qué hacer: %q", aviso)
	}
	if aviso := AvisoDeVersion(root, "v2.9.7"); aviso != "" {
		t.Fatalf("ya instalada, el aviso se va: %q", aviso)
	}
}

func TestVersionAlDia_UnBinarioManualNoPreguntaNada(t *testing.T) {
	root := proyectoConCola(t)
	VersionAlDia(root, "/home/ana/qdd/tmp/qdd-rama", "v2.9.6", enCola)
	marca, err := RutaLocal(root, "version-revisada")
	if err != nil {
		t.Fatal(err)
	}
	if exists(marca) {
		t.Fatal("sin instalación de npm no hay nada que preguntar")
	}
}

// FND-142: con qdd.exe fuera del paquete, npm no toca lo que corre y el botón funciona también en Windows.
func TestTraerVersion_EnWindowsConBinarioPorVersionOfreceElBoton(t *testing.T) {
	root := proyectoConCola(t)
	obtener := func() (map[string]string, error) { return map[string]string{"latest": "2.9.7"}, nil }
	if err := traerVersionCon(obtener, "windows", `C:\Users\ana\AppData\Local\qdd\bin\2.9.6\qdd.exe`, root, "v2.9.6", enCola); err != nil {
		t.Fatal(err)
	}
	if p, ok := pedidoDeVersion(root); !ok || p.Version != "2.9.7" {
		t.Fatalf("se ofrece el botón: %+v", p)
	}
	if aviso := AvisoDeVersion(root, "v2.9.6"); aviso != "" {
		t.Fatalf("con botón no hace falta el aviso: %q", aviso)
	}
}

// FND-194 (MEJ-049, Windows rc.44): el panel ofreció «Actualizar qdd a 2.10.0-rc.40» cuando dist-tags ya la nombraba,
// pero npm todavía no la entregaba, y el pedido falló con ETARGET. Ahora el comando pregunta a npm primero y, si npm no
// la ve, dice «todavía no» (75): el pedido queda armado y se reintenta solo.
func TestComandoActualizarQDD_EsperaAQueNpmLaEntregue_FND194(t *testing.T) {
	dir := t.TempDir()
	registro := filepath.Join(dir, "npm.log")
	shell, flag, nombre := "sh", "-c", "npm"
	falso := "#!/bin/sh\necho \"$*\" >> " + registro + "\n[ \"$1\" = view ] && [ \"$SIN_VERSION\" = 1 ] && exit 1\nexit 0\n"
	if runtime.GOOS == "windows" {
		shell, flag, nombre = "cmd", "/C", "npm.cmd"
		falso = "@echo off\r\necho %* >> \"" + registro + "\"\r\nif \"%1\"==\"view\" if \"%SIN_VERSION%\"==\"1\" exit /b 1\r\nexit /b 0\r\n"
	}
	if err := os.WriteFile(filepath.Join(dir, nombre), []byte(falso), 0o755); err != nil {
		t.Fatal(err)
	}
	// Como correrComando: la shell del sistema con el comando entero.
	correr := func(sinVersion string) (int, string) {
		os.Remove(registro)
		c := exec.Command(shell, flag, ComandoActualizarQDD("2.10.0-rc.40"))
		c.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"), "SIN_VERSION="+sinVersion)
		c.Run()
		b, _ := os.ReadFile(registro)
		return c.ProcessState.ExitCode(), string(b)
	}
	codigo, llamadas := correr("1")
	if codigo != CodigoTodaviaNo || strings.Contains(llamadas, "i -g") {
		t.Fatalf("sin la versión en npm tiene que decir «todavía no» sin instalar: código %d, llamadas %q", codigo, llamadas)
	}
	codigo, llamadas = correr("0")
	if codigo != 0 || !strings.Contains(llamadas, "i -g qdd-framework@2.10.0-rc.40") || !strings.Contains(llamadas, "--prefer-online") {
		t.Fatalf("con la versión en npm instala sin la caché vieja: código %d, llamadas %q", codigo, llamadas)
	}
}
