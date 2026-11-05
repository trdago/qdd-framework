package remoto

import "testing"

// Rule: Prohibido usar 'else'

func TestServicio_TareaSinEstaVersion_FND142(t *testing.T) {
	dir := t.TempDir()
	vieja := Servicio{SO: "windows", Bin: `C:\AppData\Local\qdd\bin\2.10.0\qdd.exe`, Usuario: dir, DirQDD: dir}
	nueva := vieja
	nueva.Bin = `C:\AppData\Local\qdd\bin\2.10.1\qdd.exe`
	if nueva.TareaSinEstaVersion() {
		t.Fatal("sin tarea instalada no hay nada que revisar")
	}
	if _, err := vieja.Instalar(func(string, ...string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if vieja.TareaSinEstaVersion() {
		t.Fatal("la tarea lanza esta versión")
	}
	if !nueva.TareaSinEstaVersion() {
		t.Fatal("la tarea lanza una versión anterior")
	}
	manual := vieja
	manual.Bin = `C:\repo\qdd.exe`
	if manual.TareaSinEstaVersion() {
		t.Fatal("un binario compilado a mano no cambia la tarea")
	}
	linux := nueva
	linux.SO = "linux"
	if linux.TareaSinEstaVersion() {
		t.Fatal("solo Windows guarda el binario por versión")
	}
}
