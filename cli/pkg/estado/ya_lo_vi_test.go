package estado

import (
	"strings"
	"testing"
	"time"
)

var ahoraVisto = time.Date(2026, 11, 5, 21, 0, 0, 0, time.UTC)

// tareaParaVer deja una tarea con Código ✅, Producción ⏳ y Validación con el estado pedido.
func tareaParaVer(t *testing.T, validacion Cambio) string {
	t.Helper()
	root := t.TempDir()
	if err := Init(root, []string{"Código", "Producción", "Validación"}, ahoraVisto); err != nil {
		t.Fatal(err)
	}
	cambios := []Cambio{
		{Frente: "a", Nombre: "Tarea A", Etapa: "Código", Estado: "ok", Evidencia: "abc123"},
		{Frente: "a", Etapa: "Producción", Estado: "espera", Bloqueo: "probar en producción", DeQuien: "humano"},
		validacion,
	}
	for _, c := range cambios {
		if err := Set(root, c, ahoraVisto); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func tareaA(t *testing.T, root string) Frente {
	t.Helper()
	e, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return e.Frentes[e.buscar("a")]
}

// «✓ Ya lo vi» deja la etapa lista con una nota que no finge una medición, y sin nada más que hacer cierra la tarea.
func TestYaLoVio_DejaListasLasEsperasConSuNotaYCierraLaTarea(t *testing.T) {
	root := tareaParaVer(t, Cambio{Frente: "a", Etapa: "Validación", Estado: "espera", Bloqueo: "usarla", DeQuien: "humano"})
	v, err := YaLoVio(root, "a", ahoraVisto)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(v.Etapas, ",") != "Producción,Validación" || !v.Cerrada {
		t.Fatalf("resultado: %+v", v)
	}
	f := tareaA(t, root)
	c := f.Celdas["Producción"]
	if c.Estado != Hecho || c.Evidencia != "la vio la persona el 05/11, sin medición" || c.Bloqueo != "" || c.DeQuien != "" {
		t.Fatalf("celda: %+v", c)
	}
	if f.Estado != EstadoCerrado || !strings.Contains(f.MotivoCierre, "la vio la persona") {
		t.Fatalf("tarea: %+v", f)
	}
	if f.Celdas["Código"].Evidencia != "abc123" {
		t.Fatalf("una etapa lista no se toca: %+v", f.Celdas["Código"])
	}
}

// Una etapa sin empezar o con falla es trabajo que falta: la tarea sigue abierta y esas celdas no cambian.
func TestYaLoVio_LaTareaSigueAbiertaSiQuedaAlgoPorHacer(t *testing.T) {
	casos := map[string]Cambio{
		"sin empezar": {Frente: "a", Etapa: "Validación", Estado: "sin-empezar"},
		"con falla":   {Frente: "a", Etapa: "Validación", Estado: "falla", Falla: "no abre"},
	}
	for nombre, validacion := range casos {
		t.Run(nombre, func(t *testing.T) {
			root := tareaParaVer(t, validacion)
			antes := tareaA(t, root).Celdas["Validación"]
			v, err := YaLoVio(root, "a", ahoraVisto)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Join(v.Etapas, ",") != "Producción" || v.Cerrada {
				t.Fatalf("resultado: %+v", v)
			}
			f := tareaA(t, root)
			if f.Estado != EstadoAbierto || f.Celdas["Validación"].Estado != antes.Estado {
				t.Fatalf("tarea: %+v", f)
			}
		})
	}
}

// El ✅ de una etapa que requiere aprobación lo da quien debía (EPF-C06): la persona que mira no lo reemplaza.
func TestYaLoVio_NoSaltaUnaAprobacionPendiente(t *testing.T) {
	root := tareaParaVer(t, Cambio{Frente: "a", Etapa: "Validación", Estado: "espera", Bloqueo: "aprobar", DeQuien: "legal", RequiereAprobacion: "legal"})
	e, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(e.PorVer("a"), ","); got != "Producción" {
		t.Fatalf("por ver: %q", got)
	}
	v, err := YaLoVio(root, "a", ahoraVisto)
	if err != nil {
		t.Fatal(err)
	}
	f := tareaA(t, root)
	if v.Cerrada || f.Estado != EstadoAbierto || f.Celdas["Validación"].Estado != Espera {
		t.Fatalf("resultado %+v, tarea %+v", v, f)
	}
}

func TestYaLoVio_SinNadaPorVerEsUnError(t *testing.T) {
	root := tareaParaVer(t, Cambio{Frente: "a", Etapa: "Validación", Estado: "espera", Bloqueo: "usarla", DeQuien: "humano"})
	if _, err := YaLoVio(root, "a", ahoraVisto); err != nil {
		t.Fatal(err)
	}
	if _, err := YaLoVio(root, "a", ahoraVisto); err == nil {
		t.Fatal("una tarea sin ⏳ no tiene nada que marcar")
	}
	if _, err := YaLoVio(root, "no-existe", ahoraVisto); err == nil {
		t.Fatal("una tarea que no existe es un error")
	}
}
