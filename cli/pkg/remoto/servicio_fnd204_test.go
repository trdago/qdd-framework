package remoto

import (
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// FND-204 (MEJ-064, REP-483): en Windows, al actualizar qdd la tarea se volvía a registrar y nada la arrancaba. Lo que
// corría terminaba con salida 3 al ver la versión nueva, pero RestartOnFailure de una tarea programada solo reintenta
// cuando la tarea no pudo arrancar, no cuando el programa sale con error: el teléfono quedaba sin PC hasta el próximo
// inicio de sesión. Además quedaba la tarea «qdd-remoto» que se registró a mano mientras qdd no podía (FND-195),
// apuntando a una versión ya desinstalada.

var reReintento = regexp.MustCompile(`<TimeTrigger>.*?</TimeTrigger>`)

// tareaDeAntes deja instalada la tarea como la escribía qdd antes de FND-204: solo con el disparador al iniciar sesión.
func tareaDeAntes(t *testing.T, s Servicio) string {
	t.Helper()
	archivo, err := s.Instalar(func(string, ...string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(archivo)
	if err := os.WriteFile(archivo, utf16ConBOM(reReintento.ReplaceAllString(desdeUTF16(b), "")), 0o600); err != nil {
		t.Fatal(err)
	}
	return archivo
}

func TestServicio_LaTareaDeWindowsVuelveSiSeCae_FND204(t *testing.T) {
	dir := t.TempDir()
	s := Servicio{SO: "windows", Bin: `C:\qdd.exe`, Usuario: dir, DirQDD: dir, Cuenta: `PC\ana`}
	archivo, err := s.Instalar(func(string, ...string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(archivo)
	xml := desdeUTF16(b)
	if !strings.Contains(xml, "<LogonTrigger>") {
		t.Fatalf("sigue arrancando al iniciar sesión: %s", xml)
	}
	want := `<TimeTrigger><Repetition><Interval>PT5M</Interval><StopAtDurationEnd>false</StopAtDurationEnd></Repetition><StartBoundary>2026-01-01T00:00:00</StartBoundary><Enabled>true</Enabled></TimeTrigger></Triggers>`
	if !strings.Contains(xml, want) {
		t.Fatalf("cada 5 minutos la vuelve a lanzar si no corre (IgnoreNew no abre otra): %s", xml)
	}
	if !strings.Contains(xml, "<MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>") {
		t.Fatalf("una corriendo no se duplica: %s", xml)
	}
	if s.TareaSinReintento() {
		t.Fatal("una tarea recién escrita ya vuelve sola")
	}
}

func TestServicio_RenovarTareaLaArrancaYLeAgregaElReintento_FND204(t *testing.T) {
	dir := t.TempDir()
	vieja := Servicio{SO: "windows", Bin: `C:\L\qdd\bin\2.10.0-rc.62\qdd.exe`, Usuario: dir, DirQDD: dir, Cuenta: `PC\ana`}
	archivo := tareaDeAntes(t, vieja)
	nueva := vieja
	nueva.Bin = `C:\L\qdd\bin\2.10.0-rc.70\qdd.exe`
	if !nueva.TareaSinReintento() {
		t.Fatal("ve que la tarea de antes no vuelve si se cae")
	}
	var corrio []string
	anota := func(n string, a ...string) error {
		corrio = append(corrio, n+" "+strings.Join(a, " "))
		return nil
	}
	if err := nueva.RenovarTarea(anota); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(archivo)
	xml := desdeUTF16(b)
	if !strings.Contains(xml, `2.10.0-rc.70\qdd.exe</Command>`) || strings.Count(xml, "<TimeTrigger>") != 1 {
		t.Fatalf("lanza la versión nueva y vuelve sola: %s", xml)
	}
	if strings.Join(corrio, "; ") != `schtasks /Create /TN QDD remoto /XML `+archivo+` /F; schtasks /Run /TN QDD remoto` {
		t.Fatalf("la registra y la arranca ya, sin esperar al próximo inicio de sesión: %v", corrio)
	}
	corrio = nil
	if err := nueva.RenovarTarea(anota); err != nil || len(corrio) != 0 || nueva.TareaSinReintento() {
		t.Fatalf("ya al día no hace nada: %v %v", err, corrio)
	}
}

func TestServicio_RenovarTareaSoloPorElReintento_FND204(t *testing.T) {
	dir := t.TempDir()
	s := Servicio{SO: "windows", Bin: `C:\a mano\qdd.exe`, Usuario: dir, DirQDD: dir, Cuenta: `PC\ana`}
	tareaDeAntes(t, s)
	var corrio []string
	if err := s.RenovarTarea(func(n string, a ...string) error { corrio = append(corrio, a[0]); return nil }); err != nil {
		t.Fatal(err)
	}
	if strings.Join(corrio, " ") != "/Create /Run" || s.TareaSinReintento() {
		t.Fatalf("aunque el binario sea uno compilado a mano, el reintento se agrega: %v", corrio)
	}
}

func TestServicio_ArrancarFallaLoDice_FND204(t *testing.T) {
	dir := t.TempDir()
	s := Servicio{SO: "windows", Bin: `C:\qdd.exe`, Usuario: dir, DirQDD: dir, Cuenta: `PC\ana`}
	tareaDeAntes(t, s)
	err := s.RenovarTarea(func(n string, a ...string) error {
		if a[0] == "/Run" {
			return errors.New("acceso denegado")
		}
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "/Run") {
		t.Fatalf("si no arranca, lo dice: %v", err)
	}
}

func TestServicio_FueraDeWindowsNoHayReintentoQueAgregar_FND204(t *testing.T) {
	dir := t.TempDir()
	s := Servicio{SO: "linux", Bin: "/opt/qdd", Usuario: dir, DirQDD: dir}
	if _, err := s.Instalar(func(string, ...string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if s.TareaSinReintento() {
		t.Fatal("systemd ya reinicia ante la salida 3 (Restart=on-failure)")
	}
	if s.TareaSobrante(func(string, ...string) (string, error) { t.Fatal("fuera de Windows no consulta"); return "", nil }) {
		t.Fatal("fuera de Windows no hay tarea sobrante")
	}
	if len(s.ComandosAlQuitarse()) != 0 {
		t.Fatal("systemd y launchd no reinician una salida 0: no hay nada que pausar")
	}
}

func TestServicio_TareaSobranteDeAntes_FND204(t *testing.T) {
	s := Servicio{SO: "windows", Bin: `C:\qdd.exe`}
	var pregunto []string
	consulta := func(salida string, err error) Consulta {
		return func(n string, a ...string) (string, error) {
			pregunto = append(pregunto, n+" "+strings.Join(a, " "))
			return salida, err
		}
	}
	// schtasks /Query /XML puede salir en UTF-16: los NUL entre letras no esconden lo que lanza.
	conNul := strings.Join(strings.Split(`<Command>C:\L\qdd\bin\2.10.0-rc.59\qdd.exe</Command><Arguments>remoto correr</Arguments>`, ""), "\x00")
	if !s.TareaSobrante(consulta(conNul, nil)) {
		t.Fatal("la tarea «qdd-remoto» que lanza qdd remoto correr sobra")
	}
	if pregunto[0] != "schtasks /Query /TN qdd-remoto /XML" {
		t.Fatalf("consulta por su nombre: %v", pregunto)
	}
	if s.TareaSobrante(consulta("", errors.New("no existe"))) {
		t.Fatal("sin esa tarea no sobra nada")
	}
	if s.TareaSobrante(consulta(`<Command>C:\otra cosa.exe</Command>`, nil)) {
		t.Fatal("una tarea con ese nombre que lanza otra cosa no es de qdd: no se toca")
	}
	var corrio []string
	if err := s.QuitarTareaSobrante(func(n string, a ...string) error { corrio = append(corrio, n+" "+strings.Join(a, " ")); return nil }); err != nil {
		t.Fatal(err)
	}
	if strings.Join(corrio, "; ") != "schtasks /End /TN qdd-remoto; schtasks /Delete /TN qdd-remoto /F" {
		t.Fatalf("la apaga y la borra: %v", corrio)
	}
}

func TestServicio_UnPCQuitadoNoSeRelanzaCadaCincoMinutos_FND204(t *testing.T) {
	s := Servicio{SO: "windows", Bin: `C:\qdd.exe`}
	c := s.ComandosAlQuitarse()
	if len(c) != 1 || strings.Join(c[0], " ") != "schtasks /Change /TN QDD remoto /DISABLE" {
		t.Fatalf("si el teléfono quitó este PC, la tarea se desactiva hasta volver a emparejarlo: %v", c)
	}
}
