package remoto

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// MEJ-079 (FND-230): un solo qdd remoto correr por QDD_HOME. Dos en el mismo tópico publican dos veces el estado,
// duplican respuestas y se pisan el seq. El candado es del sistema operativo: lo suelta la muerte del proceso.

func TestUnaSolaVez_UnSegundoConElMismoQDDHomeNoCorreYDiceElPid_MEJ079(t *testing.T) {
	homeTemporal(t)
	soltar, err := UnaSolaVez()
	if err != nil {
		t.Fatal(err)
	}
	defer soltar()
	_, err = UnaSolaVez()
	if !errors.Is(err, ErrYaCorre) {
		t.Fatalf("con el candado tomado, el segundo no corre: %v", err)
	}
	if !strings.Contains(err.Error(), "pid "+strconv.Itoa(os.Getpid())) {
		t.Fatalf("dice qué pid lo tiene: %v", err)
	}
}

func TestUnaSolaVez_ConOtroQDDHomeCorreJunto_MEJ079(t *testing.T) {
	homeTemporal(t)
	soltar, err := UnaSolaVez()
	if err != nil {
		t.Fatal(err)
	}
	defer soltar()
	t.Setenv("QDD_HOME", t.TempDir())
	otro, err := UnaSolaVez()
	if err != nil {
		t.Fatalf("el PC de muestra (otro QDD_HOME) corre junto al principal: %v", err)
	}
	otro()
}

func TestUnaSolaVez_UnCorrerPidQueQuedoNoBloquea_MEJ079(t *testing.T) {
	homeTemporal(t)
	dir, _ := DirRemoto()
	os.MkdirAll(dir, 0o700)
	os.WriteFile(filepath.Join(dir, "correr.pid"), []byte("1"), 0o600)
	soltar, err := UnaSolaVez()
	if err != nil {
		t.Fatalf("un pid vivo (o reusado) en correr.pid sin el candado no bloquea: %v", err)
	}
	defer soltar()
	if b, _ := os.ReadFile(filepath.Join(dir, "correr.pid")); string(b) != strconv.Itoa(os.Getpid()) {
		t.Fatalf("correr.pid queda con este proceso: %s", b)
	}
	if !Corriendo() {
		t.Fatal("Corriendo lo ve")
	}
}

func TestUnaSolaVez_AlSoltarQuedaLibreYNoBorraElPidDeOtro_MEJ079(t *testing.T) {
	homeTemporal(t)
	soltar, err := UnaSolaVez()
	if err != nil {
		t.Fatal(err)
	}
	dir, _ := DirRemoto()
	os.WriteFile(filepath.Join(dir, "correr.pid"), []byte("424242"), 0o600)
	soltar()
	if b, _ := os.ReadFile(filepath.Join(dir, "correr.pid")); string(b) != "424242" {
		t.Fatalf("no borra un correr.pid que ya no es suyo: %q", b)
	}
	otra, err := UnaSolaVez()
	if err != nil {
		t.Fatalf("suelto, se vuelve a tomar: %v", err)
	}
	otra()
}

// TestAyudanteCandado_MEJ079 no es una prueba: es el otro proceso de la de abajo (toma el candado y espera).
func TestAyudanteCandado_MEJ079(t *testing.T) {
	if os.Getenv("QDD_AYUDANTE_CANDADO") == "" {
		t.Skip("solo como proceso hijo")
	}
	if _, err := UnaSolaVez(); err != nil {
		t.Fatal(err)
	}
	os.Stdout.WriteString("tomado\n")
	time.Sleep(time.Minute)
}

func TestUnaSolaVez_OtroProcesoLoTieneHastaQueMuere_MEJ079(t *testing.T) {
	home := homeTemporal(t)
	bin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	hijo := exec.Command(bin, "-test.run=^TestAyudanteCandado_MEJ079$")
	hijo.Env = append(os.Environ(), "QDD_AYUDANTE_CANDADO=1", "QDD_HOME="+home)
	salida, _ := hijo.StdoutPipe()
	if err := hijo.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { hijo.Process.Kill(); hijo.Wait() })
	linea, _ := bufio.NewReader(salida).ReadString('\n')
	if strings.TrimSpace(linea) != "tomado" {
		t.Fatalf("el hijo no tomó el candado: %q", linea)
	}
	_, err = UnaSolaVez()
	if !errors.Is(err, ErrYaCorre) || !strings.Contains(err.Error(), "pid "+strconv.Itoa(hijo.Process.Pid)) {
		t.Fatalf("otro proceso con el mismo QDD_HOME lo tiene: %v", err)
	}
	hijo.Process.Kill()
	hijo.Wait()
	soltar, err := UnaSolaVez()
	if err != nil {
		t.Fatalf("muerto el otro (aunque lo maten sin soltar), el candado queda libre: %v", err)
	}
	soltar()
}
