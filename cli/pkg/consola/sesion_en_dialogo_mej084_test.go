package consola

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// MEJ-084 (FND-234): una sesión de Claude Code detenida en un permiso (un trabajo en segundo plano, desde la mañana)
// seguía latiendo y tomaba las instrucciones del teléfono; Claude Code las dejaba en cola detrás del diálogo y el pedido
// quedaba «ejecutando» 15 min. Claude Code anota en <config>/sessions/<pid>.json que la sesión espera un diálogo: esa
// sesión no toma instrucciones; las toma otra abierta, o la misma cuando el diálogo se cierra.

// sesionDeClaude anota en un CLAUDE_CONFIG_DIR temporal lo que Claude Code publica de un proceso.
func sesionDeClaude(t *testing.T, config string, pid int, sesion, status string) {
	t.Helper()
	d := filepath.Join(config, "sessions")
	os.MkdirAll(d, 0o755)
	b, _ := json.Marshal(map[string]any{"pid": pid, "sessionId": sesion, "status": status, "waitingFor": "permission prompt", "kind": "bg"})
	if err := os.WriteFile(filepath.Join(d, strconv.Itoa(pid)+".json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func configDeClaude(t *testing.T) string {
	t.Helper()
	c := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", c)
	return c
}

func pidMuerto(t *testing.T) int {
	t.Helper()
	c := exec.Command(os.Args[0], "-test.run=^$")
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	return c.Process.Pid
}

func TestTelefono_UnaSesionEnUnDialogoNoTomaLaInstruccion_MEJ084(t *testing.T) {
	config := configDeClaude(t)
	root := proyectoConCola(t)
	LatirSesion(root, "sesion-bg", enTelefono)
	LatirSesion(root, "sesion-libre", enTelefono)
	sesionDeClaude(t, config, os.Getpid(), "sesion-bg", "waiting")
	RegistrarEjecucion(root, instruccion("s1", "responde con una línea"), enTelefono)
	if tomadas, _ := TomarParaEntregar(root, "sesion-bg", enTelefono.Add(time.Second)); len(tomadas) != 0 {
		t.Fatalf("una sesión que espera un diálogo no la toma: %+v", tomadas)
	}
	if tomadas, _ := TomarParaEntregar(root, "sesion-libre", enTelefono.Add(2*time.Second)); len(tomadas) != 1 {
		t.Fatalf("la toma la otra sesión abierta: %+v", tomadas)
	}
}

func TestTelefono_CuandoElDialogoSeCierraLaToma_MEJ084(t *testing.T) {
	config := configDeClaude(t)
	root := proyectoConCola(t)
	LatirSesion(root, "sesion-a", enTelefono)
	sesionDeClaude(t, config, os.Getpid(), "sesion-a", "waiting")
	RegistrarEjecucion(root, instruccion("s1", "responde con una línea"), enTelefono)
	if tomadas, _ := TomarParaEntregar(root, "sesion-a", enTelefono.Add(time.Second)); len(tomadas) != 0 {
		t.Fatalf("con el diálogo abierto espera: %+v", tomadas)
	}
	sesionDeClaude(t, config, os.Getpid(), "sesion-a", "idle")
	if tomadas, _ := TomarParaEntregar(root, "sesion-a", enTelefono.Add(2*time.Second)); len(tomadas) != 1 {
		t.Fatalf("libre, la toma: %+v", tomadas)
	}
}

func TestTelefono_UnaInstruccionDirigidaAUnaSesionEnUnDialogoLaTomaOtra_MEJ084(t *testing.T) {
	config := configDeClaude(t)
	root := proyectoConCola(t)
	LatirSesion(root, "sesion-bg", enTelefono)
	LatirSesion(root, "sesion-libre", enTelefono)
	sesionDeClaude(t, config, os.Getpid(), "sesion-bg", "waiting")
	x := instruccion("s1", "responde con una línea")
	x.Para = "sesion-bg"
	RegistrarEjecucion(root, x, enTelefono)
	if tomadas, _ := TomarParaEntregar(root, "sesion-libre", enTelefono.Add(time.Second)); len(tomadas) != 1 {
		t.Fatalf("la sesión destino no puede recibirla: la toma otra: %+v", tomadas)
	}
}

func TestTelefono_LoQueQuedaDeUnProcesoMuertoNoCuenta_MEJ084(t *testing.T) {
	config := configDeClaude(t)
	root := proyectoConCola(t)
	LatirSesion(root, "sesion-a", enTelefono)
	sesionDeClaude(t, config, pidMuerto(t), "sesion-a", "waiting")
	os.WriteFile(filepath.Join(config, "sessions", "roto.json"), []byte("{no es json"), 0o600)
	RegistrarEjecucion(root, instruccion("s1", "responde con una línea"), enTelefono)
	if tomadas, _ := TomarParaEntregar(root, "sesion-a", enTelefono.Add(time.Second)); len(tomadas) != 1 {
		t.Fatalf("un archivo de un proceso que ya no corre (o ilegible) no la detiene: %+v", tomadas)
	}
}

func TestTelefono_MientrasEsperaElDialogoElPedidoDiceElMotivoYVenceConEl_MEJ084(t *testing.T) {
	config := configDeClaude(t)
	root := proyectoConCola(t)
	LatirSesion(root, "5a61b6fd-a104", enTelefono)
	sesionDeClaude(t, config, os.Getpid(), "5a61b6fd-a104", "waiting")
	RegistrarEjecucion(root, instruccion("s1", "responde con una línea"), enTelefono)
	TomarParaEntregar(root, "5a61b6fd-a104", enTelefono.Add(time.Second))
	a := accionesDelTelefono(root, enTelefono.Add(time.Second))
	if len(a) != 1 || !strings.Contains(a[0].Resultado, "espera un diálogo en la terminal") {
		t.Fatalf("el teléfono ve por qué espera: %+v", a)
	}
	vencida, _ := EjecucionDe(root, "s1", enTelefono.Add(VidaInstruccion+time.Minute))
	if vencida.Estado != EstadoVencida || !strings.Contains(vencida.Motivo, "espera un diálogo en la terminal") {
		t.Fatalf("vence con el motivo, no con «ninguna sesión se abrió»: %+v", vencida)
	}
}

func TestTelefono_LaQueLaTomaDespuesBorraElMotivoDeEspera_MEJ084(t *testing.T) {
	config := configDeClaude(t)
	root := proyectoConCola(t)
	LatirSesion(root, "sesion-a", enTelefono)
	sesionDeClaude(t, config, os.Getpid(), "sesion-a", "waiting")
	RegistrarEjecucion(root, instruccion("s1", "responde con una línea"), enTelefono)
	TomarParaEntregar(root, "sesion-a", enTelefono.Add(time.Second))
	sesionDeClaude(t, config, os.Getpid(), "sesion-a", "idle")
	tomadas, _ := TomarParaEntregar(root, "sesion-a", enTelefono.Add(2*time.Second))
	if len(tomadas) != 1 || tomadas[0].Motivo != "" {
		t.Fatalf("tomada, ya no espera: %+v", tomadas)
	}
}
