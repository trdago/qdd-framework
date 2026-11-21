package cmd

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// MEJ-020: el panel lee 'qdd consola seguir' (un proceso por sesión) en vez de 'qdd consola panel --json' cada 2 s; lo
// que dibuja no puede cambiar según cuál de los dos lea.

// comoQDD es este binario de prueba corriendo como qdd, con un HOME y un QDD_HOME propios (sin variables de Claude).
func comoQDD(dir, home, qddHome string, args ...string) *exec.Cmd {
	c := exec.Command(os.Args[0], args...)
	c.Dir = dir
	for _, v := range os.Environ() {
		nombre, _, _ := strings.Cut(v, "=")
		if nombre == "CI" || nombre == "QDD_PRUEBAS_COMPLETAS" || strings.HasPrefix(nombre, "CLAUDE") {
			continue
		}
		c.Env = append(c.Env, v)
	}
	c.Env = append(c.Env, envComoQDD+"=1", "HOME="+home, "USERPROFILE="+home, "QDD_HOME="+qddHome, "XDG_CONFIG_HOME="+filepath.Join(home, ".config"))
	return c
}

// primeraLineaDeSeguir lee la primera línea con datos de 'qdd consola seguir' y lo cierra.
func primeraLineaDeSeguir(t *testing.T, c *exec.Cmd) string {
	t.Helper()
	salida, err := c.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = c.Process.Kill()
		_ = c.Wait()
	}()
	lineas := make(chan string)
	go func() {
		s := bufio.NewScanner(salida)
		s.Buffer(make([]byte, 64<<10), 16<<20)
		for s.Scan() {
			if strings.TrimSpace(s.Text()) != "" {
				lineas <- s.Text()
				return
			}
		}
		close(lineas)
	}()
	select {
	case l := <-lineas:
		return l
	case <-time.After(30 * time.Second):
		t.Fatal("qdd consola seguir no entregó nada en 30 s")
	}
	return ""
}

// contrato: estado#5420d388
// Caja negra: en un proyecto con tablero, una espera y un pedido, la primera línea de 'qdd consola seguir' es byte a
// byte la salida de 'qdd consola panel --json' con lo mismo pedido.
func TestConsolaSeguir_EntregaExactamenteElJSONDelPanel_MEJ020(t *testing.T) {
	home := t.TempDir()
	qddHome := filepath.Join(home, "qdd")
	t.Setenv("QDD_HOME", qddHome)
	root := t.TempDir()
	hace := time.Now().Add(-72 * time.Hour)
	if err := estado.Init(root, []string{"Código", "Pruebas"}, hace); err != nil {
		t.Fatal(err)
	}
	if err := estado.Set(root, estado.Cambio{Frente: "t", Nombre: "Tarea", Etapa: "Código", Estado: "⏳", Bloqueo: "¿Sigo?", DeQuien: "la persona"}, hace); err != nil {
		t.Fatal(err)
	}
	if _, err := consola.Pedir(root, "git push origin v9.9.9", "Publicar v9.9.9", "", hace); err != nil {
		t.Fatal(err)
	}
	con := "atender,mejoras,flujos,esperas_persona,pregunta"
	// Dos lecturas seguidas pueden caer a ambos lados de un minuto: igual a la primera o a la segunda vuelta.
	for intento := 1; ; intento++ {
		panel, err := comoQDD(root, home, qddHome, "consola", "panel", "--json", "--con", con).Output()
		if err != nil {
			t.Fatalf("qdd consola panel --json: %v", err)
		}
		seguir := primeraLineaDeSeguir(t, comoQDD(root, home, qddHome, "consola", "seguir", "--con", con))
		if !strings.Contains(seguir, "Publicar v9.9.9") || !strings.Contains(seguir, "¿Sigo?") {
			t.Fatalf("la lectura trae el pedido y la espera: %s", seguir)
		}
		if seguir == strings.TrimRight(string(panel), "\n") {
			return
		}
		if intento == 2 {
			t.Fatalf("seguir y panel --json entregan lo mismo:\nseguir: %s\npanel:  %s", seguir, panel)
		}
	}
}

// contrato: estado#5420d388
// La otra línea del flujo, la del latido con teléfono, solo trae telefono_entregar: nada del panel que lo reemplace.
func TestLatidoDelFlujo_SoloTraeTelefonoEntregar_MEJ020(t *testing.T) {
	emparejado(t)
	root := t.TempDir()
	now := time.Now()
	instruccionDelTelefono(t, root, now)
	var lineas [][]byte
	if err := latidoDelFlujo(root, "s", now, func(l []byte) error { lineas = append(lineas, l); return nil }); err != nil {
		t.Fatal(err)
	}
	if len(lineas) != 1 {
		t.Fatalf("una línea: %q", lineas)
	}
	var campos map[string]json.RawMessage
	if err := json.Unmarshal(lineas[0], &campos); err != nil {
		t.Fatal(err)
	}
	if _, ok := campos["telefono_entregar"]; !ok || len(campos) != 1 {
		t.Fatalf("solo telefono_entregar, sin datos del panel: %s", lineas[0])
	}
}
