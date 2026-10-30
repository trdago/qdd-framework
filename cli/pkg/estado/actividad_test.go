package estado

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// RFC 0006 §3.13, blueprint estado-por-tareas: la tarea en curso la declara el agente y late con sus herramientas,
// sus subagentes y los comandos externos; se apaga sola a los VidaActividad sin pulso.

func proyectoConTareas(t *testing.T, ids ...string) string {
	t.Helper()
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	yaml := "formato: 1\netapas: [Código]\nfrentes:\n"
	for _, id := range ids {
		yaml += "  - {id: " + id + ", nombre: Tarea " + id + ", estado: abierto, celdas: {Código: {estado: ❌, falla: rota}}}\n"
	}
	ruta := filepath.Join(root, Ruta)
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestActividad_ElAgenteDeclaraYLaTareaQuedaEnCurso_EPT_C01(t *testing.T) {
	root := proyectoConTareas(t, "a", "b")
	if err := Activar(root, "a", "claude-code", ahora); err != nil {
		t.Fatal(err)
	}
	en := EnCurso(LeerActividad(root), ahora.Add(10*time.Second))
	if len(en) != 1 || en["a"].Fuente != FuenteAgente || en["a"].Quien != "claude-code" {
		t.Fatalf("la tarea declarada queda en curso: %+v", en)
	}
	ruta, err := RutaLocal(root, CarpetaActividad)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ruta); err != nil {
		t.Fatal("la actividad vive en la carpeta central de QDD (local, fuera del repositorio y de estado.yaml)")
	}
	antes, _ := os.ReadFile(filepath.Join(root, Ruta))
	if strings.Contains(string(antes), "pulso") {
		t.Fatal("estado.yaml no cambia: la actividad no es estado")
	}
}

func TestActividad_SoloTareasAbiertasQueExisten_EPT_C02(t *testing.T) {
	root := proyectoConTareas(t, "a")
	if err := Activar(root, "nada", "", ahora); err == nil {
		t.Fatal("una tarea que no existe no se puede activar")
	}
	if _, err := Pulsar(root, "nada", FuenteComando, ahora); err == nil {
		t.Fatal("un pulso con id de una tarea que no existe es un error")
	}
	if _, err := Pulsar(root, "a", "otra", ahora); err == nil {
		t.Fatal("la fuente es agente, subagente o comando")
	}
}

func TestActividad_PulsoSinIdLateLaUltimaActivada_EPT_C03(t *testing.T) {
	root := proyectoConTareas(t, "a", "b")
	if id, err := Pulsar(root, "", FuenteAgente, ahora); err != nil || id != "" {
		t.Fatalf("sin tarea en curso, un pulso sin id no hace nada y no falla: %q %v", id, err)
	}
	_ = Activar(root, "a", "", ahora)
	_ = Activar(root, "b", "", ahora.Add(time.Second))
	id, err := Pulsar(root, "", FuenteSubagente, ahora.Add(30*time.Second))
	if err != nil || id != "b" {
		t.Fatalf("late la última activada: %q %v", id, err)
	}
	b := LeerActividad(root).Tareas["b"]
	if b.Fuente != FuenteSubagente || b.Pulso != ahora.Add(30*time.Second).Format(time.RFC3339) {
		t.Fatalf("el pulso guarda su hora y su fuente: %+v", b)
	}
}

func TestActividad_UnComandoExternoEnciendeYLateSuTarea_EPT_C04(t *testing.T) {
	root := proyectoConTareas(t, "a")
	if id, err := Pulsar(root, "a", FuenteComando, ahora); err != nil || id != "a" {
		t.Fatalf("un comando externo con id deja su pulso: %q %v", id, err)
	}
	if en := EnCurso(LeerActividad(root), ahora); en["a"].Fuente != FuenteComando {
		t.Fatalf("la tarea queda en curso por el comando: %+v", en)
	}
}

func TestActividad_SeApagaSolaSinPulsos_EPT_C05(t *testing.T) {
	root := proyectoConTareas(t, "a")
	_ = Activar(root, "a", "", ahora)
	if en := EnCurso(LeerActividad(root), ahora.Add(VidaActividad-time.Second)); len(en) != 1 {
		t.Fatal("dentro de VidaActividad sigue en curso")
	}
	if en := EnCurso(LeerActividad(root), ahora.Add(VidaActividad+time.Second)); len(en) != 0 {
		t.Fatal("un agente que se cae no deja la tarea encendida")
	}
	if id, _ := Pulsar(root, "", FuenteAgente, ahora.Add(VidaActividad+time.Second)); id != "" {
		t.Fatal("un pulso sin id no revive una tarea vencida")
	}
}

