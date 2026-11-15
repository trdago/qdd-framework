package remoto

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// MEJ-078 (FND-228): la cola inicial son las últimas colaInicial líneas visibles, no las que caben en los últimos
// 512 KB. Un resultado de herramienta (una captura, una salida larga) pesa cientos de KB y no da ninguna línea.

// resultado es un turno de la persona que solo trae un resultado de herramienta de ese tamaño: no se muestra.
func resultado(kb int) map[string]any {
	return turno("user", []map[string]any{{"type": "tool_result", "tool_use_id": "x", "content": strings.Repeat("a", kb<<10)}})
}

func TestLector_LaColaInicialSonLasUltimasLineasVisiblesAunqueHayaResultadosGrandes_MEJ078(t *testing.T) {
	homeTemporal(t)
	cwd := t.TempDir()
	var lineas []map[string]any
	for i := 0; i < colaInicial+5; i++ {
		lineas = append(lineas, turno("assistant", fmt.Sprintf("turno %02d", i)), resultado(150))
	}
	archivo := escribirTranscript(t, cwd, "s", lineas...)
	_, cola := abrirLector(archivo)
	if len(cola) != colaInicial {
		t.Fatalf("con resultados de 150 KB entre medio, igual se ven las últimas %d: %d", colaInicial, len(cola))
	}
	for i, l := range cola {
		if want := fmt.Sprintf("turno %02d", i+5); l.Texto != want {
			t.Fatalf("en orden y completas, aunque una línea cruce de un bloque a otro: [%d] = %q, esperaba %q", i, l.Texto, want)
		}
	}
}

func TestLector_LaColaInicialNoLeeMasDeOchoMB_MEJ078(t *testing.T) {
	homeTemporal(t)
	cwd := t.TempDir()
	lineas := []map[string]any{turno("assistant", "lejana")}
	for i := 0; i < 20; i++ {
		lineas = append(lineas, resultado(512))
	}
	lineas = append(lineas, turno("assistant", "cercana"))
	archivo := escribirTranscript(t, cwd, "s", lineas...)
	_, cola := abrirLector(archivo)
	if len(cola) != 1 || cola[0].Texto != "cercana" {
		t.Fatalf("lo que está a más de %d MB del final no se lee: %+v", maxCola>>20, cola)
	}
}

func TestLector_UnaLineaAMedioEscribirMasGrandeQueUnBloqueSeCompletaDespues_MEJ078(t *testing.T) {
	homeTemporal(t)
	cwd := t.TempDir()
	archivo := escribirTranscript(t, cwd, "s", turno("assistant", "antes"))
	f, _ := os.OpenFile(archivo, os.O_APPEND|os.O_WRONLY, 0o600)
	f.Write([]byte(`{"type":"assistant","message":{"content":"` + strings.Repeat("b", leerDelFinal+10)))
	f.Close()
	l, cola := abrirLector(archivo)
	if len(cola) != 1 || cola[0].Texto != "antes" {
		t.Fatalf("la línea a medio escribir no se muestra todavía: %+v", cola)
	}
	f, _ = os.OpenFile(archivo, os.O_APPEND|os.O_WRONLY, 0o600)
	f.Write([]byte("fin\"}}\n"))
	f.Close()
	got := l.leer()
	if len(got) != 1 || !strings.HasPrefix(got[0].Texto, "bbbb") {
		t.Fatalf("cuando termina, se ve entera: %d líneas", len(got))
	}
}
