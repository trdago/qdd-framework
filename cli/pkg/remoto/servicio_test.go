package remoto

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

// Rule: Prohibido usar 'else'
// RFC 0013 (rc.2): el servicio de cada sistema se genera en un directorio temporal; en pruebas no se instala nada.

func TestServicio_SystemdLaunchdYTareaDeWindows(t *testing.T) {
	dir := t.TempDir()
	base := Servicio{Bin: "/opt/qdd bin/qdd", Usuario: dir, QDDHome: "/h/q", Path: "/usr/bin:/home/u/.nvm/bin", DirQDD: filepath.Join(dir, "qdd", "remoto")}
	var corridos []string
	anotar := func(n string, a ...string) error {
		corridos = append(corridos, n+" "+strings.Join(a, " "))
		return nil
	}

	linux := base
	linux.SO = "linux"
	archivo, err := linux.Instalar(anotar)
	if err != nil {
		t.Fatal(err)
	}
	if archivo != filepath.Join(dir, ".config", "systemd", "user", "qdd-remoto.service") {
		t.Fatalf("unidad de usuario: %s", archivo)
	}
	b, _ := os.ReadFile(archivo)
	u := string(b)
	for _, want := range []string{`ExecStart="/opt/qdd bin/qdd" remoto correr`, `Environment="QDD_HOME=/h/q"`, `Environment="PATH=/usr/bin:/home/u/.nvm/bin"`, "Restart=on-failure", "WantedBy=default.target"} {
		if !strings.Contains(u, want) {
			t.Errorf("la unidad lleva %q:\n%s", want, u)
		}
	}
	if strings.Join(corridos, "; ") != "systemctl --user daemon-reload; systemctl --user enable qdd-remoto.service; systemctl --user restart qdd-remoto.service" {
		t.Fatalf("enciende con systemctl --user: %v", corridos)
	}
	corridos = nil
	if err := linux.Quitar(anotar); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(archivo); err == nil || len(corridos) == 0 {
		t.Fatal("quitar lo apaga y borra la unidad")
	}

	mac := base
	mac.SO = "darwin"
	archivo, _ = mac.Instalar(func(string, ...string) error { return nil })
	b, _ = os.ReadFile(archivo)
	if !strings.HasSuffix(archivo, "Library/LaunchAgents/dev.qdd.remoto.plist") || !strings.Contains(string(b), "<string>/opt/qdd bin/qdd</string><string>remoto</string><string>correr</string>") || !strings.Contains(string(b), "<key>QDD_HOME</key><string>/h/q</string>") {
		t.Fatalf("plist: %s\n%s", archivo, b)
	}

	win := base
	win.SO = "windows"
	archivo, _ = win.Instalar(func(string, ...string) error { return nil })
	b, _ = os.ReadFile(archivo)
	if len(b) < 2 || b[0] != 0xFF || b[1] != 0xFE {
		t.Fatal("schtasks lee el XML en UTF-16 con BOM")
	}
	u16 := make([]uint16, (len(b)-2)/2)
	for i := range u16 {
		u16[i] = uint16(b[2+2*i]) | uint16(b[3+2*i])<<8
	}
	xml := string(utf16.Decode(u16))
	if !strings.Contains(xml, "<Command>/opt/qdd bin/qdd</Command><Arguments>remoto correr</Arguments>") || !strings.Contains(xml, "<LogonTrigger>") {
		t.Fatalf("tarea al iniciar sesión: %s", xml)
	}
	if c := win.Comandos(true); c[0][0] != "schtasks" || c[0][4] != "/XML" {
		t.Fatalf("schtasks /Create /XML: %v", c)
	}

	otro := base
	otro.SO = "plan9"
	if _, err := otro.Instalar(anotar); err == nil {
		t.Fatal("un sistema sin servicio lo dice")
	}
}
