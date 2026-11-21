package remoto

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// FND-244 (MEJ-091, REP-507 de qdd-movile): Claude Code agrega a una sesión vieja metadatos sin hora (cost-state,
// last-prompt) al cerrar un proceso. Su archivo pasa a ser el más nuevo y la consola del teléfono saltaba a ella, con
// su «trabajando» vacío, mientras otra sesión de la misma carpeta trabajaba.
func TestTranscriptDe_UnaSesionViejaConMetadatosNoLeGanaALaQueTrabaja_FND244(t *testing.T) {
	homeTemporal(t)
	qdd := filepath.Join(t.TempDir(), "qdd")
	ahora := time.Now().UTC()
	conHora := func(tipo string, cuando time.Time) map[string]any {
		return map[string]any{"type": tipo, "timestamp": cuando.Format(time.RFC3339Nano), "message": map[string]any{"content": "x"}}
	}
	vieja := escribirTranscript(t, qdd, "vieja", conHora("user", ahora.Add(-3*24*time.Hour)), conHora("assistant", ahora.Add(-3*24*time.Hour)))
	actual := escribirTranscript(t, qdd, "actual", conHora("user", ahora.Add(-20*time.Second)), conHora("assistant", ahora.Add(-10*time.Second)))
	os.Chtimes(vieja, ahora.Add(-time.Hour), ahora.Add(-time.Hour))
	os.Chtimes(actual, ahora.Add(-10*time.Second), ahora.Add(-10*time.Second))
	if got := TranscriptDe(qdd); got != actual {
		t.Fatalf("antes de los metadatos, la que trabaja: %s", got)
	}
	escribirTranscript(t, qdd, "vieja", map[string]any{"type": "cost-state"}, map[string]any{"type": "last-prompt"})
	if got := TranscriptDe(qdd); got != actual {
		t.Fatalf("metadatos sin hora no hacen más nueva a una sesión vieja: %s", got)
	}
	escribirTranscript(t, qdd, "vieja", conHora("user", ahora))
	if got := TranscriptDe(qdd); got != vieja {
		t.Fatalf("un turno nuevo en la vieja sí la trae: %s", got)
	}
}

func TestUltimaHora_LaUltimaLineaConHora(t *testing.T) {
	homeTemporal(t)
	qdd := filepath.Join(t.TempDir(), "qdd")
	cuando := time.Date(2026, 11, 20, 23, 3, 21, 0, time.UTC)
	a := escribirTranscript(t, qdd, "s", map[string]any{"type": "user", "timestamp": "2026-11-20T23:00:04Z"},
		map[string]any{"type": "assistant", "timestamp": cuando.Format(time.RFC3339Nano)}, map[string]any{"type": "cost-state"})
	if got := ultimaHora(a); !got.Equal(cuando) {
		t.Fatalf("la hora del último turno, no la del archivo: %v", got)
	}
	sin := escribirTranscript(t, qdd, "sin", map[string]any{"type": "cost-state"})
	if got := ultimaHora(sin); !got.IsZero() {
		t.Fatalf("sin líneas con hora, cero: %v", got)
	}
}
