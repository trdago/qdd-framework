package consola

import (
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// MEJ-082 (FND-233): con dos sesiones de Claude Code en el proyecto, la consola del teléfono mostraba una y la
// instrucción la tomaba la que latiera primero. Una instrucción dirigida (Para) solo la toma esa sesión mientras late;
// si ya no late, la toma cualquiera (no se queda sin entregar). El pedido del teléfono dice qué sesión la recibe.

func TestTelefono_UnaInstruccionParaUnaSesionVivaSoloLaTomaEsa_MEJ082(t *testing.T) {
	root := proyectoConCola(t)
	LatirSesion(root, "sesion-a", enTelefono)
	LatirSesion(root, "sesion-b", enTelefono)
	x := instruccion("s1", "responde con una línea")
	x.Para = "sesion-b"
	RegistrarEjecucion(root, x, enTelefono)
	if otra, _ := TomarParaEntregar(root, "sesion-a", enTelefono.Add(time.Second)); len(otra) != 0 {
		t.Fatalf("otra sesión viva no toma una instrucción dirigida: %+v", otra)
	}
	tomadas, _ := TomarParaEntregar(root, "sesion-b", enTelefono.Add(2*time.Second))
	if len(tomadas) != 1 || tomadas[0].Sesion != "sesion-b" {
		t.Fatalf("la toma la sesión que el teléfono mira: %+v", tomadas)
	}
}

func TestTelefono_SiLaSesionDestinoYaNoLateLaTomaOtra_MEJ082(t *testing.T) {
	root := proyectoConCola(t)
	LatirSesion(root, "sesion-a", enTelefono)
	x := instruccion("s1", "responde con una línea")
	x.Para = "sesion-b"
	RegistrarEjecucion(root, x, enTelefono)
	tomadas, _ := TomarParaEntregar(root, "sesion-a", enTelefono.Add(time.Second))
	if len(tomadas) != 1 {
		t.Fatalf("la sesión destino no late: no se queda sin entregar: %+v", tomadas)
	}
}

func TestTelefono_ElPedidoDiceQueSesionLaRecibe_MEJ082(t *testing.T) {
	root := proyectoConCola(t)
	LatirSesion(root, "sesion-b", enTelefono)
	x := instruccion("s1", "responde con una línea")
	x.Para = "sesion-b"
	RegistrarEjecucion(root, x, enTelefono)
	if a := accionesDelTelefono(root, enTelefono); len(a) != 1 || a[0].Sesion != "sesion-b" {
		t.Fatalf("mientras espera, dice a qué sesión va: %+v", a)
	}
	TomarParaEntregar(root, "sesion-b", enTelefono.Add(time.Second))
	if a := accionesDelTelefono(root, enTelefono.Add(time.Second)); len(a) != 1 || a[0].Sesion != "sesion-b" {
		t.Fatalf("tomada, dice qué sesión la recibió: %+v", a)
	}
}
