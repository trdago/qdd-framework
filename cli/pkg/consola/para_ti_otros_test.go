package consola

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// «Para ti» de todos los proyectos (2026-11-15): la persona mira el panel de una sesión y no vio las fichas ni las
// preguntas de proyecto-b, abiertas en otra. Con --con otros, el panel trae lo que espera a la persona en los demás
// proyectos de la máquina (registro de proyectos), con su carpeta, para mostrarlo y responderlo desde ahí.

func proyectoRegistrado(t *testing.T, visto time.Time) string {
	t.Helper()
	root := t.TempDir()
	for _, d := range []string{".qdd", "src"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := registro.Anotar(root, visto); err != nil {
		t.Fatal(err)
	}
	return root
}

func otrosDe(root string, now time.Time) []ProyectoEsperando {
	d := Panel(root, now, TipoPregunta, ConOtros)
	if d.Otros == nil {
		return nil
	}
	return *d.Otros
}

func TestParaTiOtros_TraeLoQueEsperaALaPersonaEnOtroProyecto(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	now := time.Now()
	aqui := proyectoRegistrado(t, now)
	otro := proyectoRegistrado(t, now)
	if _, err := Pedir(otro, "git push", "Subir el arreglo", "", now); err != nil {
		t.Fatal(err)
	}
	fichaEnBorrador(t, otro)
	pregunta := registrada(t, otro, NuevaPregunta{Sesion: "s9", Herramienta: herramientaPreguntar, Entrada: askDePrueba("¿Qué base usamos?")}, now)
	escribirArchivo(t, filepath.Join(otro, estado.Ruta), estadoBaseEncima)
	if err := estado.Set(otro, estado.Cambio{Frente: "base", Etapa: "Código", Estado: "⏳", Bloqueo: "¿Sigo con la migración?", DeQuien: "la persona"}, now); err != nil {
		t.Fatal(err)
	}

	otros := otrosDe(aqui, now)
	if len(otros) != 1 || otros[0].Carpeta != otro || otros[0].Proyecto != filepath.Base(otro) {
		t.Fatalf("trae el otro proyecto, con su nombre y su carpeta: %+v", otros)
	}
	ids := map[string]bool{}
	for _, a := range otros[0].Acciones {
		if a.Turno == 0 {
			t.Fatalf("solo trae lo que espera a la persona: %+v", a)
		}
		ids[a.ID] = true
	}
	if !ids[pregunta] || !ids[prefijoAprobar+"pagos"] || len(otros[0].Acciones) != 4 {
		t.Fatalf("trae el pedido, la ficha, la pregunta y el ⏳: %v", ids)
	}
	if len(otros[0].Esperas) != 1 || otros[0].Esperas[0].Texto != "¿Sigo con la migración?" {
		t.Fatalf("trae el texto del ⏳: %+v", otros[0].Esperas)
	}
}

func TestParaTiOtros_NoRepiteElProyectoPropioNiLosQueNoEsperanNada(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	now := time.Now()
	aqui := proyectoRegistrado(t, now)
	proyectoRegistrado(t, now)
	if _, err := Pedir(aqui, "git push", "Subir", "", now); err != nil {
		t.Fatal(err)
	}
	if otros := otrosDe(aqui, now); len(otros) != 0 {
		t.Fatalf("lo propio ya está en «Para ti» y un proyecto sin nada no se nombra: %+v", otros)
	}
}

func TestParaTiOtros_UnProyectoViejoOBorradoNoSeLee(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	now := time.Now()
	aqui := proyectoRegistrado(t, now)
	viejo := proyectoRegistrado(t, now.Add(-OtrosRecientes-time.Hour))
	borrado := proyectoRegistrado(t, now)
	for _, root := range []string{viejo, borrado} {
		if _, err := Pedir(root, "git push", "Subir", "", now); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.RemoveAll(borrado); err != nil {
		t.Fatal(err)
	}
	if otros := otrosDe(aqui, now); len(otros) != 0 {
		t.Fatalf("ni uno sin abrir hace más de %v ni uno que ya no existe: %+v", OtrosRecientes, otros)
	}
}

func TestParaTiOtros_SinPedirloNoLeeOtrosProyectos(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	now := time.Now()
	aqui := proyectoRegistrado(t, now)
	otro := proyectoRegistrado(t, now)
	if _, err := Pedir(otro, "git push", "Subir", "", now); err != nil {
		t.Fatal(err)
	}
	if d := Panel(aqui, now, TipoPregunta); d.Otros != nil {
		t.Fatalf("un panel anterior (o el teléfono) no recibe otros proyectos: %+v", *d.Otros)
	}
}

func TestParaTiOtros_TieneTope(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	now := time.Now()
	aqui := proyectoRegistrado(t, now)
	for i := range MaxOtros + 2 {
		otro := proyectoRegistrado(t, now.Add(-time.Duration(i)*time.Minute))
		if _, err := Pedir(otro, "git push", "Subir", "", now); err != nil {
			t.Fatal(err)
		}
	}
	if otros := otrosDe(aqui, now); len(otros) != MaxOtros {
		t.Fatalf("lee como mucho %d proyectos, los vistos más recientemente: %d", MaxOtros, len(otros))
	}
}
