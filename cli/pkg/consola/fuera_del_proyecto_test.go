package consola

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// La cola de pedidos y la tarea en curso son de esta máquina, no del proyecto: viven en <QDD_HOME>/claude/<carpeta>/ y
// el repositorio queda como estaba.

// fotoDelProyecto es cada archivo del proyecto con su contenido.
func fotoDelProyecto(t *testing.T, root string) map[string]string {
	t.Helper()
	foto := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		rel, _ := filepath.Rel(root, p)
		foto[filepath.ToSlash(rel)] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return foto
}

func mismaFoto(t *testing.T, que string, antes, despues map[string]string) {
	t.Helper()
	for ruta, texto := range despues {
		if previo, ok := antes[ruta]; !ok || previo != texto {
			t.Fatalf("%s escribió %s en el proyecto", que, ruta)
		}
	}
	if len(antes) != len(despues) {
		t.Fatalf("%s cambió los archivos del proyecto: %d → %d", que, len(antes), len(despues))
	}
}

const estadoConEspera = `formato: 1
etapas: [Código]
frentes:
  - {id: t, nombre: Tarea, estado: abierto, celdas: {Código: {estado: "⏳", bloqueo: "Publicar: pide [a]", de_quien: la persona}}}
`

// contrato: estado#8f001589
func TestPedirYDescartar_NoEscribenNadaEnElProyecto(t *testing.T) {
	root := proyectoConCola(t)
	escribirArchivo(t, filepath.Join(root, estado.Ruta), estadoConEspera)
	antes := fotoDelProyecto(t, root)
	deTarea, err := PedirEnOrden(root, "git push origin v9.9.9", "Publicar v9.9.9", "", Orden{Tarea: "t"}, enCola)
	if err != nil {
		t.Fatal(err)
	}
	conPR := mustPedirEnOrden(t, root, "gh pr merge 14", Orden{Despues: []string{deTarea.ID}})
	conVersion := mustPedir(t, root, "npm i -g qdd@9.9.9", "Actualizar qdd", "")
	err = conCola(root, enCola, func(cola []Pedido) ([]Pedido, error) {
		for i := range cola {
			if cola[i].ID == conPR.ID {
				cola[i].PR = 14
			}
			if cola[i].ID == conVersion.ID {
				cola[i].Version = "9.9.9"
			}
		}
		return cola, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	mismaFoto(t, "pedir", antes, fotoDelProyecto(t, root))
	for _, p := range []Pedido{deTarea, conPR, conVersion} {
		if err := Descartar(root, p.ID); err != nil {
			t.Fatal(err)
		}
	}
	if cola := Pedidos(root); len(cola) != 0 {
		t.Fatalf("descartados, la cola queda vacía: %+v", cola)
	}
	mismaFoto(t, "descartar", antes, fotoDelProyecto(t, root))
}

// contrato: estado#7a3a2b72
// Activar, latir y terminar escriben solo en <QDD_HOME>/claude/actividad/, con la misma clave de proyecto que la cola de
// pedidos; estado.yaml y .qdd/working (el comando qdd en curso) quedan como estaban.
func TestActividad_ViveFueraDelRepositorioConLaClaveDeLaCola(t *testing.T) {
	root := proyectoConCola(t)
	escribirArchivo(t, filepath.Join(root, estado.Ruta), estadoConEspera)
	escribirArchivo(t, filepath.Join(root, ".qdd", "working"), "qdd audit")
	antes := fotoDelProyecto(t, root)
	ahora := time.Now()
	if err := estado.Activar(root, "t", "claude-code", ahora); err != nil {
		t.Fatal(err)
	}
	if id, err := estado.Pulsar(root, "", estado.FuenteAgente, ahora.Add(time.Second)); err != nil || id != "t" {
		t.Fatalf("late la tarea activada: %q %v", id, err)
	}
	mismaFoto(t, "activar y latir", antes, fotoDelProyecto(t, root))
	actividad, err := estado.RutaLocal(root, estado.CarpetaActividad)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(actividad); err != nil {
		t.Fatalf("la actividad queda en %s: %v", actividad, err)
	}
	cola, _ := colaPath(root)
	if filepath.Dir(actividad) != filepath.Join(os.Getenv("QDD_HOME"), "claude", "actividad") || filepath.Base(actividad) != filepath.Base(cola) {
		t.Fatalf("<QDD_HOME>/claude/actividad/ con la clave de la cola: %s y %s", actividad, cola)
	}
	if err := estado.Terminar(root, "t"); err != nil {
		t.Fatal(err)
	}
	mismaFoto(t, "terminar", antes, fotoDelProyecto(t, root))
}
