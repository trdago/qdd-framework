package estado

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// Contrato de la ficha estado-por-tareas sobre la tarea en curso (RFC 0006 §3.13): la actividad no es estado, solo
// una tarea abierta puede estar en curso y la actividad no guarda datos de personas.

// contrato: estado-por-tareas#1e3a84fe
func TestActividad_NoCambiaLaTablaNiSeCommitea(t *testing.T) {
	root := proyectoConTareas(t, "a", "b")
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}, {"-c", "user.email=t@t", "-c", "user.name=t", "commit", "-qm", "base"}} {
		c := exec.Command("git", args...)
		c.Dir = root
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	antes, err := os.ReadFile(filepath.Join(root, Ruta))
	if err != nil {
		t.Fatal(err)
	}
	if err := Activar(root, "a", "claude-code", ahora); err != nil {
		t.Fatal(err)
	}
	if _, err := Pulsar(root, "b", FuenteComando, ahora.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := Pulsar(root, "", FuenteAgente, ahora.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	if en := EnCurso(LeerActividad(root), ahora.Add(3*time.Second)); len(en) != 2 {
		t.Fatalf("las dos tareas quedan en curso: %+v", en)
	}
	despues, _ := os.ReadFile(filepath.Join(root, Ruta))
	if string(despues) != string(antes) {
		t.Fatalf("activar y pulsar no cambian la tabla:\nantes:\n%s\ndespués:\n%s", antes, despues)
	}
	ruta, err := RutaLocal(root, CarpetaActividad)
	if err != nil {
		t.Fatal(err)
	}
	if dentro, _ := filepath.Rel(root, ruta); !strings.HasPrefix(dentro, "..") {
		t.Fatalf("la actividad vive fuera del proyecto: %s", ruta)
	}
	c := exec.Command("git", "status", "--porcelain", "--ignored")
	c.Dir = root
	out, err := c.Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != "" {
		t.Fatalf("la actividad no deja nada que commitear ni ignorar en el proyecto:\n%s", out)
	}
}

// contrato: estado-por-tareas#b8d58263
func TestActividad_SoloUnaTareaAbiertaEstaEnCurso(t *testing.T) {
	root := proyectoConTareas(t, "a", "b", "c")
	if err := Cerrar(root, "c", "hecha en otro PR", false, ahora); err != nil {
		t.Fatal(err)
	}
	if err := Activar(root, "c", "claude-code", ahora); err == nil {
		t.Fatal("una tarea cerrada no se puede activar")
	}
	if _, err := Pulsar(root, "c", FuenteComando, ahora); err == nil {
		t.Fatal("un pulso no enciende una tarea cerrada")
	}
	_ = Activar(root, "a", "claude-code", ahora)
	_ = Activar(root, "b", "claude-code", ahora)
	if err := Cerrar(root, "a", "lista y validada", false, ahora); err != nil {
		t.Fatal(err)
	}
	if err := Cerrar(root, "b", "ya no hace falta", true, ahora); err != nil {
		t.Fatal(err)
	}
	if en := EnCurso(LeerActividad(root), ahora); len(en) != 0 {
		t.Fatalf("cerrarla o descartarla la apaga: %+v", en)
	}
	// Una actividad que quedó de antes (p. ej. el archivo se cerró a mano) no hace «● running» a una tarea cerrada.
	act := Actividades{Ultima: "a", Tareas: map[string]Actividad{"a": {Desde: ahora.Format(time.RFC3339), Pulso: ahora.Format(time.RFC3339), Fuente: FuenteAgente}}}
	d := DatosEnCurso(cargar(t, root), act, ahora.Add(time.Second), 4)
	for _, f := range d.Frentes {
		if f.Actividad != nil {
			t.Fatalf("el panel no muestra en curso una tarea cerrada: %+v", f)
		}
	}
	if d.Actual != "" {
		t.Fatalf("ni la da por actual: %q", d.Actual)
	}
}

// contrato: estado-por-tareas#96a91e48
func TestActividad_QuienEsUnaHerramientaNoUnaPersona_FND276(t *testing.T) {
	root := proyectoConTareas(t, "a")
	for _, persona := range []string{"juan@empresa.cl", "Juan Pérez", "+56 9 1234 5678", "12.345.678-5", "token=abc123", "ghp_abcdefghijklmnopqrstuv"} {
		if err := Activar(root, "a", persona, ahora); err == nil {
			t.Errorf("«%s» no es una herramienta: no se guarda", persona)
		}
		data, _ := os.ReadFile(rutaActividad(t, root))
		if strings.Contains(string(data), persona) {
			t.Fatalf("la actividad no guarda «%s»:\n%s", persona, data)
		}
	}
	for _, herramienta := range []string{"claude-code", "mcp", "gemini-cli", "opencode", ""} {
		if err := Activar(root, "a", herramienta, ahora); err != nil {
			t.Errorf("«%s» es una herramienta: %v", herramienta, err)
		}
	}
	data, _ := os.ReadFile(rutaActividad(t, root))
	for _, campo := range []string{"@", "Pérez"} {
		if strings.Contains(string(data), campo) {
			t.Fatalf("la actividad no guarda datos de personas:\n%s", data)
		}
	}
}

func rutaActividad(t *testing.T, root string) string {
	t.Helper()
	ruta, err := RutaLocal(root, CarpetaActividad)
	if err != nil {
		t.Fatal(err)
	}
	return ruta
}
