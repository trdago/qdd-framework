//go:build windows

package consola

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// FND-193 (MEJ-048, Windows rc.44): «Actualizar qdd» falló con «exit -1 · el proceso que lo ejecutaba terminó sin
// anotar el resultado». El pedido ya corría aparte (FND-089), pero en Windows CREATE_NEW_PROCESS_GROUP no lo saca del
// Job Object de quien lo lanza: si ese job mata todo al cerrarse, al cerrar o reiniciar Claude Code el pedido muere
// con él. Estas pruebas meten al lanzador en un job así, lo cierran y miran si lo lanzado sigue vivo.

const (
	envLanzadorJob = "QDD_PRUEBA_LANZADOR_JOB"
	envNietoJob    = "QDD_PRUEBA_NIETO_JOB"
)

// TestJobAyudante no prueba nada por sí sola: es el lanzador (lanza como el panel y espera) y el nieto (espera) que
// las pruebas de abajo levantan con el binario de prueba.
func TestJobAyudante(t *testing.T) {
	if os.Getenv(envNietoJob) != "" {
		time.Sleep(60 * time.Second)
		return
	}
	dir := os.Getenv(envLanzadorJob)
	if dir == "" {
		t.Skip("solo corre como ayudante")
	}
	for !existe(filepath.Join(dir, "listo")) {
		time.Sleep(20 * time.Millisecond)
	}
	c := exec.Command(os.Args[0], "-test.run=^TestJobAyudante$")
	c.Env = append(os.Environ(), envNietoJob+"=1")
	separar(c)
	c, err := arrancar(c)
	if err != nil {
		os.WriteFile(filepath.Join(dir, "error"), []byte(err.Error()), 0o600)
		return
	}
	os.WriteFile(filepath.Join(dir, "nieto"), []byte(strconv.Itoa(c.Process.Pid)), 0o600)
	time.Sleep(60 * time.Second)
}

func TestPedidoSigueVivoAlCerrarElJobDelPanel_FND193(t *testing.T) {
	nieto := lanzarDesdeUnJob(t, windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE|windows.JOB_OBJECT_LIMIT_BREAKAWAY_OK)
	if !procesoVivo(nieto) {
		t.Fatal("al cerrar el job de quien lo lanzó, el pedido murió con él")
	}
}

func TestPedidoSeLanzaAunqueElJobNoDejeSalir_FND193(t *testing.T) {
	// Un job que no permite salir rechaza CREATE_BREAKAWAY_FROM_JOB: el pedido se lanza igual, como antes.
	lanzarDesdeUnJob(t, windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE)
}

// lanzarDesdeUnJob levanta el lanzador dentro de un job con esos límites, espera a que lance al nieto, cierra el job
// (lo que hace Windows al cerrarse Claude Code) y devuelve el PID del nieto.
func lanzarDesdeUnJob(t *testing.T, limites uint32) int {
	t.Helper()
	dir := t.TempDir()
	job := jobConLimites(t, limites)
	lanzador := exec.Command(os.Args[0], "-test.run=^TestJobAyudante$")
	lanzador.Env = append(os.Environ(), envLanzadorJob+"="+dir)
	if err := lanzador.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { lanzador.Process.Kill() })
	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(lanzador.Process.Pid))
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(h)
	if err := windows.AssignProcessToJobObject(job, h); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "listo"), nil, 0o600)
	pid := esperarNieto(t, dir)
	t.Cleanup(func() {
		if p, err := os.FindProcess(pid); err == nil {
			p.Kill()
		}
	})
	windows.CloseHandle(job)
	time.Sleep(500 * time.Millisecond)
	if procesoVivo(lanzador.Process.Pid) {
		t.Fatal("el job no mató al lanzador: la prueba no reproduce el cierre de Claude Code")
	}
	return pid
}

func jobConLimites(t *testing.T, limites uint32) windows.Handle {
	t.Helper()
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = limites
	_, err = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)))
	if err != nil {
		t.Fatal(err)
	}
	return job
}

func esperarNieto(t *testing.T, dir string) int {
	t.Helper()
	tope := time.Now().Add(20 * time.Second)
	for time.Now().Before(tope) {
		if b, err := os.ReadFile(filepath.Join(dir, "error")); err == nil {
			t.Fatalf("el lanzador no pudo lanzar: %s", b)
		}
		b, err := os.ReadFile(filepath.Join(dir, "nieto"))
		pid, _ := strconv.Atoi(strings.TrimSpace(string(b)))
		if err == nil && pid > 0 {
			return pid
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("el lanzador no lanzó al nieto")
	return 0
}

func existe(ruta string) bool {
	_, err := os.Stat(ruta)
	return err == nil
}