func TestActividad_TerminarYCerrarLaApagan_EPT_C06(t *testing.T) {
	root := proyectoConTareas(t, "a", "b")
	_ = Activar(root, "a", "", ahora)
	_ = Activar(root, "b", "", ahora)
	if err := Terminar(root, "a"); err != nil {
		t.Fatal(err)
	}
	if err := Cerrar(root, "b", "descartada", true, ahora); err != nil {
		t.Fatal(err)
	}
	if en := EnCurso(LeerActividad(root), ahora); len(en) != 0 {
		t.Fatalf("terminar y cerrar (o descartar) apagan la tarea: %+v", en)
	}
	if err := Terminar(root, "a"); err != nil {
		t.Fatal("terminar una tarea que no está en curso no es un error")
	}
}

func TestActividad_ArchivoIlegibleEsSinTareasEnCurso(t *testing.T) {
	root := proyectoConTareas(t, "a")
	ruta, _ := RutaLocal(root, CarpetaActividad)
	_ = os.MkdirAll(filepath.Dir(ruta), 0o755)
	_ = os.WriteFile(ruta, []byte("{roto"), 0o644)
	if en := EnCurso(LeerActividad(root), ahora); len(en) != 0 {
		t.Fatal("sin archivo legible no hay tareas en curso")
	}
	if err := Activar(root, "a", "", ahora); err != nil {
		t.Fatalf("activar reescribe un archivo ilegible: %v", err)
	}
}

func TestDatosEnCurso_LaTareaEnCursoSiempreSeVe_EPT_C07(t *testing.T) {
	yaml := "formato: 1\netapas: [Código]\nfrentes:\n"
	for _, id := range []string{"t1", "t2", "t3", "t4", "t5", "t6"} {
		yaml += "  - {id: " + id + ", nombre: " + id + ", estado: abierto, celdas: {Código: {estado: ❌, falla: rota}}}\n"
	}
	act := Actividades{Ultima: "t1", Tareas: map[string]Actividad{
		"t1": {Desde: ahora.Format(time.RFC3339), Pulso: ahora.Format(time.RFC3339), Fuente: FuenteAgente},
	}}
	d := DatosEnCurso(mustParse(t, yaml), act, ahora.Add(5*time.Second), 4)
	var ids []string
	for _, f := range d.Frentes {
		ids = append(ids, f.ID)
	}
	if strings.Join(ids, ",") != "t1,t3,t4,t5,t6" || d.Ocultos != 1 {
		t.Fatalf("t1 en curso se ve aunque no sea de las 4 más nuevas: %v ocultos=%d", ids, d.Ocultos)
	}
	if d.Frentes[0].Actividad == nil || !d.Frentes[0].Actividad.Running || d.Frentes[1].Actividad != nil || d.Actual != "t1" {
		t.Fatalf("solo la tarea en curso lleva actividad, y actual es la última activada: %+v %q", d.Frentes[:2], d.Actual)
	}
	vencida := DatosEnCurso(mustParse(t, yaml), act, ahora.Add(VidaActividad+time.Second), 4)
	if len(vencida.Frentes) != 4 || vencida.Actual != "" {
		t.Fatalf("vencida, vuelve el tope de 4 y no hay actual: %d %q", len(vencida.Frentes), vencida.Actual)
	}
}

// FND-079: .qdd/working es el archivo con el comando qdd en curso (root.go) y no puede ser una carpeta; con la
// actividad dentro de él, activar fallaba con «not a directory» y la tarea nunca quedaba «● running».
func TestActividad_NoChocaConElArchivoWorking_FND079(t *testing.T) {
	root := proyectoConTareas(t, "a")
	working := filepath.Join(root, ".qdd", "working")
	if err := os.WriteFile(working, []byte("qdd"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Activar(root, "a", "claude-code", ahora); err != nil {
		t.Fatalf("activar funciona con .qdd/working como archivo: %v", err)
	}
	if got, err := Pulsar(root, "", FuenteAgente, ahora.Add(time.Second)); err != nil || got != "a" {
		t.Fatalf("el pulso late la tarea: %q, %v", got, err)
	}
	if en := EnCurso(LeerActividad(root), ahora.Add(2*time.Second)); len(en) != 1 {
		t.Fatalf("la tarea queda en curso: %+v", en)
	}
	data, _ := os.ReadFile(working)
	if string(data) != "qdd" {
		t.Fatal(".qdd/working no se toca")
	}
}

func TestRutaLocal_FueraDelRepositorioYPorProyecto(t *testing.T) {
	home := t.TempDir()
	t.Setenv("QDD_HOME", home)
	uno, dos := filepath.Join(t.TempDir(), "Mi Proyecto"), filepath.Join(t.TempDir(), "Mi Proyecto")
	r1, err := RutaLocal(uno, "actividad")
	if err != nil {
		t.Fatal(err)
	}
	r2, _ := RutaLocal(dos, "actividad")
	if !strings.HasPrefix(r1, filepath.Join(home, "claude", "actividad", "mi-proyecto-")) || !strings.HasSuffix(r1, ".json") {
		t.Fatalf("<QDD_HOME>/claude/<carpeta>/<slug>-<hash>.json: %s", r1)
	}
	if r1 == r2 {
		t.Fatal("dos proyectos con el mismo nombre no comparten archivo")
	}
}
