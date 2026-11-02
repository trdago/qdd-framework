package remoto

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
)

// Rule: Prohibido usar 'else'
// RFC 0013 (rc.4): de qué transcript sale la consola de un proyecto y qué se saca de cada línea.

// escribirTranscript deja un transcript de Claude Code para cwd con esas líneas (cada una, un objeto JSON).
func escribirTranscript(t *testing.T, cwd, sesion string, lineas ...map[string]any) string {
	t.Helper()
	dir := filepath.Join(consola.ClaudeDir(), "projects", carpetaCodificada(cwd))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	archivo := filepath.Join(dir, sesion+".jsonl")
	f, err := os.OpenFile(archivo, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, l := range lineas {
		if _, ok := l["cwd"]; !ok {
			l["cwd"] = cwd
		}
		b, _ := json.Marshal(l)
		f.Write(append(b, '\n'))
	}
	return archivo
}

func turno(tipo string, contenido any) map[string]any {
	return map[string]any{"type": tipo, "timestamp": "2026-11-02T10:00:00Z", "message": map[string]any{"content": contenido}}
}

func TestTranscriptDe_ElMasRecienteDelProyectoYNoElDeUnVecino(t *testing.T) {
	homeTemporal(t)
	base := t.TempDir()
	qdd, vecino := filepath.Join(base, "qdd"), filepath.Join(base, "qdd-movil")
	sub := filepath.Join(qdd, "cli")
	viejo := escribirTranscript(t, qdd, "vieja", turno("user", "hola"))
	nuevo := escribirTranscript(t, sub, "nueva", turno("user", "desde cli"))
	delVecino := escribirTranscript(t, vecino, "otra", turno("user", "no es mío"))
	ahora := time.Now()
	os.Chtimes(viejo, ahora.Add(-time.Hour), ahora.Add(-time.Hour))
	os.Chtimes(nuevo, ahora.Add(-time.Minute), ahora.Add(-time.Minute))
	os.Chtimes(delVecino, ahora, ahora)
	if got := TranscriptDe(qdd); got != nuevo {
		t.Fatalf("el más reciente del proyecto o de una subcarpeta, nunca el de qdd-movil: %s", got)
	}
	if got := TranscriptDe(vecino); got != delVecino {
		t.Fatalf("el vecino tiene el suyo: %s", got)
	}
	if got := TranscriptDe(filepath.Join(base, "nada")); got != "" {
		t.Fatalf("sin transcript, vacío: %s", got)
	}
}

func TestLineasDe_TurnosYHerramientasSinResultadosNiRazonamiento(t *testing.T) {
	linea := func(m map[string]any) []byte {
		b, _ := json.Marshal(m)
		return b
	}
	persona := LineasDe(linea(turno("user", "arregla el test")))
	if len(persona) != 1 || persona[0].Rol != "persona" || persona[0].Texto != "arregla el test" || persona[0].Fuente != "conversacion" {
		t.Fatalf("un turno de la persona: %+v", persona)
	}
	ia := LineasDe(linea(turno("assistant", []map[string]any{
		{"type": "thinking", "thinking": "lo que pienso"},
		{"type": "text", "text": "Voy a mirar el test."},
		{"type": "tool_use", "name": "Bash", "input": map[string]any{"command": "go test ./..."}},
	})))
	if len(ia) != 2 || ia[0].Rol != "ia" || ia[1].Rol != "herramienta" || ia[1].Texto != "Bash" {
		t.Fatalf("texto y nombre de la herramienta, sin razonamiento ni entrada: %+v", ia)
	}
	nada := [][]byte{
		linea(turno("user", []map[string]any{{"type": "tool_result", "content": strings.Repeat("salida enorme ", 1000)}})),
		linea(turno("user", "<system-reminder>algo</system-reminder>")),
		linea(map[string]any{"type": "user", "isSidechain": true, "message": map[string]any{"content": "de un subagente"}}),
		linea(map[string]any{"type": "user", "isMeta": true, "message": map[string]any{"content": "meta"}}),
		linea(map[string]any{"type": "summary", "summary": "resumen"}),
		[]byte("no es json"),
	}
	for _, l := range nada {
		if got := LineasDe(l); len(got) != 0 {
			t.Fatalf("no se muestra: %s → %+v", l, got)
		}
	}
	largo := LineasDe(linea(turno("assistant", strings.Repeat("x", 5000))))
	if len([]rune(largo[0].Texto)) > maxTexto+1 {
		t.Fatalf("recorta los turnos largos: %d", len(largo[0].Texto))
	}
}

func TestLector_ColaInicialYDespuesSoloLoNuevo(t *testing.T) {
	homeTemporal(t)
	cwd := t.TempDir()
	var lineas []map[string]any
	for i := 0; i < colaInicial+10; i++ {
		lineas = append(lineas, turno("user", "turno"))
	}
	archivo := escribirTranscript(t, cwd, "s", lineas...)
	l, cola := abrirLector(archivo)
	if len(cola) != colaInicial {
		t.Fatalf("quien empieza a mirar ve las últimas %d: %d", colaInicial, len(cola))
	}
	if got := l.leer(); len(got) != 0 {
		t.Fatalf("sin nada nuevo, nada: %+v", got)
	}
	escribirTranscript(t, cwd, "s", turno("assistant", "nuevo"))
	f, _ := os.OpenFile(archivo, os.O_APPEND|os.O_WRONLY, 0o600)
	f.Write([]byte(`{"type":"user","message":{"content":"a me`))
	f.Close()
	got := l.leer()
	if len(got) != 1 || got[0].Texto != "nuevo" {
		t.Fatalf("lo nuevo y no la línea a medio escribir: %+v", got)
	}
	f, _ = os.OpenFile(archivo, os.O_APPEND|os.O_WRONLY, 0o600)
	f.Write([]byte("dias\"}}\n"))
	f.Close()
	got = l.leer()
	if len(got) != 1 || got[0].Texto != "a medias" {
		t.Fatalf("la completa cuando termina: %+v", got)
	}
}
